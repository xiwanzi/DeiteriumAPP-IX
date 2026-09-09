package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

type AIKnowledgeV206 struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
}
type AISettingsV206 struct {
	Version         int64             `json:"version"`
	Enabled         bool              `json:"enabled"`
	PaidEnabled     bool              `json:"paidPlansEnabled"`
	AdminExempt     bool              `json:"adminQuotaExempt"`
	AssistantName   string            `json:"assistantName"`
	Model           string            `json:"model"`
	Prompt          string            `json:"systemPrompt"`
	WebSearch       bool              `json:"webSearch"`
	ReasoningEffort string            `json:"reasoningEffort"`
	Temperature     float64           `json:"temperature"`
	MaxInput        int               `json:"maxInputChars"`
	MaxContext      int               `json:"maxContextMessages"`
	MaxOutputTokens int               `json:"maxOutputTokens"`
	TimeoutSeconds  int               `json:"timeoutSeconds"`
	MaxConcurrent   int               `json:"maxConcurrent"`
	Knowledge       []AIKnowledgeV206 `json:"knowledge"`
	Plans           []AIPlanV2        `json:"plans,omitempty"`
}

func (s *Store) AISettingsV206(ctx context.Context) (AISettingsV206, bool, error) {
	var v AISettingsV206
	var raw string
	err := s.DB.QueryRowContext(ctx, "SELECT settings_json,version FROM ai_settings_v206 WHERE id=1").Scan(&raw, &v.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return v, false, nil
	}
	if err != nil {
		return v, false, err
	}
	version := v.Version
	err = json.Unmarshal([]byte(raw), &v)
	v.Version = version
	return v, true, err
}

func (v AISettingsV206) Validate() error {
	if !ValidSocialText(v.AssistantName, 1, 80) || !ValidSocialText(v.Model, 1, 120) || strings.ContainsAny(v.Model, "\r\n") || !ValidSocialText(v.Prompt, 0, 32000) || v.Temperature < 0 || v.Temperature > 2 || v.MaxInput < 1 || v.MaxInput > 2000 || v.MaxContext < 2 || v.MaxContext > 40 || v.MaxOutputTokens < 256 || v.MaxOutputTokens > 16384 || v.TimeoutSeconds < 5 || v.TimeoutSeconds > 300 || v.MaxConcurrent < 1 || v.MaxConcurrent > 16 || !catalogEnum(v.ReasoningEffort, "none", "low", "high") || len(v.Knowledge) > 50 || len(v.Plans) < 1 || len(v.Plans) > 20 {
		return ErrSocialInvalid
	}
	seen := map[string]bool{}
	size := 0
	for _, k := range v.Knowledge {
		if !ValidSocialID(k.ID) || seen[k.ID] || !ValidSocialText(k.Title, 1, 120) || !ValidSocialText(k.Content, 1, 12000) {
			return ErrSocialInvalid
		}
		seen[k.ID] = true
		size += len(k.Content)
	}
	if size > 120000 {
		return ErrSocialInvalid
	}
	free := 0
	seen = map[string]bool{}
	codes := map[string]bool{}
	for _, p := range v.Plans {
		if !CatalogReferenceV2(p.PlanID) || !CatalogReferenceV2(p.Code) || len(p.Code) > 32 || seen[p.PlanID] || codes[p.Code] || !ValidSocialText(p.Name, 1, 80) || !ValidSocialText(p.Description, 0, 1000) || p.QuotaPerWindow < 1 || p.QuotaPerWindow > 100000 || p.WindowHours < 1 || p.WindowHours > 168 || p.DurationDays < 0 || p.DurationDays > 3650 {
			return ErrSocialInvalid
		}
		seen[p.PlanID] = true
		codes[p.Code] = true
		if p.Code == "free" {
			free++
			if p.PlanID != "plan_free" || p.Price != "0.00" || p.DurationDays != 0 || !p.Active {
				return ErrSocialInvalid
			}
		} else if p.DurationDays < 1 || commerceWithinExecutionLimitV2(p.Price) != nil {
			return ErrSocialInvalid
		}
	}
	if free != 1 {
		return ErrSocialInvalid
	}
	return nil
}

func (s *Store) SaveAISettingsV206(ctx context.Context, actor, key string, expected int64, v AISettingsV206) (json.RawMessage, error) {
	if expected < 0 || v.Validate() != nil {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "ai.settings", key, struct {
		Expected int64
		Settings AISettingsV206
	}{expected, v}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO ai_settings_v206 VALUES(1,'{}',0,UTC_TIMESTAMP(6))"); err != nil {
			return nil, err
		}
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT version FROM ai_settings_v206 WHERE id=1 FOR UPDATE").Scan(&version); err != nil {
			return nil, err
		}
		if version != expected {
			return nil, ErrSocialVersion
		}
		rows, err := tx.QueryContext(ctx, "SELECT plan_id,code FROM ai_plans_v2 FOR UPDATE")
		if err != nil {
			return nil, err
		}
		existing := map[string]string{}
		for rows.Next() {
			var id, code string
			if err = rows.Scan(&id, &code); err != nil {
				rows.Close()
				return nil, err
			}
			existing[id] = code
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		for i, p := range v.Plans {
			if old, ok := existing[p.PlanID]; ok && old != p.Code {
				return nil, ErrSocialInvalid
			}
			delete(existing, p.PlanID)
			_, err = tx.ExecContext(ctx, `INSERT INTO ai_plans_v2(plan_id,code,name,description,price,quota_limit,window_hours,duration_days,model_tier,active,sort_order,version) VALUES(?,?,?,?,?,?,?,?,?,?,?,1) ON DUPLICATE KEY UPDATE name=VALUES(name),description=VALUES(description),price=VALUES(price),quota_limit=VALUES(quota_limit),window_hours=VALUES(window_hours),duration_days=VALUES(duration_days),model_tier=VALUES(model_tier),active=VALUES(active),sort_order=VALUES(sort_order),version=version+1`, p.PlanID, p.Code, p.Name, p.Description, p.Price, p.QuotaPerWindow, p.WindowHours, p.DurationDays, "flash", p.Active, i)
			if err != nil {
				return nil, err
			}
		}
		// Keep sold/disabled plan identifiers so orders and renewals remain meaningful.
		if len(existing) > 0 {
			return nil, ErrSocialInvalid
		}
		v.Version = version + 1
		v.Plans = nil
		if _, err = tx.ExecContext(ctx, "UPDATE ai_settings_v206 SET settings_json=?,version=?,updated_at=UTC_TIMESTAMP(6) WHERE id=1", catalogJSON(v), v.Version); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'ai.settings','saki',UTC_TIMESTAMP(6))", actor); err != nil {
			return nil, err
		}
		return map[string]any{"version": v.Version}, nil
	})
}
