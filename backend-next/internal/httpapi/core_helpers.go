package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"strings"
)

// CoreError distinguishes a business rejection from an unconfirmed operation.
type CoreError struct{ Code, Message string }

func (e *CoreError) Error() string { return e.Code + ": " + e.Message }
func CoreResult(op store.CoreOperation, target any) error {
	if op.State != "COMPLETED" {
		var reply struct {
			Error *CoreError `json:"error"`
		}
		_ = json.Unmarshal(op.Result, &reply)
		if op.State == "FAILED" && reply.Error != nil {
			return reply.Error
		}
		return &CoreError{"RESULT_UNKNOWN", "操作结果尚未确认，请查询原操作。"}
	}
	var reply struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(op.Result, &reply) != nil || len(reply.Data) == 0 {
		return bridge.ErrProtocol
	}
	return json.Unmarshal(reply.Data, target)
}

// CoreCall persists each command before sending. Callers must interpret CoreResult
// and (for Mail) the nested domain code before advancing their business state.
func (s *Server) CoreCall(ctx context.Context, actor, client, node, command string, payload any) (store.CoreOperation, error) {
	return s.coreCall(ctx, actor, client, node, command, payload)
}

// QueryCoreOperation returns the persisted operation even when its node is
// offline. Recovery only asks for the original committed receipt.
func (s *Server) QueryCoreOperation(ctx context.Context, id string) (store.CoreOperation, error) {
	op, err := s.Store.CoreOperation(ctx, id)
	if err != nil {
		return op, err
	}
	if op.State == "UNKNOWN" || op.State == "PROCESSING" || op.State == "SENT" {
		if err = s.reconcileCoreOperation(ctx, id); err != nil {
			return op, err
		}
		return s.Store.CoreOperation(ctx, id)
	}
	return op, nil
}
func (s *Server) economyNode() (string, error) {
	for _, n := range s.Config.Nodes {
		if n.Economy {
			return n.ID, nil
		}
	}
	return "", &CoreError{"ECONOMY_UNAVAILABLE", "受控经济节点尚未配置。"}
}
func (s *Server) resolveCorePlayer(ctx context.Context, name string, onlineOnly bool) (store.CorePlayerIdentity, error) {
	if !identity.ValidGameID(name) || identity.SystemGameID(name) {
		return store.CorePlayerIdentity{}, bridge.ErrProtocol
	}
	live, err := s.Core.Player(name, "")
	if err != nil {
		return live, err
	}
	if live.PlayerUUID != "" {
		return live, nil
	}
	if onlineOnly {
		return live, &CoreError{"PLAYER_OFFLINE", "请先使用该游戏账号进入服务器。"}
	}
	var found store.CorePlayerIdentity
	for _, node := range s.Config.Nodes {
		if !s.Core.Online(node.ID) {
			continue
		}
		op, e := s.coreCall(ctx, "identity-resolver", store.ID("resolve_"), node.ID, "player.resolve", map[string]string{"gameId": name})
		if e != nil {
			continue
		}
		var p store.CorePlayerIdentity
		if e = CoreResult(op, &p); e != nil {
			var ce *CoreError
			if errors.As(e, &ce) && ce.Code == "IDENTITY_AMBIGUOUS" {
				return p, e
			}
			continue
		}
		if !identity.ValidUUID(p.PlayerUUID) || !strings.EqualFold(p.GameID, name) {
			return p, bridge.ErrProtocol
		}
		if found.PlayerUUID != "" && found.PlayerUUID != p.PlayerUUID {
			return p, &CoreError{"IDENTITY_AMBIGUOUS", "名称对应多个历史游戏身份。"}
		}
		found = p
	}
	if found.PlayerUUID == "" {
		return found, &CoreError{"PLAYER_NOT_FOUND", "未找到服务器确认的玩家身份。"}
	}
	_, err = s.Store.RememberCorePlayer(ctx, found)
	return found, err
}
