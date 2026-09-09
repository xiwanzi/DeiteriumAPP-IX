package httpapi

import (
	"context"
	"database/sql"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"time"
)

// Native /pay also uses the committed XConomy ledger. A durable page cursor
// catches up across restarts; a short overlap tolerates commit timing at a page
// boundary. Stable operation keys deduplicate the overlap.
func (s *Server) walletNoticeWorkerV206() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
			_ = s.walletNoticesV206(ctx, time.Now().UTC().Add(-2*time.Second))
			cancel()
		}
	}
}
func (s *Server) walletNoticesV206(ctx context.Context, now time.Time) error {
	f := ledgerFilterV204{BeforeSequence: "0", Snapshot: "0", Direction: "income"}
	var until, before sql.NullTime
	var version int64
	err := s.Store.DB.QueryRowContext(ctx, "SELECT from_time,until_time,before_time,before_sequence,snapshot,version FROM wallet_notice_cursor_v206 WHERE id=1").Scan(&f.From, &until, &before, &f.BeforeSequence, &f.Snapshot, &version)
	if err != nil {
		return err
	}
	if until.Valid {
		f.To = until.Time
	} else {
		f.To = now
		if f.To.After(f.From.Add(24 * time.Hour)) {
			f.To = f.From.Add(24 * time.Hour)
		}
	}
	if !f.From.Before(f.To) {
		return nil
	}
	if before.Valid {
		f.BeforeTime = before.Time
	}
	page, err := s.readLedgerV204(ctx, "wallet-notifications", "", f, 25, "")
	if err != nil {
		return err
	}
	tx, err := s.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var actual int64
	if err = tx.QueryRowContext(ctx, "SELECT version FROM wallet_notice_cursor_v206 WHERE id=1 FOR UPDATE").Scan(&actual); err != nil {
		return err
	}
	if actual != version {
		return nil
	}
	rows := page.Records
	more := len(rows) > 25
	if more {
		rows = rows[:25]
	}
	for _, row := range rows {
		if row.Source != "GAME" || row.BusinessType != "TRANSFER" || row.Direction != "income" || row.OtherUUID == nil || row.OperationID == "" {
			continue
		}
		var user string
		err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE server_uuid=? AND status='active'", row.PlayerUUID).Scan(&user)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		sender := "一位玩家"
		var name string
		err = tx.QueryRowContext(ctx, "SELECT game_id FROM identities WHERE server_uuid=? UNION SELECT game_id FROM core_player_directory WHERE player_uuid=? LIMIT 1", *row.OtherUUID, *row.OtherUUID).Scan(&name)
		if err == nil {
			sender = name
		} else if err != sql.ErrNoRows {
			return err
		}
		if err = store.AddNotificationV2(ctx, tx, user, "game-transfer:"+row.OperationID, "WALLET", "转账到账", sender+" 向你转账 "+row.Amount+" 信用点，已到账。", store.SocialTarget{Kind: "WALLET", ReferenceID: "econ_" + row.Sequence, StateVersion: 1}); err != nil {
			return err
		}
	}
	if more {
		last := rows[len(rows)-1]
		_, err = tx.ExecContext(ctx, "UPDATE wallet_notice_cursor_v206 SET until_time=?,before_time=?,before_sequence=?,snapshot=?,version=version+1 WHERE id=1", f.To, last.OccurredAt, last.Sequence, page.Snapshot)
	} else {
		next := f.To
		if f.To.Sub(f.From) > time.Minute {
			next = next.Add(-time.Minute)
		}
		_, err = tx.ExecContext(ctx, "UPDATE wallet_notice_cursor_v206 SET from_time=?,until_time=NULL,before_time=NULL,before_sequence='0',snapshot='0',version=version+1 WHERE id=1", next)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
