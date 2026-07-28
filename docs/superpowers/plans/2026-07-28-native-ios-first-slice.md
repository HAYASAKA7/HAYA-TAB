# Native iOS First Slice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a native SwiftUI iPhone/iPad application that connects to a HAYA-TAB WebDAV volume, persists its library, downloads a PDF or Guitar Pro file, opens it offline, and restores that state after restart without changing desktop behavior.

**Architecture:** The native target lives under `mobile/ios` and is generated deterministically with XcodeGen 2.45.4. SwiftUI owns navigation and presentation; Swift actors own WebDAV, persistence, Keychain, and downloads; a local `WKWebView` package owns document rendering. Go and Swift share JSON fixtures and contracts, not runtime code.

**Tech Stack:** Swift 6, SwiftUI, Observation, SwiftData, Security, URLSession, WebKit, XCTest/XCUITest, XcodeGen 2.45.4, Vite 5, AlphaTab 1.3, Go 1.25.

---

## Scope and file structure

This plan implements the first end-to-end slice from the approved design.
Search, multi-account and multi-volume discovery, annotation editing, upload,
Android, StoreKit, widgets, and advanced conflict merging remain outside this
slice. The account URL for this slice points to one existing HAYA-TAB volume
root.

Create these focused units:

```text
shared/
├── contracts/v1/fingerprint-bucket.schema.json
└── fixtures/fingerprint/v1/
    ├── bucket-00.valid.json
    ├── bucket-03.valid.json
    └── bucket.invalid.json

mobile/ios/
├── project.yml
├── Config/
│   ├── App.xcconfig
│   └── Test.xcconfig
├── HayaTab/
│   ├── App/HayaTabApp.swift
│   ├── App/AppEnvironment.swift
│   ├── Domain/LibraryItem.swift
│   ├── Domain/AppError.swift
│   ├── Data/CredentialStore.swift
│   ├── Data/WebDAVClient.swift
│   ├── Data/LibraryStore.swift
│   ├── Data/LibraryRepository.swift
│   ├── Data/DownloadStore.swift
│   ├── Features/Root/RootView.swift
│   ├── Features/Library/LibraryView.swift
│   ├── Features/Library/LibraryViewModel.swift
│   ├── Features/Downloads/DownloadsView.swift
│   ├── Features/Search/SearchView.swift
│   ├── Features/Settings/SettingsView.swift
│   ├── Features/Settings/AccountSheet.swift
│   ├── Features/Reader/DocumentReaderView.swift
│   ├── Features/Reader/ReaderWebView.swift
│   └── Resources/Viewer/**
├── HayaTabTests/
│   ├── ContractFixtureTests.swift
│   ├── WebDAVClientTests.swift
│   ├── LibraryRepositoryTests.swift
│   ├── DownloadStoreTests.swift
│   └── ReaderBridgeTests.swift
└── HayaTabUITests/FirstSliceUITests.swift

frontend/mobile-viewer/
├── index.html
├── main.ts
└── style.css

scripts/
├── generate-native-ios-project.sh
└── package-mobile-viewer.mjs
```

The generated `mobile/ios/HayaTab.xcodeproj` is ignored and recreated from
`project.yml`; source and configuration files are committed.

### Task 1: Freeze the desktop/mobile fingerprint contract

**Files:**
- Create: `shared/contracts/v1/fingerprint-bucket.schema.json`
- Create: `shared/fixtures/fingerprint/v1/bucket-00.valid.json`
- Create: `shared/fixtures/fingerprint/v1/bucket-03.valid.json`
- Create: `shared/fixtures/fingerprint/v1/bucket.invalid.json`
- Create: `pkg/sync/mobile_contract_test.go`

- [ ] **Step 1: Write the failing Go compatibility test**

```go
package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMobileFingerprintFixturesMatchDesktopContract(t *testing.T) {
	root := filepath.Join("..", "..", "shared", "fixtures", "fingerprint", "v1")
	data0, err := os.ReadFile(filepath.Join(root, "bucket-00.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bucket0 struct {
		Metadata FingerprintMetadata `json:"metadata"`
		Files    []FingerprintFile   `json:"files"`
	}
	if err := json.Unmarshal(data0, &bucket0); err != nil {
		t.Fatal(err)
	}
	if bucket0.Metadata.VolumeID == "" || bucket0.Metadata.BucketCount != BucketCount {
		t.Fatalf("invalid bucket-00 metadata: %#v", bucket0.Metadata)
	}

	data3, err := os.ReadFile(filepath.Join(root, "bucket-03.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bucket3 BucketData
	if err := json.Unmarshal(data3, &bucket3); err != nil {
		t.Fatal(err)
	}
	if bucket3.BucketNumber != 3 {
		t.Fatalf("bucket_number = %d, want 3", bucket3.BucketNumber)
	}
	for _, file := range append(bucket0.Files, bucket3.Files...) {
		if file.RelativePath == "" || file.Title == "" || file.Type == "" {
			t.Fatalf("incomplete file %#v", file)
		}
	}
}

func TestMobileInvalidFingerprintFixtureIsRejected(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "shared", "fixtures", "fingerprint", "v1", "bucket.invalid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bucket BucketData
	if err := json.Unmarshal(data, &bucket); err == nil {
		t.Fatal("invalid fixture unexpectedly decoded")
	}
}
```

- [ ] **Step 2: Run the test and verify it fails because fixtures are absent**

Run: `go test ./pkg/sync -run MobileFingerprint -v`

Expected: FAIL with `The system cannot find the path specified` on Windows or
`no such file or directory` on macOS.

- [ ] **Step 3: Add schema and fixtures**

The schema uses `oneOf`: bucket zero requires `metadata` and `files`, while
buckets 1–15 require `bucket_number` and `files`. Every file requires
`relative_path`, `title`, `artist`, `album`, `type`, `categories`,
`uploaded_at`, and `uploaded_by`. Use this nonzero-bucket shape:

```json
{
  "bucket_number": 3,
  "files": [{
    "relative_path": "scores/etude.gp5",
    "title": "Etude",
    "artist": "HAYA",
    "album": "",
    "type": "gp",
    "categories": ["Practice"],
    "uploaded_at": "2026-07-28T00:00:00Z",
    "uploaded_by": "fixture"
  }]
}
```

Use this bucket-zero envelope:

```json
{
  "metadata": {
    "volume_id": "volume-fixture",
    "volume_name": "Fixture",
    "created_at": "2026-07-28T00:00:00Z",
    "app_version": "3.1.7",
    "device_name": "fixture",
    "last_updated": "2026-07-28T00:00:00Z",
    "bucket_count": 16
  },
  "files": []
}
```

Make `bucket.invalid.json` syntactically invalid (`{"bucket_number":`), ensuring
both implementations prove decoder failure rather than relying on optional
field policy.

- [ ] **Step 4: Run the contract test**

Run: `go test ./pkg/sync -run MobileFingerprint -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add shared/contracts shared/fixtures pkg/sync/mobile_contract_test.go
git commit -m "test: freeze mobile fingerprint contract"
```

### Task 2: Add a deterministic native Xcode project

**Files:**
- Create: `mobile/ios/project.yml`
- Create: `mobile/ios/Config/App.xcconfig`
- Create: `mobile/ios/Config/Test.xcconfig`
- Create: `scripts/generate-native-ios-project.sh`
- Modify: `.gitignore`

- [ ] **Step 1: Add a project-generation smoke check**

Create `scripts/generate-native-ios-project.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

readonly version="2.45.4"
readonly tool_root="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/xcodegen-${version}"
if [[ ! -x "${tool_root}/.build/release/xcodegen" ]]; then
  git clone --branch "${version}" --depth 1 \
    https://github.com/yonaskolb/XcodeGen.git "${tool_root}"
  swift build --package-path "${tool_root}" -c release --product xcodegen
fi
"${tool_root}/.build/release/xcodegen" \
  generate --spec mobile/ios/project.yml
```

- [ ] **Step 2: Define the project**

`mobile/ios/project.yml` must define `HayaTab`, `HayaTabTests`, and
`HayaTabUITests`; use iOS 18.0, Swift 6, device families `1,2`, generated
Info.plists, and a shared `HayaTab` scheme:

```yaml
name: HayaTab
options:
  deploymentTarget:
    iOS: "18.0"
configs:
  Debug: debug
  Release: release
targets:
  HayaTab:
    type: application
    platform: iOS
    sources:
      - HayaTab
    configFiles:
      Debug: Config/App.xcconfig
      Release: Config/App.xcconfig
    settings:
      base:
        SWIFT_VERSION: "6.0"
        TARGETED_DEVICE_FAMILY: "1,2"
        GENERATE_INFOPLIST_FILE: YES
        INFOPLIST_KEY_CFBundleDisplayName: HAYA-TAB
        INFOPLIST_KEY_UIApplicationSceneManifest_Generation: YES
        CODE_SIGNING_ALLOWED: NO
  HayaTabTests:
    type: bundle.unit-test
    platform: iOS
    sources:
      - HayaTabTests
      - path: ../../shared/fixtures
        buildPhase: resources
    dependencies:
      - target: HayaTab
    configFiles:
      Debug: Config/Test.xcconfig
  HayaTabUITests:
    type: bundle.ui-testing
    platform: iOS
    sources:
      - HayaTabUITests
    dependencies:
      - target: HayaTab
schemes:
  HayaTab:
    build:
      targets:
        HayaTab: all
    test:
      targets:
        - HayaTabTests
        - HayaTabUITests
```

Set `PRODUCT_BUNDLE_IDENTIFIER = com.hayasaka7.hayatab` in `App.xcconfig` and
test-only identifiers in `Test.xcconfig`.

- [ ] **Step 3: Ignore generated Xcode state**

Add:

```gitignore
mobile/ios/HayaTab.xcodeproj/
mobile/ios/DerivedData/
mobile/ios/*.xcresult
```

- [ ] **Step 4: Generate and inspect on macOS CI**

Run: `bash scripts/generate-native-ios-project.sh`

Expected: `mobile/ios/HayaTab.xcodeproj/project.pbxproj` exists and XcodeGen
reports a successful generation. This step is macOS-only.

- [ ] **Step 5: Commit**

```bash
git add .gitignore mobile/ios/project.yml mobile/ios/Config scripts/generate-native-ios-project.sh
git commit -m "build: define native iOS project"
```

### Task 3: Establish native domain types and persistence

**Files:**
- Create: `mobile/ios/HayaTab/Domain/LibraryItem.swift`
- Create: `mobile/ios/HayaTab/Domain/AppError.swift`
- Create: `mobile/ios/HayaTab/Data/LibraryStore.swift`
- Create: `mobile/ios/HayaTabTests/ContractFixtureTests.swift`
- Create: `mobile/ios/HayaTabTests/LibraryStoreTests.swift`

- [ ] **Step 1: Write fixture decoding and persistence tests**

```swift
import XCTest
import SwiftData
@testable import HayaTab

extension LibraryItem {
    static func fixture() -> LibraryItem {
        LibraryItem(
            id: "volume-fixture:etude",
            relativePath: "scores/etude.gp5",
            title: "Etude",
            artist: "HAYA",
            album: "",
            kind: .gp,
            categories: ["Practice"],
            localFilename: nil,
            remoteRevision: nil)
    }
}

final class ContractFixtureTests: XCTestCase {
    func testDecodesDesktopFingerprintBucket() throws {
        let url = try XCTUnwrap(Bundle(for: Self.self).url(
            forResource: "bucket-03.valid", withExtension: "json"))
        let bucket = try JSONDecoder().decode(
            FingerprintBucket.self, from: Data(contentsOf: url))
        XCTAssertEqual(bucket.bucketNumber, 3)
        XCTAssertEqual(bucket.files.first?.relativePath, "scores/etude.gp5")
    }
}

final class LibraryStoreTests: XCTestCase {
    func testReplaceThenRestoreLibrary() async throws {
        let configuration = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(
            for: LibraryRecord.self, configurations: configuration)
        let store = LibraryStore(container: container)
        let item = LibraryItem.fixture()
        try await store.replace(with: [item])
        XCTAssertEqual(try await store.all(), [item])
    }
}
```

- [ ] **Step 2: Run tests and verify missing-type failures**

Run:

```bash
xcodebuild test -project mobile/ios/HayaTab.xcodeproj -scheme HayaTab \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro'
```

Expected: compile failure for missing `FingerprintBucket` and `LibraryStore`.

- [ ] **Step 3: Implement Codable domain models**

Define `FingerprintMetadata`, `FingerprintBucket`, `FingerprintFile`,
`LibraryItem`, `DocumentKind`, and `OfflineState`. Map snake-case fields
explicitly. `FingerprintBucket` uses a custom decoder because desktop bucket
zero contains `metadata` instead of `bucket_number`:

```swift
struct FingerprintMetadata: Decodable, Sendable {
    let volumeID: String
    let volumeName: String
    let createdAt: String
    let appVersion: String
    let deviceName: String
    let lastUpdated: String
    let bucketCount: Int
    enum CodingKeys: String, CodingKey {
        case volumeID = "volume_id"
        case volumeName = "volume_name"
        case createdAt = "created_at"
        case appVersion = "app_version"
        case deviceName = "device_name"
        case lastUpdated = "last_updated"
        case bucketCount = "bucket_count"
    }
}

struct FingerprintFile: Decodable, Sendable {
    let relativePath: String
    let title: String
    let artist: String
    let album: String
    let type: String
    let categories: [String]
    let uploadedAt: String
    let uploadedBy: String
    enum CodingKeys: String, CodingKey {
        case relativePath = "relative_path"
        case title, artist, album, type, categories
        case uploadedAt = "uploaded_at"
        case uploadedBy = "uploaded_by"
    }
}

struct FingerprintBucket: Decodable, Sendable {
    let bucketNumber: Int
    let metadata: FingerprintMetadata?
    let files: [FingerprintFile]
    enum CodingKeys: String, CodingKey {
        case bucketNumber = "bucket_number"
        case metadata
        case files
    }

    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        files = try values.decode([FingerprintFile].self, forKey: .files)
        metadata = try values.decodeIfPresent(FingerprintMetadata.self, forKey: .metadata)
        if let number = try values.decodeIfPresent(Int.self, forKey: .bucketNumber) {
            guard (1..<16).contains(number), metadata == nil else {
                throw DecodingError.dataCorruptedError(
                    forKey: .bucketNumber, in: values,
                    debugDescription: "invalid nonzero fingerprint bucket")
            }
            bucketNumber = number
        } else {
            guard metadata?.bucketCount == 16 else {
                throw DecodingError.keyNotFound(
                    CodingKeys.metadata,
                    .init(codingPath: decoder.codingPath,
                          debugDescription: "bucket zero requires metadata"))
            }
            bucketNumber = 0
        }
    }
}

enum DocumentKind: String, Codable, Sendable {
    case pdf
    case gp
}

struct LibraryItem: Identifiable, Codable, Hashable, Sendable {
    let id: String
    let relativePath: String
    let title: String
    let artist: String
    let album: String
    let kind: DocumentKind
    let categories: [String]
    var localFilename: String?
    var remoteRevision: String?
}
```

Derive `id` as a stable SHA-256 hex digest of `volumeID + "\n" +
relativePath`; do not use Swift's process-randomized `hashValue`.

- [ ] **Step 4: Implement SwiftData storage**

Create a `@Model final class LibraryRecord` mirroring `LibraryItem`, and an
`actor LibraryStore` that creates a fresh `ModelContext` per operation.
`replace(with:)` upserts by stable ID and deletes records absent from the new
manifest only after the complete manifest decoded successfully.

- [ ] **Step 5: Run tests**

Run:

```bash
xcodebuild test -project mobile/ios/HayaTab.xcodeproj -scheme HayaTab \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro'
```

Expected: PASS for `ContractFixtureTests` and `LibraryStoreTests`.

- [ ] **Step 6: Commit**

```bash
git add mobile/ios/HayaTab/Domain mobile/ios/HayaTab/Data/LibraryStore.swift mobile/ios/HayaTabTests
git commit -m "feat: add native library domain and persistence"
```

### Task 4: Build the adaptive SwiftUI shell

**Files:**
- Create: `mobile/ios/HayaTab/App/HayaTabApp.swift`
- Create: `mobile/ios/HayaTab/App/AppEnvironment.swift`
- Create: `mobile/ios/HayaTab/Features/Root/RootView.swift`
- Create: `mobile/ios/HayaTab/Features/Library/LibraryView.swift`
- Create: `mobile/ios/HayaTab/Features/Library/LibraryViewModel.swift`
- Create: `mobile/ios/HayaTab/Features/Search/SearchView.swift`
- Create: `mobile/ios/HayaTab/Features/Downloads/DownloadsView.swift`
- Create: `mobile/ios/HayaTab/Features/Settings/SettingsView.swift`
- Create: `mobile/ios/HayaTabUITests/FirstSliceUITests.swift`

- [ ] **Step 1: Write the navigation UI test**

```swift
import XCTest

final class FirstSliceUITests: XCTestCase {
    func testRootDestinationsExist() {
        let app = XCUIApplication()
        app.launchArguments = ["-use-fixture-library"]
        app.launch()
        XCTAssertTrue(app.tabBars.buttons["Library"].exists)
        XCTAssertTrue(app.tabBars.buttons["Search"].exists)
        XCTAssertTrue(app.tabBars.buttons["Downloads"].exists)
        XCTAssertTrue(app.tabBars.buttons["Settings"].exists)
        XCTAssertFalse(app.staticTexts["Keyboard shortcuts"].exists)
    }
}
```

- [ ] **Step 2: Verify the UI test fails**

Run:

```bash
xcodebuild test -project mobile/ios/HayaTab.xcodeproj -scheme HayaTab \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro'
```

Expected: FAIL because the application entry point and tabs do not exist.

- [ ] **Step 3: Implement dependency injection and state**

`AppEnvironment` owns protocol-typed repositories and supplies `.live()` and
`.fixture()` factories. `LibraryViewModel` is `@MainActor @Observable` and
exposes one typed state:

```swift
enum LoadState: Equatable {
    case idle
    case loading
    case loaded
    case failed(AppError)
}

@MainActor @Observable
final class LibraryViewModel {
    private let repository: any LibraryRepositoryProtocol
    var items: [LibraryItem] = []
    var state: LoadState = .idle

    init(repository: any LibraryRepositoryProtocol) {
        self.repository = repository
    }

    func load() async {
        state = .loading
        do {
            items = try await repository.cachedLibrary()
            state = .loaded
        } catch {
            state = .failed(AppError(error))
        }
    }
}
```

- [ ] **Step 4: Implement phone and tablet navigation**

Use a native `TabView` in compact width and `NavigationSplitView` in regular
width. Destinations are Library, Search, Downloads, and Settings. Use system
labels and SF Symbols. Do not add keyboard, MIDI, plugin, storage-path, or
desktop-update settings.

- [ ] **Step 5: Run UI tests**

Expected: PASS on an iPhone simulator. Add an iPad destination run and verify
the sidebar contains the same four labels.

- [ ] **Step 6: Commit**

```bash
git add mobile/ios/HayaTab/App mobile/ios/HayaTab/Features mobile/ios/HayaTabUITests
git commit -m "feat: add adaptive native iOS shell"
```

### Task 5: Add Keychain credentials and a strict WebDAV transport

**Files:**
- Create: `mobile/ios/HayaTab/Data/CredentialStore.swift`
- Create: `mobile/ios/HayaTab/Data/WebDAVClient.swift`
- Create: `mobile/ios/HayaTabTests/CredentialStoreTests.swift`
- Create: `mobile/ios/HayaTabTests/WebDAVClientTests.swift`

- [ ] **Step 1: Write transport tests with a custom URL protocol**

Test that:

- Basic authentication is present only on the configured origin.
- `PROPFIND` sends `Depth: 0`.
- 401 maps to `.authenticationRequired`.
- 404 maps to `.remoteNotFound`.
- paths containing `..`, a foreign host, or a non-HTTPS scheme are rejected.
- cancellation remains `CancellationError`.

Use this handler shape:

```swift
final class URLProtocolStub: URLProtocol {
    nonisolated(unsafe) static var handler:
        (@Sendable (URLRequest) throws -> (HTTPURLResponse, Data))?
    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        do {
            let (response, data) = try Self.handler!(request)
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }
    override func stopLoading() {}
}
```

- [ ] **Step 2: Run and verify failures for missing clients**

Run:

```bash
xcodebuild test -project mobile/ios/HayaTab.xcodeproj -scheme HayaTab \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  -only-testing:HayaTabTests/WebDAVClientTests
```

Expected: compile failure for `WebDAVClient`.

- [ ] **Step 3: Implement Keychain storage**

Define credentials without a custom description that could leak the password:

```swift
struct WebDAVCredential: Codable, Sendable {
    let baseURL: URL
    let username: String
    let password: String
}
```

`CredentialStore` stores `WebDAVCredential` as generic-password data under
service `com.hayasaka7.hayatab.webdav`. Use
`kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`. Treat
`errSecItemNotFound` as no configured account and map every other status to a
sanitized `AppError`; never include password data in descriptions.

- [ ] **Step 4: Implement WebDAV transport**

Use an injected ephemeral `URLSession`. Normalize the base URL once, require
HTTPS outside test builds, create relative URLs with `URLComponents`, and
reject traversal before sending. Supply explicit operations for `propfind`,
`get`, and `download`; do not expose a generic arbitrary request API to views.

- [ ] **Step 5: Run focused tests**

Expected: PASS with no credentials emitted in test diagnostics.

- [ ] **Step 6: Commit**

```bash
git add mobile/ios/HayaTab/Data/CredentialStore.swift mobile/ios/HayaTab/Data/WebDAVClient.swift mobile/ios/HayaTabTests
git commit -m "feat: add secure native WebDAV transport"
```

### Task 6: Refresh and persist the cloud library

**Files:**
- Create: `mobile/ios/HayaTab/Data/LibraryRepository.swift`
- Create: `mobile/ios/HayaTabTests/LibraryRepositoryTests.swift`
- Modify: `mobile/ios/HayaTab/Features/Library/LibraryViewModel.swift`
- Modify: `mobile/ios/HayaTab/Features/Library/LibraryView.swift`
- Create: `mobile/ios/HayaTab/Features/Settings/AccountSheet.swift`

- [ ] **Step 1: Write repository tests**

Cover:

- cached records return before network refresh;
- buckets `00` through `15` decode and merge;
- a missing nonzero bucket is treated as empty;
- malformed bucket data leaves the previous database untouched;
- duplicate `(volumeID, relativePath)` records collapse deterministically;
- refresh cancellation leaves cached state intact.

Define the seam:

```swift
struct WebDAVResponse: Sendable {
    let statusCode: Int
    let data: Data
    let etag: String?
}

protocol WebDAVServing: Sendable {
    func get(path: String) async throws -> WebDAVResponse
    func testConnection() async throws
}

protocol LibraryRepositoryProtocol: Sendable {
    func cachedLibrary() async throws -> [LibraryItem]
    func refresh() async throws -> [LibraryItem]
}
```

- [ ] **Step 2: Verify tests fail**

Expected: compile failure for `LibraryRepository`.

- [ ] **Step 3: Implement repository refresh**

Read `haya-metadata/bucket-00.json` first, require its metadata envelope and
volume ID, then fetch buckets 1–15 with a concurrency limit of four. Decode all
responses before calling `LibraryStore.replace(with:)`. Map file extensions to
`.pdf` or `.gp`; reject unsupported and unsafe relative paths.

- [ ] **Step 4: Connect account and library UI**

`AccountSheet` validates HTTPS URL, username, and nonempty password, tests the
connection, saves only after success, then triggers refresh. `LibraryView`
shows cached rows immediately, uses `.refreshable`, and presents empty,
offline, loading, and failed states without modal alerts for recoverable
network failures.

- [ ] **Step 5: Run repository and UI tests**

Expected: PASS, including a launch with cached fixture data and no network.

- [ ] **Step 6: Commit**

```bash
git add mobile/ios/HayaTab/Data/LibraryRepository.swift mobile/ios/HayaTab/Features mobile/ios/HayaTabTests
git commit -m "feat: sync native cloud library"
```

### Task 7: Download safely and restore offline state

**Files:**
- Create: `mobile/ios/HayaTab/Data/DownloadStore.swift`
- Create: `mobile/ios/HayaTabTests/DownloadStoreTests.swift`
- Modify: `mobile/ios/HayaTab/Data/LibraryStore.swift`
- Modify: `mobile/ios/HayaTab/Features/Downloads/DownloadsView.swift`
- Modify: `mobile/ios/HayaTab/Features/Library/LibraryView.swift`

- [ ] **Step 1: Write atomic-download tests**

Test successful promotion, cancellation cleanup, a mismatched expected byte
count, replacement of a stale temporary file, preservation of an existing
valid file on failure, rejection when the response ETag differs from the
manifest revision, and restoration from persisted `localFilename`.

- [ ] **Step 2: Verify tests fail**

Expected: compile failure for `DownloadStore`.

- [ ] **Step 3: Implement download storage**

`DownloadStore` is an actor. Resolve all destinations beneath
`Application Support/Documents`; create a random `.partial` sibling; stream to
that file; validate nonzero length and optional expected size; set complete
file protection; atomically replace the destination; then persist
`localFilename`. A `defer` removes only the partial file.

Send the manifest revision as `If-Match` when available and compare the response
ETag before promotion. A changed remote revision becomes `.remoteChanged`; the
old local copy remains valid and the library offers refresh rather than
silently replacing either version.

Use stable filenames of `<library-id>.<validated-extension>`. Never use a
remote basename directly as a local path.

- [ ] **Step 4: Add UI state**

Expose queued, downloading with progress, available offline, and failed states.
Downloads are cancellable. The Downloads tab lists only active and offline
items and provides a recoverable delete action with a native confirmation.

- [ ] **Step 5: Run tests**

Expected: all DownloadStore tests pass and a failed download leaves the
preexisting valid fixture byte-for-byte unchanged.

- [ ] **Step 6: Commit**

```bash
git add mobile/ios/HayaTab/Data mobile/ios/HayaTab/Features mobile/ios/HayaTabTests
git commit -m "feat: add safe offline downloads"
```

### Task 8: Package a local PDF and AlphaTab reader

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/mobile-viewer/index.html`
- Create: `frontend/mobile-viewer/main.ts`
- Create: `frontend/mobile-viewer/style.css`
- Create: `frontend/vite.mobile-viewer.config.ts`
- Create: `scripts/package-mobile-viewer.mjs`
- Create: `mobile/ios/HayaTab/Features/Reader/ReaderWebView.swift`
- Create: `mobile/ios/HayaTab/Features/Reader/DocumentReaderView.swift`
- Create: `mobile/ios/HayaTabTests/ReaderBridgeTests.swift`

- [ ] **Step 1: Write browser contract tests**

Add a Playwright test that opens the packaged viewer with a fixture bridge,
posts `{version: 1, type: "ready"}`, accepts only `load`, `playPause`, `stop`,
and `setTempo` commands, and reports a typed error for unknown commands.

- [ ] **Step 2: Verify the test fails**

Run with Node 24 or newer:

`npm --prefix frontend run test:mobile-viewer`

Expected: FAIL because the mobile viewer entry does not exist.

- [ ] **Step 3: Build the dedicated viewer**

The mobile entry imports AlphaTab directly, accepts document bytes only through
the bridge, and never calls Wails bindings or localhost. Add:

```json
{
  "build:mobile-viewer": "vite build --config vite.mobile-viewer.config.ts",
  "test:mobile-viewer": "playwright test tests/e2e/mobile-viewer.spec.ts"
}
```

Output to `frontend/dist-mobile-viewer`. The packaging script deletes and
recreates only `mobile/ios/HayaTab/Resources/Viewer`, then copies the manifest,
HTML, JavaScript, CSS, fonts, and soundfont from the build output. It rejects
absolute or parent-relative manifest entries.

- [ ] **Step 4: Implement the native reader boundary**

For PDF, load the downloaded file with `WKWebView.loadFileURL`, allowing read
access only to its containing Documents directory. For Guitar Pro, load the
packaged viewer and transfer bytes through an allow-listed
`WKScriptMessageHandlerWithReply`. Reject messages whose version is not `1`,
whose type is unknown, or whose payload exceeds the downloaded document size.
Remove handlers when dismantling the view.

- [ ] **Step 5: Test packaging and bridge**

Run:

```bash
npm --prefix frontend run build:mobile-viewer
npm --prefix frontend run test:mobile-viewer
node scripts/package-mobile-viewer.mjs
```

Expected: PASS; packaged HTML contains no `localhost`, `127.0.0.1`, `wails`, or
external script URL.

- [ ] **Step 6: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/mobile-viewer frontend/vite.mobile-viewer.config.ts frontend/tests/e2e/mobile-viewer.spec.ts scripts/package-mobile-viewer.mjs mobile/ios/HayaTab
git commit -m "feat: add offline native document reader"
```

### Task 9: Complete first-slice errors, accessibility, and restoration

**Files:**
- Modify: `mobile/ios/HayaTab/Domain/AppError.swift`
- Modify: `mobile/ios/HayaTab/App/HayaTabApp.swift`
- Modify: `mobile/ios/HayaTab/Features/**/*.swift`
- Modify: `mobile/ios/HayaTabUITests/FirstSliceUITests.swift`

- [ ] **Step 1: Add failing acceptance tests**

Cover account rejection, cached offline launch, download/open, restart and
reader restoration, Dynamic Type XXXL, VoiceOver identifiers, and iPad narrow
split view. Assert recoverable network failures appear inline and not as
blocking alerts.

- [ ] **Step 2: Implement typed presentation**

`AppError` contains a stable code, localized title, recovery suggestion, and
retryability. It never stores raw response bodies or credentials. Use inline
content-unavailable states and status banners; reserve confirmation dialogs for
offline-file deletion and account removal.

- [ ] **Step 3: Add restoration and accessibility**

Persist the selected root destination and last opened library ID with
`@SceneStorage`. Restore the reader only when its validated local file exists.
Add labels, hints, values, identifiers, focus order, reduced-motion handling,
and Dynamic Type-safe layouts. Native tab bars, toolbars, sheets, and controls
receive the system appearance automatically.

For any custom floating reader control on iOS 26 or newer, isolate the new SDK
API:

```swift
@ViewBuilder
func readerControlSurface<Content: View>(
    @ViewBuilder content: () -> Content
) -> some View {
    if #available(iOS 26.0, *) {
        content().padding().glassEffect(.regular.interactive())
    } else {
        content().padding().background(.regularMaterial, in: Capsule())
    }
}
```

- [ ] **Step 4: Run unit and UI tests on iPhone and iPad**

Expected: PASS on one iPhone and one iPad simulator destination.

- [ ] **Step 5: Commit**

```bash
git add mobile/ios/HayaTab mobile/ios/HayaTabTests mobile/ios/HayaTabUITests
git commit -m "feat: harden native iOS first slice"
```

### Task 10: Replace the Wails iOS smoke gate with native CI

**Files:**
- Modify: `.github/workflows/mobile-smoke.yml`
- Modify: `Taskfile.yml`
- Modify: `docs/MOBILE_DEVELOPMENT.md`
- Create: `mobile/ios/README.md`

- [ ] **Step 1: Update workflow path filters**

Include `mobile/ios/**`, `shared/contracts/**`, `shared/fixtures/**`,
`frontend/mobile-viewer/**`, and the native generation/packaging scripts.

- [ ] **Step 2: Replace only the iOS job**

Keep the existing desktop/Android behavior unchanged. The native iOS job must:

1. use the explicit `macos-26` runner;
2. record `xcodebuild -version` and fail unless the simulator SDK is `26.*`;
3. set up Node 24 and Go 1.25.4;
4. run the Go contract tests;
5. build and package the mobile viewer;
6. generate the Xcode project with XcodeGen 2.45.4;
7. run `xcodebuild test` with `CODE_SIGNING_ALLOWED=NO`;
8. upload `.xcresult`, logs, and the unsigned simulator `.app`.

Remove Wails installation, Go iOS archive compilation, and the obsolete Wails
iOS 27 lifecycle gate from this iOS job. Do not remove desktop Wails or Android
jobs.

- [ ] **Step 3: Add task commands**

Add non-Wails tasks:

```yaml
ios-native:generate:
  cmds:
    - bash scripts/generate-native-ios-project.sh

ios-native:test:
  deps: [ios-native:generate]
  cmds:
    - xcodebuild test -project mobile/ios/HayaTab.xcodeproj -scheme HayaTab -destination 'platform=iOS Simulator,name=iPhone 17 Pro' CODE_SIGNING_ALLOWED=NO
```

- [ ] **Step 4: Document Windows and Mac workflows**

Document that Windows supports contract tests, viewer tests, source authoring,
and CI configuration. State explicitly that compilation, signing, Simulator,
XCUITest, archive, TestFlight, and physical-device acceptance require macOS.
Provide exact `task ios-native:generate`, `task ios-native:test`, and Xcode-open
commands.

- [ ] **Step 5: Run Windows verification**

Run:

```powershell
go test ./pkg/sync -run MobileFingerprint -v
fnm exec --using=26.2.0 -- npm.cmd --prefix frontend run build:mobile-viewer
fnm exec --using=26.2.0 -- npm.cmd --prefix frontend run test:mobile-viewer
go test ./...
```

Expected: all available Windows checks pass.

- [ ] **Step 6: Run macOS CI verification**

Run the generated project and both simulator destinations. Expected: native
build, unit tests, UI tests, and unsigned application artifact succeed.

- [ ] **Step 7: Commit**

```bash
git add .github/workflows/mobile-smoke.yml Taskfile.yml docs/MOBILE_DEVELOPMENT.md mobile/ios/README.md
git commit -m "ci: validate native iOS application"
```

## Final verification

- [ ] Run `git diff --check`.
- [ ] Run the full Go suite after a production frontend build.
- [ ] Run the dedicated mobile-viewer build and Playwright suite with Node 24+
  or 26.
- [ ] Confirm the desktop application has no imports from `mobile/ios`.
- [ ] Confirm packaged viewer files contain no development-server URLs.
- [ ] Confirm no password, Authorization header, or document bytes appear in
  logs or JavaScript messages.
- [ ] Confirm GitHub Actions compiles and tests the native app with pinned
  Xcode/XcodeGen inputs.
- [ ] On macOS, open the generated project and complete interactive iPhone,
  iPad, offline, restart, Dynamic Type, VoiceOver, and background/foreground
  checks.
- [ ] Record any macOS-only failure as an implementation defect; do not describe
  the native target as fully validated until those checks pass.
