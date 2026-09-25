package static

import (
	"io/fs"
	"strings"
	"testing"
)

// findAsset resolves a glob (e.g. "js/app*.js") to the single matching path in the built static
// FileSystem. filenameHashing in web/app/vue.config.js means the exact filename isn't fixed --
// this keeps the test correct whether hashing is on or off, instead of hardcoding "app.js".
func findAsset(t *testing.T, staticFileSystem fs.FS, pattern string) string {
	t.Helper()
	matches, err := fs.Glob(staticFileSystem, pattern)
	if err != nil {
		t.Fatalf("glob %s: %s", pattern, err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one match for %s, got %v", pattern, matches)
	}
	return matches[0]
}

func TestEmbed(t *testing.T) {
	staticFileSystem, err := fs.Sub(FileSystem, RootPath)
	if err != nil {
		t.Fatal(err)
	}
	scenarios := []struct {
		path                  string
		shouldExist           bool
		expectedContainString string
	}{
		{
			path:                  "index.html",
			shouldExist:           true,
			expectedContainString: "</body>",
		},
		{
			path:                  "favicon.ico",
			shouldExist:           true,
			expectedContainString: "", // not checking because it's an image
		},
		{
			path:                  findAsset(t, staticFileSystem, "img/logo*.svg"),
			shouldExist:           true,
			expectedContainString: "</svg>",
		},
		{
			path:                  findAsset(t, staticFileSystem, "css/app*.css"),
			shouldExist:           true,
			expectedContainString: "background-color",
		},
		{
			path:                  findAsset(t, staticFileSystem, "js/app*.js"),
			shouldExist:           true,
			expectedContainString: "function",
		},
		{
			path:                  findAsset(t, staticFileSystem, "js/chunk-vendors*.js"),
			shouldExist:           true,
			expectedContainString: "function",
		},
		{
			path:        "file-that-does-not-exist.html",
			shouldExist: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.path, func(t *testing.T) {
			content, err := fs.ReadFile(staticFileSystem, scenario.path)
			if !scenario.shouldExist {
				if err == nil {
					t.Errorf("%s should not have existed", scenario.path)
				}
			} else {
				if err != nil {
					t.Errorf("opening %s should not have returned an error, got %s", scenario.path, err.Error())
				}
				if len(content) == 0 {
					t.Errorf("%s should have existed in the static FileSystem, but was empty", scenario.path)
				}
				if !strings.Contains(string(content), scenario.expectedContainString) {
					t.Errorf("%s should have contained %s, but did not", scenario.path, scenario.expectedContainString)
				}
			}
		})
	}
}
