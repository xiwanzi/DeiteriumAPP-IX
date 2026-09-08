package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type Recipient struct {
	PlayerRef   string     `json:"playerRef"`
	GameID      string     `json:"gameId"`
	QQ          *string    `json:"qq"`
	Online      bool       `json:"online"`
	Registered  bool       `json:"registered"`
	Source      string     `json:"source"`
	ConfirmedAt time.Time  `json:"confirmedAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	UUID        string     `json:"-"`
}

func (s *Store) WalletRecipient(ctx context.Context, ref string) (p Recipient, err error) {
	var qq string
	err = s.DB.QueryRowContext(ctx, "SELECT player_ref,game_id,qq,server_uuid FROM identities WHERE player_ref=? AND status='active'", ref).Scan(&p.PlayerRef, &p.GameID, &qq, &p.UUID)
	if err == nil {
		p.QQ = &qq
		p.Registered = true
	} else if errors.Is(err, sql.ErrNoRows) {
		err = s.DB.QueryRowContext(ctx, "SELECT player_ref,game_id,player_uuid FROM core_player_directory WHERE player_ref=?", ref).Scan(&p.PlayerRef, &p.GameID, &p.UUID)
	}
	p.Source = "game_id"
	p.ConfirmedAt = time.Now().UTC()
	return
}
func (s *Store) WalletRecipientSearch(ctx context.Context, query, kind string) (out []Recipient, err error) {
	out = []Recipient{}
	var rows *sql.Rows
	if kind == "qq" {
		rows, err = s.DB.QueryContext(ctx, "SELECT player_ref FROM identities WHERE qq=? AND status='active' LIMIT 20", query)
	} else {
		comparison := "LOWER(game_id)=LOWER(?)"
		if kind == "search" {
			comparison = "LOCATE(LOWER(?),LOWER(game_id))=1"
		}
		rows, err = s.DB.QueryContext(ctx, `SELECT player_ref FROM identities WHERE status='active' AND `+comparison+` UNION SELECT player_ref FROM core_player_directory WHERE `+comparison+` LIMIT 20`, query, query)
	}
	if err != nil {
		return
	}
	var refs []string
	for rows.Next() {
		var ref string
		if err = rows.Scan(&ref); err != nil {
			rows.Close()
			return
		}
		refs = append(refs, ref)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return
	}
	for _, ref := range refs {
		p, e := s.WalletRecipient(ctx, ref)
		if e != nil {
			return nil, e
		}
		if kind == "qq" {
			p.Source = "qq"
		}
		out = append(out, p)
	}
	return
}

type WalletTransfer struct {
	ID              string          `json:"transferId"`
	ClientRequestID string          `json:"clientRequestId"`
	ActorID         string          `json:"-"`
	FromUUID        string          `json:"-"`
	ToUUID          string          `json:"-"`
	Recipient       Recipient       `json:"recipient"`
	Amount          string          `json:"amount"`
	Currency        string          `json:"currency"`
	Note            *string         `json:"note"`
	Status          string          `json:"status"`
	OperationID     string          `json:"operationId"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	Error           json.RawMessage `json:"error,omitempty"`
}

func (s *Store) CreateWalletTransfer(ctx context.Context, u User, client, node string, p Recipient, amount string, note *string) (id string, replay bool, err error) {
	raw, _ := json.Marshal(map[string]any{"fromUuid": u.ServerUUID, "toUuid": p.UUID, "amount": amount})
	fingerprintRaw, _ := json.Marshal(map[string]any{"recipient": p.PlayerRef, "amount": amount, "note": note})
	fingerprint := Digest(fingerprintRaw)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	id = "transfer_" + Digest([]byte(u.ID + ":" + client))[:40]
	opID := "coreop_" + Digest([]byte("wallet:" + u.ID + ":" + client))[:40]
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transfers_next VALUES (?,?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, id, u.ID, client, fingerprint, u.ServerUUID, p.UUID, p.PlayerRef, amount, note, opID)
	if duplicate(err) {
		var old string
		err = tx.QueryRowContext(ctx, "SELECT transfer_id,fingerprint FROM wallet_transfers_next WHERE actor_id=? AND client_request_id=?", u.ID, client).Scan(&id, &old)
		if err == nil && old != fingerprint {
			err = ErrConflict
		}
		return id, true, err
	}
	if err != nil {
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO core_operations VALUES (?,?,?,?,?,?,?,'QUEUED',NULL,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND),UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, opID, u.ID, "wallet:"+Digest([]byte(client)), node, "wallet.transfer", Digest(append([]byte(node+":wallet.transfer:"), raw...)), raw)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}
func (s *Store) WalletTransfer(ctx context.Context, id string) (t WalletTransfer, err error) {
	var ref, state string
	var result sql.NullString
	err = s.DB.QueryRowContext(ctx, `SELECT t.transfer_id,t.client_request_id,t.actor_id,t.from_uuid,t.to_uuid,t.recipient_ref,t.amount,t.note,t.operation_id,t.created_at,o.updated_at,o.state,o.result FROM wallet_transfers_next t JOIN core_operations o ON o.operation_id=t.operation_id WHERE t.transfer_id=?`, id).Scan(&t.ID, &t.ClientRequestID, &t.ActorID, &t.FromUUID, &t.ToUUID, &ref, &t.Amount, &t.Note, &t.OperationID, &t.CreatedAt, &t.UpdatedAt, &state, &result)
	if err != nil {
		return
	}
	t.Recipient, err = s.WalletRecipient(ctx, ref)
	t.Currency = "CREDIT"
	t.Status = "processing"
	switch state {
	case "COMPLETED":
		t.Status = "success"
	case "FAILED":
		t.Status = "failed"
	case "UNKNOWN":
		t.Status = "unknown"
	}
	if result.Valid {
		var v struct {
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal([]byte(result.String), &v) == nil {
			t.Error = v.Error
		}
	}
	return
}
func (s *Store) WalletTransfers(ctx context.Context, uuid, direction string, from, to time.Time, cursor string, limit int) ([]WalletTransfer, error) {
	field := "(from_uuid=? OR to_uuid=?)"
	args := []any{uuid, uuid}
	if direction == "income" {
		field = "to_uuid=?"
		args = []any{uuid}
	} else if direction == "expense" {
		field = "from_uuid=?"
		args = []any{uuid}
	}
	args = append(args, from.UTC(), to.UTC(), cursor, limit)
	rows, err := s.DB.QueryContext(ctx, "SELECT transfer_id FROM wallet_transfers_next WHERE "+field+" AND created_at>=? AND created_at<? AND transfer_id>? ORDER BY transfer_id LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []WalletTransfer{}
	for _, id := range ids {
		t, err := s.WalletTransfer(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}
