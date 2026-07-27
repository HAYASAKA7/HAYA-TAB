# Mobile Platform Design

## Purpose

HAYA-TAB was chosen as a Wails v3 application so its Go services and web
frontend could support desktop and mobile from one codebase. The repository
already contains iOS and Android build scaffolding, but the application still
assumes a desktop filesystem, a loopback file server, folder watching, and a
desktop-oriented interface.

This design turns the existing application into an iOS- and iPadOS-first mobile
application while supporting Android through the same architecture. Desktop,
iOS, iPadOS, and Android remain products of one repository, one shared business
layer, and one Vue frontend.

## Product Direction

The mobile application is cloud-first:

- WebDAV holds the canonical shared library.
- The local SQLite database provides an immediately available library index.
- Tab files and covers download on demand rather than as a full-library mirror.
- Recently opened files remain cached within a size limit.
- Users can explicitly mark files as available offline.
- Offline metadata and annotation changes are queued and synchronized when the
  connection returns.
- Concurrent changes never silently overwrite either version.

iOS and iPadOS are the primary design targets. Android must support the same
core workflows from the beginning, but iOS conventions take priority when a
shared visual compromise would make the Apple experience worse.

## First Mobile Release Scope

The first mobile release includes:

- Secure WebDAV account setup.
- Cloud volume discovery and metadata synchronization.
- Library browsing, categories, search, and recent items.
- Native document import into the application sandbox.
- PDF and Guitar Pro viewing.
- PDF annotations.
- Metadata editing.
- On-demand downloads and explicit offline pinning.
- Upload of queued changes after reconnection.
- Conflict detection and preservation.
- Native share, document-picker, dialog, haptic, and secure-storage surfaces
  where Wails provides them.

The first mobile release intentionally excludes:

- Desktop folder watching and arbitrary sync folders.
- User-selectable external storage directories.
- Desktop system-tray and window-management behavior.
- JavaScript plugins.
- Web MIDI and desktop pedal configuration.
- Multiple Wails windows.
- A full-library automatic download.
- Native extensions, widgets, purchases, and advanced Apple Pencil features.

Excluded features remain available on desktop. The mobile UI hides them based
on declared capabilities rather than allowing them to fail after interaction.

## Approaches Considered

### Shared application with platform adapters

This is the selected approach. Business logic and frontend features remain
shared, while operating-system behavior is accessed through narrow platform
interfaces. Go build tags select the desktop, iOS, or Android implementation.
The Vue interface adapts using a capability description supplied at startup.

This preserves feature parity without pretending that a desktop filesystem and
a mobile sandbox are interchangeable.

### Separate mobile application in the same repository

A separate entrypoint and frontend would allow unrestricted mobile design, but
bindings, translations, state stores, viewer behavior, and feature fixes would
need to be duplicated. It is rejected because the long-term maintenance cost is
larger than the platform differences.

### Separate mobile repository

A separate repository would isolate release schedules, but it would duplicate
the data model, WebDAV protocol, metadata logic, annotations, and most of the
frontend. It is rejected because the desktop and mobile products would drift.

## Repository and Branch Strategy

Development occurs on a temporary `feat/mobile-runtime` branch or an isolated
worktree for that branch. It is not a permanent mobile fork.

The branch is merged into `main` after the shared desktop regression suite and
the mobile completion matrix pass. Platform-specific behavior remains selected
at build time, allowing ordinary desktop work to continue in the same
repository after the merge.

## Architecture

The existing `pkg/store`, `pkg/sync`, `pkg/metadata`, annotation logic, data
models, and most application services remain shared.

Operating-system behavior moves behind a platform boundary:

```text
Vue application
      │
      ▼
Bound application services
      │
      ▼
Platform interfaces
      ├── capabilities and device class
      ├── application storage and cache paths
      ├── secure credential storage
      ├── document import and export
      ├── tab and cover content delivery
      ├── lifecycle and connectivity
      └── optional folder watching
             │
             ├── desktop implementation
             ├── iOS/iPadOS implementation
             └── Android implementation
```

Application services depend on these interfaces rather than directly calling
desktop Wails APIs or global platform functions. Build-tagged constructors
assemble the correct implementation.

The platform boundary returns typed unsupported-capability errors. Unsupported
features are also absent from the user interface, but typed errors protect
against stale clients and direct binding calls.

## Runtime Capabilities

At startup, the frontend receives one capability object:

```ts
interface RuntimeCapabilities {
  platform: "windows" | "macos" | "linux" | "ios" | "android"
  formFactor: "desktop" | "phone" | "tablet"
  supportsFolderWatching: boolean
  supportsCustomStoragePaths: boolean
  supportsPlugins: boolean
  supportsWebMIDI: boolean
  supportsOfflineCache: boolean
  supportsNativeTabs: boolean
  supportsNativeShare: boolean
}
```

A single frontend composable owns this value. Components ask the composable
about capabilities instead of scattering operating-system string checks across
the UI.

Device class is determined from Wails platform information and current screen
metrics. It can change when an iPad enters split-screen, so responsive layout
continues to rely on viewport constraints even when the nominal form factor is
`tablet`.

## Wails Upgrade Policy

The existing `v3.0.0-alpha.74` dependency predates the current mobile runtime.
The initial migration baseline is the exact
`github.com/wailsapp/wails/v3 v3.0.0-alpha2.118` release.

The Go module, Wails CLI, generated bindings, mobile build assets, and CI use
the same exact version. `@latest` is prohibited in project scripts and CI.

The upgrade is its own checkpoint:

1. Update the module and CLI version.
2. Regenerate bindings and build assets using that version.
3. Reconcile documented Wails API changes.
4. Restore desktop builds and tests before changing mobile behavior.
5. Record any newer Wails version as a separate reviewed dependency change.

## Content Delivery

Desktop initially retains the loopback file server so the dependency upgrade
does not alter proven desktop viewer behavior.

Mobile does not open a loopback port. Tab files, covers, and cloud streams are
served through the Wails in-process asset handler and its `wails://` transport.
The existing HTTP handler semantics can be retained behind this handler, using
relative `/api/file`, `/api/cover`, and `/api/cloud-stream` routes. The
frontend must not construct `127.0.0.1` URLs on mobile.

The content-delivery interface gives viewers a platform-correct URL without
exposing file paths or transport details to Vue components.

## Cloud Index and Cache

Startup opens the local SQLite index before contacting WebDAV. The user can
browse cached metadata immediately.

When connected, synchronization reads WebDAV volume fingerprints and updates
the local index without downloading every file. Covers are lazy-loaded into a
separate size-limited image cache.

Opening a tab follows this flow:

```text
Request tab
    ├── valid cached copy → update last-access time → open viewer
    └── cache miss
          ├── create temporary download
          ├── stream with progress and cancellation
          ├── verify completion and remote identity
          ├── atomically promote into the cache
          └── open viewer
```

Cache metadata records:

- Tab identifier.
- Sandbox-relative local path.
- Remote revision, ETag, or fingerprint identity.
- Byte size.
- Last access time.
- Offline-pin status.
- Download state and integrity information.

Automatic eviction removes least-recently-used unpinned files. Pinned files are
never automatically removed. Partial downloads remain temporary and are safe
to retry or discard.

## Offline Outbox and Conflicts

Metadata, annotation, and organization changes are committed locally before
network transmission. Each change also creates an outbox operation with a
stable operation identifier, target identity, base remote revision, payload,
attempt count, and retry time.

Retries are idempotent. A successful remote operation is recorded before its
outbox entry is removed, preventing duplicate application after interruption.
Retry delays use bounded exponential backoff and respond immediately when the
device reports restored connectivity.

If the remote revision differs from the operation's base revision:

- Neither local nor remote content is deleted.
- The operation enters a conflict state.
- Both versions remain readable.
- The library displays a conflict indicator.
- The user can keep local, keep remote, or perform a supported merge.

Annotation conflicts preserve both annotation documents. Metadata can offer a
field-by-field merge, but no automatic policy may discard a user change.

## Secure Credentials

WebDAV credentials and tokens are stored through iOS Keychain and Android
secure storage using the Wails mobile secure-storage APIs. Desktop retains its
existing credential mechanism behind the same interface.

Credentials, authorization headers, signed URLs, annotation content, and file
contents must not appear in logs. Removing an account removes its secure
credential material while preserving locally cached user data until the user
chooses whether to delete it.

## Native Language Policy

Go remains the business and synchronization language. Vue and TypeScript remain
the interface implementation.

Wails supplies the iOS UIKit and `WKWebView` host. Native Wails APIs are used
before adding custom platform code. Swift or Objective-C is introduced only
for a required iOS capability that Wails cannot expose cleanly, such as a share
extension, widget, StoreKit integration, or advanced Apple Pencil support.

Any custom native module:

- Implements a narrow platform interface.
- Does not duplicate business logic.
- Avoids modifying generated Wails files where possible.
- Has a fallback or an explicit unsupported state on Android and desktop.

There is no Swift rewrite of the application.

## iOS Appearance and Liquid Glass

Standard UIKit surfaces adopt the current Apple appearance when built with the
corresponding Xcode SDK. HAYA-TAB therefore uses native iOS surfaces for:

- Top-level `UITabBar` navigation with SF Symbols.
- Alerts and confirmations.
- Document pickers.
- Share sheets.
- Haptics and other system interactions.

On iOS, Wails native-tab selection events drive the same Vue routes used by
other platforms. Android uses a Vue bottom navigation styled for Android.
Desktop retains its sidebar.

The `WKWebView` content does not automatically become native Liquid Glass.
Library content and viewers remain web-rendered. CSS may use restrained
translucency for floating controls, but it must not attempt to imitate every
native optical effect or reduce readability. Authentic `UIGlassEffect` overlays
would require a small native bridge and are outside the first mobile release.

## Adaptive Interface

### iPhone

- Native bottom tabs for Library, Offline, Search, and Settings.
- Full-screen details and viewers.
- Long-press action menus instead of right-click.
- Bottom sheets instead of desktop-sized dialogs.
- A compact, touch-oriented viewer toolbar.

### iPad

- Native top-level tabs plus a Vue split-view layout.
- Persistent library sidebar and detail area when width permits.
- Collapsible sidebar in portrait and narrow split-screen sizes.
- Centered dialogs where appropriate and sheets for action-heavy flows.
- Viewer layouts that use the larger canvas instead of stretching phone UI.

### Shared mobile rules

- Interactive targets are at least 44 points.
- Layouts respect safe-area insets.
- Dynamic viewport units account for the software keyboard.
- Every hover or right-click action has an explicit touch equivalent.
- Offline, downloading, queued, syncing, and conflict states are visible.
- Download progress and offline-pin state are available from tab surfaces.

## Lifecycle

Backgrounding persists the outbox and cache metadata, pauses nonessential
workers, and closes or flushes resources that cannot safely remain active.
Foregrounding refreshes connectivity, resumes eligible downloads, validates
stale cache entries when necessary, and restarts queued synchronization.

Long operations are cancellable and tolerate application suspension. A
background task is requested only for a bounded operation that Wails and the
operating system explicitly support.

## iOS 27 Compatibility Gate

Apple requires apps built with the iOS 27 SDK to use the scene-based lifecycle.
Inspection of Wails `v3.0.0-alpha2.118` shows the older
`UIApplicationDelegate` startup and no `UISceneDelegate` or scene manifest.

Therefore:

- iOS and iPadOS 26 are the initial supported Apple targets.
- An iOS 27 build is not releasable until Wails adopts the scene lifecycle or a
  reviewed upstream-compatible patch supplies it.
- HAYA-TAB does not silently claim iOS 27 support because the web content
  happens to render in a simulator.
- A required lifecycle patch should be contributed upstream.
- A temporary patch may be maintained only if it is small, tested, documented,
  and removable after upstream adoption.

The project does not maintain a broad permanent Wails fork.

## Error Handling

- Missing credentials open the app in signed-out or cached-offline mode.
- WebDAV failure never blocks access to the local index or valid cached files.
- Download failures retain no partially promoted cache entry.
- Cache corruption triggers removal and a clean re-download.
- Sync failures attach to individual outbox operations instead of stopping the
  entire queue.
- Unsupported operations return typed capability errors.
- Viewer failures offer retry and preserve the downloaded source for diagnosis
  unless integrity verification failed.
- Account removal, cache clearing, and conflict resolution require explicit
  confirmation appropriate to their data-loss risk.

## Testing

### Unit and contract tests

- Cache insertion, validation, pinning, and eviction.
- Temporary-download promotion and interruption cleanup.
- Outbox idempotency, retry scheduling, and persistence.
- Conflict detection and preservation.
- Capability decisions and platform adapter contracts.
- Sandbox path validation and traversal rejection.

### Integration tests

- WebDAV fingerprint synchronization.
- Lazy cover and tab download.
- Offline metadata and annotation edits.
- Reconnect and outbox processing.
- Credential rejection and replacement.
- Remote revision conflicts.

### Frontend tests

- Phone and tablet navigation.
- Native-tab event routing on iOS.
- Safe-area and software-keyboard layout.
- Long-press actions and bottom sheets.
- Offline, download, queue, sync, and conflict indicators.
- Absence of unsupported mobile settings.
- Desktop sidebar and dialog regression coverage.

### Device matrix

The release-critical matrix is:

- iPhone on iOS 26.
- iPad on iPadOS 26 in portrait and landscape.
- iPad in narrow split-screen mode.
- Android phone.
- Android tablet.

Simulator and emulator checks are necessary but insufficient. File import,
secure storage, background/foreground transitions, offline behavior, WebDAV
reconnection, and viewer performance must pass on physical devices.

## Delivery Sequence

1. Upgrade and pin Wails; restore all desktop tests.
2. Introduce platform interfaces and capability reporting without changing
   desktop behavior.
3. Replace mobile loopback content delivery with the in-process handler.
4. Add secure credentials, the cloud index, cache, and offline outbox.
5. Build iOS native navigation and the adaptive phone/tablet Vue layouts.
6. Connect Android to the same services and mobile flows.
7. Add device-level lifecycle, offline, conflict, and packaging verification.
8. Merge the branch after the completion criteria pass.

## Completion Criteria

The first mobile milestone is complete only when every supported device class
can:

1. Store credentials securely and connect to WebDAV.
2. Browse and search the cloud library from the local index.
3. Open and cache PDF and Guitar Pro files.
4. Pin files and read them offline.
5. Edit annotations and metadata offline.
6. Reconnect and synchronize without losing local or remote changes.
7. Display and resolve conflicts while preserving both versions.

In addition:

- Desktop Go tests, frontend builds, and Playwright regressions pass.
- Mobile builds use one exact Wails module and CLI version.
- No mobile code depends on a loopback port.
- No secret material is emitted to logs.
- iOS signing and TestFlight packaging pass on the supported Xcode toolchain.
- Unsupported features are capability-gated and absent from mobile settings.
- iOS 27 remains blocked until the scene-lifecycle gate is satisfied.
