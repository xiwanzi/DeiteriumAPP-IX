package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
)

var supplyCatalogClient = &http.Client{Timeout: 12 * time.Second}

func (s *Server) adminSupplyImportNode(preferred string) (config.Node, bool) {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		if node, err := s.node(preferred); err == nil && node.ItemPrefix != "" && node.ClaimEnabled {
			return node, true
		}
	}
	if s.Config.SupplyEditorNode != "" {
		if node, err := s.node(s.Config.SupplyEditorNode); err == nil && node.ItemPrefix != "" && node.ClaimEnabled {
			return node, true
		}
	}
	for _, node := range s.Config.Nodes {
		if node.ItemPrefix != "" && node.ClaimEnabled && s.Core.Online(node.ID) {
			return node, true
		}
	}
	for _, node := range s.Config.Nodes {
		if node.ItemPrefix != "" && node.ClaimEnabled {
			return node, true
		}
	}
	return config.Node{}, false
}

// adminSupplyImportItem registers an item stack (from creative catalog, snapshot, or SNBT) into the
// authoritative Core item library so it can be immediately bound or added to a supply pool.
func (s *Server) adminSupplyImportItem(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		supplyEditorSelection
		NodeID string `json:"nodeId"`
	}
	if socialBodyV2(w, r, &input, 32000) != nil {
		failure(w, r, 400, "INVALID_INPUT", "物品数据无效或超过入库大小限制。")
		return
	}
	node, ok := s.adminSupplyImportNode(input.NodeID)
	if !ok {
		failure(w, r, 503, "ITEM_IMPORT_UNAVAILABLE", "当前没有已配置物品库权限的在线子服节点。")
		return
	}
	selection := input.supplyEditorSelection
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

func (s *Server) proxyItemCatalog(w http.ResponseWriter, r *http.Request, method, upstreamPath string, body []byte) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	base := strings.TrimRight(s.Config.ItemCatalogURL, "/")
	if base == "" {
		failure(w, r, 503, "ITEM_CATALOG_UNAVAILABLE", "服务器尚未配置客户端物品资源目录（itemCatalogUrl），可改用游戏快照导入、在线玩家背包或已有物品库。")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+upstreamPath, reqBody)
	if err != nil {
		failure(w, r, 502, "ITEM_CATALOG_UNAVAILABLE", "无法构造物品目录请求。")
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.Config.ItemCatalogToken != "" {
		req.Header.Set("X-Deuterium-Web-Token", s.Config.ItemCatalogToken)
	}
	resp, err := supplyCatalogClient.Do(req)
	if err != nil {
		failure(w, r, 503, "ITEM_CATALOG_UNAVAILABLE", "物品资源目录服务暂不可达，请稍后重试或改用玩家背包/已有物品库。")
		return
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		failure(w, r, 502, "ITEM_CATALOG_UNAVAILABLE", "读取物品目录响应失败。")
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var detail struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &detail)
		msg := strings.TrimSpace(detail.Message)
		if msg == "" {
			msg = "物品资源目录查询失败。"
		}
		status := resp.StatusCode
		if status == 401 || status == 403 {
			status = 503
		}
		failure(w, r, status, "ITEM_CATALOG_ERROR", msg)
		return
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		failure(w, r, 502, "ITEM_CATALOG_UNAVAILABLE", "物品资源目录返回了无效数据。")
		return
	}
	v2Success(w, r, decoded)
}

func (s *Server) adminSupplyItemCatalog(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	pageText := strings.TrimSpace(r.URL.Query().Get("page"))
	page := 0
	if pageText != "" {
		parsed, err := strconv.Atoi(pageText)
		if err != nil || parsed < 0 || parsed > 10000 {
			failure(w, r, 400, "INVALID_INPUT", "页码无效。")
			return
		}
		page = parsed
	}
	if len(query) > 120 || len(category) > 100 {
		failure(w, r, 400, "INVALID_INPUT", "查询参数过长。")
		return
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", strconv.Itoa(page))
	params.Set("category", category)
	s.proxyItemCatalog(w, r, http.MethodGet, "/api/v1/web/dungeons/item-catalog?"+params.Encode(), nil)
}

func (s *Server) adminSupplyItemPreviews(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 280*1024))
	if err != nil || len(raw) == 0 {
		failure(w, r, 400, "INVALID_INPUT", "物品预览请求无效。")
		return
	}
	s.proxyItemCatalog(w, r, http.MethodPost, "/api/v1/web/dungeons/item-previews", raw)
}

func (s *Server) adminSupplyCompatibleGuns(w http.ResponseWriter, r *http.Request) {
	attachment := strings.TrimSpace(r.URL.Query().Get("attachment"))
	if attachment == "" || len(attachment) > 160 {
		failure(w, r, 400, "INVALID_INPUT", "配件标识无效。")
		return
	}
	params := url.Values{}
	params.Set("attachment", attachment)
	s.proxyItemCatalog(w, r, http.MethodGet, "/api/v1/web/dungeons/item-compatible-guns?"+params.Encode(), nil)
}
