# China Mobile WebDAV Referer Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the supported China Mobile cloud Referer hostname and make the existing WebDAV transport regression test pass.

**Architecture:** Keep the provider-specific transport switch intact and correct only the stale 10086 Referer literal. The existing table-driven test remains the behavior contract.

**Tech Stack:** Go 1.25, `net/http`, standard Go testing.

---

### Task 1: Correct the 10086 Referer

**Files:**
- Modify: `pkg/sync/webdav.go:78-80`
- Test: `pkg/sync/webdav_test.go:61-67`

- [ ] **Step 1: Run the existing failing regression test**

Run:

```powershell
go test ./pkg/sync -run 'TestCustomTransport_RoundTrip/10086' -count=1
```

Expected: FAIL because the Referer contains `caiyun.islfeixin.10086.cn`.

- [ ] **Step 2: Make the minimal production correction**

Change the 10086 mapping to:

```go
case strings.Contains(host, "10086.cn"):
	clone.Header.Set("Referer", "https://caiyun.feixin.10086.cn/")
	clone.Header.Set("Origin", "https://caiyun.feixin.10086.cn")
```

- [ ] **Step 3: Format and run the focused test**

Run:

```powershell
gofmt -w pkg/sync/webdav.go
go test ./pkg/sync -run 'TestCustomTransport_RoundTrip/10086' -count=1
```

Expected: PASS.

- [ ] **Step 4: Run the package test suite**

Run:

```powershell
go test ./pkg/sync -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the focused fix**

```powershell
git add pkg/sync/webdav.go
git commit -m "fix: restore China Mobile WebDAV referer"
```

