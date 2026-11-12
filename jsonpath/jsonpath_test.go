package jsonpath

import (
	"testing"
)

func TestEval(t *testing.T) {
	type Scenario struct {
		Name                 string
		Path                 string
		Data                 string
		ExpectedOutput       string
		ExpectedOutputLength int
		ExpectedError        bool
	}
	scenarios := []Scenario{
		{
			Name:                 "simple",
			Path:                 "key",
			Data:                 `{"key": "value"}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "simple-with-invalid-data",
			Path:                 "key",
			Data:                 "invalid data",
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "invalid-path",
			Path:                 "key",
			Data:                 `{}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "long-simple-walk",
			Path:                 "long.simple.walk",
			Data:                 `{"long": {"simple": {"walk": "value"}}}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-objects",
			Path:                 "ids[1].id",
			Data:                 `{"ids": [{"id": 1}, {"id": 2}]}`,
			ExpectedOutput:       "2",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-values",
			Path:                 "ids[0]",
			Data:                 `{"ids": [1, 2]}`,
			ExpectedOutput:       "1",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-values-with-no-path",
			Path:                 "",
			Data:                 `[1, 2]`,
			ExpectedOutput:       "[1 2]", // the output is an array
			ExpectedOutputLength: 2,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-values-and-invalid-index",
			Path:                 "ids[wat]",
			Data:                 `{"ids": [1, 2]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "array-of-values-at-root",
			Path:                 "[1]",
			Data:                 `[1, 2]`,
			ExpectedOutput:       "2",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-objects-at-root",
			Path:                 "[0]",
			Data:                 `[{"id": 1}, {"id": 2}]`,
			ExpectedOutput:       `{"id":1}`,
			ExpectedOutputLength: 8,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-objects-with-int-at-root",
			Path:                 "[0].id",
			Data:                 `[{"id": 1}, {"id": 2}]`,
			ExpectedOutput:       "1",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-objects-at-root-and-invalid-index",
			Path:                 "[5].id",
			Data:                 `[{"id": 1}, {"id": 2}]`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "long-walk-and-array",
			Path:                 "data.ids[0].id",
			Data:                 `{"data": {"ids": [{"id": 1}, {"id": 2}, {"id": 3}]}}`,
			ExpectedOutput:       "1",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "nested-array",
			Path:                 "[3][2]",
			Data:                 `[[1, 2], [3, 4], [], [5, 6, 7]]`,
			ExpectedOutput:       "7",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "object-with-nested-arrays",
			Path:                 "data[1][1]",
			Data:                 `{"data": [["a", "b", "c"], ["d", "eeeee", "f"]]}`,
			ExpectedOutput:       "eeeee",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "object-with-arrays-of-objects",
			Path:                 "data[0].apps[1].name",
			Data:                 `{"data": [{"apps": [{"name":"app1"}, {"name":"app2"}, {"name":"app3"}]}]}`,
			ExpectedOutput:       "app2",
			ExpectedOutputLength: 4,
			ExpectedError:        false,
		},
		{
			Name:                 "object-with-arrays-of-objects-with-missing-element",
			Path:                 "data[0].apps[1].name",
			Data:                 `{"data": [{"apps": []}]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "partially-invalid-path-issue122",
			Path:                 "data.name.invalid",
			Data:                 `{"data": {"name": "john"}}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "float-as-string",
			Path:                 "balance",
			Data:                 `{"balance": "123.40000000000005"}`,
			ExpectedOutput:       "123.40000000000005",
			ExpectedOutputLength: 18,
			ExpectedError:        false,
		},
		{
			Name:                 "float-as-number",
			Path:                 "balance",
			Data:                 `{"balance": 123.40000000000005}`,
			ExpectedOutput:       "123.40000000000005",
			ExpectedOutputLength: 18,
			ExpectedError:        false,
		},
		{
			Name:                 "negative-index-on-keyed-array",
			Path:                 "data[-1]",
			Data:                 `{"data": [1, 2, 3]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "negative-index-on-root-array",
			Path:                 "[-1]",
			Data:                 `[1, 2, 3]`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "negative-index-followed-by-key",
			Path:                 "data[-1].name",
			Data:                 `{"data": [{"name": "value"}]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "negative-index-nested-array",
			Path:                 "data[0][-1]",
			Data:                 `{"data": [[1, 2, 3]]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "double-quoted-key-with-dot",
			Path:                 `"m.server"`,
			Data:                 `{"m.server": "example.com:443"}`,
			ExpectedOutput:       "example.com:443",
			ExpectedOutputLength: 15,
			ExpectedError:        false,
		},
		{
			Name:                 "single-quoted-key-with-dot",
			Path:                 `'m.server'`,
			Data:                 `{"m.server": "example.com:443"}`,
			ExpectedOutput:       "example.com:443",
			ExpectedOutputLength: 15,
			ExpectedError:        false,
		},
		{
			Name:                 "bracket-double-quoted-key-with-dot",
			Path:                 `["m.server"]`,
			Data:                 `{"m.server": "example.com:443"}`,
			ExpectedOutput:       "example.com:443",
			ExpectedOutputLength: 15,
			ExpectedError:        false,
		},
		{
			Name:                 "bracket-single-quoted-key-with-dot",
			Path:                 `['m.server']`,
			Data:                 `{"m.server": "example.com:443"}`,
			ExpectedOutput:       "example.com:443",
			ExpectedOutputLength: 15,
			ExpectedError:        false,
		},
		{
			Name:                 "nested-quoted-key-with-dot",
			Path:                 `data."m.server".port`,
			Data:                 `{"data": {"m.server": {"port": 443}}}`,
			ExpectedOutput:       "443",
			ExpectedOutputLength: 3,
			ExpectedError:        false,
		},
		{
			Name:                 "nested-bracket-key-with-dot",
			Path:                 `data["m.server"].port`,
			Data:                 `{"data": {"m.server": {"port": 443}}}`,
			ExpectedOutput:       "443",
			ExpectedOutputLength: 3,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-with-dot-and-index",
			Path:                 `"a.b"[1]`,
			Data:                 `{"a.b": [1, 2]}`,
			ExpectedOutput:       "2",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "index-followed-by-bracket-key",
			Path:                 `data[0]["m.server"]`,
			Data:                 `{"data": [{"m.server": "value"}]}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "bracket-key-on-array",
			Path:                 `["m.server"]`,
			Data:                 `[1, 2]`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "unterminated-quoted-key",
			Path:                 `"m.server`,
			Data:                 `{"m.server": "value"}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "key-with-apostrophe",
			Path:                 `don't.stop`,
			Data:                 `{"don't": {"stop": "value"}}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "double-quoted-key-with-dots",
			Path:                 `status["core.integrations.key_1"]`,
			Data:                 `{"status": {"core.integrations.key_1": "ok", "core.key_2.out": "nok"}}`,
			ExpectedOutput:       "ok",
			ExpectedOutputLength: 2,
			ExpectedError:        false,
		},
		{
			Name:                 "single-quoted-key-with-dots",
			Path:                 `status['core.key_2.out']`,
			Data:                 `{"status": {"core.integrations.key_1": "ok", "core.key_2.out": "nok"}}`,
			ExpectedOutput:       "nok",
			ExpectedOutputLength: 3,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-with-dots-preceded-by-a-dot",
			Path:                 `status.["core.integrations.key_1"]`,
			Data:                 `{"status": {"core.integrations.key_1": "ok"}}`,
			ExpectedOutput:       "ok",
			ExpectedOutputLength: 2,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-with-dots-at-root",
			Path:                 `["key.with.dots"]`,
			Data:                 `{"key.with.dots": "value"}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-without-dots",
			Path:                 `["key"]`,
			Data:                 `{"key": "value"}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-with-dots-followed-by-a-key",
			Path:                 `["key.with.dots"].name`,
			Data:                 `{"key.with.dots": {"name": "john"}}`,
			ExpectedOutput:       "john",
			ExpectedOutputLength: 4,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-with-dots-followed-by-an-index",
			Path:                 `["key.with.dots"][1]`,
			Data:                 `{"key.with.dots": [10, 20]}`,
			ExpectedOutput:       "20",
			ExpectedOutputLength: 2,
			ExpectedError:        false,
		},
		{
			Name:                 "index-followed-by-a-quoted-key-with-dots",
			Path:                 `data[0]["key.with.dots"]`,
			Data:                 `{"data": [{"key.with.dots": "value"}]}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-with-dots-resolving-to-an-object",
			Path:                 `["key.with.dots"]`,
			Data:                 `{"key.with.dots": {"name": "john"}}`,
			ExpectedOutput:       `{"name":"john"}`,
			ExpectedOutputLength: 15,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-containing-a-bracket",
			Path:                 `["key].with.dots"]`,
			Data:                 `{"key].with.dots": "value"}`,
			ExpectedOutput:       "value",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "quoted-key-that-does-not-exist",
			Path:                 `status["does.not.exist"]`,
			Data:                 `{"status": {"core.integrations.key_1": "ok"}}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "quoted-key-applied-to-a-string",
			Path:                 `status["core.integrations.key_1"]`,
			Data:                 `{"status": "ok"}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "quoted-key-applied-to-an-array",
			Path:                 `data["core.integrations.key_1"]`,
			Data:                 `{"data": [1, 2]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			// Used to panic, because the key before the accessor was type-asserted to a map without checking
			Name:                 "key-with-accessor-against-an-array-body",
			Path:                 `data[0]`,
			Data:                 `[1, 2]`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "unquoted-key-with-dots-is-still-treated-as-separate-keys",
			Path:                 `status[core.integrations.key_1]`,
			Data:                 `{"status": {"core.integrations.key_1": "ok"}}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			output, outputLength, err := Eval(scenario.Path, []byte(scenario.Data))
			if (err != nil) != scenario.ExpectedError {
				if scenario.ExpectedError {
					t.Errorf("Expected error, got '%v'", err)
				} else {
					t.Errorf("Expected no error, got '%v'", err)
				}
			}
			if outputLength != scenario.ExpectedOutputLength {
				t.Errorf("Expected output length to be %v, but was %v", scenario.ExpectedOutputLength, outputLength)
			}
			if output != scenario.ExpectedOutput {
				t.Errorf("Expected output to be %v, but was %v", scenario.ExpectedOutput, output)
			}
		})
	}
}
