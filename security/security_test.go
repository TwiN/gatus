package security

import "testing"

func TestBasicConfig_IsValid(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:           "admin",
		PasswordSha512Hash: Sha512("test"),
	}
	if !basicConfig.IsValid() {
		t.Error("basicConfig should've been valid")
	}
}

func TestBasicConfig_IsValidWhenPasswordIsInvalid(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:           "admin",
		PasswordSha512Hash: "",
	}
	if basicConfig.IsValid() {
		t.Error("basicConfig shouldn't have been valid")
	}
}
