//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestProductMailPublicationAndPersistentOrderPlan(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	f.server.CommerceCore = newCommerceTestCore()
	p, content := catalogProductFixture(t, f)
	content["mailTitle"], content["mailBody"] = "你的物资已送达", "感谢支持！\n请在生存服领取。"
	if _, err := f.s.CatalogEditV2(ctx, f.buyer.ID, p.ID, "product", "foreign-mail-edit", p.Version, content, false); err == nil {
		t.Fatal("buyer edited merchant mail content")
	}
	p, err := f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", "mail-draft", p.Version, content, false)
	if err != nil {
		t.Fatal(err)
	}
	legacy := commerceOfficialTest(t, f, p, "before-mail-publish")
	legacyRecord := commerceRecordTest(t, f, legacy.ResourceID)
	legacyPlan := legacyRecord.Body["mailboxPlan"].(map[string]any)
	if legacyPlan["title"] == "你的物资已送达" {
		t.Fatal("unpublished mail leaked into an order")
	}
	p, err = f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.ID, "product", true)
	if err != nil {
		t.Fatal(err)
	}
	p, err = f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "mail-publish", p.Version, "publish", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	m := commerceOfficialTest(t, f, p, "custom-mail-order")
	before := commerceRecordTest(t, f, m.ResourceID)
	plan := before.Body["mailboxPlan"].(map[string]any)
	if plan["title"] != "你的物资已送达" || plan["body"] != "感谢支持！\n请在生存服领取。" {
		t.Fatal("custom content did not reach order plan", plan)
	}
	content["mailTitle"], content["mailBody"] = "下一版邮件", "修改后的正文"
	p, err = f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.ID, "product", true)
	if err != nil {
		t.Fatal(err)
	}
	p, err = f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", "next-mail-draft", p.Version, content, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "next-mail-publish", p.Version, "publish", "", 0); err != nil {
		t.Fatal(err)
	}
	replayed, err := f.s.PrepareOrderV2(ctx, f.buyer.ID, "custom-mail-order", before.QuoteID, 1, "OFFICIAL_STORE", false, nil)
	if err != nil || replayed.ResourceID != m.ResourceID || !replayed.Replayed {
		t.Fatal("same order was not recovered", replayed, err)
	}
	op, err := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if err != nil || op.State != "COMPLETED" {
		t.Fatal(op, err)
	}
	found := false
	for _, step := range op.Steps {
		if step.Command == "mailbox.create" {
			found = true
			if step.Payload["title"] != plan["title"] || step.Payload["body"] != plan["body"] || step.Payload["snapshotSha256"] != plan["snapshotSha256"] {
				t.Fatal("queued delivery changed after publication", step.Payload)
			}
		}
	}
	if !found {
		t.Fatal("mailbox delivery step missing")
	}
	after := commerceRecordTest(t, f, m.ResourceID)
	if fmt.Sprint(after.Body["mailboxPlan"]) != fmt.Sprint(before.Body["mailboxPlan"]) {
		t.Fatal("persisted order plan changed")
	}
}

func TestProductMailOversizeCombinedQuoteCreatesNoFinancialWork(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	content["mailBody"] = strings.Repeat("中", 1000)
	p, err := f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", "long-body", p.Version, content, false)
	if err != nil {
		t.Fatal(err)
	}
	p, err = f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "long-publish", p.Version, "publish", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.s.CatalogCreateV2(ctx, f.admin.ID, "product", p.StoreID, "second-long-product", content)
	if err != nil {
		t.Fatal(err)
	}
	second, err = f.s.CatalogActionV2(ctx, f.admin.ID, second.ID, "product", "second-long-publish", second.Version, "publish", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.CatalogQuoteV2(ctx, f.buyer.ID, "oversize-mail-quote", store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": []any{map[string]any{"productId": p.ID, "quantity": 1, "expectedProductVersion": p.PublishedVersion}, map[string]any{"productId": second.ID, "quantity": 1, "expectedProductVersion": second.PublishedVersion}}, "delivery": map[string]any{"method": "MAILBOX"}})
	assertCatalogCode(t, err, "MAIL_CONTENT_TOO_LONG")
	for _, table := range []string{"commerce_resources_v2", "commerce_operations_v2", "core_operations"} {
		if catalogTestCount(t, f.s, table) != 0 {
			t.Fatal("oversize mail started financial work", table)
		}
	}
}
