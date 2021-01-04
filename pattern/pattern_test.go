package pattern

import (
	"fmt"
	"testing"
)

func TestMatch(t *testing.T) {
	testMatch(t, "*", "livingroom_123", true)
	testMatch(t, "**", "livingroom_123", true)
	testMatch(t, "living*", "livingroom_123", true)
	testMatch(t, "*living*", "livingroom_123", true)
	testMatch(t, "*123", "livingroom_123", true)
	testMatch(t, "*_*", "livingroom_123", true)
	testMatch(t, "living*_*3", "livingroom_123", true)
	testMatch(t, "living*room_*3", "livingroom_123", true)
	testMatch(t, "living*room_*3", "livingroom_123", true)
	testMatch(t, "*vin*om*2*", "livingroom_123", true)
	testMatch(t, "livingroom_123", "livingroom_123", true)
	testMatch(t, "*livingroom_123*", "livingroom_123", true)
	testMatch(t, "*test*", "\\test", true)
	testMatch(t, "livingroom", "livingroom_123", false)
	testMatch(t, "livingroom123", "livingroom_123", false)
	testMatch(t, "what", "livingroom_123", false)
	testMatch(t, "*what*", "livingroom_123", false)
	testMatch(t, "*.*", "livingroom_123", false)
	testMatch(t, "room*123", "livingroom_123", false)
}

func testMatch(t *testing.T, pattern, key string, expectedToMatch bool) {
	t.Run(fmt.Sprintf("pattern '%s' from '%s'", pattern, key), func(t *testing.T) {
		matched := Match(pattern, key)
		if expectedToMatch {
			if !matched {
				t.Errorf("%s should've matched pattern '%s'", key, pattern)
			}
		} else {
			if matched {
				t.Errorf("%s shouldn't have matched pattern '%s'", key, pattern)
			}
		}
	})
}
