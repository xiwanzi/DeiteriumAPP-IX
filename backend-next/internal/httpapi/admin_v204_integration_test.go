//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestAnnouncementPermanentDeleteRemovesContentAndReplayCopiesV204(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "announcements.manage"); err != nil {
		t.Fatal(err)
	}
	create := map[string]any{"clientRequestId": "permanent-create", "title": "Sensitive announcement title", "summary": "Permanent removal test", "contentBlocks": []map[string]any{{"blockId": "body", "type": "PARAGRAPH", "text": "Permanently remove this unique body"}}, "coverAssetId": nil, "pinned": false, "priority": "NORMAL"}
	ann := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/admin/announcements", create, 200))
	id := ann["announcementId"].(string)
	f.request(t, "Alice", "POST", "/api/v1/admin/announcements/"+id+"/publish", map[string]any{"clientRequestId": "permanent-publish", "expectedVersion": 1}, 200)
	path := "/api/v1/admin/announcements/" + id + "/delete"
	input := map[string]any{"clientRequestId": "permanent-delete", "expectedVersion": 2}
	f.request(t, "Bob", "POST", path, input, 403)
	f.request(t, "Alice", "POST", path, map[string]any{"clientRequestId": "stale-delete", "expectedVersion": 1}, 409)
	f.request(t, "Alice", "POST", path, input, 200)
	f.request(t, "Alice", "POST", path, input, 200)
	f.request(t, "Alice", "GET", "/api/v1/admin/announcements/"+id, nil, 404)
	f.request(t, "Bob", "GET", "/api/v1/announcements/"+id, nil, 404)
	replay := f.request(t, "Alice", "POST", "/api/v1/admin/announcements", create, 200)
	raw, _ := json.Marshal(replay)
	if strings.Contains(string(raw), "unique body") || socialDataV2(replay)["deleted"] != true {
		t.Fatal("deleted body returned by replay")
	}
	for _, query := range []string{`SELECT COUNT(*) FROM social_announcements_v2`, `SELECT COUNT(*) FROM social_notifications_v2 WHERE topic='ANNOUNCEMENTS'`, `SELECT COUNT(*) FROM social_requests_v2 WHERE response_json LIKE '%unique body%'`} {
		var n int
		if err := f.store.DB.QueryRowContext(ctx, query).Scan(&n); err != nil || n != 0 {
			t.Fatalf("permanent cleanup failed: count=%d err=%v", n, err)
		}
	}
	var audits int
	f.store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events_next WHERE action='announcement.delete'`).Scan(&audits)
	if audits != 1 {
		t.Fatal("delete audit duplicated")
	}
}

func TestWalletReadsCommittedCoreLedgerAndEnforcesPlayerScopeV204(t *testing.T) {
	f := newSocialFixtureV2(t)
	f.app.Close()
	ctx := context.Background()
	app := New(f.store, config.Config{Development: true, PublicOrigin: "http://127.0.0.1", Nodes: []config.Node{{ID: "amiya", Economy: true}}})
	defer app.Close()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "audit.read"); err != nil {
		t.Fatal(err)
	}
	var queries []map[string]string
	app.Core.Connect("amiya", func(ctx context.Context, kind string, payload any) error {
		frame := payload.(map[string]any)
		if frame["command"] != "wallet.records" {
			t.Fatalf("read path dispatched unexpected command %v", frame["command"])
		}
		var query map[string]string
		if err := json.Unmarshal(frame["payload"].(json.RawMessage), &query); err != nil {
			return err
		}
		queries = append(queries, query)
		uuid := query["playerUuid"]
		if uuid == "" {
			uuid = f.users["Bob"].ServerUUID
		}
		at, _ := time.Parse(time.RFC3339Nano, query["to"])
		at = at.Add(-time.Minute)
		row := ledgerRowV204{Sequence: "30", PlayerUUID: uuid, GameID: "CommittedPlayer", Source: "GAME", BusinessType: "TRANSFER", Direction: "income", Amount: "9.50", BeforeBalance: "10.00", AfterBalance: "19.50", OccurredAt: at, OperationID: "native_pay", BusinessRef: "native_pay"}
		rows := []ledgerRowV204{row}
		if query["recordId"] != "" && query["recordId"] != "30" {
			rows = []ledgerRowV204{}
		}
		if query["beforeSequence"] == "0" && query["recordId"] == "" {
			older := row
			older.Sequence = "29"
			older.OccurredAt = at.Add(-time.Minute)
			rows = append(rows, older)
		} else if query["beforeSequence"] != "0" {
			row.Sequence = "28"
			row.OccurredAt = at.Add(-2 * time.Minute)
			rows = []ledgerRowV204{row}
		}
		response, _ := json.Marshal(map[string]any{"operationId": frame["operationId"], "status": "COMPLETED", "data": ledgerPageV204{Records: rows, Snapshot: "30"}})
		return f.store.CoreReply(ctx, "amiya", frame["operationId"].(string), "COMPLETED", response)
	})
	defer app.Core.Disconnect("amiya")
	request := func(user, path string, want int) map[string]any {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer "+f.tokens[user])
		w := httptest.NewRecorder()
		app.Handler().ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		var result map[string]any
		json.Unmarshal(w.Body.Bytes(), &result)
		return result
	}
	first := request("Alice", "/api/v1/wallet/records?limit=1", 200)
	row := socialDataV2(first)["records"].([]any)[0].(map[string]any)
	if row["title"] != "游戏内收入" || row["recordId"] != "econ_30" || queries[0]["playerUuid"] != f.users["Alice"].ServerUUID {
		t.Fatal("committed ledger was not correctly scoped and labelled")
	}
	cursor := first["page"].(map[string]any)["nextCursor"].(string)
	request("Bob", "/api/v1/wallet/records?cursor="+url.QueryEscape(cursor), 400)
	request("Alice", "/api/v1/wallet/records?cursor="+url.QueryEscape(cursor), 200)
	if queries[1]["snapshot"] != "30" || queries[1]["beforeSequence"] != "30" {
		t.Fatal("ledger page lost its stable snapshot")
	}
	request("Bob", "/api/v1/wallet/records?playerUuid="+f.users["Alice"].ServerUUID, 400)
	request("Bob", "/api/v1/admin/transactions", 403)
	request("Alice", "/api/v1/admin/transactions?limit=1&playerRef="+f.users["Bob"].PlayerRef, 200)
	if queries[len(queries)-1]["playerUuid"] != f.users["Bob"].ServerUUID {
		t.Fatal("admin player filter not resolved")
	}
	detail := socialDataV2(request("Alice", "/api/v1/wallet/records/econ_30", 200))
	if detail["status"] != "SUCCESS" {
		t.Fatal("ledger detail status mismatch")
	}
	request("Alice", "/api/v1/wallet/records/econ_31", 404)
	if dir := os.Getenv("DEUTERIUM_CONTRACT_SAMPLES"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]any{"WalletRecordsResponse": first, "V2WalletRecordDetailResponse": request("Alice", "/api/v1/wallet/records/econ_30", 200)})
		if err := os.WriteFile(filepath.Join(dir, "v204-wallet.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAdminAuditIncludesHiddenHistoryAndChecksEveryRequestV204(t *testing.T) {
	f := newCatalogFixture(t)
	f.server.CommerceCore = newCommerceTestCore()
	ctx := context.Background()
	first, listing := commerceOrderTest(t, f, "audit-first")
	second, _ := commerceOrderTest(t, f, "audit-second")
	mutation, err := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, first.ID, "ORDER", "audit-refund", first.Version, commerceRefundInput("audit-refund", first.Version), true)
	if err != nil {
		t.Fatal(err)
	}
	first = commerceRunTest(t, f, mutation)
	if err = f.s.HideRecordV203(ctx, f.buyer.ID, "ORDER", first.ID, "hide-audit-order", first.Version); err != nil {
		t.Fatal(err)
	}
	// Simulate an existing personal preference independently of source record state.
	if _, err = f.s.DB.Exec(`INSERT INTO personal_record_visibility_v203 VALUES(?,'LISTING',?,UTC_TIMESTAMP(6))`, f.admin.ID, listing.ID); err != nil {
		t.Fatal(err)
	}
	request := func(token, path string, want int) map[string]any {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		f.server.Handler().ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s returned %d: %s", path, w.Code, w.Body.String())
		}
		var result map[string]any
		json.Unmarshal(w.Body.Bytes(), &result)
		return result
	}
	for _, path := range []string{"/api/v1/admin/orders", "/api/v1/admin/products", "/api/v1/admin/players", "/api/v1/admin/orders/" + first.ID, "/api/v1/admin/products/" + listing.ID, "/api/v1/admin/transactions"} {
		request(f.buyerToken, path, 403)
	}
	firstPage := request(f.adminToken, "/api/v1/admin/orders?limit=1&playerRef="+f.buyer.PlayerRef, 200)
	if firstPage["data"].([]any)[0].(map[string]any)["orderId"] != second.ID {
		t.Fatal("orders are not newest first")
	}
	cursor := firstPage["page"].(map[string]any)["nextCursor"].(string)
	next := request(f.adminToken, "/api/v1/admin/orders?cursor="+cursor, 200)
	if next["data"].([]any)[0].(map[string]any)["orderId"] != first.ID {
		t.Fatal("hidden history missing from audit")
	}
	request(f.adminToken, "/api/v1/admin/products?cursor="+cursor, 400)
	view := socialDataV2(request(f.adminToken, "/api/v1/admin/orders/"+first.ID, 200))
	if view["readOnly"] != true || len(view["availableActions"].([]any)) != 0 {
		t.Fatal("audit exposed write controls")
	}
	products := request(f.adminToken, "/api/v1/admin/products?playerRef="+f.admin.PlayerRef, 200)["data"].([]any)
	if len(products) != 2 {
		t.Fatal("hidden or sold listings absent")
	}
	request(f.adminToken, "/api/v1/admin/products/"+listing.ID, 200)
	if len(request(f.adminToken, "/api/v1/admin/orders?playerRef="+f.other.PlayerRef, 200)["data"].([]any)) != 0 {
		t.Fatal("unrelated player orders leaked through filter")
	}
	if err = f.s.Grant(ctx, f.buyer.ID, "audit.read"); err != nil {
		t.Fatal(err)
	}
	request(f.buyerToken, "/api/v1/admin/orders", 200)
	request(f.buyerToken, "/api/v1/admin/orders?cursor="+cursor, 400)
	if _, err = f.s.DB.Exec(`DELETE FROM identity_permissions WHERE user_id=? AND permission='audit.read'`, f.buyer.ID); err != nil {
		t.Fatal(err)
	}
	request(f.buyerToken, "/api/v1/admin/orders", 403)
	if dir := os.Getenv("DEUTERIUM_CONTRACT_SAMPLES"); dir != "" {
		os.MkdirAll(dir, 0700)
		samples := map[string]any{"OrderListResponse": firstPage, "OrderDetailResponse": request(f.adminToken, "/api/v1/admin/orders/"+first.ID, 200), "PlayerListResponse": request(f.adminToken, "/api/v1/admin/players", 200), "ProductListResponse": request(f.adminToken, "/api/v1/admin/products", 200), "ProductDetailResponse": request(f.adminToken, "/api/v1/admin/products/"+listing.ID, 200)}
		raw, _ := json.Marshal(samples)
		if err := os.WriteFile(filepath.Join(dir, "v204-admin.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInterventionChangesQueueEmailOnceAndRollbackTogetherV204(t *testing.T) {
	f := newCatalogFixture(t)
	f.server.CommerceCore = newCommerceTestCore()
	ctx := context.Background()
	d, _ := commerceOrderTest(t, f, "email-case")
	m, err := f.s.CommerceFulfillmentV2(ctx, f.admin.ID, d.ID, "ORDER", "mail-ship", "ship", d.Version, store.CatalogObjectV2{"clientRequestId": "mail-ship", "expectedVersion": d.Version})
	if err != nil {
		t.Fatal(err)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	m, err = f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "mail-refund", d.Version, commerceRefundInput("mail-refund", d.Version), true)
	if err != nil {
		t.Fatal(err)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	m, err = f.s.ResolveCommerceRefundV2(ctx, f.admin.ID, d.ID, "ORDER", d.RefundID, "mail-reject", 1, "REJECT", "请平台核对本次交付", false)
	if err != nil {
		t.Fatal(err)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	input := store.CatalogObjectV2{"clientRequestId": "mail-case", "expectedVersion": d.Version, "reasonCode": "REFUND_DISAGREEMENT", "description": "申请核对已经交付物品与约定的差异。", "desiredResolution": "FULL_REFUND", "evidenceAssetIds": []any{}}
	m, err = f.s.CreateCommerceInterventionV2(ctx, f.buyer.ID, d.ID, "ORDER", "mail-case", d.Version, input)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.CreateCommerceInterventionV2(ctx, f.buyer.ID, d.ID, "ORDER", "mail-case", d.Version, input)
	if err != nil {
		t.Fatal(err)
	}
	if catalogTestCount(t, f.s, "admin_email_outbox_v204") != 1 {
		t.Fatal("new case notification duplicated")
	}
	_, err = f.s.CommerceCaseActionV2(ctx, f.admin.ID, m.ResourceID, "bad-assign", "assign", 999, store.CatalogObjectV2{"clientRequestId": "bad-assign", "expectedVersion": 999}, true)
	if err == nil {
		t.Fatal("stale case accepted")
	}
	if catalogTestCount(t, f.s, "admin_email_outbox_v204") != 1 {
		t.Fatal("rolled back action queued mail")
	}
	_, err = f.s.CommerceCaseActionV2(ctx, f.admin.ID, m.ResourceID, "good-assign", "assign", 1, store.CatalogObjectV2{"clientRequestId": "good-assign", "expectedVersion": 1}, true)
	if err != nil {
		t.Fatal(err)
	}
	if catalogTestCount(t, f.s, "admin_email_outbox_v204") != 2 {
		t.Fatal("new state did not queue mail")
	}
	event, err := f.s.ClaimEmailV204(ctx)
	if err != nil || event.CaseID == nil {
		t.Fatal("pending case message absent")
	}
	next, err := f.s.ClaimEmailV204(ctx)
	if err != nil || next.EventID == event.EventID {
		t.Fatal("lease allowed duplicate claim")
	}
}

func TestAdminSMTPEncryptionVersionReplayAndDurableRetryV204(t *testing.T) {
	f := newSocialFixtureV2(t)
	f.app.Close() // Exercise delivery explicitly; never send real test emails.
	ctx := context.Background()
	f.app.Config.SMTPKey = bytes.Repeat([]byte{9}, 32)
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Grant(ctx, f.users["Bob"].ID, "audit.read"); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/admin/email-settings"
	f.request(t, "Bob", "GET", path, nil, 403)
	input := map[string]any{"clientRequestId": "smtp-save", "expectedVersion": 0, "enabled": true, "host": "smtp.example.com", "port": 587, "security": "STARTTLS", "username": "sender@example.com", "password": "private-smtp-test-password", "from": "sender@example.com", "recipients": []string{"recipient@example.com"}}
	first := f.request(t, "Alice", "PUT", path, input, 200)
	again := f.request(t, "Alice", "PUT", path, input, 200)
	if socialDataV2(first)["version"] != socialDataV2(again)["version"] {
		t.Fatal("retry changed version")
	}
	result := f.request(t, "Alice", "GET", path, nil, 200)
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "private-smtp-test-password") || strings.Contains(string(raw), "passwordCipher") {
		t.Fatal("secret exposed")
	}
	saved, err := f.store.EmailSettingsV204(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saved.PasswordCipher == "" || strings.Contains(saved.PasswordCipher, "private-smtp-test-password") {
		t.Fatal("secret not encrypted")
	}
	input["clientRequestId"] = "smtp-stale"
	f.request(t, "Alice", "PUT", path, input, 409)
	input["clientRequestId"] = "smtp-preserve"
	input["expectedVersion"] = 1
	delete(input, "password")
	f.request(t, "Alice", "PUT", path, input, 200)
	preserved, _ := f.store.EmailSettingsV204(ctx)
	if preserved.PasswordCipher != saved.PasswordCipher {
		t.Fatal("omitted password did not preserve stored secret")
	}
	test := map[string]any{"clientRequestId": "smtp-test"}
	f.request(t, "Alice", "POST", path+"/test", test, 200)
	f.request(t, "Alice", "POST", path+"/test", test, 200)
	sends := 0
	sender := func(_ context.Context, s notify.Settings, password, id, subject, body string) error {
		sends++
		if password != "private-smtp-test-password" || len(s.Recipients) != 1 {
			t.Fatal("wrong delivery settings")
		}
		if sends == 1 {
			return errors.New("synthetic retry")
		}
		return nil
	}
	if f.app.processEmailV204(ctx, sender) == nil {
		t.Fatal("expected deferred delivery")
	}
	status, _ := f.store.EmailStatusV204(ctx)
	if status["retrying"] != 1 {
		t.Fatal("failure was lost")
	}
	if f.app.processEmailV204(ctx, sender) == nil || sends != 1 {
		t.Fatal("retry ignored backoff")
	}
	if _, err = f.store.DB.ExecContext(ctx, `UPDATE admin_email_outbox_v204 SET next_attempt_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND)`); err != nil {
		t.Fatal(err)
	}
	if err = f.app.processEmailV204(ctx, sender); err != nil {
		t.Fatal(err)
	}
	status, _ = f.store.EmailStatusV204(ctx)
	if status["pending"] != 0 || sends != 2 {
		t.Fatal("delivery was not recorded")
	}
	f.app.processEmailV204(ctx, sender)
	if sends != 2 {
		t.Fatal("sent email duplicated")
	}
}
