package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

type SupplyDrawRequest struct {
	ClientRequestID string `json:"clientRequestId"`
	PoolID          string `json:"poolId"`
	Revision        int64  `json:"revision"`
	Scheme          string `json:"scheme"`
	Quantity        int    `json:"quantity"`
}

type SupplyOrder struct {
	ID         string        `json:"id"`
	Sequence   int64         `json:"sequence"`
	PlayerUUID string        `json:"playerUuid"`
	PoolID     string        `json:"poolId"`
	PoolName   string        `json:"poolName"`
	Revision   int64         `json:"revision"`
	Scheme     string        `json:"scheme"`
	Currency   string        `json:"currency"`
	Quantity   int           `json:"quantity"`
	Scope      string        `json:"scope"`
	Rolls      []supply.Roll `json:"rolls"`
	CreatedAt  time.Time     `json:"createdAt"`
}

func (s *Store) DrawSupply(ctx context.Context, uuid, name string, in SupplyDrawRequest) (json.RawMessage, error) {
	return s.drawSupply(ctx, uuid, name, in, supply.SecureRandom)
}

func (s *Store) drawSupply(ctx context.Context, uuid, name string, in SupplyDrawRequest, random supply.Random) (json.RawMessage, error) {
	if !supply.ValidID(in.PoolID) || in.Revision < 1 || (in.Quantity != 1 && in.Quantity != 10) {
		return nil, ErrSocialInvalid
	}
	return s.supplyMutate(ctx, uuid, in.ClientRequestID, struct {
		Action string
		Input  SupplyDrawRequest
	}{"draw", in}, uuid, name, func(tx *sql.Tx) (any, error) {
		var revision int64
		var enabled bool
		var raw string
		if err := tx.QueryRowContext(ctx, `SELECT p.published_revision,p.enabled,r.definition_json FROM supply_pools p JOIN supply_revisions r ON r.pool_id=p.id AND r.revision=p.published_revision WHERE p.id=? LOCK IN SHARE MODE`, in.PoolID).Scan(&revision, &enabled, &raw); err != nil {
			return nil, socialMissing(err)
		}
		if revision != in.Revision {
			return nil, catalogError(409, "SUPPLY_CHANGED", "卡池已更新，请刷新后重新确认")
		}
		var d supply.Definition
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			return nil, err
		}
		var now time.Time
		if err := tx.QueryRowContext(ctx, "SELECT UTC_TIMESTAMP(6)").Scan(&now); err != nil {
			return nil, err
		}
		if !enabled || !d.Open(now) {
			return nil, supplyFailure("SUPPLY_CLOSED", "卡池尚未开放、已结束或已暂停")
		}
		rule, scope, currency, err := d.Rule(in.Scheme)
		if err != nil {
			return nil, ErrSocialInvalid
		}
		if err = supplyPost(ctx, tx, uuid, currency, -int64(in.Quantity), "DRAW", in.ClientRequestID, uuid, d.Name); err != nil {
			return nil, err
		}
		var state supply.Pity
		err = tx.QueryRowContext(ctx, "SELECT misses,target,cycle FROM supply_pity WHERE player_uuid=? AND scope=?", uuid, scope).Scan(&state.Count, &state.Target, &state.Cycle)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		rolls, next, err := supply.Draw(rule, state, d.Rewards, in.Quantity, random)
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO supply_pity(player_uuid,scope,misses,target,cycle) VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE misses=VALUES(misses),target=VALUES(target),cycle=VALUES(cycle)`, uuid, scope, next.Count, next.Target, next.Cycle); err != nil {
			return nil, err
		}
		order := SupplyOrder{ID: ID("sd_"), PlayerUUID: uuid, PoolID: in.PoolID, PoolName: d.Name, Revision: revision, Scheme: in.Scheme, Currency: currency, Quantity: in.Quantity, Scope: scope, Rolls: rolls, CreatedAt: now}
		rollJSON, err := json.Marshal(rolls)
		if err != nil {
			return nil, err
		}
		ruleJSON, err := json.Marshal(rule)
		if err != nil {
			return nil, err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO supply_orders(id,player_uuid,pool_id,pool_name,revision,scheme,currency,quantity,scope,rule_json,definition_json,result_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, order.ID, uuid, in.PoolID, d.Name, revision, in.Scheme, currency, in.Quantity, scope, ruleJSON, raw, rollJSON, now)
		if err != nil {
			return nil, err
		}
		order.Sequence, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}
		// The ledger uses the public order ID so an operator can follow the same receipt in both views.
		if _, err = tx.ExecContext(ctx, "UPDATE supply_ledger SET business_id=? WHERE player_uuid=? AND currency=? AND business_id=? AND business_type='DRAW'", order.ID, uuid, currency, in.ClientRequestID); err != nil {
			return nil, err
		}
		for i, roll := range rolls {
			rewardJSON, e := json.Marshal(roll.Reward)
			if e != nil {
				return nil, e
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO supply_deliveries(id,order_id,ordinal,player_uuid,reward_json,state,created_at,updated_at) VALUES(?,?,?,?,?,'PENDING',?,?)`, fmt.Sprintf("%s_%02d", order.ID, i+1), order.ID, i+1, uuid, rewardJSON, now, now); err != nil {
				return nil, err
			}
		}
		return order, nil
	})
}
