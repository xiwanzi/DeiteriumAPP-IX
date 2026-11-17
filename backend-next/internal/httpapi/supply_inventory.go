package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var supplyInventorySlotID = regexp.MustCompile(`^(storage:([0-9]|[12][0-9]|3[0-5])|head|chest|legs|feet|offhand)$`)
var supplyItemFingerprint = regexp.MustCompile(`^[a-f0-9]{64}$`)

type supplyInventorySlot struct {
	Slot        string `json:"slot"`
	Name        string `json:"name"`
	ItemID      string `json:"itemId"`
	Amount      int    `json:"amount"`
	Fingerprint string `json:"fingerprint"`
	Savable     bool   `json:"savable"`
	Reason      string `json:"reason"`
}

type supplyInventorySnapshot struct {
	PlayerUUID   string                `json:"playerUuid"`
	SessionEpoch string                `json:"sessionEpoch"`
	Slots        []supplyInventorySlot `json:"slots"`
}

type supplyCaptureInput struct {
	ClientRequestID string `json:"clientRequestId"`
	ServerID        string `json:"serverId"`
	PlayerUUID      string `json:"playerUuid"`
	SessionEpoch    string `json:"sessionEpoch"`
	Slot            string `json:"slot"`
	Fingerprint     string `json:"fingerprint"`
}

func (s *Server) registerSupplyInventory(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/supply/inventory", s.readSupplyInventory)
	mux.HandleFunc("POST /api/v1/admin/supply/inventory/capture", s.captureSupplyInventory)
}

func (s *Server) readSupplyInventory(w http.ResponseWriter, r *http.Request) {
	actor, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("player"))
	var player store.CorePlayerIdentity
	if identity.ValidUUID(strings.ToLower(query)) {
		player, err = s.Core.Player("", strings.ToLower(query))
	} else {
		player, err = s.resolveCorePlayer(r.Context(), query, true)
	}
	if err != nil || !player.Online {
		failure(w, r, 409, "PLAYER_OFFLINE", "未找到唯一在线玩家，请确认游戏 ID，并让玩家进入游戏服务器。")
		return
	}
	node, err := s.supplyInventoryNode(player.ServerID)
	if err != nil {
		failError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	op, err := s.coreCall(ctx, actor.ID, store.ID("supply_inventory_"), node.ID, "items.inventory", map[string]string{"playerUuid": player.PlayerUUID, "sessionEpoch": player.SessionEpoch})
	if err != nil {
		failError(w, r, err)
		return
	}
	var snapshot supplyInventorySnapshot
	if err = CoreResult(op, &snapshot); err != nil {
		failError(w, r, err)
		return
	}
	if snapshot.PlayerUUID != player.PlayerUUID || snapshot.SessionEpoch != player.SessionEpoch || len(snapshot.Slots) > 41 {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	seen := map[string]bool{}
	for _, slot := range snapshot.Slots {
		if seen[slot.Slot] || !supplyInventorySlotID.MatchString(slot.Slot) || slot.Amount < 0 || slot.Savable && !supplyItemFingerprint.MatchString(slot.Fingerprint) {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		seen[slot.Slot] = true
	}
	v2Success(w, r, map[string]any{"playerUuid": player.PlayerUUID, "gameId": player.GameID, "serverId": node.ID, "sessionEpoch": snapshot.SessionEpoch, "slots": snapshot.Slots})
}

func (s *Server) supplyInventoryNode(id string) (config.Node, error) {
	node, err := s.node(id)
	if err != nil || node.ItemPrefix == "" || !strings.HasSuffix(node.ItemPrefix, ":") {
		return node, &CoreError{"ITEM_CAPTURE_UNAVAILABLE", "玩家所在节点尚未开放背包物品入库，请进入已配置物品库的子服。"}
	}
	return node, nil
}

func (s *Server) captureSupplyInventory(w http.ResponseWriter, r *http.Request) {
	actor, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	var in supplyCaptureInput
	if socialBodyV2(w, r, &in, 4096) != nil || !bridge.ValidMessageID(in.ClientRequestID) || len(in.ClientRequestID) > 80 || !identity.ValidUUID(in.PlayerUUID) || len(in.SessionEpoch) == 0 || len(in.SessionEpoch) > 80 || !supplyInventorySlotID.MatchString(in.Slot) || !supplyItemFingerprint.MatchString(in.Fingerprint) {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	node, err := s.supplyInventoryNode(in.ServerID)
	if err != nil {
		failError(w, r, err)
		return
	}
	key := "inventory_" + in.ClientRequestID
	_, lookup := s.Store.CoreOperationByRequest(r.Context(), actor.ID, key)
	if errors.Is(lookup, sql.ErrNoRows) {
		if _, err = s.supplyPlayer(node, in.PlayerUUID, in.SessionEpoch); err != nil {
			failure(w, r, 409, "PLAYER_SESSION_CHANGED", "玩家已离线或切服，请刷新背包后重新选择。")
			return
		}
	} else if lookup != nil {
		failError(w, r, lookup)
		return
	}
	payload := map[string]string{"playerUuid": in.PlayerUUID, "sessionEpoch": in.SessionEpoch, "slot": in.Slot, "fingerprint": in.Fingerprint, "actor": actor.ID}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	op, err := s.coreCall(ctx, actor.ID, key, node.ID, "items.capture", payload)
	if errors.Is(err, store.ErrConflict) {
		catalogFailV2(w, r, err)
		return
	}
	if op.ID == "" {
		failError(w, r, err)
		return
	}
	if op.State == "UNKNOWN" {
		if recovered, e := s.QueryCoreOperation(ctx, op.ID); e == nil {
			op = recovered
		}
		if op.State == "UNKNOWN" && s.Core.Online(op.NodeID) {
			// Capture is create-once by operation ID; a replay first returns a committed item.
			if e := s.Store.RetryCore(ctx, op.ID); e == nil {
				op, err = s.coreCall(ctx, actor.ID, key, node.ID, "items.capture", payload)
			}
		}
	}
	if op.State == "FAILED" {
		var failureResult map[string]any
		cause := CoreResult(op, &failureResult)
		var core *CoreError
		if errors.As(cause, &core) {
			failure(w, r, 409, core.Code, core.Message)
		} else {
			failure(w, r, 409, "ITEM_CAPTURE_FAILED", "物品未能保存，请刷新背包并检查物品是否支持入库。")
		}
		return
	}
	if op.State != "COMPLETED" {
		failure(w, r, 503, "RESULT_UNKNOWN", "保存结果仍在确认，请恢复原保存请求；不会另建物品版本。")
		return
	}
	var result struct {
		Item store.ItemVersion `json:"item"`
	}
	if err = CoreResult(op, &result); err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, result.Item)
}

// Index the same canonical publication event as Core's durable outbox before reporting success.
// A later outbox delivery is an ordinary replay, including after a lost capture response.
func (s *Server) acceptSupplyCaptureReceipt(ctx context.Context, node config.Node, op store.CoreOperation, data json.RawMessage) error {
	var result struct {
		Item json.RawMessage `json:"item"`
	}
	if bridge.Decode(data, &result) != nil {
		return bridge.ErrProtocol
	}
	var item store.ItemVersion
	if json.Unmarshal(result.Item, &item) != nil || item.ItemRef != node.ItemPrefix+"supply_"+store.Digest([]byte(op.ID))[:32] || item.Revision != 1 || item.Codec != "bukkit-bytes-v1" || item.InventoryDomain != node.InventoryDomain {
		return bridge.ErrProtocol
	}
	event := bridge.Envelope{EventID: "item_" + store.Digest([]byte(item.ItemRef))[:32] + "_1", Type: "item.version.published", Payload: result.Item}
	payload, _, accepted, err := bridge.ValidateCore(s.Config, node, event)
	if err != nil {
		return err
	}
	_, _, err = s.Store.CoreEvent(ctx, node.ID, event.EventID, event.Type, payload, nil, accepted)
	return err
}
