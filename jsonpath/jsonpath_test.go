package jsonpath

import "testing"

func TestEval(t *testing.T) {
	path := "simple"
	data := `{"simple": "value"}`

	expectedOutput := "value"

	output, outputLength, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if outputLength != len(expectedOutput) {
		t.Errorf("Expected output length to be %v, but was %v", len(expectedOutput), outputLength)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithInvalidData(t *testing.T) {
	path := "simple"
	data := `invalid data`
	_, _, err := Eval(path, []byte(data))
	if err == nil {
		t.Error("expected an error")
	}
}

func TestEvalWithInvalidPath(t *testing.T) {
	path := "errors"
	data := `{}`
	_, _, err := Eval(path, []byte(data))
	if err == nil {
		t.Error("Expected error, but got", err)
	}
}

func TestEvalWithLongSimpleWalk(t *testing.T) {
	path := "long.simple.walk"
	data := `{"long": {"simple": {"walk": "value"}}}`

	expectedOutput := "value"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithArrayOfMaps(t *testing.T) {
	path := "ids[1].id"
	data := `{"ids": [{"id": 1}, {"id": 2}]}`

	expectedOutput := "2"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}

	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithArrayOfValues(t *testing.T) {
	path := "ids[0]"
	data := `{"ids": [1, 2]}`

	expectedOutput := "1"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithArrayOfValuesAndInvalidIndex(t *testing.T) {
	path := "ids[wat]"
	data := `{"ids": [1, 2]}`

	_, _, err := Eval(path, []byte(data))
	if err == nil {
		t.Error("Expected an error")
	}
}

func TestEvalWithRootArrayOfValues(t *testing.T) {
	path := "[1]"
	data := `[1, 2]`

	expectedOutput := "2"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithRootArrayOfMaps(t *testing.T) {
	path := "[0].id"
	data := `[{"id": 1}, {"id": 2}]`

	expectedOutput := "1"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithRootArrayOfMapsUsingInvalidArrayIndex(t *testing.T) {
	path := "[5].id"
	data := `[{"id": 1}, {"id": 2}]`

	_, _, err := Eval(path, []byte(data))
	if err == nil {
		t.Error("Should've returned an error, but didn't")
	}
}

func TestEvalWithLongWalkAndArray(t *testing.T) {
	path := "data.ids[0].id"
	data := `{"data": {"ids": [{"id": 1}, {"id": 2}, {"id": 3}]}}`

	expectedOutput := "1"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithNestedArray(t *testing.T) {
	path := "[3][2]"
	data := `[[1, 2], [3, 4], [], [5, 6, 7]]`

	expectedOutput := "7"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}

func TestEvalWithMapOfNestedArray(t *testing.T) {
	path := "data[1][1]"
	data := `{"data": [["a", "b", "c"], ["d", "e", "f"]]}`

	expectedOutput := "e"

	output, _, err := Eval(path, []byte(data))
	if err != nil {
		t.Error("Didn't expect any error, but got", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output to be %v, but was %v", expectedOutput, output)
	}
}
