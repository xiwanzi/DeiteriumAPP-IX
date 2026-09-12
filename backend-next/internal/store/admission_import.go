package store

import (
	"context"
	"database/sql"
	"errors"
)

type AdmissionImportPlayer struct {
	UUID   string `json:"uuid"`
	GameID string `json:"gameId"`
	QQ     string `json:"qq"`
}
type AdmissionImportResult struct {
	Candidates int  `json:"candidates"`
	New        int  `json:"new"`
	Existing   int  `json:"existing"`
	Applied    bool `json:"applied"`
}

// This local operator command never reactivates a previously revoked entry.
func (s *Store) ImportAdmission(ctx context.Context, players []AdmissionImportPlayer, apply bool) (AdmissionImportResult, error) {
	out := AdmissionImportResult{Candidates: len(players), Applied: apply}
	if len(players) == 0 || len(players) > 50000 {
		return out, ErrSocialInvalid
	}
	seen := map[string]bool{}
	for _, p := range players {
		if !ValidAdmissionUUID(p.UUID) || !ValidAdmissionName(p.GameID) || (p.QQ != "" && !admissionQQ.MatchString(p.QQ)) || seen[p.UUID] {
			return out, ErrSocialInvalid
		}
		seen[p.UUID] = true
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = lockAdmission(ctx, tx); err != nil {
		return out, err
	}
	for _, p := range players {
		var version int64
		err = tx.QueryRowContext(ctx, "SELECT version FROM admission_entries WHERE server_uuid=?", p.UUID).Scan(&version)
		if err == nil {
			out.Existing++
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return out, err
		}
		out.New++
		if !apply {
			continue
		}
		if _, err = grantAdmission(ctx, tx, AdmissionProfile{UUID: p.UUID, Name: p.GameID}, p.QQ, "IMPORT", "", "上线前已有游戏玩家"); err != nil {
			return out, err
		}
		if err = admissionEvent(ctx, tx, "legacy-player-import", p.UUID, "import", "", "上线前已有游戏玩家身份迁移"); err != nil {
			return out, err
		}
	}
	if !apply {
		return out, nil
	}
	return out, tx.Commit()
}
