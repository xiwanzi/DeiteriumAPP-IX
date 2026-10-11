package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const DefaultTitleID = "title_default"

var titleInternalID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,47}$`)
var titleColor = regexp.MustCompile(`#[0-9a-fA-F]{6}`)

type GameTitle struct {
	ID              string `json:"id"`
	InternalID      string `json:"internalId"`
	DisplayName     string `json:"displayName"`
	Description     string `json:"description"`
	Public          bool   `json:"public"`
	Weight          int    `json:"weight"`
	Status          string `json:"status"`
	PublishSequence int64  `json:"publishSequence"`
	Version         int64  `json:"version"`
}
type GameTitleCatalog struct {
	Revision int64       `json:"revision"`
	Items    []GameTitle `json:"items"`
}
type GameTitleChange struct {
	ClientRequestID string `json:"clientRequestId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	InternalID      string `json:"internalId"`
	DisplayName     string `json:"displayName"`
	Description     string `json:"description"`
	Public          bool   `json:"public"`
	Weight          int    `json:"weight"`
	Status          string `json:"status"`
}

func ValidTitleDisplay(s string) bool {
	if !utf8.ValidString(s) || len(s) > 512 {
		return false
	}
	plain := titleColor.ReplaceAllString(s, "")
	if strings.TrimSpace(plain) == "" || utf8.RuneCountInString(plain) > 32 {
		return false
	}
	for _, r := range plain {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '§' || r == '#' {
			return false
		}
	}
	return true
}

const titleColumns = "id,internal_id,display_name,weight,status,publish_sequence,version,description,is_public"

func scanGameTitle(row interface{ Scan(...any) error }) (t GameTitle, err error) {
	err = row.Scan(&t.ID, &t.InternalID, &t.DisplayName, &t.Weight, &t.Status, &t.PublishSequence, &t.Version, &t.Description, &t.Public)
	return
}
func (s *Store) GameTitles(ctx context.Context, admin bool) (GameTitleCatalog, error) {
	out := GameTitleCatalog{Items: []GameTitle{}}
	// Consistent snapshot keeps revision and definitions from different edits apart.
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, "SELECT revision FROM game_title_revision WHERE id=1").Scan(&out.Revision); err != nil {
		return out, err
	}
	where := "status='PUBLISHED'"
	if admin {
		where = "status<>'DELETED'"
	}
	rows, err := tx.QueryContext(ctx, "SELECT "+titleColumns+" FROM game_title_catalog WHERE "+where+" ORDER BY weight DESC,publish_sequence DESC,id")
	if err != nil {
		return out, err
	}
	for rows.Next() {
		t, e := scanGameTitle(rows)
		if e != nil {
			rows.Close()
			return out, e
		}
		out.Items = append(out.Items, t)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}
func (s *Store) ChangeGameTitle(ctx context.Context, actor, id string, in GameTitleChange) (json.RawMessage, error) {
	if !titleInternalID.MatchString(in.InternalID) || !ValidTitleDisplay(in.DisplayName) || !ValidSocialText(in.Description, 0, 1000) || !catalogEnum(in.Status, "DRAFT", "PUBLISHED", "DISABLED", "DELETED") || in.Weight < -1000000 || in.Weight > 1000000 {
		return nil, ErrSocialInvalid
	}
	if id == "" && (in.ExpectedVersion != 0 || in.Status == "DELETED" || in.InternalID == "default") {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "title.manage:"+id, in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var revision int64
		if err := tx.QueryRowContext(ctx, "SELECT revision FROM game_title_revision WHERE id=1 FOR UPDATE").Scan(&revision); err != nil {
			return nil, err
		}
		current := GameTitle{ID: id}
		if id != "" {
			var err error
			current, err = scanGameTitle(tx.QueryRowContext(ctx, "SELECT "+titleColumns+" FROM game_title_catalog WHERE id=? FOR UPDATE", id))
			if err != nil {
				return nil, socialMissing(err)
			}
			if current.Version != in.ExpectedVersion || current.Status == "DELETED" {
				return nil, ErrConflict
			}
			if current.InternalID != in.InternalID {
				return nil, catalogError(400, "TITLE_INTERNAL_ID_IMMUTABLE", "内部 ID 创建后不能修改。")
			}
			if id == DefaultTitleID && in.Status != "PUBLISHED" {
				return nil, catalogError(400, "DEFAULT_TITLE_REQUIRED", "默认头衔不能停用或删除。")
			}
		} else {
			var n int
			if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM game_title_catalog WHERE internal_id=?", in.InternalID).Scan(&n); err != nil {
				return nil, err
			}
			if n > 0 {
				return nil, catalogError(409, "TITLE_INTERNAL_ID_TAKEN", "内部 ID 已使用，删除后也不能复用。")
			}
			current.ID = ID("ttl_")
		}
		seq := current.PublishSequence
		if in.Status == "PUBLISHED" && current.Status != "PUBLISHED" {
			seq = revision + 1
		}
		out := GameTitle{ID: current.ID, InternalID: in.InternalID, DisplayName: in.DisplayName, Description: in.Description, Public: in.Public, Weight: in.Weight, Status: in.Status, PublishSequence: seq, Version: current.Version + 1}
		if id == "" {
			_, err := tx.ExecContext(ctx, "INSERT INTO game_title_catalog ("+titleColumns+") VALUES(?,?,?,?,?,?,?,?,?)", out.ID, out.InternalID, out.DisplayName, out.Weight, out.Status, out.PublishSequence, out.Version, out.Description, out.Public)
			if err != nil {
				return nil, err
			}
		} else {
			if _, err := tx.ExecContext(ctx, "UPDATE game_title_catalog SET display_name=?,weight=?,status=?,publish_sequence=?,version=?,description=?,is_public=? WHERE id=?", out.DisplayName, out.Weight, out.Status, out.PublishSequence, out.Version, out.Description, out.Public, id); err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE game_title_revision SET revision=revision+1 WHERE id=1"); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", actor, "title."+strings.ToLower(in.Status), out.ID); err != nil {
			return nil, err
		}
		return out, nil
	})
}
