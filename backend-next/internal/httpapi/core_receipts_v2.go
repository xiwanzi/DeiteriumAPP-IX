package httpapi

import (
	"context"
	"encoding/json"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// The same proof checks apply to a live command reply and a receipt recovered
// after a disconnect. Transport completion alone is not a financial receipt.
func (s *Server) acceptCoreReceiptV2(ctx context.Context, node config.Node, op store.CoreOperation, status string, data json.RawMessage) error {
	if status != "COMPLETED" {
		return nil
	}
	if op.Command == "wallet.transfer" {
		var expected struct {
			FromUUID string `json:"fromUuid"`
			ToUUID   string `json:"toUuid"`
			Amount   string `json:"amount"`
		}
		var proof struct {
			OperationID string          `json:"operationId"`
			PayerUUID   json.RawMessage `json:"payerUuid"`
			PayeeUUID   json.RawMessage `json:"payeeUuid"`
			FromUUID    json.RawMessage `json:"fromUuid"`
			ToUUID      json.RawMessage `json:"toUuid"`
			Amount      string          `json:"amount"`
			Currency    string          `json:"currency"`
			Status      string          `json:"status"`
			CommittedAt time.Time       `json:"committedAt"`
		}
		if bridge.Decode(op.Payload, &expected) != nil || json.Unmarshal(data, &proof) != nil || proof.OperationID != op.ID || !transferProofPartyV2(expected.FromUUID, proof.PayerUUID, proof.FromUUID) || !transferProofPartyV2(expected.ToUUID, proof.PayeeUUID, proof.ToUUID) || proof.Amount != expected.Amount || proof.Currency != "CREDIT" || proof.Status != "COMPLETED" || proof.CommittedAt.IsZero() {
			return bridge.ErrProtocol
		}
		return nil
	}
	if op.Command != "mailbox.create" && op.Command != "mailbox.query" && op.Command != "mailbox.revoke" {
		return nil
	}
	var result struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Value   json.RawMessage `json:"value"`
	}
	if bridge.Decode(data, &result) != nil {
		return bridge.ErrProtocol
	}
	if len(result.Value) == 0 || string(result.Value) == "null" {
		return nil
	}
	if node.MailCluster == "" {
		return bridge.ErrProtocol
	}
	var receipt *store.MailReceipt
	var payload map[string]json.RawMessage
	_ = json.Unmarshal(op.Payload, &payload)
	if op.Command == "mailbox.revoke" && payload["snapshotJson"] != nil {
		var err error
		receipt, err = validateMailCancellationProofV2(s.Config, node, op, result.Value)
		if err != nil {
			return err
		}
	} else {
		receipt = &store.MailReceipt{}
		if bridge.Decode(result.Value, receipt) != nil || bridge.ValidateReceipt(s.Config, *receipt) != nil {
			return bridge.ErrProtocol
		}
	}
	if receipt != nil {
		return s.Store.RememberMailReceipt(ctx, node.MailCluster, *receipt)
	}
	return nil
}

// XConomy transfer receipts use payerUuid/payeeUuid. Older receipts may use
// fromUuid/toUuid; every field supplied must identify the original participant.
func transferProofPartyV2(expected string, current, legacy json.RawMessage) bool {
	if expected == "" {
		return false
	}
	found := false
	for _, raw := range []json.RawMessage{current, legacy} {
		if len(raw) == 0 {
			continue
		}
		var actual string
		if json.Unmarshal(raw, &actual) != nil || actual != expected {
			return false
		}
		found = true
	}
	return found
}
