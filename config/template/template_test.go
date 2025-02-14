package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// nolint: funlen
func TestTemplateFuncMap(t *testing.T) {
	testcases := []struct {
		name string
		vars interface{}
		tmpl string
		exp  string
		err  bool
	}{
		{
			name: "simple render",
			tmpl: `{{ .Key }}`,
			vars: map[string]interface{}{"Key": "Value"},
			exp:  "Value",
		},
		{
			name: "missing key",
			tmpl: `{{ .Key }}`,
			err:  true,
		},
		{
			name: "truncate",
			tmpl: `{{ "hello!" | toUpper | repeat 5 }}`,
			exp:  "HELLO!HELLO!HELLO!HELLO!HELLO!",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Render([]byte(tc.tmpl), tc.vars)

			if tc.err {
				require.Error(t, err, "expected an error but did not get one")

				return
			}

			require.NoError(t, err, "expected no error but got one")
			require.Equal(t, tc.exp, out.String())
		})
	}
}
