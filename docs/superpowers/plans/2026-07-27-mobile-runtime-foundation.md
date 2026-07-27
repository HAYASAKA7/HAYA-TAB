# Mobile Runtime Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish a tested Wails v3 mobile runtime foundation for HAYA-TAB that preserves desktop behavior, runs without a loopback HTTP server on iOS/Android, exposes explicit platform capabilities, and gives iOS/iPadOS a native top-level tab bar.

**Architecture:** Keep the existing Go application and Vue frontend, but move platform decisions behind two narrow boundaries: a Go `platform` capability model and a content-transport adapter. Desktop keeps its loopback file server; mobile routes `/api/*` through Wails' in-process asset handler. The Vue shell consumes capabilities instead of detecting user agents, allowing iOS native tabs, Android web navigation, and desktop navigation to coexist without feature-conditionals scattered through components.

**Tech Stack:** Go 1.25.4, Wails v3.0.0-alpha2.118 CLI/Go module, `@wailsio/runtime` 3.0.0-alpha2.117, Vue 3, TypeScript, Pinia, Vite, Playwright, Go `testing`/`httptest`.

---

## Scope and sequencing

This is plan 1 of the approved mobile-platform design. It deliberately establishes the runtime contracts that the cloud cache, offline outbox, adaptive reader interactions, and release-validation plans will consume. Those later plans must use the capability and content URL APIs defined here instead of introducing parallel platform checks or transport paths.

The completed foundation must provide:

- reproducible, exact Wails versions in source and CI;
- desktop, iOS, and Android capability profiles;
- an in-process `/api/file`, `/api/cover`, and `/api/cloud-stream` path on mobile;
- unchanged loopback streaming on desktop;
- native iOS/iPadOS top-level tabs using SF Symbols;
- a Vue navigation fallback for Android;
- capability-gated desktop-only services;
- simulator/build smoke checks and an explicit iOS 27 scene-lifecycle gate.

Do not add the cloud cache database, sync outbox, conflict UI, or secure credential migration in this plan. Their schemas and failure semantics belong to the cloud/offline plan, after this foundation has fixed the transport and platform contracts.

## Task 1: Record a clean baseline and pin the Wails toolchain

**Files:**

- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `.github/workflows/release.yml`
- Modify: `README.md`
- Modify: `docs/DEVELOPMENT.md`

- [ ] **Step 1: Confirm the feature worktree and record the baseline**

Run:

```powershell
git status --short
git branch --show-current
go test ./...
npm --prefix frontend run build
```

Expected:

- the branch is `feat/mobile-runtime`;
- the worktree contains only this plan file before implementation;
- all Go tests pass;
- the frontend production build exits with code 0.

If the baseline fails, record the exact pre-existing failure in the plan execution notes before changing dependencies. Do not weaken a test to make an upgrade appear green.

- [ ] **Step 2: Add a version consistency test that fails on the old versions**

Create `internal/platform/doc.go`:

```go
// Package platform owns the compile-time runtime target and capability contract.
package platform
```

Create `internal/platform/version_test.go`:

```go
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
```

Run:

```powershell
go test ./internal/platform -run TestPinnedWailsVersions -v
```

Expected: FAIL because the repository still uses Wails `v3.0.0-alpha.74` and a ranged frontend runtime.

- [ ] **Step 3: Pin the Go module and frontend runtime**

Run:

```powershell
go get github.com/wailsapp/wails/v3@v3.0.0-alpha2.118
go mod tidy
npm --prefix frontend install --save-exact @wailsio/runtime@3.0.0-alpha2.117
```

Expected:

- `go.mod` contains `github.com/wailsapp/wails/v3 v3.0.0-alpha2.118`;
- `frontend/package.json` contains `"@wailsio/runtime": "3.0.0-alpha2.117"` with no `^` or `~`;
- both lock files contain the same exact dependency versions.

- [ ] **Step 4: Pin CI to the same CLI version**

Replace every Wails install using `@latest` with:

```yaml
- name: Install Wails CLI
  run: go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.118
```

On Linux, keep the GTK4/WebKitGTK 6 development-package installation before `go install`, because the Wails CLI compiles platform-linked packages during installation. Do not reintroduce `@latest`.

For Ubuntu 24.04 the dependency step is:

```yaml
- name: Install Linux dependencies
  if: matrix.platform == 'linux'
  run: |
    sudo apt-get update
    sudo apt-get install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
```

Add a verification step:

```yaml
- name: Verify pinned Wails CLI
  shell: bash
  run: |
    set -euo pipefail
    wails3 version
```

- [ ] **Step 5: Regenerate bindings and reconcile compiler errors**

Install the exact CLI locally:

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.118
wails3 task common:generate:bindings
go test ./...
npm --prefix frontend run build
```

Expected: bindings regenerate successfully, Go tests pass, and the TypeScript build passes. Commit generated binding changes; they are part of the versioned API contract.

- [ ] **Step 6: Document the exact development command**

In `docs/DEVELOPMENT.md`, document:

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.118
$env:Path += ";$(go env GOPATH)\bin"
wails3 task dev
```

State that Node 24 or 26 is supported by this repository and that Node 22 is not required. Update the README link to point to this guide.

- [ ] **Step 7: Run the version test and commit**

Run:

```powershell
go test ./internal/platform -run TestPinnedWailsVersions -v
git diff --check
git add go.mod go.sum frontend/package.json frontend/package-lock.json .github/workflows/release.yml README.md docs/DEVELOPMENT.md internal/platform/doc.go internal/platform/version_test.go frontend/bindings
git commit -m "build: pin mobile Wails toolchain"
```

Expected: PASS and one focused commit.

## Task 2: Introduce a single platform capability contract

**Files:**

- Create: `internal/platform/capabilities.go`
- Create: `internal/platform/capabilities_test.go`
- Modify: `internal/app/app.go`
- Create: `internal/app/app_capabilities.go`
- Modify: `frontend/src/vite-env.d.ts`
- Create: `frontend/src/types/platform.ts`

- [ ] **Step 1: Write table-driven capability tests**

Create `internal/platform/capabilities_test.go`:

```go
package platform

import "testing"

func TestCapabilitiesFor(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		width  int
		want   Capabilities
	}{
		{
			name:   "desktop",
			target: TargetDesktop,
			width:  1440,
			want: Capabilities{
				Target: TargetDesktop, FormFactor: FormFactorDesktop,
				LoopbackContent: true, FolderWatcher: true,
				CustomStoragePaths: true, Plugins: true, WebMIDI: true,
				SelfUpdate: true,
			},
		},
		{
			name:   "iPhone",
			target: TargetIOS,
			width:  390,
			want: Capabilities{
				Target: TargetIOS, FormFactor: FormFactorPhone,
				NativeTopLevelTabs: true, InProcessContent: true,
				NativeFileImport: true, SafeAreaInsets: true,
			},
		},
		{
			name:   "iPad",
			target: TargetIOS,
			width:  1024,
			want: Capabilities{
				Target: TargetIOS, FormFactor: FormFactorTablet,
				NativeTopLevelTabs: true, InProcessContent: true,
				NativeFileImport: true, SafeAreaInsets: true,
			},
		},
		{
			name:   "Android phone",
			target: TargetAndroid,
			width:  412,
			want: Capabilities{
				Target: TargetAndroid, FormFactor: FormFactorPhone,
				WebTopLevelTabs: true, InProcessContent: true,
				NativeFileImport: true, SafeAreaInsets: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CapabilitiesFor(tt.target, tt.width); got != tt.want {
				t.Fatalf("CapabilitiesFor(%q, %d) = %#v, want %#v", tt.target, tt.width, got, tt.want)
			}
		})
	}
}

func TestCapabilitiesForRejectsUnknownTarget(t *testing.T) {
	if got := CapabilitiesFor("webos", 390); got.Target != TargetDesktop {
		t.Fatalf("unknown targets must fail closed to desktop, got %#v", got)
	}
}
```

Run:

```powershell
go test ./internal/platform -run TestCapabilitiesFor -v
```

Expected: FAIL because the types do not exist.

- [ ] **Step 2: Implement the pure capability model**

Create `internal/platform/capabilities.go` with:

```go
package platform

type Target string
type FormFactor string

const (
	TargetDesktop Target = "desktop"
	TargetIOS     Target = "ios"
	TargetAndroid Target = "android"

	FormFactorPhone   FormFactor = "phone"
	FormFactorTablet  FormFactor = "tablet"
	FormFactorDesktop FormFactor = "desktop"
)

type Capabilities struct {
	Target             Target     `json:"target"`
	FormFactor         FormFactor `json:"formFactor"`
	NativeTopLevelTabs bool       `json:"nativeTopLevelTabs"`
	WebTopLevelTabs    bool       `json:"webTopLevelTabs"`
	InProcessContent   bool       `json:"inProcessContent"`
	LoopbackContent    bool       `json:"loopbackContent"`
	NativeFileImport   bool       `json:"nativeFileImport"`
	SafeAreaInsets     bool       `json:"safeAreaInsets"`
	FolderWatcher      bool       `json:"folderWatcher"`
	CustomStoragePaths bool       `json:"customStoragePaths"`
	Plugins            bool       `json:"plugins"`
	WebMIDI            bool       `json:"webMIDI"`
	SelfUpdate         bool       `json:"selfUpdate"`
}

func CapabilitiesFor(target Target, viewportWidth int) Capabilities {
	if target != TargetIOS && target != TargetAndroid {
		return Capabilities{
			Target: TargetDesktop, FormFactor: FormFactorDesktop,
			LoopbackContent: true, FolderWatcher: true,
			CustomStoragePaths: true, Plugins: true, WebMIDI: true,
			SelfUpdate: true,
		}
	}

	formFactor := FormFactorPhone
	if viewportWidth >= 768 {
		formFactor = FormFactorTablet
	}
	result := Capabilities{
		Target: target, FormFactor: formFactor,
		InProcessContent: true, NativeFileImport: true, SafeAreaInsets: true,
	}
	result.NativeTopLevelTabs = target == TargetIOS
	result.WebTopLevelTabs = target == TargetAndroid
	return result
}
```

Run the test again. Expected: PASS.

- [ ] **Step 3: Bind compile-time target selection to the application**

Create three small build-tag files:

- `internal/platform/target_default.go` with `//go:build !ios && !android`
- `internal/platform/target_ios.go` with `//go:build ios`
- `internal/platform/target_android.go` with `//go:build android`

Each defines:

```go
func CurrentTarget() Target
```

Return only its corresponding constant. Do not inspect `navigator.userAgent`.

Create `internal/app/app_capabilities.go`:

```go
package app

import "haya-tab/internal/platform"

func (a *App) GetRuntimeCapabilities(viewportWidth int) platform.Capabilities {
	return platform.CapabilitiesFor(platform.CurrentTarget(), viewportWidth)
}
```

In `internal/app/app.go`, compute the startup profile once and guard the desktop folder-sync loop and watcher initialization with it:

```go
runtimeCapabilities := platform.CapabilitiesFor(platform.CurrentTarget(), 0)
if runtimeCapabilities.FolderWatcher {
	go a.runAutoSync()
	a.initFileWatcher()
}
```

Remove the existing unconditional `go a.runAutoSync()` call. This guard is required even though the controls are hidden in Vue: mobile startup itself must never run desktop folder synchronization or initialize `fsnotify` paths. WebDAV initialization remains universal.

Add `GetRuntimeCapabilities(viewportWidth: number): Promise<RuntimeCapabilities>` to `frontend/src/vite-env.d.ts`.

- [ ] **Step 4: Define the matching TypeScript type**

Create `frontend/src/types/platform.ts`:

```ts
export type RuntimeTarget = 'desktop' | 'ios' | 'android'
export type FormFactor = 'phone' | 'tablet' | 'desktop'

export interface RuntimeCapabilities {
  target: RuntimeTarget
  formFactor: FormFactor
  nativeTopLevelTabs: boolean
  webTopLevelTabs: boolean
  inProcessContent: boolean
  loopbackContent: boolean
  nativeFileImport: boolean
  safeAreaInsets: boolean
  folderWatcher: boolean
  customStoragePaths: boolean
  plugins: boolean
  webMIDI: boolean
  selfUpdate: boolean
}
```

The Go JSON tags and TypeScript field names must match exactly.

- [ ] **Step 5: Verify target-specific compilation**

Run:

```powershell
go test ./internal/platform ./internal/app
$env:GOOS='ios'; $env:GOARCH='arm64'; go list -f "{{.GoFiles}}" ./internal/platform
$env:GOOS='android'; $env:GOARCH='arm64'; go list -f "{{.GoFiles}}" ./internal/platform
npm --prefix frontend run build
```

Expected: the host tests pass; the iOS list contains `target_ios.go`; and the Android list contains `target_android.go`. Do not pass `ios` or `android` through `-tags`: they are reserved GOOS tags and would select conflicting standard-library files on a Windows host. Device linking remains in Task 9.

- [ ] **Step 6: Commit**

```powershell
git add internal/platform internal/app/app.go internal/app/app_capabilities.go frontend/src/types/platform.ts frontend/src/vite-env.d.ts
git commit -m "feat: add mobile runtime capabilities"
```

## Task 3: Load capabilities before platform-specific frontend services

**Files:**

- Create: `frontend/src/stores/platform.ts`
- Modify: `frontend/src/stores/index.ts`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/components/SettingsView.vue`
- Modify: `frontend/src/components/PluginsView.vue`
- Modify: `frontend/src/services/MidiService.ts`
- Create: `frontend/tests/e2e/platform-capabilities.spec.ts`

- [ ] **Step 1: Add an E2E backend mock helper**

In `frontend/tests/e2e/platform-capabilities.spec.ts`, add an init script before loading `/`:

```ts
import { expect, test, type Page } from '@playwright/test'

const mobileCapabilities = {
  target: 'ios',
  formFactor: 'phone',
  nativeTopLevelTabs: true,
  webTopLevelTabs: false,
  inProcessContent: true,
  loopbackContent: false,
  nativeFileImport: true,
  safeAreaInsets: true,
  folderWatcher: false,
  customStoragePaths: false,
  plugins: false,
  webMIDI: false,
  selfUpdate: false,
}

async function mockCapabilities(page: Page, capabilities = mobileCapabilities) {
  await page.addInitScript((value) => {
    ;(window as Window & { __HAYA_TEST_CAPABILITIES__?: unknown })
      .__HAYA_TEST_CAPABILITIES__ = value
  }, capabilities)
}
```

The store may read this override only when `import.meta.env.DEV` is true. Production builds must always ask the Go binding.

- [ ] **Step 2: Write failing behavior tests**

Add:

```ts
test('iOS hides unsupported desktop controls', async ({ page }) => {
  await mockCapabilities(page)
  await page.goto('/')
  await expect(page.locator('html')).toHaveAttribute('data-runtime-target', 'ios')
  await expect(page.getByTestId('plugins-navigation')).toHaveCount(0)
  await expect(page.getByTestId('custom-storage-settings')).toHaveCount(0)
  await expect(page.getByTestId('self-update-settings')).toHaveCount(0)
})

test('desktop keeps existing controls', async ({ page }) => {
  await mockCapabilities(page, {
    ...mobileCapabilities,
    target: 'desktop',
    formFactor: 'desktop',
    nativeTopLevelTabs: false,
    inProcessContent: false,
    loopbackContent: true,
    folderWatcher: true,
    customStoragePaths: true,
    plugins: true,
    webMIDI: true,
    selfUpdate: true,
  })
  await page.goto('/')
  await expect(page.locator('html')).toHaveAttribute('data-runtime-target', 'desktop')
  await expect(page.getByTestId('plugins-navigation')).toBeVisible()
  await expect(page.getByTestId('self-update-settings')).toBeVisible()
})
```

Run:

```powershell
npm --prefix frontend run test:e2e -- platform-capabilities.spec.ts
```

Expected: FAIL because the store, attributes, and test IDs do not exist.

- [ ] **Step 3: Implement the Pinia platform store**

Create `frontend/src/stores/platform.ts` with:

```ts
import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { callBackend } from '../services/api'
import type { RuntimeCapabilities } from '../types/platform'

const desktopFallback: RuntimeCapabilities = {
  target: 'desktop',
  formFactor: 'desktop',
  nativeTopLevelTabs: false,
  webTopLevelTabs: false,
  inProcessContent: false,
  loopbackContent: true,
  nativeFileImport: false,
  safeAreaInsets: false,
  folderWatcher: true,
  customStoragePaths: true,
  plugins: true,
  webMIDI: true,
  selfUpdate: true,
}

export const usePlatformStore = defineStore('platform', () => {
  const capabilities = ref<RuntimeCapabilities>(desktopFallback)
  const ready = ref(false)

  async function load(viewportWidth = window.innerWidth) {
    const testValue = import.meta.env.DEV
      ? (window as Window & {
          __HAYA_TEST_CAPABILITIES__?: RuntimeCapabilities
        }).__HAYA_TEST_CAPABILITIES__
      : undefined

    capabilities.value = testValue
      ?? await callBackend<RuntimeCapabilities>('GetRuntimeCapabilities', viewportWidth)
    document.documentElement.dataset.runtimeTarget = capabilities.value.target
    document.documentElement.dataset.formFactor = capabilities.value.formFactor
    ready.value = true
  }

  return {
    capabilities,
    ready,
    isMobile: computed(() => capabilities.value.target !== 'desktop'),
    load,
  }
})
```

Export it from `frontend/src/stores/index.ts`.

- [ ] **Step 4: Make application startup capability-first**

First remove the eager Web MIDI side effect. In `MidiService.ts`, do not call `init()` from the constructor. Rename it to an idempotent public method:

```ts
private initialized = false

public async initialize() {
  if (this.initialized) return
  this.initialized = true
  if (!navigator.requestMIDIAccess) return

  try {
    this.access = await navigator.requestMIDIAccess()
    this.access.onstatechange = () => this.updateInputs()
    this.updateInputs()
  } catch (error) {
    this.initialized = false
    console.error('Failed to access MIDI devices:', error)
  }
}
```

Keeping `export const midiService = MidiService.getInstance()` is safe after the constructor becomes side-effect free.

In `App.vue`, replace the side-effect-only MIDI import with `import { midiService } from '@/services/MidiService'`. Load the platform store before initializing settings, updates, plugins, WebDAV monitoring, MIDI, or watcher-dependent UI:

```ts
onMounted(async () => {
  await platformStore.load()

  if (platformStore.capabilities.plugins) {
    await pluginService.initialize()
  }
  if (platformStore.capabilities.webMIDI) {
    await midiService.initialize()
  }

  if (platformStore.capabilities.selfUpdate) {
    const updateInfo = await UpdateService.checkForUpdates()
    if (updateInfo) uiStore.updateInfo = updateInfo
  }

  // Existing universal initialization follows.
})
```

Keep the existing `PluginService.hasPlugins()` call inside the plugins capability branch. Render `<PluginsView>` with `v-if="platformStore.capabilities.plugins"`; the current hidden view is still mounted and would otherwise call `PluginService.getPlugins()` on mobile. The required behavior is:

- no `PluginService` call when `plugins` is false;
- no `MidiService` call when `webMIDI` is false;
- no custom storage/folder controls when `customStoragePaths` is false;
- no plugin route/control when `plugins` is false.
- no GitHub release update check or update settings when `selfUpdate` is false; iOS and Android distribution updates belong to their stores.

Add stable `data-testid` attributes only at the relevant navigation and settings containers.

- [ ] **Step 5: Run focused and regression tests**

```powershell
npm --prefix frontend run test:e2e -- platform-capabilities.spec.ts
npm --prefix frontend run test:e2e
npm --prefix frontend run build
```

Expected: all pass and the existing desktop modal-consistency suite remains green.

- [ ] **Step 6: Commit**

```powershell
git add frontend/src/stores frontend/src/App.vue frontend/src/components/SettingsView.vue frontend/src/components/PluginsView.vue frontend/src/services/MidiService.ts frontend/tests/e2e/platform-capabilities.spec.ts
git commit -m "feat: gate frontend features by runtime capability"
```

## Task 4: Route mobile content through the Wails asset handler

**Files:**

- Modify: `internal/app/server.go`
- Create: `internal/app/server_router_test.go`
- Modify: `cmd/haya-tab/main.go`

- [ ] **Step 1: Write router precedence tests**

Create `internal/app/server_router_test.go`:

```go
package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssetHandlerRoutesAPIRequestsToFileHandler(t *testing.T) {
	apiCalled := false
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("frontend fallback must not receive /api requests")
	})

	request := httptest.NewRequest(http.MethodGet, "/api/file/tab-1", nil)
	response := httptest.NewRecorder()
	NewAssetHandler(api, fallback).ServeHTTP(response, request)

	if !apiCalled || response.Code != http.StatusNoContent {
		t.Fatalf("apiCalled=%v status=%d", apiCalled, response.Code)
	}
}

func TestAssetHandlerDelegatesFrontendAssets(t *testing.T) {
	fallbackCalled := false
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusAccepted)
	})

	request := httptest.NewRequest(http.MethodGet, "/assets/index.js", nil)
	response := httptest.NewRecorder()
	NewAssetHandler(http.NotFoundHandler(), fallback).ServeHTTP(response, request)

	if !fallbackCalled || response.Code != http.StatusAccepted {
		t.Fatalf("fallbackCalled=%v status=%d", fallbackCalled, response.Code)
	}
}

func TestAssetHandlerDoesNotTreatAPIPrefixLookalikeAsAPI(t *testing.T) {
	fallbackCalled := false
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/apiary", nil)
	response := httptest.NewRecorder()
	NewAssetHandler(http.NotFoundHandler(), fallback).ServeHTTP(response, request)

	if !fallbackCalled {
		t.Fatal("/apiary must go to the frontend fallback")
	}
}
```

Run:

```powershell
go test ./internal/app -run TestAssetHandler -v
```

Expected: FAIL because `NewAssetHandler` does not exist.

- [ ] **Step 2: Implement the exact `/api/` router boundary**

Add to `internal/app/server.go`:

```go
func NewAssetHandler(api http.Handler, frontend http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		frontend.ServeHTTP(w, r)
	})
}
```

Keep `FileHandler` as the sole implementation of file, cover, and cloud-stream authorization and response headers. The router must not duplicate those cases.

- [ ] **Step 3: Compose the in-process asset handler**

In `cmd/haya-tab/main.go`, construct:

```go
frontendAssets := application.AssetFileServerFS(appassets.FS)
assetHandler := app.NewAssetHandler(app.NewFileHandler(myApp), frontendAssets)
```

and set:

```go
Assets: application.AssetOptions{
	Handler: assetHandler,
},
```

This composition is safe on every platform. Desktop still reaches the same `FileHandler` via its loopback adapter; mobile reaches it through Wails' `wails://` asset transport.

- [ ] **Step 4: Verify handler behavior and commit**

```powershell
go test ./internal/app -run "TestAssetHandler|TestFileHandler" -v
go test ./...
git add internal/app/server.go internal/app/server_router_test.go cmd/haya-tab/main.go
git commit -m "feat: serve mobile content in process"
```

Expected: all tests pass.

## Task 5: Return platform-correct content URLs

**Files:**

- Create: `internal/app/content_url.go`
- Create: `internal/app/content_url_test.go`
- Modify: `frontend/src/services/FileService.ts`
- Modify: `frontend/src/components/grid/TabCard.vue`
- Modify: `frontend/src/components/viewers/PdfViewer.vue`
- Modify: `frontend/src/components/viewers/GpViewer.vue`
- Modify: `frontend/tests/e2e/platform-capabilities.spec.ts`

- [ ] **Step 1: Write URL construction tests**

Create `internal/app/content_url_test.go`:

```go
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
	if _, err := contentURL(platform.TargetDesktop, 0, "file", "tab-1"); err == nil {
		t.Fatal("desktop URL without a port must fail")
	}
	if _, err := contentURL(platform.TargetIOS, 0, "unknown", "tab-1"); err == nil {
		t.Fatal("unknown content kind must fail")
	}
}
```

Run:

```powershell
go test ./internal/app -run TestContentURL -v
```

Expected: FAIL because `contentURL` does not exist.

- [ ] **Step 2: Implement content URL methods**

Create `internal/app/content_url.go`:

```go
package app

import (
	"fmt"
	"net/url"

	"haya-tab/internal/platform"
)

func contentURL(target platform.Target, port int, kind, id string) (string, error) {
	switch kind {
	case "file", "cover", "cloud-stream":
	default:
		return "", fmt.Errorf("unsupported content kind %q", kind)
	}
	path := fmt.Sprintf("/api/%s/%s", kind, url.PathEscape(id))
	if target == platform.TargetIOS || target == platform.TargetAndroid {
		return path, nil
	}
	if port <= 0 {
		return "", fmt.Errorf("desktop file server is not running")
	}
	return fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil
}

func (a *App) GetTabContentURL(id string) (string, error) {
	return contentURL(platform.CurrentTarget(), a.GetFileServerPort(), "file", id)
}

func (a *App) GetCoverContentURL(id string) (string, error) {
	return contentURL(platform.CurrentTarget(), a.GetFileServerPort(), "cover", id)
}
```

Do not accept a raw filesystem path from the frontend.

- [ ] **Step 3: Centralize frontend URL access**

Add to `FileService.ts`:

```ts
async getTabContentURL(tabId: string): Promise<string> {
  return callBackend<string>('GetTabContentURL', tabId)
},

async getCoverContentURL(tabId: string): Promise<string> {
  return callBackend<string>('GetCoverContentURL', tabId)
},
```

Replace every `http://127.0.0.1:${port}/api/...` construction in:

- `TabCard.vue`;
- `PdfViewer.vue`;
- `GpViewer.vue`.

Components must not read the file-server port. Preserve the existing cover cache-busting query by appending it with `URL`/`URLSearchParams`, not string-concatenating an unescaped identifier.

- [ ] **Step 4: Test that iOS rendering never constructs loopback URLs**

Extend the mobile E2E mock to record backend method calls and return:

```ts
'/api/file/tab-1'
'/api/cover/tab-1'
```

Assert:

```ts
expect(await page.locator('body').innerHTML()).not.toContain('127.0.0.1')
```

Also assert the relevant viewer request uses `/api/file/tab-1`. Mock the response so the test does not require a real PDF or Guitar Pro file.

- [ ] **Step 5: Verify and commit**

```powershell
go test ./internal/app -run TestContentURL -v
rg -n "127\\.0\\.0\\.1.*api/(file|cover|cloud-stream)" frontend/src
npm --prefix frontend run build
npm --prefix frontend run test:e2e -- platform-capabilities.spec.ts
git add internal/app/content_url.go internal/app/content_url_test.go frontend/src/services/FileService.ts frontend/src/components/grid/TabCard.vue frontend/src/components/viewers/PdfViewer.vue frontend/src/components/viewers/GpViewer.vue frontend/tests/e2e/platform-capabilities.spec.ts
git commit -m "refactor: centralize platform content URLs"
```

Expected:

- Go and frontend tests pass;
- `rg` returns no matches;
- one focused commit is created.

## Task 6: Stop opening a loopback listener on mobile

**Files:**

- Create: `cmd/haya-tab/content_transport_default.go`
- Create: `cmd/haya-tab/content_transport_mobile.go`
- Create: `cmd/haya-tab/content_transport_test.go`
- Modify: `cmd/haya-tab/main.go`

- [ ] **Step 1: Define and test a transport-starting seam**

Create `cmd/haya-tab/content_transport_test.go`:

```go
//go:build !ios && !android

package main

import "testing"

type fakeFileServer struct {
	started bool
	port    int
}

func (f *fakeFileServer) StartFileServer() (int, error) {
	f.started = true
	return f.port, nil
}

func (f *fakeFileServer) SetFileServerPort(port int) {
	f.port = port
}

func TestConfigureContentTransportStartsDesktopServer(t *testing.T) {
	server := &fakeFileServer{port: 42123}
	if err := configureContentTransport(server); err != nil {
		t.Fatal(err)
	}
	if !server.started || server.port != 42123 {
		t.Fatalf("started=%v port=%d", server.started, server.port)
	}
}
```

Run:

```powershell
go test ./cmd/haya-tab -run TestConfigureContentTransport -v
```

Expected: FAIL because the seam does not exist.

- [ ] **Step 2: Implement desktop and mobile adapters**

In `content_transport_default.go`:

```go
//go:build !ios && !android

package main

type fileServer interface {
	StartFileServer() (int, error)
	SetFileServerPort(int)
}

func configureContentTransport(server fileServer) error {
	port, err := server.StartFileServer()
	if err != nil {
		return err
	}
	server.SetFileServerPort(port)
	return nil
}
```

In `content_transport_mobile.go`:

```go
//go:build ios || android

package main

type fileServer interface {
	StartFileServer() (int, error)
	SetFileServerPort(int)
}

func configureContentTransport(_ fileServer) error {
	return nil
}
```

Keeping the interface identical avoids build-tag divergence in `main.go`.

- [ ] **Step 3: Replace unconditional startup**

In `main.go`, replace the direct `StartFileServer`/`SetFileServerPort` block with:

```go
if err := configureContentTransport(myApp); err != nil {
	log.Fatal("Error configuring content transport:", err)
}
```

- [ ] **Step 4: Verify mobile source selection**

Run:

```powershell
go test ./cmd/haya-tab -run TestConfigureContentTransport -v
go list -tags ios -f "{{.GoFiles}}" ./cmd/haya-tab
go list -tags android -f "{{.GoFiles}}" ./cmd/haya-tab
```

Expected:

- the desktop test passes;
- the iOS file list contains `content_transport_mobile.go`, not `content_transport_default.go`;
- the Android file list contains `content_transport_mobile.go`, not `content_transport_default.go`.

- [ ] **Step 5: Commit**

```powershell
git add cmd/haya-tab/main.go cmd/haya-tab/content_transport_default.go cmd/haya-tab/content_transport_mobile.go cmd/haya-tab/content_transport_test.go
git commit -m "fix: avoid loopback server on mobile"
```

## Task 7: Add native iOS tabs and an Android web fallback

**Files:**

- Modify: `cmd/haya-tab/app_options_ios.go`
- Create: `cmd/haya-tab/ios_options.go`
- Create: `cmd/haya-tab/ios_options_test.go`
- Create: `frontend/src/components/layout/MobileBottomNav.vue`
- Create: `frontend/src/composables/useNativeTabs.ts`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/stores/ui.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/views/LibraryView.vue`
- Modify: `frontend/src/i18n/locales/en.json`
- Modify: `frontend/src/i18n/locales/ja.json`
- Modify: `frontend/src/i18n/locales/zh-CN.json`
- Modify: `frontend/src/i18n/locales/zh-TW.json`
- Modify: `frontend/tests/e2e/platform-capabilities.spec.ts`

- [ ] **Step 1: Extract a host-testable native-tab definition**

Create `ios_options.go` without a build tag. Keeping the data and option mutation platform-independent allows the tests to run on Windows/Linux while the iOS build-tag file remains the only caller in production:

```go
func iosNativeTabs(locale string) []application.NativeTabItem {
	titles := map[string][4]string{
		"en":    {"Library", "Offline", "Search", "Settings"},
		"ja":    {"ライブラリ", "オフライン", "検索", "設定"},
		"zh-CN": {"曲库", "离线", "搜索", "设置"},
		"zh-TW": {"曲庫", "離線", "搜尋", "設置"},
	}
	labels, ok := titles[locale]
	if !ok {
		labels = titles["en"]
	}
	return []application.NativeTabItem{
		{Title: labels[0], SystemImage: application.NativeTabIcon("books.vertical")},
		{Title: labels[1], SystemImage: application.NativeTabIcon("arrow.down.circle")},
		{Title: labels[2], SystemImage: application.NativeTabIconMagnify},
		{Title: labels[3], SystemImage: application.NativeTabIconGear},
	}
}

func applyIOSOptions(opts *application.Options, locale string) {
	opts.DisableDefaultSignalHandler = true
	opts.IOS.EnableNativeTabs = true
	opts.IOS.NativeTabsItems = iosNativeTabs(locale)
	opts.IOS.EnableBackForwardNavigationGestures = true
}
```

Use `pkg/store.DetectSystemLocale()` when assigning these items so the first native frame uses the device locale. Do not open the settings database during options construction, and do not substitute custom bitmap icons on iOS.

- [ ] **Step 2: Write the iOS options test**

Create `ios_options_test.go` without a build tag:

```go
package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestApplyIOSOptionsConfiguresNativeTabs(t *testing.T) {
	opts := application.Options{}
	applyIOSOptions(&opts, "en")

	if !opts.DisableDefaultSignalHandler {
		t.Fatal("iOS must disable default signal handling")
	}
	if !opts.IOS.EnableNativeTabs {
		t.Fatal("iOS native tabs must be enabled")
	}
	if got := len(opts.IOS.NativeTabsItems); got != 4 {
		t.Fatalf("native tabs = %d, want 4", got)
	}
}

func TestIOSNativeTabsUsesSupportedLocale(t *testing.T) {
	if got := iosNativeTabs("ja")[0].Title; got != "ライブラリ" {
		t.Fatalf("Japanese library title = %q", got)
	}
	if got := iosNativeTabs("unsupported")[0].Title; got != "Library" {
		t.Fatalf("fallback library title = %q", got)
	}
}
```

Run on every development host:

```powershell
go test ./cmd/haya-tab -run "TestApplyIOSOptions|TestIOSNativeTabs" -v
```

Expected before implementation: FAIL.

- [ ] **Step 3: Configure Wails' native iOS tab bar**

Complete `modifyOptionsForIOS` in the existing iOS build-tag file:

```go
applyIOSOptions(opts, store.DetectSystemLocale())
```

Do not add a custom glass material, background blur, or private UIKit API. Wails' standard `UITabBarAppearance` receives the current iOS system appearance, including Liquid Glass when the app is built with the corresponding Apple SDK.

- [ ] **Step 4: Write Android bottom-navigation E2E behavior**

Add a test with `target: 'android'` and `webTopLevelTabs: true`:

```ts
await expect(page.getByRole('navigation', { name: 'Primary' })).toBeVisible()
await expect(page.getByRole('button', { name: 'Library' })).toBeVisible()
await expect(page.getByRole('button', { name: 'Offline' })).toBeVisible()
await expect(page.getByRole('button', { name: 'Search' })).toBeVisible()
await expect(page.getByRole('button', { name: 'Settings' })).toBeVisible()
```

Add an iOS assertion that this web navigation is absent because iOS owns top-level navigation natively.

- [ ] **Step 5: Implement the Android navigation component**

Create `MobileBottomNav.vue` with:

- semantic `<nav aria-label="Primary">`;
- four 44-by-44-point minimum controls;
- Library, Offline, Search, and Settings destinations;
- `padding-bottom: env(safe-area-inset-bottom)`;
- `position: fixed` only for `webTopLevelTabs`;
- no rendering when `nativeTopLevelTabs` is true.

Use the current icon system for Android. Add all four labels to each locale file and keep the visible text localized.

- [ ] **Step 6: Bridge native tab selection to the Vue UI store**

Extend `frontend/src/types/index.ts` with:

```ts
export type TopLevelDestination = 'library' | 'offline' | 'search' | 'settings'
```

In `frontend/src/stores/ui.ts`, add `topLevelDestination`, `libraryMode`, and `searchRequestKey`, then implement:

```ts
function selectTopLevelDestination(destination: TopLevelDestination) {
  topLevelDestination.value = destination
  if (destination === 'settings') {
    switchView('settings')
    return
  }

  libraryMode.value = destination === 'offline' ? 'offline' : 'all'
  switchView('library')
  if (destination === 'search') searchRequestKey.value++
}
```

`LibraryView.vue` must watch `searchRequestKey` and focus its existing `SearchBar`. Until the cache schema lands, `libraryMode === 'offline'` shows locally managed/non-cloud tabs; the cloud/offline plan replaces that predicate with the cache availability contract. This keeps the destination useful without inventing a second cache model.

Create `useNativeTabs.ts`:

```ts
import { onBeforeUnmount, onMounted } from 'vue'
import { usePlatformStore } from '../stores/platform'
import { useUIStore } from '../stores/ui'

const destinations = ['library', 'offline', 'search', 'settings'] as const

export function useNativeTabs() {
  const platform = usePlatformStore()
  const ui = useUIStore()
  const handleNativeTab = (event: Event) => {
    const index = Number((event as CustomEvent<{ index?: number }>).detail?.index)
    const destination = destinations[index]
    if (destination) ui.selectTopLevelDestination(destination)
  }

  onMounted(() => {
    if (!platform.capabilities.nativeTopLevelTabs) return
    window.addEventListener('nativeTabSelected', handleNativeTab)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('nativeTabSelected', handleNativeTab)
  })
}
```

Wails alpha2.118 dispatches this as a DOM `CustomEvent`, not a Wails runtime event. Preserve the behavior contract: one listener, bounds-checked index, cleanup on unmount, and navigation through the UI store rather than DOM manipulation.

- [ ] **Step 7: Verify and commit**

```powershell
npm --prefix frontend run test:e2e -- platform-capabilities.spec.ts
npm --prefix frontend run build
go test ./...
git add cmd/haya-tab/app_options_ios.go cmd/haya-tab/ios_options.go cmd/haya-tab/ios_options_test.go frontend/src/components/layout/MobileBottomNav.vue frontend/src/composables/useNativeTabs.ts frontend/src/App.vue frontend/src/stores/ui.ts frontend/src/types/index.ts frontend/src/views/LibraryView.vue frontend/src/i18n frontend/tests/e2e/platform-capabilities.spec.ts
git commit -m "feat: add native iOS mobile navigation"
```

## Task 8: Make the application shell safe-area and form-factor aware

**Files:**

- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/assets/style.css`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/views/LibraryView.vue`
- Modify: `frontend/tests/e2e/platform-capabilities.spec.ts`

- [ ] **Step 1: Add phone and tablet viewport tests**

Add Playwright tests using:

```ts
test.use({ viewport: { width: 390, height: 844 } })
```

for an iPhone-sized viewport and:

```ts
test.use({ viewport: { width: 1024, height: 1366 } })
```

for iPad portrait.

Assert:

- phone: desktop sidebar absent, one-column content, no horizontal overflow;
- tablet: library/sidebar split view present, content remains within viewport;
- both: the root shell has safe-area padding variables and 44-point minimum interactive controls;
- desktop: the existing 250px sidebar behavior is unchanged.

Use `document.documentElement.scrollWidth <= window.innerWidth` for the overflow assertion.

- [ ] **Step 2: Add platform-scoped layout tokens**

In `style.css`, define:

```css
:root {
  --safe-top: env(safe-area-inset-top, 0px);
  --safe-right: env(safe-area-inset-right, 0px);
  --safe-bottom: env(safe-area-inset-bottom, 0px);
  --safe-left: env(safe-area-inset-left, 0px);
  --minimum-touch-target: 44px;
}

html[data-runtime-target='ios'],
html[data-runtime-target='android'] {
  overscroll-behavior: none;
}
```

Apply safe-area padding at the shell boundary, not independently to every child. Components inside the shell should consume normal layout space.

- [ ] **Step 3: Implement the three shell modes**

In `App.vue` and `AppSidebar.vue`:

- desktop: preserve the current fixed sidebar;
- iOS phone: hide the Vue top-level nav because the native tab bar owns it;
- Android phone: reserve content space for `MobileBottomNav`;
- tablet: render a collapsible/split sidebar for library context while top-level destination remains controlled by native iOS tabs or Android web tabs.

Use capability state and `data-form-factor`; do not create CSS user-agent selectors.

- [ ] **Step 4: Run visual behavior tests**

```powershell
npm --prefix frontend run test:e2e -- platform-capabilities.spec.ts
npm --prefix frontend run test:e2e -- modal-consistency.spec.ts
npm --prefix frontend run build
```

Expected: all pass at phone, tablet, and desktop viewports.

- [ ] **Step 5: Commit**

```powershell
git add frontend/src/App.vue frontend/src/assets/style.css frontend/src/components/layout/AppSidebar.vue frontend/src/views/LibraryView.vue frontend/tests/e2e/platform-capabilities.spec.ts
git commit -m "feat: add adaptive mobile application shell"
```

## Task 9: Add mobile build smoke checks and the iOS 27 lifecycle gate

**Files:**

- Modify: `build/config.yml`
- Modify: `build/ios/Taskfile.yml`
- Modify: `build/android/Taskfile.yml`
- Modify: `build/android/app/build.gradle`
- Create: `scripts/check-wails-ios-lifecycle.ps1`
- Create: `scripts/check-wails-ios-lifecycle.sh`
- Create: `.github/workflows/mobile-smoke.yml`
- Create: `docs/MOBILE_DEVELOPMENT.md`
- Modify: `README.md`

- [ ] **Step 1: Make mobile identity explicit**

Uncomment and set the `ios` block in `build/config.yml`:

```yaml
ios:
  bundleID: "com.hayasaka7.hayatab"
  displayName: "HAYA-TAB"
  version: "3.1.7"
  company: "HAYASAKA7"
  comments: "Cloud-first music tab library and reader"
```

Keep the deployment minimum at the generated Wails-compatible value unless product support policy changes. “Validated on iOS/iPadOS 26” is a test target, not a reason to exclude older compatible devices at build time.

In `build/android/Taskfile.yml`, set:

```yaml
APP_ID: '{{.APP_ID | default "com.hayasaka7.hayatab"}}'
TARGET_SDK: '35'
```

In `build/android/app/build.gradle`, set `compileSdk 35`, `targetSdk 35`, and `applicationId "com.hayasaka7.hayatab"`. Keep the Java namespace `com.wails.app` in this phase so generated bridge sources remain aligned; change the deploy commands to launch the fully qualified component:

```text
com.hayasaka7.hayatab/com.wails.app.MainActivity
```

Add an unsigned Play Store bundle task:

```yaml
  assemble:aab:release:
    summary: Assembles an unsigned release Android App Bundle
    cmds:
      - |
        cd build/android
        ./gradlew bundleRelease
        cp app/build/outputs/bundle/release/app-release.aab "../../{{.BIN_DIR}}/{{.APP_NAME}}-release.aab"
```

- [ ] **Step 2: Write an iOS lifecycle inspection script**

The script must inspect the exact Wails module selected by `go list -m`, not a hard-coded module-cache path. It must require both the scene delegate API and the application-scene manifest; a comment or test mentioning only one term must not open the gate:

```powershell
$moduleJson = go list -m -json github.com/wailsapp/wails/v3 | ConvertFrom-Json
$nativeSource = Get-ChildItem -Path $moduleJson.Dir -Recurse -File -Include *.m,*.mm,*.h,Info.plist |
  Where-Object { $_.Name -notmatch '_test\.' }
$hasSceneDelegate = $nativeSource | Select-String -Quiet -Pattern "UIScene|SceneDelegate"
$hasSceneManifest = $nativeSource | Select-String -Quiet -Pattern "UIApplicationSceneManifest"

if (-not ($hasSceneDelegate -and $hasSceneManifest)) {
  Write-Error "Pinned Wails lacks UIScene lifecycle support. iOS 27 packaging remains blocked."
  exit 27
}
```

The shell version must perform the same module lookup and two-part source scan:

```bash
#!/usr/bin/env bash
set -euo pipefail

module_dir="$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)"
native_files=(--glob '*.m' --glob '*.mm' --glob '*.h' --glob 'Info.plist' --glob '!**/*_test.*')

if ! rg -q "UIScene|SceneDelegate" "${native_files[@]}" "$module_dir" ||
   ! rg -q "UIApplicationSceneManifest" "${native_files[@]}" "$module_dir"; then
  echo >&2 "Pinned Wails lacks UIScene lifecycle support. iOS 27 packaging remains blocked."
  exit 27
fi
```

Exit 27 is intentional and documented. The CI macOS runner must install `ripgrep` only if `rg` is absent.

- [ ] **Step 3: Add an explicit, non-default iOS 27 gate task**

In `build/ios/Taskfile.yml`, add:

```yaml
  verify:ios27:lifecycle:
    desc: Verify pinned Wails supports Apple's required scene lifecycle
    cmds:
      - bash scripts/check-wails-ios-lifecycle.sh
```

Do not add this task to the iOS 26 simulator build yet: alpha2.118 is expected to fail it. The task is a release gate that becomes mandatory before declaring iOS 27 support.

- [ ] **Step 4: Add unsigned mobile smoke CI**

Create `.github/workflows/mobile-smoke.yml` with:

- exact Go 1.25.4;
- exact Wails CLI `v3.0.0-alpha2.118`;
- exact Node 24 for the build job;
- `go test ./...` and `npm --prefix frontend run build` on all jobs;
- Android debug APK/AAB compilation without signing secrets;
- iOS simulator compilation on a macOS runner using the available iOS 26 SDK;
- artifact upload for build logs and unsigned smoke packages;
- no `@latest` dependency installation.

The iOS job must print `xcodebuild -version` and `xcrun --sdk iphonesimulator --show-sdk-version` before compilation. If GitHub no longer offers an iOS 26 SDK runner, fail with an explicit unsupported-SDK message instead of silently validating against iOS 27.

- [ ] **Step 5: Document local mobile development**

Create `docs/MOBILE_DEVELOPMENT.md` covering:

- macOS/Xcode is required to build or run iOS;
- exact Wails install command;
- `wails3 task ios:install:deps`;
- simulator and physical-device task commands already defined in `build/ios/Taskfile.yml`;
- Android SDK/NDK setup and debug task commands;
- WebDAV test account setup without committing credentials;
- iOS 26 is the initial validated Apple target;
- iOS 27 remains blocked until `verify:ios27:lifecycle` passes;
- standard UIKit navigation receives the OS appearance, while Vue/WebView content uses the app's CSS design system.

Link this guide from the README.

- [ ] **Step 6: Verify generated mobile configuration**

On Windows/Linux:

```powershell
go test ./...
npm --prefix frontend run build
wails3 task common:generate:bindings
git diff --check
```

On macOS:

```bash
wails3 task ios:install:deps
wails3 task ios:build
wails3 task verify:ios27:lifecycle
```

Expected:

- iOS 26 simulator build passes;
- the iOS 27 lifecycle task exits 27 on alpha2.118, confirming the gate is active;
- Android debug build passes on an Android-equipped runner;
- generated bindings are clean after regeneration.

- [ ] **Step 7: Commit**

```powershell
git add build/config.yml build/ios/Taskfile.yml build/android/Taskfile.yml build/android/app/build.gradle scripts/check-wails-ios-lifecycle.ps1 scripts/check-wails-ios-lifecycle.sh .github/workflows/mobile-smoke.yml docs/MOBILE_DEVELOPMENT.md README.md
git commit -m "ci: add mobile runtime smoke gates"
```

## Task 10: Full foundation verification and handoff

**Files:**

- Modify only if verification reveals a defect in files already owned by Tasks 1–9.

- [ ] **Step 1: Run the full Go suite**

```powershell
go test ./...
```

Expected: PASS with no skipped repository tests added to conceal platform failures.

- [ ] **Step 2: Run frontend build and E2E**

```powershell
npm --prefix frontend run build
npm --prefix frontend run test:e2e
```

Expected: PASS.

- [ ] **Step 3: Scan for transport and version regressions**

```powershell
rg -n "@latest|wails/v3 v3\\.0\\.0-alpha\\.74|@wailsio/runtime.*[\\^~]|127\\.0\\.0\\.1.*api/(file|cover|cloud-stream)" .github go.mod frontend/package.json frontend/src docs build
```

Expected: no matches, except explanatory documentation that is explicitly describing prohibited patterns. Rewrite such prose if it makes the automated scan ambiguous.

- [ ] **Step 4: Regenerate bindings and confirm a clean tree**

```powershell
wails3 task common:generate:bindings
git status --short
git diff --check
```

Expected: no uncommitted generated changes and no whitespace errors.

- [ ] **Step 5: Review the foundation contract**

Confirm manually:

- only `internal/platform` decides runtime target;
- only `contentURL` decides loopback versus in-process content paths;
- mobile bootstrap never calls `net.Listen`;
- iOS renders one native top-level tab bar and no duplicate Vue tab bar;
- Android renders one Vue bottom navigation;
- unsupported mobile services do not initialize;
- desktop behavior and desktop tests remain unchanged;
- iOS 27 support is not claimed while the lifecycle gate fails.

- [ ] **Step 6: Create the checkpoint commit if verification required fixes**

```powershell
git add -A
git commit -m "test: verify mobile runtime foundation"
```

Skip this commit if the tree is already clean.

## Acceptance criteria

- `go.mod`, frontend runtime, local instructions, and CI pin exact compatible Wails versions.
- All platform behavior is driven by the typed capability contract.
- iOS and Android use Wails' in-process asset transport for tab, cover, and cloud-stream requests.
- Desktop retains the loopback server without frontend components knowing its port.
- iOS uses Wails native tabs with SF Symbols and the system-provided UIKit appearance.
- Android has an accessible Vue bottom navigation; iOS does not render it.
- Phone, tablet, and desktop shell behavior is covered by Playwright viewport tests.
- The existing desktop Go, build, and modal-consistency tests remain green.
- Unsigned mobile smoke workflows are reproducible and use no signing secrets.
- The iOS 27 scene-lifecycle check remains an explicit failing gate until Wails provides compliant lifecycle support.
