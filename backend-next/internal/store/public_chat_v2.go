package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

func publicChatFingerprintV2(content, replyID string, mentions []string) (string, []string, error) {
	if !ValidSocialText(content, 1, 256) || strings.ContainsAny(content, "\r\n\t") || replyID != "" && !ValidSocialID(replyID) || len(mentions) > 20 {
		return "", nil, ErrSocialInvalid
	}
	unique := map[string]bool{}
	for _, ref := range mentions {
		if !ValidSocialID(ref) {
			return "", nil, ErrSocialInvalid
		}
		unique[ref] = true
	}
	ordered := []string{}
	for ref := range unique {
		ordered = append(ordered, ref)
	}
	sort.Strings(ordered)
	if replyID == "" && len(ordered) == 0 {
		return Digest([]byte(content)), ordered, nil
	}
	encoded, _ := json.Marshal(struct {
		Content  string   `json:"content"`
		ReplyID  string   `json:"replyToMessageId"`
		Mentions []string `json:"mentionedPlayerRefs"`
	}{content, replyID, ordered})
	return Digest(encoded), ordered, nil
}
func (s *Store) ExistingAppChatV2(ctx context.Context, userID, clientID, content, replyID string, mentions []string) (string, error) {
	fingerprint, _, err := publicChatFingerprintV2(content, replyID, mentions)
	if err != nil {
		return "", err
	}
	var id, old string
	err = s.DB.QueryRowContext(ctx, `SELECT message_id,fingerprint FROM chat_messages_next WHERE source_id=? AND client_message_id=?`, "app:"+userID, clientID).Scan(&id, &old)
	if err == nil && old != fingerprint {
		err = ErrConflict
	}
	return id, err
}

// Public content, metadata and game delivery records commit as one operation.
// Legacy messages without metadata retain their original content fingerprint.
func (s *Store) PublishAppChatV2(ctx context.Context, u User, clientID, content, replyID string, mentions, nodes []string) (messageID string, replayed bool, err error) {
	if !ValidSocialID(clientID) {
		return "", false, ErrSocialInvalid
	}
	fingerprint, ordered, err := publicChatFingerprintV2(content, replyID, mentions)
	if err != nil {
		return "", false, err
	}
	tx, err := s.beginAccountTx(ctx, u.ID)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()
	messageID = ID("msg_")
	source := "app:" + u.ID
	result, err := tx.ExecContext(ctx, `INSERT INTO chat_messages_next(message_id,source_id,client_message_id,fingerprint,sender_uuid,sender_name,sender_ref,registered,content,kind,sent_at) VALUES(?,?,?,?,?,?,?,true,?,'public_chat',UTC_TIMESTAMP(6))`, messageID, source, clientID, fingerprint, u.ServerUUID, u.GameID, u.PlayerRef, content)
	if duplicate(err) {
		var old string
		err = tx.QueryRowContext(ctx, `SELECT message_id,fingerprint FROM chat_messages_next WHERE source_id=? AND client_message_id=?`, source, clientID).Scan(&messageID, &old)
		if err == nil && old != fingerprint {
			err = ErrConflict
		}
		return messageID, true, err
	}
	if err != nil {
		return "", false, err
	}
	var replyJSON any
	if replyID != "" {
		reply, e := socialReplyAt(ctx, tx, u.ID, "public", replyID)
		if e != nil {
			return "", false, e
		}
		replyJSON, e = json.Marshal(reply)
		if e != nil {
			return "", false, e
		}
	}
	for _, ref := range ordered {
		var exists bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM identities WHERE player_ref=?) OR EXISTS(SELECT 1 FROM chat_messages_next WHERE sender_ref=?)`, ref, ref).Scan(&exists)
		if err != nil {
			return "", false, err
		}
		if !exists {
			return "", false, ErrSocialNotFound
		}
	}
	mentionsJSON, _ := json.Marshal(ordered)
	if _, err = tx.ExecContext(ctx, `INSERT INTO public_chat_metadata_v2(message_id,reply_json,mentioned_refs_json) VALUES(?,?,?)`, messageID, replyJSON, mentionsJSON); err != nil {
		return "", false, err
	}
	seq, err := result.LastInsertId()
	if err != nil {
		return "", false, err
	}
	for _, node := range nodes {
		if _, err = tx.ExecContext(ctx, `INSERT INTO core_chat_deliveries(node_id,message_sequence,expires_at) VALUES(?,?,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 10 MINUTE))`, node, seq); err != nil {
			return "", false, err
		}
	}
	err = tx.Commit()
	return messageID, false, err
}

func publicExplicitRecipientsV2(ctx context.Context, tx *sql.Tx, messageID, senderRef string, recipients map[string]string) error {
	var raw string
	err := tx.QueryRowContext(ctx, `SELECT mentioned_refs_json FROM public_chat_metadata_v2 WHERE message_id=?`, messageID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var refs []string
	if err = json.Unmarshal([]byte(raw), &refs); err != nil {
		return err
	}
	for _, ref := range refs {
		if ref == senderRef {
			continue
		}
		var userID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM identities WHERE player_ref=? AND status='active'`, ref).Scan(&userID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		recipients[userID] = "MENTIONS"
	}
	return nil
}
