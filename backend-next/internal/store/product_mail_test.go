package store

import (
	"strings"
	"testing"
)

func TestProductMailUsesPublishedTextAndKeepsLegacyDefault(t *testing.T) {
	products := map[string]CatalogRecordV2{"one": {Published: CatalogObjectV2{"title": "石材"}, Body: CatalogObjectV2{"mailTitle": "未发布"}}}
	quantities := map[string]int64{"one": 2}
	title, body, err := productMailText(products, quantities, "D123")
	if err != nil || title != "官方商城订单 D123" || body != defaultProductMailBody {
		t.Fatal(title, body, err)
	}
	products["one"].Published["mailTitle"] = "你的建造物资已送达"
	products["one"].Published["mailBody"] = "感谢支持！\n请在生存服领取。"
	title, body, err = productMailText(products, quantities, "D123")
	if err != nil || title != "你的建造物资已送达" || body != "感谢支持！\n请在生存服领取。" {
		t.Fatal(title, body, err)
	}
	products["one"].Published["mailTitle"] = "  "
	products["one"].Published["mailBody"] = "\n "
	title, body, err = productMailText(products, quantities, "D123")
	if err != nil || title != "官方商城订单 D123" || body != defaultProductMailBody {
		t.Fatal(title, body, err)
	}
}

func TestProductMailCombinesInStableOrderAndRejectsOversizeBeforePayment(t *testing.T) {
	products := map[string]CatalogRecordV2{
		"b": {Published: CatalogObjectV2{"title": "照明", "mailBody": "照明领取说明"}},
		"a": {Published: CatalogObjectV2{"title": "石材", "mailTitle": "石材送达", "mailBody": "石材领取说明"}},
	}
	quantities := map[string]int64{"b": 3, "a": 2}
	for i := 0; i < 20; i++ {
		title, body, err := productMailText(products, quantities, "D456")
		if err != nil || title != "官方商城订单 D456" || body != "石材 × 2\n石材送达\n石材领取说明\n\n照明 × 3\n照明领取说明" {
			t.Fatal(title, body, err)
		}
	}
	products["a"].Published["mailBody"] = strings.Repeat("中", 1000)
	products["b"].Published["mailBody"] = strings.Repeat("中", 1000)
	if _, _, err := productMailText(products, quantities, "D456"); err == nil {
		t.Fatal("accepted a combined body exceeding Core's 4096-byte limit")
	}
}

func TestProductMailValidatesUTF8BytesAndControls(t *testing.T) {
	for _, value := range []CatalogObjectV2{{}, {"mailTitle": "", "mailBody": ""}, {"mailTitle": strings.Repeat("中", 80), "mailBody": strings.Repeat("中", 1000)}, {"mailBody": "你好\n欢迎"}} {
		if err := validateProductMail(value); err != nil {
			t.Fatal(value, err)
		}
	}
	for _, value := range []CatalogObjectV2{{"mailTitle": nil}, {"mailTitle": 1}, {"mailTitle": strings.Repeat("🙂", 61)}, {"mailBody": strings.Repeat("🙂", 751)}, {"mailTitle": "a\nb"}, {"mailBody": "a\tb"}, {"mailBody": "a\u0085b"}, {"mailBody": strings.Repeat("x", 1001)}} {
		if validateProductMail(value) == nil {
			t.Fatalf("accepted invalid mail content: %#v", value)
		}
	}
}
