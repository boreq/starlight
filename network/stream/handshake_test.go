package stream

import (
	"testing"
)

func TestSelectParam(t *testing.T) {
	var rv string

	rv = selectParam("a,b,c", "a,b,c")
	if rv != "a" {
		t.Error("Invalid result", rv)
	}

	rv = selectParam("a,b,c", "c,b,a")
	if rv != "a" {
		t.Error("Invalid result", rv)
	}

	rv = selectParam("a,b,c", "b,c")
	if rv != "b" {
		t.Error("Invalid result", rv)
	}

	rv = selectParam("a,b,c", "d,e,d")
	if rv != "" {
		t.Error("Invalid result", rv)
	}

	rv = selectParam("", "")
	if rv != "" {
		t.Error("Invalid result", rv)
	}
}
