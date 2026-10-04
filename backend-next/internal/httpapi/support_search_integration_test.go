//go:build integration

package httpapi

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestSupportSearchUnicodeAndLiteralMatching(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	if _, err := f.store.DB.ExecContext(ctx, "UPDATE identities SET email='player@example.com' WHERE id=?", f.users["Carol"].ID); err != nil {
		t.Fatal(err)
	}
	inbox := supportTestInbox(t, f)
	result, err := f.store.IngestSupportMail(ctx, inbox, 11, 1, supportRawMail("player@example.com", "player@example.com", "search@example.com", "订单领取 HelloWorld A_B 100%", "请帮忙核查。", ""), time.Now(), f.app.Config.SMTPKey)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		query string
		count int
	}{
		{"订单", 1}, {"不存在的编号", 0}, {"🎮", 0}, {"hELLOwORLD", 1},
		{"PLAYER@EXAMPLE", 1}, {result.TicketID[4:], 1}, {"cAROL", 1},
		{"100%", 1}, {"A_B", 1}, {"A%B", 0}, {"' OR 1=1 --", 0},
	} {
		t.Run(test.query, func(t *testing.T) {
			data := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/admin/support/tickets?q="+url.QueryEscape(test.query), nil, 200))
			items := data["items"].([]any)
			if len(items) != test.count {
				t.Fatalf("search %q returned %d tickets, want %d", test.query, len(items), test.count)
			}
		})
	}
	f.request(t, "Carol", "GET", "/api/v1/admin/support/tickets?q="+url.QueryEscape("订单"), nil, 403)
}
