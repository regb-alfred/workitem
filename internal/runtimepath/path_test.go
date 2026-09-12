package runtimepath_test

import (
	"strings"
	"testing"

	"github.com/regb/workitem/internal/runtimepath"
)

func TestControlSocketPathIsBounded(t *testing.T) {
	path := runtimepath.ControlSocket("01KZYHGDCECSFS4BJ2SNTQP49V", strings.Repeat("runtime-", 30))
	if len(path) > 64 || !strings.HasSuffix(path, ".sock") {
		t.Fatalf("path = %q (%d bytes)", path, len(path))
	}
}

func TestEventSocketDiffersFromControlSocket(t *testing.T) {
	itemID, runtimeID := "01KZYHGDCECSFS4BJ2SNTQP49V", strings.Repeat("runtime-", 30)
	control, event := runtimepath.ControlSocket(itemID, runtimeID), runtimepath.EventSocket(itemID, runtimeID)
	if control == event || len(event) > 72 || !strings.HasSuffix(event, ".sock") {
		t.Fatalf("control=%q event=%q", control, event)
	}
}
