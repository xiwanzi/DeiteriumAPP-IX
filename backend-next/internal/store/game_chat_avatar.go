package store

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"
)

var avatarGameName = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)

// Names resolve through the authoritative game directory, never arbitrary client UUIDs or URLs.
func (s *Store) GameChatAvatar(ctx context.Context, name string, fetch func(context.Context, string) ([]byte, error)) ([]byte, error) {
	if !avatarGameName.MatchString(name) {
		return nil, ErrSocialInvalid
	}
	var uuid string
	err := s.DB.QueryRowContext(ctx, `SELECT server_uuid FROM identities WHERE LOWER(game_id)=LOWER(?) AND status='active' AND game_verified=TRUE
 UNION SELECT d.player_uuid FROM core_player_directory d WHERE LOWER(d.game_id)=LOWER(?)
 AND NOT EXISTS(SELECT 1 FROM identities i WHERE i.server_uuid=d.player_uuid AND i.status<>'active')
 AND NOT EXISTS(SELECT 1 FROM account_deletions a WHERE a.original_uuid=d.player_uuid) LIMIT 1`, name, name).Scan(&uuid)
	if err != nil {
		return nil, socialMissing(err)
	}
	var old []byte
	var refresh time.Time
	err = s.DB.QueryRowContext(ctx, `SELECT png,refresh_after FROM game_chat_avatars WHERE player_uuid=?`, uuid).Scan(&old, &refresh)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if time.Now().Before(refresh) {
		if len(old) == 0 {
			return nil, ErrSocialNotFound
		}
		return old, nil
	}
	png, fetchErr := fetch(ctx, uuid)
	if fetchErr != nil {
		// Keep a previously known portrait through upstream outages; throttle cold misses too.
		_, err = s.DB.ExecContext(ctx, `INSERT INTO game_chat_avatars(player_uuid,png,refresh_after,updated_at) VALUES(?,?,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 5 MINUTE),UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE refresh_after=VALUES(refresh_after)`, uuid, []byte{})
		if len(old) > 0 {
			return old, nil
		}
		if err != nil {
			return nil, err
		}
		return nil, ErrSocialNotFound
	}
	if len(png) == 0 || len(png) > 4096 {
		return nil, ErrSocialInvalid
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO game_chat_avatars(player_uuid,png,refresh_after,updated_at) VALUES(?,?,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 DAY),UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE png=VALUES(png),refresh_after=VALUES(refresh_after),updated_at=VALUES(updated_at)`, uuid, png)
	return png, err
}
