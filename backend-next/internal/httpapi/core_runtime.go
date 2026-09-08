package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coder/websocket"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) dispatchCore(op store.CoreOperation) error {
	ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
	defer cancel()
	claimed, err := s.Store.MarkCoreSent(ctx, op.ID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	frame := map[string]any{"operationId": op.ID, "command": op.Command, "expiresAt": op.ExpiresAt, "payload": op.Payload}
	if err = s.Core.Send(ctx, op.NodeID, frame); err != nil {
		_ = s.Store.MarkCoreUnknown(ctx, op.ID)
		s.Core.Notify(op.ID)
		return err
	}
	return nil
}
func (s *Server) coreCall(ctx context.Context, actor, client, node, command string, payload any) (store.CoreOperation, error) {
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) > 28000 {
		return store.CoreOperation{}, bridge.ErrProtocol
	}
	op, _, err := s.Store.CreateCoreOperation(ctx, actor, client, node, command, raw)
	if err != nil {
		return op, err
	}
	wake, unwatch := s.Core.Watch(op.ID)
	defer unwatch()
	if op.State == "QUEUED" {
		if !s.Core.Online(node) {
			return op, bridge.ErrNodeOffline
		}
		if err = s.dispatchCore(op); err != nil {
			return op, err
		}
	}
	poll := time.NewTicker(time.Second)
	defer poll.Stop()
	for {
		current, lookupErr := s.Store.CoreOperation(ctx, op.ID)
		if lookupErr != nil {
			return op, lookupErr
		}
		op = current
		if op.State == "COMPLETED" || op.State == "FAILED" || op.State == "UNKNOWN" {
			return op, nil
		}
		select {
		case <-ctx.Done():
			return op, ctx.Err()
		case <-wake:
		case <-poll.C:
		}
	}
}
func (s *Server) coreSpecial(ctx context.Context, node config.Node, c *websocket.Conn, e bridge.Envelope) (bool, error) {
	switch e.Type {
	case "core.status":
		var status struct {
			NodeID               string          `json:"nodeId"`
			ClusterID            string          `json:"clusterId"`
			Version              string          `json:"version"`
			StorageHealthy       bool            `json:"storageHealthy"`
			SharedStorage        bool            `json:"sharedStorage"`
			InventoryDomain      string          `json:"inventoryDomain"`
			CompatibilityProfile string          `json:"compatibilityProfile"`
			ChatMode             string          `json:"chatMode"`
			PlayerDataReady      bool            `json:"playerDataReady"`
			PlayerDataProvider   string          `json:"playerDataProvider"`
			Economy              bool            `json:"economy"`
			EconomyAuthority     bool            `json:"economyAuthority"`
			Mailbox              json.RawMessage `json:"mailbox"`
		}
		if bridge.Decode(e.Payload, &status) != nil || status.NodeID != node.ID || status.InventoryDomain != node.InventoryDomain || len(status.Version) > 32 || len(status.PlayerDataProvider) > 128 {
			return true, bridge.ErrProtocol
		}
		s.Core.SetStatus(node.ID, e.Payload)
		return true, nil
	case "core.presence.snapshot":
		var snapshot struct {
			Players []struct {
				PlayerUUID   string `json:"playerUuid"`
				GameID       string `json:"gameId"`
				SessionEpoch string `json:"sessionEpoch"`
			} `json:"players"`
		}
		if bridge.Decode(e.Payload, &snapshot) != nil || len(snapshot.Players) > 200 {
			return true, bridge.ErrProtocol
		}
		for _, p := range snapshot.Players {
			if !identity.ValidUUID(p.PlayerUUID) || !identity.ValidGameID(p.GameID) || !identity.ValidUUID(p.SessionEpoch) {
				return true, bridge.ErrProtocol
			}
			if _, err := s.Store.RememberCorePlayer(ctx, store.CorePlayerIdentity{PlayerUUID: p.PlayerUUID, GameID: p.GameID, ServerID: node.ID, Online: true}); err != nil {
				return true, err
			}
		}
		s.Core.SetPresence(node.ID, e.Payload)
		return true, nil
	case "core.command.result":
		var result struct {
			OperationID string          `json:"operationId"`
			Status      string          `json:"status"`
			Data        json.RawMessage `json:"data"`
			Error       json.RawMessage `json:"error"`
		}
		if bridge.Decode(e.Payload, &result) != nil || !bridge.ValidMessageID(result.OperationID) {
			return true, bridge.ErrProtocol
		}
		op, err := s.Store.CoreOperation(ctx, result.OperationID)
		if err != nil {
			return true, err
		}
		if op.NodeID != node.ID {
			return true, store.ErrUnauthorized
		}
		if result.Status == "COMPLETED" && op.Command == "player.resolve" {
			var p store.CorePlayerIdentity
			if bridge.Decode(result.Data, &p) != nil || !identity.ValidUUID(p.PlayerUUID) || !identity.ValidGameID(p.GameID) || !s.Config.HasNode(p.ServerID) {
				return true, bridge.ErrProtocol
			}
			if _, err = s.Store.RememberCorePlayer(ctx, p); err != nil {
				return true, err
			}
		}
		if err = s.acceptCoreReceiptV2(ctx, node, op, result.Status, result.Data); err != nil {
			return true, err
		}
		err = s.Store.CoreReply(ctx, node.ID, result.OperationID, result.Status, e.Payload)
		if err == nil && op.Command == "verification.deliver" {
			err = s.Store.ForgetVerificationCommand(ctx, op.ID)
		}
		if err == nil {
			s.Core.Notify(result.OperationID)
		}
		return true, err
	case "item.catalog.updated":
		catalog, err := bridge.ValidateCatalog(node, e)
		var seq int64
		var replay bool
		if err == nil {
			seq, replay, err = s.Store.CoreCatalogEvent(ctx, node.ID, e.EventID, catalog)
		}
		return true, s.coreAck(ctx, c, e, seq, replay, err)
	case "mailbox.created.event", "mailbox.claimed.event", "mailbox.revoked.event", "mailbox.uncertain.event":
		event, err := bridge.ValidateMail(s.Config, node, e)
		var seq int64
		var replay bool
		if err == nil {
			seq, replay, err = s.Store.MailEvent(ctx, event)
		}
		return true, s.coreAck(ctx, c, e, seq, replay, err)
	default:
		return false, nil
	}
}
func (s *Server) coreAck(ctx context.Context, c *websocket.Conn, e bridge.Envelope, seq int64, replay bool, err error) error {
	if err != nil {
		status, code := "unknown", "SERVICE_UNAVAILABLE"
		if errors.Is(err, bridge.ErrProtocol) {
			status, code = "rejected", "INVALID_EVENT"
		} else if errors.Is(err, store.ErrConflict) {
			status, code = "rejected", "IDEMPOTENCY_CONFLICT"
		}
		return sendSocket(ctx, c, "core.event.result", e.RequestID, map[string]any{"eventId": e.EventID, "status": status, "error": map[string]string{"code": code, "message": "事件未提交，请按原标识核对。"}})
	}
	return sendSocket(ctx, c, "core.event.result", e.RequestID, map[string]any{"eventId": e.EventID, "status": "committed", "sequence": seq, "replayed": replay})
}
