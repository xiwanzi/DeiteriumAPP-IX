package store

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const MaxKeybindings = 2048

// Names emitted by Minecraft 1.21.1 InputConstants, including numeric platform codes.
var keyValue = regexp.MustCompile(`^(key\.keyboard\.([a-z]|[0-9]{1,5}|f([1-9]|1[0-9]|2[0-5])|keypad\.([0-9]|add|decimal|enter|equal|multiply|divide|subtract)|unknown|up|down|left|right|apostrophe|backslash|comma|equal|grave\.accent|(left|right)\.(bracket|alt|control|shift|win)|minus|period|semicolon|slash|space|tab|enter|escape|backspace|delete|end|home|insert|page\.(down|up)|(caps|num|scroll)\.lock|pause|menu|print\.screen|world\.[12])|key\.mouse\.(left|right|middle|[1-9][0-9]{0,3})|scancode\.[0-9]{1,5})(:(ALT|SHIFT|CONTROL))?$`)

type Keybinding struct {
	Label    string `json:"label"`
	Category string `json:"category"`
	Context  string `json:"context"`
	Value    string `json:"value"`
	Observed string `json:"observed"`
	Conflict bool   `json:"conflict,omitempty"`
	Missing  bool   `json:"missing,omitempty"`
	Revision int64  `json:"revision"`
}

type KeyChange struct {
	Value    string `json:"value"`
	Before   string `json:"before,omitempty"`
	Ack      int64  `json:"ack,omitempty"`
	Label    string `json:"label,omitempty"`
	Category string `json:"category,omitempty"`
	Context  string `json:"context,omitempty"`
}

type KeySync struct {
	PlayerUUID   string               `json:"playerUuid"`
	Since        int64                `json:"since"`
	Seed         bool                 `json:"seed,omitempty"`
	AllowRemote  bool                 `json:"allowRemote"`
	Changes      map[string]KeyChange `json:"changes,omitempty"`
	Availability map[string]bool      `json:"availability,omitempty"`
}

type KeyProfile struct {
	UUID        string    `json:"playerUuid"`
	GameID      string    `json:"gameId"`
	Revision    int64     `json:"revision"`
	AllowRemote bool      `json:"allowRemote"`
	SyncedAt    time.Time `json:"syncedAt"`
}

type KeyState struct {
	KeyProfile
	Rows   map[string]Keybinding `json:"rows"`
	Seeded bool                  `json:"seeded,omitempty"`
	Full   bool                  `json:"full,omitempty"`
}

func keyText(s string, n int) bool {
	return s != "" && utf8.ValidString(s) && utf8.RuneCountInString(s) <= n && strings.IndexFunc(s, unicode.IsControl) < 0
}
func ValidKeyValue(s string) bool {
	return len(s) <= 80 && keyValue.MatchString(s) && !strings.HasPrefix(s, "key.keyboard.unknown:")
}
func validKeyMeta(c KeyChange) bool {
	return keyText(c.Label, 192) && keyText(c.Category, 192) && (c.Context == "game" || c.Context == "gui" || c.Context == "all" || c.Context == "custom")
}
func (in KeySync) Valid() bool {
	if in.Since < 0 || len(in.Changes) > MaxKeybindings || len(in.Availability) > MaxKeybindings || (in.Seed && (in.Since != 0 || len(in.Changes) == 0)) {
		return false
	}
	for id := range in.Availability {
		if !keyText(id, 192) {
			return false
		}
	}
	for id, c := range in.Changes {
		if !keyText(id, 192) || !ValidKeyValue(c.Value) || c.Ack < 0 || (c.Before != "" && !ValidKeyValue(c.Before)) {
			return false
		}
		if in.Seed || (c.Before == "" && c.Ack == 0) {
			if c.Before != "" || c.Ack != 0 || !validKeyMeta(c) {
				return false
			}
		}
	}
	return true
}

const keyCols = "label,category,key_context,desired,observed,conflict,missing,revision"
const profileCols = "player_uuid,game_id,revision,allow_remote,synced_at"

type keyScanner interface{ Scan(...any) error }

func scanProfile(s keyScanner, p *KeyProfile) error {
	return s.Scan(&p.UUID, &p.GameID, &p.Revision, &p.AllowRemote, &p.SyncedAt)
}
func getKeyRows(ctx context.Context, tx *sql.Tx, uuid string, since int64) (map[string]Keybinding, error) {
	rs, err := tx.QueryContext(ctx, "SELECT action_id,"+keyCols+" FROM game_keybindings WHERE player_uuid=? AND revision>?", uuid, since)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	out := map[string]Keybinding{}
	for rs.Next() {
		var id string
		var k Keybinding
		if err := rs.Scan(&id, &k.Label, &k.Category, &k.Context, &k.Value, &k.Observed, &k.Conflict, &k.Missing, &k.Revision); err != nil {
			return nil, err
		}
		out[id] = k
	}
	return out, rs.Err()
}

func insertKeys(ctx context.Context, tx *sql.Tx, uuid string, revision int64, changes map[string]KeyChange) error {
	ids := make([]string, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for start := 0; start < len(ids); start += 128 {
		batch := ids[start:min(start+128, len(ids))]
		args := make([]any, 0, len(batch)*8)
		for _, id := range batch {
			c := changes[id]
			args = append(args, uuid, id, c.Label, c.Category, c.Context, c.Value, c.Value, revision)
		}
		query := "INSERT INTO game_keybindings(player_uuid,action_id,label,category,key_context,desired,observed,revision) VALUES " + strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?,?,?),", len(batch)), ",")
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

// An acknowledgement describes a specific delivered revision. An old retry cannot approve a newer edit.
func mergeKey(k Keybinding, c KeyChange) Keybinding {
	if c.Ack > 0 {
		if c.Ack == k.Revision && c.Value == k.Value {
			k.Observed = c.Value
			k.Conflict = false
		}
		return k
	}
	if c.Before == "" {
		return k
	} // Another installation already registered this action.
	if c.Value == k.Value {
		k.Observed = c.Value
		k.Conflict = false
		return k
	}
	if c.Before == k.Value && !k.Conflict {
		k.Value = c.Value
		k.Observed = c.Value
		return k
	}
	// A retry after an administrator resolved this exact observation must not reopen the conflict.
	if c.Value != k.Observed {
		k.Observed = c.Value
		k.Conflict = true
	}
	return k
}

func (s *Store) SyncKeybindings(ctx context.Context, uuid, gameID string, in KeySync) (out KeyState, err error) {
	out.UUID = uuid
	out.Rows = map[string]Keybinding{}
	if !in.Valid() {
		return out, ErrSocialInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if in.Seed {
		_, err = tx.ExecContext(ctx, "INSERT INTO game_keybinding_profiles(player_uuid,game_id,synced_at) VALUES(?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE player_uuid=VALUES(player_uuid)", uuid, gameID)
		if err != nil {
			return out, err
		}
	}
	err = scanProfile(tx.QueryRowContext(ctx, "SELECT "+profileCols+" FROM game_keybinding_profiles WHERE player_uuid=? FOR UPDATE", uuid), &out.KeyProfile)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	since := in.Since
	if since > out.Revision {
		since = 0
	}
	out.Full = since == 0
	next := out.Revision + 1
	if in.Seed && out.Revision == 0 {
		if err = insertKeys(ctx, tx, uuid, next, in.Changes); err != nil {
			return out, err
		}
		out.Revision = next
		out.Seeded = true
		out.Full = false
	} else if !in.Seed {
		additions := map[string]KeyChange{}
		for id, c := range in.Changes {
			var k Keybinding
			err = tx.QueryRowContext(ctx, "SELECT "+keyCols+" FROM game_keybindings WHERE player_uuid=? AND action_id=?", uuid, id).Scan(&k.Label, &k.Category, &k.Context, &k.Value, &k.Observed, &k.Conflict, &k.Missing, &k.Revision)
			if errors.Is(err, sql.ErrNoRows) {
				if c.Before == "" && c.Ack == 0 {
					additions[id] = c
				}
				continue
			}
			if err != nil {
				return out, err
			}
			updated := mergeKey(k, c)
			if updated == k {
				continue
			}
			_, err = tx.ExecContext(ctx, "UPDATE game_keybindings SET desired=?,observed=?,conflict=?,revision=? WHERE player_uuid=? AND action_id=?", updated.Value, updated.Observed, updated.Conflict, next, uuid, id)
			if err != nil {
				return out, err
			}
			out.Revision = next
		}
		if len(additions) > 0 {
			var count int
			if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM game_keybindings WHERE player_uuid=?", uuid).Scan(&count); err != nil {
				return out, err
			}
			if count+len(additions) > MaxKeybindings {
				return out, ErrSocialInvalid
			}
			if err = insertKeys(ctx, tx, uuid, next, additions); err != nil {
				return out, err
			}
			out.Revision = next
		}
	}
	for _, available := range []bool{false, true} {
		args := []any{!available, next, uuid, !available}
		for id, present := range in.Availability {
			if present == available {
				args = append(args, id)
			}
		}
		if len(args) == 4 {
			continue
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)-4), ",")
		result, e := tx.ExecContext(ctx, "UPDATE game_keybindings SET missing=?,revision=? WHERE player_uuid=? AND missing<>? AND action_id IN ("+placeholders+")", args...)
		if e != nil {
			return out, e
		}
		n, e := result.RowsAffected()
		if e != nil {
			return out, e
		}
		if n > 0 {
			out.Revision = next
		}
	}
	out.GameID = gameID
	out.AllowRemote = in.AllowRemote
	out.SyncedAt = time.Now().UTC()
	if _, err = tx.ExecContext(ctx, "UPDATE game_keybinding_profiles SET game_id=?,revision=?,allow_remote=?,synced_at=? WHERE player_uuid=?", out.GameID, out.Revision, out.AllowRemote, out.SyncedAt, uuid); err != nil {
		return out, err
	}
	if !out.Seeded {
		out.Rows, err = getKeyRows(ctx, tx, uuid, since)
		if err != nil {
			return out, err
		}
	}
	return out, tx.Commit()
}

func (s *Store) KeyProfiles(ctx context.Context, q string, offset int) ([]KeyProfile, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+profileCols+" FROM game_keybinding_profiles WHERE LOCATE(LOWER(?),LOWER(game_id))>0 OR LOCATE(?,player_uuid)>0 ORDER BY synced_at DESC,player_uuid LIMIT 31 OFFSET ?", q, strings.ToLower(q), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []KeyProfile{}
	for rows.Next() {
		var p KeyProfile
		if err = scanProfile(rows, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Store) KeyProfileDetail(ctx context.Context, uuid string, since int64) (out KeyState, err error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = scanProfile(tx.QueryRowContext(ctx, "SELECT "+profileCols+" FROM game_keybinding_profiles WHERE player_uuid=?", uuid), &out.KeyProfile); err != nil {
		return out, err
	}
	if since > out.Revision {
		since = 0
	}
	out.Full = since == 0
	out.Rows, err = getKeyRows(ctx, tx, uuid, since)
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}
func (s *Store) SetAdminKey(ctx context.Context, actor, uuid, id, value string, expected int64) error {
	if !keyText(id, 192) || !ValidKeyValue(value) || expected < 1 {
		return ErrSocialInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var profileRevision int64
	if err = tx.QueryRowContext(ctx, "SELECT revision FROM game_keybinding_profiles WHERE player_uuid=? FOR UPDATE", uuid).Scan(&profileRevision); err != nil {
		return err
	}
	var k Keybinding
	if err = tx.QueryRowContext(ctx, "SELECT desired,observed,conflict,revision FROM game_keybindings WHERE player_uuid=? AND action_id=?", uuid, id).Scan(&k.Value, &k.Observed, &k.Conflict, &k.Revision); err != nil {
		return err
	}
	if k.Revision != expected {
		return ErrConflict
	}
	if value == k.Value && !k.Conflict {
		return nil
	}
	next := profileRevision + 1
	if _, err = tx.ExecContext(ctx, "UPDATE game_keybindings SET desired=?,conflict=FALSE,revision=? WHERE player_uuid=? AND action_id=?", value, next, uuid, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE game_keybinding_profiles SET revision=? WHERE player_uuid=?", next, uuid); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'keybindings.edit',?,UTC_TIMESTAMP(6))", actor, uuid); err != nil {
		return err
	}
	return tx.Commit()
}
