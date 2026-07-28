package platform

import (
	"slices"
	"testing"
)

func TestOpenFileCommandMatchesRuntimeTarget(t *testing.T) {
	const path = "test-library/song.gp"

	name, args, err := openFileCommand(path)
	if CurrentTarget() != TargetDesktop {
		if err == nil {
			t.Fatal("openFileCommand() error = nil on mobile, want unsupported error")
		}
		return
	}

	if err != nil {
		t.Fatalf("openFileCommand() error = %v", err)
	}
	if name == "" {
		t.Fatal("openFileCommand() returned an empty executable")
	}
	if !slices.Contains(args, path) {
		t.Fatalf("openFileCommand() args = %q, want path %q", args, path)
	}
}
