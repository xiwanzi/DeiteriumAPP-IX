package store

import "testing"

func TestOfficialStoreNotificationsOnlyContainUsefulMilestonesV206(t *testing.T) {
	d := CommerceRecordV2{ID: "order_1", Kind: "ORDER", Channel: "OFFICIAL_STORE", OwnerID: "buyer", State: "PAYMENT_PROCESSING", FundsState: "PROCESSING", PendingOperationID: "op_1"}
	for _, kind := range []string{"order.payment.started", "operation.COMPLETED", "operation.UNKNOWN", "mailbox.observed"} {
		if title, _, _, _ := commerceNoticeV206(d, kind, "buyer", "buyer"); title != "" {
			t.Fatal("intermediate step notified", kind)
		}
	}
	d.State = "AWAITING_CLAIM"
	d.FundsState = "HELD"
	d.PendingOperationID = ""
	title, _, _, push := commerceNoticeV206(d, "operation.COMPLETED", "buyer", "buyer")
	if title != "商品已送达" || push {
		t.Fatal("mail delivery should be a quiet inbox milestone")
	}
	d.State = "CLAIMED"
	d.FundsState = "SETTLED"
	if title, _, _, _ := commerceNoticeV206(d, "operation.COMPLETED", "buyer", "buyer"); title != "" {
		t.Fatal("claim duplicated user's action")
	}
	d.State = "REFUNDED"
	d.FundsState = "REFUNDED"
	title, _, _, push = commerceNoticeV206(d, "operation.COMPLETED", "buyer", "buyer")
	if title != "信用点已退回" || push {
		t.Fatal("refund should stay in inbox")
	}
}
func TestMarketAndCommissionNotifyThePersonWhoMustActV206(t *testing.T) {
	d := CommerceRecordV2{Kind: "ORDER", Channel: "PLAYER_MARKET", OwnerID: "buyer", PayeeID: "seller", State: "AWAITING_SHIPMENT", FundsState: "HELD"}
	if title, _, _, push := commerceNoticeV206(d, "operation.COMPLETED", "buyer", "seller"); title != "收到新订单" || !push {
		t.Fatal("seller needs to deliver")
	}
	if title, _, _, _ := commerceNoticeV206(d, "operation.COMPLETED", "buyer", "buyer"); title != "" {
		t.Fatal("buyer should not get duplicate payment notice")
	}
	d.Kind = "COMMISSION"
	d.State = "ACTIVE"
	if title, _, _, push := commerceNoticeV206(d, "operation.COMPLETED", "seller", "buyer"); title != "委托已被接取" || !push {
		t.Fatal("commission owner needs an update")
	}
}
