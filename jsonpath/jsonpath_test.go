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
			Name:                 "no-path-non-array",
			Path:                 "",
			Data:                 `{"key": "value"}`,
			ExpectedOutput:       `{"key":"value"}`,
			ExpectedOutputLength: 15,
			ExpectedError:        false,
		},
		{
			Name:                 "nil-value-in-map",
			Path:                 "key",
			Data:                 `{"key": null}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "array-as-final-value",
			Path:                 "data",
			Data:                 `{"data": [1, 2, 3]}`,
			ExpectedOutput:       "[1 2 3]",
			ExpectedOutputLength: 3,
			ExpectedError:        false,
		},
		{
			Name:                 "nested-array-out-of-bounds",
			Path:                 "data[0][1]",
			Data:                 `{"data": [["a"]]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "nil-map-value-before-array",
			Path:                 "data[0]",
			Data:                 `{"data": null}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "boolean-value",
			Path:                 "flag",
			Data:                 `{"flag": true}`,
			ExpectedOutput:       "true",
			ExpectedOutputLength: 4,
			ExpectedError:        false,
		},
		{
			Name:                 "primitive-root-with-invalid-path",
			Path:                 "key",
			Data:                 `"hello"`,
			ExpectedOutput:       "hello",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "empty-path-with-primitive",
			Path:                 "",
			Data:                 `"hello"`,
			ExpectedOutput:       "hello",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "empty-path-with-number",
			Path:                 "",
			Data:                 `42`,
			ExpectedOutput:       "42",
			ExpectedOutputLength: 2,
			ExpectedError:        false,
		},
		{
			Name:                 "empty-path-with-boolean",
			Path:                 "",
			Data:                 `true`,
			ExpectedOutput:       "true",
			ExpectedOutputLength: 4,
			ExpectedError:        false,
		},
		{
			Name:                 "malformed-path-unclosed-bracket",
			Path:                 "data[0",
			Data:                 `{"data": [1, 2]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "malformed-path-missing-close-bracket",
			Path:                 "[0",
			Data:                 `[1, 2]`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "invalid-array-syntax-no-index",
			Path:                 "data[]",
			Data:                 `{"data": [1, 2]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "double-dot",
			Path:                 "data..value",
			Data:                 `{"data": {"value": "test"}}`,
			ExpectedOutput:       "test",
			ExpectedOutputLength: 4,
			ExpectedError:        false,
		},
		{
			Name:                 "array-negative-index",
			Path:                 "data[-1]",
			Data:                 `{"data": [1, 2, 3]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "empty-array",
			Path:                 "data",
			Data:                 `{"data": []}`,
			ExpectedOutput:       "[]",
			ExpectedOutputLength: 2,
			ExpectedError:        false,
		},
		{
			Name:                 "unexpected-token-bracket-close",
			Path:                 "data].key",
			Data:                 `{"data": {"key": "value"}}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "key-on-array",
			Path:                 "data.key",
			Data:                 `{"data": [1, 2, 3]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "array-access-on-object",
			Path:                 "data[0]",
			Data:                 `{"data": {"key": "value"}}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "nil-value-in-array",
			Path:                 "data[1]",
			Data:                 `{"data": [1, null, 3]}`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "root-array-out-of-bounds-initial",
			Path:                 "[3]",
			Data:                 `[1, 2]`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "standalone-index-token",
			Path:                 "0",
			Data:                 `{"0": "zero"}`,
			ExpectedOutput:       "zero",
			ExpectedOutputLength: 4,
			ExpectedError:        false,
		},
		{
			Name:                 "int-value",
			Path:                 "number",
			Data:                 `{"number": 42}`,
			ExpectedOutput:       "42",
			ExpectedOutputLength: 2,
			ExpectedError:        false,
		},
		{
			Name:                 "nil-as-final-value",
			Path:                 "",
			Data:                 `null`,
			ExpectedOutput:       "null",
			ExpectedOutputLength: 4,
			ExpectedError:        false,
		},
		{
			Name:                 "array-with-nil-element-as-final",
			Path:                 "data",
			Data:                 `{"data": [1, null, 3]}`,
			ExpectedOutput:       "[1 null 3]",
			ExpectedOutputLength: 3,
			ExpectedError:        false,
		},
		{
			Name:                 "nested-map-as-final",
			Path:                 "data",
			Data:                 `{"data": {"inner": {"key": "value"}}}`,
			ExpectedOutput:       `{"inner":{"key":"value"}}`,
			ExpectedOutputLength: 25,
			ExpectedError:        false,
		},
		{
			Name:                 "root-array-out-of-bounds-redundant",
			Path:                 "[10]",
			Data:                 `[1, 2, 3]`,
			ExpectedOutput:       "",
			ExpectedOutputLength: 0,
			ExpectedError:        true,
		},
		{
			Name:                 "complex-nested-array",
			Path:                 "data[0]",
			Data:                 `{"data": [[1, 2, 3]]}`,
			ExpectedOutput:       "[1 2 3]",
			ExpectedOutputLength: 3,
			ExpectedError:        false,
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
