package store

import (
	"math"
	"time"
)

type RateLimitError struct {
	RetryAfterSeconds int
	Scope             string
}

func (e *RateLimitError) Error() string { return ErrRateLimited.Error() }
func (e *RateLimitError) Unwrap() error { return ErrRateLimited }

func rateLimitedUntil(deadline time.Time) *RateLimitError {
	return &RateLimitError{RetryAfterSeconds: max(1, int(math.Ceil(time.Until(deadline).Seconds())))}
}

func (e *RateLimitError) withScope(scope string) *RateLimitError { e.Scope = scope; return e }
