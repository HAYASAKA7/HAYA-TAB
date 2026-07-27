# Dialog Consistency Release Design

## Goal

Integrate the completed dialog, notification, modal-containment, and WebDAV
referer fixes into `main` as the HAYA-TAB 3.1.7 patch release, with consistent
release metadata and accurate development documentation.

## Release Strategy

Prepare and verify the release metadata on
`fix/unified-dialogs-notifications`, then merge that branch locally into
`main`. This keeps the implementation, documentation, changelog, and version
bump together while leaving `main` untouched until the release candidate has
passed verification.

The merge does not create or push a Git tag and does not publish a release.
Those are separate external actions that were not requested.

## Version Scope

The release version changes from 3.1.6 to 3.1.7. Every tracked source that
represents the HAYA-TAB application version must be updated:

- `README.md`
- `build/config.yml`
- `build/darwin/Info.dev.plist`
- `build/darwin/Info.plist`
- `build/linux/nfpm/nfpm.yaml`
- `build/windows/info.json`
- `docs/API.md`
- `frontend/package.json`
- the root package records in `frontend/package-lock.json`
- `internal/app/app.go`

Third-party version strings and generated source-map contents are outside the
version bump.

## Documentation

`CHANGELOG.md` receives a 3.1.7 entry dated 2026-07-27. It summarizes:

- consistent modal structure, dimensions, and responsive containment;
- unified toast and notification presentation;
- input isolation so modal keystrokes do not trigger background shortcuts;
- restored China Mobile WebDAV referer compatibility.

`docs/DEVELOPMENT.md` will:

- identify Go 1.25+, Node.js 24 LTS, npm, and Wails v3 as the development
  baseline;
- retain `wails3 task dev` as the repository's development entry point;
- document installation of the matching Wails CLI version;
- explain that `$(go env GOPATH)\bin` must be on PATH;
- provide PowerShell commands for current-session PATH repair and verification.

## Existing Worktree Change

Running `npm install` removed four `peer` metadata fields from
`frontend/package-lock.json`. Those unrelated deletions must remain preserved
in the worktree but excluded from release commits and from the merge. Only the
two root package-version fields in that file belong to the 3.1.7 change.

## Verification

Before merging:

1. Confirm all tracked HAYA-TAB 3.1.6 version references were intentionally
   updated or retained only as changelog history.
2. Run `go test ./...`.
3. Run the frontend production build with Node.js 24 or a newer compatible
   installed version.
4. Run the Chromium Playwright end-to-end suite.
5. Review the staged diff to ensure the unrelated lockfile metadata changes are
   absent.

After merging into `main`, rerun the Go test suite and frontend production
build, then confirm `main` contains the release commits and remains free of
new unintended changes.

## Merge and Cleanup

Merge `fix/unified-dialogs-notifications` locally into `main`. Do not push,
tag, or publish. Preserve the feature worktree if its unrelated lockfile
metadata change prevents safe cleanup; report that residual change explicitly
instead of discarding it.
