//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

const admissionTestUUID = "00000000-0000-4000-8000-000000000091"
const admissionTestGateway = "admission-test-gateway-credential-1234567890"

func admissionFixture(t *testing.T) *socialFixtureV2 {
	f := newSocialFixtureV2(t)
	f.app.Config.AdmissionPublicOrigin = f.http.URL
	f.app.Config.AdmissionGatewayTokenSHA256 = store.Digest([]byte(admissionTestGateway))
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: admissionTestUUID, Name: name}, nil
	}
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	return f
}
func admissionInput() store.AdmissionSubmission {
	return store.AdmissionSubmission{ReceiptToken: identity.Secret(), GameID: "NewPlayer", QQ: "123456789", Interests: []string{"建筑创造"}, Message: "一起建设", CovenantVersion: store.AdmissionCovenantVersion, CovenantAccepted: true}
}
func admissionRaw(t *testing.T, f *socialFixtureV2, path string, in any, headers map[string]string, status int) map[string]any {
	t.Helper()
	data, _ := json.Marshal(in)
	r, _ := http.NewRequest("POST", f.http.URL+path, bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	response, err := f.http.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result map[string]any
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != status {
		t.Fatalf("%s status %d want %d: %v", path, response.StatusCode, status, result)
	}
	return result
}

func TestAdmissionPublicReceiptIsolationAndReplay(t *testing.T) {
	f := admissionFixture(t)
	in := admissionInput()
	in.CovenantAccepted = false
	f.request(t, "", "POST", "/api/v1/admission/applications", in, 400)
	in.CovenantAccepted = true
	admissionRaw(t, f, "/api/v1/admission/applications", in, map[string]string{"Origin": "https://untrusted.invalid"}, 403)
	result := socialDataV2(f.request(t, "", "POST", "/api/v1/admission/applications", in, 200))
	if result["status"] != "PENDING" || result["accessAllowed"] != false {
		t.Fatal("submission granted access")
	}
	replay := socialDataV2(f.request(t, "", "POST", "/api/v1/admission/applications", in, 200))
	if result["applicationId"] != replay["applicationId"] {
		t.Fatal("replayed request duplicated")
	}
	changed := in
	changed.Message = "different"
	f.request(t, "", "POST", "/api/v1/admission/applications", changed, 409)
	other := in
	other.ReceiptToken = identity.Secret()
	f.request(t, "", "POST", "/api/v1/admission/applications", other, 409)
	f.request(t, "", "POST", "/api/v1/admission/status", map[string]string{"receiptToken": other.ReceiptToken}, 404)
	queried := socialDataV2(f.request(t, "", "POST", "/api/v1/admission/status", map[string]string{"receiptToken": in.ReceiptToken}, 200))
	if queried["applicationId"] != result["applicationId"] {
		t.Fatal("wrong receipt")
	}
	for _, key := range []string{"qq", "uuid", "reviewer", "receiptToken"} {
		if _, ok := queried[key]; ok {
			t.Fatalf("unnecessary private field %s", key)
		}
	}
	var count int
	f.store.DB.QueryRow("SELECT COUNT(*) FROM admission_applications").Scan(&count)
	if count != 1 {
		t.Fatal("duplicate applications")
	}
}

func TestAdmissionReviewGrantRevokeAndKickReceipt(t *testing.T) {
	f := admissionFixture(t)
	in := admissionInput()
	created := socialDataV2(f.request(t, "", "POST", "/api/v1/admission/applications", in, 200))
	id := created["applicationId"].(string)
	decision := store.AdmissionDecision{ClientRequestID: "approve-one", ExpectedVersion: 1, Decision: "APPROVE"}
	path := "/api/v1/admin/whitelist/applications/" + id + "/review"
	f.request(t, "", "POST", path, decision, 401)
	f.request(t, "Bob", "POST", path, decision, 403)
	f.request(t, "Alice", "POST", path, decision, 400)
	decision.QQMemberConfirmed = true
	decision.IdentityConfirmed = true
	f.request(t, "Alice", "POST", path, decision, 200)
	f.request(t, "Alice", "POST", path, decision, 200)
	var events int
	f.store.DB.QueryRow("SELECT COUNT(*) FROM admission_events").Scan(&events)
	if events != 1 {
		t.Fatal("duplicate audit")
	}
	entry, err := f.store.AdmissionCheck(context.Background(), admissionTestUUID)
	if err != nil || !entry.Allowed || entry.Version != 1 {
		t.Fatal("approval not active", entry, err)
	}
	revoke := store.AdmissionRevoke{ClientRequestID: "revoke-one", ExpectedVersion: 1, Reason: "审核资料需重新核对", KickOnline: true}
	revokePath := "/api/v1/admin/whitelist/entries/" + admissionTestUUID + "/revoke"
	f.request(t, "Alice", "POST", revokePath, revoke, 200)
	f.request(t, "Alice", "POST", revokePath, revoke, 200)
	entry, err = f.store.AdmissionCheck(context.Background(), admissionTestUUID)
	if err != nil || entry.Allowed || entry.Version != 2 {
		t.Fatal("revocation did not take effect")
	}
	state := socialDataV2(f.request(t, "", "POST", "/api/v1/admission/status", map[string]string{"receiptToken": in.ReceiptToken}, 200))
	if state["status"] != "APPROVED" || state["accessStatus"] != "REVOKED" {
		t.Fatal("lost application history", state)
	}
	inPoll := store.AdmissionHeartbeat{InstanceID: "proxy-one", PluginVersion: "1.0.0", OnlinePlayers: 3, Acks: []store.AdmissionKickAck{}}
	admissionRaw(t, f, "/bridge/v1/admission/poll", inPoll, nil, 401)
	headers := map[string]string{"Authorization": "Bearer " + admissionTestGateway}
	poll := socialDataV2(admissionRaw(t, f, "/bridge/v1/admission/poll", inPoll, headers, 200))
	commands := poll["commands"].([]any)
	if len(commands) != 1 {
		t.Fatal("kick command missing")
	}
	command := commands[0].(map[string]any)
	inPoll.Acks = []store.AdmissionKickAck{{ID: command["commandId"].(string), Status: "DISCONNECTED"}}
	poll = socialDataV2(admissionRaw(t, f, "/bridge/v1/admission/poll", inPoll, headers, 200))
	if len(poll["commands"].([]any)) != 0 {
		t.Fatal("acknowledged kick replayed")
	}
	summary := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/admin/whitelist/summary", nil, 200))
	if summary["gateway"].(map[string]any)["online"] != true {
		t.Fatal("missing heartbeat")
	}
}

func TestAdmissionConcurrentReviewHasSingleDecision(t *testing.T) {
	f := admissionFixture(t)
	created := socialDataV2(f.request(t, "", "POST", "/api/v1/admission/applications", admissionInput(), 200))
	path := "/api/v1/admin/whitelist/applications/" + created["applicationId"].(string) + "/review"
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, decision := range []string{"APPROVE", "REJECT"} {
		wg.Add(1)
		go func(value string) {
			defer wg.Done()
			status, _, err := f.call("Alice", "POST", path, store.AdmissionDecision{ClientRequestID: "race-" + value, ExpectedVersion: 1, Decision: value, Reason: "review", QQMemberConfirmed: true, IdentityConfirmed: true})
			if err != nil {
				statuses <- 0
			} else {
				statuses <- status
			}
		}(decision)
	}
	wg.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal("conflicting approvals", counts)
	}
	var n int
	f.store.DB.QueryRow("SELECT COUNT(*) FROM admission_events").Scan(&n)
	if n != 1 {
		t.Fatal("multiple decision events")
	}
}

func TestAdmissionManualAddAndRegrantCancelOldKick(t *testing.T) {
	f := admissionFixture(t)
	in := store.AdmissionManualAdd{ClientRequestID: "manual-one", GameID: "NewPlayer", ExpectedUUID: admissionTestUUID, QQ: "123456789", Reason: "管理员核对后添加", IdentityConfirmed: true, QQMemberConfirmed: true}
	f.request(t, "Bob", "POST", "/api/v1/admin/whitelist/entries", in, 403)
	f.request(t, "Alice", "POST", "/api/v1/admin/whitelist/entries", in, 200)
	f.request(t, "Alice", "POST", "/api/v1/admin/whitelist/entries", in, 200)
	f.request(t, "Alice", "POST", "/api/v1/admin/whitelist/entries/"+admissionTestUUID+"/revoke", store.AdmissionRevoke{ClientRequestID: "revoke", ExpectedVersion: 1, Reason: "临时撤销", KickOnline: true}, 200)
	in.ClientRequestID = "manual-again"
	f.request(t, "Alice", "POST", "/api/v1/admin/whitelist/entries", in, 200)
	var status string
	f.store.DB.QueryRow("SELECT status FROM admission_kicks").Scan(&status)
	if status != "CANCELLED" {
		t.Fatal("regrant left stale kick")
	}
	list := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/admin/whitelist/entries?status=ACTIVE&q=NewPlayer", nil, 200))
	if list["total"] != float64(1) {
		t.Fatal("manual grant missing")
	}
	var apps int
	f.store.DB.QueryRow("SELECT COUNT(*) FROM admission_applications").Scan(&apps)
	if apps != 0 {
		t.Fatal("manual add invented application")
	}
}

func TestAdmissionAdminCookieRequiresCSRF(t *testing.T) {
	f := admissionFixture(t)
	token, csrf := identity.Secret(), identity.Secret()
	if err := f.store.CreateSession(context.Background(), f.users["Alice"], store.Digest([]byte(token)), "web", csrf, time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	in := store.AdmissionManualAdd{ClientRequestID: "cookie-add", GameID: "NewPlayer", ExpectedUUID: admissionTestUUID, QQ: "123456789", Reason: "核对", IdentityConfirmed: true, QQMemberConfirmed: true}
	headers := map[string]string{"Cookie": f.app.cookieName() + "=" + token, "Origin": f.http.URL}
	admissionRaw(t, f, "/api/v1/admin/whitelist/entries", in, headers, 403)
	headers["X-CSRF-Token"] = csrf
	admissionRaw(t, f, "/api/v1/admin/whitelist/entries", in, headers, 200)
}

func TestAdmissionGatewayDeniesBadIdentityAndClosedStorage(t *testing.T) {
	f := admissionFixture(t)
	headers := map[string]string{"Authorization": "Bearer " + admissionTestGateway}
	admissionRaw(t, f, "/bridge/v1/admission/check", map[string]string{"uuid": admissionTestUUID}, map[string]string{"Authorization": "Bearer " + strings.Repeat("x", 43)}, 401)
	a := socialDataV2(admissionRaw(t, f, "/bridge/v1/admission/check", map[string]string{"uuid": admissionTestUUID}, headers, 200))
	if a["allowed"] != false {
		t.Fatal("unknown user allowed")
	}
	admissionRaw(t, f, "/bridge/v1/admission/check", map[string]string{"uuid": "not-a-uuid"}, headers, 400)
	f.app.Close()
	f.store.DB.Close()
	admissionRaw(t, f, "/bridge/v1/admission/check", map[string]string{"uuid": admissionTestUUID}, headers, 503)
}

func TestAdmissionImportIsDryByDefaultAndNeverReactivatesRevokedPlayers(t *testing.T) {
	f := admissionFixture(t)
	players := []store.AdmissionImportPlayer{{UUID: admissionTestUUID, GameID: "NewPlayer", QQ: "123456789"}}
	ctx := context.Background()
	result, err := f.store.ImportAdmission(ctx, players, false)
	if err != nil || result.New != 1 || result.Applied {
		t.Fatal(result, err)
	}
	access, _ := f.store.AdmissionCheck(ctx, admissionTestUUID)
	if access.Allowed {
		t.Fatal("dry import mutated whitelist")
	}
	result, err = f.store.ImportAdmission(ctx, players, true)
	if err != nil || result.New != 1 {
		t.Fatal(result, err)
	}
	f.request(t, "Alice", "POST", "/api/v1/admin/whitelist/entries/"+admissionTestUUID+"/revoke", store.AdmissionRevoke{ClientRequestID: "revoke-import", ExpectedVersion: 1, Reason: "撤销权限"}, 200)
	result, err = f.store.ImportAdmission(ctx, players, true)
	if err != nil || result.New != 0 || result.Existing != 1 {
		t.Fatal(result, err)
	}
	access, _ = f.store.AdmissionCheck(ctx, admissionTestUUID)
	if access.Allowed || access.Version != 2 {
		t.Fatal("import reactivated revoked player")
	}
}
