package security

import (
	"crypto/sha512"
	"fmt"
)

// Sha512 hashes a provided string using SHA512 and returns the resulting hash as a string
// Deprecated: Use bcrypt instead
func Sha512(s string) string {
	hash := sha512.New()
	hash.Write([]byte(s))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
