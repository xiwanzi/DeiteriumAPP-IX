package store

import "context"

func ValidPermission(permission string) bool {
	switch permission {
	case "core.read", "core.manage", "platform.admin", "announcements.manage", "moderation.read", "store.manage", "store.members.manage", "commerce.manage", "intervention.manage", "interventions.manage", "audit.read", "releases.manage", "appearance.manage":
		return true
	default:
		return false
	}
}

func (s *Store) PermissionsV2(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT permission FROM identity_permissions WHERE user_id=? ORDER BY permission", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	permissions := []string{}
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	return permissions, rows.Err()
}
