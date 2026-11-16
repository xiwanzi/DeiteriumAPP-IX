package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

type SupplyFilter struct {
	PlayerUUID string     `json:"playerUuid"`
	PoolID     string     `json:"poolId"`
	Scheme     string     `json:"scheme"`
	Currency   string     `json:"currency"`
	State      string     `json:"state"`
	Before     int64      `json:"before"`
	Snapshot   int64      `json:"snapshot"`
	Limit      int        `json:"limit"`
	From       *time.Time `json:"from"`
	To         *time.Time `json:"to"`
}

func (f SupplyFilter) validate() error {
	if f.PlayerUUID != "" && !supplyUUID.MatchString(f.PlayerUUID) || f.PoolID != "" && !supply.ValidID(f.PoolID) || f.Currency != "" && !supply.ValidCurrency(f.Currency) || f.Before < 0 || f.Snapshot < 0 || f.Limit < 1 || f.Limit > 100 || !catalogEnum(f.Scheme, "", "80", "120", "event") {
		return ErrSocialInvalid
	}
	if f.From != nil && f.To != nil && !f.To.After(*f.From) {
		return ErrSocialInvalid
	}
	return nil
}

func (f SupplyFilter) conditions(alias string) ([]string, []any) {
	conditions := []string{"1=1"}
	args := []any{}
	if f.PlayerUUID != "" {
		conditions = append(conditions, alias+"player_uuid=?")
		args = append(args, f.PlayerUUID)
	}
	if f.Before > 0 {
		conditions = append(conditions, alias+"sequence_id<?")
		args = append(args, f.Before)
	}
	if f.Snapshot > 0 {
		conditions = append(conditions, alias+"sequence_id<=?")
		args = append(args, f.Snapshot)
	}
	if f.From != nil {
		conditions = append(conditions, alias+"created_at>=?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conditions = append(conditions, alias+"created_at<?")
		args = append(args, *f.To)
	}
	if f.Currency != "" {
		conditions = append(conditions, alias+"currency=?")
		args = append(args, f.Currency)
	}
	return conditions, args
}

type SupplyLedgerRow struct {
	Sequence     int64     `json:"sequence"`
	PlayerUUID   string    `json:"playerUuid"`
	GameID       string    `json:"gameId"`
	Currency     string    `json:"currency"`
	Amount       int64     `json:"amount"`
	Before       int64     `json:"before"`
	After        int64     `json:"after"`
	BusinessType string    `json:"businessType"`
	BusinessID   string    `json:"businessId"`
	Actor        string    `json:"actor"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (s *Store) SupplyLedger(ctx context.Context, f SupplyFilter) (map[string]any, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	if f.Snapshot == 0 {
		if err := s.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence_id),0) FROM supply_ledger").Scan(&f.Snapshot); err != nil {
			return nil, err
		}
	}
	where, args := f.conditions("l.")
	args = append(args, f.Limit+1)
	rows, err := s.DB.QueryContext(ctx, `SELECT l.sequence_id,l.player_uuid,COALESCE(a.game_id,''),l.currency,l.amount,l.before_balance,l.after_balance,l.business_type,l.business_id,l.actor_id,l.reason,l.created_at FROM supply_ledger l LEFT JOIN supply_accounts a ON a.player_uuid=l.player_uuid WHERE `+strings.Join(where, " AND ")+" ORDER BY l.sequence_id DESC LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SupplyLedgerRow{}
	for rows.Next() {
		var row SupplyLedgerRow
		if err = rows.Scan(&row.Sequence, &row.PlayerUUID, &row.GameID, &row.Currency, &row.Amount, &row.Before, &row.After, &row.BusinessType, &row.BusinessID, &row.Actor, &row.Reason, &row.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	next := int64(0)
	if len(items) > f.Limit {
		items = items[:f.Limit]
		next = items[len(items)-1].Sequence
	}
	return map[string]any{"items": items, "next": next, "snapshot": f.Snapshot}, rows.Err()
}

func scanSupplyOrder(row interface{ Scan(...any) error }) (SupplyOrder, error) {
	var order SupplyOrder
	var raw string
	err := row.Scan(&order.ID, &order.Sequence, &order.PlayerUUID, &order.PoolID, &order.Revision, &order.Scheme, &order.Currency, &order.Quantity, &order.Scope, &raw, &order.PoolName, &order.CreatedAt)
	if err != nil {
		return order, err
	}
	if err = json.Unmarshal([]byte(raw), &order.Rolls); err != nil {
		return order, err
	}
	return order, nil
}

const supplyOrderColumns = "o.id,o.sequence_id,o.player_uuid,o.pool_id,o.revision,o.scheme,o.currency,o.quantity,o.scope,o.result_json,o.pool_name,o.created_at"

func (s *Store) SupplyOrders(ctx context.Context, f SupplyFilter) (map[string]any, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	if f.Snapshot == 0 {
		if err := s.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence_id),0) FROM supply_orders").Scan(&f.Snapshot); err != nil {
			return nil, err
		}
	}
	where, args := f.conditions("o.")
	if f.PoolID != "" {
		where = append(where, "o.pool_id=?")
		args = append(args, f.PoolID)
	}
	if f.Scheme != "" {
		where = append(where, "o.scheme=?")
		args = append(args, f.Scheme)
	}
	if f.State != "" {
		if !validSupplyDeliveryState(f.State) {
			return nil, ErrSocialInvalid
		}
		where = append(where, "EXISTS(SELECT 1 FROM supply_deliveries d WHERE d.order_id=o.id AND d.state=?)")
		args = append(args, f.State)
	}
	args = append(args, f.Limit+1)
	rows, err := s.DB.QueryContext(ctx, "SELECT "+supplyOrderColumns+" FROM supply_orders o WHERE "+strings.Join(where, " AND ")+" ORDER BY o.sequence_id DESC LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	items := []SupplyOrder{}
	for rows.Next() {
		item, e := scanSupplyOrder(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	next := int64(0)
	if len(items) > f.Limit {
		items = items[:f.Limit]
		next = items[len(items)-1].Sequence
	}
	return map[string]any{"items": items, "next": next, "snapshot": f.Snapshot}, nil
}

func (s *Store) SupplyOrder(ctx context.Context, id, player string) (map[string]any, error) {
	where := "o.id=?"
	args := []any{id}
	if player != "" {
		where += " AND o.player_uuid=?"
		args = append(args, player)
	}
	order, err := scanSupplyOrder(s.DB.QueryRowContext(ctx, "SELECT "+supplyOrderColumns+" FROM supply_orders o WHERE "+where, args...))
	if err != nil {
		return nil, socialMissing(err)
	}
	deliveries, err := s.supplyDeliveries(ctx, "order_id=?", []any{id}, 100)
	return map[string]any{"order": order, "deliveries": deliveries}, err
}

type SupplyDelivery struct {
	ID             string          `json:"id"`
	OrderID        string          `json:"orderId"`
	Ordinal        int             `json:"ordinal"`
	PlayerUUID     string          `json:"playerUuid"`
	Reward         supply.Reward   `json:"reward"`
	State          string          `json:"state"`
	NodeID         string          `json:"nodeId"`
	SessionEpoch   string          `json:"sessionEpoch"`
	InventoryEpoch string          `json:"inventoryEpoch"`
	Token          string          `json:"token"`
	LastError      string          `json:"lastError"`
	Receipt        json.RawMessage `json:"receipt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

const supplyDeliveryColumns = "id,order_id,ordinal,player_uuid,reward_json,state,node_id,session_epoch,inventory_epoch,token,last_error,COALESCE(receipt_json,'null'),updated_at"

func validSupplyDeliveryState(state string) bool {
	return catalogEnum(state, "PENDING", "PREPARING", "ISSUING", "INVENTORY", "MAIL_PENDING", "MAIL", "REVIEW")
}

func scanSupplyDelivery(row interface{ Scan(...any) error }) (SupplyDelivery, error) {
	var d SupplyDelivery
	var reward, receipt string
	err := row.Scan(&d.ID, &d.OrderID, &d.Ordinal, &d.PlayerUUID, &reward, &d.State, &d.NodeID, &d.SessionEpoch, &d.InventoryEpoch, &d.Token, &d.LastError, &receipt, &d.UpdatedAt)
	if err == nil {
		err = json.Unmarshal([]byte(reward), &d.Reward)
		d.Receipt = json.RawMessage(receipt)
	}
	return d, err
}

func (s *Store) supplyDeliveries(ctx context.Context, where string, args []any, limit int) ([]SupplyDelivery, error) {
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, "SELECT "+supplyDeliveryColumns+" FROM supply_deliveries WHERE "+where+" ORDER BY created_at,order_id,ordinal LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SupplyDelivery{}
	for rows.Next() {
		d, e := scanSupplyDelivery(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) PendingSupplyDeliveries(ctx context.Context, players []string) ([]SupplyDelivery, error) {
	if len(players) == 0 {
		return []SupplyDelivery{}, nil
	}
	if len(players) > 256 {
		return nil, ErrSocialInvalid
	}
	args := make([]any, 0, len(players))
	marks := make([]string, len(players))
	for i, p := range players {
		if !supplyUUID.MatchString(p) {
			return nil, ErrSocialInvalid
		}
		args = append(args, p)
		marks[i] = "?"
	}
	return s.pollSupplyDeliveries(ctx, "player_uuid IN ("+strings.Join(marks, ",")+") AND state IN ('PENDING','PREPARING','ISSUING','REVIEW')", args, 32)
}

func (s *Store) SupplyDeliveryQueue(ctx context.Context, state string, limit int) ([]SupplyDelivery, error) {
	if !validSupplyDeliveryState(state) || limit < 1 || limit > 100 {
		return nil, ErrSocialInvalid
	}
	return s.pollSupplyDeliveries(ctx, "state=?", []any{state}, limit)
}

// Rotate polled rows without changing their delivery state, token or proof. A full batch of
// unresolved saves or unavailable items must not starve later rewards indefinitely.
func (s *Store) pollSupplyDeliveries(ctx context.Context, where string, args []any, limit int) ([]SupplyDelivery, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT "+supplyDeliveryColumns+" FROM supply_deliveries WHERE "+where+" ORDER BY updated_at,id LIMIT ? FOR UPDATE SKIP LOCKED", append(args, limit)...)
	if err != nil {
		return nil, err
	}
	out := []SupplyDelivery{}
	ids := []any{}
	marks := []string{}
	for rows.Next() {
		d, e := scanSupplyDelivery(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		out = append(out, d)
		ids = append(ids, d.ID)
		marks = append(marks, "?")
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		if _, err = tx.ExecContext(ctx, "UPDATE supply_deliveries SET updated_at=UTC_TIMESTAMP(6) WHERE id IN ("+strings.Join(marks, ",")+")", ids...); err != nil {
			return nil, err
		}
	}
	return out, tx.Commit()
}

func (s *Store) SupplyDelivery(ctx context.Context, id string) (SupplyDelivery, error) {
	return scanSupplyDelivery(s.DB.QueryRowContext(ctx, "SELECT "+supplyDeliveryColumns+" FROM supply_deliveries WHERE id=?", id))
}

func (s *Store) SupplyPlayerSearch(ctx context.Context, q string) ([]SupplyAccount, error) {
	if len(q) > 80 {
		return nil, ErrSocialInvalid
	}
	q = strings.TrimSpace(q)
	rows, err := s.DB.QueryContext(ctx, "SELECT player_uuid FROM supply_accounts WHERE player_uuid=? OR game_id LIKE ? ORDER BY updated_at DESC LIMIT 25", q, strings.ReplaceAll(strings.ReplaceAll(q, "%", "\\%"), "_", "\\_")+"%")
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []SupplyAccount{}
	for _, id := range ids {
		a, e := s.SupplyAccount(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, nil
}

func supplyRevisionRef(pool string, revision int64) string {
	return fmt.Sprintf("%s_%d", pool, revision)
}
