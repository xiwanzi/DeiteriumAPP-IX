package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type DesktopLauncherDocument struct {
	Bootstrap json.RawMessage `json:"bootstrap"`
	Content   json.RawMessage `json:"content"`
}
type DesktopLauncherSettings struct {
	Version          int64                    `json:"version"`
	Draft            DesktopLauncherDocument  `json:"draft"`
	Published        *DesktopLauncherDocument `json:"published"`
	PublishedVersion int64                    `json:"publishedVersion"`
	UpdatedAt        time.Time                `json:"updatedAt"`
	PublishedAt      *time.Time               `json:"publishedAt"`
}
type DesktopLauncherSync struct {
	ID        string    `json:"id"`
	State     string    `json:"state"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ValidateDesktopDocument(v DesktopLauncherDocument) error {
	if len(v.Bootstrap) > 32768 || len(v.Content) > 768000 || !json.Valid(v.Bootstrap) || !json.Valid(v.Content) {
		return ErrSocialInvalid
	}
	var bootstrap struct {
		Minecraft string `json:"minecraftVersion"`
		Loader    string `json:"loaderVersion"`
		Instance  string `json:"instanceVersion"`
		Baseline  string `json:"baselineVersion"`
		URL       string `json:"mcpatchUrl"`
	}
	if json.Unmarshal(v.Bootstrap, &bootstrap) != nil {
		return ErrSocialInvalid
	}
	safe := regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	for _, id := range []string{bootstrap.Minecraft, bootstrap.Loader, bootstrap.Instance, bootstrap.Baseline} {
		if !safe.MatchString(id) {
			return ErrSocialInvalid
		}
	}
	if !DesktopPublicURL(bootstrap.URL) {
		return ErrSocialInvalid
	}
	var content struct {
		Entries []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"entries"`
	}
	if json.Unmarshal(v.Content, &content) != nil || len(content.Entries) < 1 || len(content.Entries) > 12 {
		return ErrSocialInvalid
	}
	seen := map[string]bool{}
	for _, entry := range content.Entries {
		if !safe.MatchString(entry.Key) || seen[entry.Key] || len([]rune(entry.Name)) < 1 || len([]rune(entry.Name)) > 60 {
			return ErrSocialInvalid
		}
		seen[entry.Key] = true
	}
	return nil
}

func DesktopPublicURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Fragment == "" && !strings.ContainsAny(raw, "\r\n")
}

func ValidateDesktopPublication(v DesktopLauncherDocument) error {
	if err := ValidateDesktopDocument(v); err != nil {
		return err
	}
	var artifacts struct {
		Java struct {
			URL    string `json:"url"`
			Size   int64  `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"java"`
		Installer struct {
			URL    string `json:"url"`
			Size   int64  `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"installer"`
	}
	digest := regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	if json.Unmarshal(v.Bootstrap, &artifacts) != nil || !DesktopPublicURL(artifacts.Java.URL) || !DesktopPublicURL(artifacts.Installer.URL) || artifacts.Java.Size < 1 || artifacts.Installer.Size < 1 || !digest.MatchString(artifacts.Java.SHA256) || !digest.MatchString(artifacts.Installer.SHA256) {
		return ErrSocialInvalid
	}
	var content struct {
		Entries []struct {
			Key        string            `json:"key"`
			Icon       string            `json:"icon"`
			Background string            `json:"background"`
			Video      string            `json:"video"`
			Gallery    string            `json:"gallery"`
			Cover      string            `json:"cover"`
			Accent     string            `json:"accent"`
			Hover      string            `json:"hover"`
			Pressed    string            `json:"pressed"`
			Target     string            `json:"launchTarget"`
			Tabs       []string          `json:"tabs"`
			Sidebars   []json.RawMessage `json:"sidebars"`
			Banners    []struct {
				Title string `json:"title"`
				Image string `json:"image"`
			} `json:"banners"`
			News []struct {
				Title  string   `json:"title"`
				Tab    string   `json:"tab"`
				Date   string   `json:"date"`
				Body   string   `json:"body"`
				Images []string `json:"images"`
			} `json:"news"`
		} `json:"entries"`
	}
	if json.Unmarshal(v.Content, &content) != nil || len(content.Entries) != 3 {
		return ErrSocialInvalid
	}
	asset := func(path string) bool {
		return DesktopPublicURL(path) || (strings.HasPrefix(path, "hypergryph/") && !strings.Contains(path, "..") && !strings.ContainsAny(path, "\\:\r\n"))
	}
	color := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	for i, entry := range content.Entries {
		if entry.Key != []string{"arknights", "endfield", "popucom"}[i] || len(entry.Tabs) < 1 || len(entry.Tabs) > 5 || entry.Sidebars == nil || len(entry.Sidebars) > 16 || len(entry.Banners) > 8 || len(entry.News) > 60 {
			return ErrSocialInvalid
		}
		for _, value := range []string{entry.Icon, entry.Background, entry.Gallery, entry.Cover} {
			if !asset(value) {
				return ErrSocialInvalid
			}
		}
		if entry.Video != "" && !asset(entry.Video) {
			return ErrSocialInvalid
		}
		for _, value := range []string{entry.Accent, entry.Hover, entry.Pressed} {
			if !color.MatchString(value) {
				return ErrSocialInvalid
			}
		}
		if entry.Target != "" && entry.Target != "minecraft" {
			return ErrSocialInvalid
		}
		for _, banner := range entry.Banners {
			if !asset(banner.Image) || len([]rune(banner.Title)) > 100 {
				return ErrSocialInvalid
			}
		}
		for _, news := range entry.News {
			validTab := false
			for _, tab := range entry.Tabs {
				if news.Tab == tab {
					validTab = true
				}
			}
			if !validTab || strings.TrimSpace(news.Title) == "" || len([]rune(news.Title)) > 150 || len([]rune(news.Body)) > 20000 || len(news.Images) > 8 || len(news.Date) > 30 {
				return ErrSocialInvalid
			}
			for _, image := range news.Images {
				if !asset(image) {
					return ErrSocialInvalid
				}
			}
		}
	}
	return nil
}

func (s *Store) DesktopLauncher(ctx context.Context) (DesktopLauncherSettings, error) {
	var v DesktopLauncherSettings
	var draft string
	var published sql.NullString
	var at sql.NullTime
	err := s.DB.QueryRowContext(ctx, "SELECT version,draft_json,published_json,published_version,updated_at,published_at FROM desktop_launcher_settings WHERE id=1").Scan(&v.Version, &draft, &published, &v.PublishedVersion, &v.UpdatedAt, &at)
	if err != nil {
		return v, err
	}
	if json.Unmarshal([]byte(draft), &v.Draft) != nil {
		return v, ErrSocialInvalid
	}
	if published.Valid {
		v.Published = &DesktopLauncherDocument{}
		if json.Unmarshal([]byte(published.String), v.Published) != nil {
			return v, ErrSocialInvalid
		}
	}
	if at.Valid {
		v.PublishedAt = &at.Time
	}
	return v, nil
}

func (s *Store) SeedDesktopLauncher(ctx context.Context, v DesktopLauncherDocument) error {
	if err := ValidateDesktopDocument(v); err != nil {
		return err
	}
	raw, _ := json.Marshal(v)
	_, err := s.DB.ExecContext(ctx, "UPDATE desktop_launcher_settings SET draft_json=?,updated_at=UTC_TIMESTAMP(6) WHERE id=1 AND version=1 AND draft_json='{}'", raw)
	return err
}

func (s *Store) SaveDesktopLauncher(ctx context.Context, actor, request string, expected int64, v DesktopLauncherDocument, publish bool) (json.RawMessage, error) {
	if expected < 1 {
		return nil, ErrSocialInvalid
	}
	if err := ValidateDesktopDocument(v); err != nil {
		return nil, err
	}
	if publish {
		if err := ValidateDesktopPublication(v); err != nil {
			return nil, err
		}
	}
	return s.socialMutate(ctx, actor, "desktop-launcher.settings", request, struct {
		Expected int64
		Document DesktopLauncherDocument
		Publish  bool
	}{expected, v, publish}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT version FROM desktop_launcher_settings WHERE id=1 FOR UPDATE").Scan(&version); err != nil {
			return nil, err
		}
		if version != expected {
			return nil, ErrSocialVersion
		}
		raw, _ := json.Marshal(v)
		now := time.Now().UTC().Truncate(time.Microsecond)
		if _, err := tx.ExecContext(ctx, "UPDATE desktop_launcher_settings SET version=version+1,draft_json=?,updated_at=? WHERE id=1", raw, now); err != nil {
			return nil, err
		}
		if publish {
			if _, err := tx.ExecContext(ctx, "UPDATE desktop_launcher_settings SET published_json=?,published_version=version,published_at=? WHERE id=1", raw, now); err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,?)", actor, "desktop-launcher.settings", "desktop-launcher", now); err != nil {
			return nil, err
		}
		return map[string]any{"version": version + 1, "published": publish}, nil
	})
}

func (s *Store) LatestDesktopSync(ctx context.Context) (*DesktopLauncherSync, error) {
	var v DesktopLauncherSync
	err := s.DB.QueryRowContext(ctx, "SELECT id,state,message,created_at,updated_at FROM desktop_launcher_sync_runs ORDER BY created_at DESC LIMIT 1").Scan(&v.ID, &v.State, &v.Message, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &v, err
}

func (s *Store) BeginDesktopSync(ctx context.Context, actor, request string) (json.RawMessage, error) {
	return s.socialMutate(ctx, actor, "desktop-launcher.sync", request, struct{}{}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT version FROM desktop_launcher_settings WHERE id=1 FOR UPDATE").Scan(&version); err != nil {
			return nil, err
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM desktop_launcher_sync_runs WHERE state='RUNNING'").Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrConflict
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		v := DesktopLauncherSync{ID: ID("ls_"), State: "RUNNING", Message: "正在调用 McPatch 同步到 OSS。", CreatedAt: now, UpdatedAt: now}
		if _, err := tx.ExecContext(ctx, "INSERT INTO desktop_launcher_sync_runs(id,actor_id,state,message,created_at,updated_at) VALUES(?,?,?,?,?,?)", v.ID, actor, v.State, v.Message, now, now); err != nil {
			return nil, err
		}
		return v, nil
	})
}

func (s *Store) FinishDesktopSync(ctx context.Context, id, state, message string, index []byte) error {
	if state != "SUCCEEDED" && state != "FAILED" {
		return ErrSocialInvalid
	}
	if len(message) > 3000 {
		message = message[:3000]
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current string
	if err = tx.QueryRowContext(ctx, "SELECT state FROM desktop_launcher_sync_runs WHERE id=? FOR UPDATE", id).Scan(&current); err != nil {
		return err
	}
	if current != "RUNNING" {
		return nil
	}
	if _, err = tx.ExecContext(ctx, "UPDATE desktop_launcher_settings SET published_index=? WHERE id=1 AND ?='SUCCEEDED'", string(index), state); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE desktop_launcher_sync_runs SET state=?,message=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state='RUNNING'", state, message, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DesktopPublishedIndex(ctx context.Context) (json.RawMessage, error) {
	var raw sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT published_index FROM desktop_launcher_settings WHERE id=1").Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || !json.Valid([]byte(raw.String)) {
		return json.RawMessage("[]"), nil
	}
	return json.RawMessage(raw.String), nil
}
