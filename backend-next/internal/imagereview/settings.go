package imagereview

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrSettingsConflict = errors.New("review settings changed")
var ErrSettingsInvalid = errors.New("invalid review settings")

type Settings struct {
	Threshold int        `json:"threshold"`
	Prompt    string     `json:"prompt"`
	Version   int64      `json:"version"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{Threshold: Threshold, Prompt: reviewPrompt, Version: 1}
}
func (s Settings) Validate() error {
	if s.Threshold < 0 || s.Threshold > 100 || strings.TrimSpace(s.Prompt) == "" || !utf8.ValidString(s.Prompt) || utf8.RuneCountInString(s.Prompt) > 12000 || strings.ContainsRune(s.Prompt, 0) || s.Version < 1 {
		return ErrSettingsInvalid
	}
	return nil
}
func (r *Repository) Settings(ctx context.Context) (s Settings, e error) {
	e = r.DB.QueryRowContext(ctx, "SELECT threshold,prompt,version,updated_at FROM image_review_settings WHERE id=1").Scan(&s.Threshold, &s.Prompt, &s.Version, &s.UpdatedAt)
	return
}
func (r *Repository) SaveSettings(ctx context.Context, actor string, next Settings) (Settings, error) {
	next.Prompt = strings.TrimSpace(next.Prompt)
	if e := next.Validate(); e != nil {
		return Settings{}, e
	}
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return Settings{}, e
	}
	defer tx.Rollback()
	var current Settings
	e = tx.QueryRowContext(ctx, "SELECT threshold,prompt,version,updated_at FROM image_review_settings WHERE id=1 FOR UPDATE").Scan(&current.Threshold, &current.Prompt, &current.Version, &current.UpdatedAt)
	if e != nil {
		return current, e
	}
	// Retrying a save with an unknown response is a no-op if its exact content won.
	if current.Threshold == next.Threshold && current.Prompt == next.Prompt {
		return current, nil
	}
	if current.Version != next.Version {
		return current, ErrSettingsConflict
	}
	_, e = tx.ExecContext(ctx, "UPDATE image_review_settings SET threshold=?,prompt=?,version=version+1,updated_at=UTC_TIMESTAMP(6),updated_by=? WHERE id=1", next.Threshold, next.Prompt, actor)
	if e != nil {
		return current, e
	}
	if e = tx.Commit(); e != nil {
		return current, e
	}
	return r.Settings(ctx)
}

type settingsKey struct{}

func withSettings(ctx context.Context, s Settings) context.Context {
	return context.WithValue(ctx, settingsKey{}, s)
}
func activeSettings(ctx context.Context) Settings {
	if s, ok := ctx.Value(settingsKey{}).(Settings); ok {
		return s
	}
	return DefaultSettings()
}
func activePolicyVersion(ctx context.Context) string {
	return PolicyVersion + ":" + strconv.FormatInt(activeSettings(ctx).Version, 10)
}
