package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var ErrSocialNotFound = errors.New("social resource is not visible")
var ErrSocialInvalid = errors.New("invalid social request")
var ErrSocialVersion = errors.New("social resource version changed")
var ErrSocialCapability = errors.New("social asset capability unavailable")

type SocialProfile struct {
	PlayerRef  string     `json:"playerRef"`
	GameID     string     `json:"gameId"`
	QQ         string     `json:"qq"`
	Bio        string     `json:"bio"`
	Avatar     any        `json:"avatar"`
	Online     bool       `json:"online"`
	LastSeenAt *time.Time `json:"lastSeenAt"`
	Followed   bool       `json:"followed"`
	Version    int64      `json:"version"`
}
type SocialProfilePatch struct {
	ClientRequestID string          `json:"clientRequestId"`
	ExpectedVersion int64           `json:"expectedVersion"`
	Bio             *string         `json:"bio,omitempty"`
	AvatarAssetID   json.RawMessage `json:"avatarAssetId,omitempty"`
}
type SocialReply struct {
	MessageID    string `json:"messageId"`
	Sender       Player `json:"sender"`
	Content      string `json:"content"`
	Availability string `json:"availability"`
}
type SocialMessage struct {
	MessageID       string       `json:"messageId"`
	Sender          Player       `json:"sender"`
	Content         string       `json:"content"`
	Kind            string       `json:"kind"`
	SentAt          time.Time    `json:"sentAt"`
	ConversationID  string       `json:"conversationId"`
	Reply           *SocialReply `json:"reply"`
	Forwarded       *SocialReply `json:"forwarded,omitempty"`
	ClientMessageID string       `json:"clientMessageId"`
	Sequence        int64        `json:"-"`
}
type SocialSendRequest struct {
	ClientMessageID     string   `json:"clientMessageId"`
	Content             string   `json:"content"`
	ReplyToMessageID    string   `json:"replyToMessageId,omitempty"`
	MentionedPlayerRefs []string `json:"mentionedPlayerRefs,omitempty"`
}
type SocialForwardRequest struct {
	ClientMessageID      string `json:"clientMessageId"`
	SourceMessageID      string `json:"sourceMessageId"`
	SourceConversationID string `json:"sourceConversationId,omitempty"`
}
type SocialConversation struct {
	ConversationID string         `json:"conversationId"`
	OtherPlayer    SocialProfile  `json:"otherPlayer"`
	LastMessage    *SocialMessage `json:"lastMessage"`
	UnreadCount    int64          `json:"unreadCount"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	Version        int64          `json:"version"`
}
type SocialTarget struct {
	Kind         string `json:"kind"`
	ReferenceID  string `json:"referenceId"`
	StateVersion int64  `json:"stateVersion"`
}
type SocialNotification struct {
	SystemPush     bool         `json:"systemPush"`
	NotificationID string       `json:"notificationId"`
	Topic          string       `json:"topic"`
	Title          string       `json:"title"`
	Body           string       `json:"body"`
	Target         SocialTarget `json:"target"`
	CreatedAt      time.Time    `json:"createdAt"`
	ReadAt         *time.Time   `json:"readAt"`
	Sequence       int64        `json:"-"`
}
type SocialContentRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type SocialContentBlock struct {
	BlockID string             `json:"blockId"`
	Type    string             `json:"type"`
	Heading string             `json:"heading,omitempty"`
	Text    string             `json:"text,omitempty"`
	AssetID *string            `json:"assetId,omitempty"`
	AltText string             `json:"altText,omitempty"`
	Rows    []SocialContentRow `json:"rows,omitempty"`
}
type SocialAnnouncementWrite struct {
	ClientRequestID string               `json:"clientRequestId"`
	ExpectedVersion int64                `json:"expectedVersion,omitempty"`
	Title           string               `json:"title"`
	Summary         string               `json:"summary"`
	ContentBlocks   []SocialContentBlock `json:"contentBlocks"`
	CoverAssetID    *string              `json:"coverAssetId"`
	Pinned          bool                 `json:"pinned"`
	Priority        string               `json:"priority"`
}
type SocialAnnouncement struct {
	AnnouncementID        string               `json:"announcementId"`
	Title                 string               `json:"title"`
	Summary               string               `json:"summary"`
	ContentBlocks         []SocialContentBlock `json:"contentBlocks"`
	CoverAssetID          *string              `json:"coverAssetId"`
	Pinned                bool                 `json:"pinned"`
	Priority              string               `json:"priority"`
	PublishedAt           *time.Time           `json:"publishedAt"`
	UpdatedAt             time.Time            `json:"updatedAt"`
	Version               int64                `json:"version"`
	Status                string               `json:"status,omitempty"`
	PublishedVersion      *int64               `json:"publishedVersion,omitempty"`
	HasUnpublishedChanges *bool                `json:"hasUnpublishedChanges,omitempty"`
	Cover                 any                  `json:"cover,omitempty"`
	Media                 []map[string]any     `json:"media,omitempty"`
	Sequence              int64                `json:"-"`
}

func ValidSocialID(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if r > 127 || !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == ':' || r == '.') {
			return false
		}
	}
	return true
}
func ValidSocialText(value string, min, max int) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max || utf8.RuneCountInString(strings.TrimSpace(value)) < min {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
func socialMissing(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSocialNotFound
	}
	return err
}

// The durable request row serializes identical mutations. Effects and response
// commit together, so retrying after a lost response cannot create another item.
func (s *Store) socialMutate(ctx context.Context, user, action, key string, input any, run func(*sql.Tx) (any, error)) (json.RawMessage, error) {
	if !ValidSocialID(key) {
		return nil, ErrSocialInvalid
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	fingerprint := Digest(encoded)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO social_requests_v2(actor_id,action,request_id,fingerprint,created_at) VALUES(?,?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE request_id=VALUES(request_id)`, user, action, key, fingerprint); err != nil {
		return nil, err
	}
	var old string
	var response sql.NullString
	if err = tx.QueryRowContext(ctx, `SELECT fingerprint,response_json FROM social_requests_v2 WHERE actor_id=? AND action=? AND request_id=? FOR UPDATE`, user, action, key).Scan(&old, &response); err != nil {
		return nil, err
	}
	if old != fingerprint {
		return nil, ErrConflict
	}
	if response.Valid {
		return json.RawMessage(response.String), nil
	}
	result, err := run(tx)
	if err != nil {
		return nil, err
	}
	encoded, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE social_requests_v2 SET response_json=? WHERE actor_id=? AND action=? AND request_id=?`, encoded, user, action, key); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return encoded, nil
}

func (s *Store) PublicProfileV2(ctx context.Context, viewerID, playerRef string) (SocialProfile, error) {
	var p SocialProfile
	var avatar sql.NullString
	var ownerID string
	err := s.DB.QueryRowContext(ctx, `SELECT i.id,i.player_ref,i.game_id,i.qq,COALESCE(p.bio,''),p.avatar_json,COALESCE(p.version,1),EXISTS(SELECT 1 FROM social_follows_v2 f WHERE f.follower_id=? AND f.followed_id=i.id)
 FROM identities i LEFT JOIN social_profiles_v2 p ON p.user_id=i.id
 JOIN identities viewer ON viewer.id=? AND viewer.status='active'
	 WHERE i.player_ref=?`, viewerID, viewerID, playerRef).Scan(&ownerID, &p.PlayerRef, &p.GameID, &p.QQ, &p.Bio, &avatar, &p.Version, &p.Followed)
	if err != nil {
		return p, socialMissing(err)
	}
	if avatar.Valid {
		var stored struct {
			AssetID string `json:"assetId"`
		}
		if err = json.Unmarshal([]byte(avatar.String), &stored); err != nil {
			return p, err
		}
		p.Avatar, err = s.AssetV2(ctx, ownerID, stored.AssetID)
		if err != nil {
			return p, err
		}
	}
	// Presence remains unknown until an authenticated Core presence source supplies it.
	return p, nil
}
func (s *Store) PatchProfileV2(ctx context.Context, u User, input SocialProfilePatch) (json.RawMessage, error) {
	if input.ExpectedVersion < 1 || (input.Bio == nil && input.AvatarAssetID == nil) || input.Bio != nil && !ValidSocialText(*input.Bio, 0, 200) {
		return nil, ErrSocialInvalid
	}
	var avatarValue any
	var avatarJSON []byte
	var avatarIDs []string
	if input.AvatarAssetID != nil && string(input.AvatarAssetID) != "null" {
		var assetID string
		if json.Unmarshal(input.AvatarAssetID, &assetID) != nil || !ValidSocialID(assetID) {
			return nil, ErrSocialInvalid
		}
		if err := s.ValidateOwnedAssetsV2(ctx, u.ID, []string{assetID}, "AVATAR"); err != nil {
			return nil, err
		}
		var err error
		avatarValue, err = s.AssetV2(ctx, u.ID, assetID)
		if err != nil {
			return nil, err
		}
		avatarJSON, _ = json.Marshal(map[string]string{"assetId": assetID})
		avatarIDs = []string{assetID}
	}
	return s.socialMutate(ctx, u.ID, "profile.patch", input.ClientRequestID, input, func(tx *sql.Tx) (any, error) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO social_profiles_v2(user_id,bio,version,updated_at) VALUES(?,'',1,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE user_id=VALUES(user_id)`, u.ID); err != nil {
			return nil, err
		}
		var p SocialProfile
		var avatar sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT bio,avatar_json,version FROM social_profiles_v2 WHERE user_id=? FOR UPDATE`, u.ID).Scan(&p.Bio, &avatar, &p.Version); err != nil {
			return nil, err
		}
		if p.Version != input.ExpectedVersion {
			return nil, ErrSocialVersion
		}
		if input.Bio != nil {
			p.Bio = *input.Bio
		}
		if input.AvatarAssetID != nil {
			avatar = sql.NullString{String: string(avatarJSON), Valid: len(avatarJSON) > 0}
			if err := s.SetAssetBindingsV2(ctx, tx, u.ID, "PROFILE", u.ID, avatarIDs); err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE social_profiles_v2 SET bio=?,avatar_json=?,version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE user_id=?`, p.Bio, avatar, u.ID); err != nil {
			return nil, err
		}
		p.PlayerRef, p.GameID, p.QQ, p.Version = u.PlayerRef, u.GameID, u.QQ, p.Version+1
		if input.AvatarAssetID != nil {
			p.Avatar = avatarValue
		} else if avatar.Valid {
			var stored struct {
				AssetID string `json:"assetId"`
			}
			if err := json.Unmarshal([]byte(avatar.String), &stored); err != nil {
				return nil, err
			}
			value, err := s.AssetV2(ctx, u.ID, stored.AssetID)
			if err != nil {
				return nil, err
			}
			p.Avatar = value
		}
		return p, nil
	})
}
func (s *Store) PlayerDirectoryV2(ctx context.Context, userID, query string, limit int) ([]SocialProfile, error) {
	if limit < 1 || limit > 100 || !ValidSocialText(query, 0, 80) {
		return nil, ErrSocialInvalid
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT player_ref FROM identities WHERE status='active' AND (game_id LIKE ? OR qq=?) ORDER BY game_id,id LIMIT ?`, "%"+strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(query)+"%", query, limit)
	if err != nil {
		return nil, err
	}
	refs := []string{}
	for rows.Next() {
		var ref string
		if err = rows.Scan(&ref); err != nil {
			rows.Close()
			return nil, err
		}
		refs = append(refs, ref)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	result := []SocialProfile{}
	for _, ref := range refs {
		p, e := s.PublicProfileV2(ctx, userID, ref)
		if e != nil {
			return nil, e
		}
		result = append(result, p)
	}
	return result, nil
}
func (s *Store) FollowsV2(ctx context.Context, userID string) ([]SocialProfile, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT i.player_ref FROM social_follows_v2 f JOIN identities i ON i.id=f.followed_id WHERE f.follower_id=? AND i.status='active' ORDER BY i.game_id LIMIT 1000`, userID)
	if err != nil {
		return nil, err
	}
	refs := []string{}
	for rows.Next() {
		var ref string
		if err = rows.Scan(&ref); err != nil {
			rows.Close()
			return nil, err
		}
		refs = append(refs, ref)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	result := []SocialProfile{}
	for _, ref := range refs {
		p, e := s.PublicProfileV2(ctx, userID, ref)
		if e != nil {
			return nil, e
		}
		result = append(result, p)
	}
	return result, nil
}
func (s *Store) FollowV2(ctx context.Context, u User, ref string, follow bool) (SocialProfile, error) {
	p, err := s.PublicProfileV2(ctx, u.ID, ref)
	if err != nil {
		return p, err
	}
	if ref == u.PlayerRef {
		return p, ErrSocialInvalid
	}
	if follow {
		_, err = s.DB.ExecContext(ctx, `INSERT IGNORE INTO social_follows_v2(follower_id,followed_id,created_at) SELECT ?,id,UTC_TIMESTAMP(6) FROM identities WHERE player_ref=? AND status='active'`, u.ID, ref)
	} else {
		_, err = s.DB.ExecContext(ctx, `DELETE f FROM social_follows_v2 f JOIN identities i ON i.id=f.followed_id WHERE f.follower_id=? AND i.player_ref=?`, u.ID, ref)
	}
	p.Followed = follow
	return p, err
}

func socialMember(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, userID, conversation string) (string, error) {
	var other string
	err := q.QueryRowContext(ctx, `SELECT CASE WHEN c.user_lo=? THEN c.user_hi ELSE c.user_lo END FROM social_conversations_v2 c JOIN social_members_v2 m ON m.conversation_id=c.conversation_id AND m.user_id=? WHERE c.conversation_id=?`, userID, userID, conversation).Scan(&other)
	return other, socialMissing(err)
}
func (s *Store) CreateConversationV2(ctx context.Context, u User, key, ref string) (json.RawMessage, error) {
	if !ValidSocialID(ref) || ref == u.PlayerRef {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, u.ID, "conversation.create", key, map[string]string{"otherPlayerRef": ref}, func(tx *sql.Tx) (any, error) {
		var other string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM identities WHERE player_ref=? AND status='active'`, ref).Scan(&other); err != nil {
			return nil, socialMissing(err)
		}
		pair := []string{u.ID, other}
		sort.Strings(pair)
		conversation := ID("conv_")
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO social_conversations_v2(conversation_id,user_lo,user_hi,updated_at) VALUES(?,?,?,UTC_TIMESTAMP(6))`, conversation, pair[0], pair[1]); err != nil {
			return nil, err
		}
		var result SocialConversation
		if err := tx.QueryRowContext(ctx, `SELECT conversation_id,updated_at,version FROM social_conversations_v2 WHERE user_lo=? AND user_hi=?`, pair[0], pair[1]).Scan(&result.ConversationID, &result.UpdatedAt, &result.Version); err != nil {
			return nil, err
		}
		for _, member := range pair {
			if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO social_members_v2(conversation_id,user_id) VALUES(?,?)`, result.ConversationID, member); err != nil {
				return nil, err
			}
		}
		// Read through the same transaction so a newly created conversation is complete.
		var avatar sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT i.player_ref,i.game_id,i.qq,COALESCE(p.bio,''),p.avatar_json,COALESCE(p.version,1),EXISTS(SELECT 1 FROM social_follows_v2 WHERE follower_id=? AND followed_id=i.id) FROM identities i LEFT JOIN social_profiles_v2 p ON p.user_id=i.id WHERE i.id=?`, u.ID, other).Scan(&result.OtherPlayer.PlayerRef, &result.OtherPlayer.GameID, &result.OtherPlayer.QQ, &result.OtherPlayer.Bio, &avatar, &result.OtherPlayer.Version, &result.OtherPlayer.Followed); err != nil {
			return nil, err
		}
		if avatar.Valid {
			var stored struct {
				AssetID string `json:"assetId"`
			}
			if err := json.Unmarshal([]byte(avatar.String), &stored); err != nil {
				return nil, err
			}
			asset, err := s.AssetV2(ctx, other, stored.AssetID)
			if err != nil {
				return nil, err
			}
			result.OtherPlayer.Avatar = asset
		}
		messages, err := socialMessages(ctx, tx, u.ID, result.ConversationID, 0, 1)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 {
			result.LastMessage = &messages[0]
		}
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM social_messages_v2 x JOIN social_members_v2 m ON m.conversation_id=x.conversation_id AND m.user_id=? WHERE x.conversation_id=? AND x.sequence_id>m.last_read_sequence AND x.sender_id<>?`, u.ID, result.ConversationID, u.ID).Scan(&result.UnreadCount); err != nil {
			return nil, err
		}
		return result, nil
	})
}

type socialQuery interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

const socialMessageColumns = `x.sequence_id,x.message_id,x.conversation_id,x.client_message_id,x.content,x.reply_json,x.forwarded_json,x.sent_at,i.player_ref,i.game_id`

func socialMessages(ctx context.Context, q socialQuery, userID, conversation string, before int64, limit int) ([]SocialMessage, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+socialMessageColumns+` FROM social_messages_v2 x JOIN identities i ON i.id=x.sender_id JOIN social_members_v2 m ON m.conversation_id=x.conversation_id AND m.user_id=? WHERE x.conversation_id=? AND (?=0 OR x.sequence_id<?) ORDER BY x.sequence_id DESC LIMIT ?`, userID, conversation, before, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SocialMessage{}
	for rows.Next() {
		var m SocialMessage
		var reply, forward sql.NullString
		if err = rows.Scan(&m.Sequence, &m.MessageID, &m.ConversationID, &m.ClientMessageID, &m.Content, &reply, &forward, &m.SentAt, &m.Sender.PlayerRef, &m.Sender.GameID); err != nil {
			return nil, err
		}
		m.Kind = "private_chat"
		m.Sender.Registered = true
		m.Sender.Source = "game_id"
		if reply.Valid {
			if err = json.Unmarshal([]byte(reply.String), &m.Reply); err != nil {
				return nil, err
			}
		}
		if forward.Valid {
			if err = json.Unmarshal([]byte(forward.String), &m.Forwarded); err != nil {
				return nil, err
			}
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
func (s *Store) ConversationMessagesV2(ctx context.Context, userID, conversation, beforeID string, before int64, limit int) ([]SocialMessage, error) {
	if _, err := socialMember(ctx, s.DB, userID, conversation); err != nil {
		return nil, err
	}
	if beforeID != "" {
		if err := s.DB.QueryRowContext(ctx, `SELECT sequence_id FROM social_messages_v2 WHERE conversation_id=? AND message_id=?`, conversation, beforeID).Scan(&before); err != nil {
			return nil, socialMissing(err)
		}
	}
	return socialMessages(ctx, s.DB, userID, conversation, before, limit)
}
func (s *Store) ConversationsV2(ctx context.Context, userID string, beforeTime time.Time, beforeID string, limit int) ([]SocialConversation, error) {
	query := `SELECT c.conversation_id,c.updated_at,c.version,i.player_ref,(SELECT COUNT(*) FROM social_messages_v2 x WHERE x.conversation_id=c.conversation_id AND x.sequence_id>m.last_read_sequence AND x.sender_id<>?) FROM social_conversations_v2 c JOIN social_members_v2 m ON m.conversation_id=c.conversation_id AND m.user_id=? JOIN identities i ON i.id=CASE WHEN c.user_lo=? THEN c.user_hi ELSE c.user_lo END`
	args := []any{userID, userID, userID}
	if !beforeTime.IsZero() {
		query += ` WHERE (c.updated_at<? OR (c.updated_at=? AND c.conversation_id<?))`
		args = append(args, beforeTime, beforeTime, beforeID)
	}
	query += ` ORDER BY c.updated_at DESC,c.conversation_id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	result := []SocialConversation{}
	refs := []string{}
	for rows.Next() {
		var c SocialConversation
		var ref string
		if err = rows.Scan(&c.ConversationID, &c.UpdatedAt, &c.Version, &ref, &c.UnreadCount); err != nil {
			rows.Close()
			return nil, err
		}
		result = append(result, c)
		refs = append(refs, ref)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for n := range result {
		p, e := s.PublicProfileV2(ctx, userID, refs[n])
		if e != nil {
			return nil, e
		}
		result[n].OtherPlayer = p
		m, e := socialMessages(ctx, s.DB, userID, result[n].ConversationID, 0, 1)
		if e != nil {
			return nil, e
		}
		if len(m) > 0 {
			result[n].LastMessage = &m[0]
		}
	}
	return result, nil
}

func socialReplyAt(ctx context.Context, tx *sql.Tx, userID, conversation, messageID string) (*SocialReply, error) {
	var r SocialReply
	r.MessageID = messageID
	r.Availability = "AVAILABLE"
	r.Sender.Source = "game_id"
	var err error
	if conversation == "" || conversation == "public" {
		err = tx.QueryRowContext(ctx, `SELECT sender_ref,sender_name,registered,content FROM chat_messages_next WHERE message_id=?`, messageID).Scan(&r.Sender.PlayerRef, &r.Sender.GameID, &r.Sender.Registered, &r.Content)
	} else {
		err = tx.QueryRowContext(ctx, `SELECT i.player_ref,i.game_id,x.content FROM social_messages_v2 x JOIN identities i ON i.id=x.sender_id JOIN social_members_v2 m ON m.conversation_id=x.conversation_id AND m.user_id=? WHERE x.conversation_id=? AND x.message_id=?`, userID, conversation, messageID).Scan(&r.Sender.PlayerRef, &r.Sender.GameID, &r.Content)
		r.Sender.Registered = true
	}
	return &r, socialMissing(err)
}
func reserveSocialSend(ctx context.Context, tx *sql.Tx, userID string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO social_send_budgets_v2(user_id,attempts,expires_at) VALUES(?,1,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE)) ON DUPLICATE KEY UPDATE attempts=IF(expires_at<=UTC_TIMESTAMP(6),1,attempts+1),expires_at=IF(expires_at<=UTC_TIMESTAMP(6),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE),expires_at)`, userID); err != nil {
		return err
	}
	var attempts int
	if err := tx.QueryRowContext(ctx, `SELECT attempts FROM social_send_budgets_v2 WHERE user_id=?`, userID).Scan(&attempts); err != nil {
		return err
	}
	if attempts > 60 {
		return ErrRateLimited
	}
	return nil
}
func AddNotificationV2(ctx context.Context, tx *sql.Tx, userID, eventKey, topic, title, body string, target SocialTarget) error {
	return AddNotificationWithPushV206(ctx, tx, userID, eventKey, topic, title, body, target, true)
}
func AddNotificationWithPushV206(ctx context.Context, tx *sql.Tx, userID, eventKey, topic, title, body string, target SocialTarget, push bool) error {
	encoded, err := json.Marshal(target)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO social_notifications_v2(notification_id,user_id,event_key,topic,title,body,target_json,system_push,created_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, ID("not_"), userID, eventKey, topic, title, body, encoded, push)
	return err
}
func (s *Store) SendDirectV2(ctx context.Context, u User, conversation string, input SocialSendRequest, forward *SocialForwardRequest) (json.RawMessage, error) {
	if !ValidSocialID(conversation) || !ValidSocialID(input.ClientMessageID) || len(input.MentionedPlayerRefs) > 20 {
		return nil, ErrSocialInvalid
	}
	if forward == nil && !ValidSocialText(input.Content, 1, 256) {
		return nil, ErrSocialInvalid
	}
	if input.ReplyToMessageID != "" && !ValidSocialID(input.ReplyToMessageID) {
		return nil, ErrSocialInvalid
	}
	payload := struct {
		ConversationID string
		Message        SocialSendRequest
		Forward        *SocialForwardRequest
	}{conversation, input, forward}
	return s.socialMutate(ctx, u.ID, "direct.send", input.ClientMessageID, payload, func(tx *sql.Tx) (any, error) {
		other, err := socialMember(ctx, tx, u.ID, conversation)
		if err != nil {
			return nil, err
		}
		var version int64
		if err = tx.QueryRowContext(ctx, `SELECT version FROM social_conversations_v2 WHERE conversation_id=? FOR UPDATE`, conversation).Scan(&version); err != nil {
			return nil, err
		}
		var otherRef string
		if err = tx.QueryRowContext(ctx, `SELECT player_ref FROM identities WHERE id=? AND status='active'`, other).Scan(&otherRef); err != nil {
			return nil, socialMissing(err)
		}
		for _, ref := range input.MentionedPlayerRefs {
			if ref != otherRef && ref != u.PlayerRef {
				return nil, ErrSocialNotFound
			}
		}
		if err = reserveSocialSend(ctx, tx, u.ID); err != nil {
			return nil, err
		}
		m := SocialMessage{MessageID: ID("msg_"), ConversationID: conversation, ClientMessageID: input.ClientMessageID, Content: input.Content, Kind: "private_chat", Sender: Player{PlayerRef: u.PlayerRef, GameID: u.GameID, Registered: true, Source: "game_id"}, SentAt: time.Now().UTC()}
		var replyJSON, forwardJSON any
		if input.ReplyToMessageID != "" {
			m.Reply, err = socialReplyAt(ctx, tx, u.ID, conversation, input.ReplyToMessageID)
			if err != nil {
				return nil, err
			}
			replyJSON, err = json.Marshal(m.Reply)
			if err != nil {
				return nil, err
			}
		}
		if forward != nil {
			if !ValidSocialID(forward.SourceMessageID) {
				return nil, ErrSocialInvalid
			}
			m.Forwarded, err = socialReplyAt(ctx, tx, u.ID, forward.SourceConversationID, forward.SourceMessageID)
			if err != nil {
				return nil, err
			}
			m.Content = m.Forwarded.Content
			forwardJSON, err = json.Marshal(m.Forwarded)
			if err != nil {
				return nil, err
			}
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO social_messages_v2(message_id,conversation_id,sender_id,client_message_id,content,reply_json,forwarded_json,sent_at) VALUES(?,?,?,?,?,?,?,?)`, m.MessageID, conversation, u.ID, input.ClientMessageID, m.Content, replyJSON, forwardJSON, m.SentAt)
		if err != nil {
			if duplicate(err) {
				return nil, ErrConflict
			}
			return nil, err
		}
		m.Sequence, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE social_conversations_v2 SET updated_at=?,version=version+1 WHERE conversation_id=?`, m.SentAt, conversation); err != nil {
			return nil, err
		}
		topic, title := "DIRECT_MESSAGES", u.GameID+" 发来私信"
		for _, ref := range input.MentionedPlayerRefs {
			if ref == otherRef {
				topic, title = "MENTIONS", u.GameID+" 提及了你"
			}
		}
		if err = AddNotificationV2(ctx, tx, other, "message:"+m.MessageID, topic, title, m.Content, SocialTarget{Kind: "CONVERSATION", ReferenceID: conversation, StateVersion: 0}); err != nil {
			return nil, err
		}
		return m, nil
	})
}
func (s *Store) ReadConversationV2(ctx context.Context, userID, conversation, key, messageID string) (json.RawMessage, error) {
	if !ValidSocialID(messageID) {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, userID, "conversation.read", key, map[string]string{"conversationId": conversation, "messageId": messageID}, func(tx *sql.Tx) (any, error) {
		if _, err := socialMember(ctx, tx, userID, conversation); err != nil {
			return nil, err
		}
		var sequence int64
		if err := tx.QueryRowContext(ctx, `SELECT sequence_id FROM social_messages_v2 WHERE conversation_id=? AND message_id=?`, conversation, messageID).Scan(&sequence); err != nil {
			return nil, socialMissing(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE social_members_v2 SET last_read_sequence=GREATEST(last_read_sequence,?) WHERE conversation_id=? AND user_id=?`, sequence, conversation, userID); err != nil {
			return nil, err
		}
		var count int64
		var latest string
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM social_messages_v2 x JOIN social_members_v2 m ON m.conversation_id=x.conversation_id AND m.user_id=? WHERE x.conversation_id=? AND x.sender_id<>? AND x.sequence_id>m.last_read_sequence`, userID, conversation, userID).Scan(&count); err != nil {
			return nil, err
		}
		if err := tx.QueryRowContext(ctx, `SELECT x.message_id FROM social_messages_v2 x JOIN social_members_v2 m ON m.conversation_id=x.conversation_id AND m.user_id=? WHERE x.conversation_id=? AND x.sequence_id=m.last_read_sequence`, userID, conversation).Scan(&latest); err != nil {
			return nil, err
		}
		return map[string]any{"unreadCount": count, "lastReadMessageId": latest}, nil
	})
}

type SocialPreferencesPatch struct {
	ClientRequestID string `json:"clientRequestId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Enabled         *bool  `json:"enabled,omitempty"`
	ShowPreviews    *bool  `json:"showPreviews,omitempty"`
	DirectMessages  *bool  `json:"directMessages,omitempty"`
	Mentions        *bool  `json:"mentions,omitempty"`
	FollowedPlayers *bool  `json:"followedPlayers,omitempty"`
	Wallet          *bool  `json:"wallet,omitempty"`
	MarketOrders    *bool  `json:"marketOrders,omitempty"`
	Commissions     *bool  `json:"commissions,omitempty"`
	Announcements   *bool  `json:"announcements,omitempty"`
	AppUpdates      *bool  `json:"appUpdates,omitempty"`
}

func defaultSocialPreferences() map[string]bool {
	return map[string]bool{"enabled": true, "showPreviews": true, "directMessages": true, "mentions": true, "followedPlayers": true, "wallet": true, "marketOrders": true, "commissions": true, "announcements": true, "appUpdates": true}
}
func socialPreferencesView(values map[string]bool, version int64) map[string]any {
	result := map[string]any{"version": version}
	for k, v := range values {
		result[k] = v
	}
	return result
}
func (s *Store) NotificationPreferencesV2(ctx context.Context, userID string) (map[string]any, error) {
	values := defaultSocialPreferences()
	var data string
	version := int64(1)
	err := s.DB.QueryRowContext(ctx, `SELECT preferences_json,version FROM social_notification_preferences_v2 WHERE user_id=?`, userID).Scan(&data, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return socialPreferencesView(values, version), nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal([]byte(data), &values); err != nil {
		return nil, err
	}
	return socialPreferencesView(values, version), nil
}
func (s *Store) PatchNotificationPreferencesV2(ctx context.Context, userID string, input SocialPreferencesPatch) (json.RawMessage, error) {
	fields := map[string]*bool{"enabled": input.Enabled, "showPreviews": input.ShowPreviews, "directMessages": input.DirectMessages, "mentions": input.Mentions, "followedPlayers": input.FollowedPlayers, "wallet": input.Wallet, "marketOrders": input.MarketOrders, "commissions": input.Commissions, "announcements": input.Announcements, "appUpdates": input.AppUpdates}
	changed := false
	for _, v := range fields {
		if v != nil {
			changed = true
		}
	}
	if input.ExpectedVersion < 1 || !changed {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, userID, "notifications.preferences", input.ClientRequestID, input, func(tx *sql.Tx) (any, error) {
		values := defaultSocialPreferences()
		encoded, _ := json.Marshal(values)
		if _, err := tx.ExecContext(ctx, `INSERT INTO social_notification_preferences_v2(user_id,preferences_json,version) VALUES(?,?,1) ON DUPLICATE KEY UPDATE user_id=VALUES(user_id)`, userID, encoded); err != nil {
			return nil, err
		}
		var version int64
		var stored string
		if err := tx.QueryRowContext(ctx, `SELECT preferences_json,version FROM social_notification_preferences_v2 WHERE user_id=? FOR UPDATE`, userID).Scan(&stored, &version); err != nil {
			return nil, err
		}
		if version != input.ExpectedVersion {
			return nil, ErrSocialVersion
		}
		if err := json.Unmarshal([]byte(stored), &values); err != nil {
			return nil, err
		}
		for k, v := range fields {
			if v != nil {
				values[k] = *v
			}
		}
		encoded, _ = json.Marshal(values)
		if _, err := tx.ExecContext(ctx, `UPDATE social_notification_preferences_v2 SET preferences_json=?,version=version+1 WHERE user_id=?`, encoded, userID); err != nil {
			return nil, err
		}
		return socialPreferencesView(values, version+1), nil
	})
}
func (s *Store) NotificationsV2(ctx context.Context, userID string, before int64, unreadOnly bool, limit int) ([]SocialNotification, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT sequence_id,notification_id,topic,title,body,target_json,created_at,read_at,system_push FROM social_notifications_v2 WHERE user_id=? AND inbox_visible=TRUE AND (?=0 OR sequence_id<?) AND (?=false OR read_at IS NULL) ORDER BY sequence_id DESC LIMIT ?`, userID, before, before, unreadOnly, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SocialNotification{}
	for rows.Next() {
		var n SocialNotification
		var target string
		var read sql.NullTime
		if err = rows.Scan(&n.Sequence, &n.NotificationID, &n.Topic, &n.Title, &n.Body, &target, &n.CreatedAt, &read, &n.SystemPush); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(target), &n.Target); err != nil {
			return nil, err
		}
		if read.Valid {
			n.ReadAt = &read.Time
		}
		if n.Topic == "STORE_ORDERS" || n.Topic == "MARKET_ORDERS" || n.Topic == "COMMISSIONS" {
			n.Body = customerCommerceTextV206(n.Body)
		}
		result = append(result, n)
	}
	return result, rows.Err()
}
func (s *Store) ReadNotificationsV2(ctx context.Context, userID, key string, ids []string) (json.RawMessage, error) {
	if len(ids) < 1 || len(ids) > 100 {
		return nil, ErrSocialInvalid
	}
	unique := map[string]bool{}
	for _, id := range ids {
		if !ValidSocialID(id) {
			return nil, ErrSocialInvalid
		}
		unique[id] = true
	}
	ordered := []string{}
	for id := range unique {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	return s.socialMutate(ctx, userID, "notifications.read", key, ordered, func(tx *sql.Tx) (any, error) {
		count := 0
		for _, id := range ordered {
			var read sql.NullTime
			if err := tx.QueryRowContext(ctx, `SELECT read_at FROM social_notifications_v2 WHERE user_id=? AND notification_id=? FOR UPDATE`, userID, id).Scan(&read); err != nil {
				return nil, socialMissing(err)
			}
			if !read.Valid {
				count++
			}
			if _, err := tx.ExecContext(ctx, `UPDATE social_notifications_v2 SET read_at=COALESCE(read_at,UTC_TIMESTAMP(6)) WHERE user_id=? AND notification_id=?`, userID, id); err != nil {
				return nil, err
			}
		}
		return map[string]int{"readCount": count}, nil
	})
}
func validateSocialAnnouncement(input SocialAnnouncementWrite) error {
	if !ValidSocialID(input.ClientRequestID) || !ValidSocialText(input.Title, 1, 100) || !ValidSocialText(input.Summary, 1, 300) || len(input.ContentBlocks) < 1 || len(input.ContentBlocks) > 100 || (input.Priority != "NORMAL" && input.Priority != "IMPORTANT") {
		return ErrSocialInvalid
	}
	ids := map[string]bool{}
	for _, b := range input.ContentBlocks {
		if !ValidSocialID(b.BlockID) || ids[b.BlockID] {
			return ErrSocialInvalid
		}
		ids[b.BlockID] = true
		switch b.Type {
		case "HEADING":
			if !ValidSocialText(b.Heading, 1, 100) {
				return ErrSocialInvalid
			}
		case "PARAGRAPH":
			if !ValidSocialText(b.Text, 1, 3000) {
				return ErrSocialInvalid
			}
		case "IMAGE":
			if b.AssetID == nil || !ValidSocialID(*b.AssetID) || !ValidSocialText(b.AltText, 0, 200) {
				return ErrSocialInvalid
			}
		case "KEY_VALUE_LIST":
			if len(b.Rows) < 1 || len(b.Rows) > 30 {
				return ErrSocialInvalid
			}
			for _, row := range b.Rows {
				if !ValidSocialText(row.Label, 1, 80) || !ValidSocialText(row.Value, 1, 500) {
					return ErrSocialInvalid
				}
			}
		default:
			return ErrSocialInvalid
		}
	}
	return nil
}
func announcementAssetIDsV2(cover *string, blocks []SocialContentBlock) []string {
	unique := map[string]bool{}
	if cover != nil {
		unique[*cover] = true
	}
	for _, b := range blocks {
		if b.Type == "IMAGE" && b.AssetID != nil {
			unique[*b.AssetID] = true
		}
	}
	result := []string{}
	for id := range unique {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
func (s *Store) validateAnnouncementAssetsV2(ctx context.Context, userID, announcementID string, input SocialAnnouncementWrite) error {
	ids := []string{}
	if input.CoverAssetID != nil {
		if !ValidSocialID(*input.CoverAssetID) {
			return ErrSocialInvalid
		}
		ids = append(ids, *input.CoverAssetID)
	}
	for _, b := range input.ContentBlocks {
		if b.Type == "IMAGE" && b.AssetID != nil {
			ids = append(ids, *b.AssetID)
		}
	}
	for _, id := range ids {
		if announcementID != "" {
			var count int
			if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM asset_bindings_v2 b JOIN asset_uploads_v2 a ON a.asset_id=b.asset_id WHERE b.business_type='ANNOUNCEMENT' AND b.business_ref=? AND b.asset_id=? AND a.purpose='ANNOUNCEMENT_MEDIA' AND a.status='READY' AND a.removed_at IS NULL`, announcementID, id).Scan(&count); err != nil {
				return err
			}
			if count == 1 {
				continue
			}
		}
		if err := s.ValidateOwnedAssetsV2(ctx, userID, []string{id}, "ANNOUNCEMENT_MEDIA"); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) WriteAnnouncementV2(ctx context.Context, userID, announcementID string, input SocialAnnouncementWrite) (json.RawMessage, error) {
	if err := validateSocialAnnouncement(input); err != nil {
		return nil, err
	}
	if err := s.validateAnnouncementAssetsV2(ctx, userID, announcementID, input); err != nil {
		return nil, err
	}
	if announcementID != "" && input.ExpectedVersion < 1 {
		return nil, ErrSocialInvalid
	}
	action := "announcement.create"
	if announcementID != "" {
		action = "announcement.edit"
	}
	return s.socialMutate(ctx, userID, action, input.ClientRequestID, struct {
		ID    string
		Input SocialAnnouncementWrite
	}{announcementID, input}, func(tx *sql.Tx) (any, error) {
		now := time.Now().UTC()
		version := int64(1)
		status := "DRAFT"
		publishedVersion := int64(0)
		var published sql.NullString
		if announcementID == "" {
			announcementID = ID("ann_")
		} else {
			if err := tx.QueryRowContext(ctx, `SELECT version,state,published_version,published_json FROM social_announcements_v2 WHERE announcement_id=? FOR UPDATE`, announcementID).Scan(&version, &status, &publishedVersion, &published); err != nil {
				return nil, socialMissing(err)
			}
			if version != input.ExpectedVersion {
				return nil, ErrSocialVersion
			}
			version++
		}
		value := SocialAnnouncement{AnnouncementID: announcementID, Title: input.Title, Summary: input.Summary, ContentBlocks: input.ContentBlocks, CoverAssetID: input.CoverAssetID, Pinned: input.Pinned, Priority: input.Priority, UpdatedAt: now, Version: version}
		bindings := announcementAssetIDsV2(value.CoverAssetID, value.ContentBlocks)
		if published.Valid {
			var old SocialAnnouncement
			if err := json.Unmarshal([]byte(published.String), &old); err != nil {
				return nil, err
			}
			bindings = append(bindings, announcementAssetIDsV2(old.CoverAssetID, old.ContentBlocks)...)
		}
		if err := s.SetAssetBindingsV2(ctx, tx, userID, "ANNOUNCEMENT", announcementID, bindings); err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if action == "announcement.create" {
			_, err = tx.ExecContext(ctx, `INSERT INTO social_announcements_v2(announcement_id,draft_json,state,version,created_at,updated_at) VALUES(?,?,'DRAFT',1,?,?)`, announcementID, encoded, now, now)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE social_announcements_v2 SET draft_json=?,version=?,updated_at=? WHERE announcement_id=?`, encoded, version, now, announcementID)
		}
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))`, userID, action, announcementID); err != nil {
			return nil, err
		}
		dirty := version != publishedVersion
		value.Status = status
		value.PublishedVersion = &publishedVersion
		value.HasUnpublishedChanges = &dirty
		return value, nil
	})
}
func (s *Store) ChangeAnnouncementV2(ctx context.Context, userID, announcementID, key string, version int64, publish bool, reason string) (json.RawMessage, error) {
	if !ValidSocialID(announcementID) || version < 1 || !publish && !ValidSocialText(reason, 1, 500) {
		return nil, ErrSocialInvalid
	}
	action := "announcement.unpublish"
	if publish {
		action = "announcement.publish"
	}
	return s.socialMutate(ctx, userID, action, key, struct {
		ID      string
		Version int64
		Reason  string
	}{announcementID, version, reason}, func(tx *sql.Tx) (any, error) {
		var stored string
		var actual, publishedVersion int64
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT draft_json,version,published_version,state FROM social_announcements_v2 WHERE announcement_id=? FOR UPDATE`, announcementID).Scan(&stored, &actual, &publishedVersion, &status); err != nil {
			return nil, socialMissing(err)
		}
		if actual != version {
			return nil, ErrSocialVersion
		}
		var value SocialAnnouncement
		if err := json.Unmarshal([]byte(stored), &value); err != nil {
			return nil, err
		}
		if err := s.SetAssetBindingsV2(ctx, tx, userID, "ANNOUNCEMENT", announcementID, announcementAssetIDsV2(value.CoverAssetID, value.ContentBlocks)); err != nil {
			return nil, err
		}
		actual++
		now := time.Now().UTC()
		value.UpdatedAt = now
		value.Version = actual
		if publish {
			status = "PUBLISHED"
			publishedVersion = actual
			value.PublishedAt = &now
		} else {
			status = "WITHDRAWN"
		}
		draftJSON, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if publish {
			_, err = tx.ExecContext(ctx, `UPDATE social_announcements_v2 SET draft_json=?,published_json=?,version=?,published_version=?,state=?,updated_at=? WHERE announcement_id=?`, draftJSON, draftJSON, actual, publishedVersion, status, now, announcementID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE social_announcements_v2 SET draft_json=?,published_json=NULL,version=?,state=?,updated_at=? WHERE announcement_id=?`, draftJSON, actual, status, now, announcementID)
		}
		if err != nil {
			return nil, err
		}
		if publish {
			target, _ := json.Marshal(SocialTarget{Kind: "ANNOUNCEMENT", ReferenceID: announcementID, StateVersion: actual})
			_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO social_notifications_v2(notification_id,user_id,event_key,topic,title,body,target_json,created_at) SELECT CONCAT('not_',REPLACE(UUID(),'-','')),id,?,'ANNOUNCEMENTS',?,?,?,UTC_TIMESTAMP(6) FROM identities WHERE status='active'`, announcementID+":"+strconvInt(actual), value.Title, value.Summary, target)
			if err != nil {
				return nil, err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))`, userID, action, announcementID); err != nil {
			return nil, err
		}
		dirty := actual != publishedVersion
		value.Status = status
		value.PublishedVersion = &publishedVersion
		value.HasUnpublishedChanges = &dirty
		return value, nil
	})
}
func strconvInt(value int64) string { return strconv.FormatInt(value, 10) }
func (s *Store) AnnouncementsV2(ctx context.Context, admin bool, before int64, limit int) ([]SocialAnnouncement, error) {
	query := `SELECT sequence_id,draft_json,published_json,state,version,published_version FROM social_announcements_v2 WHERE (?=0 OR sequence_id<?)`
	if !admin {
		query += ` AND state='PUBLISHED' AND published_json IS NOT NULL`
	}
	query += ` ORDER BY sequence_id DESC LIMIT ?`
	rows, err := s.DB.QueryContext(ctx, query, before, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SocialAnnouncement{}
	for rows.Next() {
		var sequence, version, publishedVersion int64
		var draft, status string
		var published sql.NullString
		if err = rows.Scan(&sequence, &draft, &published, &status, &version, &publishedVersion); err != nil {
			return nil, err
		}
		raw := draft
		if !admin {
			raw = published.String
		}
		var a SocialAnnouncement
		if err = json.Unmarshal([]byte(raw), &a); err != nil {
			return nil, err
		}
		a.Sequence = sequence
		if admin {
			dirty := version != publishedVersion
			a.Status = status
			a.Version = version
			a.PublishedVersion = &publishedVersion
			a.HasUnpublishedChanges = &dirty
		}
		result = append(result, a)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range result {
		if err = s.resolveAnnouncementMediaV2(ctx, &result[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (s *Store) AnnouncementV2(ctx context.Context, announcementID string, admin bool) (SocialAnnouncement, error) {
	var a SocialAnnouncement
	var draft, status string
	var published sql.NullString
	var version, publishedVersion int64
	err := s.DB.QueryRowContext(ctx, `SELECT draft_json,published_json,state,version,published_version FROM social_announcements_v2 WHERE announcement_id=?`, announcementID).Scan(&draft, &published, &status, &version, &publishedVersion)
	if err != nil {
		return a, socialMissing(err)
	}
	if !admin && (status != "PUBLISHED" || !published.Valid) {
		return a, ErrSocialNotFound
	}
	raw := draft
	if !admin {
		raw = published.String
	}
	if err = json.Unmarshal([]byte(raw), &a); err != nil {
		return a, err
	}
	if admin {
		dirty := version != publishedVersion
		a.Status = status
		a.Version = version
		a.PublishedVersion = &publishedVersion
		a.HasUnpublishedChanges = &dirty
	}
	if err = s.resolveAnnouncementMediaV2(ctx, &a); err != nil {
		return a, err
	}
	return a, nil
}

func (s *Store) resolveAnnouncementMediaV2(ctx context.Context, value *SocialAnnouncement) error {
	for _, id := range announcementAssetIDsV2(value.CoverAssetID, value.ContentBlocks) {
		asset, err := s.AssetForBindingV2(ctx, "ANNOUNCEMENT", value.AnnouncementID, id)
		if err != nil {
			return err
		}
		value.Media = append(value.Media, asset)
		if value.CoverAssetID != nil && *value.CoverAssetID == id {
			value.Cover = asset
		}
	}
	return nil
}

var socialMentionV2 = regexp.MustCompile(`@([A-Za-z0-9_]{1,32})\b`)

// Call after the existing public-chat transaction commits, with its immutable
// source/client key. This preserves the current Core/WS delivery protocol.
func (s *Store) PublicSocialNotificationsV2(ctx context.Context, sourceID, clientMessageID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var messageID, senderRef, senderName, content string
	var sentAt time.Time
	if err = tx.QueryRowContext(ctx, `SELECT message_id,sender_ref,sender_name,content,sent_at FROM chat_messages_next WHERE source_id=? AND client_message_id=?`, sourceID, clientMessageID).Scan(&messageID, &senderRef, &senderName, &content, &sentAt); err != nil {
		return socialMissing(err)
	}
	recipients := map[string]string{}
	rows, err := tx.QueryContext(ctx, `SELECT f.follower_id FROM social_follows_v2 f JOIN identities author ON author.id=f.followed_id JOIN identities recipient ON recipient.id=f.follower_id WHERE author.player_ref=? AND recipient.status='active' AND f.created_at<=?`, senderRef, sentAt)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		recipients[id] = "FOLLOWED_PLAYERS"
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, match := range socialMentionV2.FindAllStringSubmatch(content, 20) {
		name := strings.ToLower(match[1])
		if seen[name] {
			continue
		}
		seen[name] = true
		var id, ref string
		err = tx.QueryRowContext(ctx, `SELECT id,player_ref FROM identities WHERE LOWER(game_id)=? AND status='active'`, name).Scan(&id, &ref)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if ref != senderRef {
			recipients[id] = "MENTIONS"
		}
	}
	ids := []string{}
	if err = publicExplicitRecipientsV2(ctx, tx, messageID, senderRef, recipients); err != nil {
		return err
	}
	for id := range recipients {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		topic := recipients[id]
		title := senderName + " 发布了消息"
		if topic == "MENTIONS" {
			title = senderName + " 提及了你"
		}
		if err = AddNotificationV2(ctx, tx, id, "public-message:"+messageID, topic, title, content, SocialTarget{Kind: "PUBLIC_CHAT", ReferenceID: messageID, StateVersion: 0}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Reconcile a bounded page of committed public messages. Each durable cursor
// step occurs only after its notifications commit; replay is safe after a crash.
func (s *Store) ReconcilePublicSocialNotificationsV2(ctx context.Context) error {
	for n := 0; n < 50; n++ {
		processed, err := s.reconcilePublicSocialOneV2(ctx)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}
	}
	return nil
}
func (s *Store) reconcilePublicSocialOneV2(ctx context.Context) (bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO social_reconcile_v2(worker_name,last_sequence,updated_at) VALUES('public_chat',0,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE worker_name=VALUES(worker_name)`); err != nil {
		return false, err
	}
	var cursor, sequence int64
	var source, clientID string
	if err = tx.QueryRowContext(ctx, `SELECT last_sequence FROM social_reconcile_v2 WHERE worker_name='public_chat' FOR UPDATE`).Scan(&cursor); err != nil {
		return false, err
	}
	err = tx.QueryRowContext(ctx, `SELECT sequence_id,source_id,client_message_id FROM chat_messages_next WHERE sequence_id>? ORDER BY sequence_id LIMIT 1`, cursor).Scan(&sequence, &source, &clientID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, tx.Commit()
	}
	if err != nil {
		return false, err
	}
	if err = s.PublicSocialNotificationsV2(ctx, source, clientID); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE social_reconcile_v2 SET last_sequence=?,updated_at=UTC_TIMESTAMP(6) WHERE worker_name='public_chat'`, sequence); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
