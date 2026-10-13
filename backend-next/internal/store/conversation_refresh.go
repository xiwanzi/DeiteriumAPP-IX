package store

import "context"

func (s *Store) ConversationUsers(ctx context.Context, id string) (string, string, error) {
	var lo, hi string
	err := s.DB.QueryRowContext(ctx, "SELECT user_lo,user_hi FROM social_conversations_v2 WHERE conversation_id=?", id).Scan(&lo, &hi)
	return lo, hi, err
}

// A single changed row can be refreshed without paging all conversations.
func (s *Store) ConversationV2(ctx context.Context, user, id string) (SocialConversation, error) {
	var out SocialConversation
	var ref string
	err := s.DB.QueryRowContext(ctx, `SELECT c.conversation_id,c.updated_at,c.version,i.player_ref,
        (SELECT COUNT(*) FROM social_messages_v2 x WHERE x.conversation_id=c.conversation_id AND x.sequence_id>m.last_read_sequence AND x.sender_id<>?)
        FROM social_conversations_v2 c JOIN social_members_v2 m ON m.conversation_id=c.conversation_id AND m.user_id=?
        JOIN identities i ON i.id=CASE WHEN c.user_lo=? THEN c.user_hi ELSE c.user_lo END WHERE c.conversation_id=?`, user, user, user, id).
		Scan(&out.ConversationID, &out.UpdatedAt, &out.Version, &ref, &out.UnreadCount)
	if err != nil {
		return out, socialMissing(err)
	}
	if out.OtherPlayer, err = s.PublicProfileV2(ctx, user, ref); err != nil {
		return out, err
	}
	messages, err := socialMessages(ctx, s.DB, user, id, 0, 1)
	if err == nil && len(messages) > 0 {
		out.LastMessage = &messages[0]
	}
	return out, err
}
