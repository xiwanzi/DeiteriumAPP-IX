package identity

import "strings"

// Historical display data is retained without granting an invalid QQ a login
// alias. New registrations still use the strict numeric QQ validator.
func validLegacyQQ(value string) bool {
	if len(value) < 1 || len(value) > 20 {
		return false
	}
	for _, r := range value {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}
func legacyAliases(u LegacyUser) []string {
	aliases := []string{strings.ToLower(u.GameID)}
	if qqPattern.MatchString(u.QQ) && u.QQ != aliases[0] {
		aliases = append(aliases, u.QQ)
	}
	return aliases
}
