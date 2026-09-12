//go:build integration

package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

const deletionAdminPassword = "current-admin-password-123"
const deletionTargetPassword = "target-user-password-456"

func deletionFixture(t *testing.T) *socialFixtureV2 {
	t.Helper()
	f := newSocialFixtureV2(t)
	for name, password := range map[string]string{"Alice": deletionAdminPassword, "Bob": deletionTargetPassword} {
		u := f.users[name]
		u.PasswordHash = identity.Hash(password)
		f.users[name] = u
		if _, err := f.store.DB.Exec("UPDATE identities SET password_hash=? WHERE id=?", u.PasswordHash, u.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range f.users {
		if _, err := f.store.DB.Exec("INSERT INTO identity_aliases(alias_key,user_id) VALUES(?,?),(?,?)", strings.ToLower(u.GameID), u.ID, u.QQ, u.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	return f
}
func deletionProof(f *socialFixtureV2, name string) store.AccountDeletionProof {
	return store.AccountDeletionProof{ActorID: f.users[name].ID, SessionHash: store.Digest([]byte(f.tokens[name])), CredentialHash: f.users[name].PasswordHash}
}
func deleteBody(password, key string) map[string]any {
	return map[string]any{"clientRequestId": key, "expectedVersion": 1, "password": password}
}
func deletionSQL(t *testing.T, f *socialFixtureV2, q string, args ...any) {
	t.Helper()
	if _, err := f.store.DB.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}
func deletionCount(t *testing.T, f *socialFixtureV2, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.store.DB.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func deletionOrder(t *testing.T, f *socialFixtureV2, state, funds string) string {
	t.Helper()
	buyer, seller := f.users["Carol"], f.users["Bob"]
	party := func(u store.User) map[string]any {
		return map[string]any{"kind": "PLAYER", "playerRef": u.PlayerRef, "displayName": u.GameID, "contactQq": u.QQ}
	}
	body, _ := json.Marshal(map[string]any{"buyer": party(buyer), "seller": party(seller), "items": []any{}, "delivery": map[string]any{"method": "SELF_PICKUP"}})
	sha := store.Digest(body)
	deletionSQL(t, f, `INSERT INTO commerce_resources_v2(resource_id,resource_kind,channel,owner_id,owner_uuid,payee_id,payee_uuid,escrow_ref,amount,settled_amount,refunded_amount,state,funds_state,body,snapshot_id,snapshot_sha256,version,refund_attempts,automatic,created_at,updated_at) VALUES('order_erasure','ORDER','PLAYER_MARKET',?,?,?,?, 'escrow_erasure','20.00','0.00','0.00',?,?,?,'snapshot_erasure',?,1,0,FALSE,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, buyer.ID, buyer.ServerUUID, seller.ID, seller.ServerUUID, state, funds, string(body), sha)
	deletionSQL(t, f, `INSERT INTO commerce_snapshots_v2(snapshot_id,resource_id,resource_kind,resource_version,body,sha256,created_at) VALUES('snapshot_erasure','order_erasure','ORDER',1,?,?,UTC_TIMESTAMP(6))`, string(body), sha)
	return sha
}

func TestAccountDeletionPasswordAndFullPresentationCleanup(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	bob := f.users["Bob"]
	path := "/api/v1/admin/accounts/" + bob.ID + "/delete"
	ab := f.conversation(t, "Alice", "Bob", "ab")
	bc := f.conversation(t, "Bob", "Carol", "bc")
	ac := f.conversation(t, "Alice", "Carol", "ac")
	message := f.send(t, "Bob", ab, "bob-message", "private erasure secret")["messageId"].(string)
	f.send(t, "Carol", bc, "carol-message", "other participant private text")
	f.request(t, "Alice", "POST", "/api/v1/chat/conversations/"+ac+"/forwards", map[string]any{"clientMessageId": "forward-bob", "sourceConversationId": ab, "sourceMessageId": message}, 200)
	f.request(t, "Alice", "POST", "/api/v1/chat/follows", map[string]any{"playerRef": bob.PlayerRef}, 200)
	f.request(t, "Carol", "POST", "/api/v1/chat/follows", map[string]any{"playerRef": bob.PlayerRef}, 200)
	pub, _, err := f.store.PublishAppChatV2(ctx, bob, "public-bob", "old public message", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.store.PublishAppChatV2(ctx, f.users["Carol"], "reply-bob", "@Bob hello", pub, []string{bob.PlayerRef}, nil); err != nil {
		t.Fatal(err)
	}
	deletionSQL(t, f, `INSERT INTO social_profiles_v2(user_id,bio,version,updated_at) VALUES(?,'old private bio',1,UTC_TIMESTAMP(6))`, bob.ID)
	deletionSQL(t, f, `INSERT INTO catalog_records_v2(resource_id,kind,owner_id,version,body,state,available_stock,title,created_at,updated_at) VALUES('listing_bob','listing',?,1,'{"title":"private listing"}','ACTIVE',2,'private listing',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, bob.ID)
	sha := deletionOrder(t, f, "CANCELLED", "UNPAID")
	if _, err = f.store.RememberCorePlayer(ctx, store.CorePlayerIdentity{PlayerUUID: bob.ServerUUID, GameID: bob.GameID, ServerID: "amiya"}); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Bob", "POST", path, deleteBody(deletionTargetPassword, "denied"), 403)
	wrong := f.request(t, "Alice", "POST", path, deleteBody(deletionTargetPassword, "delete-once"), 400)
	if wrong["error"].(map[string]any)["code"] != "ADMIN_PASSWORD_INVALID" {
		t.Fatal(wrong)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM identities WHERE id=? AND status='active'", bob.ID) != 1 {
		t.Fatal("wrong password changed target")
	}
	body := deleteBody(deletionAdminPassword, "delete-once")
	result := socialDataV2(f.request(t, "Alice", "POST", path, body, 200))
	if result["deleted"] != true {
		t.Fatal(result)
	}
	f.request(t, "Alice", "POST", path, body, 200)
	if deletionCount(t, f, "SELECT COUNT(*) FROM account_deletions WHERE user_id=?", bob.ID) != 1 {
		t.Fatal("replay duplicated deletion")
	}
	f.request(t, "Bob", "GET", "/api/v1/account/me", nil, 401)
	f.request(t, "Alice", "GET", "/api/v1/players/"+bob.PlayerRef, nil, 404)
	f.request(t, "Alice", "GET", "/api/v1/market/listings/listing_bob", nil, 404)
	f.request(t, "Alice", "GET", "/api/v1/chat/conversations/"+ab+"/messages", nil, 404)
	for _, q := range []string{"SELECT COUNT(*) FROM identity_aliases WHERE user_id=?", "SELECT COUNT(*) FROM identity_sessions WHERE user_id=?", "SELECT COUNT(*) FROM social_profiles_v2 WHERE user_id=?", "SELECT COUNT(*) FROM social_follows_v2 WHERE followed_id=?"} {
		if deletionCount(t, f, q, bob.ID) != 0 {
			t.Fatal("account reference remained", q)
		}
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM social_conversations_v2") != 1 {
		t.Fatal("unrelated conversation lost or old one retained")
	}
	var content, forward string
	if err = f.store.DB.QueryRow("SELECT content,forwarded_json FROM social_messages_v2 WHERE conversation_id=?", ac).Scan(&content, &forward); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content+forward, "private erasure secret") || !strings.Contains(forward, "UNAVAILABLE") {
		t.Fatal("forward exposed deleted private content")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM chat_messages_next WHERE sender_ref=?", bob.PlayerRef) != 0 {
		t.Fatal("public author remained")
	}
	var public string
	if err = f.store.DB.QueryRow("SELECT content FROM chat_messages_next WHERE client_message_id='reply-bob'").Scan(&public); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(public, "@Bob") {
		t.Fatal("structured mention remained")
	}
	view, err := f.store.CommerceViewV2(ctx, f.users["Carol"].ID, "order_erasure", false)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(view)
	if strings.Contains(string(encoded), `"displayName":"Bob"`) || !strings.Contains(string(encoded), store.DeletedAccountName) {
		t.Fatal("historical party was not anonymized", string(encoded))
	}
	auditOrders, err := f.store.AdminOrdersV204(ctx, f.users["Alice"].ID, store.AdminCommerceFilterV204{PlayerRef: bob.PlayerRef}, 30)
	if err != nil || len(auditOrders) != 1 {
		t.Fatal("erased account lost its audit trail", err)
	}
	var persistedSHA string
	if err = f.store.DB.QueryRow("SELECT sha256 FROM commerce_snapshots_v2 WHERE snapshot_id='snapshot_erasure'").Scan(&persistedSHA); err != nil || persistedSHA != sha {
		t.Fatal("financial evidence changed", err)
	}
	feed := socialDataV2(f.request(t, "Carol", "GET", "/api/v1/account/deletions", nil, 200))
	items := feed["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["playerRef"] != bob.PlayerRef || len(items[0].(map[string]any)) != 2 {
		t.Fatal("deletion sync leaked identity or missed target", feed)
	}
	f.request(t, "Alice", "POST", "/api/v1/admin/accounts/"+bob.ID, store.AdminAccountChangeV206{ClientRequestID: "revive", ExpectedVersion: 2, Action: "unban"}, 409)
	if _, err = f.store.UserByAlias(ctx, "bob"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("login binding not released", err)
	}
	if _, err = f.store.RememberCorePlayer(ctx, store.CorePlayerIdentity{PlayerUUID: bob.ServerUUID, GameID: "Bob", ServerID: "amiya"}); !errors.Is(err, store.ErrSocialNotFound) {
		t.Fatal("game directory revived erased identity", err)
	}
	if _, _, err = f.store.PublishAppChatV2(ctx, bob, "late-public", "must not return", "", nil, nil); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("in-flight user could write after deletion", err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM social_requests_v2 WHERE LOCATE(?,response_json)>0", deletionAdminPassword) != 0 {
		t.Fatal("confirmation password persisted")
	}

	v := store.GameVerification{ID: "verify_fresh_account", TokenHash: store.Digest([]byte("new-token")), CodeHash: store.Digest([]byte("new-code")), Purpose: "register", PlayerUUID: bob.ServerUUID, GameID: "BobNew", QQ: bob.QQ, NodeID: "amiya", ExpiresAt: time.Now().Add(time.Minute)}
	if err = f.store.NewGameVerification(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err = f.store.ActivateGameVerification(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	newUser, err := f.store.RegisterGameUser(ctx, v, identity.Hash(deletionTargetPassword))
	if err != nil {
		t.Fatal(err)
	}
	if newUser.ID == bob.ID || newUser.PlayerRef == bob.PlayerRef {
		t.Fatal("new account reused erased identity")
	}
	newOrders, err := f.store.AdminOrdersV204(ctx, f.users["Alice"].ID, store.AdminCommerceFilterV204{PlayerRef: newUser.PlayerRef}, 30)
	if err != nil || len(newOrders) != 0 {
		t.Fatal("new registration inherited the old account's audit orders", err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM social_conversations_v2 WHERE user_lo=? OR user_hi=?", newUser.ID, newUser.ID) != 0 {
		t.Fatal("new account inherited conversations")
	}
}

func TestAccountDeletionBlocksOpenTransactionsAndStaleProof(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	bob := f.users["Bob"]
	request := store.AccountDeletionRequest{ClientRequestID: "blocked", ExpectedVersion: 1}
	proof := deletionProof(f, "Alice")
	deletionOrder(t, f, "PAID", "HELD")
	if _, err := f.store.DeleteAccount(ctx, proof, bob.ID, request); err == nil {
		t.Fatal("deleted an account with held funds")
	}
	deletionSQL(t, f, "UPDATE commerce_resources_v2 SET state='CANCELLED',funds_state='UNPAID'")
	deletionSQL(t, f, "UPDATE identities SET password_hash=? WHERE id=?", identity.Hash("changed-admin-password"), proof.ActorID)
	if _, err := f.store.DeleteAccount(ctx, proof, bob.ID, request); err == nil {
		t.Fatal("password change did not invalidate confirmation")
	}
	deletionSQL(t, f, "UPDATE identities SET password_hash=? WHERE id=?", proof.CredentialHash, proof.ActorID)
	if err := f.store.Revoke(ctx, proof.SessionHash); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.DeleteAccount(ctx, proof, bob.ID, request); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("revoked session confirmation accepted", err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM account_deletions") != 0 {
		t.Fatal("blocked deletion left effects")
	}
}

func TestAccountDeletionConcurrentAdminsPreserveOneAdmin(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Bob"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, pair := range [][2]string{{"Alice", "Bob"}, {"Bob", "Alice"}} {
		wg.Add(1)
		go func(pair [2]string) {
			defer wg.Done()
			_, err := f.store.DeleteAccount(ctx, deletionProof(f, pair[0]), f.users[pair[1]].ID, store.AccountDeletionRequest{ClientRequestID: "concurrent", ExpectedVersion: 1})
			results <- err
		}(pair)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM identities i JOIN identity_permissions p ON p.user_id=i.id WHERE i.status='active' AND p.permission='platform.admin'") != 1 {
		t.Fatal("mutual deletion removed all administrators", success)
	}
}

func TestAccountDeletionPasswordAttemptsAreBounded(t *testing.T) {
	f := deletionFixture(t)
	path := "/api/v1/admin/accounts/" + f.users["Bob"].ID + "/delete"
	for i := 0; i < 8; i++ {
		f.request(t, "Alice", "POST", path, deleteBody("incorrect-password", "retry"), 400)
	}
	f.request(t, "Alice", "POST", path, deleteBody(deletionAdminPassword, "retry"), 429)
	if deletionCount(t, f, "SELECT COUNT(*) FROM account_deletions") != 0 {
		t.Fatal("failed password attempts changed accounts")
	}
}

func TestAccountDeletionWebSessionRequiresCsrfAndRejectsSelf(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	actor := f.users["Alice"]
	token := identity.Secret()
	if err := f.store.CreateSession(ctx, actor, store.Digest([]byte(token)), "web", "confirmation-csrf", time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	post := func(target, origin, csrf string, want int) {
		t.Helper()
		raw, _ := json.Marshal(deleteBody(deletionAdminPassword, "web-delete"))
		r, _ := http.NewRequest("POST", f.http.URL+"/api/v1/admin/accounts/"+target+"/delete", strings.NewReader(string(raw)))
		r.AddCookie(&http.Cookie{Name: "deuterium_dev_session", Value: token})
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		r.Header.Set("X-CSRF-Token", csrf)
		resp, err := f.http.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("web deletion status=%d want=%d", resp.StatusCode, want)
		}
	}
	post(f.users["Bob"].ID, "https://outside.invalid", "confirmation-csrf", 403)
	post(f.users["Bob"].ID, f.http.URL, "", 403)
	post(actor.ID, f.http.URL, "confirmation-csrf", 409)
	if deletionCount(t, f, "SELECT COUNT(*) FROM account_deletions") != 0 {
		t.Fatal("unauthorized web action changed accounts")
	}
	post(f.users["Bob"].ID, f.http.URL, "confirmation-csrf", 200)
}

func TestAccountDeletionRemovesProfileImagesButRetainsOrderEvidence(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	bob := f.users["Bob"]
	deletionOrder(t, f, "CANCELLED", "UNPAID")
	for _, key := range []string{"profile-only", "order-evidence"} {
		u, err := f.store.CreateAssetUploadV2(ctx, store.AssetUploadV2{UploadID: "upload-" + key, AssetID: "asset-" + key, UserID: bob.ID, ClientRequestID: key, Fingerprint: store.Digest([]byte(key)), Purpose: "AVATAR", BusinessType: "PROFILE", BusinessRef: bob.ID, ObjectKey: "test/" + key, ContentType: "image/png", SizeBytes: 100, MD5: "AAAAAAAAAAAAAAAAAAAAAA==", AltText: "old profile description"})
		if err != nil {
			t.Fatal(err)
		}
		deletionSQL(t, f, "UPDATE asset_uploads_v2 SET status='READY',width=10,height=10,sha256=?,was_bound=TRUE WHERE asset_id=?", strings.Repeat("a", 64), u.AssetID)
		deletionSQL(t, f, "INSERT INTO asset_bindings_v2 VALUES(?,'PROFILE',?)", u.AssetID, bob.ID)
		if key == "order-evidence" {
			deletionSQL(t, f, "INSERT INTO asset_bindings_v2 VALUES(?,'ORDER_SNAPSHOT','snapshot_erasure')", u.AssetID)
		}
	}
	if _, err := f.store.DeleteAccount(ctx, deletionProof(f, "Alice"), bob.ID, store.AccountDeletionRequest{ClientRequestID: "delete-images", ExpectedVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM asset_bindings_v2 WHERE business_type='PROFILE' AND business_ref=?", bob.ID) != 0 {
		t.Fatal("old profile asset remained accessible")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM asset_uploads_v2 WHERE client_request_id='profile-only' AND removed_at IS NOT NULL") != 1 {
		t.Fatal("unbound private image not removed")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM asset_uploads_v2 a JOIN asset_bindings_v2 b ON b.asset_id=a.asset_id WHERE b.business_type='ORDER_SNAPSHOT' AND a.removed_at IS NULL") != 1 {
		t.Fatal("other party's order image was deleted")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM asset_uploads_v2 WHERE user_id=? AND alt_text<>''", bob.ID) != 0 {
		t.Fatal("old profile image description remained")
	}
}

func TestAccountDeletionKeepsHistoricalTransferRecoveryAndBlocksNewPayment(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	sender, bob := f.users["Alice"], f.users["Bob"]
	recipient, err := f.store.WalletRecipient(ctx, bob.PlayerRef)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := f.store.CreateWalletTransfer(ctx, sender, "original-transfer", "amiya", recipient, "5.00", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, pending := range []string{"QUEUED", "UNKNOWN"} {
		deletionSQL(t, f, "UPDATE core_operations SET state=? WHERE command_type='wallet.transfer'", pending)
		_, blockedErr := f.store.DeleteAccount(ctx, deletionProof(f, "Alice"), bob.ID, store.AccountDeletionRequest{ClientRequestID: "delete-recipient", ExpectedVersion: 1})
		var blocked *store.CatalogErrorV2
		if !errors.As(blockedErr, &blocked) || blocked.Code != "ACCOUNT_HAS_PENDING_TRANSFER" {
			t.Fatal("pending transfer did not block deletion", pending, blockedErr)
		}
	}
	deletionSQL(t, f, "UPDATE core_operations SET state='COMPLETED' WHERE command_type='wallet.transfer'")
	if _, err = f.store.DeleteAccount(ctx, deletionProof(f, "Alice"), bob.ID, store.AccountDeletionRequest{ClientRequestID: "delete-recipient", ExpectedVersion: 1}); err != nil {
		t.Fatal(err)
	}
	history, err := f.store.WalletTransfer(ctx, id)
	if err != nil || history.Recipient.GameID != store.DeletedAccountName || history.Amount != "5.00" {
		t.Fatal("historical receipt was lost", err, history)
	}
	request := map[string]any{"clientRequestId": "original-transfer", "recipientPlayerRef": bob.PlayerRef, "amount": "5.00"}
	f.request(t, "Alice", "POST", "/api/v1/wallet/transfers", request, 200)
	request["clientRequestId"] = "new-transfer"
	f.request(t, "Alice", "POST", "/api/v1/wallet/transfers", request, 400)
	if deletionCount(t, f, "SELECT COUNT(*) FROM wallet_transfers_next") != 1 {
		t.Fatal("recovery created another payment")
	}
}

func TestGameOnlyTransferHistorySurvivesRecipientRegistration(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	uuid := "e97161f9-2a7c-4abd-a8e7-6fd64a64c009"
	ref, err := f.store.RememberCorePlayer(ctx, store.CorePlayerIdentity{PlayerUUID: uuid, GameID: "NewPlayer", ServerID: "amiya"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.store.WalletRecipient(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := f.store.CreateWalletTransfer(ctx, f.users["Alice"], "before-registration", "amiya", p, "2.00", nil)
	if err != nil {
		t.Fatal(err)
	}
	deletionSQL(t, f, "UPDATE core_operations SET state='COMPLETED' WHERE command_type='wallet.transfer'")
	v := store.GameVerification{ID: "verify_new_game_user", TokenHash: store.Digest([]byte("new-token")), CodeHash: store.Digest([]byte("new-code")), Purpose: "register", PlayerUUID: uuid, GameID: "NewPlayer", QQ: "10009", NodeID: "amiya", ExpiresAt: time.Now().Add(time.Minute)}
	if err = f.store.NewGameVerification(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err = f.store.ActivateGameVerification(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	u, err := f.store.RegisterGameUser(ctx, v, identity.Hash(deletionTargetPassword))
	if err != nil {
		t.Fatal(err)
	}
	if u.PlayerRef == ref {
		t.Fatal("registration reused the prior unregistered reference")
	}
	receipt, err := f.store.WalletTransfer(ctx, id)
	if err != nil || receipt.Recipient.GameID != "NewPlayer" || receipt.Amount != "2.00" {
		t.Fatal("registration broke old receipt", err, receipt)
	}
}
