package store

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
)

// Presentation projection only. It never rewrites stored trade/message snapshots,
// free text, nicknames, titles, amounts, signatures, or any identity binding.
func (s *Store) ProjectPlayerNames(ctx context.Context, value any) (any, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return value, e
	}
	var out any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if e = decoder.Decode(&out); e != nil {
		return value, e
	}
	refs := map[string]bool{}
	skip := func(key string) bool {
		switch key {
		case "snapshot", "snapshotJson", "raw", "signedPayload", "configuration", "settings", "body", "content", "payload", "result":
			return true
		}
		return false
	}
	identity := func(v map[string]any) string {
		for _, key := range []string{"playerRef", "senderRef", "playerUuid", "serverUuid", "profileUuid", "uuid"} {
			if ref, ok := v[key].(string); ok && ref != "" {
				return ref
			}
		}
		return ""
	}
	var collect func(any)
	collect = func(v any) {
		switch x := v.(type) {
		case []any:
			for _, part := range x {
				collect(part)
			}
		case map[string]any:
			if ref := identity(x); ref != "" {
				refs[ref] = true
			}
			for k, part := range x {
				if !skip(k) {
					collect(part)
				}
			}
		}
	}
	collect(out)
	if len(refs) == 0 {
		return value, nil
	}
	if len(refs) > 1000 {
		return value, ErrSocialInvalid
	}
	args := []any{}
	marks := []string{}
	for ref := range refs {
		args = append(args, ref)
		marks = append(marks, "?")
	}
	args = append(args, args...)
	query := `SELECT n.player_uuid,n.player_ref,COALESCE(i.game_id,n.game_id) FROM player_names_state n LEFT JOIN identities i ON i.server_uuid=n.player_uuid WHERE n.version>0 AND (n.player_ref IN (` + strings.Join(marks, ",") + `) OR n.player_uuid IN (` + strings.Join(marks, ",") + `)) AND (i.id IS NULL OR i.status<>'deleted') AND NOT EXISTS(SELECT 1 FROM account_deletions d WHERE d.player_ref=n.player_ref)`
	rows, e := s.DB.QueryContext(ctx, query, args...)
	if e != nil {
		return value, e
	}
	names := map[string]string{}
	for rows.Next() {
		var uuid, ref, name string
		if e = rows.Scan(&uuid, &ref, &name); e != nil {
			rows.Close()
			return value, e
		}
		names[uuid] = name
		names[ref] = name
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return value, e
	}
	if len(names) == 0 {
		return value, nil
	}
	var apply func(any)
	apply = func(v any) {
		switch x := v.(type) {
		case []any:
			for _, part := range x {
				apply(part)
			}
		case map[string]any:
			if name := names[identity(x)]; name != "" {
				for _, key := range []string{"gameId", "senderName", "playerName"} {
					if _, present := x[key]; present {
						x[key] = name
					}
				}
				if x["kind"] == "PLAYER" {
					if _, present := x["displayName"]; present {
						x["displayName"] = name
					}
					x["gameId"] = name
				}
			}
			for k, part := range x {
				if !skip(k) {
					apply(part)
				}
			}
		}
	}
	apply(out)
	return out, nil
}
