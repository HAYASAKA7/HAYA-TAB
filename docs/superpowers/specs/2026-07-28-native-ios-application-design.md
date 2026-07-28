# Native iOS Application Design

## Status

Approved design direction, pending implementation planning.

This document supersedes `2026-07-27-mobile-platform-design.md` for mobile
product architecture. The earlier design remains an historical record of the
Wails-based approach; it must not be used as the implementation target for the
native iOS application.

## Purpose

HAYA-TAB's responsive Vue interface is a desktop application adapted to a
small viewport. It does not provide a sufficiently native iPhone or iPad
experience, and capability hiding still leaves desktop concepts such as
keyboard configuration, folder watching, plugins, and desktop update behavior
in the mobile product architecture.

The first mobile product will therefore be a native iOS and iPadOS application
written in Swift. The existing Go, Wails, and Vue desktop application remains
unchanged. The iOS application shares data formats, cloud protocol contracts,
compatibility fixtures, and selected packaged viewer assets with desktop, but
does not embed the desktop Go backend or Wails runtime.

Quality and Apple-platform fit take priority over maximum source-code reuse.

## Product Scope

The first mobile release is iOS- and iPadOS-only. Android design and
implementation are deferred until the native Apple product has been validated.

The mobile product is cloud-first and includes:

- WebDAV account setup and secure credential storage.
- Cloud library browsing, recent items, favorites, and collections.
- Library search.
- On-demand downloads and explicit offline availability.
- Native document import into the application sandbox.
- PDF and Guitar Pro document viewing.
- Safe synchronization, retry, and conflict preservation.
- Native iPhone and iPad navigation.
- Native appearance, accessibility, lifecycle, and background-transfer
  behavior.

The first release excludes:

- Desktop folder watching and arbitrary sync folders.
- User-selected desktop storage directories.
- Desktop window, tray, and application-update settings.
- JavaScript plugins.
- Desktop keyboard-shortcut configuration.
- MIDI and pedal configuration.
- Android.

Excluded desktop features are not compiled into the iOS interface. They are not
represented as disabled or hidden desktop settings.

## Approaches Considered

### Native SwiftUI application with native Swift data layer

This is the selected approach. SwiftUI owns the product interface, and native
Swift services use Apple frameworks for persistence, credentials, networking,
background execution, and file management. Existing viewer technology may run
inside a controlled `WKWebView`.

This approach produces the best Apple-platform behavior and avoids a binary
bridge between Swift and a Go runtime.

### Native SwiftUI application with an embedded Go core

This would reuse more sync and domain implementation. It was rejected because
the required bridge would complicate Swift concurrency, errors, background
execution, file coordination, packaging, and debugging. It would also preserve
desktop lifecycle assumptions inside the mobile process.

### Continue adapting the Wails/Vue application

This would minimize initial duplication but would keep mobile navigation,
settings, lifecycle, and interaction design coupled to the desktop product. It
was rejected because the existing mobile simulation demonstrated that
responsive layout and capability hiding are not enough to create a high-quality
iOS application.

## Repository and Branch Strategy

The iOS application lives in the existing repository as a separate application
target. It does not use a separate repository.

Initial work occurs on a temporary `feat/ios-native` branch or an isolated
worktree for that branch. The branch is not a permanent product fork. Complete,
tested vertical slices are merged into `main` so desktop and iOS changes remain
visible in one history.

The intended layout is:

```text
HAYA-TAB/
├── existing Go/Wails/Vue desktop application
├── mobile/
│   └── ios/
│       ├── HayaTab.xcodeproj
│       ├── App/
│       ├── Features/
│       ├── Data/
│       ├── Platform/
│       ├── Viewer/
│       └── Tests/
├── shared/
│   ├── contracts/
│   └── fixtures/
└── viewer/
    └── packaged viewer assets
```

Exact Xcode groups may differ from filesystem directories, but feature
ownership and platform boundaries must remain clear.

## Runtime Architecture

Wails remains the desktop runtime only. The iOS application does not start a
Wails process, a Vite development server, or a loopback web server.

```text
SwiftUI screens
      │
      ▼
Feature models and Swift actors
      │
      ▼
Native repositories
      ├── URLSession and WebDAV
      ├── native persistence
      ├── Keychain
      ├── background scheduling
      └── protected file storage

Selected document
      │
      ▼
SwiftUI reader container
      │
      ▼
Local WKWebView package
      │
      └── existing PDF or AlphaTab rendering assets
```

Swift actors isolate mutable library, sync, and download state. UI updates occur
through typed feature models on the main actor. Long-running work uses
structured concurrency and supports cancellation.

The Swift data layer implements the mobile behavior directly. Desktop and iOS
stay compatible through versioned contracts and fixtures rather than shared
runtime code.

## Compatibility Contracts

The repository defines versioned, implementation-neutral contracts for:

- Remote library paths and identifiers.
- Metadata encoding and optional fields.
- WebDAV discovery and synchronization semantics.
- Remote revisions, ETags, or equivalent fingerprints.
- Conflict representation.
- Viewer document descriptors.
- Annotation formats included in the first release.

Fixtures include valid, missing-field, legacy, conflicting, malformed, and
large-library examples. Both Go desktop tests and Swift iOS tests consume the
same fixtures. A format or protocol change is incomplete until both
implementations pass compatibility tests.

Contracts must not expose Go-specific or Swift-specific types.

## Native Application Structure

Feature modules own their screens, models, and feature-specific services.
Cross-feature services are limited to well-defined platform and data
interfaces.

The primary modules are:

- `Library`: cloud items, recent items, favorites, and collections.
- `Search`: library search and filtering.
- `Downloads`: offline state, progress, retry, and storage usage.
- `Settings`: account, cloud connection, appearance, sync, and support.
- `Reader`: document lifecycle and the packaged viewer bridge.
- `Sync`: manifest comparison, transfer coordination, retries, and conflicts.

No feature accesses Keychain, raw persistence, or filesystem paths directly.
Those responsibilities remain behind native repository and platform
interfaces.

## Navigation and Adaptive Layout

### iPhone

The root interface uses four native destinations:

- Library.
- Search.
- Downloads.
- Settings.

The reader is a focused full-screen navigation destination. Secondary actions
use native menus, sheets, and toolbars rather than desktop context menus or
fixed-size dialogs.

### iPad

The same destinations use `NavigationSplitView` with a sidebar and detail
column. The layout supports portrait, landscape, narrow split view, pointer
input, and hardware-keyboard navigation where it improves standard iPad use.

Opening a document replaces the detail area with the reader. Narrow layouts
collapse through native navigation behavior rather than maintaining a separate
tablet route hierarchy.

## Appearance and Accessibility

The application uses native SwiftUI navigation bars, tab bars, lists, menus,
sheets, alerts, controls, and materials. It does not reproduce Apple's current
appearance with custom blur simulations.

The minimum deployment target is iOS and iPadOS 18. Builds made with newer SDKs
use system-provided Liquid Glass behavior on supported iOS and iPadOS versions,
including iOS 26 and 27, through availability-safe native components. Earlier
supported systems retain their standard native appearance.

The application supports:

- Dynamic Type without clipped primary actions.
- VoiceOver labels, traits, reading order, and focus restoration.
- Increased contrast and reduced transparency.
- Reduced motion.
- Light and dark appearance.
- Safe areas and software-keyboard avoidance.
- At least 44-point interactive targets.
- iPad pointer and keyboard navigation where platform-standard.

There is no desktop keyboard-shortcut settings page. Any iPad keyboard commands
are contextual native commands and are discoverable through the system.

## Cloud Library and Persistence

WebDAV is the cloud source for the shared library. The local database is an
offline-capable index and application cache, not an independent desktop folder
mirror.

Startup opens the local index first so the user can browse valid cached state
without waiting for a network request. A refresh then reconciles the remote
manifest.

Native persistence stores:

- Library metadata and organization.
- Remote identity and revision information.
- Download and offline-pin state.
- Pending operations and retry state.
- Conflict records.
- Reader restoration state.

Credentials and tokens are stored in Keychain. Secrets, authorization headers,
document contents, and signed resource locations must not appear in logs.

## Downloads and Offline Behavior

Swift owns background transfers because iOS controls application suspension and
background execution. Transfers use system networking and background APIs where
appropriate.

Opening a remote document follows this flow:

```text
Request document
    ├── valid local copy → update access state → open reader
    └── cache miss
          ├── create temporary destination
          ├── download with progress and cancellation
          ├── validate completion and remote identity
          ├── atomically promote the file
          └── open reader
```

An interrupted or invalid transfer never replaces a valid file. Documents
marked for offline use are not automatically evicted. The user can see pending,
downloading, available-offline, failed, and conflict states.

Imported documents are selected through the native document picker, copied
into protected application storage, and then indexed. The application does not
retain an unsafe dependency on an external security-scoped URL.

The interface does not promise exact background execution times because iOS
controls scheduling.

## Synchronization and Conflicts

Synchronization is expressed as explicit phases:

1. Read local and remote manifests.
2. Compute a deterministic plan.
3. Execute transfers and metadata operations.
4. Validate remote and local identities.
5. Commit state atomically.

Retries are idempotent and use bounded backoff. Connectivity restoration may
trigger an eligible retry, but a single failing item does not stop unrelated
work.

When both local and remote content changed from the same base revision:

- Neither version is silently deleted.
- Both versions remain recoverable.
- The operation enters a conflict state.
- The library shows the conflict.
- A native resolution workflow explains the choices and their consequences.

Automatic conflict resolution is allowed only where it is demonstrably
lossless.

## Reader and Web Content Boundary

The existing PDF and AlphaTab renderer may be reused as packaged local web
assets inside `WKWebView`. This is a reader implementation detail, not the
application shell.

The reader:

- Loads bundled assets without Vite, Wails, or localhost.
- Receives document access through a controlled local mechanism.
- Uses a versioned, allow-listed Swift-to-JavaScript message contract.
- Validates message type and payload before invoking native behavior.
- Does not expose credentials, unrestricted paths, or arbitrary native calls.
- Works after restart with a valid offline document.
- Reports load and render failures through typed native state.

Navigation, settings, library, downloads, alerts, and sheets remain native
SwiftUI.

## Error Presentation

Errors are typed at service boundaries and converted into actionable user
states:

- Field validation appears inline.
- Recoverable network and sync failures use status rows or banners.
- Account setup and conflict resolution use native sheets.
- Confirmation dialogs are reserved for consequential actions.
- Destructive choices state what will be removed and whether recovery is
  possible.
- Offline mode retains access to valid local data.

The iOS application does not use desktop-style pop-up windows or arbitrary
dialog dimensions.

## Security

The initial security baseline requires:

- Keychain-backed secrets.
- Appropriate file data protection for cached and imported documents.
- HTTPS by default.
- Explicit trust handling rather than silent certificate bypass.
- Sanitized diagnostic logging.
- No secrets passed into JavaScript.
- A narrow, validated `WKWebView` bridge.
- Path normalization and containment checks.
- Atomic replacement of downloaded files.
- Privacy manifest and permission text review before distribution.

## Testing Strategy

### Swift tests

- Library mapping and persistence.
- Manifest comparison and sync planning.
- Retry and cancellation behavior.
- Conflict detection and preservation.
- Download promotion and interrupted-transfer recovery.
- Reader state restoration.
- Error-to-presentation mapping.

### Protocol and compatibility tests

- Swift and Go consume the same contract fixtures.
- WebDAV tests cover pagination, missing properties, malformed responses,
  authentication failure, revisions, and reconnect behavior.
- Files and metadata produced by either client remain readable by the other.

### Reader integration tests

- Every native-to-JavaScript and JavaScript-to-native message is covered.
- Invalid message types, oversized payloads, and unsafe paths are rejected.
- PDF and Guitar Pro fixtures open from packaged local storage without a
  network server.

### UI and device tests

- XCUITest covers account setup, library refresh, search, download, offline
  open, restart restoration, and conflict presentation.
- Representative iPhone and iPad layouts are tested, including narrow iPad
  split view.
- Accessibility tests cover labels, focus order, Dynamic Type, contrast, and
  reduced motion.
- Physical-device tests cover background transfer, suspension, reconnection,
  memory pressure, and offline recovery.

## Windows-First and macOS Validation Workflow

Initial repository work is performed on Windows. Windows can support:

- Contract and fixture definition.
- Swift source and project structure authoring.
- Viewer packaging and browser-level viewer tests.
- Existing Go and frontend regression tests.
- Repository scripts and CI configuration.
- Review of generated project files and deterministic assets.

Windows cannot provide authoritative validation of the native target. Native
iOS compilation, signing, Simulator execution, XCUITest, archives, and physical
device debugging require macOS with Xcode.

A macOS GitHub Actions job uses an explicit Xcode version to compile and test
the iOS target. It must not silently track an unpinned `latest` toolchain.
Remote CI is an early correctness gate, not a substitute for interactive
Simulator and device testing.

After the Windows implementation stage, the user performs the final Xcode,
Simulator, signing, and device checks on a Mac. Until those checks pass, the
work may be described as implemented and statically/contract tested, but not as
fully validated on iOS.

## Delivery Sequence

1. Define shared contracts, compatibility fixtures, and the native target
   structure.
2. Build the SwiftUI iPhone shell and iPad split navigation.
3. Implement Keychain-backed WebDAV setup and the local library index.
4. Deliver a vertical slice that connects, lists, downloads, and opens one
   document offline.
5. Add search, downloads management, import, retries, and conflicts.
6. Package the reader and harden its native message boundary.
7. Complete accessibility, state restoration, platform appearance, and iPad
   behavior.
8. Add pinned-Xcode CI, then validate through Xcode, Simulator, TestFlight, and
   physical devices on macOS.
9. Merge tested vertical slices into `main`; do not maintain a permanent mobile
   fork.

## First Releasable Slice

The first releasable slice is complete when a user can:

1. Launch a native iPhone or iPad application.
2. Securely connect a WebDAV account.
3. Browse the cloud library from a locally persisted index.
4. Download a supported document with visible progress.
5. Open that document without a network connection.
6. Restart the application and recover the same library and reader state.
7. Encounter a failed or conflicting sync without losing either valid version.

In addition:

- The desktop application retains its existing behavior.
- Desktop Go and frontend regression tests pass.
- Swift and Go compatibility fixtures pass.
- The viewer requires no development server or loopback port.
- Unsupported desktop settings are absent from the native application.
- Secrets are protected and excluded from logs and web content.
- The pinned macOS CI build and tests pass.
- Simulator and physical-device acceptance checks pass before release.

## Deferred Decisions

The following are intentionally deferred until the first iOS product is
validated:

- Android architecture and language.
- Widgets, share extensions, and other app extensions.
- StoreKit purchases.
- Advanced Apple Pencil workflows.
- Replacing the existing PDF or AlphaTab web renderer with a native renderer.
- Cross-platform client-code sharing beyond contracts and fixtures.
