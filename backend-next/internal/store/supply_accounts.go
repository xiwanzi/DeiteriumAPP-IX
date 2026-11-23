package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

var supplyUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type SupplyAccount struct {
	PlayerUUID string                 `json:"playerUuid"`
	GameID     string                 `json:"gameId"`
	Standard   int64                  `json:"standard"`
	Limited    int64                  `json:"limited"`
	Balances   map[string]int64       `json:"balances"`
	Version    int64                  `json:"version"`
	Pity       map[string]supply.Pity `json:"pity"`
}

type SupplyAdjustment struct {
	ClientRequestID string `json:"clientRequestId"`
	PlayerUUID      string `json:"playerUuid"`
	Currency        string `json:"currency"`
	Action          string `json:"action"`
	Amount          int64  `json:"amount"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Reason          string `json:"reason"`
}

func supplyFailure(code, message string) error { return catalogError(400, code, message) }

// Supply requests also belong to UUID-only game players without a website account.
// The account row serializes draws and adjustments; request fingerprints exclude transport sessions.
func (s *Store) supplyMutate(ctx context.Context, actor, key string, input any, player, name string, run func(*sql.Tx) (any, error)) (json.RawMessage, error) {
	if actor == "" || len(actor) > 80 || !supply.ValidID(key) || player != "" && !supplyUUID.MatchString(player) {
		return nil, ErrSocialInvalid
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	fingerprint := Digest(encoded)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if player != "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO supply_accounts(player_uuid,game_id,updated_at) VALUES(?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE game_id=IF(VALUES(game_id)='',game_id,VALUES(game_id))`, player, name); err != nil {
			return nil, err
		}
		var version int64
		if err = tx.QueryRowContext(ctx, "SELECT version FROM supply_accounts WHERE player_uuid=? FOR UPDATE", player).Scan(&version); err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO supply_requests(actor_id,request_id,fingerprint,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE request_id=VALUES(request_id)`, actor, key, fingerprint); err != nil {
		return nil, err
	}
	var old string
	var response sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT fingerprint,response_json FROM supply_requests WHERE actor_id=? AND request_id=? FOR UPDATE", actor, key).Scan(&old, &response); err != nil {
		return nil, err
	}
	if old != fingerprint {
		return nil, ErrConflict
	}
	if response.Valid {
		return json.RawMessage(response.String), nil
	}
	result, err := run(tx)
	if err != nil {
		return nil, err
	}
	encoded, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE supply_requests SET response_json=? WHERE actor_id=? AND request_id=?", encoded, actor, key); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return encoded, nil
}

func supplyAccountTx(ctx context.Context, tx *sql.Tx, uuid string) (SupplyAccount, error) {
	a := SupplyAccount{PlayerUUID: uuid, Pity: map[string]supply.Pity{}, Balances: map[string]int64{}}
	err := tx.QueryRowContext(ctx, "SELECT game_id,standard_tickets,limited_tickets,version FROM supply_accounts WHERE player_uuid=?", uuid).Scan(&a.GameID, &a.Standard, &a.Limited, &a.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return a, nil
	}
	if err != nil {
		return a, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT scope,misses,target,cycle FROM supply_pity WHERE player_uuid=?", uuid)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	for rows.Next() {
		var scope string
		var p supply.Pity
		if err = rows.Scan(&scope, &p.Count, &p.Target, &p.Cycle); err != nil {
			return a, err
		}
		a.Pity[scope] = p
	}
	if err = rows.Err(); err != nil {
		return a, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT currency,balance FROM supply_custom_ticket_balances WHERE player_uuid=?", uuid)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	for rows.Next() {
		var currency string
		var balance int64
		if err = rows.Scan(&currency, &balance); err != nil {
			return a, err
		}
		a.Balances[currency] = balance
	}
	return a, rows.Err()
}

func (s *Store) SupplyAccount(ctx context.Context, uuid string) (SupplyAccount, error) {
	if !supplyUUID.MatchString(uuid) {
		return SupplyAccount{}, ErrSocialInvalid
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return SupplyAccount{}, err
	}
	defer tx.Rollback()
	a, err := supplyAccountTx(ctx, tx, uuid)
	if err != nil {
		return a, err
	}
	return a, tx.Commit()
}

func supplyBalance(ctx context.Context, tx *sql.Tx, uuid, currency string) (int64, error) {
	var value int64
	if !supply.ValidCurrency(currency) {
		return 0, ErrSocialInvalid
	}
	if !supply.LegacyCurrency(currency) {
		err := tx.QueryRowContext(ctx, "SELECT balance FROM supply_custom_ticket_balances WHERE player_uuid=? AND currency=?", uuid, currency).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return value, err
	}
	column := "standard_tickets"
	if currency == "LIMITED" {
		column = "limited_tickets"
	}
	err := tx.QueryRowContext(ctx, "SELECT "+column+" FROM supply_accounts WHERE player_uuid=?", uuid).Scan(&value)
	return value, err
}

func supplyPost(ctx context.Context, tx *sql.Tx, uuid, currency string, amount int64, kind, business, actor, reason string) error {
	before, err := supplyBalance(ctx, tx, uuid, currency)
	if err != nil {
		return err
	}
	after := before + amount
	if after < 0 {
		return supplyFailure("TICKETS_INSUFFICIENT", "抽卡券不足")
	}
	if amount > supply.MaxBalance || amount < -supply.MaxBalance || after > supply.MaxBalance {
		return supplyFailure("TICKETS_LIMIT", "票券数量超出上限")
	}
	if err = saveSupplyBalance(ctx, tx, uuid, currency, after); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO supply_ledger(player_uuid,currency,amount,before_balance,after_balance,business_type,business_id,actor_id,reason,created_at) VALUES(?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, uuid, currency, amount, before, after, kind, business, actor, reason)
	return err
}

func saveSupplyBalance(ctx context.Context, tx *sql.Tx, uuid, currency string, balance int64) error {
	if !supply.LegacyCurrency(currency) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO supply_custom_ticket_balances(player_uuid,currency,balance)
			VALUES(?,?,?) ON DUPLICATE KEY UPDATE balance=VALUES(balance)`, uuid, currency, balance); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE supply_accounts SET version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE player_uuid=?", uuid)
		return err
	}
	column := "standard_tickets"
	if currency == "LIMITED" {
		column = "limited_tickets"
	}
	_, err := tx.ExecContext(ctx, "UPDATE supply_accounts SET "+column+"=?,version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE player_uuid=?", balance, uuid)
	return err
}

func (s *Store) AdjustSupplyTickets(ctx context.Context, actor string, in SupplyAdjustment, webAdmin bool) (json.RawMessage, error) {
	if !supply.ValidCurrency(in.Currency) || in.Amount < 0 || in.Amount > supply.MaxBalance || !ValidSocialText(in.Reason, 1, 200) || !catalogEnum(in.Action, "add", "take", "remove", "set") {
		return nil, ErrSocialInvalid
	}
	return s.supplyMutate(ctx, actor, in.ClientRequestID, struct {
		Action string
		Input  SupplyAdjustment
	}{"tickets.adjust", in}, in.PlayerUUID, "", func(tx *sql.Tx) (any, error) {
		if webAdmin {
			if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
				return nil, err
			}
		}
		if !supply.LegacyCurrency(in.Currency) {
			tickets, err := supplyTickets(ctx, tx, SupplySettings{})
			if err != nil {
				return nil, err
			}
			if _, ok := tickets[in.Currency]; !ok {
				return nil, supplyFailure("SUPPLY_TICKET_UNKNOWN", "抽奖券未发布")
			}
		}
		account, err := supplyAccountTx(ctx, tx, in.PlayerUUID)
		if err != nil {
			return nil, err
		}
		if in.ExpectedVersion > 0 && account.Version != in.ExpectedVersion {
			return nil, ErrConflict
		}
		amount := in.Amount
		if in.Action == "take" || in.Action == "remove" {
			amount = -amount
		}
		if in.Action == "set" {
			before, e := supplyBalance(ctx, tx, in.PlayerUUID, in.Currency)
			if e != nil {
				return nil, e
			}
			amount -= before
		}
		if amount > 0 {
			before, e := supplyBalance(ctx, tx, in.PlayerUUID, in.Currency)
			if e != nil {
				return nil, e
			}
			var reserved int64
			if e = tx.QueryRowContext(ctx, "SELECT COALESCE(SUM(quantity),0) FROM supply_exchanges WHERE player_uuid=? AND currency=? AND status IN ('PENDING','RESERVED','REVIEW')", in.PlayerUUID, in.Currency).Scan(&reserved); e != nil {
				return nil, e
			}
			if before+reserved+amount > supply.MaxBalance {
				return nil, supplyFailure("TICKETS_LIMIT", "余额和待入账兑换将超过票券上限")
			}
		}
		id := in.ClientRequestID
		if err = supplyPost(ctx, tx, in.PlayerUUID, in.Currency, amount, "ADMIN", id, actor, in.Reason); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", actor, "supply.tickets."+in.Action, id); err != nil {
			return nil, err
		}
		return supplyAccountTx(ctx, tx, in.PlayerUUID)
	})
}
