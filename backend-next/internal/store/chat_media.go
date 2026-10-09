package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
)

const ChatMediaRetention = 72 * time.Hour
const ChatHistoryDays = 30
const ChatStickerLimit = 10

var ErrStickerLimit = errors.New("sticker library full")
var ErrMediaExpired = errors.New("chat media expired")

type ChatMediaInput struct {
	ClientRequestID string `json:"clientRequestId"`
	SHA256          string `json:"sha256"`
	ThumbnailSHA256 string `json:"thumbnailSha256"`
	MD5             string `json:"contentMd5"`
	ThumbnailMD5    string `json:"thumbnailMd5"`
	SizeBytes       int64  `json:"sizeBytes"`
	ThumbnailBytes  int64  `json:"thumbnailBytes"`
}

func (in ChatMediaInput) Valid() bool {
	validHash := func(v string) bool {
		b, e := hex.DecodeString(v)
		return e == nil && len(b) == 32 && hex.EncodeToString(b) == v
	}
	validMD5 := func(v string) bool {
		b, e := base64.StdEncoding.DecodeString(v)
		return e == nil && len(b) == 16 && base64.StdEncoding.EncodeToString(b) == v
	}
	return ValidSocialID(in.ClientRequestID) && validHash(in.SHA256) && validHash(in.ThumbnailSHA256) && validMD5(in.MD5) && validMD5(in.ThumbnailMD5) && in.SizeBytes > 0 && in.SizeBytes <= 2<<20 && in.ThumbnailBytes > 0 && in.ThumbnailBytes <= 80<<10
}

type ChatMedia struct {
	ID              string     `json:"mediaId"`
	SHA256          string     `json:"sha256"`
	ThumbnailSHA256 string     `json:"thumbnailSha256"`
	SizeBytes       int64      `json:"sizeBytes"`
	ThumbnailBytes  int64      `json:"thumbnailBytes"`
	Width           int        `json:"width"`
	Height          int        `json:"height"`
	Animated        bool       `json:"animated"`
	FrameCount      int        `json:"frameCount"`
	DurationMS      int        `json:"durationMs"`
	Type            string     `json:"type,omitempty"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	State           string     `json:"-"`
	MD5             string     `json:"-"`
	ThumbnailMD5    string     `json:"-"`
	CreatedAt       time.Time  `json:"-"`
}

type ChatMediaSend struct {
	ID   string `json:"mediaId"`
	Type string `json:"type"`
}

func (m *ChatMediaSend) Valid() bool {
	return m != nil && ValidSocialID(m.ID) && (m.Type == "image" || m.Type == "sticker")
}
func (m *ChatMediaSend) Summary() string {
	if m.Type == "sticker" {
		return "[贴纸]"
	}
	return "[图片]"
}

type ChatMediaUpload struct {
	ID        string    `json:"uploadId"`
	Media     ChatMedia `json:"media"`
	ExpiresAt time.Time `json:"expiresAt"`
	StagedID  string    `json:"-"`
}

const mediaColumns = `media_id,sha256,thumbnail_sha256,size_bytes,thumbnail_bytes,width,height,animated,frame_count,duration_ms,state,content_md5,thumbnail_md5,created_at`

func scanChatMedia(row interface{ Scan(...any) error }) (m ChatMedia, err error) {
	err = row.Scan(&m.ID, &m.SHA256, &m.ThumbnailSHA256, &m.SizeBytes, &m.ThumbnailBytes, &m.Width, &m.Height, &m.Animated, &m.FrameCount, &m.DurationMS, &m.State, &m.MD5, &m.ThumbnailMD5, &m.CreatedAt)
	return
}
func mediaHash(in ChatMediaInput) string { return in.SHA256 }

// A hash is not an access capability. Reuse without uploading requires an owned
// grant or a currently visible public message. Other bytes must first be verified.
func chatMediaAllowed(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, userID, id string) (bool, error) {
	var ok bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chat_stickers WHERE user_id=? AND media_id=?) OR EXISTS(SELECT 1 FROM chat_media_uploads WHERE user_id=? AND media_id=? AND expires_at>UTC_TIMESTAMP(6)) OR EXISTS(SELECT 1 FROM public_chat_media p JOIN chat_messages_next m ON m.message_id=p.message_id WHERE p.media_id=? AND p.expires_at>UTC_TIMESTAMP(6))`, userID, id, userID, id, id).Scan(&ok)
	return ok, err
}

func (s *Store) CreateChatMediaUpload(ctx context.Context, userID string, in ChatMediaInput) (ChatMediaUpload, error) {
	if !in.Valid() {
		return ChatMediaUpload{}, ErrSocialInvalid
	}
	tx, err := s.beginAccountTx(ctx, userID)
	if err != nil {
		return ChatMediaUpload{}, err
	}
	defer tx.Rollback()
	var lock string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? FOR UPDATE", userID).Scan(&lock); err != nil {
		return ChatMediaUpload{}, err
	}
	raw, _ := json.Marshal(in)
	fingerprint := Digest(raw)
	var upload ChatMediaUpload
	var existing string
	err = tx.QueryRowContext(ctx, `SELECT upload_id,media_id,staged_media_id,expires_at,fingerprint FROM chat_media_uploads WHERE user_id=? AND client_request_id=?`, userID, in.ClientRequestID).Scan(&upload.ID, &upload.Media.ID, &upload.StagedID, &upload.ExpiresAt, &existing)
	if err == nil {
		if existing != fingerprint {
			return upload, ErrConflict
		}
		if !upload.ExpiresAt.After(time.Now()) {
			return upload, ErrMediaExpired
		}
		upload.Media, err = scanChatMedia(tx.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE media_id=?", upload.Media.ID))
		return upload, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return upload, err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_media_uploads WHERE user_id=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY)`, userID).Scan(&count); err != nil {
		return upload, err
	}
	if count >= 200 {
		return upload, ErrRateLimited
	}
	canonical, err := scanChatMedia(tx.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE canonical_hash=? AND state='READY' FOR UPDATE", mediaHash(in)))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return upload, err
	}
	if err == nil {
		ok, e := chatMediaAllowed(ctx, tx, userID, canonical.ID)
		if e != nil {
			return upload, e
		}
		if ok {
			upload.Media = canonical
		}
	}
	if upload.Media.ID == "" {
		upload.Media = ChatMedia{ID: ID("cm_"), SHA256: in.SHA256, ThumbnailSHA256: in.ThumbnailSHA256, MD5: in.MD5, ThumbnailMD5: in.ThumbnailMD5, SizeBytes: in.SizeBytes, ThumbnailBytes: in.ThumbnailBytes, State: "UPLOADING", CreatedAt: time.Now().UTC()}
		_, err = tx.ExecContext(ctx, `INSERT INTO chat_media_objects(media_id,sha256,thumbnail_sha256,content_md5,thumbnail_md5,size_bytes,thumbnail_bytes,created_at) VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, upload.Media.ID, in.SHA256, in.ThumbnailSHA256, in.MD5, in.ThumbnailMD5, in.SizeBytes, in.ThumbnailBytes)
		if err != nil {
			return upload, err
		}
	} else if upload.Media.SizeBytes != in.SizeBytes || upload.Media.MD5 != in.MD5 {
		return upload, ErrSocialInvalid
	}
	upload.ID = ID("cu_")
	upload.StagedID = upload.Media.ID
	upload.ExpiresAt = time.Now().UTC().Add(ChatMediaRetention)
	_, err = tx.ExecContext(ctx, `INSERT INTO chat_media_uploads VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6),?)`, upload.ID, userID, in.ClientRequestID, fingerprint, upload.Media.ID, upload.StagedID, upload.ExpiresAt)
	if err != nil {
		return upload, err
	}
	return upload, tx.Commit()
}

func (s *Store) ChatMediaUpload(ctx context.Context, userID, id string) (u ChatMediaUpload, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT upload_id,media_id,staged_media_id,expires_at FROM chat_media_uploads WHERE upload_id=? AND user_id=?`, id, userID).Scan(&u.ID, &u.Media.ID, &u.StagedID, &u.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrAssetForbidden
	}
	if err != nil {
		return
	}
	if !u.ExpiresAt.After(time.Now()) {
		return u, ErrMediaExpired
	}
	u.Media, err = scanChatMedia(s.DB.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE media_id=?", u.Media.ID))
	return
}

func (s *Store) CompleteChatMedia(ctx context.Context, userID, uploadID string, v objectstorage.ChatImage) (ChatMediaUpload, error) {
	tx, err := s.beginAccountTx(ctx, userID)
	if err != nil {
		return ChatMediaUpload{}, err
	}
	defer tx.Rollback()
	var u ChatMediaUpload
	err = tx.QueryRowContext(ctx, `SELECT upload_id,media_id,staged_media_id,expires_at FROM chat_media_uploads WHERE upload_id=? AND user_id=? FOR UPDATE`, uploadID, userID).Scan(&u.ID, &u.Media.ID, &u.StagedID, &u.ExpiresAt)
	if err != nil {
		return u, socialMissing(err)
	}
	if !u.ExpiresAt.After(time.Now()) {
		return u, ErrMediaExpired
	}
	u.Media, err = scanChatMedia(tx.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE media_id=? FOR UPDATE", u.Media.ID))
	if err != nil {
		return u, err
	}
	if u.Media.State == "READY" {
		return u, nil
	}
	if u.Media.State != "UPLOADING" {
		return u, ErrMediaExpired
	}
	if v.SHA256 != u.Media.SHA256 {
		return u, ErrSocialInvalid
	}
	hash := u.Media.SHA256
	_, err = tx.ExecContext(ctx, `UPDATE chat_media_objects SET canonical_hash=?,state='READY',width=?,height=?,animated=?,frame_count=?,duration_ms=? WHERE media_id=?`, hash, v.Width, v.Height, v.Animated, v.Frames, v.DurationMS, u.Media.ID)
	if duplicate(err) {
		canonical, e := scanChatMedia(tx.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE canonical_hash=? AND state='READY' FOR UPDATE", hash))
		if e != nil {
			return u, e
		}
		// The staged copy stays tracked until its 72h upload lease expires. This
		// also prevents a still-valid presigned PUT from recreating an orphan.
		_, err = tx.ExecContext(ctx, "UPDATE chat_media_uploads SET media_id=? WHERE upload_id=?", canonical.ID, u.ID)
		u.Media = canonical
	} else if err == nil {
		u.Media.Width = v.Width
		u.Media.Height = v.Height
		u.Media.Animated = v.Animated
		u.Media.FrameCount = v.Frames
		u.Media.DurationMS = v.DurationMS
		u.Media.State = "READY"
	}
	if err != nil {
		return u, err
	}
	return u, tx.Commit()
}

func (s *Store) ChatStickers(ctx context.Context, userID string) ([]ChatMedia, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE media_id IN (SELECT media_id FROM chat_stickers WHERE user_id=?) AND state='READY' ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatMedia{}
	for rows.Next() {
		m, e := scanChatMedia(rows)
		if e != nil {
			return nil, e
		}
		m.Type = "sticker"
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) SaveChatSticker(ctx context.Context, userID, id string) (ChatMedia, error) {
	tx, err := s.beginAccountTx(ctx, userID)
	if err != nil {
		return ChatMedia{}, err
	}
	defer tx.Rollback()
	var lock string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? FOR UPDATE", userID).Scan(&lock); err != nil {
		return ChatMedia{}, err
	}
	m, err := scanChatMedia(tx.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE media_id=? FOR UPDATE", id))
	if err != nil {
		return m, socialMissing(err)
	}
	if m.State != "READY" {
		return m, ErrMediaExpired
	}
	if m.Width > 512 || m.Height > 512 || !m.Animated && m.SizeBytes > 512<<10 {
		return m, ErrSocialInvalid
	}
	ok, err := chatMediaAllowed(ctx, tx, userID, id)
	if err != nil {
		return m, err
	}
	if !ok {
		return m, ErrAssetForbidden
	}
	var count, exists int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(media_id=?),0) FROM chat_stickers WHERE user_id=?`, id, userID).Scan(&count, &exists); err != nil {
		return m, err
	}
	if exists == 0 && count >= ChatStickerLimit {
		return m, ErrStickerLimit
	}
	_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO chat_stickers VALUES(?,?,UTC_TIMESTAMP(6))`, userID, id)
	if err != nil {
		return m, err
	}
	m.Type = "sticker"
	return m, tx.Commit()
}

func (s *Store) DeleteChatSticker(ctx context.Context, userID, id string) error {
	tx, err := s.beginAccountTx(ctx, userID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var lock string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? FOR UPDATE", userID).Scan(&lock); err != nil {
		return err
	}
	// Lock the object before removing its durable reference, as sending and GC do.
	if err = tx.QueryRowContext(ctx, "SELECT media_id FROM chat_media_objects WHERE media_id=? FOR UPDATE", id).Scan(&lock); errors.Is(err, sql.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM chat_stickers WHERE user_id=? AND media_id=?", userID, id); err != nil {
		return err
	}
	return tx.Commit()
}

func chatMediaForSend(ctx context.Context, tx *sql.Tx, userID string, in *ChatMediaSend) (ChatMedia, error) {
	if !in.Valid() {
		return ChatMedia{}, ErrSocialInvalid
	}
	m, err := scanChatMedia(tx.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE media_id=? FOR UPDATE", in.ID))
	if err != nil {
		return m, socialMissing(err)
	}
	if m.State != "READY" {
		return m, ErrMediaExpired
	}
	ok, err := chatMediaAllowed(ctx, tx, userID, in.ID)
	if err != nil {
		return m, err
	}
	if !ok {
		return m, ErrAssetForbidden
	}
	if in.Type == "sticker" && (m.Width > 512 || m.Height > 512 || !m.Animated && m.SizeBytes > 512<<10) {
		return m, ErrSocialInvalid
	}
	m.Type = in.Type
	until := time.Now().UTC().Add(ChatMediaRetention)
	m.ExpiresAt = &until
	return m, nil
}

// For a message, its own deadline is authoritative even if another message or
// sticker keeps the underlying object alive. Clients may still use local bytes.
func (s *Store) ChatMediaAccess(ctx context.Context, userID, id, messageID string) (ChatMedia, time.Time, error) {
	m, err := scanChatMedia(s.DB.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM chat_media_objects WHERE media_id=?", id))
	if err != nil {
		return m, time.Time{}, socialMissing(err)
	}
	deadline := time.Now().UTC().Add(15 * time.Minute)
	if messageID != "" {
		var expires time.Time
		err = s.DB.QueryRowContext(ctx, `SELECT p.expires_at FROM public_chat_media p JOIN chat_messages_next m ON m.message_id=p.message_id WHERE p.media_id=? AND p.message_id=? AND m.sent_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 30 DAY)`, id, messageID).Scan(&expires)
		if err != nil {
			return m, deadline, socialMissing(err)
		}
		if !expires.After(time.Now()) {
			return m, deadline, ErrMediaExpired
		}
		if expires.Before(deadline) {
			deadline = expires
		}
	} else {
		var sticker bool
		var lease sql.NullTime
		err = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chat_stickers WHERE user_id=? AND media_id=?),(SELECT MAX(expires_at) FROM chat_media_uploads WHERE user_id=? AND media_id=?)`, userID, id, userID, id).Scan(&sticker, &lease)
		if err != nil {
			return m, deadline, err
		}
		if !sticker && (!lease.Valid || !lease.Time.After(time.Now())) {
			return m, deadline, ErrAssetForbidden
		}
		if !sticker && lease.Time.Before(deadline) {
			deadline = lease.Time
		}
	}
	if m.State != "READY" {
		return m, deadline, ErrMediaExpired
	}
	return m, deadline, nil
}

type ChatMediaObjects interface {
	Delete(context.Context, string) error
}

func ChatMediaKey(prefix, id string, thumbnail bool) string {
	if thumbnail {
		return prefix + "chat-media/" + id + "/preview.webp"
	}
	return prefix + "chat-media/" + id + "/image.webp"
}

// Mark before I/O; a committed DELETING row cannot acquire new references. Each
// object generation uses a fresh key, so retries never delete replacement bytes.
func (s *Store) RunChatMediaGC(ctx context.Context, objects ChatMediaObjects, prefix string, limit int) (int, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT media_id FROM chat_media_objects o WHERE state='DELETING' OR (state IN ('READY','UPLOADING') AND created_at<=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 72 HOUR) AND NOT EXISTS(SELECT 1 FROM chat_stickers s WHERE s.media_id=o.media_id) AND NOT EXISTS(SELECT 1 FROM chat_media_uploads u WHERE u.media_id=o.media_id AND u.expires_at>UTC_TIMESTAMP(6)) AND NOT EXISTS(SELECT 1 FROM public_chat_media p WHERE p.media_id=o.media_id AND p.expires_at>UTC_TIMESTAMP(6))) ORDER BY created_at LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, id := range ids {
		tx, e := s.beginAccountTx(ctx, "")
		if e != nil {
			return count, e
		}
		var state string
		e = tx.QueryRowContext(ctx, "SELECT state FROM chat_media_objects WHERE media_id=? FOR UPDATE", id).Scan(&state)
		if e != nil {
			tx.Rollback()
			return count, e
		}
		if state != "DELETING" {
			var used bool
			e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chat_stickers WHERE media_id=?) OR EXISTS(SELECT 1 FROM chat_media_uploads WHERE media_id=? AND expires_at>UTC_TIMESTAMP(6)) OR EXISTS(SELECT 1 FROM public_chat_media WHERE media_id=? AND expires_at>UTC_TIMESTAMP(6))`, id, id, id).Scan(&used)
			if e != nil {
				tx.Rollback()
				return count, e
			}
			if used || state == "PURGED" {
				tx.Rollback()
				continue
			}
			_, e = tx.ExecContext(ctx, "UPDATE chat_media_objects SET state='DELETING',canonical_hash=NULL WHERE media_id=?", id)
		}
		if e != nil {
			tx.Rollback()
			return count, e
		}
		if e = tx.Commit(); e != nil {
			return count, e
		}
		if e = objects.Delete(ctx, ChatMediaKey(prefix, id, false)); e != nil {
			return count, e
		}
		if e = objects.Delete(ctx, ChatMediaKey(prefix, id, true)); e != nil {
			return count, e
		}
		if _, e = s.DB.ExecContext(ctx, "UPDATE chat_media_objects SET state='PURGED' WHERE media_id=? AND state='DELETING'", id); e != nil {
			return count, e
		}
		count++
	}
	return count, nil
}
