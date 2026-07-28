# Mobile Development

HAYA-TAB's mobile runtime is iOS-first and cloud-first. The shared Go/Vue application remains the product core, while Wails supplies each native host. Mobile packaging is experimental until the smoke workflows and physical-device checks pass for a release.

## Pinned toolchain

- Go 1.25.4
- Node.js 24 for CI; Node.js 24 or 26 for local frontend work
- Wails v3.0.0-alpha2.118

Install the exact Wails CLI and ensure Go's binary directory is on `PATH`:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.118
export PATH="$(go env GOPATH)/bin:$PATH"
wails3 version
```

Do not install an unpinned CLI version: the CLI must match the version selected by this repository.

## iOS and iPadOS

Building or running the Apple target requires macOS and Xcode. The initial validation target is iOS/iPadOS 26 with the Wails-compatible minimum deployment target left at iOS 15.0.

Install or verify the required Apple tooling:

```bash
wails3 task ios:install:deps
```

Build or run in a booted simulator:

```bash
wails3 task ios:build
wails3 task ios:run
```

For a physical iPhone, first generate/open the Xcode project and configure signing for your Apple Developer team:

```bash
wails3 task ios:xcode
wails3 task common:ios:device:list
wails3 task common:ios:run:device \
  PROJECT=build/ios/xcode/main.xcodeproj \
  SCHEME=HAYA-TAB \
  UDID=<device-udid> \
  BUNDLE_ID=com.hayasaka7.hayatab \
  TEAM_ID=<apple-team-id>
```

The project intentionally uses UIKit's native tab bar on iOS. Standard UIKit navigation receives the operating system's current appearance, including Liquid Glass where the installed iOS version supplies it. The Vue/WebView content remains governed by HAYA-TAB's CSS design system; it does not inherit native materials automatically.

### iOS 27 lifecycle gate

iOS 27 packaging is blocked until the pinned Wails native host provides both UIScene delegate support and an application-scene manifest. Check the selected module, without relying on a hard-coded Go module cache path:

```bash
wails3 task verify:ios27:lifecycle
```

Exit code 27 means the gate is working and iOS 27 packaging must not be declared supported. With Wails v3.0.0-alpha2.118 that failure is currently expected. Do not add this gate to the iOS 26 simulator build until the pinned Wails version passes it and the lifecycle behavior has been reviewed on device.

## Android

Install Android Studio or the command-line SDK, Android platform/API 35, build tools 35.0.0, and Android NDK r26d. Set `ANDROID_HOME`; set `ANDROID_NDK_HOME` if the NDK is not located at the task's default path.

```bash
wails3 task android:install:deps
wails3 task android:build ARCH=arm64
wails3 task android:run
```

Unsigned smoke packages can be assembled locally with:

```bash
wails3 task android:assemble:apk
wails3 task android:assemble:aab:release
```

The APK is intended for local smoke installation. The AAB is unsigned and cannot be published until release signing is configured outside the repository.

## Cloud/WebDAV test account

Use a dedicated test-only WebDAV account and an isolated remote directory. Enter the endpoint, username, and password through HAYA-TAB Settings. Never put credentials in source files, task variables, workflow YAML, screenshots, logs, or committed database files. CI credentials, when device integration tests are added, must come from protected repository secrets and should have the minimum permissions needed for the test directory.

At minimum, verify on each target that a cloud library can be listed, one tab can be downloaded for offline use, a local change can be uploaded, and reconnecting does not duplicate metadata.

## Smoke CI contract

`.github/workflows/mobile-smoke.yml` uses exact Go, Node, and Wails versions. It compiles unsigned Android APK/AAB artifacts and an iOS 26 simulator binary without signing credentials. The Apple job prints the Xcode and simulator SDK versions and fails explicitly if the runner no longer exposes an iOS 26.x simulator SDK.
