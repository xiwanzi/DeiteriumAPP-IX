package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrHash = errors.New("unsupported password hash")

type hashParameters struct {
	kind           string
	memory, rounds uint32
	threads        uint8
	salt, hash     []byte
}

// Only PHC Argon2 v19 is imported. Bounds are checked before allocating KDF memory.
func parseHash(encoded string) (p hashParameters, err error) {
	if len(encoded) > 255 {
		return p, ErrHash
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || (parts[1] != "argon2i" && parts[1] != "argon2id") || parts[2] != "v=19" {
		return p, ErrHash
	}
	p.kind = parts[1]
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return p, ErrHash
	}
	values := make([]uint64, 3)
	for i, prefix := range []string{"m=", "t=", "p="} {
		if !strings.HasPrefix(params[i], prefix) {
			return p, ErrHash
		}
		values[i], err = strconv.ParseUint(strings.TrimPrefix(params[i], prefix), 10, 32)
		if err != nil {
			return p, ErrHash
		}
	}
	if values[0] < 19456 || values[0] > 65536 || values[1] < 2 || values[1] > 5 || values[2] < 1 || values[2] > 4 {
		return p, ErrHash
	}
	p.memory, p.rounds, p.threads = uint32(values[0]), uint32(values[1]), uint8(values[2])
	p.salt, err = base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(p.salt) < 16 || len(p.salt) > 32 {
		return p, ErrHash
	}
	p.hash, err = base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(p.hash) != 32 {
		return p, ErrHash
	}
	return p, nil
}

func ValidateHash(encoded string) error { _, err := parseHash(encoded); return err }

func Verify(encoded, password string) bool {
	p, err := parseHash(encoded)
	if err != nil {
		return false
	}
	var actual []byte
	if p.kind == "argon2i" {
		actual = argon2.Key([]byte(password), p.salt, p.rounds, p.memory, p.threads, uint32(len(p.hash)))
	} else {
		actual = argon2.IDKey([]byte(password), p.salt, p.rounds, p.memory, p.threads, uint32(len(p.hash)))
	}
	defer clear(actual)
	return subtle.ConstantTimeCompare(actual, p.hash) == 1
}

func Hash(password string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	key := argon2.IDKey([]byte(password), salt, 3, 65536, 2, 32)
	defer clear(key)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}

func NeedsUpgrade(encoded string) bool {
	return !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=2$")
}

func Secret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
