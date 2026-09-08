package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

type Player struct {
	PlayerRef  string `json:"playerRef"`
	GameID     string `json:"gameId"`
	Online     bool   `json:"online"`
	Registered bool   `json:"registered"`
	Source     string `json:"source"`
}

type ChatMessage struct {
	ID                  string       `json:"messageId"`
	Sender              Player       `json:"sender"`
	Content             string       `json:"content"`
	Kind                string       `json:"kind"`
	SentAt              time.Time    `json:"sentAt"`
	Sequence            int64        `json:"-"`
	ServerUUID          string       `json:"-"`
	ClientMessageID     string       `json:"clientMessageId,omitempty"`
	Reply               *SocialReply `json:"reply,omitempty"`
	MentionedPlayerRefs []string     `json:"mentionedPlayerRefs,omitempty"`
}

type CoreChat struct {
	PlayerUUID string `json:"playerUuid"`
	GameID     string `json:"gameId"`
	Content    string `json:"content"`
}
type ItemVersion struct {
	ItemRef              string   `json:"itemRef"`
	Revision             int64    `json:"revision"`
	PayloadSHA256        string   `json:"payloadSha256"`
	DisplayName          string   `json:"displayName"`
	Description          string   `json:"description"`
	MaxQuantity          int64    `json:"maxQuantity"`
	CompatibleServerIDs  []string `json:"compatibleServerIds"`
	RequiredMods         []string `json:"requiredMods"`
	Codec                string   `json:"codec,omitempty"`
	ItemID               string   `json:"itemId,omitempty"`
	InventoryDomain      string   `json:"inventoryDomain,omitempty"`
	CompatibilityProfile string   `json:"compatibilityProfile,omitempty"`
}

func ID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)
}

func (s *Store) CoreEvent(ctx context.Context, node, eventID, eventType string, payload []byte, chat *CoreChat, item *ItemVersion) (sequence int64, replayed bool, err error) {
	fingerprint := Digest(append([]byte(eventType+":"), payload...))
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "INSERT INTO core_events (source_id,event_id,event_type,fingerprint,payload,created_at) VALUES (?,?,?,?,?,UTC_TIMESTAMP(6))", node, eventID, eventType, fingerprint, payload)
	if duplicate(err) {
		var old string
		err = tx.QueryRowContext(ctx, "SELECT sequence_id,fingerprint FROM core_events WHERE source_id=? AND event_id=?", node, eventID).Scan(&sequence, &old)
		if err == nil && old != fingerprint {
			err = ErrConflict
		}
		return sequence, true, err
	}
	if err != nil {
		return
	}
	sequence, err = result.LastInsertId()
	if err != nil {
		return
	}
	if chat != nil {
		ref := "player_" + Digest([]byte("deuterium-player:" + chat.PlayerUUID))[:40]
		registered := false
		var userRef string
		err = tx.QueryRowContext(ctx, "SELECT player_ref FROM identities WHERE server_uuid=?", chat.PlayerUUID).Scan(&userRef)
		if err == nil {
			ref = userRef
			registered = true
		} else if !errors.Is(err, sql.ErrNoRows) {
			return
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO chat_messages_next (message_id,source_id,client_message_id,fingerprint,sender_uuid,sender_name,sender_ref,registered,content,kind,sent_at) VALUES (?,?,?,?,?,?,?,?,?,'public_chat',UTC_TIMESTAMP(6))`, ID("msg_"), "core:"+node, eventID, fingerprint, chat.PlayerUUID, chat.GameID, ref, registered, chat.Content)
		if err != nil {
			return
		}
	}
	if item != nil {
		_, err = tx.ExecContext(ctx, "INSERT INTO item_versions VALUES (?,?,?,?,?,?,UTC_TIMESTAMP(6))", item.ItemRef, item.Revision, node, item.PayloadSHA256, payload, fingerprint)
		if duplicate(err) {
			var old, publisher string
			err = tx.QueryRowContext(ctx, "SELECT fingerprint,publisher_node FROM item_versions WHERE item_ref=? AND revision=?", item.ItemRef, item.Revision).Scan(&old, &publisher)
			if err == nil && old != fingerprint {
				err = ErrConflict
			}
		}
		if err != nil {
			return
		}
	}
	err = tx.Commit()
	return
}

func (s *Store) PublishAppChat(ctx context.Context, u User, clientID, content string, nodes []string) (messageID string, replayed bool, err error) {
	fingerprint := Digest([]byte(content))
	source := "app:" + u.ID
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	messageID = ID("msg_")
	result, err := tx.ExecContext(ctx, `INSERT INTO chat_messages_next (message_id,source_id,client_message_id,fingerprint,sender_uuid,sender_name,sender_ref,registered,content,kind,sent_at) VALUES (?,?,?,?,?,?,?,true,?,'public_chat',UTC_TIMESTAMP(6))`, messageID, source, clientID, fingerprint, u.ServerUUID, u.GameID, u.PlayerRef, content)
	if duplicate(err) {
		var old string
		err = tx.QueryRowContext(ctx, "SELECT message_id,fingerprint FROM chat_messages_next WHERE source_id=? AND client_message_id=?", source, clientID).Scan(&messageID, &old)
		if err == nil && old != fingerprint {
			err = ErrConflict
		}
		return messageID, true, err
	}
	if err != nil {
		return
	}
	seq, err := result.LastInsertId()
	if err != nil {
		return
	}
	for _, node := range nodes {
		if _, err = tx.ExecContext(ctx, "INSERT INTO core_chat_deliveries (node_id,message_sequence,expires_at) VALUES (?,?,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 10 MINUTE))", node, seq); err != nil {
			return
		}
	}
	err = tx.Commit()
	return
}

const chatColumns = "m.sequence_id,m.message_id,m.sender_uuid,m.sender_name,m.sender_ref,m.registered,m.content,m.kind,m.sent_at,m.client_message_id,p.reply_json,p.mentioned_refs_json"

func scanMessages(rows *sql.Rows) ([]ChatMessage, error) {
	defer rows.Close()
	messages := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		var reply, mentions sql.NullString
		if err := rows.Scan(&m.Sequence, &m.ID, &m.ServerUUID, &m.Sender.GameID, &m.Sender.PlayerRef, &m.Sender.Registered, &m.Content, &m.Kind, &m.SentAt, &m.ClientMessageID, &reply, &mentions); err != nil {
			return nil, err
		}
		m.Sender.Source = "game_id"
		if reply.Valid {
			if err := json.Unmarshal([]byte(reply.String), &m.Reply); err != nil {
				return nil, err
			}
		}
		if mentions.Valid {
			if err := json.Unmarshal([]byte(mentions.String), &m.MentionedPlayerRefs); err != nil {
				return nil, err
			}
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (s *Store) Messages(ctx context.Context, cursor int64, after bool, limit int) ([]ChatMessage, error) {
	q := "SELECT " + chatColumns + " FROM chat_messages_next m LEFT JOIN public_chat_metadata_v2 p ON p.message_id=m.message_id"
	var rows *sql.Rows
	var err error
	if after {
		rows, err = s.DB.QueryContext(ctx, q+" WHERE m.sequence_id>? ORDER BY m.sequence_id LIMIT ?", cursor, limit)
	} else if cursor > 0 {
		rows, err = s.DB.QueryContext(ctx, q+" WHERE m.sequence_id<? ORDER BY m.sequence_id DESC LIMIT ?", cursor, limit)
	} else {
		rows, err = s.DB.QueryContext(ctx, q+" ORDER BY m.sequence_id DESC LIMIT ?", limit)
	}
	if err != nil {
		return nil, err
	}
	return scanMessages(rows)
}

func (s *Store) LatestChatSequence(ctx context.Context) (seq int64, err error) {
	err = s.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence_id),0) FROM chat_messages_next").Scan(&seq)
	return
}

func (s *Store) PendingChats(ctx context.Context, node string) ([]ChatMessage, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+chatColumns+` FROM core_chat_deliveries d JOIN chat_messages_next m ON m.sequence_id=d.message_sequence LEFT JOIN public_chat_metadata_v2 p ON p.message_id=m.message_id WHERE d.node_id=? AND d.acknowledged_at IS NULL AND d.expires_at>UTC_TIMESTAMP(6) ORDER BY d.message_sequence LIMIT 50`, node)
	if err != nil {
		return nil, err
	}
	return scanMessages(rows)
}

func (s *Store) AckChat(ctx context.Context, node, messageID string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE core_chat_deliveries d JOIN chat_messages_next m ON m.sequence_id=d.message_sequence SET d.acknowledged_at=COALESCE(d.acknowledged_at,UTC_TIMESTAMP(6)) WHERE d.node_id=? AND m.message_id=?`, node, messageID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var count int
		err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM core_chat_deliveries d JOIN chat_messages_next m ON m.sequence_id=d.message_sequence WHERE d.node_id=? AND m.message_id=?`, node, messageID).Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrConflict
		}
	}
	return nil
}

func (s *Store) Items(ctx context.Context, afterRef string, afterRevision int64, limit int) ([]ItemVersion, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT metadata FROM item_versions WHERE item_ref>? OR (item_ref=? AND revision>?) ORDER BY item_ref,revision LIMIT ?", afterRef, afterRef, afterRevision, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ItemVersion{}
	for rows.Next() {
		var payload []byte
		var item ItemVersion
		if err = rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ExistingAppChat(ctx context.Context, userID, clientID, content string) (id string, err error) {
	var hash string
	err = s.DB.QueryRowContext(ctx, "SELECT message_id,fingerprint FROM chat_messages_next WHERE source_id=? AND client_message_id=?", "app:"+userID, clientID).Scan(&id, &hash)
	if err == nil && hash != Digest([]byte(content)) {
		err = ErrConflict
	}
	return
}

func (s *Store) ReserveChat(ctx context.Context, userID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	key := Digest([]byte("chat:" + userID))
	_, err = tx.ExecContext(ctx, `INSERT INTO auth_budgets VALUES (?,1,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE)) ON DUPLICATE KEY UPDATE attempts=IF(expires_at<=UTC_TIMESTAMP(6),1,attempts+1),expires_at=IF(expires_at<=UTC_TIMESTAMP(6),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE),expires_at)`, key)
	if err != nil {
		return err
	}
	var attempts int
	if err = tx.QueryRowContext(ctx, "SELECT attempts FROM auth_budgets WHERE budget_key=?", key).Scan(&attempts); err != nil {
		return err
	}
	if attempts > 30 {
		return ErrRateLimited
	}
	return tx.Commit()
}
