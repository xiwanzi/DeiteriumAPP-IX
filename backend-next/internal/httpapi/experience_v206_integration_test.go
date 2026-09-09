//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestSakiConcurrentDifferentRequestsCannotCreateTwoPendingPurchasesV206(t *testing.T) {
	f, _, v := sakiFixtureV206(t)
	sakiSaveV206(t, f, v, 0, "open")
	ctx := context.Background()
	var wg sync.WaitGroup
	var accepted atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := f.db.PrepareAIPurchaseV206(ctx, f.users["Alice"].ID, store.AIPurchaseInputV206{ClientRequestID: fmt.Sprintf("different-%d", n), PlanID: "plan_pro", ExpectedPlanVersion: 2}, true, true)
			if err == nil {
				accepted.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("accepted %d different concurrent purchases", accepted.Load())
	}
}

func TestAppTransferNotificationOnlyAfterCommitAndOnlyForRecipientV206(t *testing.T) {
	f, _, _ := sakiFixtureV206(t)
	ctx := context.Background()
	sender := f.users["Alice"]
	receiver := f.users["Bob"]
	id, _, err := f.db.CreateWalletTransfer(ctx, sender, "transfer-once", "amiya", store.Recipient{PlayerRef: receiver.PlayerRef, UUID: receiver.ServerUUID}, "12.50", nil)
	if err != nil {
		t.Fatal(err)
	}
	transfer, err := f.db.WalletTransfer(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.db.MarkCoreSent(ctx, transfer.OperationID); err != nil {
		t.Fatal(err)
	}
	if err = f.db.CoreReply(ctx, "amiya", transfer.OperationID, "UNKNOWN", []byte(`{"status":"UNKNOWN"}`)); err != nil {
		t.Fatal(err)
	}
	notices, _ := f.db.NotificationsV2(ctx, receiver.ID, 0, false, 100)
	if len(notices) != 0 {
		t.Fatal("unconfirmed money notified")
	}
	reply := []byte(`{"status":"COMPLETED"}`)
	for i := 0; i < 3; i++ {
		if err = f.db.CoreReply(ctx, "amiya", transfer.OperationID, "COMPLETED", reply); err != nil {
			t.Fatal(err)
		}
	}
	notices, err = f.db.NotificationsV2(ctx, receiver.ID, 0, false, 100)
	if err != nil || len(notices) != 1 || !notices[0].SystemPush || !strings.Contains(notices[0].Body, "Alice 向你转账 12.50") {
		t.Fatal("missing/doubled receipt", notices, err)
	}
	notices, _ = f.db.NotificationsV2(ctx, sender.ID, 0, false, 100)
	if len(notices) != 0 {
		t.Fatal("receipt leaked to sender")
	}
}

func TestGameTransferNoticesPersistCursorDeduplicateAndIgnoreNonTransfersV206(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	app := New(f.store, config.Config{Development: true, PublicOrigin: "http://127.0.0.1", Nodes: []config.Node{{ID: "amiya", Economy: true}}})
	defer app.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := f.store.DB.Exec("UPDATE wallet_notice_cursor_v206 SET from_time=?", now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	sender := f.users["Alice"].ServerUUID
	app.Core.Connect("amiya", func(ctx context.Context, _ string, payload any) error {
		frame := payload.(map[string]any)
		if frame["command"] != "wallet.records" {
			t.Fatal("notification wrote money")
		}
		rows := []ledgerRowV204{{Sequence: "30", PlayerUUID: f.users["Bob"].ServerUUID, GameID: "Bob", OtherUUID: &sender, OperationID: "native-pay-once", BusinessRef: "native-pay-once", Source: "GAME", BusinessType: "TRANSFER", Direction: "income", Amount: "9.50", OccurredAt: now.Add(-20 * time.Second)}, {Sequence: "29", PlayerUUID: f.users["Bob"].ServerUUID, GameID: "Bob", Source: "GAME", BusinessType: "NATIVE_ADD", Direction: "income", Amount: "10.00", OccurredAt: now.Add(-30 * time.Second)}}
		reply, _ := json.Marshal(map[string]any{"operationId": frame["operationId"], "status": "COMPLETED", "data": ledgerPageV204{Records: rows, Snapshot: "30"}})
		return f.store.CoreReply(ctx, "amiya", frame["operationId"].(string), "COMPLETED", reply)
	})
	defer app.Core.Disconnect("amiya")
	for i := 0; i < 2; i++ {
		if err := app.walletNoticesV206(ctx, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	notices, err := f.store.NotificationsV2(ctx, f.users["Bob"].ID, 0, false, 100)
	if err != nil || len(notices) != 1 || notices[0].Target.ReferenceID != "econ_30" || !notices[0].SystemPush {
		t.Fatal("bad native receipt", notices, err)
	}
	var version int64
	f.store.DB.QueryRow("SELECT version FROM wallet_notice_cursor_v206").Scan(&version)
	if version != 3 {
		t.Fatal("cursor did not advance")
	}
}

func TestStoreCreationAvatarUploadIsAdminOnlyV206(t *testing.T) {
	f := newCatalogFixture(t)
	mux := http.NewServeMux()
	f.server.registerAssetsV2(mux)
	for _, test := range []struct {
		token   string
		allowed bool
	}{{f.adminToken, true}, {f.buyerToken, false}} {
		request := httptest.NewRequest("POST", "/api/v1/assets/uploads", strings.NewReader(`{"clientRequestId":"new-store-avatar","purpose":"STORE_MEDIA","businessType":"STORE","businessRef":"","fileName":"avatar.png","contentType":"image/png","sizeBytes":1024,"contentMd5":"AAAAAAAAAAAAAAAAAAAAAA==","altText":"商店头像"}`))
		request.Header.Set("Authorization", "Bearer "+test.token)
		request.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if test.allowed {
			if w.Code < 200 || w.Code >= 300 {
				t.Fatalf("admin avatar rejected: %d %s", w.Code, w.Body)
			}
		} else if w.Code != 403 {
			t.Fatalf("user avatar grant status %d", w.Code)
		}
	}
}
