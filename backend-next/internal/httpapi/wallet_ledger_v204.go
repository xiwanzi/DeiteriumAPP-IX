package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type ledgerFilterV204 struct {
	Actor          string    `json:"actor"`
	Scope          string    `json:"scope"`
	PlayerRef      string    `json:"playerRef"`
	Direction      string    `json:"direction"`
	BusinessType   string    `json:"businessType"`
	From           time.Time `json:"from"`
	To             time.Time `json:"to"`
	BeforeTime     time.Time `json:"beforeTime"`
	BeforeSequence string    `json:"beforeSequence"`
	Snapshot       string    `json:"snapshot"`
}

type ledgerRowV204 struct {
	Sequence      string    `json:"sequence"`
	PlayerUUID    string    `json:"playerUuid"`
	GameID        string    `json:"gameId"`
	OtherUUID     *string   `json:"otherUuid"`
	OperationID   string    `json:"operationId"`
	BusinessRef   string    `json:"businessRef"`
	BusinessType  string    `json:"businessType"`
	Source        string    `json:"source"`
	Direction     string    `json:"direction"`
	Amount        string    `json:"amount"`
	BeforeBalance string    `json:"beforeBalance"`
	AfterBalance  string    `json:"afterBalance"`
	OccurredAt    time.Time `json:"occurredAt"`
}
type ledgerPageV204 struct {
	Records  []ledgerRowV204 `json:"records"`
	Snapshot string          `json:"snapshot"`
}

func ledgerQueryV204(r *http.Request, actor, scope string, now time.Time) (ledgerFilterV204, int, error) {
	f := ledgerFilterV204{Actor: actor, Scope: scope, From: now.AddDate(-1, 0, 0), To: now, BeforeSequence: "0", Snapshot: "0"}
	limit := 25 // Keep Core replies below its 32 KiB frame bound.
	invalid := func() (ledgerFilterV204, int, error) { return f, 0, bridge.ErrProtocol }
	q := r.URL.Query()
	for key, values := range q {
		if len(values) != 1 {
			return invalid()
		}
		switch key {
		case "from", "to", "direction", "businessType", "limit", "cursor":
		case "playerRef":
			if scope != "admin" {
				return invalid()
			}
		default:
			return invalid()
		}
	}
	if raw := q.Get("cursor"); raw != "" {
		if len(raw) > 4096 {
			return invalid()
		}
		data, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || bridge.Decode(data, &f) != nil || f.Actor != actor || f.Scope != scope {
			return invalid()
		}
		before, e1 := strconv.ParseInt(f.BeforeSequence, 10, 64)
		snapshot, e2 := strconv.ParseInt(f.Snapshot, 10, 64)
		if e1 != nil || e2 != nil || before <= 0 || snapshot < before || f.BeforeTime.Before(f.From) || !f.BeforeTime.Before(f.To) {
			return invalid()
		}
	}
	for _, v := range []struct {
		key   string
		value *string
	}{{"playerRef", &f.PlayerRef}, {"direction", &f.Direction}, {"businessType", &f.BusinessType}} {
		if q.Has(v.key) {
			if f.BeforeSequence != "0" && *v.value != q.Get(v.key) {
				return invalid()
			}
			*v.value = q.Get(v.key)
		}
	}
	for _, v := range []struct {
		key   string
		value *time.Time
	}{{"from", &f.From}, {"to", &f.To}} {
		if q.Has(v.key) {
			parsed, err := time.Parse(time.RFC3339Nano, q.Get(v.key))
			if err != nil || (f.BeforeSequence != "0" && !parsed.Equal(*v.value)) {
				return invalid()
			}
			*v.value = parsed.UTC()
		}
	}
	if q.Has("to") && !q.Has("from") && f.BeforeSequence == "0" {
		f.From = f.To.AddDate(-1, 0, 0)
	}
	if !f.From.Before(f.To) || f.To.Sub(f.From) > 366*24*time.Hour || f.To.After(now.Add(time.Second)) || len(f.PlayerRef) > 64 || (scope != "admin" && f.PlayerRef != "") {
		return invalid()
	}
	if f.Direction != "" && f.Direction != "income" && f.Direction != "expense" {
		return invalid()
	}
	switch f.BusinessType {
	case "", "TRANSFER", "GAME", "OFFICIAL_STORE", "MARKET_ORDER", "COMMISSION":
	default:
		return invalid()
	}
	if q.Has("limit") {
		n, err := strconv.Atoi(q.Get("limit"))
		if err != nil || n < 1 || n > 100 {
			return invalid()
		}
		limit = min(n, 25)
	}
	return f, limit, nil
}

func (s *Server) readLedgerV204(ctx context.Context, actor, uuid string, f ledgerFilterV204, limit int, record string) (ledgerPageV204, error) {
	var result ledgerPageV204
	node, err := s.economyNode()
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	payload := map[string]string{"playerUuid": uuid, "from": f.From.UTC().Format(time.RFC3339Nano), "to": f.To.UTC().Format(time.RFC3339Nano), "direction": f.Direction, "businessType": f.BusinessType, "beforeTime": "", "beforeSequence": f.BeforeSequence, "snapshot": f.Snapshot, "limit": strconv.Itoa(limit), "recordId": record}
	if !f.BeforeTime.IsZero() {
		payload["beforeTime"] = f.BeforeTime.UTC().Format(time.RFC3339Nano)
	}
	op, err := s.coreCall(ctx, actor, store.ID("ledger_"), node, "wallet.records", payload)
	if err == nil {
		err = CoreResult(op, &result)
	}
	if err != nil {
		return result, err
	}
	snapshot, e := strconv.ParseInt(result.Snapshot, 10, 64)
	if e != nil || snapshot < 0 || result.Records == nil || len(result.Records) > limit+1 {
		return result, bridge.ErrProtocol
	}
	for i, row := range result.Records {
		sequence, e := strconv.ParseInt(row.Sequence, 10, 64)
		_, amountErr := amountText(row.Amount)
		if e != nil || sequence < 1 || sequence > snapshot || amountErr != nil || !identity.ValidUUID(row.PlayerUUID) || (uuid != "" && uuid != row.PlayerUUID) || row.OccurredAt.Before(f.From) || !row.OccurredAt.Before(f.To) || (row.Direction != "income" && row.Direction != "expense") || (row.Source != "APP" && row.Source != "GAME") {
			return result, bridge.ErrProtocol
		}
		if i > 0 {
			prev := result.Records[i-1]
			prevSequence, _ := strconv.ParseInt(prev.Sequence, 10, 64)
			if row.OccurredAt.After(prev.OccurredAt) || (row.OccurredAt.Equal(prev.OccurredAt) && sequence >= prevSequence) {
				return result, bridge.ErrProtocol
			}
		}
	}
	return result, nil
}

func (s *Server) ledgerRecordV204(ctx context.Context, row ledgerRowV204, admin bool) (map[string]any, error) {
	kind := row.BusinessType
	for _, prefix := range []string{"OFFICIAL_STORE", "MARKET_ORDER", "COMMISSION"} {
		if strings.HasPrefix(kind, prefix+"_") {
			kind = prefix
			break
		}
	}
	title := "信用点收入"
	if row.Direction == "expense" {
		title = "信用点支出"
	}
	if row.Source == "GAME" {
		kind = "GAME"
		if row.Direction == "income" {
			title = "游戏内收入"
		} else {
			title = "游戏内支出"
		}
	} else {
		switch kind {
		case "TRANSFER":
			title = "玩家转账"
		case "OFFICIAL_STORE":
			title = "商城交易"
		case "MARKET_ORDER":
			title = "市场交易"
		case "COMMISSION":
			title = "委托收支"
		}
	}
	note, ref := title, row.BusinessRef
	var other any
	if row.OtherUUID != nil {
		var playerRef, gameID string
		err := s.Store.DB.QueryRowContext(ctx, `SELECT player_ref,game_id FROM identities WHERE server_uuid=? UNION SELECT player_ref,game_id FROM core_player_directory WHERE player_uuid=? LIMIT 1`, *row.OtherUUID, *row.OtherUUID).Scan(&playerRef, &gameID)
		if err == nil {
			other = map[string]any{"playerRef": playerRef, "gameId": gameID}
		}
	}
	if row.Source == "APP" && kind == "TRANSFER" {
		var transferID string
		var transferNote *string
		err := s.Store.DB.QueryRowContext(ctx, `SELECT transfer_id,note FROM wallet_transfers_next WHERE operation_id=?`, row.OperationID).Scan(&transferID, &transferNote)
		if err == nil {
			ref = transferID
			if transferNote != nil && *transferNote != "" {
				note = *transferNote
			}
		}
	}
	value := map[string]any{"recordId": "econ_" + row.Sequence, "direction": row.Direction, "amount": row.Amount, "currency": "CREDIT", "status": "success", "title": title, "note": note, "occurredAt": row.OccurredAt, "businessType": kind, "businessRef": ref, "source": row.Source, "otherPlayer": other, "beforeBalance": row.BeforeBalance, "afterBalance": row.AfterBalance}
	if admin {
		value["player"] = map[string]any{"uuid": row.PlayerUUID, "gameId": row.GameID}
		value["operationId"] = row.OperationID
		value["ledgerType"] = row.BusinessType
	}
	return value, nil
}

func (s *Server) walletLedgerHTTPV204(w http.ResponseWriter, r *http.Request, admin bool) {
	var actor, uuid, scope string
	if admin {
		u, err := s.admin(r, "audit.read")
		if err != nil {
			failError(w, r, err)
			return
		}
		actor = u.ID
		scope = "admin"
	} else {
		u, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		actor = u.User.ID
		uuid = u.User.ServerUUID
		scope = "personal"
	}
	f, limit, err := ledgerQueryV204(r, actor, scope, time.Now().UTC())
	if err != nil {
		failError(w, r, err)
		return
	}
	if admin && f.PlayerRef != "" {
		e := s.Store.DB.QueryRowContext(r.Context(), `SELECT server_uuid FROM identities WHERE player_ref=? UNION SELECT player_uuid FROM core_player_directory WHERE player_ref=? LIMIT 1`, f.PlayerRef, f.PlayerRef).Scan(&uuid)
		if e != nil {
			failError(w, r, e)
			return
		}
	}
	page, err := s.readLedgerV204(r.Context(), actor, uuid, f, limit, "")
	if err != nil {
		failError(w, r, err)
		return
	}
	hasMore := len(page.Records) > limit
	var next any
	if hasMore {
		page.Records = page.Records[:limit]
		last := page.Records[len(page.Records)-1]
		f.Snapshot = page.Snapshot
		f.BeforeSequence = last.Sequence
		f.BeforeTime = last.OccurredAt
		raw, _ := json.Marshal(f)
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	records := []map[string]any{}
	for _, row := range page.Records {
		value, e := s.ledgerRecordV204(r.Context(), row, admin)
		if e != nil {
			failError(w, r, e)
			return
		}
		records = append(records, value)
	}
	if admin {
		v2List(w, r, records, next, hasMore)
	} else {
		writeJSON(w, 200, map[string]any{"requestId": requestID(r), "data": map[string]any{"records": records}, "page": map[string]any{"nextCursor": next, "hasMore": hasMore}})
	}
}

func (s *Server) walletLedgerDetailV204(w http.ResponseWriter, r *http.Request) {
	u, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	id := strings.TrimPrefix(r.PathValue("recordId"), "econ_")
	seq, e := strconv.ParseInt(id, 10, 64)
	if e != nil || seq < 1 {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	f := ledgerFilterV204{From: time.Unix(0, 0), To: time.Now().UTC(), BeforeSequence: "0", Snapshot: "0"}
	page, err := s.readLedgerV204(r.Context(), u.User.ID, u.User.ServerUUID, f, 1, id)
	if err != nil {
		failError(w, r, err)
		return
	}
	if len(page.Records) != 1 {
		failure(w, r, 404, "NOT_FOUND", "账单不存在。")
		return
	}
	value, err := s.ledgerRecordV204(r.Context(), page.Records[0], false)
	if err != nil {
		failError(w, r, err)
		return
	}
	value["status"] = "SUCCESS"
	party := map[string]any{"kind": "SYSTEM", "playerRef": nil, "storeId": nil, "displayName": value["title"], "contactQq": nil}
	if other, ok := value["otherPlayer"].(map[string]any); ok {
		party["kind"] = "PLAYER"
		party["playerRef"] = other["playerRef"]
		party["displayName"] = other["gameId"]
	}
	value["counterparty"] = party
	v2Success(w, r, value)
}
