package template

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

// Render renders the given content using the sprig template functions.
func Render(content []byte, vars interface{}) (bytes.Buffer, error) {
	handler := sprout.New()
	if err := handler.AddGroups(all.RegistryGroup()); err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to add sprout groups: %w", err)
	}

	var buf bytes.Buffer

	tpl, err := template.New("template").
		Option("missingkey=error").
		Funcs(handler.Build()).
		Parse(string(content))
	if err != nil {
		return buf, err
	}

	if err := tpl.Execute(&buf, vars); err != nil {
		return buf, err
	}

	return buf, nil
}
