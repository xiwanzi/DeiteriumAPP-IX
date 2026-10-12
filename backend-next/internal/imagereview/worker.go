package imagereview

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type Service struct {
	Repo     *Repository
	Source   *Source
	Objects  Objects
	Reviewer Reviewer
	cursors  map[string]string
}

func (s *Service) Scan(ctx context.Context) error {
	if s.cursors == nil {
		s.cursors = map[string]string{}
	}
	for _, kind := range []string{"CHAT"} {
		for batch := 0; batch < 4; batch++ {
			rows, e := s.Source.Batch(ctx, kind, s.cursors[kind], 200)
			if e != nil {
				_ = s.Repo.ScanStatus(ctx, false)
				return e
			}
			for _, c := range rows {
				if e = s.Repo.Discover(ctx, c); e != nil {
					return e
				}
				s.cursors[kind] = c.SourceID
			}
			if len(rows) < 200 {
				s.cursors[kind] = ""
				break
			}
		}
	}
	return s.Repo.ScanStatus(ctx, true)
}
func (s *Service) Process(ctx context.Context, i Item) error {
	if i.State == "DELETING" {
		return s.Cleanup(ctx, i)
	}
	if i.State == "RETAINING" {
		return s.retain(ctx, i)
	}
	if i.Animated {
		return s.Repo.Finish(ctx, i.ID, "SKIPPED")
	}
	if i.SourceState == "PURGED" || i.SourceState == "DELETING" || i.SourceState == "REMOVED" {
		return s.Repo.Finish(ctx, i.ID, "UNAVAILABLE")
	}
	info, e := s.Objects.Inspect(ctx, i.Candidate)
	if errors.Is(e, ErrAnimated) {
		return s.Repo.Finish(ctx, i.ID, "SKIPPED")
	}
	if errors.Is(e, ErrInvalidImage) {
		return s.Repo.Finish(ctx, i.ID, "UNAVAILABLE")
	}
	if errors.Is(e, ErrObjectMissing) && time.Now().After(i.ExpiresAt) && i.Attempts >= 1 {
		return s.Repo.Finish(ctx, i.ID, "UNAVAILABLE")
	}
	if e != nil {
		return e
	}
	result, e := s.Repo.Cached(ctx, i)
	if e != nil {
		return e
	}
	if result == nil {
		imageURL, e := s.Objects.URL(ctx, i.SourceKey)
		if e != nil {
			return e
		}
		value, e := s.Reviewer.Review(ctx, imageURL)
		if e != nil {
			return e
		}
		if e = value.Validate(); e != nil {
			return &ReviewError{Code: "MODEL_RESPONSE_INVALID"}
		}
		result = &value
	}
	key := ""
	if result.Scores.Total() > Threshold {
		key = s.Objects.EvidenceKey(i.ID)
	}
	if e = s.Repo.Scored(ctx, i, *result, key, info.ETag); e != nil {
		return e
	}
	if key == "" {
		return nil
	}
	// Refresh persisted state so a failed copy retries retention, not paid scoring.
	i, e = s.Repo.Item(ctx, i.ID)
	if e != nil {
		return e
	}
	return s.retain(ctx, i)
}
func (s *Service) retain(ctx context.Context, i Item) error {
	if e := s.Objects.Retain(ctx, i.SourceKey, i.EvidenceKey, i.EvidenceETag, i.SizeBytes); e != nil {
		return e
	}
	if i.SourceKind == "CHAT" {
		events, e := s.Source.Events(ctx, i.MediaID)
		if e != nil {
			return e
		}
		if e = s.Repo.SaveEvents(ctx, i.ID, events); e != nil {
			return e
		}
	}
	return s.Repo.Finish(ctx, i.ID, "FLAGGED")
}
func (s *Service) Cleanup(ctx context.Context, i Item) error {
	if i.State == "DELETED" {
		return nil
	}
	if i.State != "DELETING" {
		return ErrNotFound
	}
	if e := s.Objects.DeleteEvidence(ctx, i.EvidenceKey); e != nil {
		return e
	}
	return s.Repo.Deleted(ctx, i.ID)
}
func (s *Service) RefreshEvents(ctx context.Context, i Item) error {
	if i.SourceKind != "CHAT" {
		return nil
	}
	events, e := s.Source.Events(ctx, i.MediaID)
	if e != nil {
		return e
	}
	return s.Repo.SaveEvents(ctx, i.ID, events)
}
func (s *Service) RunScanner(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		scanCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		if e := s.Scan(scanCtx); e != nil && ctx.Err() == nil {
			slog.Warn("image review source scan deferred")
			_ = s.Repo.ScanStatus(ctx, false)
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunWorker is run by one advisory-lock owner. No hot loop on an empty queue.
func (s *Service) RunWorker(ctx context.Context, checkLease func(context.Context) error) error {
	for {
		if e := checkLease(ctx); e != nil {
			return e
		}
		i, e := s.Repo.Next(ctx)
		if e == nil {
			jobCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			e = s.Process(jobCtx, i)
			cancel()
			if e != nil && ctx.Err() == nil {
				// Retention may have been persisted before the failure.
				if fresh, loadErr := s.Repo.Item(ctx, i.ID); loadErr == nil {
					i = fresh
				}
				if retryErr := s.Repo.Retry(ctx, i, e); retryErr != nil {
					slog.Warn("image review retry persistence failed")
				}
				slog.Warn("image review deferred", "record", i.ID, "code", errorCode(e))
			}
		} else if !errors.Is(e, ErrNotFound) && ctx.Err() == nil {
			slog.Warn("image review queue unavailable")
		}
		_ = s.Repo.Heartbeat(ctx)
		if items, err := s.Repo.EventWork(ctx); err == nil {
			for _, v := range items {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				eventCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				_ = s.RefreshEvents(eventCtx, v)
				cancel()
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
