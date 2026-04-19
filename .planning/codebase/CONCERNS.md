# Concerns

> Technical debt, known issues, security considerations, and fragile areas in lora-config-SX1262.

## Technical Debt

### Ignored Errors

| Location | Issue | Risk |
|----------|-------|------|
| `internal/device/at.go:25-26` | `_ = conn.port.ResetInputBuffer()` in enterAT retry loop | Silent failure could leave stale data in buffer |
| `internal/device/at.go:67-68` | `_ = exitAT(conn)` in SetParam after command failure | Device may remain in AT mode if exit also fails |
| `internal/device/at.go:71` | `_ = exitAT(conn)` in SetParam after non-OK response | Same - device stuck in AT mode |
| `internal/device/at.go:89-90` | `_ = exitAT(conn)` in ReadAllParamsAndVersion after ALLP write error | Same pattern |
| `internal/device/at.go:95` | `_ = exitAT(conn)` after ALLP parse error | Same pattern |
| `internal/device/at.go:100` | `verResp, _ := conn.sendAndRead("AT+VER\r\n")` | Version read error silently ignored |
| `internal/device/serial.go:57` | `_ = s.port.ResetInputBuffer()` in sendAndRead | Silent failure before command write |

### Hardcoded Values

| Location | Value | Issue |
|----------|-------|-------|
| `internal/tui/model.go:564` | `115200` baud rate | Hardcoded in `connectCmd()`, not configurable by user |
| `internal/device/serial.go:33` | `100ms` read timeout | Hardcoded in `OpenSerial`, affects all reads |
| `internal/device/at.go:10-14` | `atTimeout=3s`, `postExitDelay=1s`, `preEnterDelay=300ms`, `interCmdDelay=100ms` | Constants not tunable for different device behaviors |
| `internal/tui/field.go:50` | `maxVisible: 8` | Dropdown max visible items hardcoded |
| `internal/tui/model.go:779` | `strings.Repeat("─", 70)` | Fixed-width separator, doesn't adapt to terminal width |

## Known Bugs / Edge Cases

### Timeout Returns Partial Data Without Error
**File:** `internal/device/serial.go:81-87`

When `sendAndRead` times out but has received partial data, it returns the partial response *without* an error. Callers may interpret incomplete data as valid responses.

```go
// Line 84-87: returns partial data as success
resp := buf.String()
if resp == "" {
    return "", fmt.Errorf("timeout: no response")
}
return strings.TrimSpace(resp), nil  // partial data, no error!
```

### Numeric Input Leading Zeros
**File:** `internal/tui/field.go:101-114`

`ValidateNumInput` normalizes leading zeros via `fmt.Sprintf("%d", num)`, but the raw text input accepts them. Input "007" validates to "7" which is correct, but there's no input-level filtering to prevent entry of non-numeric characters (filtering depends on BubbleTea textinput, which allows any character).

### Dropdown Scroll Offset Edge Case
**File:** `internal/tui/field.go:137-147`

`ToggleOpen` centers scroll on selected item (`Selected - maxVisible/2`), but doesn't account for the case where `maxVisible` exceeds `len(Options)`. The `maxOff < 0` guard at line 143 handles this, but the intermediate negative `scrollOffset` at line 138 briefly exists.

### Field Capture by Value in setParamCmd
**File:** `internal/tui/model.go:584-588`

`setParamCmd` captures `field` by value (line 585: `field := m.fields[fieldIdx]`), which means the closure sees a snapshot. If the field is modified between dispatch and execution, the AT command uses stale data. This is intentional but subtle.

## Security Considerations

### No Device Path Validation
**File:** `internal/device/serial.go:22`

`OpenSerial` passes the user-provided device path directly to `serial.Open()` without sanitization. The path comes from TUI text input (`internal/tui/model.go:562`). Since this is a local CLI tool, the risk is low, but path traversal or special device files could theoretically be opened.

### Plain Text Serial Communication
All AT commands and parameters are sent as plain text over serial (`internal/device/serial.go:59`). Parameters include network IDs, addresses, and encryption keys. No encryption or authentication on the serial link itself.

### No Input Sanitization for AT Commands
**File:** `internal/device/at.go:64`

`SetParam` formats user values directly into AT commands:
```go
cmd := fmt.Sprintf("AT+%s=%s\r\n", atCmd, value)
```
The `atCmd` comes from the hardcoded `allParams` table (safe), but `value` comes from user selection or numeric input. Dropdown values are constrained, but numeric input could theoretically inject AT command fragments if validation is bypassed.

## Performance Bottlenecks

### Sequential Pre-Enter Delay in Retry Loop
**File:** `internal/device/at.go:21-23`

Each retry in `enterAT` sleeps `preEnterDelay` (300ms) before and after `ResetInputBuffer`. With 3 retries, worst case is ~1.8s of sleeping plus 3x `atTimeout` (3s each) = ~10.8s before failure.

### Full TUI Grid Re-render Every Frame
**File:** `internal/tui/model.go:747-897`

`View()` rebuilds the entire string output on every call. It iterates all fields, joins columns, overlays dropdowns, and constructs the full view. No caching or dirty-region tracking. For a simple TUI this is fine, but dropdown overlay logic (lines 815-846) involves string manipulation per frame.

### Synchronous Serial Reads with Blocking Timeouts
**File:** `internal/device/serial.go:68-81`

`sendAndRead` polls in a tight loop (`for time.Now().Before(deadline)`) with 100ms read timeout. Each `Read` call blocks for up to 100ms. This is standard for serial communication but means the goroutine is blocked during device operations.

## Fragile Areas

### ALLP Parser Field Count and Order
**File:** `internal/device/at.go:126-151`

`parseALLP` depends on exactly 15 fields in a specific order. The key list is hardcoded:
```go
keys := []string{"SF", "BW", "CR", "PWR", "NETID", "LBT", "MODE", "TXCH", "RXCH", "RSSI", "ADDR", "PORT", "COMM", "BAUD", "KEY"}
```
If firmware adds/removes/reorders fields, parsing silently produces wrong mappings. No version-based field detection.

### Field Index Coupling Between AllpIndex and Params Map
**Files:** `internal/tui/params.go` (AllpIndex values), `internal/device/at.go:140` (key order)

The `AllpIndex` field in `ParamDef` must match the positional index in the ALLP response. This coupling is validated by `TestAllParamsConsistency` in `params_test.go`, but only checks for duplicates, not that indices match the AT protocol spec.

### Dropdown Overlay Positioning
**File:** `internal/tui/model.go:815-846`

The dropdown overlay calculation uses magic numbers:
- `colOffset := 15` (left column label width)
- `startGridLine := openRow*3 + 3` (assumes 3 lines per field row)

If field rendering changes height or label width changes, dropdown positioning breaks silently.

## Missing Features

- **No device discovery/enumeration** - user must know the device path
- **No settings persistence** - device path and preferences lost on exit
- **No batch operations** - parameters set one at a time with full AT session cycle
- **No undo/revert** - no way to revert a parameter change except manual re-entry or factory restore
- **No connection health monitoring** - no periodic heartbeat or reconnection logic
- **No logging** - no debug/trace logging for serial communication troubleshooting

## Test Coverage Gaps

- No integration tests with actual serial devices
- No TUI end-to-end tests (visual regression)
- No serial error cascade tests (multiple failures in sequence)
- No concurrent access tests for `sessionMu`
- ANSI rendering not tested across terminal types
- No fuzz testing for `parseALLP` or `splitALLP`
