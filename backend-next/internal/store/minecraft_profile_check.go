package store

import "context"

func (s *Store) MinecraftProfileChecked(ctx context.Context, name string) (bool, error) {
	var checked bool
	err := s.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM minecraft_profile_checks WHERE game_id=LOWER(?) AND expires_at>UTC_TIMESTAMP(6))", name).Scan(&checked)
	return checked, err
}

func (s *Store) CacheMinecraftProfileCheck(ctx context.Context, name string) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO minecraft_profile_checks(game_id,expires_at) VALUES(LOWER(?),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 10 MINUTE)) ON DUPLICATE KEY UPDATE expires_at=VALUES(expires_at)", name)
	return err
}
