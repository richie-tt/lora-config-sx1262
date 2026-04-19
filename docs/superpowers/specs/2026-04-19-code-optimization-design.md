# Code Optimization Design: lora-config-SX1262

**Date:** 2026-04-19
**Goal:** Improve code quality and runtime efficiency while preserving all serial timing constants.
**Approach:** Code quality refactor + cherry-picked performance wins (Approach A+).

## Constraints

All serial timing constants are frozen and must not be modified:

| Constant | Value | Location |
|----------|-------|----------|
| `atTimeout` | 3s | `internal/device/at.go:10` |
| `postExitDelay` | 1s | `internal/device/at.go:11` |
| `preEnterDelay` | 300ms | `internal/device/at.go:12` |
| `interCmdDelay` | 100ms | `internal/device/at.go:13` |
| `enterATRetries` | 3 | `internal/device/at.go:14` |
| Read timeout | 100ms | `internal/device/serial.go:33` |
| Baud rate | 115200 | `internal/tui/model.go:564` |

## Change 1: Split `model.go` into focused files

`internal/tui/model.go` (898 lines) handles too many concerns. Split into:

| New File | Responsibility | Source Lines |
|----------|----------------|--------------|
| `model.go` | Model struct, `InitialModel`, `Init`, `Update` message dispatcher | ~100 |
| `navigation.go` | Focus navigation: `focusNext`, `focusPrev`, `focusNextInColumn`, `focusPrevInColumn`, `focusSwitchColumn`, `fieldPosition`, `updateFocus`, `openDropdownIndex`, `editingFieldIndex` | ~160 |
| `keys.go` | Key handling: `handleKey`, `handleNumInputKey`, `handleDropdownKey`, `handleEnter` | ~140 |
| `commands.go` | BubbleTea commands: `connectCmd`, `setParamCmd`, `restoreCmd`, `rebootCmd` | ~50 |
| `view.go` | `View()` method, all `lipgloss.NewStyle` style variables | ~200 |
| `ansi.go` | ANSI string utilities: `overlayString`, `truncateToWidth`, `skipWidth`, `stripAnsi` | ~90 |

All files remain in package `tui`. No public API changes. Existing tests continue to work without modification since they test package-level functions.

## Change 2: Extract `withATSession` helper

Replace the repeated `enterAT -> do work -> exitAT` pattern with a session helper on `SerialConn`:

```go
func (s *SerialConn) withATSession(fn func() error) error {
    s.LockSession()
    defer s.UnlockSession()

    if err := enterAT(s); err != nil {
        return err
    }

    fnErr := fn()
    exitErr := exitAT(s)

    if fnErr != nil {
        return fnErr
    }
    return exitErr
}
```

**Functions refactored:**
- `SetParam` -- uses `withATSession`, callback sends AT command and checks OK
- `ReadAllParamsAndVersion` -- uses `withATSession`, callback reads ALLP + version
- `Restore` -- becomes `SetParam(conn, "RESTORE", "1")` (already is, no change)

**Functions NOT refactored:**
- `Reboot` -- no `exitAT` (device reboots), keeps manual `LockSession`/`UnlockSession`

**Errors fixed:**
- 4 instances of `_ = exitAT(conn)` eliminated; exit errors now surfaced via `withATSession`
- 3 instances of manual lock/unlock eliminated; defer handles cleanup

## Change 3: Fix `sendAndRead` partial response and polling

### 3a: Partial response on timeout becomes an error

**Current behavior (`serial.go:84-87`):** When deadline passes with data in buffer but no OK/ERROR/+++ terminator, returns partial data with `nil` error. Callers treat incomplete data as valid.

**New behavior:** Return both the partial data and an error:
```go
return strings.TrimSpace(resp), fmt.Errorf("timeout: incomplete response: %s", resp)
```

### 3b: Yield on empty/error reads to prevent CPU spin

**Current behavior:** Tight `for` loop with `time.Now().Before(deadline)`. If serial driver returns immediately with 0 bytes (no error), loop busy-spins.

**New behavior:** Add 1ms yield after zero-byte or error reads:
```go
if err != nil {
    time.Sleep(time.Millisecond)
    continue
}
```

This does NOT change any timing constant. The 1ms yield only prevents CPU spin on fast-returning drivers.

**Caller impact note:** No caller currently handles the partial-data case intentionally. `parseALLP` would fail on incomplete data anyway, and `enterAT` checks for `+++` in the response. Making the error explicit surfaces failures that are currently silent.

## Change 4: Proper error handling for ignored errors

### 4a: `ResetInputBuffer` in `sendAndRead` (`serial.go:57`)

**Current:** `_ = s.port.ResetInputBuffer()`
**New:** Return error. If buffer can't be cleared, the command will produce unreliable results:
```go
if err := s.port.ResetInputBuffer(); err != nil {
    return "", fmt.Errorf("reset buffer: %w", err)
}
```

### 4b: Version read in `ReadAllParamsAndVersion` (`at.go:100`)

**Current:** `verResp, _ := conn.sendAndRead("AT+VER\r\n")`
**New:** Handle error gracefully -- params succeed even if version fails:
```go
verResp, verErr := s.sendAndRead("AT+VER\r\n")
version := ""
if verErr == nil {
    version = parseVersion(verResp)
}
```

### 4c: `ResetInputBuffer` in `enterAT` retry loop (`at.go:25`)

**Current:** `_ = conn.port.ResetInputBuffer()`
**New:** Return error as retry failure:
```go
if err := conn.port.ResetInputBuffer(); err != nil {
    lastErr = fmt.Errorf("enter AT mode: reset buffer: %w", err)
    continue
}
```

## Test Impact

- **Existing tests:** All pass without modification after file split (same package, same functions)
- **New tests needed for `withATSession`:** Test that exit errors are surfaced, test callback error takes priority over exit error
- **Updated tests for `sendAndRead`:** Tests that expect partial-data-without-error must be updated to expect the new error
- **Mock port tests:** May need adjustment for `ResetInputBuffer` error paths

## Files Modified

| File | Change Type |
|------|------------|
| `internal/tui/model.go` | Split into 6 files |
| `internal/tui/navigation.go` | New (extracted from model.go) |
| `internal/tui/keys.go` | New (extracted from model.go) |
| `internal/tui/commands.go` | New (extracted from model.go) |
| `internal/tui/view.go` | New (extracted from model.go) |
| `internal/tui/ansi.go` | New (extracted from model.go) |
| `internal/device/serial.go` | Fix sendAndRead: partial timeout error, yield, ResetInputBuffer error |
| `internal/device/at.go` | Extract withATSession, fix ResetInputBuffer and version read errors |
| `internal/device/serial_test.go` | Update tests for new error behaviors |
| `internal/device/at_test.go` | Add withATSession tests, update for error changes |

## Out of Scope

- View() caching / dirty-region tracking
- ALLP parsing optimization
- New `protocol` package / transport abstraction
- Device discovery / enumeration
- Any timing constant changes
