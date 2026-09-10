package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type commerceCoreAdapter struct{ server *Server }

func (a *commerceCoreAdapter) creditDeliveryReady(node config.Node) bool {
	var status struct {
		Mailbox struct {
			APIVersion    int  `json:"apiVersion"`
			CreditRewards bool `json:"creditRewards"`
		} `json:"mailbox"`
	}
	return a.nodeReady(node, "mailbox.create") && json.Unmarshal(a.server.Core.Status(node.ID), &status) == nil && status.Mailbox.APIVersion >= 3 && status.Mailbox.CreditRewards
}

func commerceCommandAllowed(command string) bool {
	switch command {
	case CommerceReserveV2, CommerceBindV2, CommerceSettleV2, CommerceRefundV2, "mailbox.create", "mailbox.revoke":
		return true
	}
	return false
}

func (a *commerceCoreAdapter) nodeReady(node config.Node, command string) bool {
	if !a.server.Core.Online(node.ID) {
		return false
	}
	var status struct {
		StorageHealthy   bool `json:"storageHealthy"`
		Economy          bool `json:"economy"`
		EconomyAuthority bool `json:"economyAuthority"`
		Mailbox          struct {
			Available                   bool   `json:"available"`
			CommerceReady               bool   `json:"commerceReady"`
			ClusterID                   string `json:"clusterId"`
			APIVersion                  int    `json:"apiVersion"`
			MissingDeliveryCancellation bool   `json:"missingDeliveryCancellation"`
		} `json:"mailbox"`
	}
	if json.Unmarshal(a.server.Core.Status(node.ID), &status) != nil || !status.StorageHealthy {
		return false
	}
	if strings.HasPrefix(command, "wallet.") {
		return node.Economy && status.Economy && status.EconomyAuthority
	}
	if command == "mailbox.revoke" && (status.Mailbox.APIVersion < 2 || !status.Mailbox.MissingDeliveryCancellation) {
		return false
	}
	return node.MailCluster != "" && status.Mailbox.Available && status.Mailbox.CommerceReady && status.Mailbox.ClusterID == node.MailCluster
}

func (a *commerceCoreAdapter) Available(command string) bool {
	if !commerceCommandAllowed(command) {
		return false
	}
	for _, node := range a.server.Config.Nodes {
		if a.nodeReady(node, command) {
			return true
		}
	}
	return false
}

func (a *commerceCoreAdapter) selectNode(ctx context.Context, command string, payload map[string]any) (string, error) {
	cluster, domain := "", ""
	requiresCredits := false
	if command == "mailbox.create" {
		var snapshot struct {
			CreditAmount int64 `json:"creditAmount"`
		}
		raw, _ := payload["snapshotJson"].(string)
		if json.Unmarshal([]byte(raw), &snapshot) != nil {
			return "", bridge.ErrProtocol
		}
		requiresCredits = snapshot.CreditAmount > 0
	}
	allowed := map[string]bool{}
	if command == "mailbox.create" {
		domain, _ = payload["inventoryDomain"].(string)
		raw, err := json.Marshal(payload["allowedServerIds"])
		var ids []string
		if err != nil || json.Unmarshal(raw, &ids) != nil || len(ids) == 0 || domain == "" {
			return "", bridge.ErrProtocol
		}
		for _, id := range ids {
			node, err := a.server.node(id)
			if err != nil || !node.ClaimEnabled || node.InventoryDomain != domain || node.MailCluster == "" || (cluster != "" && cluster != node.MailCluster) {
				return "", bridge.ErrProtocol
			}
			cluster = node.MailCluster
			allowed[id] = true
		}
	} else if command == "mailbox.revoke" {
		// A missing delivery may be closed using the original frozen snapshot.
		// Existing deliveries remain tied to the cluster that acknowledged them.
		err := a.server.Store.DB.QueryRowContext(ctx, "SELECT mail_cluster FROM core_mail_receipts WHERE delivery_id=? AND order_id=? AND snapshot_sha256=?", payload["deliveryId"], payload["orderId"], payload["expectedSnapshotSha256"]).Scan(&cluster)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		if _, present := payload["snapshotJson"]; present {
			raw, marshalError := json.Marshal(payload)
			if marshalError != nil {
				return "", bridge.ErrProtocol
			}
			_, snapshot, parseError := parseMailCancellationV2(raw)
			if parseError != nil {
				return "", parseError
			}
			domain = snapshot.InventoryDomain
			for _, id := range snapshot.AllowedServerIDs {
				node, lookupError := a.server.node(id)
				if lookupError != nil || node.InventoryDomain != domain || node.MailCluster == "" || (cluster != "" && cluster != node.MailCluster) {
					return "", bridge.ErrProtocol
				}
				cluster = node.MailCluster
				allowed[id] = true
			}
		} else if err != nil {
			return "", err
		}
	}
	for _, node := range a.server.Config.Nodes {
		if cluster != "" && node.MailCluster != cluster {
			continue
		}
		if len(allowed) > 0 && !allowed[node.ID] {
			continue
		}
		if requiresCredits && !a.creditDeliveryReady(node) {
			continue
		}
		if a.nodeReady(node, command) {
			return node.ID, nil
		}
	}
	return "", bridge.ErrNodeOffline
}

func (a *commerceCoreAdapter) Execute(ctx context.Context, actor, client, command string, payload map[string]any) (CommerceCoreResultV2, error) {
	if !commerceCommandAllowed(command) || actor == "" || !bridge.ValidMessageID(client) {
		return CommerceCoreResultV2{}, bridge.ErrProtocol
	}
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) > 28000 {
		return CommerceCoreResultV2{}, bridge.ErrProtocol
	}
	var id, node, fingerprint string
	err = a.server.Store.DB.QueryRowContext(ctx, "SELECT operation_id,node_id,fingerprint FROM core_operations WHERE actor_id=? AND client_request_id=?", actor, client).Scan(&id, &node, &fingerprint)
	if err == nil {
		if fingerprint != store.Digest(append([]byte(node+":"+command+":"), raw...)) {
			return CommerceCoreResultV2{}, store.ErrConflict
		}
		return a.Query(ctx, actor, id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CommerceCoreResultV2{}, err
	}
	node, err = a.selectNode(ctx, command, payload)
	if err != nil {
		return CommerceCoreResultV2{}, err
	}
	op, err := a.server.CoreCall(ctx, actor, client, node, command, payload)
	return commerceCoreResult(op), err
}

func (a *commerceCoreAdapter) Query(ctx context.Context, actor, id string) (CommerceCoreResultV2, error) {
	// Check ownership before asking a game node to reconcile a receipt.
	op, err := a.server.Store.CoreOperation(ctx, id)
	if err != nil {
		return CommerceCoreResultV2{}, err
	}
	if op.ActorID != actor || !commerceCommandAllowed(op.Command) {
		return CommerceCoreResultV2{}, ErrForbidden
	}
	op, err = a.server.QueryCoreOperation(ctx, id)
	return commerceCoreResult(op), err
}

func commerceCoreResult(op store.CoreOperation) CommerceCoreResultV2 {
	result := CommerceCoreResultV2{OperationID: op.ID, Status: op.State}
	if op.ID == "" {
		return result
	}
	if op.State != "COMPLETED" && op.State != "FAILED" {
		return result
	}
	var reply struct {
		Data  map[string]any `json:"data"`
		Error *CoreError     `json:"error"`
	}
	if json.Unmarshal(op.Result, &reply) != nil {
		result.Status, result.ErrorCode = "UNKNOWN", "INVALID_CORE_RECEIPT"
		return result
	}
	result.Data = reply.Data
	if op.State == "FAILED" {
		if reply.Error != nil {
			result.ErrorCode = reply.Error.Code
		} else {
			result.ErrorCode = "CORE_COMMAND_FAILED"
		}
		return result
	}
	if reply.Data == nil {
		result.Status, result.ErrorCode = "UNKNOWN", "INVALID_CORE_RECEIPT"
		return result
	}
	if strings.HasPrefix(op.Command, "mailbox.") {
		code, _ := reply.Data["code"].(string)
		value, ok := reply.Data["value"].(map[string]any)
		status, _ := value["status"].(string)
		switch code {
		case "OK", "ALREADY_REVOKED":
			if op.Command == "mailbox.revoke" && value["proofKind"] != nil {
				kind, _ := value["proofKind"].(string)
				cancelledAt, _ := value["cancelledAt"].(float64)
				valid := value["operationId"] == op.ID && cancelledAt > 0
				if kind == "CANCELLED_BEFORE_CREATE" {
					valid = valid && value["mailReceipt"] == nil
				} else if kind == "REVOKED_MAIL" {
					receipt, exists := value["mailReceipt"].(map[string]any)
					valid = valid && exists && receipt["status"] == "REVOKED"
				} else {
					valid = false
				}
				if !valid {
					result.Status, result.ErrorCode = "UNKNOWN", "INVALID_CANCELLATION_PROOF"
				}
				break
			}
			if !ok || (op.Command == "mailbox.revoke" && status != "REVOKED") || (op.Command == "mailbox.create" && status != "CREATED" && status != "CLAIMING" && status != "CLAIMED") {
				result.Status, result.ErrorCode = "UNKNOWN", "INVALID_MAIL_RECEIPT"
			}
		case "UNKNOWN", "CLAIM_IN_PROGRESS", "STORAGE_UNAVAILABLE", "":
			result.Status, result.ErrorCode = "UNKNOWN", code
		default:
			result.Status, result.ErrorCode = "FAILED", code
		}
	}
	return result
}
