package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/TwiN/gatus/v5/config"
	"github.com/invopop/jsonschema"
)

func main() {
	fmt.Println("Generate JSON-schema from config types")

	filepath := "config.schema.json"

	reflector := &jsonschema.Reflector{
		FieldNameTag: "yaml",
		Mapper: func(t reflect.Type) *jsonschema.Schema {
			name := t.Name()
			pkgPath := t.PkgPath()

			if pkgPath == "time" && name == "Duration" {
				return &jsonschema.Schema{
					Type: "string",
					// Regex from time.ParseDuration()
					// https://cs.opensource.google/go/go/+/master:src/time/format.go;l=1635
					Pattern: "[-+]?([0-9]*(\\.[0-9]*)?[a-z]+)+",
				}
			}

			return nil
		},
		Namer: func(t reflect.Type) string {
			name := t.Name()
			pkgPath := t.PkgPath()

			// required since multiple type names are identical (like "Config")
			return getTypeNameWithoutConflict(name, pkgPath)
		},
	}

	if err := reflector.AddGoComments("github.com/TwiN/gatus/v5", "./"); err != nil {
		fmt.Println(err)
		return
	}

	schema := reflector.Reflect(&config.Config{})

	bytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = os.WriteFile(filepath, bytes, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("File generated:", filepath)
}

var pathsByName = map[string]string{}

func getTypeNameWithoutConflict(name string, pkgPath string) string {
	// native type
	if pkgPath == "" {
		return name
	}

	existingPath, exists := pathsByName[name]

	// first type occurence
	if !exists {
		pathsByName[name] = pkgPath
		return name
	}

	if existingPath == pkgPath {
		return name
	}

	pathParts := strings.Split(pkgPath, "/")

	directory := pathParts[len(pathParts)-1]
	directoryChars := strings.Split(directory, "")

	// get directory name in PascalCase
	prefix := strings.ToUpper(directoryChars[0]) + strings.Join(directoryChars[1:], "")
	if prefix != name {
		// Directory + Name
		name = prefix + name
	}

	pkgPath = strings.Join(pathParts[:1], "/")

	return getTypeNameWithoutConflict(name, pkgPath)
}
