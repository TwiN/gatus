package security

import "testing"

func TestSha512(t *testing.T) {
	input := "password"
	expectedHash := "b109f3bbbc244eb82441917ed06d618b9008dd09b3befd1b5e07394c706a8bb980b1d7785e5976ec049b46df5f1326af5a2ea6d103fd07c95385ffab0cacbc86"
	hash := Sha512(input)
	if hash != expectedHash {
		t.Errorf("Expected hash to be '%s', but was '%s'", expectedHash, hash)
	}
}
