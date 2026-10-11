package store

import "context"

// YSMIdentity is an authorization projection of the existing website account.
// Neither this query nor Cloud login creates or binds a website account.
type YSMIdentity struct {
	UserID        string `json:"userId"`
	ProfileUUID   string `json:"profileUuid"`
	ModelAccess   bool   `json:"modelAccess"`
	Administrator bool   `json:"administrator"`
}

func (s *Store) YSMIdentity(ctx context.Context, uuid string) (v YSMIdentity, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT i.id,i.server_uuid,TRUE,
 EXISTS(SELECT 1 FROM identity_permissions p WHERE p.user_id=i.id AND p.permission='platform.admin')
 FROM identities i WHERE i.server_uuid=? AND i.game_verified=TRUE AND i.status='active'`, uuid).
		Scan(&v.UserID, &v.ProfileUUID, &v.ModelAccess, &v.Administrator)
	return
}
