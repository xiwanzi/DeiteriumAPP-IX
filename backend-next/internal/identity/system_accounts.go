package identity

import "strings"

// Reserved by the owner for Core-controlled XConomy service identities.
func SystemGameID(name string) bool {
	return strings.EqualFold(name, "DIMA") || strings.EqualFold(name, "DaoYu")
}
func ValidQQ(value string) bool { return qqPattern.MatchString(value) }
