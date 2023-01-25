package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
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
	ErrDirDetected						= errors.New("expected a file path but a dir passed")
)

type LoadedFile struct {
	filePath string						// file path
	lastFileModTime	time.Time // last modification time
	content string						// file context (text only) 
}

// UpdateLastFileModTime refreshes LoadedFile.lastFileModTime
func (loadedFile *LoadedFile) UpdateLastFileModTime() {
	loadedFile.lastFileModTime = time.Now()
}

func (loadedFile *LoadedFile) UpdateContent(content string) {
	loadedFile.content = content
}

func (loadedFile *LoadedFile) HasLoadedFileBeenModified() bool {
	lastMod := loadedFile.lastFileModTime.Unix()
	fileInfo, err := os.Stat(loadedFile.filePath)
	if err != nil {
		return false
	}

	fmt.Println(fileInfo)
	return !fileInfo.ModTime().IsZero() && lastMod < fileInfo.ModTime().Unix()
}

// The 1st param, loadedFiles is a type of cache. If cache miss happens or cache hit but modified, 
// the file will be reloaded into the loadedFiles. 
func loadFileIfModified(loadedFiles map[string]LoadedFile, path string) (*LoadedFile, error) {
	var shouldUpdate bool = false		// found in cache but need to update
	
	loadedFile, ok := loadedFiles[strings.TrimSpace(path)]
	if !ok {
		loadedFile = LoadedFile {
			filePath: path,
		}
	}

	// determine if it should be updated
	for _, v := range loadedFiles {
		if v.HasLoadedFileBeenModified() {
			shouldUpdate = true
			break
		}
	}

	if shouldUpdate {
		if loadedText, err := loadPlainText(loadedFile.filePath, true); err != nil {
			return nil, err
		} else {
			// update the cache
			loadedFile.UpdateLastFileModTime()
			loadedFile.UpdateContent(loadedText)
		}
	}

	loadedFiles[path] = loadedFile
	return &loadedFile, nil
}

// HandleLoadFuctionIfExist tries to match the load() function that takes 1 argument, which is a plain text file path
// to load. If exists, load(<file path>) will be replaced to the loaded text. If not found, the passed value will be 
// returned back. You can prepend "~" in the path to indicate user home folder. 
func HandleLoadFuctionIfExist(loadedFiles map[string]LoadedFile, value string) (string, error) {
	reg := regexp.MustCompile(LoadFuncRegExp)
	data := reg.FindSubmatch([]byte(value))
	var loadedFile *LoadedFile
	var retStr string = value

	if data != nil {
		var path string
		i := 0
		for _, one := range data {					
			if i == 1 {
				path = string(one)
			}
			i++
		}

		if strings.HasPrefix(path, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
					return "", fmt.Errorf("%v: %w", ErrResolvingHomeFolder, err)
			}
			path = strings.ReplaceAll(path, "~", homeDir)
		}

		var err error
		if loadedFile, err = loadFileIfModified(loadedFiles, path); err != nil {
			return value, err
		} else {
			retStr = reg.ReplaceAllString(value, loadedFile.content)
		}	
	}
	// if the passed value doesn't contain, (value, nil, nil) will be returned.
	return retStr, nil
}

func loadPlainText(fpath string, stripNewlineSuffix bool) (string, error) {
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
