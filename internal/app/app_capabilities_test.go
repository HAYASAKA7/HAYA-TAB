package app

import (
	"testing"

	"haya-tab/internal/platform"
)

func TestGetRuntimeCapabilitiesUsesCompileTimeTarget(t *testing.T) {
	capabilities := NewApp().GetRuntimeCapabilities(390)
	if capabilities.Target != platform.TargetDesktop {
		t.Fatalf("target = %q, want %q", capabilities.Target, platform.TargetDesktop)
	}
	if capabilities.FormFactor != platform.FormFactorDesktop {
		t.Fatalf("form factor = %q, want %q", capabilities.FormFactor, platform.FormFactorDesktop)
	}
}
