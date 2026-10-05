package shellword

import (
	"reflect"
	"testing"
)

func TestSplit(t *testing.T) {
	got := Split(`grep -e "a b" 'c d' e\ f "x\"y" *.go`)
	want := []Word{
		{"grep", "grep"}, {"-e", "-e"}, {`"a b"`, "a b"}, {`'c d'`, "c d"},
		{`e\ f`, "e f"}, {`"x\"y"`, `x"y`}, {"*.go", "*.go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Split = %#v", got)
	}
}
