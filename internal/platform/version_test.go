package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	wailsModuleVersion  = "v3.0.0-alpha2.118"
	wailsRuntimeVersion = "3.0.0-alpha2.117"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func TestPinnedWailsVersions(t *testing.T) {
	root := repositoryRoot(t)

	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "github.com/wailsapp/wails/v3 "+wailsModuleVersion) {
		t.Fatalf("go.mod must pin Wails %s", wailsModuleVersion)
	}

	rawPackage, err := os.ReadFile(filepath.Join(root, "frontend", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var packageJSON struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(rawPackage, &packageJSON); err != nil {
		t.Fatal(err)
	}
	if got := packageJSON.Dependencies["@wailsio/runtime"]; got != wailsRuntimeVersion {
		t.Fatalf("@wailsio/runtime = %q, want exact %q", got, wailsRuntimeVersion)
	}
}

func TestPinnedWailsToolingDocumentation(t *testing.T) {
	root := repositoryRoot(t)
	mustContain := func(path string, values ...string) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		content := string(raw)
		for _, value := range values {
			if !strings.Contains(content, value) {
				t.Errorf("%s must contain %q", path, value)
			}
		}
		if strings.Contains(content, "wails/v3/cmd/wails3@latest") {
			t.Errorf("%s must not install Wails with @latest", path)
		}
	}

	mustContain(
		".github/workflows/release.yml",
		"github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.118",
		"libgtk-4-dev",
		"libwebkitgtk-6.0-dev",
	)
	mustContain(
		"docs/DEVELOPMENT.md",
		"github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.118",
		"Node.js 24 or 26",
	)
}
