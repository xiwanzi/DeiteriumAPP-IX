//go:build integration

package imagereview

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestImageReviewUploadAndMessageAttributionAreSeparate(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	for n, name := range []string{"Uploader", "Sender", "Collector"} {
		_, e := db.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES(?,?,?,?,?,'unused','active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)`, name, "player_"+name, fmt.Sprintf("11111111-1111-1111-1111-%012d", n), name, fmt.Sprintf("100%d", n), strings.Repeat("0", 64))
		if e != nil {
			t.Fatal(e)
		}
	}
	must := func(q string, args ...any) {
		t.Helper()
		if _, e := db.DB.Exec(q, args...); e != nil {
			t.Fatal(e)
		}
	}
	must(`INSERT INTO chat_media_objects(media_id,sha256,thumbnail_sha256,content_md5,thumbnail_md5,size_bytes,thumbnail_bytes,state,created_at) VALUES('media_one',?,?,'md5','md5',100,20,'READY',UTC_TIMESTAMP(6))`, strings.Repeat("a", 64), strings.Repeat("b", 64))
	must(`INSERT INTO chat_media_uploads VALUES('upload_one','Uploader','request',?,'media_one','media_one',UTC_TIMESTAMP(6),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 72 HOUR))`, strings.Repeat("c", 64))
	must(`INSERT INTO chat_stickers VALUES('Collector','media_one',UTC_TIMESTAMP(6))`)
	must(`INSERT INTO chat_messages_next(message_id,source_id,client_message_id,fingerprint,sender_uuid,sender_name,sender_ref,registered,content,kind,sent_at) VALUES('message_one','app:Sender','send',?,'11111111-1111-1111-1111-000000000001','Sender','player_Sender',TRUE,'[图片]','IMAGE',UTC_TIMESTAMP(6))`, strings.Repeat("d", 64))
	must(`INSERT INTO public_chat_media VALUES('message_one','media_one',DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 72 HOUR))`)
	source := &Source{DB: db.DB, Prefix: "deuterium-test/"}
	if e := source.Detect(ctx); e != nil {
		t.Fatal(e)
	}
	rows, e := source.Batch(ctx, "CHAT", "", 100)
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
	if rows[0].GameID != "Uploader" || !rows[0].Verified || rows[0].SourceKey != "deuterium-test/chat-media/media_one/image.webp" {
		t.Fatal(rows[0])
	}
	source.StartAt = time.Now().UTC().Add(time.Hour)
	skipped, skipErr := source.Batch(ctx, "CHAT", "", 100)
	if skipErr != nil || len(skipped) != 0 {
		t.Fatal("old uploads not excluded", skipped, skipErr)
	}
	source.StartAt = time.Time{}
	events, e := source.Events(ctx, "media_one")
	if e != nil || len(events) != 1 {
		t.Fatal(events, e)
	}
	if events[0].GameID != "Sender" || events[0].Channel != "world" {
		t.Fatal("upload/collection incorrectly treated as message sender", events)
	}
	audit := testdb.New(t)
	repo := &Repository{DB: audit.DB}
	if e = repo.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	if e = repo.Discover(ctx, rows[0]); e != nil {
		t.Fatal(e)
	}
	item, _ := repo.Item(ctx, rows[0].ID)
	if e = repo.Scored(ctx, item, Result{Scores: Scores{Promotion: 75}, Summary: "联系信息", Reasons: []string{"测试"}}, "evidence", `"etag"`); e != nil {
		t.Fatal(e)
	}
	if e = repo.SaveEvents(ctx, item.ID, events); e != nil {
		t.Fatal(e)
	}
	must(`DELETE FROM public_chat_media WHERE message_id='message_one'`)
	saved, _, e := repo.Events(ctx, item.ID)
	if e != nil || len(saved) != 1 || saved[0].GameID != "Sender" {
		t.Fatal("event snapshot lost with original reference", saved, e)
	}
	repo.Finish(ctx, item.ID, "FLAGGED")
	repo.BeginDelete(ctx, item.ID)
	repo.Deleted(ctx, item.ID)
	if e = repo.SaveEvents(ctx, item.ID, events); e != nil {
		t.Fatal(e)
	}
	saved, _, e = repo.Events(ctx, item.ID)
	if e != nil || len(saved) != 0 {
		t.Fatal("late event sync resurrected deleted provenance", saved, e)
	}
}

type fakeReviewObjects struct {
	copied, deleted      int
	failCopy, failDelete bool
	inspectErr           error
}

func (f *fakeReviewObjects) Inspect(context.Context, Candidate) (ObjectInfo, error) {
	return ObjectInfo{ETag: `"etag"`, Size: 100, ContentType: "image/png"}, f.inspectErr
}
func (f *fakeReviewObjects) URL(context.Context, string) (string, error) {
	return "https://objects.example/signed", nil
}
func (f *fakeReviewObjects) EvidenceKey(id string) string { return "audit/" + id }
func (f *fakeReviewObjects) Retain(context.Context, string, string, string, int64) error {
	if f.failCopy {
		return errors.New("temporary")
	}
	f.copied++
	return nil
}
func (f *fakeReviewObjects) DeleteEvidence(context.Context, string) error {
	if f.failDelete {
		return errors.New("temporary")
	}
	f.deleted++
	return nil
}

type fakeReviewer struct {
	calls, score int
	err          error
}

func (f *fakeReviewer) Review(context.Context, string) (Result, error) {
	f.calls++
	return Result{Scores: Scores{Promotion: f.score}, Summary: "审核结果", Reasons: []string{"测试依据"}}, f.err
}
func TestImageReviewDurableScoringRetentionDeletionAndRetry(t *testing.T) {
	ctx := context.Background()
	source := testdb.New(t)
	audit := testdb.New(t)
	repo := &Repository{DB: audit.DB}
	if e := repo.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	objects := &fakeReviewObjects{}
	model := &fakeReviewer{score: 50}
	service := &Service{Repo: repo, Source: &Source{DB: source.DB}, Objects: objects, Reviewer: model}
	makeItem := func(id string) Item {
		c := Candidate{ID: ReviewID("ASSET", id), SourceKind: "ASSET", SourceID: id, SourceKey: "uploads/" + id, SizeBytes: 100, UploaderID: "uploader", GameID: "Sender", DisplayName: "发送者", Purpose: "AVATAR", UploadedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour), SourceState: "READY", SHA256: strings.Repeat("b", 64), Verified: true}
		if e := repo.Discover(ctx, c); e != nil {
			t.Fatal(e)
		}
		i, e := repo.Item(ctx, c.ID)
		if e != nil {
			t.Fatal(e)
		}
		return i
	}
	low := makeItem("low")
	if e := service.Process(ctx, low); e != nil {
		t.Fatal(e)
	}
	low, _ = repo.Item(ctx, low.ID)
	if low.State != "CLEAR" || objects.copied != 0 {
		t.Fatal("50 should not retain", low.State)
	}
	high := makeItem("high")
	high.SHA256 = strings.Repeat("c", 64)
	if e := repo.Discover(ctx, high.Candidate); e != nil {
		t.Fatal(e)
	}
	model.score = 51
	objects.failCopy = true
	if e := service.Process(ctx, high); e == nil {
		t.Fatal("copy failure hidden")
	}
	high, _ = repo.Item(ctx, high.ID)
	if high.State != "RETAINING" || *high.Score != 51 {
		t.Fatal("result not durable", high.State)
	}
	calls := model.calls
	objects.failCopy = false
	if e := service.Process(ctx, high); e != nil {
		t.Fatal(e)
	}
	high, _ = repo.Item(ctx, high.ID)
	if high.State != "FLAGGED" || model.calls != calls {
		t.Fatal("retention retried paid model")
	}
	rows, more, e := repo.List(ctx, 0)
	if e != nil || more || len(rows) != 1 || rows[0].ID != high.ID {
		t.Fatal(rows, more, e)
	}
	item, e := repo.BeginDelete(ctx, high.ID)
	if e != nil {
		t.Fatal(e)
	}
	objects.failDelete = true
	if e = service.Cleanup(ctx, item); e == nil {
		t.Fatal("deletion failure hidden")
	}
	item, _ = repo.Item(ctx, high.ID)
	if item.State != "DELETING" {
		t.Fatal(item.State)
	}
	objects.failDelete = false
	if e = service.Cleanup(ctx, item); e != nil {
		t.Fatal(e)
	}
	if e = repo.Discover(ctx, high.Candidate); e != nil {
		t.Fatal(e)
	}
	deleted, _ := repo.Item(ctx, high.ID)
	if deleted.State != "DELETED" || deleted.GameID != "" || deleted.Result != nil || deleted.SourceKey != "" {
		t.Fatal("deleted record rebuilt or PII retained", deleted)
	}
	retry := makeItem("retry")
	retry.Verified = false
	repo.Discover(ctx, retry.Candidate)
	model.err = &ReviewError{Code: "MODEL_RESPONSE_INVALID"}
	if e = service.Process(ctx, retry); e == nil {
		t.Fatal("provider failure accepted")
	}
	if e = repo.Retry(ctx, retry, model.err); e != nil {
		t.Fatal(e)
	}
	retry, _ = repo.Item(ctx, retry.ID)
	if retry.State != "RETRY" || retry.Score != nil {
		t.Fatal("failure treated as safe")
	}
	moving := makeItem("moving")
	moving.Animated = true
	repo.Discover(ctx, moving.Candidate)
	calls = model.calls
	if e = service.Process(ctx, moving); e != nil {
		t.Fatal(e)
	}
	moving, _ = repo.Item(ctx, moving.ID)
	if moving.State != "SKIPPED" || model.calls != calls {
		t.Fatal("animated sent to model")
	}
}
