# China Mobile WebDAV Referer Regression Design

## Problem

`TestCustomTransport_RoundTrip/10086` expects the China Mobile cloud Referer
and Origin to use `caiyun.feixin.10086.cn`. Commit `89377c7` changed only the
Referer to `caiyun.islfeixin.10086.cn`, leaving the Origin and test unchanged.

The current official China Mobile cloud entry point remains under
`caiyun.feixin.10086.cn` and redirects to the current 139 Cloud application.
No authoritative source supports the inserted `islfeixin` hostname.

## Design

Restore the 10086 Referer to:

```text
https://caiyun.feixin.10086.cn/
```

Keep the existing Origin:

```text
https://caiyun.feixin.10086.cn
```

The existing table-driven transport test already reproduces the regression and
therefore serves as the required failing test. No production behavior outside
the 10086 host mapping changes.

## Verification

- Run the focused `10086` subtest and observe it pass after the change.
- Run all `pkg/sync` tests.
- Run the full Go suite as final verification.

