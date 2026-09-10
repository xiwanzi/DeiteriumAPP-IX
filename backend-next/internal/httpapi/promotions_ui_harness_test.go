//go:build integration && uipreview

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Explicitly opted-in local UI harness. Business mutations go through real HTTP
// and MariaDB transactions; only the Core transport uses the isolated test double.
func TestPromotionUIHarnessV209(t *testing.T) {
	directory := os.Getenv("DEUTERIUM_UI_PREVIEW_DIR")
	if directory == "" {
		t.Skip("local UI harness not requested")
	}
	abs, e := filepath.Abs(directory)
	if e != nil || !strings.Contains(filepath.ToSlash(abs), "/.tools/") {
		t.Fatal("UI evidence must stay in an ignored .tools directory")
	}
	if e = os.MkdirAll(abs, 0700); e != nil {
		t.Fatal(e)
	}
	f := newCatalogFixture(t)
	ctx := context.Background()
	core := newCommerceTestCore()
	core.mailMode = "revoked"
	f.server.CommerceCore = core
	f.server.Config.PublicOrigin = "http://127.0.0.1:5199"
	f.server.Config.Nodes = []config.Node{{ID: "amiya", ClaimEnabled: true, InventoryDomain: "survival", MailCluster: "local-preview"}, {ID: "odyssey", ClaimEnabled: true, InventoryDomain: "survival", MailCluster: "local-preview"}}
	p, content := catalogProductFixture(t, f)
	content["title"] = "EOS 探索补给包"
	content["subtitle"] = "为下一次出发，备好所需。"
	content["price"] = "120.00"
	content["discountRate"] = 8500
	content["deliveryCredits"] = 20
	content["purchaseLimits"] = map[string]any{"daily": 5, "weekly": 15, "weeklyDay": 1, "weeklyTime": "04:00"}
	p = promotionPublishTestV209(t, f, p, content)
	shop, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.StoreID, "store", true)
	if e != nil {
		t.Fatal(e)
	}
	shop.Body["name"] = "EOS Lab旗舰店"
	shop.Body["intro"] = "为你的每一次探索，提供可靠补给。"
	if _, e = f.s.CatalogEditV2(ctx, f.admin.ID, shop.ID, "store", "shop-name", shop.Version, shop.Body, false); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 137; i++ {
		ref := fmt.Sprintf("catalog:gear_%03d", i)
		name := fmt.Sprintf("补给物资 %03d", i)
		if i == 136 {
			name = "TACZ 精密步枪"
		}
		metadata := map[string]any{"itemRef": ref, "revision": 1, "displayName": name, "description": "不可变物品快照", "maxQuantity": 9999, "codec": "bukkit-bytes-v1", "payloadSha256": strings.Repeat("a", 64), "inventoryDomain": "survival", "compatibleServerIds": []string{"amiya", "odyssey"}, "itemId": "minecraft:stone"}
		body, _ := json.Marshal(metadata)
		if _, e = f.s.DB.Exec("INSERT INTO core_catalog_heads VALUES(?,1,1,FALSE,?)", ref, strings.Repeat("b", 64)); e != nil {
			t.Fatal(e)
		}
		if _, e = f.s.DB.Exec("INSERT INTO item_versions VALUES(?,1,'amiya',?,?,?,UTC_TIMESTAMP(6))", ref, strings.Repeat("a", 64), string(body), strings.Repeat("b", 64)); e != nil {
			t.Fatal(e)
		}
	}
	for i := 0; i < 105; i++ {
		name := fmt.Sprintf("探索物资模板 %03d", i)
		if i == 104 {
			name = "EOS 联名装备交付"
		}
		_, e = f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, p.StoreID, "", fmt.Sprintf("template-%d", i), 0, store.CatalogObjectV2{"name": name, "summary": "探索服实物补给 · 原样发放", "inventoryDomain": "survival", "allowedServerIds": []any{"amiya", "odyssey"}, "attachments": []any{map[string]any{"itemRef": "catalog:gear_136", "revision": 1, "quantity": 1, "payloadSha256": strings.Repeat("a", 64)}}, "active": true}, nil)
		if e != nil {
			t.Fatal(e)
		}
	}
	for i, name := range []string{"开业礼遇", "探索者单品礼", "周末专享"} {
		c := promotionCouponTestV209()
		c["name"] = name
		c["amountOff"] = "20.00"
		c["minimumSpend"] = "100.00"
		c["storeIds"] = []any{p.StoreID}
		c["endsAt"] = time.Now().Add(time.Duration(i+3) * 24 * time.Hour).UTC().Format(time.RFC3339)
		if i == 1 {
			c["type"] = "ITEM"
			c["benefit"] = "PERCENT"
			c["discountRate"] = 7000
			c["minimumSpend"] = "0.00"
			c["maxDiscount"] = "50.00"
		}
		promotionSaveTestV209(t, f, c)
	}
	if _, e = f.s.CatalogCartChangeV2(ctx, f.admin.ID, p.ID, "bag", 1, 2, false); e != nil {
		t.Fatal(e)
	}
	token, csrf := strings.Repeat("L", 43), "local-ui-preview-csrf-v209"
	if e = f.s.CreateSession(ctx, f.admin, store.Digest([]byte(token)), "web", csrf, time.Now().Add(time.Hour), ""); e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(f.server.Handler())
	defer server.Close()
	info := map[string]any{"origin": server.URL, "storeId": p.StoreID, "productId": p.ID, "cookieName": f.server.cookieName(), "token": token}
	body, _ := json.Marshal(info)
	if e = os.WriteFile(filepath.Join(abs, "ready.json"), body, 0600); e != nil {
		t.Fatal(e)
	}
	t.Log("isolated UI preview ready")
	deadline := time.Now().Add(55 * time.Minute)
	for time.Now().Before(deadline) {
		if _, e := os.Stat(filepath.Join(abs, "stop")); e == nil {
			return
		}
		time.Sleep(time.Second)
	}
}
