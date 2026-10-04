package store

import (
	"context"
	"database/sql"
	"errors"
)

// Merged ticket numbers remain usable in old mail and bookmarked URLs.
type supportQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func resolveSupportTicket(ctx context.Context, db supportQuery, id string) (string, error) {
	var canonical string
	err := db.QueryRowContext(ctx, "SELECT ticket_id FROM support_ticket_aliases WHERE alias_ticket_id=?", id).Scan(&canonical)
	if errors.Is(err, sql.ErrNoRows) {
		return id, nil
	}
	return canonical, err
}
