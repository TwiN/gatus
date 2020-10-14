package security

import (
	"crypto/sha512"
	"fmt"
)

func Sha512(s string) string {
	hash := sha512.New()
	hash.Write([]byte(s))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
