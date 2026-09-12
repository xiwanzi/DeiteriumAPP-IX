package store

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
)

var ErrAssetForbidden = errors.New("asset access denied")
var ErrAssetUnavailable = errors.New("asset is not ready")
var ErrAssetInUse = errors.New("asset is used by a business resource")
var ErrAssetBusy = errors.New("asset lifecycle work is in progress")

type AssetUploadV2 struct {
	UploadID, AssetID, UserID, ClientRequestID, Fingerprint, Purpose, BusinessType, BusinessRef, ObjectKey, ContentType, MD5, AltText, Status string
	SizeBytes                                                                                                                                 int64
	Width, Height                                                                                                                             sql.NullInt64
	SHA256, Rejection                                                                                                                         sql.NullString
	CreatedAt, ExpiresAt                                                                                                                      time.Time
	VerificationStarted, RemovedAt                                                                                                            sql.NullTime
	Lifecycle, SourceKey, PendingKey                                                                                                          string
	RetiredAt, RetainUntil, KeepUntil                                                                                                         sql.NullTime
}

const assetSelectV2 = `SELECT upload_id,asset_id,user_id,client_request_id,fingerprint,purpose,business_type,business_ref,object_key,content_type,size_bytes,content_md5,alt_text,status,width,height,sha256,rejection_code,created_at,expires_at,verification_started_at,removed_at,lifecycle_state,COALESCE(source_object_key,''),COALESCE(pending_object_key,''),retired_at,retain_until,lifecycle_keep_until FROM asset_uploads_v2`

type assetScannerV2 interface{ Scan(...any) error }

func scanAssetV2(row assetScannerV2) (u AssetUploadV2, err error) {
	err = row.Scan(&u.UploadID, &u.AssetID, &u.UserID, &u.ClientRequestID, &u.Fingerprint, &u.Purpose, &u.BusinessType, &u.BusinessRef, &u.ObjectKey, &u.ContentType, &u.SizeBytes, &u.MD5, &u.AltText, &u.Status, &u.Width, &u.Height, &u.SHA256, &u.Rejection, &u.CreatedAt, &u.ExpiresAt, &u.VerificationStarted, &u.RemovedAt, &u.Lifecycle, &u.SourceKey, &u.PendingKey, &u.RetiredAt, &u.RetainUntil, &u.KeepUntil)
	return
}

func (s *Store) CreateAssetUploadV2(ctx context.Context, u AssetUploadV2) (AssetUploadV2, error) {
	tx, err := s.beginAccountTx(ctx, u.UserID)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	var lockedUser string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? FOR UPDATE", u.UserID).Scan(&lockedUser); err != nil {
		return u, err
	}
	existing, err := scanAssetV2(tx.QueryRowContext(ctx, assetSelectV2+" WHERE user_id=? AND client_request_id=?", u.UserID, u.ClientRequestID))
	if err == nil {
		if existing.Fingerprint != u.Fingerprint {
			return u, ErrConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return u, err
	}
	var recent int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_uploads_v2 WHERE user_id=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 HOUR)", u.UserID).Scan(&recent); err != nil {
		return u, err
	}
	if recent >= 100 {
		return u, ErrRateLimited
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO asset_uploads_v2 (upload_id,asset_id,user_id,client_request_id,fingerprint,purpose,business_type,business_ref,object_key,content_type,size_bytes,content_md5,alt_text,status,created_at,expires_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,'AUTHORIZED',UTC_TIMESTAMP(6),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 HOUR))`, u.UploadID, u.AssetID, u.UserID, u.ClientRequestID, u.Fingerprint, u.Purpose, u.BusinessType, u.BusinessRef, u.ObjectKey, u.ContentType, u.SizeBytes, u.MD5, u.AltText)
	if err != nil {
		return u, err
	}
	existing, err = scanAssetV2(tx.QueryRowContext(ctx, assetSelectV2+" WHERE upload_id=?", u.UploadID))
	if err != nil {
		return u, err
	}
	return existing, tx.Commit()
}

func (s *Store) AssetUploadV2(ctx context.Context, userID, uploadID string) (AssetUploadV2, error) {
	u, err := scanAssetV2(s.DB.QueryRowContext(ctx, assetSelectV2+" WHERE upload_id=? AND user_id=?", uploadID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrAssetForbidden
	}
	return u, err
}

func (s *Store) BeginAssetVerificationV2(ctx context.Context, u AssetUploadV2) (bool, error) {
	result, err := s.DB.ExecContext(ctx, `UPDATE asset_uploads_v2 SET status='VERIFYING',verification_started_at=UTC_TIMESTAMP(6) WHERE upload_id=? AND user_id=? AND removed_at IS NULL AND lifecycle_state='ACTIVE' AND expires_at>UTC_TIMESTAMP(6) AND (status='AUTHORIZED' OR (status='VERIFYING' AND verification_started_at<DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 60 SECOND)))`, u.UploadID, u.UserID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s *Store) FinishAssetVerificationV2(ctx context.Context, u AssetUploadV2, v objectstorage.VerifiedImage, verificationError error) error {
	if verificationError != nil {
		status, code := "AUTHORIZED", any(nil)
		if errors.Is(verificationError, objectstorage.ErrInvalidObject) {
			status, code = "REJECTED", "INVALID_IMAGE"
		}
		_, err := s.DB.ExecContext(ctx, "UPDATE asset_uploads_v2 SET status=?,rejection_code=?,verification_started_at=NULL WHERE upload_id=? AND user_id=? AND status='VERIFYING' AND removed_at IS NULL", status, code, u.UploadID, u.UserID)
		return err
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE asset_uploads_v2 SET status='READY',width=?,height=?,sha256=?,verified_at=UTC_TIMESTAMP(6),verification_started_at=NULL WHERE upload_id=? AND user_id=? AND status='VERIFYING' AND removed_at IS NULL AND lifecycle_state='ACTIVE'`, v.Width, v.Height, v.SHA256, u.UploadID, u.UserID)
	return err
}

func (s *Store) ValidateOwnedAssetsV2(ctx context.Context, userID string, ids []string, purpose string) error {
	for _, id := range ids {
		var count int
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_uploads_v2 WHERE asset_id=? AND user_id=? AND purpose=? AND status='READY' AND removed_at IS NULL", id, userID, purpose).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return ErrAssetUnavailable
		}
	}
	return s.EnsureAssetsActiveV2(ctx, userID, "", "", ids)
}

// AssetV2 resolves an owner-approved asset. Public profile/catalog callers must
// first authorize the containing resource and take this owner ID from the DB.
func (s *Store) AssetV2(ctx context.Context, viewerID, assetID string) (map[string]any, error) {
	u, err := scanAssetV2(s.DB.QueryRowContext(ctx, assetSelectV2+" WHERE asset_id=? AND user_id=? AND removed_at IS NULL", assetID, viewerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAssetForbidden
	}
	if err != nil {
		return nil, err
	}
	if u.Status != "READY" {
		return nil, ErrAssetUnavailable
	}
	view := map[string]any{"assetId": u.AssetID, "purpose": u.Purpose, "status": "READY", "url": "", "width": u.Width.Int64, "height": u.Height.Int64, "sizeBytes": u.SizeBytes, "sha256": u.SHA256.String, "altText": u.AltText, "createdAt": u.CreatedAt.UTC(), "contentMd5": u.MD5, "contentType": u.ContentType, "urlExpiresAt": nil, "retentionStatus": u.Lifecycle, "retainUntil": commerceTime(u.RetainUntil)}
	if u.Lifecycle == "EXPIRED" || u.Lifecycle == "PURGED" || (u.RetainUntil.Valid && !u.RetainUntil.Time.After(time.Now())) {
		view["status"] = "EXPIRED"
		return view, nil
	}
	c, err := objectstorage.FromEnvironment()
	if err != nil {
		return nil, err
	}
	link, expires, err := c.Download(ctx, u.ObjectKey)
	if err != nil {
		return nil, err
	}
	view["url"], view["urlExpiresAt"] = link, expires
	return view, nil
}

func (s *Store) RemoveAssetV2(ctx context.Context, userID, assetID string) error {
	tx, err := s.beginAccountTx(ctx, userID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var removed sql.NullTime
	err = tx.QueryRowContext(ctx, "SELECT removed_at FROM asset_uploads_v2 WHERE asset_id=? AND user_id=? FOR UPDATE", assetID, userID).Scan(&removed)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAssetForbidden
	}
	if err != nil {
		return err
	}
	if removed.Valid {
		return nil
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_bindings_v2 WHERE asset_id=?", assetID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrAssetInUse
	}
	if _, err = tx.ExecContext(ctx, "UPDATE asset_uploads_v2 SET removed_at=UTC_TIMESTAMP(6) WHERE asset_id=?", assetID); err != nil {
		return err
	}
	return tx.Commit()
}

// SetAssetBindingsV2 runs in the caller's business transaction. It prevents
// concurrent remove from invalidating an avatar/product/snapshot mid-write.
func (s *Store) SetAssetBindingsV2(ctx context.Context, tx *sql.Tx, userID, businessType, businessRef string, ids []string) error {
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	for _, id := range ordered {
		var owner, status, lifecycle string
		var removed sql.NullTime
		err := tx.QueryRowContext(ctx, "SELECT user_id,status,removed_at,lifecycle_state FROM asset_uploads_v2 WHERE asset_id=? FOR UPDATE", id).Scan(&owner, &status, &removed, &lifecycle)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAssetUnavailable
		}
		if err != nil {
			return err
		}
		if status != "READY" || removed.Valid || lifecycle != "ACTIVE" {
			return ErrAssetUnavailable
		}
		if owner != userID {
			var alreadyBound int
			if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_bindings_v2 WHERE asset_id=? AND business_type=? AND business_ref=?", id, businessType, businessRef).Scan(&alreadyBound); err != nil {
				return err
			}
			if alreadyBound != 1 {
				return ErrAssetUnavailable
			}
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM asset_bindings_v2 WHERE business_type=? AND business_ref=?", businessType, businessRef); err != nil {
		return err
	}
	for _, id := range ordered {
		if _, err := tx.ExecContext(ctx, "UPDATE asset_uploads_v2 SET was_bound=TRUE WHERE asset_id=?", id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO asset_bindings_v2 (asset_id,business_type,business_ref) VALUES (?,?,?)", id, businessType, businessRef); err != nil {
			return err
		}
	}
	return nil
}

// PromoteAssetBindingsV2 preserves approved multi-author media when a draft is
// published. The caller must authorize the source business record in this same
// transaction. Only that exact draft's existing bindings can be promoted.
func (s *Store) PromoteAssetBindingsV2(ctx context.Context, tx *sql.Tx, sourceType, targetType, businessRef string, ids []string) error {
	return s.CopyAssetBindingsV2(ctx, tx, sourceType, businessRef, targetType, businessRef, ids)
}

// CopyAssetBindingsV2 pins media in an authorized immutable order/commission
// snapshot without giving the buyer access to unrelated uploads of its author.
func (s *Store) CopyAssetBindingsV2(ctx context.Context, tx *sql.Tx, sourceType, sourceRef, targetType, targetRef string, ids []string) error {
	return s.copyAssetBindingsV2(ctx, tx, sourceType, sourceRef, targetType, targetRef, ids, true)
}

func (s *Store) AppendAssetBindingsV2(ctx context.Context, tx *sql.Tx, sourceType, sourceRef, targetType, targetRef string, ids []string) error {
	return s.copyAssetBindingsV2(ctx, tx, sourceType, sourceRef, targetType, targetRef, ids, false)
}

func (s *Store) copyAssetBindingsV2(ctx context.Context, tx *sql.Tx, sourceType, sourceRef, targetType, targetRef string, ids []string, replace bool) error {
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	for _, id := range ordered {
		var status, lifecycle string
		var removed sql.NullTime
		if err := tx.QueryRowContext(ctx, "SELECT status,removed_at,lifecycle_state FROM asset_uploads_v2 WHERE asset_id=? FOR UPDATE", id).Scan(&status, &removed, &lifecycle); err != nil {
			return ErrAssetUnavailable
		}
		if status != "READY" || removed.Valid || lifecycle != "ACTIVE" {
			return ErrAssetUnavailable
		}
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_bindings_v2 WHERE asset_id=? AND business_type=? AND business_ref=?", id, sourceType, sourceRef).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return ErrAssetForbidden
		}
	}
	if replace {
		if _, err := tx.ExecContext(ctx, "DELETE FROM asset_bindings_v2 WHERE business_type=? AND business_ref=?", targetType, targetRef); err != nil {
			return err
		}
	}
	for _, id := range ordered {
		if _, err := tx.ExecContext(ctx, "UPDATE asset_uploads_v2 SET was_bound=TRUE WHERE asset_id=?", id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO asset_bindings_v2(asset_id,business_type,business_ref) VALUES(?,?,?)", id, targetType, targetRef); err != nil {
			return err
		}
	}
	return nil
}

// AssetForBindingV2 resolves media only after its containing business resource
// was authorized by the caller. It never accepts an owner supplied by a client.
func (s *Store) AssetForBindingV2(ctx context.Context, businessType, businessRef, assetID string) (map[string]any, error) {
	var owner string
	err := s.DB.QueryRowContext(ctx, `SELECT a.user_id FROM asset_uploads_v2 a JOIN asset_bindings_v2 b ON b.asset_id=a.asset_id WHERE b.asset_id=? AND b.business_type=? AND b.business_ref=? AND a.status='READY' AND a.removed_at IS NULL`, assetID, businessType, businessRef).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAssetForbidden
	}
	if err != nil {
		return nil, err
	}
	return s.AssetV2(ctx, owner, assetID)
}
