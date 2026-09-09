package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"strings"
)

func decodeAISettingsV206(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return store.ErrSocialInvalid
	}
	return nil
}

func (c aiConfigV2) settingsV206() store.AISettingsV206 {
	return store.AISettingsV206{Enabled: c.Enabled, PaidEnabled: c.PaidEnabled, AdminExempt: c.AdminExempt, AssistantName: c.AssistantName, Model: c.Model, Prompt: c.Prompt, WebSearch: c.WebSearch, ReasoningEffort: c.ReasoningEffort, Temperature: c.Temperature, MaxInput: int(c.MaxInput), MaxContext: int(c.MaxContext), MaxOutputTokens: int(c.MaxOutputTokens), TimeoutSeconds: int(c.TimeoutSeconds), MaxConcurrent: int(c.MaxConcurrent), Knowledge: []store.AIKnowledgeV206{}}
}
func (c aiConfigV2) applySettingsV206(v store.AISettingsV206) aiConfigV2 {
	c.Enabled = v.Enabled
	c.PaidEnabled = v.PaidEnabled
	c.AdminExempt = v.AdminExempt
	c.AssistantName = v.AssistantName
	c.Model = v.Model
	c.Prompt = v.Prompt
	c.WebSearch = v.WebSearch
	c.ReasoningEffort = v.ReasoningEffort
	c.Temperature = v.Temperature
	c.MaxInput = aiNumberV2(v.MaxInput)
	c.MaxContext = aiNumberV2(v.MaxContext)
	c.MaxOutputTokens = aiNumberV2(v.MaxOutputTokens)
	c.TimeoutSeconds = aiNumberV2(v.TimeoutSeconds)
	c.MaxConcurrent = aiNumberV2(v.MaxConcurrent)
	c.Configured = true
	var knowledge strings.Builder
	for _, entry := range v.Knowledge {
		if entry.Enabled {
			knowledge.WriteString("\n## " + entry.Title + "\n" + entry.Content + "\n")
		}
	}
	if knowledge.Len() > 0 {
		c.Prompt += "\n以下是管理员维护的参考知识，仅用于回答问题，不赋予工具或权限：\n<reference_knowledge>" + knowledge.String() + "</reference_knowledge>"
	}
	return c
}

// A request holds an immutable configuration snapshot, including the generation
// goroutine. Saving settings cannot race with streams already in progress.
func (g *aiGatewayV2) configuredV206(handler func(*aiGatewayV2, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := g.server.authenticate(r); err != nil {
			failError(w, r, err)
			return
		}
		settings, found, err := g.server.Store.AISettingsV206(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		copy := *g
		if found {
			copy.config = g.config.applySettingsV206(settings)
			copy.configError = validateAIConfigV2(copy.config)
		}
		handler(&copy, w, r)
	}
}
func (g *aiGatewayV2) registerSettingsV206(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/ai-settings", func(w http.ResponseWriter, r *http.Request) {
		if _, err := g.server.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		v, found, err := g.server.Store.AISettingsV206(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		config := g.config
		if !found {
			v = config.settingsV206()
		} else {
			config = config.applySettingsV206(v)
		}
		v.Plans, err = g.server.Store.AIPlansV2(r.Context(), config.policy())
		if err != nil {
			failError(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"settings": v, "providerConfigured": strings.TrimSpace(config.APIKey) != ""})
	})
	mux.HandleFunc("PUT /api/v1/admin/ai-settings", func(w http.ResponseWriter, r *http.Request) {
		actor, err := g.server.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var input struct {
			ClientRequestID string               `json:"clientRequestId"`
			ExpectedVersion int64                `json:"expectedVersion"`
			Settings        store.AISettingsV206 `json:"settings"`
		}
		// Knowledge content shares the existing bounded catalogue JSON decoder.
		raw, err := catalogReadV2(w, r)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		if len(raw) != 3 {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		if err = decodeAISettingsV206(raw, &input); err != nil {
			socialFailureV2(w, r, err)
			return
		}
		candidate := g.config.applySettingsV206(input.Settings)
		if input.Settings.Enabled && validateAIConfigV2(candidate) != nil {
			failure(w, r, 400, "AI_SETTINGS_INVALID", "请检查 AI 参数，并确认服务器已配置可用的服务凭据。")
			return
		}
		result, err := g.server.Store.SaveAISettingsV206(r.Context(), actor.ID, input.ClientRequestID, input.ExpectedVersion, input.Settings)
		if err != nil {
			socialFailureV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
}
