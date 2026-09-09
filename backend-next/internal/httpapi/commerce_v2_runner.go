package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// RunCommerceOperationV2 never creates an intent. It advances only the stable
// steps of a previously authorized, committed business operation.
func (s *Server) RunCommerceOperationV2(ctx context.Context, id string, execute bool) (store.CommerceOperationV2, error) {
	op, e := s.Store.CommerceOperationV2(ctx, id)
	if e != nil {
		return op, e
	}
	for n := 0; n < 4; n++ {
		if op.State == "COMPLETED" || op.State == "FAILED" || op.StepIndex >= len(op.Steps) {
			return op, nil
		}
		step := op.Steps[op.StepIndex]
		if step.Command == "mailbox.create" || step.Command == "mailbox.revoke" {
			if e = s.RefreshCommerceMailboxV2(ctx, op.ResourceID); e != nil {
				return op, e
			}
			op, e = s.Store.CommerceOperationV2(ctx, id)
			if e != nil {
				return op, e
			}
			if op.State == "COMPLETED" || op.State == "FAILED" || op.StepIndex >= len(op.Steps) {
				return op, nil
			}
			step = op.Steps[op.StepIndex]
		}
		if s.CommerceCore == nil || (!execute && step.CoreOperationID == "") {
			if execute {
				_ = s.Store.DeferCommerceOperationV2(ctx, id, time.Now().UTC().Add(15*time.Second))
			}
			return op, nil
		}
		if step.CoreOperationID == "" && !s.CommerceCore.Available(step.Command) {
			_ = s.Store.DeferCommerceOperationV2(ctx, id, time.Now().UTC().Add(15*time.Second))
			return op, nil
		}
		claimed, _, ok, e := s.Store.ClaimCommerceStepV2(ctx, id, time.Now().UTC())
		if e != nil {
			return op, e
		}
		op = claimed
		if !ok {
			return op, nil
		}
		index := op.StepIndex
		step = op.Steps[index]
		callCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		var response CommerceCoreResultV2
		if step.CoreOperationID != "" {
			response, e = s.CommerceCore.Query(callCtx, op.ActorID, step.CoreOperationID)
		} else if execute {
			response, e = s.CommerceCore.Execute(callCtx, op.ActorID, step.ClientKey, step.Command, step.Payload)
		}
		cancel()
		if e != nil && response.Status == "" {
			response.Status = "UNKNOWN"
			response.ErrorCode = "RESULT_UNKNOWN"
		}
		if response.Status == "" {
			response.Status = "UNKNOWN"
			response.ErrorCode = "CORE_RESULT_INVALID"
		}
		saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		updated, saveError := s.Store.ApplyCommerceStepV2(saveCtx, id, index, op.DispatchToken, store.CommerceStepResultV2{CoreOperationID: response.OperationID, Status: response.Status, ErrorCode: response.ErrorCode, Data: response.Data}, time.Now().UTC())
		saveCancel()
		if saveError != nil {
			return op, saveError
		}
		op = updated
		if !execute || op.State != "PROCESSING" || op.StepIndex == index {
			return op, nil
		}
	}
	return op, nil
}

// Mailbox events are a separate durable source. An old RPC UNKNOWN is not
// rewritten into success; a matching later receipt proves the business fact.
func (s *Server) RefreshCommerceMailboxV2(ctx context.Context, id string) error {
	d, e := s.Store.CommerceRecordV2(ctx, id)
	if e != nil {
		return e
	}
	if d.Kind != "ORDER" || d.Channel != "OFFICIAL_STORE" {
		return nil
	}
	plan, ok := store.CommerceContentV2(d.Body, "mailboxPlan")
	if !ok {
		return nil
	}
	receipt, e := s.Store.MailReceipt(ctx, store.CommerceStringV2(plan, "deliveryId"))
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	pending, e := s.Store.ObserveCommerceMailboxV2(ctx, id, receipt, time.Now().UTC())
	if e != nil || pending == "" {
		return e
	}
	op, e := s.Store.CommerceOperationV2(ctx, pending)
	if e != nil {
		return e
	}
	if op.State == "COMPLETED" || op.State == "FAILED" || op.StepIndex >= len(op.Steps) {
		return nil
	}
	step := op.Steps[op.StepIndex]
	if step.Command != "mailbox.create" && step.Command != "mailbox.revoke" {
		return nil
	}
	code := "OK"
	if step.Command == "mailbox.revoke" {
		switch receipt.Status {
		case "REVOKED":
			code = "ALREADY_REVOKED"
		case "CLAIMED":
			code = "ALREADY_CLAIMED"
		default:
			return nil
		}
	} else {
		switch receipt.Status {
		case "CREATED", "CLAIMING", "CLAIMED":
		case "REVOKED":
			code = "ALREADY_REVOKED"
		default:
			return nil
		}
	}
	raw, _ := json.Marshal(receipt)
	var value map[string]any
	if e = json.Unmarshal(raw, &value); e != nil {
		return e
	}
	_, e = s.Store.ApplyCommerceStepV2(ctx, op.ID, op.StepIndex, "", store.CommerceStepResultV2{CoreOperationID: step.CoreOperationID, Status: "COMPLETED", Data: map[string]any{"code": code, "value": value}, FromMailboxEvent: true}, time.Now().UTC())
	return e
}

func (s *Server) StartCommerceV2() {
	go s.walletNoticeWorkerV206()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
			s.reconcileCommerceV2(ctx)
			cancel()
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *Server) reconcileCommerceV2(ctx context.Context) {
	official, e := s.Store.OfficialCommerceResourcesV2(ctx, 20)
	if e == nil {
		for _, d := range official {
			if ctx.Err() != nil {
				return
			}
			_ = s.RefreshCommerceMailboxV2(ctx, d.ID)
		}
	}
	// Record due work before slow/unavailable Core calls, so neither the mailbox
	// feed nor deadline handling is starved by a queue of unknown operations.
	if s.CommerceCore != nil && s.CommerceCore.Available(CommerceSettleV2) {
		now := time.Now().UTC()
		due, e := s.Store.DueCommerceResourcesV2(ctx, now, 20)
		if e == nil {
			for _, d := range due {
				if ctx.Err() != nil {
					return
				}
				key := fmt.Sprintf("auto-confirm:%s:%d", d.ID, d.Version)
				_, _ = s.Store.PrepareCommerceSettlementV2(ctx, d.OwnerID, d.ID, d.Kind, key, d.Version, true, true, now)
			}
		}
	}
	operations, e := s.Store.PendingCommerceOperationsV2(ctx, 20)
	if e == nil {
		for _, op := range operations {
			if ctx.Err() != nil {
				return
			}
			_, _ = s.RunCommerceOperationV2(ctx, op.ID, true)
		}
	}
}
