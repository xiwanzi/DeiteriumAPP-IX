package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var moneyPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,12})(\.[0-9]{1,2})?$`)

func amountText(value string) (string, error) {
	if !moneyPattern.MatchString(value) {
		return "", bridge.ErrProtocol
	}
	parts := strings.Split(value, ".")
	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
	}
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	cents, _ := strconv.ParseInt(fraction, 10, 64)
	if whole*100+cents <= 0 || whole*100+cents > 100000000000000 {
		return "", bridge.ErrProtocol
	}
	return parts[0] + "." + fraction, nil
}
func (s *Server) walletRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/wallet/balance", s.walletBalance)
	mux.HandleFunc("POST /api/v1/wallet/balance/refresh", s.walletBalance)
	mux.HandleFunc("GET /api/v1/wallet/recipients/search", s.walletRecipients)
	mux.HandleFunc("POST /api/v1/wallet/transfers", s.walletTransferCreate)
	mux.HandleFunc("GET /api/v1/wallet/transfers/{transferId}", s.walletTransferGet)
	mux.HandleFunc("GET /api/v1/wallet/records", s.walletRecords)
	mux.HandleFunc("GET /api/v1/admin/transactions", func(w http.ResponseWriter, r *http.Request) { s.walletLedgerHTTPV204(w, r, true) })
	mux.HandleFunc("GET /api/v1/wallet/records/{recordId}", s.walletRecordGet)
}
func (s *Server) walletBalance(w http.ResponseWriter, r *http.Request) {
	u, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	node, err := s.economyNode()
	if err != nil {
		failError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	op, err := s.coreCall(ctx, u.User.ID, store.ID("balance_"), node, "wallet.balance", map[string]string{"playerUuid": u.User.ServerUUID})
	var balance map[string]any
	if err == nil {
		err = CoreResult(op, &balance)
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	balance["fresh"] = true
	balance["availableAmount"] = balance["amount"]
	balance["serverNow"] = time.Now().UTC()
	if balance["heldAmount"] == nil || balance["heldBreakdown"] == nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	success(w, r, map[string]any{"balance": balance})
}
func (s *Server) walletRecipients(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authenticate(r); err != nil {
		failError(w, r, err)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	kind := r.URL.Query().Get("type")
	if kind == "" || kind == "auto" {
		kind = "game_id"
		if identity.ValidQQ(query) {
			kind = "qq"
		}
	}
	if !identity.ValidGameID(query) || identity.SystemGameID(query) || (kind != "game_id" && kind != "qq" && kind != "search") {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	list, err := s.Store.WalletRecipientSearch(ctx, query, kind)
	if err != nil {
		failError(w, r, err)
		return
	}
	if len(list) == 0 && kind == "game_id" {
		p, e := s.resolveCorePlayer(ctx, query, false)
		if e == nil {
			ref, e := s.Store.RememberCorePlayer(ctx, p)
			if e == nil {
				candidate, e := s.Store.WalletRecipient(ctx, ref)
				if e == nil {
					list = append(list, candidate)
				}
			}
		} else {
			var ce *CoreError
			if !errors.As(e, &ce) || ce.Code != "PLAYER_NOT_FOUND" {
				failError(w, r, e)
				return
			}
		}
	}
	filtered := []store.Recipient{}
	seen := map[string]bool{}
	for _, p := range list {
		if identity.SystemGameID(p.GameID) || seen[p.UUID] {
			continue
		}
		seen[p.UUID] = true
		live, _ := s.Core.Player("", p.UUID)
		p.Online = live.Online
		filtered = append(filtered, p)
	}
	success(w, r, map[string]any{"candidates": filtered})
}
func (s *Server) walletTransferCreate(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	var in struct {
		ClientRequestID    string  `json:"clientRequestId"`
		RecipientPlayerRef string  `json:"recipientPlayerRef"`
		Amount             string  `json:"amount"`
		Note               *string `json:"note"`
	}
	if body(w, r, &in) != nil || !bridge.ValidMessageID(in.ClientRequestID) || (in.Note != nil && (utf8.RuneCountInString(*in.Note) > 80 || len(*in.Note) > 320)) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	amount, err := amountText(in.Amount)
	if err != nil {
		failError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	p, err := s.Store.WalletRecipient(ctx, in.RecipientPlayerRef)
	if err != nil || p.UUID == session.User.ServerUUID || identity.SystemGameID(p.GameID) {
		failure(w, r, 400, "RECIPIENT_INVALID", "收款人无效或不能向该账号转账。")
		return
	}
	node, err := s.economyNode()
	if err != nil {
		failError(w, r, err)
		return
	}
	id, _, err := s.Store.CreateWalletTransfer(ctx, session.User, in.ClientRequestID, node, p, amount, in.Note)
	if err != nil {
		failError(w, r, err)
		return
	}
	t, err := s.Store.WalletTransfer(ctx, id)
	if err != nil {
		failError(w, r, err)
		return
	}
	op, err := s.Store.CoreOperation(ctx, t.OperationID)
	if err == nil && op.State == "QUEUED" && s.Core.Online(op.NodeID) {
		_ = s.dispatchCore(op)
		s.awaitCore(ctx, op.ID)
	}
	t, err = s.Store.WalletTransfer(ctx, id)
	if err != nil {
		failError(w, r, err)
		return
	}
	s.transferResponse(w, r, t)
}
func (s *Server) awaitCore(ctx context.Context, id string) {
	wake, done := s.Core.Watch(id)
	defer done()
	timer := time.NewTicker(250 * time.Millisecond)
	defer timer.Stop()
	for {
		op, err := s.Store.CoreOperation(ctx, id)
		if err != nil || op.State == "COMPLETED" || op.State == "FAILED" || op.State == "UNKNOWN" {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-wake:
		case <-timer.C:
		}
	}
}
func (s *Server) transferResponse(w http.ResponseWriter, r *http.Request, t store.WalletTransfer) {
	status := 200
	if t.Status == "processing" || t.Status == "unknown" {
		status = 202
	}
	writeJSON(w, status, map[string]any{"requestId": requestID(r), "data": map[string]any{"transfer": t}})
}
func (s *Server) walletTransferGet(w http.ResponseWriter, r *http.Request) {
	u, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	t, err := s.Store.WalletTransfer(r.Context(), r.PathValue("transferId"))
	if err != nil || t.ActorID != u.User.ID {
		failure(w, r, 404, "NOT_FOUND", "转账不存在。")
		return
	}
	if t.Status == "unknown" || t.Status == "processing" {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		_ = s.reconcileCoreOperation(ctx, t.OperationID)
		cancel()
		if refreshed, e := s.Store.WalletTransfer(r.Context(), t.ID); e == nil {
			t = refreshed
		}
	}
	s.transferResponse(w, r, t)
}

// Read-only recovery never reruns a monetary command, even when the connection
// broke between provider success and delivery of the command result.
func (s *Server) reconcileCoreOperation(ctx context.Context, id string) error {
	original, err := s.Store.CoreOperation(ctx, id)
	if err != nil {
		return err
	}
	if original.State == "COMPLETED" || original.State == "FAILED" {
		return nil
	}
	if original.State == "QUEUED" {
		return nil
	}
	query, err := s.coreCall(ctx, "core-reconcile", store.ID("lookup_"), original.NodeID, "operation.query", map[string]string{"operationId": id})
	if err != nil {
		return err
	}
	var entry struct {
		Acquired bool    `json:"acquired"`
		State    string  `json:"state"`
		Result   *string `json:"result"`
	}
	if err = CoreResult(query, &entry); err != nil {
		return err
	}
	if entry.Result == nil || (entry.State != "COMPLETED" && entry.State != "FAILED") {
		return nil
	}
	var reply struct {
		OperationID string          `json:"operationId"`
		Status      string          `json:"status"`
		Data        json.RawMessage `json:"data"`
		Error       json.RawMessage `json:"error"`
	}
	if json.Unmarshal([]byte(*entry.Result), &reply) != nil || reply.OperationID != id || reply.Status != entry.State {
		return bridge.ErrProtocol
	}
	node, err := s.node(original.NodeID)
	if err != nil {
		return err
	}
	if err = s.acceptCoreReceiptV2(ctx, node, original, entry.State, reply.Data); err != nil {
		return err
	}
	return s.Store.CoreReply(ctx, original.NodeID, id, entry.State, []byte(*entry.Result))
}
func (s *Server) walletRecords(w http.ResponseWriter, r *http.Request) {
	s.walletLedgerHTTPV204(w, r, false)
}
func (s *Server) transferRecord(ctx context.Context, u store.User, t store.WalletTransfer, detail bool) (map[string]any, error) {
	direction := "expense"
	other := t.Recipient
	if u.ServerUUID == t.ToUUID {
		direction = "income"
		var ref string
		if err := s.Store.DB.QueryRowContext(ctx, "SELECT player_ref FROM identities WHERE id=?", t.ActorID).Scan(&ref); err != nil {
			return nil, err
		}
		p, err := s.Store.WalletRecipient(ctx, ref)
		if err != nil {
			return nil, err
		}
		other = p
	}
	result := map[string]any{"recordId": t.ID, "direction": direction, "amount": t.Amount, "currency": "CREDIT", "status": t.Status, "note": t.Note, "occurredAt": t.CreatedAt, "businessType": "TRANSFER", "businessRef": t.ID, "otherPlayer": other}
	if detail {
		result["status"] = strings.ToUpper(t.Status)
		result["counterparty"] = map[string]any{"kind": "PLAYER", "playerRef": other.PlayerRef, "storeId": nil, "displayName": other.GameID, "contactQq": other.QQ}
	}
	return result, nil
}
func (s *Server) walletRecordGet(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.PathValue("recordId"), "econ_") {
		s.walletLedgerDetailV204(w, r)
		return
	}
	u, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	t, err := s.Store.WalletTransfer(r.Context(), r.PathValue("recordId"))
	if err != nil || (u.User.ServerUUID != t.FromUUID && u.User.ServerUUID != t.ToUUID) {
		failure(w, r, 404, "NOT_FOUND", "账单不存在。")
		return
	}
	record, err := s.transferRecord(r.Context(), u.User, t, true)
	if err != nil {
		failError(w, r, err)
		return
	}
	success(w, r, record)
}
