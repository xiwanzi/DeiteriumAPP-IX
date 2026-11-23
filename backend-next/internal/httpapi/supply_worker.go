package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func supplyMoney(minor int64) string { return fmt.Sprintf("%d.%02d", minor/100, minor%100) }

func (s *Server) supplyWorker() {
	defer close(s.supplyDone)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
		}
		ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
		exchanges, err := s.Store.PendingSupplyExchanges(ctx)
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("Supply exchange queue unavailable", "error", err)
			}
			cancel()
			continue
		}
		var wg sync.WaitGroup
		slots := make(chan struct{}, 4)
		for _, x := range exchanges {
			wg.Add(1)
			go func(x store.SupplyExchange) {
				defer wg.Done()
				if e := s.processSupplyExchange(ctx, x.ID, x.PlayerUUID); e != nil && ctx.Err() == nil {
					slog.Warn("Supply exchange awaiting recovery", "exchange", x.ID, "error", e)
				}
			}(x)
		}
		wg.Wait()
		if e := s.Store.MailExpiredSupplyReveals(ctx); e != nil && ctx.Err() == nil {
			slog.Warn("Supply reveal recovery pending", "error", e)
		}
		for _, state := range []string{"PENDING", "PREPARING"} {
			pending, e := s.Store.SupplyDeliveryQueue(ctx, state, 32)
			if e != nil {
				continue
			}
			for _, d := range pending {
				player, e := s.Core.Player("", d.PlayerUUID)
				if e == nil && !player.Online {
					if e = s.Store.RouteOfflineSupplyDelivery(ctx, d.ID); e != nil {
						slog.Warn("Supply offline routing pending", "delivery", d.ID, "error", e)
					}
				}
			}
		}
		mail, e := s.Store.SupplyDeliveryQueue(ctx, "MAIL_PENDING", 16)
		if e == nil {
			for _, d := range mail {
				slots <- struct{}{}
				wg.Add(1)
				go func(d store.SupplyDelivery) {
					defer wg.Done()
					defer func() { <-slots }()
					if e := s.processSupplyMail(ctx, d); e != nil && ctx.Err() == nil {
						slog.Warn("Supply mail awaiting recovery", "delivery", d.ID, "error", e)
					}
				}(d)
			}
			wg.Wait()
		}
		cancel()
	}
}

// The request and the recovery scan share one attempt for each committed exchange.
func (s *Server) processSupplyExchange(ctx context.Context, id, playerUUID string) error {
	done := make(chan struct{})
	if running, exists := s.supplyExchangeRuns.LoadOrStore(id, done); exists {
		select {
		case <-running.(chan struct{}):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	defer func() {
		s.supplyExchangeRuns.Delete(id)
		close(done)
	}()
	select {
	case s.supplyExchangeSlots <- struct{}{}:
		defer func() { <-s.supplyExchangeSlots }()
	case <-ctx.Done():
		return ctx.Err()
	}
	x, err := s.Store.SupplyExchange(ctx, id, playerUUID)
	if err != nil {
		return err
	}
	if x.Status == "PENDING" {
		if err = s.executeSupplyExchangeStep(ctx, x); err != nil {
			return err
		}
		x, err = s.Store.SupplyExchange(ctx, id, playerUUID)
		if err != nil {
			return err
		}
	}
	if x.Status == "RESERVED" {
		return s.executeSupplyExchangeStep(ctx, x)
	}
	return nil
}

func (s *Server) executeSupplyExchangeStep(ctx context.Context, x store.SupplyExchange) error {
	command := "wallet.escrow.reserve"
	key := "reserve_" + x.ID
	payload := map[string]any{"escrowRef": x.ID, "businessRef": x.ID, "amount": supplyMoney(x.TotalPrice), "currency": "CREDIT"}
	if x.Status == "PENDING" {
		payload["businessType"] = "OFFICIAL_STORE"
		payload["payerUuid"] = x.PlayerUUID
		payload["payeeUuid"] = nil
	} else if x.Status == "RESERVED" {
		command = "wallet.escrow.settle"
		key = "settle_" + x.ID
	} else {
		return nil
	}
	result, err := s.CommerceCore.Execute(ctx, x.PlayerUUID, key, command, payload)
	if result.OperationID != "" {
		if e := s.Store.SupplyExchangeProgress(ctx, x.ID, x.Status, result.OperationID, result.ErrorCode); e != nil {
			return e
		}
	}
	if err != nil {
		return err
	}
	if result.Status == "FAILED" {
		next := "FAILED"
		if x.Status == "RESERVED" {
			next = "REVIEW"
		}
		return s.Store.UpdateSupplyExchange(ctx, x.ID, x.Status, next, result.ErrorCode)
	}
	if result.Status != "COMPLETED" {
		return nil
	}
	if err = verifySupplyMoney(result, x); err != nil {
		return s.Store.UpdateSupplyExchange(ctx, x.ID, x.Status, "REVIEW", "经济回执与兑换订单不一致")
	}
	if x.Status == "PENDING" {
		return s.Store.UpdateSupplyExchange(ctx, x.ID, "PENDING", "RESERVED", "")
	}
	return s.Store.CompleteSupplyExchange(ctx, x)
}

func verifySupplyMoney(result CommerceCoreResultV2, x store.SupplyExchange) error {
	d := result.Data
	amount := supplyMoney(x.TotalPrice)
	if d == nil || d["operationId"] != result.OperationID || d["businessRef"] != x.ID || d["escrowRef"] != x.ID || d["payerUuid"] != x.PlayerUUID || d["currency"] != "CREDIT" || d["businessType"] != "OFFICIAL_STORE" || d["status"] != "COMPLETED" || d["amount"] != amount || d["reservedAmount"] != amount {
		return errors.New("invalid supply money receipt")
	}
	stamp, ok := d["committedAt"].(string)
	if !ok {
		return errors.New("missing economic commit")
	}
	if _, err := time.Parse(time.RFC3339Nano, stamp); err != nil {
		return err
	}
	if x.Status == "RESERVED" && (d["settledAmount"] != amount || d["heldAmount"] != "0.00" || d["refundedAmount"] != "0.00") {
		return errors.New("incomplete supply settlement")
	}
	if x.Status == "PENDING" && (d["heldAmount"] != amount || d["settledAmount"] != "0.00" || d["refundedAmount"] != "0.00") {
		return errors.New("incomplete supply reservation")
	}
	return nil
}

func supplyMailPayload(d store.SupplyDelivery) (map[string]any, error) {
	r := d.Reward
	snapshot := map[string]any{"schemaVersion": 1, "orderId": d.OrderID, "recipientUuid": d.PlayerUUID, "inventoryDomain": r.InventoryDomain, "allowedServerIds": r.Servers, "attachments": []map[string]any{{"itemRef": r.ItemRef, "revision": r.ItemRevision, "quantity": 1, "payloadSha256": r.PayloadSHA256}}}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return map[string]any{"source": "deuterium-commerce", "deliveryId": d.ID, "orderId": d.OrderID, "recipientUuid": d.PlayerUUID, "title": "武器补给奖励", "body": "本次抽取的奖励已送达。订单：" + d.OrderID, "sender": "武器补给", "snapshotJson": string(raw), "snapshotSha256": store.Digest(raw), "allowedServerIds": r.Servers, "inventoryDomain": r.InventoryDomain}, nil
}

func (s *Server) processSupplyMail(ctx context.Context, d store.SupplyDelivery) error {
	payload, err := supplyMailPayload(d)
	if err != nil {
		return err
	}
	result, err := s.CommerceCore.Execute(ctx, d.PlayerUUID, "mail_"+d.ID, "mailbox.create", payload)
	if err != nil {
		if mark := s.Store.SupplyDeliveryError(ctx, d.ID, "邮箱服务暂不可用，等待恢复"); mark != nil {
			return mark
		}
		return err
	}
	if result.Status != "COMPLETED" {
		return s.Store.SupplyDeliveryError(ctx, d.ID, result.ErrorCode)
	}
	value, ok := result.Data["value"].(map[string]any)
	if !ok || value["deliveryId"] != d.ID || value["orderId"] != d.OrderID || value["recipientUuid"] != d.PlayerUUID || value["snapshotSha256"] != payload["snapshotSha256"] {
		return s.Store.SupplyDeliveryError(ctx, d.ID, "邮箱回执与抽取订单不一致")
	}
	status, _ := value["status"].(string)
	if status != "CREATED" && status != "CLAIMING" && status != "CLAIMED" {
		return s.Store.SupplyDeliveryError(ctx, d.ID, "邮箱尚未确认交付")
	}
	return s.Store.CompleteSupplyMail(ctx, d.ID, value)
}
