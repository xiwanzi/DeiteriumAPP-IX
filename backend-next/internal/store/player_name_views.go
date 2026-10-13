package store

import (
	"context"
	"strings"
)

// Names are resolved where typed display objects are assembled, never by
// serializing and rewriting arbitrary HTTP/WebSocket payloads.
func currentNames(ctx context.Context, q socialQuery, refs []string) (map[string]string, error) {
	unique := map[string]bool{}
	args := []any{}
	marks := []string{}
	for _, ref := range refs {
		if ref != "" && !unique[ref] {
			unique[ref] = true
			args = append(args, ref)
			marks = append(marks, "?")
		}
	}
	names := map[string]string{}
	if len(args) == 0 {
		return names, nil
	}
	if len(args) > 1000 {
		return nil, ErrSocialInvalid
	}
	filter := strings.Join(marks, ",")
	args = append(args, args...)
	args = append(args, args...)
	rows, err := q.QueryContext(ctx, `SELECT player_ref,COALESCE(server_uuid,''),game_id FROM identities WHERE status<>'deleted' AND (player_ref IN (`+filter+`) OR server_uuid IN (`+filter+`))
        UNION ALL SELECT n.player_ref,n.player_uuid,n.game_id FROM player_names_state n WHERE (n.player_ref IN (`+filter+`) OR n.player_uuid IN (`+filter+`))
        AND NOT EXISTS(SELECT 1 FROM identities i WHERE i.server_uuid=n.player_uuid)
        AND NOT EXISTS(SELECT 1 FROM account_deletions d WHERE d.player_ref=n.player_ref)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ref, uuid, name string
		if err = rows.Scan(&ref, &uuid, &name); err != nil {
			return nil, err
		}
		names[ref] = name
		if uuid != "" {
			names[uuid] = name
		}
	}
	return names, rows.Err()
}

func refreshPlayers(ctx context.Context, q socialQuery, players ...*Player) error {
	refs := make([]string, 0, len(players))
	for _, p := range players {
		if p != nil {
			refs = append(refs, p.PlayerRef)
		}
	}
	names, err := currentNames(ctx, q, refs)
	if err != nil {
		return err
	}
	for _, p := range players {
		if p != nil {
			if name := names[p.PlayerRef]; name != "" {
				p.GameID = name
			}
		}
	}
	return nil
}
func refreshChatNames(ctx context.Context, q socialQuery, messages []ChatMessage) error {
	players := []*Player{}
	for i := range messages {
		m := &messages[i]
		players = append(players, &m.Sender)
		if m.Reply != nil {
			players = append(players, &m.Reply.Sender)
		}
	}
	return refreshPlayers(ctx, q, players...)
}
func refreshDirectNames(ctx context.Context, q socialQuery, messages []SocialMessage) error {
	players := []*Player{}
	for i := range messages {
		m := &messages[i]
		// Direct senders already join current identities in socialMessageColumns.
		if m.Reply != nil {
			players = append(players, &m.Reply.Sender)
		}
		if m.Forwarded != nil {
			players = append(players, &m.Forwarded.Sender)
		}
	}
	return refreshPlayers(ctx, q, players...)
}

// Only explicit party slots are display names. The returned maps are copies;
// body text, nested item/signed snapshots, nicknames and source maps stay intact.
func projectPartyViews(ctx context.Context, q socialQuery, value map[string]any) (map[string]any, error) {
	roles := []string{"buyer", "seller", "owner", "worker", "applicant", "respondent", "counterparty"}
	refs := []string{}
	for _, role := range roles {
		if p, ok := catalogObject(value[role]); ok && p["kind"] == "PLAYER" && p["deleted"] != true {
			refs = append(refs, catalogString(p, "playerRef"))
		}
	}
	names, err := currentNames(ctx, q, refs)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range value {
		out[k] = v
	}
	for _, role := range roles {
		p, ok := catalogObject(value[role])
		if !ok || p["kind"] != "PLAYER" || p["deleted"] == true {
			continue
		}
		name := names[catalogString(p, "playerRef")]
		if name == "" {
			continue
		}
		copy := CatalogObjectV2{}
		for k, v := range p {
			copy[k] = v
		}
		copy["displayName"] = name
		copy["gameId"] = name
		out[role] = copy
	}
	return out, nil
}
func (s *Store) ProjectPartyViews(ctx context.Context, value map[string]any) (map[string]any, error) {
	return projectPartyViews(ctx, s.DB, value)
}
func (s *Store) CurrentPlayerNames(ctx context.Context, refs []string) (map[string]string, error) {
	return currentNames(ctx, s.DB, refs)
}
