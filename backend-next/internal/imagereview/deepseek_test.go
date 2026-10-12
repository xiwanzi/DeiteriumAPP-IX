package imagereview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewScoreThresholdAndCombinedRisk(t *testing.T) {
	for _, tc := range []struct {
		s     Scores
		total int
		flag  bool
	}{{Scores{50, 0, 0}, 50, false}, {Scores{0, 51, 0}, 51, true}, {Scores{0, 0, 95}, 95, true}, {Scores{40, 40, 20}, 55, true}, {Scores{100, 100, 100}, 100, true}, {Scores{}, 0, false}} {
		got := tc.s.Total()
		if got != tc.total || (got > Threshold) != tc.flag {
			t.Fatalf("%+v => %d", tc.s, got)
		}
	}
}
func TestReviewModelOutputRejectsIncompleteOrUnboundedResults(t *testing.T) {
	good := `{"scores":{"r18":0,"politics":0,"promotion":61},"summary":"存在联系导流","reasons":["可见QQ群联系方式"]}`
	v, e := parseResult(good)
	if e != nil || v.Scores.Total() != 61 {
		t.Fatal(v, e)
	}
	for _, raw := range []string{
		`{}`, `{"error":"无法读取"}`, strings.Replace(good, `"politics":0,`, "", 1), strings.Replace(good, `:61`, `:101`, 1), strings.Replace(good, `:61`, `:60.5`, 1), strings.Replace(good, `:61`, `:null`, 1), good + ` {}`, "```json\n" + good + "\n```", strings.Replace(good, `"summary":`, `"unexpected":true,"summary":`, 1), strings.Replace(good, `"存在联系导流"`, `""`, 1),
	} {
		if _, e := parseResult(raw); e == nil {
			t.Fatalf("accepted invalid model result: %s", raw)
		}
	}
}
func TestReviewDeepSeekRequestAndRedactedFailures(t *testing.T) {
	status := 200
	finish := "stop"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-private-key" {
			t.Error("missing auth")
		}
		var in map[string]any
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			t.Error("bad JSON")
		}
		if in["model"] != Model || in["stream"] != false {
			t.Error("incorrect model or stream")
		}
		encoded, _ := json.Marshal(in)
		if !strings.Contains(string(encoded), "https://objects.example/image?signature=private") || !strings.Contains(string(encoded), "json_object") {
			t.Error("image URL/JSON mode not provided")
		}
		w.WriteHeader(status)
		if status != 200 {
			fmt.Fprint(w, `{"error":"secret-token-and-private-url"}`)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": finish, "message": map[string]string{"content": `{"scores":{"r18":0,"politics":0,"promotion":0},"summary":"普通截图","reasons":["未发现相关内容"]}`}}}})
	}))
	defer server.Close()
	client, e := NewDeepSeek("test-private-key")
	if e != nil {
		t.Fatal(e)
	}
	client.endpoint = server.URL
	if _, e = client.Review(context.Background(), "https://objects.example/image?signature=private"); e != nil {
		t.Fatal(e)
	}
	for _, code := range []int{401, 402, 429, 500} {
		status = code
		_, e = client.Review(context.Background(), "https://objects.example/image?signature=private")
		var re *ReviewError
		if !errors.As(e, &re) || strings.Contains(e.Error(), "secret") || strings.Contains(e.Error(), "signature") {
			t.Fatalf("unredacted failure %v", e)
		}
	}
	status = 200
	finish = "length"
	if _, e = client.Review(context.Background(), "https://objects.example/image?signature=private"); e == nil {
		t.Fatal("truncated output accepted")
	}
}
