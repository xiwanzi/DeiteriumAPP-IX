package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// The caller is authenticated Core, which supplies the server-owned audience.
// Neither a bearer client nor an App can supply recipient UUIDs to this method.
type GameChatMediaRequest struct {
	PlayerUUID      string        `json:"playerUuid"`
	ClientMessageID string        `json:"clientMessageId"`
	Channel         string        `json:"channel"`
	Media           ChatMediaSend `json:"media"`
	Members         []string      `json:"members"`
	TeamID          string        `json:"teamId"`
	TeamRevision    int64         `json:"teamRevision"`
}
type GameChatMessage struct {
	ChatMessage
	Type         string   `json:"type"`
	Channel      string   `json:"channel"`
	Members      []string `json:"members"`
	TeamID       string   `json:"teamId,omitempty"`
	TeamRevision int64    `json:"teamRevision,omitempty"`
}

func (s *Store) GameChatMedia(ctx context.Context, node string, in GameChatMediaRequest) (GameChatMessage, error) {
	var u User
	err := s.DB.QueryRowContext(ctx, `SELECT id,player_ref,game_id,server_uuid FROM identities WHERE server_uuid=? AND status='active' AND game_verified=TRUE`, in.PlayerUUID).Scan(&u.ID, &u.PlayerRef, &u.GameID, &u.ServerUUID)
	if err != nil {
		return GameChatMessage{}, socialMissing(err)
	}
	raw, _ := json.Marshal(struct {
		Channel string
		Media   ChatMediaSend
	}{in.Channel, in.Media})
	fingerprint := Digest(raw)
	tx, err := s.beginAccountTx(ctx, u.ID)
	if err != nil {
		return GameChatMessage{}, err
	}
	defer tx.Rollback()
	var oldHash string
	var old []byte
	err = tx.QueryRowContext(ctx, `SELECT fingerprint,message_json FROM game_chat_messages WHERE sender_id=? AND client_message_id=?`, u.ID, in.ClientMessageID).Scan(&oldHash, &old)
	if err == nil {
		if oldHash != fingerprint {
			return GameChatMessage{}, ErrConflict
		}
		var message GameChatMessage
		err = json.Unmarshal(old, &message)
		if err == nil {
			err = refreshPlayers(ctx, tx, &message.Sender)
		}
		return message, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return GameChatMessage{}, err
	}
	if err = reserveSocialSend(ctx, tx, u.ID); err != nil {
		return GameChatMessage{}, err
	}
	media, err := chatMediaForSend(ctx, tx, u.ID, &in.Media)
	if err != nil {
		return GameChatMessage{}, err
	}
	m := GameChatMessage{ChatMessage: ChatMessage{ID: ID("gmsg_"), ClientMessageID: in.ClientMessageID, ServerID: node, Content: in.Media.Summary(), Media: &media, SentAt: time.Now().UTC(), Kind: "game_chat", Sender: Player{PlayerRef: in.PlayerUUID, GameID: u.GameID, Registered: true, Source: "game_id"}}, Type: "message", Channel: in.Channel, Members: in.Members, TeamID: in.TeamID, TeamRevision: in.TeamRevision}
	raw, err = json.Marshal(m)
	if err != nil {
		return m, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO game_chat_messages(message_id,sender_id,client_message_id,fingerprint,channel,message_json,sent_at) VALUES(?,?,?,?,?,?,?)`, m.ID, u.ID, in.ClientMessageID, fingerprint, in.Channel, raw, m.SentAt); err != nil {
		return m, err
	}
	for _, uuid := range in.Members {
		if _, err = tx.ExecContext(ctx, `INSERT IGNORE INTO game_chat_recipients(message_id,player_uuid) VALUES(?,?)`, m.ID, uuid); err != nil {
			return m, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO game_chat_media(message_id,media_id,expires_at) VALUES(?,?,?)`, m.ID, media.ID, media.ExpiresAt); err != nil {
		return m, err
	}
	return m, tx.Commit()
}
func (s *Store) GameChatMessages(ctx context.Context, uuid, channel string) ([]GameChatMessage, error) {
	retentionSeconds := 30 * 24 * 60 * 60
	if channel == "nearby" {
		retentionSeconds = 5 * 60
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT m.message_json FROM game_chat_messages m JOIN game_chat_recipients r ON r.message_id=m.message_id AND r.player_uuid=? WHERE m.channel=? AND m.sent_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL ? SECOND) ORDER BY m.sequence_id DESC LIMIT 100`, uuid, channel, retentionSeconds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GameChatMessage{}
	for rows.Next() {
		var raw []byte
		var m GameChatMessage
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	players := make([]*Player, 0, len(out))
	for i := range out {
		players = append(players, &out[i].Sender)
	}
	return out, refreshPlayers(ctx, s.DB, players...)
}
