package imagereview

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

//go:embed schema.sql
var schema string

type Repository struct{ DB *sql.DB }

func (r *Repository) Migrate(ctx context.Context) error {
	for _, q := range strings.Split(schema, ";") {
		if strings.TrimSpace(q) == "" {
			continue
		}
		if _, err := r.DB.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	_, err := r.DB.ExecContext(ctx, "INSERT IGNORE INTO image_review_settings(id,threshold,prompt,version) VALUES(1,?,?,1)", Threshold, reviewPrompt)
	return err
}

func (r *Repository) Discover(ctx context.Context, c Candidate) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO image_review_items(id,source_kind,source_id,media_id,source_key,sha256,verified,content_type,size_bytes,animated,uploader_id,player_ref,game_id,display_name,purpose,uploaded_at,expires_at,source_state)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE
 media_id=IF(state NOT IN ('DELETED','DELETING'),VALUES(media_id),media_id),
 source_key=IF(state IN ('PENDING','RETRY','RETAINING'),VALUES(source_key),source_key),
 sha256=IF(state IN ('PENDING','RETRY'),VALUES(sha256),sha256),
 verified=IF(state IN ('PENDING','RETRY'),VALUES(verified),verified),
 animated=IF(state IN ('PENDING','RETRY'),VALUES(animated),animated),
 source_state=IF(state IN ('PENDING','RETRY'),VALUES(source_state),source_state)`,
		c.ID, c.SourceKind, c.SourceID, c.MediaID, c.SourceKey, c.SHA256, c.Verified, c.ContentType, c.SizeBytes, c.Animated, c.UploaderID, c.PlayerRef, c.GameID, c.DisplayName, c.Purpose, c.UploadedAt, c.ExpiresAt, c.SourceState)
	return err
}

const itemColumns = `id,source_kind,source_id,media_id,source_key,sha256,verified,content_type,size_bytes,animated,uploader_id,player_ref,game_id,display_name,purpose,uploaded_at,expires_at,source_state,state,score,threshold_used,result_json,policy_version,model,evidence_key,evidence_etag,error_code,attempts,reviewed_at,next_attempt_at`

func scanItem(row interface{ Scan(...any) error }) (i Item, err error) {
	var raw []byte
	err = row.Scan(&i.ID, &i.SourceKind, &i.SourceID, &i.MediaID, &i.SourceKey, &i.SHA256, &i.Verified, &i.ContentType, &i.SizeBytes, &i.Animated, &i.UploaderID, &i.PlayerRef, &i.GameID, &i.DisplayName, &i.Purpose, &i.UploadedAt, &i.ExpiresAt, &i.SourceState, &i.State, &i.Score, &i.ThresholdUsed, &raw, &i.PolicyVersion, &i.Model, &i.EvidenceKey, &i.EvidenceETag, &i.ErrorCode, &i.Attempts, &i.ReviewedAt, &i.NextAttemptAt)
	if err == nil && len(raw) > 0 {
		var v Result
		if err = json.Unmarshal(raw, &v); err == nil {
			i.Result = &v
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return
}
func (r *Repository) Item(ctx context.Context, id string) (Item, error) {
	return scanItem(r.DB.QueryRowContext(ctx, "SELECT "+itemColumns+" FROM image_review_items WHERE id=?", id))
}
func (r *Repository) Next(ctx context.Context) (Item, error) {
	return scanItem(r.DB.QueryRowContext(ctx, "SELECT "+itemColumns+" FROM image_review_items WHERE state IN ('PENDING','RETRY','RETAINING','DELETING') AND next_attempt_at<=UTC_TIMESTAMP(6) ORDER BY (state='DELETING') DESC,(state='RETAINING') DESC,uploaded_at DESC,next_attempt_at,id LIMIT 1"))
}

func (r *Repository) Cached(ctx context.Context, i Item) (*Result, error) {
	if !i.Verified || len(i.SHA256) != 64 {
		return nil, nil
	}
	var raw []byte
	err := r.DB.QueryRowContext(ctx, `SELECT result_json FROM image_review_items WHERE sha256=? AND verified=TRUE AND policy_version=? AND state IN ('CLEAR','FLAGGED') AND id<>? AND result_json IS NOT NULL ORDER BY reviewed_at DESC LIMIT 1`, i.SHA256, activePolicyVersion(ctx), i.ID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v Result
	if err = json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return &v, v.Validate()
}
func (r *Repository) Finish(ctx context.Context, id, state string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE image_review_items SET state=?,error_code='',reviewed_at=UTC_TIMESTAMP(6),updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state IN ('PENDING','RETRY','RETAINING')`, state, id)
	return err
}
func (r *Repository) Scored(ctx context.Context, i Item, result Result, key, etag string) error {
	state := "CLEAR"
	if result.Scores.Total() > activeSettings(ctx).Threshold {
		state = "RETAINING"
	}
	// Clear results retain category numbers only; discard any extracted content.
	saved := result
	if state == "CLEAR" {
		saved.Summary = "未超过标记阈值"
		saved.Reasons = []string{"三类风险未达到人工复核阈值"}
	}
	_, err := r.DB.ExecContext(ctx, `UPDATE image_review_items SET state=?,score=?,threshold_used=?,result_json=?,policy_version=?,model=?,evidence_key=?,evidence_etag=?,error_code='',reviewed_at=UTC_TIMESTAMP(6),next_attempt_at=UTC_TIMESTAMP(6),updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state IN ('PENDING','RETRY')`, state, result.Scores.Total(), activeSettings(ctx).Threshold, resultJSON(saved), activePolicyVersion(ctx), Model, key, etag, i.ID)
	return err
}
func (r *Repository) Retry(ctx context.Context, i Item, err error) error {
	delay := time.Duration(1<<min(i.Attempts, 9)) * 30 * time.Second
	var re *ReviewError
	if errors.As(err, &re) && re.Delay > delay {
		delay = re.Delay
	}
	state := i.State
	if state != "RETAINING" && state != "DELETING" {
		state = "RETRY"
	}
	_, e := r.DB.ExecContext(ctx, `UPDATE image_review_items SET state=?,attempts=attempts+1,error_code=?,next_attempt_at=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state IN ('PENDING','RETRY','RETAINING','DELETING')`, state, errorCode(err), time.Now().UTC().Add(delay), i.ID)
	return e
}
func (r *Repository) ScanStatus(ctx context.Context, ok bool) error {
	code := ""
	if !ok {
		code = "SOURCE_SCAN_FAILED"
	}
	_, e := r.DB.ExecContext(ctx, "UPDATE image_review_runtime SET last_scan_at=IF(?='',UTC_TIMESTAMP(6),last_scan_at),scan_error=? WHERE id=1", code, code)
	return e
}
func (r *Repository) Heartbeat(ctx context.Context) error {
	_, e := r.DB.ExecContext(ctx, "UPDATE image_review_runtime SET last_work_at=UTC_TIMESTAMP(6) WHERE id=1")
	return e
}

func (r *Repository) BeginDelete(ctx context.Context, id string) (Item, error) {
	_, err := r.DB.ExecContext(ctx, `UPDATE image_review_items SET state='DELETING',next_attempt_at=UTC_TIMESTAMP(6),updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state='FLAGGED'`, id)
	if err != nil {
		return Item{}, err
	}
	i, err := r.Item(ctx, id)
	if err == nil && i.State != "DELETING" && i.State != "DELETED" {
		return Item{}, ErrNotFound
	}
	return i, err
}
func (r *Repository) Deleted(ctx context.Context, id string) error {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "DELETE FROM image_review_events WHERE item_id=?", id); e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `UPDATE image_review_items SET state='DELETED',media_id='',source_key='',sha256='',uploader_id='',player_ref='',game_id='',display_name='',purpose='',score=NULL,result_json=NULL,evidence_key='',evidence_etag='',error_code='',updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state='DELETING'`, id)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (r *Repository) SaveEvents(ctx context.Context, id string, events []Event) error {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var state string
	if e = tx.QueryRowContext(ctx, "SELECT state FROM image_review_items WHERE id=? FOR UPDATE", id).Scan(&state); e != nil {
		return e
	}
	if state != "FLAGGED" && state != "RETAINING" {
		return nil
	}
	for _, v := range events {
		if _, e = tx.ExecContext(ctx, `INSERT IGNORE INTO image_review_events VALUES(?,?,?,?,?,?,?,?)`, id, v.ID, v.Channel, v.SenderID, v.PlayerRef, v.GameID, v.DisplayName, v.SentAt); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, "UPDATE image_review_items SET events_checked_at=UTC_TIMESTAMP(6) WHERE id=?", id); e != nil {
		return e
	}
	return tx.Commit()
}
func (r *Repository) Events(ctx context.Context, id string) ([]Event, bool, error) {
	rows, e := r.DB.QueryContext(ctx, "SELECT event_id,channel,sender_id,player_ref,game_id,display_name,sent_at FROM image_review_events WHERE item_id=? ORDER BY sent_at DESC,event_id DESC LIMIT 101", id)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var v Event
		if e = rows.Scan(&v.ID, &v.Channel, &v.SenderID, &v.PlayerRef, &v.GameID, &v.DisplayName, &v.SentAt); e != nil {
			return nil, false, e
		}
		out = append(out, v)
	}
	if e = rows.Err(); e != nil {
		return nil, false, e
	}
	more := len(out) > 100
	if more {
		out = out[:100]
	}
	return out, more, nil
}
func (r *Repository) EventWork(ctx context.Context) ([]Item, error) {
	rows, e := r.DB.QueryContext(ctx, "SELECT "+itemColumns+" FROM image_review_items WHERE state='FLAGGED' AND source_kind='CHAT' AND (events_checked_at IS NULL OR events_checked_at<DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 5 MINUTE)) ORDER BY events_checked_at LIMIT 10")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		v, e := scanItem(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) Summary(ctx context.Context) (Summary, error) {
	s := Summary{Model: Model, ModelLabel: "DeepSeek V4.1 Flash", Threshold: Threshold}
	settings, settingsErr := r.Settings(ctx)
	if settingsErr != nil {
		return s, settingsErr
	}
	s.Threshold = settings.Threshold
	rows, e := r.DB.QueryContext(ctx, "SELECT state,COUNT(*) FROM image_review_items GROUP BY state")
	if e != nil {
		return s, e
	}
	for rows.Next() {
		var state string
		var n int
		if e = rows.Scan(&state, &n); e != nil {
			rows.Close()
			return s, e
		}
		switch state {
		case "FLAGGED":
			s.Flagged = n
		case "CLEAR":
			s.Clear = n
		case "PENDING":
			s.Pending = n
		case "RETRY":
			s.Retrying += n
		case "RETAINING":
			s.Retrying += n
		case "SKIPPED":
			s.Skipped = n
		case "UNAVAILABLE":
			s.Unavailable = n
		case "DELETING":
			s.Deleting = n
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return s, e
	}
	e = r.DB.QueryRowContext(ctx, "SELECT last_scan_at,last_work_at,scan_error FROM image_review_runtime WHERE id=1").Scan(&s.LastScanAt, &s.LastWorkAt, &s.ScanError)
	return s, e
}

// List uses a bounded page and a deterministic score/date/id order.
func (r *Repository) List(ctx context.Context, page int) ([]Item, bool, error) {
	if page < 0 || page > 10000 {
		return nil, false, ErrNotFound
	}
	rows, e := r.DB.QueryContext(ctx, "SELECT "+itemColumns+" FROM image_review_items WHERE state='FLAGGED' ORDER BY score DESC,uploaded_at DESC,id DESC LIMIT 25 OFFSET ?", page*24)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		i, e := scanItem(rows)
		if e != nil {
			return nil, false, e
		}
		out = append(out, i)
	}
	if e = rows.Err(); e != nil {
		return nil, false, e
	}
	more := len(out) > 24
	if more {
		out = out[:24]
	}
	return out, more, nil
}
