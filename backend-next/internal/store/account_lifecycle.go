package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const DeletedAccountName = "已注销用户"

// A shared gate lets ordinary transactions run concurrently, while erasure
// waits for every earlier write and makes all later writes recheck identity.
func (s *Store) beginAccountTx(ctx context.Context, actor string) (*sql.Tx, error) {
	// Status is read before the existing per-resource locks. A current read
	// avoids freezing an older InnoDB snapshot before an idempotent upsert waits.
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	var guard int
	if err = tx.QueryRowContext(ctx, "SELECT id FROM account_lifecycle_guard WHERE id=1 LOCK IN SHARE MODE").Scan(&guard); err == nil && actor != "" {
		var status string
		err = tx.QueryRowContext(ctx, "SELECT status FROM identities WHERE id=?", actor).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) || err == nil && status != "active" {
			err = ErrUnauthorized
		}
	}
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func deletedUUIDTx(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, uuid string) (bool, error) {
	var deleted bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM account_deletions WHERE original_uuid=?) AND NOT EXISTS(SELECT 1 FROM identities WHERE server_uuid=? AND status='active')`, uuid, uuid).Scan(&deleted)
	return deleted, err
}

func (s *Store) DeletedGameIdentity(ctx context.Context, uuid string) (bool, error) {
	return deletedUUIDTx(ctx, s.DB, uuid)
}

// Ledger identity is resolved at the historical event, not assigned to a later
// registration of the same game UUID.
func (s *Store) DeletedWalletParty(ctx context.Context, uuid string, at time.Time) (string, bool, error) {
	var ref string
	err := s.DB.QueryRowContext(ctx, `SELECT player_ref FROM account_deletions WHERE original_uuid=? AND (deleted_at>=? OR NOT EXISTS(SELECT 1 FROM identities WHERE server_uuid=? AND status='active')) ORDER BY sequence_id DESC LIMIT 1`, uuid, at.UTC(), uuid).Scan(&ref)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return ref, err == nil, err
}

type DeletedAccount struct {
	Sequence  int64  `json:"sequence"`
	PlayerRef string `json:"playerRef"`
}

func (s *Store) AccountDeletions(ctx context.Context, after int64, limit int) ([]DeletedAccount, error) {
	if after < 0 || limit < 1 || limit > 101 {
		return nil, ErrSocialInvalid
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT sequence_id,player_ref FROM account_deletions WHERE sequence_id>? ORDER BY sequence_id LIMIT ?", after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeletedAccount{}
	for rows.Next() {
		var d DeletedAccount
		if err = rows.Scan(&d.Sequence, &d.PlayerRef); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Only structured identity fields are anonymized. Monetary evidence and its
// immutable source snapshot are never rewritten by this rendering step.
func redactDeletedPartiesTx(ctx context.Context, tx *sql.Tx, value any) error {
	rows, err := tx.QueryContext(ctx, "SELECT player_ref FROM identities WHERE status='deleted'")
	if err != nil {
		return err
	}
	refs := map[string]bool{}
	for rows.Next() {
		var ref string
		if err = rows.Scan(&ref); err != nil {
			rows.Close()
			return err
		}
		refs[ref] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case CatalogObjectV2:
			visit(map[string]any(v))
		case map[string]any:
			if ref, ok := v["playerRef"].(string); ok && refs[ref] {
				for _, key := range []string{"gameId", "displayName", "name"} {
					if _, ok := v[key]; ok {
						v[key] = DeletedAccountName
					}
				}
				for _, key := range []string{"qq", "bio", "avatar", "contactQq"} {
					if _, ok := v[key]; ok {
						v[key] = nil
					}
				}
				v["deleted"] = true
				if _, ok := v["registered"]; ok {
					v["registered"] = false
				}
			}
			for _, item := range v {
				visit(item)
			}
		case []any:
			for _, item := range v {
				visit(item)
			}
		case []map[string]any:
			for _, item := range v {
				visit(item)
			}
		case []CatalogObjectV2:
			for _, item := range v {
				visit(item)
			}
		}
	}
	visit(value)
	return nil
}
