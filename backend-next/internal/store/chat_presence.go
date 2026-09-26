package store

import (
	"context"
	"strings"
)

// ChatOnlinePlayers resolves identities without writing to the game directory or
// creating accounts. The snapshot, never last_seen, determines who is online.
func (s *Store) ChatOnlinePlayers(ctx context.Context, snapshot []CorePlayerIdentity) ([]Player, error) {
	result := make([]Player, 0, len(snapshot))
	if len(snapshot) == 0 {
		return result, nil
	}
	args := make([]any, len(snapshot))
	for i, p := range snapshot {
		args[i] = p.PlayerUUID
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT d.player_uuid,COALESCE(i.player_ref,d.player_ref),COALESCE(i.status='active',FALSE)
 FROM core_player_directory d LEFT JOIN identities i ON i.server_uuid=d.player_uuid
 WHERE d.player_uuid IN (`+strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")+`)
 AND (i.id IS NULL OR i.status='active')
 AND NOT EXISTS(SELECT 1 FROM account_deletions a WHERE a.original_uuid=d.player_uuid)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resolved := make(map[string]Player, len(snapshot))
	for rows.Next() {
		var uuid string
		p := Player{Online: true, Source: "game_id"}
		if err = rows.Scan(&uuid, &p.PlayerRef, &p.Registered); err != nil {
			return nil, err
		}
		resolved[uuid] = p
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, live := range snapshot {
		if p, ok := resolved[live.PlayerUUID]; ok {
			p.GameID = live.GameID
			result = append(result, p)
		}
	}
	return result, nil
}
