package util

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"regexp"
	"errors"

	"gopkg.in/yaml.v3"
)

type Endpoint struct {
	Url string `yaml:"url"`
	Grpc struct {
		Service string `yaml:"service"`
		Method string `yaml:"method"`
		Body string `yaml:"body"`
	} `yaml:"grpc"`
	Client struct {
		Insecure bool `yaml:"insecure"`
		Timeout string `yaml:"timeout"`
		Cert string `yaml:"cert"`
	} `yaml:"client"`
	Headers []string `yaml:"headers"`
}
type Config struct {
  Endpoints []Endpoint `yaml:"endpoints"`
}

func LoadConfig(yFilePath string) Config {
	var config Config

	yfile, err1 := ioutil.ReadFile(yFilePath)
	if err1 != nil {
    log.Fatal(err1)
  }

	err2 := yaml.Unmarshal(yfile, &config)
	if err2 != nil {
		log.Fatal(err2)
	}

	return config
}

// define own flag.Value and use flag.Var() for binding it
// credit: https://stackoverflow.com/questions/28322997/how-to-get-a-list-of-values-into-a-flag-in-golang
type ArrayFlags []string
func (i *ArrayFlags) String() string {
  return "string representation"
}

func (i *ArrayFlags) Set(value string) error {
  *i = append(*i, value)
  return nil
}

func PrintStringArray(title string, dataArray *[]string) {
	if len(*dataArray) > 0 { 
    fmt.Printf("\n[%v: %v]\n", title, len(*dataArray))
    for i := range *dataArray {
      str := (*dataArray)[i]
      fmt.Printf("%v\n", str)
    }
  }
}

func HandleLoadFuction(value string) string {
	reg := regexp.MustCompile(`load\({1}'?([^']+)'?\){1}`)
	data := reg.FindSubmatch([]byte(value))

	// load the file if the given value contains load() function
	if data != nil {
		var path string
		i := 0
		for _, one := range data {					
			if i == 1 {
				path = string(one)
			}
			i++
		}
		log.Println("loading text data from ", path)
		loadedText := LoadPlainText(path)
		value = reg.ReplaceAllString(value, loadedText)
	}

	return value
}

func LoadPlainText(fpath string) string {
	// handle the prepended ~ in the given file path
	fpath = strings.TrimSpace(fpath)
	if strings.HasPrefix(fpath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
				log.Fatal(err)
		}
		fpath = strings.ReplaceAll(fpath, "~", homeDir)
	}

	// load the plain text file
	fileBytes, err := os.ReadFile(fpath)
	if err != nil {
		log.Fatalf("error loading %s", fpath)
	}

	retStr := string(fileBytes)

	// strip the new line if the file ends with the trailing new line.
	return strings.TrimSuffix(retStr, "\n")
}

func PreProsessor(headers ArrayFlags) (ArrayFlags, error) {
	var retHeaders ArrayFlags

	if len(headers) > 0 {
		for _, header := range headers {
			keyValue := strings.Split(header, ":")
			if len(keyValue) != 2 || keyValue[0] == "" || keyValue[1] == "" {
				return nil, errors.New("Invalid header format: "+header)
			}

			key := strings.TrimSpace(keyValue[0])
			value := strings.TrimSpace(keyValue[1])

			// handle the pre-defined functions in case a header value contains a build-in function 
			value = HandleLoadFuction(value)	// currently only load() is available

			retHeaders.Set(key+":"+value)
		}
	}

	return retHeaders, nil
}
