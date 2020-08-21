package reader

import (
	"testing"
)

func Test_OpenAndReadFile(t *testing.T) {
	expectedSerials := []string{"Test0-0", "Test0-1", "Test0-2", "Test0-3", "Test0-4"}

	result, err := OpenAndReadFile("../../testhelper/testlist.xlsx")
	if err != nil {
		t.Fatalf("Recieved error from test: %v", err)
	}
	for i := range expectedSerials {
		if expectedSerials[i] != result[i] {
			t.Fatalf("Expected value at position 0-%v to be %v, but it was %v", i, expectedSerials[i], result[i])
		}
	}
}
