package identity

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)
var gamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}$`)
var qqPattern = regexp.MustCompile(`^[0-9]{5,20}$`)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidUUID(s string) bool   { return uuidPattern.MatchString(s) }
func ValidGameID(s string) bool { return gamePattern.MatchString(s) }

type LegacyUser struct {
	ID           string    `json:"id"`
	ServerUUID   string    `json:"server_uuid"`
	GameID       string    `json:"current_game_id"`
	QQ           string    `json:"qq"`
	PasswordHash string    `json:"password_hash"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ImportResult struct {
	Total            int  `json:"total"`
	Create           int  `json:"create"`
	Unchanged        int  `json:"unchanged"`
	Applied          bool `json:"applied"`
	SkippedQQAliases int  `json:"skippedQqAliases"`
}

// ReadLegacy accepts a bounded JSONL export, never raw SQL. Credentials stay in
// the local file; error messages expose row numbers, not account data or hashes.
func ReadLegacy(r io.Reader) ([]LegacyUser, error) {
	scanner := bufio.NewScanner(io.LimitReader(r, 64<<20+1))
	scanner.Buffer(make([]byte, 4096), 65536)
	var users []LegacyUser
	total := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		total += len(line) + 1
		if total > 64<<20 || len(users) >= 100000 {
			return nil, errors.New("legacy export exceeds limit")
		}
		if strings.TrimSpace(string(line)) == "" {
			continue
		}
		var u LegacyUser
		dec := json.NewDecoder(strings.NewReader(string(line)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&u); err != nil {
			return nil, fmt.Errorf("invalid legacy JSON at row %d", len(users)+1)
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			return nil, fmt.Errorf("trailing data at row %d", len(users)+1)
		}
		users = append(users, u)
	}
	if scanner.Err() != nil {
		return nil, errors.New("legacy export could not be read")
	}
	if len(users) == 0 {
		return nil, errors.New("empty legacy export")
	}
	return users, nil
}

func validateLegacy(users []LegacyUser) error {
	ids, uuids, aliases := map[string]bool{}, map[string]bool{}, map[string]string{}
	for n, u := range users {
		if !idPattern.MatchString(u.ID) || !ValidUUID(u.ServerUUID) || !ValidGameID(u.GameID) || !validLegacyQQ(u.QQ) || u.CreatedAt.IsZero() || u.UpdatedAt.Before(u.CreatedAt) || ValidateHash(u.PasswordHash) != nil || (u.Status != "active" && u.Status != "disabled" && u.Status != "locked") {
			return fmt.Errorf("unsupported legacy record at row %d", n+1)
		}
		if ids[u.ID] || uuids[u.ServerUUID] {
			return fmt.Errorf("duplicate identity at row %d", n+1)
		}
		ids[u.ID], uuids[u.ServerUUID] = true, true
		for _, alias := range legacyAliases(u) {
			if owner, exists := aliases[alias]; exists && owner != u.ID {
				return fmt.Errorf("ambiguous login alias at row %d", n+1)
			}
			aliases[alias] = u.ID
		}
	}
	return nil
}

func Import(ctx context.Context, s *store.Store, users []LegacyUser, apply bool) (result ImportResult, err error) {
	result.Total = len(users)
	if err = validateLegacy(users); err != nil {
		return
	}
	for _, u := range users {
		if !qqPattern.MatchString(u.QQ) {
			result.SkippedQQAliases++
		}
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: !apply, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return
	}
	defer tx.Rollback()
	for row, u := range users {
		body, _ := json.Marshal(u)
		fingerprint := store.Digest(body)
		var foundID, foundFingerprint string
		err = tx.QueryRowContext(ctx, "SELECT id,legacy_fingerprint FROM identities WHERE id=? OR server_uuid=?", u.ID, u.ServerUUID).Scan(&foundID, &foundFingerprint)
		if err == nil {
			if foundID != u.ID || foundFingerprint != fingerprint {
				return result, fmt.Errorf("existing identity conflict at row %d", row+1)
			}
			result.Unchanged++
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return result, errors.New("legacy preflight database failure")
		}
		aliasKeys := legacyAliases(u)
		for _, alias := range aliasKeys {
			var owner string
			err = tx.QueryRowContext(ctx, "SELECT user_id FROM identity_aliases WHERE alias_key=?", alias).Scan(&owner)
			if err == nil {
				return result, fmt.Errorf("existing alias conflict at row %d", row+1)
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return result, errors.New("legacy preflight database failure")
			}
		}
		result.Create++
		if !apply {
			continue
		}
		ref := "player_" + store.Digest([]byte("deuterium-player:" + u.ServerUUID))[:40]
		_, err = tx.ExecContext(ctx, `INSERT INTO identities (id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES (?,?,?,?,?,?,?,?,?,?)`, u.ID, ref, u.ServerUUID, u.GameID, u.QQ, u.PasswordHash, u.Status, u.CreatedAt.UTC(), u.UpdatedAt.UTC(), fingerprint)
		if err != nil {
			return result, fmt.Errorf("identity write conflict at row %d; batch rolled back", row+1)
		}
		for _, alias := range aliasKeys {
			if _, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases VALUES (?,?)", alias, u.ID); err != nil {
				return result, fmt.Errorf("alias write conflict at row %d; batch rolled back", row+1)
			}
		}
	}
	if !apply {
		return result, nil
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next (actor_id,action,resource_id,created_at) VALUES ('local-cli','identity.import',?,UTC_TIMESTAMP(6))", fmt.Sprintf("rows:%d", len(users)))
	if err != nil {
		return result, errors.New("import audit write failed; batch rolled back")
	}
	if err = tx.Commit(); err != nil {
		return result, errors.New("import commit result unknown; rerun dry-run to reconcile")
	}
	result.Applied = true
	return result, nil
}
