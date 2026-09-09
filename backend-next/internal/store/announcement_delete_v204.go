package store

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Delete the announcement itself, including cached mutation bodies and copies in
// notifications. Retain only identifiers, audit facts and replay tombstones.
func (s *Store) DeleteAnnouncementV204(ctx context.Context, actor, id, key string, version int64) (json.RawMessage, error) {
	if !ValidSocialID(id) || version < 1 {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "announcement.delete", key, struct {
		ID      string
		Version int64
	}{id, version}, func(tx *sql.Tx) (any, error) {
		var actual int64
		if err := tx.QueryRowContext(ctx, `SELECT version FROM social_announcements_v2 WHERE announcement_id=? FOR UPDATE`, id).Scan(&actual); err != nil {
			return nil, socialMissing(err)
		}
		if actual != version {
			return nil, ErrSocialVersion
		}
		if err := s.SetAssetBindingsV2(ctx, tx, actor, "ANNOUNCEMENT", id, nil); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM social_notifications_v2 WHERE topic='ANNOUNCEMENTS' AND JSON_UNQUOTE(JSON_EXTRACT(target_json,'$.referenceId'))=?`, id); err != nil {
			return nil, err
		}
		tombstone := map[string]any{"announcementId": id, "deleted": true}
		raw, _ := json.Marshal(tombstone)
		if _, err := tx.ExecContext(ctx, `UPDATE social_requests_v2 SET response_json=? WHERE action IN ('announcement.create','announcement.edit','announcement.publish','announcement.unpublish') AND JSON_UNQUOTE(JSON_EXTRACT(response_json,'$.announcementId'))=?`, raw, id); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM social_announcements_v2 WHERE announcement_id=?`, id); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'announcement.delete',?,UTC_TIMESTAMP(6))`, actor, id); err != nil {
			return nil, err
		}
		return tombstone, nil
	})
}
