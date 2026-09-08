package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) coreRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/admin/core/operations", s.createCoreOperation)
	mux.HandleFunc("GET /api/v1/admin/core/operations/{operationId}", s.getCoreOperation)
	mux.HandleFunc("POST /api/v1/admin/core/operations/{operationId}/retry", s.retryCoreOperation)
	mux.HandleFunc("POST /api/v1/admin/core/mail-deliveries", s.createCoreMail)
	mux.HandleFunc("GET /api/v1/admin/core/mail-deliveries/{deliveryId}", s.getCoreMail)
}
func (s *Server) node(id string) (config.Node, error) {
	for _, node := range s.Config.Nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return config.Node{}, bridge.ErrProtocol
}
func (s *Server) createCoreOperation(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "core.manage")
	if err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		ClientRequestID string `json:"clientRequestId"`
		ServerID        string `json:"serverId"`
		Command         string `json:"command"`
		GameID          string `json:"gameId"`
		PlayerRef       string `json:"playerRef"`
		DeliveryID      string `json:"deliveryId"`
		ReasonCode      string `json:"reasonCode"`
	}
	if err = body(w, r, &input); err != nil || !bridge.ValidMessageID(input.ClientRequestID) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	n, err := s.node(input.ServerID)
	if err != nil {
		failError(w, r, err)
		return
	}
	var payload any
	switch input.Command {
	case "player.resolve":
		payload = map[string]string{"gameId": input.GameID}
	case "wallet.balance":
		if !n.Economy {
			failError(w, r, ErrForbidden)
			return
		}
		uuid, e := s.Store.CoreRecipient(r.Context(), input.PlayerRef)
		if e != nil {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		payload = map[string]string{"playerUuid": uuid}
	case "mailbox.query":
		if n.MailCluster == "" || !bridge.ValidMessageID(input.DeliveryID) {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		payload = map[string]string{"source": "deuterium-commerce", "deliveryId": input.DeliveryID}
	case "mailbox.revoke":
		order, hash, e := s.Store.ManualDelivery(r.Context(), input.DeliveryID)
		if e != nil || n.MailCluster == "" || len(input.ReasonCode) == 0 || len(input.ReasonCode) > 128 {
			failError(w, r, ErrForbidden)
			return
		}
		payload = map[string]string{"source": "deuterium-commerce", "deliveryId": input.DeliveryID, "orderId": order, "expectedSnapshotSha256": hash, "reasonCode": input.ReasonCode}
	default:
		failError(w, r, ErrForbidden)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	op, err := s.coreCall(ctx, u.ID, input.ClientRequestID, n.ID, input.Command, payload)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		failError(w, r, err)
		return
	}
	success(w, r, map[string]any{"operation": op})
}
func (s *Server) getCoreOperation(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "core.manage"); err != nil {
		failError(w, r, err)
		return
	}
	op, err := s.Store.CoreOperation(r.Context(), r.PathValue("operationId"))
	if err != nil {
		failure(w, r, 404, "NOT_FOUND", "操作不存在。")
		return
	}
	success(w, r, map[string]any{"operation": op})
}
func (s *Server) retryCoreOperation(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "core.manage"); err != nil {
		failError(w, r, err)
		return
	}
	id := r.PathValue("operationId")
	op, err := s.Store.CoreOperation(r.Context(), id)
	if err != nil {
		failure(w, r, 404, "NOT_FOUND", "操作不存在。")
		return
	}
	if !s.Core.Online(op.NodeID) {
		failError(w, r, bridge.ErrNodeOffline)
		return
	}
	if err = s.Store.RetryCore(r.Context(), id); err != nil {
		failError(w, r, err)
		return
	}
	op, err = s.Store.CoreOperation(r.Context(), id)
	if err == nil {
		err = s.dispatchCore(op)
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	success(w, r, map[string]any{"operation": op})
}

type CoreMailAttachment struct {
	ItemRef       string `json:"itemRef"`
	Revision      int64  `json:"revision"`
	Quantity      int64  `json:"quantity"`
	PayloadSHA256 string `json:"payloadSha256,omitempty"`
}

func (s *Server) createCoreMail(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "core.manage")
	if err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		ClientRequestID    string               `json:"clientRequestId"`
		ServerID           string               `json:"serverId"`
		RecipientPlayerRef string               `json:"recipientPlayerRef"`
		Title              string               `json:"title"`
		Body               string               `json:"body"`
		AllowedServerIDs   []string             `json:"allowedServerIds"`
		Attachments        []CoreMailAttachment `json:"attachments"`
	}
	if err = body(w, r, &input); err != nil || !bridge.ValidMessageID(input.ClientRequestID) || len(input.Attachments) == 0 || len(input.Attachments) > 16 || len(input.Title) == 0 || len(input.Title) > 512 || len(input.Body) > 2048 || len(input.AllowedServerIDs) == 0 || len(input.AllowedServerIDs) > 32 {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	n, err := s.node(input.ServerID)
	if err != nil || n.MailCluster == "" || !n.ClaimEnabled {
		failError(w, r, ErrForbidden)
		return
	}
	if !s.Core.Online(n.ID) {
		failError(w, r, bridge.ErrNodeOffline)
		return
	}
	uuid, err := s.Store.CoreRecipient(r.Context(), input.RecipientPlayerRef)
	if err != nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	domain := ""
	seen := map[string]bool{}
	sort.Strings(input.AllowedServerIDs)
	for i, a := range input.Attachments {
		if a.Revision < 1 || a.Quantity < 1 || a.Quantity > 99999 {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		item, archived, e := s.Store.CoreItem(r.Context(), a.ItemRef, a.Revision)
		if e != nil || archived || item.Codec != "bukkit-bytes-v1" || item.InventoryDomain == "" || a.Quantity > item.MaxQuantity {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		if seen[a.ItemRef] {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		seen[a.ItemRef] = true
		if domain == "" {
			domain = item.InventoryDomain
		}
		if domain != item.InventoryDomain {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		for j, target := range input.AllowedServerIDs {
			if j > 0 && input.AllowedServerIDs[j-1] == target {
				failError(w, r, bridge.ErrProtocol)
				return
			}
			targetNode, e := s.node(target)
			if e != nil || !targetNode.ClaimEnabled || targetNode.InventoryDomain != domain {
				failError(w, r, bridge.ErrProtocol)
				return
			}
			compatible := false
			for _, supported := range item.CompatibleServerIDs {
				if supported == target {
					compatible = true
				}
			}
			if !compatible {
				failError(w, r, bridge.ErrProtocol)
				return
			}
		}
		input.Attachments[i].PayloadSHA256 = item.PayloadSHA256
	}
	seed := store.Digest([]byte(u.ID + ":" + input.ClientRequestID))[:32]
	delivery := "delivery_" + seed
	order := "manual_" + seed
	snapshot, _ := json.Marshal(map[string]any{"schemaVersion": 1, "orderId": order, "recipientUuid": uuid, "inventoryDomain": domain, "allowedServerIds": input.AllowedServerIDs, "attachments": input.Attachments})
	payload := map[string]any{"source": "deuterium-commerce", "deliveryId": delivery, "orderId": order, "recipientUuid": uuid, "title": strings.TrimSpace(input.Title), "body": input.Body, "sender": "Deuterium", "snapshotJson": string(snapshot), "snapshotSha256": store.Digest(snapshot), "allowedServerIds": input.AllowedServerIDs, "inventoryDomain": domain}
	raw, _ := json.Marshal(payload)
	if len(raw) > 28000 {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	op, _, err := s.Store.CreateCoreOperation(r.Context(), u.ID, input.ClientRequestID, n.ID, "mailbox.create", raw)
	if err == nil {
		err = s.Store.SaveManualDelivery(r.Context(), delivery, u.ID, op.ID, snapshot)
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	if op.State == "QUEUED" {
		if err = s.dispatchCore(op); err != nil {
			failError(w, r, err)
			return
		}
	}
	writeJSON(w, 202, map[string]any{"requestId": requestID(r), "data": map[string]any{"deliveryId": delivery, "operation": op}})
}
func (s *Server) getCoreMail(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "core.manage"); err != nil {
		failError(w, r, err)
		return
	}
	receipt, err := s.Store.MailReceipt(r.Context(), r.PathValue("deliveryId"))
	if err != nil {
		failure(w, r, 404, "NOT_FOUND", "尚未收到权威邮箱回执，请查询对应操作。")
		return
	}
	success(w, r, map[string]any{"receipt": receipt})
}
