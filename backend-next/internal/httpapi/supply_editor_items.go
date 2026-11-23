package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

const (
	supplyEditorActor   = store.SupplyEditorActor
	supplyEditorImport  = store.SupplyEditorImportCommand
	supplyEditorArchive = "items.editor_archive"
	supplyEditorPath    = "/internal/v1/supply-editor/items"
)

var supplyResourceID = regexp.MustCompile(`^[a-z0-9_.-]+:[a-z0-9_./-]+$`)
var supplyEditorKey = regexp.MustCompile(`^[a-f0-9]{32}$`)

type supplyEditorSelection struct {
	ItemID    string `json:"itemId"`
	StackData string `json:"stackData"`
	Name      string `json:"name"`
}

func (in supplyEditorSelection) reference(prefix string) string {
	return prefix + "gacha_" + store.Digest([]byte(in.ItemID + "\x00" + in.StackData + "\x00" + in.Name))[:32]
}

type supplyEditorArchiveInput struct {
	ItemRef        string `json:"itemRef"`
	CatalogVersion int64  `json:"catalogVersion"`
}

func (s *Server) registerSupplyEditorItems(mux *http.ServeMux) {
	mux.HandleFunc("GET "+supplyEditorPath, s.supplyEditorItems)
	mux.HandleFunc("POST "+supplyEditorPath, s.supplyEditorRegister)
	mux.HandleFunc("POST "+supplyEditorPath+"/archive", s.supplyEditorRemove)
}

func (s *Server) supplyEditorNode(w http.ResponseWriter, r *http.Request) (config.Node, bool) {
	auth := r.Header.Get("Authorization")
	token, bearer := strings.CutPrefix(auth, "Bearer ")
	digest := sha256.Sum256([]byte(token))
	expected, _ := hex.DecodeString(s.Config.SupplyEditorTokenSHA256)
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if !net.ParseIP(host).IsLoopback() || !bearer || len(token) < 32 ||
		len(expected) != sha256.Size || subtle.ConstantTimeCompare(expected, digest[:]) != 1 {
		failure(w, r, 401, "UNAUTHENTICATED", "需要抽卡编辑器服务身份。")
		return config.Node{}, false
	}
	node, err := s.node(s.Config.SupplyEditorNode)
	if err != nil || node.ItemPrefix == "" || !node.ClaimEnabled {
		failure(w, r, 503, "ITEM_IMPORT_UNAVAILABLE", "物品库节点尚未配置。")
		return config.Node{}, false
	}
	return node, true
}

func (s *Server) supplyEditorItems(w http.ResponseWriter, r *http.Request) {
	node, ok := s.supplyEditorNode(w, r)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	pageText := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageText)
	if pageText == "" {
		page, err = 0, nil
	}
	if err != nil || page < 0 || page > 10000 || len(query) > 256 {
		failure(w, r, 400, "INVALID_INPUT", "物品库查询参数无效。")
		return
	}
	items, more, err := s.Store.SupplyEditorItems(r.Context(), node.ID, node.ItemPrefix+"gacha_", query, page)
	if err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, map[string]any{"items": items, "hasMore": more})
}

func (s *Server) supplyEditorRegister(w http.ResponseWriter, r *http.Request) {
	node, ok := s.supplyEditorNode(w, r)
	if !ok {
		return
	}
	var selection supplyEditorSelection
	if socialBodyV2(w, r, &selection, 32000) != nil {
		failure(w, r, 400, "INVALID_INPUT", "物品数据无效或超过入库大小限制。")
		return
	}
	selection.Name = strings.TrimSpace(selection.Name)
	selection.StackData = strings.TrimSpace(selection.StackData)
	if selection.Name == "" {
		selection.Name = selection.ItemID
	}
	if len(selection.ItemID) > 256 || !supplyResourceID.MatchString(selection.ItemID) || selection.ItemID == "minecraft:air" ||
		utf8.RuneCountInString(selection.Name) > 128 || strings.ContainsRune(selection.Name, 0) ||
		strings.ContainsRune(selection.StackData, 0) {
		failure(w, r, 400, "INVALID_INPUT", "请选择有效的物品。")
		return
	}
	ref := selection.reference(node.ItemPrefix)
	existing, err := s.Store.SupplyEditorItem(r.Context(), node.ID, node.ItemPrefix+"gacha_", ref)
	if err == nil && !existing.Archived {
		v2Success(w, r, existing)
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		failError(w, r, err)
		return
	}
	payload := struct {
		supplyEditorSelection
		CatalogVersion int64 `json:"catalogVersion"`
	}{selection, existing.CatalogVersion}
	encoded, err := json.Marshal(payload)
	if err != nil || len(encoded) > 26000 {
		failure(w, r, 400, "INVALID_INPUT", "物品数据超过入库大小限制。")
		return
	}
	s.writeSupplyEditorItem(w, r, node, supplyEditorImport, ref, existing.CatalogVersion, payload)
}

func (s *Server) supplyEditorRemove(w http.ResponseWriter, r *http.Request) {
	node, ok := s.supplyEditorNode(w, r)
	if !ok {
		return
	}
	prefix := node.ItemPrefix + "gacha_"
	var input supplyEditorArchiveInput
	if socialBodyV2(w, r, &input, 1024) != nil || !strings.HasPrefix(input.ItemRef, prefix) ||
		!supplyEditorKey.MatchString(strings.TrimPrefix(input.ItemRef, prefix)) || input.CatalogVersion < 1 {
		failure(w, r, 400, "INVALID_INPUT", "请选择抽卡编辑器登记的物品。")
		return
	}
	item, err := s.Store.SupplyEditorItem(r.Context(), node.ID, prefix, input.ItemRef)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			failure(w, r, 404, "ITEM_NOT_FOUND", "找不到该编辑器物品。")
		} else {
			failError(w, r, err)
		}
		return
	}
	if item.Archived {
		v2Success(w, r, item)
		return
	}
	if item.CatalogVersion != input.CatalogVersion {
		failure(w, r, 409, "ITEM_CATALOG_CHANGED", "物品目录已变化，请刷新后重试。")
		return
	}
	inUse, err := s.Store.SupplyItemReferenced(r.Context(), input.ItemRef)
	if err != nil {
		failError(w, r, err)
		return
	}
	if inUse {
		failure(w, r, 409, "ITEM_IN_USE", "请先将该物品从正式卡池的奖品中移除。")
		return
	}
	s.writeSupplyEditorItem(w, r, node, supplyEditorArchive, input.ItemRef, input.CatalogVersion, input)
}

func (s *Server) writeSupplyEditorItem(w http.ResponseWriter, r *http.Request, node config.Node, command, ref string, version int64, payload any) {
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	key := store.SupplyEditorOperationKey(node.ID, command, ref, version)
	previous, err := s.Store.CoreOperationByRequest(ctx, supplyEditorActor, key)
	if err == nil && (previous.State == "UNKNOWN" || previous.State == "QUEUED" && time.Now().After(previous.ExpiresAt)) {
		if err = s.Store.RetryCore(ctx, previous.ID); err != nil {
			failError(w, r, err)
			return
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		failError(w, r, err)
		return
	}
	op, err := s.coreCall(ctx, supplyEditorActor, key, node.ID, command, payload)
	if err != nil {
		if errors.Is(err, bridge.ErrNodeOffline) {
			failure(w, r, 503, "NODE_OFFLINE", "物品库节点尚未连接，请稍后重试。")
			return
		}
		failError(w, r, err)
		return
	}
	var receipt map[string]json.RawMessage
	if err := CoreResult(op, &receipt); err != nil {
		var core *CoreError
		if errors.As(err, &core) {
			status := http.StatusConflict
			if op.State != "FAILED" {
				status = http.StatusServiceUnavailable
			}
			failure(w, r, status, core.Code, core.Message)
		} else {
			failError(w, r, err)
		}
		return
	}
	item, err := s.Store.SupplyEditorItem(ctx, node.ID, node.ItemPrefix+"gacha_", ref)
	if err != nil {
		failError(w, r, err)
		return
	}
	if item.Archived != (command == supplyEditorArchive) {
		failure(w, r, 409, "ITEM_CATALOG_CHANGED", "物品目录已变化，请刷新后重试。")
		return
	}
	v2Success(w, r, item)
}

func (s *Server) acceptSupplyEditorReceipt(ctx context.Context, node config.Node, op store.CoreOperation, data json.RawMessage) error {
	var response struct {
		Item json.RawMessage   `json:"item"`
		Head store.CoreCatalog `json:"head"`
	}
	if op.ActorID != supplyEditorActor || node.ID != op.NodeID || json.Unmarshal(data, &response) != nil {
		return bridge.ErrProtocol
	}
	prefix := node.ItemPrefix + "gacha_"
	head := response.Head
	if !strings.HasPrefix(head.ItemRef, prefix) || !supplyEditorKey.MatchString(strings.TrimPrefix(head.ItemRef, prefix)) {
		return bridge.ErrProtocol
	}
	refHash := store.Digest([]byte(head.ItemRef))[:32]
	if op.Command == supplyEditorImport {
		var in supplyEditorSelection
		var item store.ItemVersion
		if json.Unmarshal(op.Payload, &in) != nil || json.Unmarshal(response.Item, &item) != nil ||
			head.Archived || head.ItemRef != in.reference(node.ItemPrefix) || item.ItemRef != head.ItemRef ||
			item.ItemID != in.ItemID || item.Revision != head.LatestRevision || item.Codec != "bukkit-bytes-v1" {
			return bridge.ErrProtocol
		}
		event := bridge.Envelope{EventID: "item_" + refHash + "_" + strconv.FormatInt(item.Revision, 10),
			Type: "item.version.published", Payload: response.Item}
		payload, _, accepted, err := bridge.ValidateCore(s.Config, node, event)
		if err != nil {
			return err
		}
		if _, _, err = s.Store.CoreEvent(ctx, node.ID, event.EventID, event.Type, payload, nil, accepted); err != nil {
			return err
		}
	} else {
		var in supplyEditorArchiveInput
		if json.Unmarshal(op.Payload, &in) != nil || !head.Archived || head.ItemRef != in.ItemRef ||
			(head.CatalogVersion != in.CatalogVersion && head.CatalogVersion != in.CatalogVersion+1) {
			return bridge.ErrProtocol
		}
	}
	raw, err := json.Marshal(head)
	if err != nil {
		return err
	}
	event := bridge.Envelope{EventID: "catalog_" + refHash + "_" + strconv.FormatInt(head.CatalogVersion, 10),
		Type: "item.catalog.updated", Payload: raw}
	checked, err := bridge.ValidateCatalog(node, event)
	if err != nil {
		return err
	}
	_, _, err = s.Store.CoreCatalogEvent(ctx, node.ID, event.EventID, checked)
	return err
}
