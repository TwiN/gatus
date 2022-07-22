package security

import "testing"

func TestBasicConfig_IsValidUsingBcrypt(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:                        "admin",
		PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
	}
	if !basicConfig.isValid() {
		t.Error("basicConfig should've been valid")
	}
}

func TestBasicConfig_IsValidWhenPasswordIsInvalidUsingBcrypt(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:                        "admin",
		PasswordBcryptHashBase64Encoded: "",
	}
	if basicConfig.isValid() {
		t.Error("basicConfig shouldn't have been valid")
	}
}
