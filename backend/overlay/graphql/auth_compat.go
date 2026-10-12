package graphql

import (
	"crypto/subtle"
	"encoding/base64"
	"golang.org/x/crypto/argon2"
	"strconv"
	"strings"
)

// Rust accounts use PHC-encoded Argon2. Verification needs the original password;
// a copied hash cannot be converted into Go's SHAKE256 representation offline.
func verifyLegacyPassword(encoded, password string) bool {
	fields := strings.Split(encoded, "$")
	if len(fields) != 6 || fields[0] != "" || fields[2] != "v=19" || (fields[1] != "argon2id" && fields[1] != "argon2i") {
		return false
	}
	params := map[string]uint64{}
	for _, entry := range strings.Split(fields[3], ",") {
		pair := strings.Split(entry, "=")
		if len(pair) != 2 {
			return false
		}
		if pair[0] != "m" && pair[0] != "t" && pair[0] != "p" {
			return false
		}
		if _, exists := params[pair[0]]; exists {
			return false
		}
		value, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return false
		}
		params[pair[0]] = value
	}
	memory, iterations, parallel := params["m"], params["t"], params["p"]
	if parallel < 1 || parallel > 8 || memory < 8*parallel || memory > 131072 || iterations < 1 || iterations > 6 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(fields[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(fields[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return false
	}
	var actual []byte
	if fields[1] == "argon2id" {
		actual = argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory), uint8(parallel), uint32(len(expected)))
	} else {
		actual = argon2.Key([]byte(password), salt, uint32(iterations), uint32(memory), uint8(parallel), uint32(len(expected)))
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
