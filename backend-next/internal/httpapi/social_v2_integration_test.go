//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

type socialFixtureV2 struct {
	store  *store.Store
	app    *Server
	http   *httptest.Server
	users  map[string]store.User
	tokens map[string]string
}

func newSocialFixtureV2(t *testing.T) *socialFixtureV2 {
	t.Helper()
	db := testdb.New(t)
	f := &socialFixtureV2{store: db, users: map[string]store.User{}, tokens: map[string]string{}}
	for n, name := range []string{"Alice", "Bob", "Carol"} {
		u := store.User{ID: "social_" + name, PlayerRef: "player_" + name, GameID: name, QQ: fmt.Sprintf("1000%d", n+1), ServerUUID: fmt.Sprintf("d97161f9-2a7c-4abd-a8e7-6fd64a64c00%d", n+1), PasswordHash: "test-unused-hash", IdentityStatus: "bound", Status: "active"}
		_, err := db.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES(?,?,?,?,?,?,'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)`, u.ID, u.PlayerRef, u.ServerUUID, u.GameID, u.QQ, u.PasswordHash, strings.Repeat("0", 64))
		if err != nil {
			t.Fatal(err)
		}
		token := identity.Secret()
		if err = db.CreateSession(context.Background(), u, store.Digest([]byte(token)), "app", "", time.Now().Add(time.Hour), ""); err != nil {
			t.Fatal(err)
		}
		f.users[name] = u
		f.tokens[name] = token
	}
	f.app = New(db, config.Config{Development: true, PublicOrigin: "http://127.0.0.1:8080"})
	f.http = httptest.NewServer(f.app.Handler())
	f.app.Config.PublicOrigin = f.http.URL
	t.Cleanup(func() { f.app.Close(); f.http.Close() })
	return f
}
func (f *socialFixtureV2) call(user, method, path string, input any) (int, map[string]any, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return 0, nil, err
	}
	request, err := http.NewRequest(method, f.http.URL+path, bytes.NewReader(encoded))
	if err != nil {
		return 0, nil, err
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		request.Header.Set("Authorization", "Bearer "+f.tokens[user])
	}
	reply, err := f.http.Client().Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer reply.Body.Close()
	var body map[string]any
	if err = json.NewDecoder(reply.Body).Decode(&body); err != nil {
		return 0, nil, err
	}
	return reply.StatusCode, body, nil
}
func (f *socialFixtureV2) request(t *testing.T, user, method, path string, input any, status int) map[string]any {
	t.Helper()
	got, body, err := f.call(user, method, path, input)
	if err != nil {
		t.Fatal(err)
	}
	if got != status {
		t.Fatalf("%s %s as %s: got %d want %d body=%v", method, path, user, got, status, body)
	}
	return body
}
func socialDataV2(body map[string]any) map[string]any { return body["data"].(map[string]any) }
func (f *socialFixtureV2) conversation(t *testing.T, user, other, key string) string {
	t.Helper()
	return socialDataV2(f.request(t, user, "POST", "/api/v1/chat/conversations", map[string]any{"clientRequestId": key, "otherPlayerRef": f.users[other].PlayerRef}, 201))["conversationId"].(string)
}
func (f *socialFixtureV2) send(t *testing.T, user, conversation, key, content string) map[string]any {
	t.Helper()
	return socialDataV2(f.request(t, user, "POST", "/api/v1/chat/conversations/"+conversation+"/messages", map[string]any{"clientMessageId": key, "content": content}, 200))
}

func TestSocialV2ProfilesFollowsAndNoSeedData(t *testing.T) {
	f := newSocialFixtureV2(t)
	for _, path := range []string{"/api/v1/announcements", "/api/v1/notifications", "/api/v1/chat/conversations"} {
		body := f.request(t, "Alice", "GET", path, nil, 200)
		if len(body["data"].([]any)) != 0 {
			t.Fatalf("unexpected seed data at %s", path)
		}
	}
	f.request(t, "", "GET", "/api/v1/players/player_Alice", nil, 401)
	original := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/players/player_Alice", nil, 200))
	if original["version"] != float64(1) || original["bio"] != "" || original["lastSeenAt"] != nil {
		t.Fatal("initial profile invents metadata")
	}
	input := map[string]any{"clientRequestId": "bio-1", "expectedVersion": 1, "bio": "真实个人简介"}
	changed := socialDataV2(f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", input, 200))
	if changed["bio"] != "真实个人简介" || changed["version"] != float64(2) {
		t.Fatal("profile update not persisted")
	}
	replay := socialDataV2(f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", input, 200))
	if replay["version"] != float64(2) {
		t.Fatal("idempotent profile update changed version")
	}
	input["bio"] = "different"
	f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", input, 409)
	input["clientRequestId"] = "bio-stale"
	f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", input, 409)
	input["playerRef"] = "player_Bob"
	f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", input, 400)
	other := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/players/player_Alice", nil, 200))
	if other["bio"] != "真实个人简介" {
		t.Fatal("cross-device profile is stale")
	}
	f.request(t, "Alice", "POST", "/api/v1/chat/follows", map[string]string{"playerRef": "player_Alice"}, 400)
	f.request(t, "Alice", "POST", "/api/v1/chat/follows", map[string]string{"playerRef": "player_Bob"}, 200)
	followed := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/chat/follows", nil, 200))["players"].([]any)
	if len(followed) != 1 {
		t.Fatal("follow not stored")
	}
	stranger := socialDataV2(f.request(t, "Carol", "GET", "/api/v1/chat/follows", nil, 200))["players"].([]any)
	if len(stranger) != 0 {
		t.Fatal("follow lists leaked")
	}
	f.request(t, "Alice", "DELETE", "/api/v1/chat/follows/player_Bob", nil, 200)
	directory := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/chat/player-directory?query=Alice", nil, 200))["players"].([]any)
	if len(directory) != 1 {
		t.Fatal("directory search not scoped to query")
	}
}

func TestSocialV2PrivateMessagesReplyForwardAndNotificationIsolation(t *testing.T) {
	f := newSocialFixtureV2(t)
	ab := f.conversation(t, "Alice", "Bob", "ab-first")
	again := f.conversation(t, "Bob", "Alice", "ba-first")
	if ab != again {
		t.Fatal("pair created a second conversation")
	}
	first := f.send(t, "Alice", ab, "message-one", "只给 Bob 的消息")
	messageID := first["messageId"].(string)
	replay := f.send(t, "Alice", ab, "message-one", "只给 Bob 的消息")
	if replay["messageId"] != messageID {
		t.Fatal("message replay duplicated")
	}
	path := "/api/v1/chat/conversations/" + ab + "/messages"
	f.request(t, "Alice", "POST", path, map[string]any{"clientMessageId": "message-one", "content": "changed"}, 409)
	f.request(t, "Carol", "GET", path, nil, 404)
	f.request(t, "Carol", "POST", path, map[string]any{"clientMessageId": "unauthorized", "content": "forged"}, 404)
	f.request(t, "Alice", "POST", path, map[string]any{"clientMessageId": "forged-sender", "content": "x", "sender": map[string]string{"gameId": "Carol"}}, 400)
	f.request(t, "Alice", "POST", path, map[string]any{"clientMessageId": "cross-mention", "content": "hello", "mentionedPlayerRefs": []string{"player_Carol"}}, 404)
	reply := socialDataV2(f.request(t, "Bob", "POST", path, map[string]any{"clientMessageId": "reply-one", "content": "收到", "replyToMessageId": messageID}, 200))
	snapshot := reply["reply"].(map[string]any)
	if snapshot["content"] != "只给 Bob 的消息" || snapshot["sender"].(map[string]any)["gameId"] != "Alice" {
		t.Fatal("reply did not use authoritative source")
	}
	ac := f.conversation(t, "Alice", "Carol", "ac-first")
	forwardPath := "/api/v1/chat/conversations/" + ac + "/forwards"
	f.request(t, "Carol", "POST", forwardPath, map[string]any{"clientMessageId": "stolen-forward", "sourceMessageId": messageID, "sourceConversationId": ab}, 404)
	forwarded := socialDataV2(f.request(t, "Alice", "POST", forwardPath, map[string]any{"clientMessageId": "valid-forward", "sourceMessageId": messageID, "sourceConversationId": ab}, 200))
	if forwarded["content"] != "只给 Bob 的消息" || forwarded["forwarded"].(map[string]any)["sender"].(map[string]any)["gameId"] != "Alice" {
		t.Fatal("forward changed source or failed to record origin")
	}
	f.request(t, "Carol", "POST", "/api/v1/chat/conversations/"+ac+"/messages", map[string]any{"clientMessageId": "wrong-reply", "content": "x", "replyToMessageId": messageID}, 404)
	bnot := f.request(t, "Bob", "GET", "/api/v1/notifications", nil, 200)["data"].([]any)
	cnot := f.request(t, "Carol", "GET", "/api/v1/notifications", nil, 200)["data"].([]any)
	if len(bnot) != 1 || len(cnot) != 1 {
		t.Fatal("private notifications duplicated or leaked")
	}
	notificationID := bnot[0].(map[string]any)["notificationId"].(string)
	f.request(t, "Carol", "POST", "/api/v1/notifications/read", map[string]any{"clientRequestId": "stolen-read", "notificationIds": []string{notificationID}}, 404)
	read := socialDataV2(f.request(t, "Bob", "POST", "/api/v1/notifications/read", map[string]any{"clientRequestId": "read-own", "notificationIds": []string{notificationID}}, 200))
	if read["readCount"] != float64(1) {
		t.Fatal("own notification not marked")
	}
	f.request(t, "Bob", "POST", "/api/v1/chat/conversations/"+ab+"/read", map[string]any{"clientRequestId": "read-message", "lastReadMessageId": messageID}, 200)
	f.request(t, "Carol", "POST", "/api/v1/chat/conversations/"+ab+"/read", map[string]any{"clientRequestId": "wrong-conversation-read", "lastReadMessageId": messageID}, 404)
	var messages int
	if err := f.store.DB.QueryRow(`SELECT COUNT(*) FROM social_messages_v2`).Scan(&messages); err != nil {
		t.Fatal(err)
	}
	if messages != 3 {
		t.Fatalf("unexpected message side effects: %d", messages)
	}
}

func TestSocialV2ConcurrentRetryPaginationAndMonotonicRead(t *testing.T) {
	f := newSocialFixtureV2(t)
	ab := f.conversation(t, "Alice", "Bob", "ab")
	path := "/api/v1/chat/conversations/" + ab + "/messages"
	const concurrent = 12
	results := make(chan string, concurrent)
	failures := make(chan string, concurrent)
	var wait sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			status, body, err := f.call("Alice", "POST", path, map[string]any{"clientMessageId": "same-send", "content": "one durable message"})
			if err != nil || status != 200 {
				failures <- fmt.Sprintf("%d %v %v", status, body, err)
				return
			}
			results <- socialDataV2(body)["messageId"].(string)
		}()
	}
	wait.Wait()
	close(results)
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
	first := ""
	for result := range results {
		if first == "" {
			first = result
		} else if result != first {
			t.Fatal("concurrent retry generated different messages")
		}
	}
	messages := []string{first}
	for n := 0; n < 5; n++ {
		messages = append(messages, f.send(t, "Alice", ab, fmt.Sprintf("message-%d", n), fmt.Sprintf("message %d", n))["messageId"].(string))
	}
	page := f.request(t, "Bob", "GET", path+"?limit=2", nil, 200)
	if len(page["data"].([]any)) != 2 {
		t.Fatal("page size ignored")
	}
	cursor := page["page"].(map[string]any)["nextCursor"].(string)
	next := f.request(t, "Bob", "GET", path+"?limit=2&cursor="+cursor, nil, 200)
	seen := map[string]bool{}
	for _, raw := range append(page["data"].([]any), next["data"].([]any)...) {
		id := raw.(map[string]any)["messageId"].(string)
		if seen[id] {
			t.Fatal("message duplicated between pages")
		}
		seen[id] = true
	}
	f.request(t, "Alice", "GET", path+"?limit=2&cursor="+cursor, nil, 400)
	last := messages[len(messages)-1]
	f.request(t, "Bob", "POST", "/api/v1/chat/conversations/"+ab+"/read", map[string]any{"clientRequestId": "read-newest", "lastReadMessageId": last}, 200)
	read := socialDataV2(f.request(t, "Bob", "POST", "/api/v1/chat/conversations/"+ab+"/read", map[string]any{"clientRequestId": "read-older", "lastReadMessageId": first}, 200))
	if read["lastReadMessageId"] != last || read["unreadCount"] != float64(0) {
		t.Fatal("read cursor moved backwards")
	}
}

func TestSocialV2AnnouncementDraftPublishingPermissionsAndPreferences(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "announcements.manage"); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Grant(ctx, f.users["Carol"].ID, "core.read"); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"clientRequestId": "announcement-create", "title": "原公告", "summary": "发布前不可见", "contentBlocks": []map[string]any{{"blockId": "body", "type": "PARAGRAPH", "text": "真实公告内容"}}, "coverAssetId": nil, "pinned": false, "priority": "NORMAL"}
	f.request(t, "Carol", "POST", "/api/v1/admin/announcements", input, 403)
	f.request(t, "Bob", "GET", "/api/v1/admin/announcements", nil, 403)
	created := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/admin/announcements", input, 200))
	id := created["announcementId"].(string)
	if created["status"] != "DRAFT" {
		t.Fatal("new announcement is not a draft")
	}
	f.request(t, "Bob", "GET", "/api/v1/announcements/"+id, nil, 404)
	publishPath := "/api/v1/admin/announcements/" + id + "/publish"
	published := socialDataV2(f.request(t, "Alice", "POST", publishPath, map[string]any{"clientRequestId": "publish-one", "expectedVersion": 1}, 200))
	if published["version"] != float64(2) {
		t.Fatal("publish version incorrect")
	}
	f.request(t, "Alice", "POST", publishPath, map[string]any{"clientRequestId": "publish-one", "expectedVersion": 1}, 200)
	input["clientRequestId"] = "edit-one"
	input["expectedVersion"] = 2
	input["title"] = "新草稿"
	f.request(t, "Alice", "PUT", "/api/v1/admin/announcements/"+id, input, 200)
	visible := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/announcements/"+id, nil, 200))
	if visible["title"] != "原公告" {
		t.Fatal("draft edit leaked before publish")
	}
	f.request(t, "Alice", "POST", publishPath, map[string]any{"clientRequestId": "stale-publish", "expectedVersion": 2}, 409)
	f.request(t, "Alice", "POST", publishPath, map[string]any{"clientRequestId": "publish-two", "expectedVersion": 3}, 200)
	visible = socialDataV2(f.request(t, "Bob", "GET", "/api/v1/announcements/"+id, nil, 200))
	if visible["title"] != "新草稿" {
		t.Fatal("published changes not visible")
	}
	notices := f.request(t, "Bob", "GET", "/api/v1/notifications", nil, 200)["data"].([]any)
	if len(notices) != 2 {
		t.Fatalf("publication replay duplicated notifications: %d", len(notices))
	}
	f.request(t, "Alice", "POST", "/api/v1/admin/announcements/"+id+"/unpublish", map[string]any{"clientRequestId": "withdraw", "expectedVersion": 4, "reason": "需要修订"}, 200)
	f.request(t, "Bob", "GET", "/api/v1/announcements/"+id, nil, 404)
	preferences := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/notifications/preferences", nil, 200))
	if preferences["directMessages"] != true {
		t.Fatal("default preferences incorrect")
	}
	patch := map[string]any{"clientRequestId": "preferences-one", "expectedVersion": 1, "directMessages": false, "showPreviews": false}
	f.request(t, "Bob", "PATCH", "/api/v1/notifications/preferences", patch, 200)
	f.request(t, "Bob", "PATCH", "/api/v1/notifications/preferences", patch, 200)
	stored := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/notifications/preferences", nil, 200))
	if stored["directMessages"] != false || stored["version"] != float64(2) {
		t.Fatal("preferences not persisted idempotently")
	}
	untouched := socialDataV2(f.request(t, "Carol", "GET", "/api/v1/notifications/preferences", nil, 200))
	if untouched["directMessages"] != true {
		t.Fatal("preferences leaked to another user")
	}
}

func TestSocialV2AssetsStayBoundAcrossPublishedDraftsAndAvatarOwnership(t *testing.T) {
	for key, value := range map[string]string{"DEUTERIUM_S3_ENDPOINT": "https://storage.example.invalid", "DEUTERIUM_S3_REGION": "test", "DEUTERIUM_S3_BUCKET": "social-test", "DEUTERIUM_S3_PREFIX": "deuterium-test/", "DEUTERIUM_S3_ACCESS_KEY": "test-only-key", "DEUTERIUM_S3_SECRET_KEY": "test-only-secret"} {
		t.Setenv(key, value)
	}
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	readyAsset := func(user, purpose string) string {
		t.Helper()
		id := store.ID("asset_")
		business := "ANNOUNCEMENT"
		if purpose == "AVATAR" {
			business = "PROFILE"
		}
		u, err := f.store.CreateAssetUploadV2(ctx, store.AssetUploadV2{UploadID: store.ID("upload_"), AssetID: id, UserID: f.users[user].ID, ClientRequestID: id, Fingerprint: store.Digest([]byte(id)), Purpose: purpose, BusinessType: business, ObjectKey: "deuterium-test/" + id + ".png", ContentType: "image/png", SizeBytes: 100, MD5: "AAAAAAAAAAAAAAAAAAAAAA=="})
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := f.store.BeginAssetVerificationV2(ctx, u); err != nil || !ok {
			t.Fatalf("asset verification fixture: %v", err)
		}
		if err = f.store.FinishAssetVerificationV2(ctx, u, objectstorage.VerifiedImage{Width: 16, Height: 16, SHA256: strings.Repeat("a", 64)}, nil); err != nil {
			t.Fatal(err)
		}
		return id
	}
	avatar := readyAsset("Alice", "AVATAR")
	otherAvatar := readyAsset("Bob", "AVATAR")
	f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", map[string]any{"clientRequestId": "foreign-avatar", "expectedVersion": 1, "avatarAssetId": otherAvatar}, 409)
	profile := socialDataV2(f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", map[string]any{"clientRequestId": "own-avatar", "expectedVersion": 1, "avatarAssetId": avatar}, 200))
	if profile["avatar"].(map[string]any)["assetId"] != avatar {
		t.Fatal("avatar not attached")
	}
	public := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/players/player_Alice", nil, 200))
	if !strings.Contains(public["avatar"].(map[string]any)["url"].(string), "X-Amz-Signature") {
		t.Fatal("public profile did not resolve bound asset download")
	}
	if err := f.store.RemoveAssetV2(ctx, f.users["Alice"].ID, avatar); !errors.Is(err, store.ErrAssetInUse) {
		t.Fatalf("bound avatar removed: %v", err)
	}
	f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", map[string]any{"clientRequestId": "clear-avatar", "expectedVersion": 2, "avatarAssetId": nil}, 200)
	if err := f.store.RemoveAssetV2(ctx, f.users["Alice"].ID, avatar); err != nil {
		t.Fatal("cleared avatar remained bound", err)
	}
	for _, user := range []string{"Alice", "Bob"} {
		if err := f.store.Grant(ctx, f.users[user].ID, "announcements.manage"); err != nil {
			t.Fatal(err)
		}
	}
	cover := readyAsset("Alice", "ANNOUNCEMENT_MEDIA")
	input := map[string]any{"clientRequestId": "asset-ann-create", "title": "有封面的公告", "summary": "封面绑定测试", "contentBlocks": []map[string]any{{"blockId": "body", "type": "PARAGRAPH", "text": "正文"}}, "coverAssetId": cover, "pinned": false, "priority": "NORMAL"}
	ann := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/admin/announcements", input, 200))
	id := ann["announcementId"].(string)
	f.request(t, "Alice", "POST", "/api/v1/admin/announcements/"+id+"/publish", map[string]any{"clientRequestId": "asset-ann-publish", "expectedVersion": 1}, 200)
	input["clientRequestId"] = "other-editor"
	input["expectedVersion"] = 2
	input["title"] = "另一管理员编辑"
	f.request(t, "Bob", "PUT", "/api/v1/admin/announcements/"+id, input, 200)
	input["clientRequestId"] = "drop-cover-draft"
	input["expectedVersion"] = 3
	input["coverAssetId"] = nil
	f.request(t, "Bob", "PUT", "/api/v1/admin/announcements/"+id, input, 200)
	if err := f.store.RemoveAssetV2(ctx, f.users["Alice"].ID, cover); !errors.Is(err, store.ErrAssetInUse) {
		t.Fatalf("old published cover was unbound by saving a draft: %v", err)
	}
	f.request(t, "Bob", "POST", "/api/v1/admin/announcements/"+id+"/publish", map[string]any{"clientRequestId": "publish-without-cover", "expectedVersion": 4}, 200)
	if err := f.store.RemoveAssetV2(ctx, f.users["Alice"].ID, cover); err != nil {
		t.Fatal("replaced published cover remained bound", err)
	}
}

func TestSocialV2PublicMentionsAndFollowsUseCommittedMessagesOnce(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if _, err := f.store.FollowV2(ctx, f.users["Bob"], f.users["Alice"].PlayerRef, true); err != nil {
		t.Fatal(err)
	}
	messageID, _, err := f.store.PublishAppChat(ctx, f.users["Alice"], "public-source", "@Bob @Carol @Alice 一起看看", nil)
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		if err = f.store.PublicSocialNotificationsV2(ctx, "app:"+f.users["Alice"].ID, "public-source"); err != nil {
			t.Fatal(err)
		}
	}
	for _, user := range []string{"Bob", "Carol"} {
		items := f.request(t, user, "GET", "/api/v1/notifications", nil, 200)["data"].([]any)
		if len(items) != 1 {
			t.Fatalf("%s received duplicated public notification", user)
		}
		n := items[0].(map[string]any)
		if n["topic"] != "MENTIONS" || n["target"].(map[string]any)["referenceId"] != messageID {
			t.Fatal("mention notification does not reference authoritative public message")
		}
	}
	if items := f.request(t, "Alice", "GET", "/api/v1/notifications", nil, 200)["data"].([]any); len(items) != 0 {
		t.Fatal("self mention generated notification")
	}
	ab := f.conversation(t, "Bob", "Alice", "public-forward-target")
	forwarded := socialDataV2(f.request(t, "Bob", "POST", "/api/v1/chat/conversations/"+ab+"/forwards", map[string]any{"clientMessageId": "public-forward", "sourceMessageId": messageID}, 200))
	if forwarded["content"] != "@Bob @Carol @Alice 一起看看" {
		t.Fatal("public forward did not copy committed source")
	}
}

func TestSocialV2PublicReconciliationRecoversMissingHookWithoutHistoricalFollows(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if _, _, err := f.store.PublishAppChat(ctx, f.users["Alice"], "before-follow", "在关注之前发布", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.FollowV2(ctx, f.users["Bob"], f.users["Alice"].PlayerRef, true); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.DB.Exec(`UPDATE social_follows_v2 SET created_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 SECOND)`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.PublishAppChat(ctx, f.users["Alice"], "missing-hook", "@Carol 这条需要补扫", nil); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ReconcilePublicSocialNotificationsV2(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ReconcilePublicSocialNotificationsV2(ctx); err != nil {
		t.Fatal(err)
	}
	if items := f.request(t, "Bob", "GET", "/api/v1/notifications", nil, 200)["data"].([]any); len(items) != 0 {
		t.Fatal("new follow backfilled historic messages")
	}
	if items := f.request(t, "Carol", "GET", "/api/v1/notifications", nil, 200)["data"].([]any); len(items) != 1 {
		t.Fatal("missed hook was not recovered exactly once")
	}
	var cursor, last int64
	if err := f.store.DB.QueryRow(`SELECT last_sequence FROM social_reconcile_v2 WHERE worker_name='public_chat'`).Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	if err := f.store.DB.QueryRow(`SELECT MAX(sequence_id) FROM chat_messages_next`).Scan(&last); err != nil {
		t.Fatal(err)
	}
	if cursor != last {
		t.Fatal("durable public reconciliation cursor did not advance")
	}
}
