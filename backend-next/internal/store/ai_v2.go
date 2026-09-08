package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrAIQuotaV2 = errors.New("ai quota exhausted")
var ErrAIBusyV2 = errors.New("ai generation in progress")
var ErrAIConversationV2 = errors.New("ai conversation changed")
var ErrAINotFoundV2 = errors.New("ai request not found")

type AIPlanV2 struct {
	PlanID         string `json:"planId"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Price          string `json:"price"`
	Currency       string `json:"currency"`
	QuotaPerWindow int    `json:"quotaPerWindow"`
	WindowHours    int    `json:"windowHours"`
	DurationDays   int    `json:"durationDays"`
	ModelTier      string `json:"modelTier"`
	Active         bool   `json:"active"`
	Purchasable    bool   `json:"purchasable"`
}
type AIQuotaV2 struct {
	Used        int       `json:"used"`
	Limit       int       `json:"limit"`
	Remaining   int       `json:"remaining"`
	WindowHours int       `json:"windowHours"`
	ResetsAt    time.Time `json:"resetsAt"`
	Unlimited   bool      `json:"unlimited"`
	Reserved    int       `json:"reserved"`
}
type AIConversationV2 struct {
	ID        string    `json:"conversationId"`
	Active    bool      `json:"active"`
	StartedAt time.Time `json:"startedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type AISourceV2 struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Origin string `json:"origin"`
}
type AIMessageV2 struct {
	ID             string       `json:"messageId"`
	ConversationID string       `json:"conversationId"`
	Role           string       `json:"role"`
	Content        string       `json:"content"`
	CreatedAt      time.Time    `json:"createdAt"`
	Status         string       `json:"status"`
	FinishReason   string       `json:"finishReason,omitempty"`
	Sources        []AISourceV2 `json:"sources"`
	SearchUsed     bool         `json:"searchUsed"`
}
type AIExchangeV2 struct {
	ID, UserID, ConversationID, ClientMessageID, Fingerprint, Input string
	UserMessageID, AssistantMessageID, Status, Progress, Answer     string
	Sources                                                         []AISourceV2
	SearchUsed                                                      bool
	FinishReason, ErrorCode, ProviderID, Model, QuotaState          string
	WindowStart                                                     time.Time
	WindowHours                                                     int
	InputTokens, OutputTokens                                       int64
	CreatedAt, UpdatedAt, DeadlineAt                                time.Time
}
type AIPendingV2 struct {
	ClientMessageID    string    `json:"clientMessageId"`
	Content            string    `json:"content"`
	UserMessageID      string    `json:"userMessageId"`
	AssistantMessageID string    `json:"assistantMessageId"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	RetryAfterSeconds  int       `json:"retryAfterSeconds"`
}
type AIStateV2 struct {
	Plan         AIPlanV2         `json:"plan"`
	Quota        AIQuotaV2        `json:"quota"`
	Conversation AIConversationV2 `json:"conversation"`
	Pending      *AIPendingV2     `json:"pendingRequest"`
}
type AIPolicyV2 struct {
	FreeQuota, WindowHours int
	AdminExempt            bool
}

func AIWindowStartV2(now time.Time, hours int) time.Time {
	seconds := int64(hours) * 3600
	return time.Unix(now.Unix()-now.Unix()%seconds, 0).UTC()
}
func (s *Store) AIPlansV2(ctx context.Context, policy AIPolicyV2) ([]AIPlanV2, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT plan_id,code,name,description,CAST(price AS CHAR),quota_limit,window_hours,duration_days,model_tier,active FROM ai_plans_v2 ORDER BY sort_order,plan_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AIPlanV2{}
	for rows.Next() {
		var p AIPlanV2
		if err = rows.Scan(&p.PlanID, &p.Code, &p.Name, &p.Description, &p.Price, &p.QuotaPerWindow, &p.WindowHours, &p.DurationDays, &p.ModelTier, &p.Active); err != nil {
			return nil, err
		}
		p.Currency = "CREDIT"
		if p.Code == "free" {
			p.QuotaPerWindow = policy.FreeQuota
			p.WindowHours = policy.WindowHours
			p.Active = true
			p.Price = "0.00"
		} else {
			p.Active = false
			p.Price = "9999999.00"
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
func aiProfileLockedV2(ctx context.Context, tx *sql.Tx, user string, now time.Time) (string, sql.NullString, sql.NullString, error) {
	var conv string
	var active, last sql.NullString
	var identityStatus string
	if err := tx.QueryRowContext(ctx, "SELECT status FROM identities WHERE id=? FOR UPDATE", user).Scan(&identityStatus); err != nil {
		return conv, active, last, err
	}
	if identityStatus != "active" {
		return conv, active, last, ErrUnauthorized
	}
	err := tx.QueryRowContext(ctx, "SELECT conversation_id,active_request_id,last_request_id FROM ai_profiles_v2 WHERE user_id=?", user).Scan(&conv, &active, &last)
	if err == nil {
		err = tx.QueryRowContext(ctx, "SELECT conversation_id,active_request_id,last_request_id FROM ai_profiles_v2 WHERE user_id=? FOR UPDATE", user).Scan(&conv, &active, &last)
	}
	if errors.Is(err, sql.ErrNoRows) {
		conv = ID("aic_")
		if _, err = tx.ExecContext(ctx, "INSERT INTO ai_conversations_v2 VALUES(?,?,TRUE,?,?)", conv, user, now, now); err != nil {
			return conv, active, last, err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO ai_profiles_v2 VALUES(?,?,NULL,NULL,?)", user, conv, now)
	}
	if err != nil {
		return conv, active, last, err
	}
	if active.Valid {
		var state string
		var deadline time.Time
		if err = tx.QueryRowContext(ctx, "SELECT status,deadline_at FROM ai_requests_v2 WHERE request_id=? AND user_id=? FOR UPDATE", active.String, user).Scan(&state, &deadline); err != nil {
			return conv, active, last, err
		}
		if (state == "pending" || state == "streaming") && !deadline.After(now) {
			if _, err = tx.ExecContext(ctx, "UPDATE ai_requests_v2 SET status='unknown',progress='unknown',error_code='AI_RESULT_UNKNOWN',updated_at=? WHERE request_id=?", now, active.String); err != nil {
				return conv, active, last, err
			}
			if _, err = tx.ExecContext(ctx, "UPDATE ai_messages_v2 SET status='unknown' WHERE request_id=? AND role='assistant'", active.String); err != nil {
				return conv, active, last, err
			}
			state = "unknown"
		}
		if state != "pending" && state != "streaming" {
			if _, err = tx.ExecContext(ctx, "UPDATE ai_profiles_v2 SET active_request_id=NULL,updated_at=? WHERE user_id=?", now, user); err != nil {
				return conv, active, last, err
			}
			active = sql.NullString{}
		}
	}
	return conv, active, last, nil
}
func aiUnlimitedV2(ctx context.Context, tx *sql.Tx, user string, policy AIPolicyV2) (bool, error) {
	if !policy.AdminExempt {
		return false, nil
	}
	var count int
	err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_permissions WHERE user_id=? AND permission='platform.admin'", user).Scan(&count)
	return count > 0, err
}
func aiQuotaLockedV2(ctx context.Context, tx *sql.Tx, user string, policy AIPolicyV2, now time.Time) (AIQuotaV2, error) {
	start := AIWindowStartV2(now, policy.WindowHours)
	q := AIQuotaV2{Limit: policy.FreeQuota, WindowHours: policy.WindowHours, ResetsAt: start.Add(time.Duration(policy.WindowHours) * time.Hour)}
	var err error
	q.Unlimited, err = aiUnlimitedV2(ctx, tx, user, policy)
	if err != nil {
		return q, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO ai_quota_v2 VALUES(?,?,?,0,0,?)", user, start, policy.WindowHours, now); err != nil {
		return q, err
	}
	err = tx.QueryRowContext(ctx, "SELECT used_count,reserved_count FROM ai_quota_v2 WHERE user_id=? AND window_start=? AND window_hours=? FOR UPDATE", user, start, policy.WindowHours).Scan(&q.Used, &q.Reserved)
	q.Remaining = max(0, q.Limit-q.Used-q.Reserved)
	if q.Unlimited {
		q.Remaining = q.Limit
	}
	return q, err
}
func (s *Store) AIStateV2(ctx context.Context, user string, policy AIPolicyV2, now time.Time) (AIStateV2, error) {
	var state AIStateV2
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return state, err
	}
	defer tx.Rollback()
	conv, _, last, err := aiProfileLockedV2(ctx, tx, user, now)
	if err != nil {
		return state, err
	}
	err = tx.QueryRowContext(ctx, "SELECT conversation_id,active,created_at,updated_at FROM ai_conversations_v2 WHERE conversation_id=? AND user_id=?", conv, user).Scan(&state.Conversation.ID, &state.Conversation.Active, &state.Conversation.StartedAt, &state.Conversation.UpdatedAt)
	if err != nil {
		return state, err
	}
	state.Plan = AIPlanV2{PlanID: "plan_free", Code: "free", Name: "ProMax", Description: "默认", Price: "0.00", Currency: "CREDIT", QuotaPerWindow: policy.FreeQuota, WindowHours: policy.WindowHours, ModelTier: "flash", Active: true}
	state.Quota, err = aiQuotaLockedV2(ctx, tx, user, policy, now)
	if err != nil {
		return state, err
	}
	if last.Valid {
		var p AIPendingV2
		err = tx.QueryRowContext(ctx, "SELECT client_message_id,input_content,user_message_id,assistant_message_id,status,created_at,updated_at FROM ai_requests_v2 WHERE request_id=? AND user_id=? AND conversation_id=?", last.String, user, conv).Scan(&p.ClientMessageID, &p.Content, &p.UserMessageID, &p.AssistantMessageID, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		if err == nil && p.Status != "completed" && p.Status != "failed" {
			p.RetryAfterSeconds = 2
			state.Pending = &p
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return state, err
		}
	}
	return state, tx.Commit()
}

const aiExchangeColumnsV2 = "request_id,user_id,conversation_id,client_message_id,fingerprint,input_content,user_message_id,assistant_message_id,status,progress,answer,sources_json,search_used,finish_reason,error_code,provider_response_id,model,window_start,window_hours,quota_state,input_tokens,output_tokens,created_at,updated_at,deadline_at"

type aiScannerV2 interface{ Scan(...any) error }

func scanAIExchangeV2(row aiScannerV2) (AIExchangeV2, error) {
	var e AIExchangeV2
	var sources string
	err := row.Scan(&e.ID, &e.UserID, &e.ConversationID, &e.ClientMessageID, &e.Fingerprint, &e.Input, &e.UserMessageID, &e.AssistantMessageID, &e.Status, &e.Progress, &e.Answer, &sources, &e.SearchUsed, &e.FinishReason, &e.ErrorCode, &e.ProviderID, &e.Model, &e.WindowStart, &e.WindowHours, &e.QuotaState, &e.InputTokens, &e.OutputTokens, &e.CreatedAt, &e.UpdatedAt, &e.DeadlineAt)
	if err != nil {
		return e, err
	}
	e.Sources = []AISourceV2{}
	if err = json.Unmarshal([]byte(sources), &e.Sources); err != nil {
		return e, err
	}
	return e, nil
}
func (s *Store) AIExchangeV2(ctx context.Context, user, id string) (AIExchangeV2, error) {
	e, err := scanAIExchangeV2(s.DB.QueryRowContext(ctx, "SELECT "+aiExchangeColumnsV2+" FROM ai_requests_v2 WHERE request_id=? AND user_id=?", id, user))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrAINotFoundV2
	}
	return e, err
}
func (s *Store) BeginAIV2(ctx context.Context, user, key, content, model string, policy AIPolicyV2, now time.Time, lifetime time.Duration) (AIExchangeV2, bool, error) {
	var e AIExchangeV2
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return e, false, err
	}
	defer tx.Rollback()
	conv, active, _, err := aiProfileLockedV2(ctx, tx, user, now)
	if err != nil {
		return e, false, err
	}
	e, err = scanAIExchangeV2(tx.QueryRowContext(ctx, "SELECT "+aiExchangeColumnsV2+" FROM ai_requests_v2 WHERE user_id=? AND client_message_id=? FOR UPDATE", user, key))
	if err == nil {
		if e.Fingerprint != Digest([]byte(content)) {
			return e, false, ErrConflict
		}
		if e.ConversationID != conv {
			return e, false, ErrAIConversationV2
		}
		return e, false, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return e, false, err
	}
	if active.Valid {
		return e, false, ErrAIBusyV2
	}
	quota, err := aiQuotaLockedV2(ctx, tx, user, policy, now)
	if err != nil {
		return e, false, err
	}
	if quota.Remaining <= 0 && !quota.Unlimited {
		return e, false, ErrAIQuotaV2
	}
	e = AIExchangeV2{ID: ID("air_"), UserID: user, ConversationID: conv, ClientMessageID: key, Fingerprint: Digest([]byte(content)), Input: content, UserMessageID: ID("aim_"), AssistantMessageID: ID("aim_"), Status: "pending", Progress: "queued", Sources: []AISourceV2{}, Model: model, WindowStart: AIWindowStartV2(now, policy.WindowHours), WindowHours: policy.WindowHours, QuotaState: "reserved", CreatedAt: now, UpdatedAt: now, DeadlineAt: now.Add(lifetime)}
	if quota.Unlimited {
		e.QuotaState = "exempt"
	} else if _, err = tx.ExecContext(ctx, "UPDATE ai_quota_v2 SET reserved_count=reserved_count+1,updated_at=? WHERE user_id=? AND window_start=? AND window_hours=?", now, user, e.WindowStart, e.WindowHours); err != nil {
		return e, false, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO ai_requests_v2 ("+aiExchangeColumnsV2+") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", e.ID, user, conv, key, e.Fingerprint, content, e.UserMessageID, e.AssistantMessageID, e.Status, e.Progress, "", "[]", false, "", "", "", model, e.WindowStart, e.WindowHours, e.QuotaState, 0, 0, now, now, e.DeadlineAt)
	if err != nil {
		return e, false, err
	}
	for _, m := range []struct{ id, role, content, status string }{{e.UserMessageID, "user", content, "completed"}, {e.AssistantMessageID, "assistant", "", "pending"}} {
		if _, err = tx.ExecContext(ctx, "INSERT INTO ai_messages_v2(message_id,user_id,conversation_id,request_id,role,content,status,finish_reason,sources_json,search_used,created_at) VALUES(?,?,?,?,?,?,?,'','[]',FALSE,?)", m.id, user, conv, e.ID, m.role, m.content, m.status, now); err != nil {
			return e, false, err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE ai_profiles_v2 SET active_request_id=?,last_request_id=?,updated_at=? WHERE user_id=?", e.ID, e.ID, now, user); err != nil {
		return e, false, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE ai_conversations_v2 SET updated_at=? WHERE conversation_id=?", now, conv); err != nil {
		return e, false, err
	}
	return e, true, tx.Commit()
}
func (s *Store) CheckpointAIV2(ctx context.Context, e AIExchangeV2) error {
	encoded, err := json.Marshal(e.Sources)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var identityID string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? FOR UPDATE", e.UserID).Scan(&identityID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "UPDATE ai_requests_v2 SET status='streaming',progress=?,answer=?,sources_json=?,search_used=?,provider_response_id=?,updated_at=UTC_TIMESTAMP(6) WHERE request_id=? AND user_id=? AND status IN ('pending','streaming')", e.Progress, e.Answer, string(encoded), e.SearchUsed, e.ProviderID, e.ID, e.UserID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrAIConversationV2
	}
	if _, err = tx.ExecContext(ctx, "UPDATE ai_messages_v2 SET content=?,status='streaming',sources_json=?,search_used=? WHERE request_id=? AND user_id=? AND role='assistant'", e.Answer, string(encoded), e.SearchUsed, e.ID, e.UserID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) FinishAIV2(ctx context.Context, e AIExchangeV2, status, code, reason string, inputTokens, outputTokens int64) error {
	if status != "completed" && status != "failed" && status != "unknown" && status != "incomplete" {
		return errors.New("invalid ai terminal state")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var identityID string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? FOR UPDATE", e.UserID).Scan(&identityID); err != nil {
		return err
	}
	var active sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT active_request_id FROM ai_profiles_v2 WHERE user_id=? FOR UPDATE", e.UserID).Scan(&active); err != nil {
		return err
	}
	current, err := scanAIExchangeV2(tx.QueryRowContext(ctx, "SELECT "+aiExchangeColumnsV2+" FROM ai_requests_v2 WHERE request_id=? AND user_id=? FOR UPDATE", e.ID, e.UserID))
	if err != nil {
		return err
	}
	if current.Status != "pending" && current.Status != "streaming" {
		return tx.Commit()
	}
	quotaState := current.QuotaState
	if quotaState == "reserved" && (status == "completed" || status == "failed") {
		increment := 0
		quotaState = "released"
		if status == "completed" {
			increment = 1
			quotaState = "used"
		}
		result, err := tx.ExecContext(ctx, "UPDATE ai_quota_v2 SET used_count=used_count+?,reserved_count=reserved_count-1,updated_at=UTC_TIMESTAMP(6) WHERE user_id=? AND window_start=? AND window_hours=? AND reserved_count>0", increment, e.UserID, current.WindowStart, current.WindowHours)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("ai quota reservation missing")
		}
	}
	if !strings.HasPrefix(e.Answer, current.Answer) {
		return errors.New("ai answer cannot retract persisted content")
	}
	encoded, err := json.Marshal(e.Sources)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE ai_requests_v2 SET status=?,progress=?,answer=?,sources_json=?,search_used=?,finish_reason=?,error_code=?,provider_response_id=?,quota_state=?,input_tokens=?,output_tokens=?,updated_at=UTC_TIMESTAMP(6) WHERE request_id=? AND user_id=?", status, status, e.Answer, string(encoded), e.SearchUsed, reason, code, e.ProviderID, quotaState, inputTokens, outputTokens, e.ID, e.UserID)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE ai_messages_v2 SET content=?,status=?,finish_reason=?,sources_json=?,search_used=? WHERE request_id=? AND user_id=? AND role='assistant'", e.Answer, status, reason, string(encoded), e.SearchUsed, e.ID, e.UserID); err != nil {
		return err
	}
	if active.Valid && active.String == e.ID {
		if _, err = tx.ExecContext(ctx, "UPDATE ai_profiles_v2 SET active_request_id=NULL,updated_at=UTC_TIMESTAMP(6) WHERE user_id=?", e.UserID); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE ai_conversations_v2 SET updated_at=UTC_TIMESTAMP(6) WHERE conversation_id=? AND user_id=?", e.ConversationID, e.UserID); err != nil {
		return err
	}
	return tx.Commit()
}
func AIUserMessageV2(e AIExchangeV2) AIMessageV2 {
	return AIMessageV2{ID: e.UserMessageID, ConversationID: e.ConversationID, Role: "user", Content: e.Input, CreatedAt: e.CreatedAt, Status: "completed", Sources: []AISourceV2{}}
}
func AIAssistantMessageV2(e AIExchangeV2) AIMessageV2 {
	return AIMessageV2{ID: e.AssistantMessageID, ConversationID: e.ConversationID, Role: "assistant", Content: e.Answer, CreatedAt: e.CreatedAt, Status: e.Status, FinishReason: e.FinishReason, Sources: e.Sources, SearchUsed: e.SearchUsed}
}
func (s *Store) AIMessagesV2(ctx context.Context, user, conversation, before string, limit int) ([]AIMessageV2, error) {
	seq := int64(0)
	if before != "" {
		if err := s.DB.QueryRowContext(ctx, "SELECT sequence_id FROM ai_messages_v2 WHERE message_id=? AND user_id=? AND conversation_id=?", before, user, conversation).Scan(&seq); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrAINotFoundV2
			}
			return nil, err
		}
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT message_id,conversation_id,role,content,created_at,status,finish_reason,sources_json,search_used FROM ai_messages_v2 WHERE user_id=? AND conversation_id=? AND (?=0 OR sequence_id<?) AND (role='user' OR content<>'') ORDER BY sequence_id DESC LIMIT ?", user, conversation, seq, seq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AIMessageV2{}
	for rows.Next() {
		var m AIMessageV2
		var sources string
		if err = rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt, &m.Status, &m.FinishReason, &sources, &m.SearchUsed); err != nil {
			return nil, err
		}
		m.Sources = []AISourceV2{}
		if err = json.Unmarshal([]byte(sources), &m.Sources); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}
func (s *Store) AIContextV2(ctx context.Context, e AIExchangeV2, limit int) ([]AIMessageV2, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT m.role,m.content FROM ai_messages_v2 m JOIN ai_requests_v2 r ON r.request_id=m.request_id WHERE m.user_id=? AND m.conversation_id=? AND r.status='completed' AND r.request_id<>? ORDER BY m.sequence_id DESC LIMIT ?", e.UserID, e.ConversationID, e.ID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AIMessageV2{}
	for rows.Next() {
		var m AIMessageV2
		if err = rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) > 0 && result[0].Role == "assistant" {
		result = result[1:]
	}
	return result, nil
}
func (s *Store) ResetAIV2(ctx context.Context, user string, now time.Time) (AIConversationV2, error) {
	var next AIConversationV2
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return next, err
	}
	defer tx.Rollback()
	old, active, _, err := aiProfileLockedV2(ctx, tx, user, now)
	if err != nil {
		return next, err
	}
	if active.Valid {
		return next, ErrAIBusyV2
	}
	next = AIConversationV2{ID: ID("aic_"), Active: true, StartedAt: now, UpdatedAt: now}
	if _, err = tx.ExecContext(ctx, "UPDATE ai_conversations_v2 SET active=FALSE,updated_at=? WHERE conversation_id=? AND user_id=?", now, old, user); err != nil {
		return next, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO ai_conversations_v2 VALUES(?,?,TRUE,?,?)", next.ID, user, now, now); err != nil {
		return next, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE ai_profiles_v2 SET conversation_id=?,active_request_id=NULL,last_request_id=NULL,updated_at=? WHERE user_id=?", next.ID, now, user); err != nil {
		return next, err
	}
	return next, tx.Commit()
}
