package static

import (
	"io/fs"
	"strings"
	"testing"
)

func TestEmbed(t *testing.T) {
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
			path:                  "img/logo.svg",
			shouldExist:           true,
			expectedContainString: "</svg>",
		},
		{
			path:                  "css/app.css",
			shouldExist:           true,
			expectedContainString: "background-color",
		},
		{
			path:                  "js/app.js",
			shouldExist:           true,
			expectedContainString: "function",
		},
		{
			path:                  "js/chunk-vendors.js",
			shouldExist:           true,
			expectedContainString: "function",
		},
		{
			path:        "file-that-does-not-exist.html",
			shouldExist: false,
		},
	}
	staticFileSystem, err := fs.Sub(FileSystem, RootPath)
	if err != nil {
		t.Fatal(err)
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
