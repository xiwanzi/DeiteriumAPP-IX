package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

const LauncherIconMinVersionCode = 20800

type LauncherIconSettings struct {
	IconID            string    `json:"iconId"`
	Version           int64     `json:"version"`
	MinAppVersionCode int       `json:"minAppVersionCode"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type LauncherIconOption struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	MinAppVersionCode int    `json:"minAppVersionCode"`
}

func LauncherIconOptions() []LauncherIconOption {
	return []LauncherIconOption{
		{"default", "默认图标", "白底黑色 · 日常使用", LauncherIconMinVersionCode},
		{"anniversary_911", "双子节图标", "911双子节 · 周年活动", LauncherIconMinVersionCode},
	}
}

func validLauncherIcon(id string) bool {
	return id == "default" || id == "anniversary_911"
}

func (s *Store) LauncherIcon(ctx context.Context) (LauncherIconSettings, error) {
	v := LauncherIconSettings{MinAppVersionCode: LauncherIconMinVersionCode}
	err := s.DB.QueryRowContext(ctx, "SELECT icon_id,version,updated_at FROM app_launcher_icon_settings WHERE id=1").Scan(&v.IconID, &v.Version, &v.UpdatedAt)
	return v, err
}

func (s *Store) SaveLauncherIcon(ctx context.Context, actor, request string, expected int64, iconID string) (json.RawMessage, error) {
	if expected < 1 || !validLauncherIcon(iconID) {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "app.launcher-icon", request, struct {
		Expected int64
		IconID   string
	}{expected, iconID}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		v := LauncherIconSettings{MinAppVersionCode: LauncherIconMinVersionCode}
		if err := tx.QueryRowContext(ctx, "SELECT icon_id,version,updated_at FROM app_launcher_icon_settings WHERE id=1 FOR UPDATE").Scan(&v.IconID, &v.Version, &v.UpdatedAt); err != nil {
			return nil, err
		}
		if v.Version != expected {
			return nil, ErrSocialVersion
		}
		if v.IconID == iconID {
			return v, nil
		}
		v.IconID = iconID
		v.Version++
		v.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
		if _, err := tx.ExecContext(ctx, "UPDATE app_launcher_icon_settings SET icon_id=?,version=?,updated_at=? WHERE id=1", v.IconID, v.Version, v.UpdatedAt); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'app.launcher-icon',?,?)", actor, iconID, v.UpdatedAt); err != nil {
			return nil, err
		}
		return v, nil
	})
}
