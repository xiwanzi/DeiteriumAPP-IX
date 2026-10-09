//go:build integration

package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func chatMediaFixture(t *testing.T, f *socialFixtureV2, user string, n int) store.ChatMediaUpload {
	t.Helper()
	in := store.ChatMediaInput{ClientRequestID: fmt.Sprintf("media-%d", n), SHA256: store.Digest([]byte(fmt.Sprint(n))), ThumbnailSHA256: store.Digest([]byte(fmt.Sprint("thumb", n))), MD5: base64.StdEncoding.EncodeToString(make([]byte, 16)), ThumbnailMD5: base64.StdEncoding.EncodeToString(make([]byte, 16)), SizeBytes: 100, ThumbnailBytes: 40}
	u, err := f.store.CreateChatMediaUpload(context.Background(), f.users[user].ID, in)
	if err != nil {
		t.Fatal(err)
	}
	u, err = f.store.CompleteChatMedia(context.Background(), f.users[user].ID, u.ID, objectstorage.ChatImage{VerifiedImage: objectstorage.VerifiedImage{Width: 64, Height: 64, SHA256: in.SHA256}, Frames: 1})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestChatMediaDedupPermissionsQuotaAndExpiry(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	a, b := f.users["Alice"], f.users["Bob"]
	u := chatMediaFixture(t, f, "Alice", 0)
	if _, _, err := f.store.ChatMediaAccess(ctx, b.ID, u.Media.ID, ""); !errors.Is(err, store.ErrAssetForbidden) {
		t.Fatalf("private upload readable: %v", err)
	}
	duplicate := chatMediaFixture(t, f, "Alice", 0)
	if duplicate.Media.ID != u.Media.ID {
		t.Fatal("retry duplicated object")
	}
	variant := store.ChatMediaInput{ClientRequestID: "different-preview", SHA256: u.Media.SHA256, ThumbnailSHA256: store.Digest([]byte("new-preview-encoder")), MD5: u.Media.MD5, ThumbnailMD5: u.Media.ThumbnailMD5, SizeBytes: u.Media.SizeBytes, ThumbnailBytes: 50}
	reused, e := f.store.CreateChatMediaUpload(ctx, a.ID, variant)
	if e != nil || reused.Media.ID != u.Media.ID || reused.Media.ThumbnailSHA256 != u.Media.ThumbnailSHA256 {
		t.Fatal("preview variation duplicated main bytes", e)
	}
	// A second owner cannot acquire an unpublished object by claiming its hash.
	in := store.ChatMediaInput{ClientRequestID: "second-owner", SHA256: u.Media.SHA256, ThumbnailSHA256: u.Media.ThumbnailSHA256, MD5: u.Media.MD5, ThumbnailMD5: u.Media.ThumbnailMD5, SizeBytes: 100, ThumbnailBytes: 40}
	other, err := f.store.CreateChatMediaUpload(ctx, b.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if other.Media.ID == u.Media.ID {
		t.Fatal("hash used as private access capability")
	}
	other, err = f.store.CompleteChatMedia(ctx, b.ID, other.ID, objectstorage.ChatImage{VerifiedImage: objectstorage.VerifiedImage{Width: 64, Height: 64, SHA256: u.Media.SHA256}, Frames: 1})
	if err != nil {
		t.Fatal(err)
	}
	if other.Media.ID != u.Media.ID {
		t.Fatal("verified bytes not deduplicated")
	}
	first, _, err := f.store.PublishAppChatV2(ctx, a, "media-first", "[图片]", "", nil, nil, &store.ChatMediaSend{ID: u.Media.ID, Type: "image"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := f.store.PublishAppChatV2(ctx, a, "media-second", "[图片]", "", nil, nil, &store.ChatMediaSend{ID: u.Media.ID, Type: "image"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.DB.Exec(`UPDATE public_chat_media SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE message_id=?`, first); err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.store.ChatMediaAccess(ctx, b.ID, u.Media.ID, first); !errors.Is(err, store.ErrMediaExpired) {
		t.Fatalf("old message inherited new deadline: %v", err)
	}
	if _, _, err = f.store.ChatMediaAccess(ctx, b.ID, u.Media.ID, second); err != nil {
		t.Fatal(err)
	}
	m, err := f.store.PublicChatMessageV203(ctx, second)
	if err != nil || m.Media == nil || m.Content != "[图片]" {
		t.Fatal("media metadata or game fallback missing", err)
	}
	if _, err = f.store.ExistingAppChatV2(ctx, a.ID, "media-second", "[贴纸]", "", nil, &store.ChatMediaSend{ID: u.Media.ID, Type: "sticker"}); !errors.Is(err, store.ErrConflict) {
		t.Fatal("changed media payload did not conflict", err)
	}
	media := []store.ChatMedia{u.Media}
	for n := 1; n < 11; n++ {
		media = append(media, chatMediaFixture(t, f, "Alice", n).Media)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	success, full := 0, 0
	for _, m := range media {
		wg.Add(1)
		go func(m store.ChatMedia) {
			defer wg.Done()
			_, e := f.store.SaveChatSticker(ctx, a.ID, m.ID)
			mu.Lock()
			defer mu.Unlock()
			if e == nil {
				success++
			} else if errors.Is(e, store.ErrStickerLimit) {
				full++
			} else {
				t.Error(e)
			}
		}(m)
	}
	wg.Wait()
	if success != 10 || full != 1 {
		t.Fatalf("quota race: success=%d full=%d", success, full)
	}
	items, err := f.store.ChatStickers(ctx, a.ID)
	if err != nil || len(items) != 10 {
		t.Fatal("library size", len(items), err)
	}
	if _, err = f.store.SaveChatSticker(ctx, a.ID, items[0].ID); err != nil {
		t.Fatal("duplicate at capacity", err)
	}
	if err = f.store.DeleteChatSticker(ctx, a.ID, items[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.SaveChatSticker(ctx, a.ID, items[0].ID); err != nil {
		t.Fatal("slot not released", err)
	}
}

type chatDeletedObjects struct{ keys []string }

func (d *chatDeletedObjects) Delete(_ context.Context, key string) error {
	d.keys = append(d.keys, key)
	return nil
}
func TestChatMediaGCAndPermanentHistory(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	a := f.users["Alice"]
	u := chatMediaFixture(t, f, "Alice", 0)
	message, _, err := f.store.PublishAppChatV2(ctx, a, "gc-message", "[图片]", "", nil, nil, &store.ChatMediaSend{ID: u.Media.ID, Type: "image"})
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`UPDATE chat_media_objects SET created_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 4 DAY)`, `UPDATE chat_media_uploads SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND)`} {
		if _, err = f.store.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	deleted := &chatDeletedObjects{}
	if n, e := f.store.RunChatMediaGC(ctx, deleted, "test/", 10); e != nil || n != 0 {
		t.Fatal("live message lost", n, e)
	}
	if _, err = f.store.SaveChatSticker(ctx, a.ID, u.Media.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.DB.Exec(`UPDATE public_chat_media SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND)`); err != nil {
		t.Fatal(err)
	}
	if n, e := f.store.RunChatMediaGC(ctx, deleted, "test/", 10); e != nil || n != 0 {
		t.Fatal("sticker lost", n, e)
	}
	if err = f.store.DeleteChatSticker(ctx, a.ID, u.Media.ID); err != nil {
		t.Fatal(err)
	}
	if n, e := f.store.RunChatMediaGC(ctx, deleted, "test/", 10); e != nil || n != 1 || len(deleted.keys) != 2 {
		t.Fatal("object/preview not purged", n, e)
	}
	if _, err = f.store.PublicChatMessageV203(ctx, message); err != nil {
		t.Fatal("message deleted by media GC", err)
	}
	if _, err = f.store.DB.Exec(`UPDATE chat_messages_next SET sent_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 31 DAY)`); err != nil {
		t.Fatal(err)
	}
	rows, err := f.store.Messages(ctx, 0, false, 100)
	if err != nil || len(rows) != 0 {
		t.Fatal("history window not enforced", err)
	}
	var count int
	if err = f.store.DB.QueryRow(`SELECT COUNT(*) FROM chat_messages_next WHERE message_id=?`, message).Scan(&count); err != nil || count != 1 {
		t.Fatal("durable history lost", err)
	}
	conversation := f.conversation(t, "Alice", "Bob", "old-direct-history")
	direct := f.send(t, "Alice", conversation, "old-direct-send", "保留在数据库的文字")
	if _, err = f.store.DB.Exec(`UPDATE social_messages_v2 SET sent_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 31 DAY)`); err != nil {
		t.Fatal(err)
	}
	view := f.request(t, "Alice", "GET", "/api/v1/chat/conversations/"+conversation+"/messages", nil, 200)["data"].([]any)
	if len(view) != 0 {
		t.Fatal("direct history window not enforced")
	}
	if err = f.store.DB.QueryRow(`SELECT COUNT(*) FROM social_messages_v2 WHERE message_id=?`, direct["messageId"]).Scan(&count); err != nil || count != 1 {
		t.Fatal("direct text not retained", err)
	}
}

func TestChatMediaHTTPAuthorizationAndSignedDeadline(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	u := chatMediaFixture(t, f, "Alice", 0)
	f.request(t, "", "GET", "/api/v1/chat/stickers", nil, 401)
	f.request(t, "Bob", "GET", "/api/v1/chat/media/files/"+u.Media.ID+"/access", nil, 404)
	f.request(t, "Bob", "POST", "/api/v1/chat/stickers", map[string]any{"mediaId": u.Media.ID}, 404)
	r, _ := http.NewRequest("GET", f.http.URL+"/api/v1/chat/stickers", nil)
	r.Header.Set("Authorization", "Bearer "+f.tokens["Bob"])
	r.Header.Set("X-Deuterium-Actor", f.users["Alice"].PlayerRef)
	response, err := f.http.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 409 {
		t.Fatal("changed browser identity accepted", response.StatusCode)
	}
	for key, value := range map[string]string{"DEUTERIUM_S3_ENDPOINT": "https://fixtures.rains3.com", "DEUTERIUM_S3_REGION": "fixture", "DEUTERIUM_S3_BUCKET": "chat-fixture", "DEUTERIUM_S3_PREFIX": "qa-chat/", "DEUTERIUM_S3_ACCESS_KEY": "fixture-access", "DEUTERIUM_S3_SECRET_KEY": "fixture-secret"} {
		t.Setenv(key, value)
	}
	id, _, err := f.store.PublishAppChatV2(ctx, f.users["Alice"], "deadline-message", "[图片]", "", nil, nil, &store.ChatMediaSend{ID: u.Media.ID, Type: "image"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.DB.Exec(`UPDATE public_chat_media SET expires_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 90 SECOND) WHERE message_id=?`, id); err != nil {
		t.Fatal(err)
	}
	view := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/chat/media/files/"+u.Media.ID+"/access?messageId="+id, nil, 200))
	link, err := url.Parse(view["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	seconds, err := strconv.Atoi(link.Query().Get("X-Amz-Expires"))
	if err != nil || seconds < 1 || seconds > 90 {
		t.Fatal("signed URL outlives message", seconds, err)
	}
	if _, err = f.store.DB.Exec(`UPDATE public_chat_media SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE message_id=?`, id); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Bob", "GET", "/api/v1/chat/media/files/"+u.Media.ID+"/access?messageId="+id, nil, 410)
}

type failingChatDelete struct {
	fail bool
	keys []string
}

func (d *failingChatDelete) Delete(_ context.Context, key string) error {
	d.keys = append(d.keys, key)
	if d.fail {
		d.fail = false
		return errors.New("fixture transient delete failure")
	}
	return nil
}
func TestChatMediaGCFailureCannotBindOrDeleteReplacement(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	u := chatMediaFixture(t, f, "Alice", 0)
	for _, q := range []string{`UPDATE chat_media_objects SET created_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 4 DAY)`, `UPDATE chat_media_uploads SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND)`} {
		if _, err := f.store.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	objects := &failingChatDelete{fail: true}
	if _, err := f.store.RunChatMediaGC(ctx, objects, "test/", 10); err == nil {
		t.Fatal("delete failure not propagated")
	}
	if _, err := f.store.SaveChatSticker(ctx, f.users["Alice"].ID, u.Media.ID); !errors.Is(err, store.ErrMediaExpired) {
		t.Fatal("deleting object rebound", err)
	}
	in := store.ChatMediaInput{ClientRequestID: "replacement", SHA256: u.Media.SHA256, ThumbnailSHA256: u.Media.ThumbnailSHA256, MD5: u.Media.MD5, ThumbnailMD5: u.Media.ThumbnailMD5, SizeBytes: u.Media.SizeBytes, ThumbnailBytes: u.Media.ThumbnailBytes}
	next, err := f.store.CreateChatMediaUpload(ctx, f.users["Alice"].ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if next.Media.ID == u.Media.ID {
		t.Fatal("replacement reused deleting key")
	}
	next, err = f.store.CompleteChatMedia(ctx, f.users["Alice"].ID, next.ID, objectstorage.ChatImage{VerifiedImage: objectstorage.VerifiedImage{Width: 64, Height: 64, SHA256: in.SHA256}, Frames: 1})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := f.store.RunChatMediaGC(ctx, objects, "test/", 10); err != nil || n != 1 {
		t.Fatal("cleanup retry failed", n, err)
	}
	for _, key := range objects.keys {
		if key == store.ChatMediaKey("test/", next.Media.ID, false) || key == store.ChatMediaKey("test/", next.Media.ID, true) {
			t.Fatal("replacement was deleted")
		}
	}
	if _, _, err = f.store.ChatMediaAccess(ctx, f.users["Alice"].ID, next.Media.ID, ""); err != nil {
		t.Fatal(err)
	}
}
