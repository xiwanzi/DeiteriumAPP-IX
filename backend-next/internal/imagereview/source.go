package imagereview

import (
	"context"
	"database/sql"
	"fmt"
)

// Source is opened with a SELECT-only database account in production.
type Source struct {
	DB     *sql.DB
	Prefix string
	Scoped bool
}

func (s *Source) Detect(ctx context.Context) error {
	var n int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name IN ('direct_chat_media','game_chat_media')`).Scan(&n); err != nil {
		return err
	}
	s.Scoped = n == 2
	return nil
}

func (s *Source) Batch(ctx context.Context, kind, after string, limit int) ([]Candidate, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	var query string
	switch kind {
	case "CHAT":
		query = `SELECT u.upload_id,u.media_id,COALESCE(o.sha256,''),COALESCE(o.state,''),COALESCE(o.size_bytes,0),COALESCE(o.animated,FALSE),u.user_id,COALESCE(i.player_ref,''),COALESCE(i.game_id,''),COALESCE(i.display_name,i.game_id,''),u.created_at,u.expires_at
   FROM chat_media_uploads u LEFT JOIN chat_media_objects o ON o.media_id=u.media_id LEFT JOIN identities i ON i.id=u.user_id WHERE u.upload_id>? ORDER BY u.upload_id LIMIT ?`
	case "ASSET":
		query = `SELECT a.upload_id,a.asset_id,a.object_key,COALESCE(a.sha256,''),a.status,a.content_type,a.size_bytes,a.user_id,COALESCE(i.player_ref,''),COALESCE(i.game_id,''),COALESCE(i.display_name,i.game_id,''),a.purpose,a.created_at,a.expires_at
   FROM asset_uploads_v2 a LEFT JOIN identities i ON i.id=a.user_id WHERE a.upload_id>? ORDER BY a.upload_id LIMIT ?`
	default:
		return nil, fmt.Errorf("invalid source kind")
	}
	rows, err := s.DB.QueryContext(ctx, query, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Candidate{}
	for rows.Next() {
		c := Candidate{SourceKind: kind}
		if kind == "CHAT" {
			err = rows.Scan(&c.SourceID, &c.MediaID, &c.SHA256, &c.SourceState, &c.SizeBytes, &c.Animated, &c.UploaderID, &c.PlayerRef, &c.GameID, &c.DisplayName, &c.UploadedAt, &c.ExpiresAt)
			c.SourceKey = sourceKey(s.Prefix, c.MediaID)
			c.ContentType = "image/webp"
			c.Purpose = "CHAT_MEDIA"
		} else {
			err = rows.Scan(&c.SourceID, &c.MediaID, &c.SourceKey, &c.SHA256, &c.SourceState, &c.ContentType, &c.SizeBytes, &c.UploaderID, &c.PlayerRef, &c.GameID, &c.DisplayName, &c.Purpose, &c.UploadedAt, &c.ExpiresAt)
		}
		if err != nil {
			return nil, err
		}
		c.ID = ReviewID(kind, c.SourceID)
		c.Verified = c.SourceState == "READY" && len(c.SHA256) == 64
		out = append(out, c)
	}
	return out, rows.Err()
}

// Events are message provenance, not sticker ownership. No chat body or recipient
// is copied to the audit database. Stored snapshots survive source media expiry.
func (s *Source) Events(ctx context.Context, media string) ([]Event, error) {
	queries := []string{`SELECT CONCAT('public:',m.message_id),'world',COALESCE(i.id,''),m.sender_ref,m.sender_name,COALESCE(i.display_name,m.sender_name),m.sent_at
  FROM public_chat_media p JOIN chat_messages_next m ON m.message_id=p.message_id LEFT JOIN identities i ON i.server_uuid=m.sender_uuid WHERE p.media_id=? ORDER BY m.sent_at DESC LIMIT 101`}
	if s.Scoped {
		queries = append(queries,
			`SELECT CONCAT('private:',m.message_id),'private',m.sender_id,COALESCE(i.player_ref,''),COALESCE(i.game_id,''),COALESCE(i.display_name,i.game_id,''),m.sent_at FROM direct_chat_media p JOIN social_messages_v2 m ON m.message_id=p.message_id LEFT JOIN identities i ON i.id=m.sender_id WHERE p.media_id=? ORDER BY m.sent_at DESC LIMIT 101`,
			`SELECT CONCAT('game:',m.message_id),m.channel,m.sender_id,COALESCE(i.player_ref,''),COALESCE(i.game_id,''),COALESCE(i.display_name,i.game_id,''),m.sent_at FROM game_chat_media p JOIN game_chat_messages m ON m.message_id=p.message_id LEFT JOIN identities i ON i.id=m.sender_id WHERE p.media_id=? ORDER BY m.sent_at DESC LIMIT 101`)
	}
	out := []Event{}
	for _, query := range queries {
		rows, err := s.DB.QueryContext(ctx, query, media)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var e Event
			if err = rows.Scan(&e.ID, &e.Channel, &e.SenderID, &e.PlayerRef, &e.GameID, &e.DisplayName, &e.SentAt); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, e)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
