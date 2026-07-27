# Dialog Consistency Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Release the completed dialog, notification, modal-containment, and WebDAV fixes as HAYA-TAB 3.1.7 and merge them locally into `main`.

**Architecture:** Update every tracked source of the application version together, document the release and verified development setup, then verify on the feature branch and again after merging. Preserve the unrelated npm-generated lockfile metadata diff outside all commits.

**Tech Stack:** Go 1.25, Wails v3, Vue 3, TypeScript, Vite, Playwright, Git

---

### Task 1: Update release metadata and documentation

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`
- Modify: `build/config.yml`
- Modify: `build/darwin/Info.dev.plist`
- Modify: `build/darwin/Info.plist`
- Modify: `build/linux/nfpm/nfpm.yaml`
- Modify: `build/windows/info.json`
- Modify: `docs/API.md`
- Modify: `docs/DEVELOPMENT.md`
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `internal/app/app.go`

- [ ] **Step 1: Change all current application-version fields from 3.1.6 to 3.1.7**

Update only HAYA-TAB release metadata. Retain `3.1.6` in changelog history and
ignore third-party/generated source-map versions.

- [ ] **Step 2: Add the 3.1.7 changelog entry**

Add a `2026-07-27` patch-release entry describing unified modal dimensions and
structure, unified notifications, modal input/mobile containment, and restored
China Mobile WebDAV referer behavior.

- [ ] **Step 3: Update the development guide**

Document Go 1.25+, Node.js 24 LTS, the Wails v3 alpha.74 CLI installation
command, `wails3 task dev`, and GOPATH-bin PATH troubleshooting for PowerShell
and Unix shells.

- [ ] **Step 4: Validate release references**

Run:

```powershell
git grep -n "3\.1\.6"
git diff --check
```

Expected: `3.1.6` remains only in release history; no whitespace errors.

- [ ] **Step 5: Commit only intentional release changes**

Temporarily restore the four unrelated `peer` fields before staging
`frontend/package-lock.json`, commit the release, then reapply those four
worktree-only deletions.

### Task 2: Verify the release candidate

- [ ] **Step 1: Run backend tests**

```powershell
go test ./...
```

Expected: all packages pass.

- [ ] **Step 2: Run the frontend production build**

```powershell
npm --prefix frontend run build
```

Expected: TypeScript and Vite build successfully.

- [ ] **Step 3: Run Chromium end-to-end tests**

```powershell
npm --prefix frontend exec playwright test -- --project=chromium
```

Expected: all Chromium tests pass.

- [ ] **Step 4: Audit committed and uncommitted changes**

Confirm the release commit contains only intended files and the four lockfile
metadata deletions remain unstaged.

### Task 3: Merge and verify `main`

- [ ] **Step 1: Merge the feature branch locally**

From the primary worktree, merge `fix/unified-dialogs-notifications` into
`main` without pushing or tagging.

- [ ] **Step 2: Verify the merged backend**

```powershell
go test ./...
```

Expected: all packages pass.

- [ ] **Step 3: Verify the merged frontend**

```powershell
npm --prefix frontend run build
```

Expected: TypeScript and Vite build successfully.

- [ ] **Step 4: Audit final state**

Confirm `main` contains version 3.1.7 and all feature commits, unrelated
untracked files in the primary worktree are untouched, and the feature
worktree retains its unrelated lockfile-only diff.
