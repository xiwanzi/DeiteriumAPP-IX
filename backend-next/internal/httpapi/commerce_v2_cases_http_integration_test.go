//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type commerceCaseHTTPFixture struct {
	catalog                        catalogFixture
	handler                        http.Handler
	respondentToken, outsiderToken string
	hashes                         map[string]string
}

func newCommerceCaseHTTPFixture(t *testing.T) commerceCaseHTTPFixture {
	t.Helper()
	f := newCatalogFixture(t)
	ctx := context.Background()
	outsider := store.User{ID: "case_outsider", PlayerRef: "player_case_outsider", GameID: "CaseOutsider", ServerUUID: "10000000-0000-0000-0000-000000000004", QQ: "123456789", PasswordHash: "test-only", Status: "active"}
	_, err := f.s.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES(?,?,?,?,?,?,'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)`, outsider.ID, outsider.PlayerRef, outsider.ServerUUID, outsider.GameID, outsider.QQ, outsider.PasswordHash, strings.Repeat("0", 64))
	if err != nil {
		t.Fatal(err)
	}
	fixture := commerceCaseHTTPFixture{catalog: f, respondentToken: strings.Repeat("C", 43), outsiderToken: strings.Repeat("D", 43), hashes: map[string]string{}}
	for _, v := range []struct {
		user  store.User
		token string
	}{{f.other, fixture.respondentToken}, {outsider, fixture.outsiderToken}} {
		if err = f.s.CreateSession(ctx, v.user, store.Digest([]byte(v.token)), "app", "", time.Now().Add(time.Hour), ""); err != nil {
			t.Fatal(err)
		}
	}
	party := func(u store.User) map[string]any {
		return map[string]any{"kind": "PLAYER", "playerRef": u.PlayerRef, "storeId": nil, "displayName": u.GameID, "contactQq": u.QQ}
	}
	for _, seed := range []struct {
		suffix, state string
		assigned      any
	}{{"a", "IN_REVIEW", f.admin.ID}, {"b", "SUBMITTED", nil}, {"c", "RESOLVED", f.admin.ID}} {
		id := "case_order_" + seed.suffix
		body, _ := json.Marshal(map[string]any{"orderNo": "QA-" + seed.suffix, "construction": false, "buyer": party(f.buyer), "seller": party(f.other), "items": []any{map[string]any{"productId": "listing_" + seed.suffix, "productVersion": 1, "title": "冻结条款", "subtitle": "", "description": "原成交描述", "unitPrice": "1.00", "quantity": 1, "photoAssetIds": []any{}, "categoryName": "其他", "includedItems": []any{}, "contentBlocks": []any{}}}, "delivery": map[string]any{"method": "PICKUP", "location": "约定地点", "projectName": ""}, "confirmationHours": 72, "shippedAt": time.Now().UTC(), "workCompletedAt": nil, "confirmedAt": nil})
		_, err = f.s.DB.Exec(`INSERT INTO commerce_resources_v2(resource_id,resource_kind,channel,owner_id,owner_uuid,payee_id,payee_uuid,escrow_ref,amount,settled_amount,refunded_amount,state,funds_state,body,snapshot_id,snapshot_sha256,version,refund_attempts,automatic,created_at,updated_at) VALUES(?,'ORDER','PLAYER_MARKET',?,?,?,?,?,'1.00','0.00','0.00','SHIPPED','INTERVENTION_HOLD',?,?,?,1,1,false,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, id, f.buyer.ID, f.buyer.ServerUUID, f.other.ID, f.other.ServerUUID, "escrow:"+id, body, "snapshot_"+id, strings.Repeat("a", 64))
		if err != nil {
			t.Fatal(err)
		}
		view, err := f.s.CommerceViewV2(ctx, f.buyer.ID, id, false)
		if err != nil {
			t.Fatal(err)
		}
		snapshot := map[string]any{"snapshotId": "evidence_" + seed.suffix, "transactionKind": "ORDER", "transactionId": id, "transactionVersion": 1, "capturedAt": time.Now().UTC(), "order": view, "commission": nil, "eventLog": []any{}}
		raw, _ := json.Marshal(snapshot)
		hash := store.Digest(raw)
		snapshot["sha256"] = hash
		raw, _ = json.Marshal(snapshot)
		caseBody, _ := json.Marshal(map[string]any{"applicant": party(f.buyer), "respondent": party(f.other), "reasonCode": "OTHER", "description": "已提交双方约定与实际交付的差异。", "desiredResolution": "FULL_REFUND", "evidenceAssetIds": []any{}})
		caseID := "case_" + seed.suffix
		_, err = f.s.DB.Exec(`INSERT INTO commerce_interventions_v2(case_id,resource_id,applicant_id,respondent_id,state,version,body,snapshot,assigned_admin_id,funds_held,desired_refund_amount,resolution,evidence_entries,created_at,updated_at) VALUES(?,?,?,?,?,1,?,?,?,true,'1.00','','[]',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, caseID, id, f.buyer.ID, f.other.ID, seed.state, caseBody, raw, seed.assigned)
		if err != nil {
			t.Fatal(err)
		}
		fixture.hashes[caseID] = hash
	}
	mux := http.NewServeMux()
	f.server.registerCommerceV2(mux)
	fixture.handler = mux
	return fixture
}

func (f commerceCaseHTTPFixture) get(t *testing.T, path, token string) (int, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid response %d: %s", w.Code, w.Body.String())
	}
	return w.Code, result
}

func TestCommerceCaseHTTPReadPermissionsAndFrozenSnapshot(t *testing.T) {
	f := newCommerceCaseHTTPFixture(t)
	for _, test := range []struct {
		path, token string
		status      int
	}{
		{"/api/v1/interventions/case_a", f.catalog.buyerToken, 200},
		{"/api/v1/interventions/case_a", f.respondentToken, 200},
		{"/api/v1/admin/interventions/case_a", f.catalog.adminToken, 200},
		{"/api/v1/interventions/case_a", f.outsiderToken, 404},
		{"/api/v1/admin/interventions/case_a", f.respondentToken, 403},
		{"/api/v1/admin/interventions/case_a", f.outsiderToken, 403},
	} {
		status, result := f.get(t, test.path, test.token)
		if status != test.status {
			t.Fatalf("%s got %d: %v", test.path, status, result)
		}
		if status == 200 {
			view := result["data"].(map[string]any)
			if view["transactionId"] != "case_order_a" || view["assignedAdminRef"] != f.catalog.admin.PlayerRef || view["snapshot"].(map[string]any)["sha256"] != f.hashes["case_a"] {
				t.Fatal(view)
			}
		}
	}
	if _, err := f.catalog.s.DB.Exec(`UPDATE commerce_resources_v2 SET body=JSON_SET(body,'$.orderNo','CURRENT-CHANGED'),version=version+1 WHERE resource_id='case_order_a'`); err != nil {
		t.Fatal(err)
	}
	status, result := f.get(t, "/api/v1/interventions/case_a", f.catalog.buyerToken)
	if status != 200 {
		t.Fatal(result)
	}
	snapshot := result["data"].(map[string]any)["snapshot"].(map[string]any)
	if snapshot["sha256"] != f.hashes["case_a"] || snapshot["order"].(map[string]any)["orderNo"] != "QA-a" {
		t.Fatal("case read replaced frozen evidence", snapshot)
	}
}

func TestCommerceCaseHTTPAdminFiltersAndScopedPagination(t *testing.T) {
	f := newCommerceCaseHTTPFixture(t)
	for _, test := range []struct {
		query string
		want  []string
	}{
		{"status=SUBMITTED", []string{"case_b"}},
		{"assignedToMe=true", []string{"case_c", "case_a"}},
		{"assignedToMe=false", []string{"case_c", "case_b", "case_a"}},
		{"transactionId=case_order_a", []string{"case_a"}},
	} {
		status, result := f.get(t, "/api/v1/admin/interventions?"+test.query, f.catalog.adminToken)
		if status != 200 {
			t.Fatal(status, result)
		}
		rows := result["data"].([]any)
		if len(rows) != len(test.want) {
			t.Fatal(test.query, rows)
		}
		for i, id := range test.want {
			if rows[i].(map[string]any)["caseId"] != id {
				t.Fatal(rows)
			}
		}
	}
	status, result := f.get(t, "/api/v1/admin/interventions?limit=1", f.catalog.adminToken)
	if status != 200 {
		t.Fatal(result)
	}
	if result["data"].([]any)[0].(map[string]any)["caseId"] != "case_c" {
		t.Fatal(result)
	}
	cursor := result["page"].(map[string]any)["nextCursor"].(string)
	status, result = f.get(t, "/api/v1/admin/interventions?limit=1&cursor="+url.QueryEscape(cursor), f.catalog.adminToken)
	if status != 200 || result["data"].([]any)[0].(map[string]any)["caseId"] != "case_b" {
		t.Fatal(status, result)
	}
	status, _ = f.get(t, "/api/v1/admin/interventions?status=SUBMITTED&cursor="+url.QueryEscape(cursor), f.catalog.adminToken)
	if status != 400 {
		t.Fatal("cursor reused across filters", status)
	}
	status, _ = f.get(t, "/api/v1/admin/interventions", f.outsiderToken)
	if status != 403 {
		t.Fatal("outsider listed cases", status)
	}
}
