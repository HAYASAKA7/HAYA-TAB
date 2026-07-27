package app

import (
	"testing"

	"haya-tab/internal/platform"
)

func TestContentURL(t *testing.T) {
	tests := []struct {
		name   string
		target platform.Target
		port   int
		kind   string
		id     string
		want   string
	}{
		{"desktop file", platform.TargetDesktop, 43210, "file", "tab 1", "http://127.0.0.1:43210/api/file/tab%201"},
		{"desktop cover", platform.TargetDesktop, 43210, "cover", "a/b", "http://127.0.0.1:43210/api/cover/a%2Fb"},
		{"iOS file", platform.TargetIOS, 0, "file", "tab 1", "/api/file/tab%201"},
		{"Android cover", platform.TargetAndroid, 0, "cover", "a/b", "/api/cover/a%2Fb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := contentURL(tt.target, tt.port, tt.kind, tt.id)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("contentURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContentURLRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		target platform.Target
		port   int
		kind   string
		id     string
	}{
		{"desktop without port", platform.TargetDesktop, 0, "file", "tab-1"},
		{"desktop with invalid port", platform.TargetDesktop, 65536, "file", "tab-1"},
		{"unknown kind", platform.TargetIOS, 0, "unknown", "tab-1"},
		{"empty identifier", platform.TargetIOS, 0, "file", ""},
		{"unknown target", platform.Target("unknown"), 0, "file", "tab-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := contentURL(tt.target, tt.port, tt.kind, tt.id); err == nil {
				t.Fatal("contentURL() must reject invalid input")
			}
		})
	}
}
