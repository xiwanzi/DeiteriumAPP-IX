package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

type SupplyExchangeRequest struct {
	ClientRequestID string `json:"clientRequestId"`
	Currency        string `json:"currency"`
	Quantity        int64  `json:"quantity"`
	SettingsVersion int64  `json:"settingsVersion"`
}

type SupplyExchange struct {
	ID         string    `json:"id"`
	PlayerUUID string    `json:"playerUuid"`
	Currency   string    `json:"currency"`
	Quantity   int64     `json:"quantity"`
	UnitPrice  int64     `json:"unitPrice"`
	TotalPrice int64     `json:"totalPrice"`
	Status     string    `json:"status"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (s *Store) CreateSupplyExchange(ctx context.Context, uuid, name string, in SupplyExchangeRequest) (json.RawMessage, error) {
	if !supply.ValidCurrency(in.Currency) || in.Quantity < 1 || in.Quantity > 10000 || in.SettingsVersion < 1 {
		return nil, ErrSocialInvalid
	}
	return s.supplyMutate(ctx, uuid, in.ClientRequestID, struct {
		Action string
		Input  SupplyExchangeRequest
	}{"exchange", in}, uuid, name, func(tx *sql.Tx) (any, error) {
		var settings SupplySettings
		if err := tx.QueryRowContext(ctx, "SELECT version,standard_price,limited_price FROM supply_settings WHERE id=1 LOCK IN SHARE MODE").Scan(&settings.Version, &settings.StandardPrice, &settings.LimitedPrice); err != nil {
			return nil, err
		}
		if settings.Version != in.SettingsVersion {
			return nil, catalogError(409, "SUPPLY_PRICE_CHANGED", "兑换价格已更新，请重新确认")
		}
		price := settings.StandardPrice
		if in.Currency == "LIMITED" {
			price = settings.LimitedPrice
		}
		if price <= 0 {
			return nil, supplyFailure("SUPPLY_EXCHANGE_DISABLED", "该票券暂未开放信用点兑换")
		}
		balance, err := supplyBalance(ctx, tx, uuid, in.Currency)
		if err != nil {
			return nil, err
		}
		var reserved int64
		if err = tx.QueryRowContext(ctx, "SELECT COALESCE(SUM(quantity),0) FROM supply_exchanges WHERE player_uuid=? AND currency=? AND status IN ('PENDING','RESERVED','REVIEW')", uuid, in.Currency).Scan(&reserved); err != nil {
			return nil, err
		}
		if balance+reserved+in.Quantity > supply.MaxBalance {
			return nil, supplyFailure("TICKETS_LIMIT", "余额和待入账票券将超过上限")
		}
		x := SupplyExchange{ID: ID("sx_"), PlayerUUID: uuid, Currency: in.Currency, Quantity: in.Quantity, UnitPrice: price, TotalPrice: price * in.Quantity, Status: "PENDING", CreatedAt: time.Now().UTC()}
		_, err = tx.ExecContext(ctx, `INSERT INTO supply_exchanges(id,player_uuid,currency,quantity,unit_price,total_price,status,created_at,updated_at) VALUES(?,?,?,?,?,?,'PENDING',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, x.ID, uuid, in.Currency, in.Quantity, price, x.TotalPrice)
		return x, err
	})
}

func scanSupplyExchange(row interface{ Scan(...any) error }) (SupplyExchange, error) {
	var x SupplyExchange
	err := row.Scan(&x.ID, &x.PlayerUUID, &x.Currency, &x.Quantity, &x.UnitPrice, &x.TotalPrice, &x.Status, &x.Error, &x.CreatedAt)
	return x, err
}

const supplyExchangeColumns = "id,player_uuid,currency,quantity,unit_price,total_price,status,error_message,created_at"

func (s *Store) SupplyExchange(ctx context.Context, id, uuid string) (SupplyExchange, error) {
	return scanSupplyExchange(s.DB.QueryRowContext(ctx, "SELECT "+supplyExchangeColumns+" FROM supply_exchanges WHERE id=? AND player_uuid=?", id, uuid))
}

func (s *Store) PendingSupplyExchanges(ctx context.Context) ([]SupplyExchange, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+supplyExchangeColumns+" FROM supply_exchanges WHERE status IN ('PENDING','RESERVED') ORDER BY updated_at LIMIT 16")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SupplyExchange{}
	for rows.Next() {
		x, e := scanSupplyExchange(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) UpdateSupplyExchange(ctx context.Context, id, expected, next, message string) error {
	if !catalogEnum(expected, "PENDING", "RESERVED") || !catalogEnum(next, "RESERVED", "FAILED", "REVIEW") || len(message) > 512 {
		return ErrSocialInvalid
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE supply_exchanges SET status=?,error_message=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND status=?", next, message, id, expected)
	return err
}

func (s *Store) CompleteSupplyExchange(ctx context.Context, x SupplyExchange) error {
	_, err := s.supplyMutate(ctx, "supply-settlement", x.ID, struct{ ID string }{x.ID}, x.PlayerUUID, "", func(tx *sql.Tx) (any, error) {
		current, e := scanSupplyExchange(tx.QueryRowContext(ctx, "SELECT "+supplyExchangeColumns+" FROM supply_exchanges WHERE id=? FOR UPDATE", x.ID))
		if e != nil {
			return nil, e
		}
		if current.PlayerUUID != x.PlayerUUID {
			return nil, ErrConflict
		}
		if current.Status == "COMPLETED" {
			return current, nil
		}
		if current.Status != "RESERVED" {
			return nil, ErrConflict
		}
		if e = supplyPost(ctx, tx, x.PlayerUUID, current.Currency, current.Quantity, "EXCHANGE", current.ID, "XCONOMY", "信用点兑换抽卡券"); e != nil {
			return nil, e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE supply_exchanges SET status='COMPLETED',error_message='',updated_at=UTC_TIMESTAMP(6) WHERE id=?", current.ID); e != nil {
			return nil, e
		}
		current.Status = "COMPLETED"
		return current, nil
	})
	return err
}

func (s *Store) SupplyExchangeProgress(ctx context.Context, id, stage, operation, message string) error {
	if !catalogEnum(stage, "PENDING", "RESERVED") || len(message) > 512 {
		return ErrSocialInvalid
	}
	column := "reserve_operation"
	if stage == "RESERVED" {
		column = "settle_operation"
	}
	if operation == "" {
		_, err := s.DB.ExecContext(ctx, "UPDATE supply_exchanges SET error_message=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND status=?", message, id, stage)
		return err
	}
	result, err := s.DB.ExecContext(ctx, "UPDATE supply_exchanges SET "+column+"=?,error_message=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND status=? AND ("+column+" IS NULL OR "+column+"=?)", operation, message, id, stage, operation)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrConflict
	}
	return nil
}
