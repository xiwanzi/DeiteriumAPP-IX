package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
)

func (s *Server) supportInboxWorker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.pollSupportInbox(s.ctx); err != nil && !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, context.Canceled) {
				slog.Warn("support inbox deferred", "reason", "mailbox will retry")
			}
		}
	}
}
func (s *Server) pollSupportInbox(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	inbox, err := s.Store.ClaimSupportInbox(ctx)
	if err != nil {
		return err
	}
	password, pollErr := notify.Open(s.Config.SMTPKey, inbox.PasswordCipher)
	if pollErr == nil {
		var validity uint32
		pollErr = notify.PollInbox(ctx, inbox.InboxSettings, password, func(v, next uint32) (uint32, error) {
			validity = v
			return s.Store.SupportInboxPosition(ctx, inbox, v, next)
		}, func(uid uint32, raw []byte, received time.Time) error {
			_, err := s.Store.IngestSupportMail(ctx, inbox, validity, uid, raw, received, s.Config.SMTPKey)
			return err
		})
	}
	finish, done := context.WithTimeout(context.Background(), 3*time.Second)
	defer done()
	if err = s.Store.FinishSupportInbox(finish, inbox.LeaseID, pollErr); err != nil {
		return err
	}
	return pollErr
}
