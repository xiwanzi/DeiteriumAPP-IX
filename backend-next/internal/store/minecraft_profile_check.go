package store

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) CachedMinecraftProfile(ctx context.Context, name string) (AdmissionProfile, error) {
	var uuid string
	err := s.DB.QueryRowContext(ctx, "SELECT COALESCE(profile_uuid,'') FROM minecraft_profile_checks WHERE game_id=LOWER(?) AND expires_at>UTC_TIMESTAMP(6)", name).Scan(&uuid)
	if errors.Is(err, sql.ErrNoRows) || err == nil && !ValidAdmissionUUID(uuid) {
		return AdmissionProfile{}, sql.ErrNoRows
	}
	return AdmissionProfile{UUID: uuid, Name: name}, err
}

func (s *Store) CacheMinecraftProfileCheck(ctx context.Context, name, uuid string) error {
	if !ValidAdmissionUUID(uuid) {
		return ErrSocialInvalid
	}
	_, err := s.DB.ExecContext(ctx, "INSERT INTO minecraft_profile_checks(game_id,expires_at,profile_uuid) VALUES(LOWER(?),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 10 MINUTE),?) ON DUPLICATE KEY UPDATE expires_at=VALUES(expires_at),profile_uuid=VALUES(profile_uuid)", name, uuid)
	return err
}
