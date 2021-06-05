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
			Name:                 "array-of-maps",
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
			Name:                 "array-of-maps-at-root",
			Path:                 "[0].id",
			Data:                 `[{"id": 1}, {"id": 2}]`,
			ExpectedOutput:       "1",
			ExpectedOutputLength: 1,
			ExpectedError:        false,
		},
		{
			Name:                 "array-of-maps-at-root-and-invalid-index",
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
			Name:                 "map-of-nested-arrays",
			Path:                 "data[1][1]",
			Data:                 `{"data": [["a", "b", "c"], ["d", "eeeee", "f"]]}`,
			ExpectedOutput:       "eeeee",
			ExpectedOutputLength: 5,
			ExpectedError:        false,
		},
		{
			Name:                 "partially-invalid-path-issue122",
			Path:                 "data.name.invalid",
			Data:                 `{"data": {"name": "john"}}`,
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
