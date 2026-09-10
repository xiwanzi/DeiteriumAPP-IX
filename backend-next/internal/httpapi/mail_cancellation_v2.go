package httpapi

import (
	"encoding/json"
	"strings"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type mailCancellationRequestV2 struct {
	Source                 string `json:"source"`
	DeliveryID             string `json:"deliveryId"`
	OrderID                string `json:"orderId"`
	ExpectedSnapshotSHA256 string `json:"expectedSnapshotSha256"`
	ReasonCode             string `json:"reasonCode"`
	SnapshotJSON           string `json:"snapshotJson"`
}

type mailCancellationSnapshotV2 struct {
	SchemaVersion    int               `json:"schemaVersion"`
	OrderID          string            `json:"orderId"`
	RecipientUUID    string            `json:"recipientUuid"`
	InventoryDomain  string            `json:"inventoryDomain"`
	AllowedServerIDs []string          `json:"allowedServerIds"`
	Attachments      []json.RawMessage `json:"attachments"`
	TemplateRef      string            `json:"templateRef,omitempty"`
	TemplateRevision int64             `json:"templateRevision,omitempty"`
	CreditAmount     int64             `json:"creditAmount,omitempty"`
}

type mailCancellationProofV2 struct {
	OperationID      string             `json:"operationId"`
	Source           string             `json:"source"`
	DeliveryID       string             `json:"deliveryId"`
	OrderID          string             `json:"orderId"`
	SnapshotSHA256   string             `json:"snapshotSha256"`
	RecipientUUID    string             `json:"recipientUuid"`
	AllowedServerIDs []string           `json:"allowedServerIds"`
	InventoryDomain  string             `json:"inventoryDomain"`
	ProofKind        string             `json:"proofKind"`
	MailReceipt      *store.MailReceipt `json:"mailReceipt"`
	CancelledAt      int64              `json:"cancelledAt"`
	Replayed         bool               `json:"replayed"`
}

func parseMailCancellationV2(raw []byte) (mailCancellationRequestV2, mailCancellationSnapshotV2, error) {
	var request mailCancellationRequestV2
	var snapshot mailCancellationSnapshotV2
	if bridge.Decode(raw, &request) != nil || request.Source != "deuterium-commerce" || !bridge.ValidMessageID(request.DeliveryID) || !bridge.ValidMessageID(request.OrderID) || len(request.SnapshotJSON) > 24000 || strings.TrimSpace(request.ReasonCode) == "" || len(request.ReasonCode) > 192 || store.Digest([]byte(request.SnapshotJSON)) != request.ExpectedSnapshotSHA256 || bridge.Decode([]byte(request.SnapshotJSON), &snapshot) != nil {
		return request, snapshot, bridge.ErrProtocol
	}
	if snapshot.SchemaVersion != 1 || snapshot.OrderID != request.OrderID || !identity.ValidUUID(snapshot.RecipientUUID) || !config.NodeID.MatchString(snapshot.InventoryDomain) || snapshot.Attachments == nil || (len(snapshot.Attachments) == 0 && snapshot.CreditAmount == 0) || snapshot.CreditAmount < 0 || snapshot.CreditAmount > 1000000000000 || len(snapshot.Attachments) > 32 || !validMailCancellationScopeV2(snapshot.AllowedServerIDs) {
		return request, snapshot, bridge.ErrProtocol
	}
	return request, snapshot, nil
}

func validMailCancellationScopeV2(ids []string) bool {
	if len(ids) == 0 || len(ids) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !config.NodeID.MatchString(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func sameMailCancellationScopeV2(left, right []string) bool {
	if len(left) != len(right) || !validMailCancellationScopeV2(left) || !validMailCancellationScopeV2(right) {
		return false
	}
	seen := map[string]bool{}
	for _, id := range left {
		seen[id] = true
	}
	for _, id := range right {
		if !seen[id] {
			return false
		}
	}
	return true
}

// A cancellation before creation is a separate committed proof, never a fake
// mail receipt. Only an actual revoked mail may update core_mail_receipts.
func validateMailCancellationProofV2(c config.Config, node config.Node, op store.CoreOperation, raw json.RawMessage) (*store.MailReceipt, error) {
	request, snapshot, err := parseMailCancellationV2(op.Payload)
	if err != nil {
		return nil, err
	}
	var proof mailCancellationProofV2
	if bridge.Decode(raw, &proof) != nil || proof.OperationID != op.ID || proof.Source != request.Source || proof.DeliveryID != request.DeliveryID || proof.OrderID != request.OrderID || proof.SnapshotSHA256 != request.ExpectedSnapshotSHA256 || proof.RecipientUUID != snapshot.RecipientUUID || proof.InventoryDomain != snapshot.InventoryDomain || !sameMailCancellationScopeV2(proof.AllowedServerIDs, snapshot.AllowedServerIDs) || proof.CancelledAt <= 0 || node.MailCluster == "" {
		return nil, bridge.ErrProtocol
	}
	for _, id := range proof.AllowedServerIDs {
		matched := false
		for _, configured := range c.Nodes {
			if configured.ID == id && configured.MailCluster == node.MailCluster {
				matched = true
				break
			}
		}
		if !matched {
			return nil, bridge.ErrProtocol
		}
	}
	switch proof.ProofKind {
	case "CANCELLED_BEFORE_CREATE":
		if proof.MailReceipt != nil {
			return nil, bridge.ErrProtocol
		}
		return nil, nil
	case "REVOKED_MAIL":
		receipt := proof.MailReceipt
		if receipt == nil || receipt.Status != "REVOKED" || receipt.MailID == "" || bridge.ValidateReceipt(c, *receipt) != nil || receipt.DeliveryID != proof.DeliveryID || receipt.OrderID != proof.OrderID || receipt.Source != proof.Source || receipt.RecipientUUID != proof.RecipientUUID || receipt.SnapshotSHA256 != proof.SnapshotSHA256 || receipt.InventoryDomain != proof.InventoryDomain || !sameMailCancellationScopeV2(receipt.AllowedServerIDs, proof.AllowedServerIDs) {
			return nil, bridge.ErrProtocol
		}
		return receipt, nil
	default:
		return nil, bridge.ErrProtocol
	}
}
