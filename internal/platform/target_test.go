//go:build !ios && !android

package platform

import "testing"

func TestCurrentTargetDefaultsToDesktop(t *testing.T) {
	if got := CurrentTarget(); got != TargetDesktop {
		t.Fatalf("CurrentTarget() = %q, want %q", got, TargetDesktop)
	}
}
