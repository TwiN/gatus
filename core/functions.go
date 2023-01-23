package core

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"regexp"
)

// Functions
const (
	// An regexp to match the load() function from the passed value
	LoadFuncRegExp = `load\({1}'?([^']+)'?\){1}`
)

var (
	ErrResolvingHomeFolder    = errors.New("not able to replace '~' to user home folder")
	ErrFileNotExist           = errors.New("file not exists")
	ErrNotAbleToLoadFile			= errors.New("not able to load a file")
)

// HandleLoadFuctionIfExist tries to match the load() function that takes 1 argument, which is a plain text file path
// to load. If exists, load(<file path> will be replaced to the loaded text context) out of the given text value 
// then returns it. If not found, the passed value will be returned. 
// You can prepend "~" in the path to indicate user home folder. 
func HandleLoadFuctionIfExist(value string) (string, bool, error) {
	reg := regexp.MustCompile(LoadFuncRegExp)
	data := reg.FindSubmatch([]byte(value))

	retStr := value
	exist := false
	if data != nil {
		exist = true
		var path string
		i := 0
		for _, one := range data {					
			if i == 1 {
				path = string(one)
			}
			i++
		}

		if loadedText, err := loadPlainText(path, true); err != nil {
			return loadedText, exist, err
		} else {
			retStr = reg.ReplaceAllString(value, loadedText)
		}
	}
	return retStr, exist, nil
}

func loadPlainText(fpath string, stripNewlineSuffix bool) (string, error) {
	fpath = strings.TrimSpace(fpath)
	if strings.HasPrefix(fpath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
				return "", fmt.Errorf("%v: %w", ErrResolvingHomeFolder, err)
		}
		fpath = strings.ReplaceAll(fpath, "~", homeDir)
	}

	// load the plain text file
	_, err := os.Stat(fpath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("%v: %w", ErrFileNotExist, err)
	}
	fileBytes, err := os.ReadFile(fpath)
	if err != nil {
		return "", fmt.Errorf("%v: %w", ErrNotAbleToLoadFile, err)
	}
	retStr := string(fileBytes)

	// strip the new line if the file ends with the trailing new line.
	if stripNewlineSuffix {
		retStr = strings.TrimSuffix(retStr, "\n")
	}
	return retStr, nil
}
