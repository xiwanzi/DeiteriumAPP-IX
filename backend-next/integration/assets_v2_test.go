//go:build integration

package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestAssetAuthorizationReplayIsolationAndBindingProtection(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	in := store.AssetUploadV2{UploadID: "upload_one", AssetID: "asset_one", UserID: user.ID, ClientRequestID: "same_request", Fingerprint: store.Digest([]byte("same_image")), Purpose: "AVATAR", BusinessType: "PROFILE", ObjectKey: "deuterium-test/uploads/one.png", ContentType: "image/png", MD5: "1B2M2Y8AsgTpgAmY7PhCfg==", SizeBytes: 100}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := s.CreateAssetUploadV2(ctx, in)
			if err != nil || u.AssetID != in.AssetID {
				t.Errorf("duplicate upload not replayed: %v", err)
			}
		}()
	}
	wg.Wait()
	if count(t, s, "asset_uploads_v2") != 1 {
		t.Fatal("duplicate uploads allocated objects")
	}
	broken := in
	broken.Fingerprint = store.Digest([]byte("different"))
	if _, err := s.CreateAssetUploadV2(ctx, broken); !errors.Is(err, store.ErrConflict) {
		t.Fatal("id reused for another image")
	}
	if _, err := s.AssetUploadV2(ctx, "another_user", in.UploadID); !errors.Is(err, store.ErrAssetForbidden) {
		t.Fatal("upload session leaked across users")
	}
	u, err := s.AssetUploadV2(ctx, user.ID, in.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.BeginAssetVerificationV2(ctx, u)
	if err != nil || !claimed {
		t.Fatal("verification not acquired", err)
	}
	if claimed, _ := s.BeginAssetVerificationV2(ctx, u); claimed {
		t.Fatal("concurrent verification acquired twice")
	}
	verified := objectstorage.VerifiedImage{Width: 4, Height: 5, SHA256: store.Digest([]byte("verified")), MD5: in.MD5}
	if err = s.FinishAssetVerificationV2(ctx, u, verified, nil); err != nil {
		t.Fatal(err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetAssetBindingsV2(ctx, tx, user.ID, "PROFILE", user.ID, []string{in.AssetID}); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = s.RemoveAssetV2(ctx, user.ID, in.AssetID); !errors.Is(err, store.ErrAssetInUse) {
		t.Fatal("bound asset removed", err)
	}
	tx, _ = s.DB.BeginTx(ctx, nil)
	if err = s.SetAssetBindingsV2(ctx, tx, user.ID, "PROFILE", user.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = s.RemoveAssetV2(ctx, user.ID, in.AssetID); err != nil {
			t.Fatal("remove not idempotent", err)
		}
	}
	if err = s.ValidateOwnedAssetsV2(ctx, user.ID, []string{in.AssetID}, "AVATAR"); !errors.Is(err, store.ErrAssetUnavailable) {
		t.Fatal("removed asset attached")
	}
}

func TestAssetVerificationRecoversAbandonedLeaseWithoutExpiredRenewal(t *testing.T) {
	s := testdb.New(t)
	user := imported(t, s)
	ctx := context.Background()
	u, err := s.CreateAssetUploadV2(ctx, store.AssetUploadV2{UploadID: "upload_lease", AssetID: "asset_lease", UserID: user.ID, ClientRequestID: "lease", Fingerprint: store.Digest([]byte("lease")), Purpose: "AVATAR", BusinessType: "PROFILE", ObjectKey: "deuterium-test/uploads/lease.png", ContentType: "image/png", MD5: "1B2M2Y8AsgTpgAmY7PhCfg==", SizeBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE asset_uploads_v2 SET status='VERIFYING',verification_started_at=? WHERE upload_id=?", time.Now().UTC().Add(-2*time.Minute), u.UploadID); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.BeginAssetVerificationV2(ctx, u); err != nil || !ok {
		t.Fatal("abandoned verification not recovered", err)
	}
	if err = s.FinishAssetVerificationV2(ctx, u, objectstorage.VerifiedImage{}, objectstorage.ErrUnavailable); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE asset_uploads_v2 SET expires_at=? WHERE upload_id=?", time.Now().UTC().Add(-time.Minute), u.UploadID); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.BeginAssetVerificationV2(ctx, u); err != nil || ok {
		t.Fatal("expired upload acquired another verification", err)
	}
}
