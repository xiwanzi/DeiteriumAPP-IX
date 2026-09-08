package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func aiTestEventV2(w http.ResponseWriter, kind string, body any) {
	encoded, _ := json.Marshal(body)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, encoded)
	w.(http.Flusher).Flush()
}
func aiTestCompleteV2(w http.ResponseWriter, text, status string, annotations []map[string]string, search bool) {
	output := []any{}
	if search {
		output = append(output, map[string]any{"type": "web_search_call", "status": "completed"})
	}
	output = append(output, map[string]any{"type": "reasoning", "content": []any{map[string]string{"type": "reasoning_text", "text": "SECRET_REASONING_MUST_NOT_LEAK"}}}, map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": text, "annotations": annotations}}})
	aiTestEventV2(w, "response."+status, map[string]any{"type": "response." + status, "response": map[string]any{"id": "provider-response", "status": status, "output": output, "usage": map[string]int{"input_tokens": 12, "output_tokens": 17}, "incomplete_details": map[string]string{"reason": "max_output_tokens"}}})
}
func TestAIProviderSeparatesReasoningAndRealSourcesV2(t *testing.T) {
	var calls int
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/responses" || r.Method != "POST" {
			t.Error("wrong provider route")
		}
		if r.Header.Get("Authorization") != "Bearer fake-provider-key" || r.Header.Get("Cookie") != "" {
			t.Error("wrong provider authorization boundary")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("bad provider payload")
		}
		if body["tool_choice"] != "auto" || body["model"] != "deepseek-v4-flash" {
			t.Error("wrong provider tool or model")
		}
		if strings.Contains(body["user"].(string), "private_user") {
			t.Error("raw user identifier sent upstream")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		aiTestEventV2(w, "response.created", map[string]any{"type": "response.created", "response": map[string]string{"id": "response_1"}})
		aiTestEventV2(w, "response.reasoning_text.delta", map[string]string{"type": "response.reasoning_text.delta", "delta": "SECRET_REASONING_MUST_NOT_LEAK"})
		aiTestEventV2(w, "response.web_search_call.completed", map[string]string{"type": "response.web_search_call.completed"})
		aiTestEventV2(w, "response.output_text.delta", map[string]string{"type": "response.output_text.delta", "delta": "真实正文"})
		aiTestCompleteV2(w, "真实正文", "completed", []map[string]string{{"type": "url_citation", "url": "https://example.test/source", "title": "实际来源"}}, true)
	}))
	defer provider.Close()
	c := aiDefaultsV2()
	c.Enabled = true
	c.APIKey = "fake-provider-key"
	c.BaseURL = provider.URL
	c.AllowHTTPForTest = true
	c.Prompt = "test only"
	e := store.AIExchangeV2{ID: "request_1", UserID: "private_user", Input: "查询资料", Sources: []store.AISourceV2{}}
	result := runAIProviderV2(context.Background(), provider.Client(), c, &e, nil, func() error {
		if strings.Contains(e.Answer, "SECRET_REASONING") {
			t.Error("reasoning persisted")
		}
		return nil
	})
	if calls != 1 || result.Status != "completed" || e.Answer != "真实正文" || !e.SearchUsed || len(e.Sources) != 1 || e.Sources[0].Origin != "annotation" {
		t.Fatalf("incorrect result: calls=%d status=%s text=%q sources=%v", calls, result.Status, e.Answer, e.Sources)
	}
	if result.InputTokens != 12 || result.OutputTokens != 17 {
		t.Fatal("provider usage was not retained")
	}
}
func TestAIProviderTextLinksAndNormalAnswerWithoutSearchV2(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		aiTestCompleteV2(w, "你好。参考 https://example.test/guide/。", "completed", nil, false)
	}))
	defer provider.Close()
	c := aiDefaultsV2()
	c.APIKey = "fake"
	c.BaseURL = provider.URL
	c.Prompt = "test"
	e := store.AIExchangeV2{UserID: "user", Input: "你好", Sources: []store.AISourceV2{}}
	result := runAIProviderV2(context.Background(), provider.Client(), c, &e, nil, func() error { return nil })
	if result.Status != "completed" || e.SearchUsed || len(e.Sources) != 1 || e.Sources[0].Origin != "provider_text" || e.Sources[0].URL != "https://example.test/guide/" {
		t.Fatalf("normal answer or source classification wrong: %+v %+v", result, e.Sources)
	}
}
func TestAIProviderNeverTreatsEOFOrIncompleteAsCompletionV2(t *testing.T) {
	for _, mode := range []string{"eof", "incomplete", "failed", "empty"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if mode == "failed" {
					http.Error(w, "upstream private key must stay private", 429)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				if mode != "empty" {
					aiTestEventV2(w, "response.output_text.delta", map[string]string{"type": "response.output_text.delta", "delta": "部分内容"})
				}
				if mode == "incomplete" {
					aiTestCompleteV2(w, "部分内容", "incomplete", nil, true)
				}
				if mode == "empty" {
					aiTestCompleteV2(w, "", "completed", nil, false)
				}
			}))
			defer provider.Close()
			c := aiDefaultsV2()
			c.APIKey = "fake"
			c.BaseURL = provider.URL
			c.Prompt = "test"
			e := store.AIExchangeV2{UserID: "user", Input: "question", Sources: []store.AISourceV2{}}
			result := runAIProviderV2(context.Background(), provider.Client(), c, &e, nil, func() error { return nil })
			if calls != 1 || result.Status == "completed" || strings.Contains(e.Answer, "private key") {
				t.Fatalf("unsafe outcome: calls=%d status=%s", calls, result.Status)
			}
			if mode == "incomplete" && result.Status != "incomplete" {
				t.Fatal("incomplete state lost")
			}
			if mode == "failed" && result.Status != "failed" {
				t.Fatal("known rejection not classified")
			}
		})
	}
}
func TestAIConfigRejectsUnsafeProviderAndInvalidBudgetV2(t *testing.T) {
	c := aiDefaultsV2()
	c.APIKey = "secret"
	c.Prompt = "private prompt"
	c.Enabled = true
	if err := validateAIConfigV2(c); err != nil {
		t.Fatal(err)
	}
	bad := c
	bad.BaseURL = "http://example.test"
	if validateAIConfigV2(bad) == nil {
		t.Fatal("cleartext provider accepted")
	}
	bad = c
	bad.FreeQuota = 0
	if validateAIConfigV2(bad) == nil {
		t.Fatal("zero quota accepted")
	}
	bad = c
	bad.APIKey = "secret\nHeader: value"
	if validateAIConfigV2(bad) == nil {
		t.Fatal("header injection accepted")
	}
	bad = c
	bad.PaidEnabled = true
	if validateAIConfigV2(bad) == nil {
		t.Fatal("paid success enabled without finance")
	}
	var n aiNumberV2
	if json.Unmarshal([]byte(`"24"`), &n) != nil || n != 24 {
		t.Fatal("existing string-number config not supported")
	}
}
func TestAIBodyAcceptsEscapedEmojiButRejectsUnknownOrTrailingV2(t *testing.T) {
	valid := `{"clientMessageId":"emoji","content":"` + strings.Repeat(`\uD83D\uDE00`, 2000) + `"}`
	for _, test := range []struct {
		body  string
		valid bool
	}{{valid, true}, {`{"clientMessageId":"id","content":"hello","admin":true}`, false}, {`{"clientMessageId":"id","content":"hello"} {}`, false}, {valid + strings.Repeat(" ", 32<<10), false}} {
		request := httptest.NewRequest("POST", "/ai/chat/stream", strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/json")
		var input struct {
			ClientMessageID string `json:"clientMessageId"`
			Content         string `json:"content"`
		}
		err := aiBodyV2(httptest.NewRecorder(), request, &input)
		if (err == nil) != test.valid {
			t.Fatalf("valid=%v error=%v", test.valid, err)
		}
		if test.valid && utf8.RuneCountInString(input.Content) != 2000 {
			t.Fatal("emoji character boundary corrupted")
		}
	}
}
