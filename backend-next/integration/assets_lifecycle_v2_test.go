//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

type lifecycleObject struct {
	data    []byte
	created time.Time
}
type lifecycleStorage struct {
	mu                 sync.Mutex
	objects            map[string]lifecycleObject
	copies             int
	failCopyAfterWrite bool
	failDelete         bool
	started            chan struct{}
	resume             chan struct{}
}

func (f *lifecycleStorage) Exists(_ context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[key]
	return ok, nil
}
func (f *lifecycleStorage) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failDelete {
		f.failDelete = false
		return objectstorage.ErrUnavailable
	}
	delete(f.objects, key)
	return nil
}
func (f *lifecycleStorage) CopyVerified(ctx context.Context, source, target, sha string) (time.Time, error) {
	if f.started != nil {
		close(f.started)
		select {
		case <-ctx.Done():
			return time.Time{}, ctx.Err()
		case <-f.resume:
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.objects[target]; ok {
		if store.Digest(existing.data) != sha {
			return time.Time{}, objectstorage.ErrInvalidObject
		}
		return existing.created, nil
	}
	src, ok := f.objects[source]
	if !ok {
		return time.Time{}, objectstorage.ErrUnavailable
	}
	if store.Digest(src.data) != sha {
		return time.Time{}, objectstorage.ErrInvalidObject
	}
	targetObject := lifecycleObject{append([]byte(nil), src.data...), time.Now().UTC().Truncate(time.Second)}
	f.objects[target] = targetObject
	f.copies++
	if f.failCopyAfterWrite {
		f.failCopyAfterWrite = false
		return time.Time{}, objectstorage.ErrUnavailable
	}
	return targetObject.created, nil
}

func lifecycleAsset(t *testing.T, s *store.Store, userID, name, purpose string) (store.AssetUploadV2, *lifecycleStorage) {
	t.Helper()
	ctx := context.Background()
	data := []byte("verified fixture image " + name)
	u, err := s.CreateAssetUploadV2(ctx, store.AssetUploadV2{UploadID: "upload_" + name, AssetID: "asset_" + name, UserID: userID, ClientRequestID: name, Fingerprint: store.Digest(data), Purpose: purpose, BusinessType: "PROFILE", ObjectKey: "deuterium-test/uploads/" + name + ".png", ContentType: "image/png", MD5: "1B2M2Y8AsgTpgAmY7PhCfg==", SizeBytes: int64(len(data))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.BeginAssetVerificationV2(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err = s.FinishAssetVerificationV2(ctx, u, objectstorage.VerifiedImage{Width: 2, Height: 2, SHA256: store.Digest(data)}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE asset_uploads_v2 SET created_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 5 DAY),expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 4 DAY) WHERE asset_id=?", u.AssetID); err != nil {
		t.Fatal(err)
	}
	f := &lifecycleStorage{objects: map[string]lifecycleObject{u.ObjectKey: {data, time.Now().Add(-5 * 24 * time.Hour)}}}
	return u, f
}
func bindLifecycle(t *testing.T, s *store.Store, user, kind, ref string, ids ...string) {
	t.Helper()
	tx, err := s.DB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = s.SetAssetBindingsV2(context.Background(), tx, user, kind, ref, ids); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
func lifecycleCommerce(t *testing.T, s *store.Store, user, ref, kind, state, funds, asset string) string {
	t.Helper()
	snapshot := "snapshot_" + ref
	body, _ := json.Marshal(map[string]any{"orderNo": "history " + ref, "items": []any{map[string]any{"title": "交易文字永久保留", "photoAssetIds": []string{asset}}}, "content": map[string]any{"coverAssetId": asset}})
	_, err := s.DB.Exec("INSERT INTO commerce_resources_v2(resource_id,resource_kind,channel,owner_id,owner_uuid,escrow_ref,amount,settled_amount,refunded_amount,state,funds_state,body,snapshot_id,snapshot_sha256,version,refund_attempts,automatic,created_at,updated_at) VALUES(?,?,?,?,?,?,'1.00','1.00','0.00',?,?,?,?,?,1,0,FALSE,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))", ref, kind, kind, user, "00000000-0000-0000-0000-000000000001", "escrow_"+ref, state, funds, string(body), snapshot, store.Digest(body))
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.DB.Exec("INSERT INTO commerce_snapshots_v2(snapshot_id,resource_id,resource_kind,resource_version,body,sha256,created_at) VALUES(?,?,?,1,?,?,UTC_TIMESTAMP(6))", snapshot, ref, kind, string(body), store.Digest(body))
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
func lifecycleCatalog(t *testing.T, s *store.Store, user, id, state, asset string) {
	t.Helper()
	_, err := s.DB.Exec("INSERT INTO catalog_records_v2(resource_id,kind,owner_id,version,body,state,available_stock,created_at,updated_at) VALUES(?,'listing',?,1,'{}',?,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))", id, user, state)
	if err != nil {
		t.Fatal(err)
	}
	bindLifecycle(t, s, user, "MARKET_LISTING", id, asset)
}
func lifecyclePhase(t *testing.T, s *store.Store, id string) (string, string, sql.NullTime) {
	t.Helper()
	var phase, key string
	var until sql.NullTime
	if err := s.DB.QueryRow("SELECT lifecycle_state,object_key,retain_until FROM asset_uploads_v2 WHERE asset_id=?", id).Scan(&phase, &key, &until); err != nil {
		t.Fatal(err)
	}
	return phase, key, until
}
func TestAssetLifecycleSharesLiveAndHistoricalImagesUntilLastCurrentUseEnds(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, objects := lifecycleAsset(t, s, user.ID, "shared", "MARKET_PHOTO")
	lifecycleCatalog(t, s, user.ID, "listing_shared", "ACTIVE", u.AssetID)
	for _, id := range []string{"history_one", "history_two"} {
		snapshot := lifecycleCommerce(t, s, user.ID, id, "ORDER", "CONFIRMED", "SETTLED", u.AssetID)
		bindLifecycle(t, s, user.ID, "ORDER_SNAPSHOT", snapshot, u.AssetID)
	}
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	if phase, _, _ := lifecyclePhase(t, s, u.AssetID); phase != "ACTIVE" || objects.copies != 0 {
		t.Fatal("a sale expired the live shared image")
	}
	if _, err := s.DB.Exec("UPDATE catalog_records_v2 SET state='UNLISTED' WHERE resource_id='listing_shared'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	phase, key, until := lifecyclePhase(t, s, u.AssetID)
	if phase != "RETIRED" || !strings.HasPrefix(key, "deuterium-test/gc/") || objects.copies != 1 {
		t.Fatalf("shared asset should move once: %s %s copies %d", phase, key, objects.copies)
	}
	if !until.Valid || time.Until(until.Time) < 29*24*time.Hour {
		t.Fatal("retention inherited original object's old age")
	}
	var bindings int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM asset_bindings_v2 WHERE asset_id=?", u.AssetID).Scan(&bindings); err != nil || bindings != 3 {
		t.Fatal("history bindings must preserve the same asset identity", err, bindings)
	}
}
func TestAssetLifecycleKeepsPendingCommissionAndUnknownFunds(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, objects := lifecycleAsset(t, s, user.ID, "commission", "COMMISSION_COVER")
	snapshot := lifecycleCommerce(t, s, user.ID, "commission_pending", "COMMISSION", "COMPLETED", "HELD", u.AssetID)
	bindLifecycle(t, s, user.ID, "COMMISSION", "commission_pending", u.AssetID)
	bindLifecycle(t, s, user.ID, "COMMISSION_SNAPSHOT", snapshot, u.AssetID)
	for _, funds := range []string{"HELD", "UNKNOWN", "INTERVENTION_HOLD"} {
		if _, err := s.DB.Exec("UPDATE commerce_resources_v2 SET funds_state=? WHERE resource_id='commission_pending'", funds); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
			t.Fatal(err)
		}
		if phase, _, _ := lifecyclePhase(t, s, u.AssetID); phase != "ACTIVE" {
			t.Fatal("active/uncertain commission expired", funds)
		}
	}
	if _, err := s.DB.Exec("UPDATE commerce_resources_v2 SET state='CONFIRMED',funds_state='SETTLED' WHERE resource_id='commission_pending'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	if phase, _, _ := lifecyclePhase(t, s, u.AssetID); phase != "RETIRED" {
		t.Fatal("finished commission still pins original forever")
	}
}
func TestAssetLifecycleCopyTimeoutAndDeleteFailureResumeWithoutResettingAge(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, objects := lifecycleAsset(t, s, user.ID, "retry", "AVATAR")
	objects.failCopyAfterWrite = true
	if result, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil || result.Deferred != 1 {
		t.Fatal(result, err)
	}
	if phase, key, _ := lifecyclePhase(t, s, u.AssetID); phase != "MOVING" || key != u.ObjectKey {
		t.Fatal("unknown copy replaced source prematurely")
	}
	if ok, _ := objects.Exists(ctx, u.ObjectKey); !ok {
		t.Fatal("original lost after copy timeout")
	}
	objects.failDelete = true
	if _, err := s.DB.Exec("UPDATE asset_uploads_v2 SET lifecycle_retry_at=NULL"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	phase, key, until := lifecyclePhase(t, s, u.AssetID)
	if phase != "RETIRED" {
		t.Fatal("verified copy not committed", phase)
	}
	if _, err := s.DB.Exec("UPDATE asset_uploads_v2 SET lifecycle_retry_at=NULL"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	_, same, after := lifecyclePhase(t, s, u.AssetID)
	if objects.copies != 1 || key != same || !until.Time.Equal(after.Time) {
		t.Fatal("retry duplicated copy or reset retention")
	}
	if ok, _ := objects.Exists(ctx, u.ObjectKey); ok {
		t.Fatal("unreferenced original was not removed after retry")
	}
}
func TestAssetLifecycleRestoreLeavesExpirationPrefixBeforeRebinding(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, objects := lifecycleAsset(t, s, user.ID, "restore", "AVATAR")
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	_, retired, _ := lifecyclePhase(t, s, u.AssetID)
	if err := s.RestoreAssetLifecycleV2(ctx, objects, "deuterium-test/", "another_user", "", "", u.AssetID); !errors.Is(err, store.ErrAssetForbidden) {
		t.Fatal("other user restored private image", err)
	}
	if err := s.RestoreAssetLifecycleV2(ctx, objects, "deuterium-test/", user.ID, "", "", u.AssetID); err != nil {
		t.Fatal(err)
	}
	phase, active, until := lifecyclePhase(t, s, u.AssetID)
	if phase != "ACTIVE" || !strings.HasPrefix(active, "deuterium-test/uploads/") || until.Valid {
		t.Fatal("restored image remained expirable")
	}
	if ok, _ := objects.Exists(ctx, retired); ok {
		t.Fatal("old retirement key remains after restore")
	}
	bindLifecycle(t, s, user.ID, "PROFILE", user.ID, u.AssetID)
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	if phase, _, _ := lifecyclePhase(t, s, u.AssetID); phase != "ACTIVE" {
		t.Fatal("restored profile expired again")
	}
}
func TestAssetLifecycleNewBindingCannotRaceWithPhysicalMove(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, objects := lifecycleAsset(t, s, user.ID, "race", "AVATAR")
	objects.started = make(chan struct{})
	objects.resume = make(chan struct{})
	done := make(chan error, 1)
	go func() { _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); done <- err }()
	select {
	case <-objects.started:
	case <-time.After(5 * time.Second):
		t.Fatal("retirement did not start")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.SetAssetBindingsV2(ctx, tx, user.ID, "PROFILE", user.ID, []string{u.AssetID})
	tx.Rollback()
	close(objects.resume)
	if !errors.Is(err, store.ErrAssetUnavailable) {
		t.Fatal("new reference raced into expiring key", err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}
func TestAssetLifecycleExpiryPreservesOrderDetailsAndAssetIdentity(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, objects := lifecycleAsset(t, s, user.ID, "expired", "MARKET_PHOTO")
	snapshot := lifecycleCommerce(t, s, user.ID, "history_expired", "ORDER", "CONFIRMED", "SETTLED", u.AssetID)
	bindLifecycle(t, s, user.ID, "ORDER_SNAPSHOT", snapshot, u.AssetID)
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	_, key, _ := lifecyclePhase(t, s, u.AssetID)
	if _, err := s.DB.Exec("UPDATE asset_uploads_v2 SET retain_until=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY) WHERE asset_id=?", u.AssetID); err != nil {
		t.Fatal(err)
	}
	if err := objects.Delete(ctx, key); err != nil {
		t.Fatal(err)
	} // Simulate provider lifecycle deletion.
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	view, err := s.CommerceViewV2(ctx, user.ID, "history_expired", false)
	if err != nil {
		t.Fatal("expired photo broke order", err)
	}
	if view["orderNo"] != "history history_expired" || view["amount"] != "1.00" {
		t.Fatal("order metadata lost")
	}
	images := view["images"].([]map[string]any)
	if len(images) != 1 || images[0]["assetId"] != u.AssetID || images[0]["status"] != "EXPIRED" || images[0]["url"] != "" {
		t.Fatal("image did not gracefully expire", images)
	}
	if err := s.RestoreAssetLifecycleV2(ctx, objects, "deuterium-test/", user.ID, "", "", u.AssetID); !errors.Is(err, store.ErrAssetUnavailable) {
		t.Fatal("purged image was restored", err)
	}
}
func TestAssetLifecycleNewUnsubmittedUploadGetsGraceButDetachedAvatarCanRetire(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, objects := lifecycleAsset(t, s, user.ID, "grace", "AVATAR")
	if _, err := s.DB.Exec("UPDATE asset_uploads_v2 SET created_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 2 HOUR) WHERE asset_id=?", u.AssetID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	if phase, _, _ := lifecyclePhase(t, s, u.AssetID); phase != "ACTIVE" {
		t.Fatal("editing upload expired too soon")
	}
	bindLifecycle(t, s, user.ID, "PROFILE", user.ID, u.AssetID)
	bindLifecycle(t, s, user.ID, "PROFILE", user.ID)
	if _, err := s.RunAssetLifecycleV2(ctx, objects, "deuterium-test/", 32); err != nil {
		t.Fatal(err)
	}
	if phase, _, _ := lifecyclePhase(t, s, u.AssetID); phase != "RETIRED" {
		t.Fatal("replaced avatar remained active")
	}
}
