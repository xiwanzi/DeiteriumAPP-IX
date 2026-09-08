package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// ExportLegacy only reads the old app_users table in a read-only snapshot.
// No sessions, OIDC clients, verification codes or economic data are exported.
func ExportLegacy(ctx context.Context, source *store.Store, out io.Writer) (int, error) {
	tx, err := source.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return 0, errors.New("legacy read-only transaction unavailable")
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,server_uuid,current_game_id,qq,password_hash,status,created_at,updated_at FROM app_users ORDER BY id`)
	if err != nil {
		return 0, errors.New("legacy account query failed")
	}
	defer rows.Close()
	encoder := json.NewEncoder(out)
	count := 0
	for rows.Next() {
		var u LegacyUser
		if err = rows.Scan(&u.ID, &u.ServerUUID, &u.GameID, &u.QQ, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return 0, errors.New("legacy record could not be read")
		}
		if err = validateLegacy([]LegacyUser{u}); err != nil {
			return 0, errors.New("legacy export contains an unsupported identity or hash")
		}
		if count >= 100000 {
			return 0, errors.New("legacy export exceeds supported record limit")
		}
		if err = encoder.Encode(u); err != nil {
			return 0, errors.New("legacy export write failed")
		}
		count++
	}
	if rows.Err() != nil {
		return 0, errors.New("legacy export interrupted")
	}
	return count, nil
}
