# Testing

> Codebase testing patterns, frameworks, and coverage for lora-config-SX1262.

## Test Framework

- **Language:** Go (standard `testing` package)
- **Assertion library:** `github.com/stretchr/testify` (`assert`, `require`, `mock`)
- **Mock framework:** `github.com/stretchr/testify/mock` via custom mock in `internal/device/mock/`
- **No external test runners** - uses `go test ./...`

## Test File Locations

| Test File | Package | Lines | What It Tests |
|-----------|---------|-------|---------------|
| `internal/device/at_test.go` | `device` | 158 | ALLP parsing, version parsing, splitALLP |
| `internal/device/serial_test.go` | `device` | 197 | enterAT/exitAT, SetParam, ReadAllParamsAndVersion, Restore, Reboot, Close |
| `internal/tui/field_test.go` | `tui` | 319 | Field creation, selection, validation, navigation, dropdown rendering, view states |
| `internal/tui/model_test.go` | `tui` | 773 | Focus navigation, key handling, message updates, view rendering, string helpers |
| `internal/tui/params_test.go` | `tui` | 66 | rangeOptions, allParams consistency (no duplicate ATCmd/AllpIndex) |
| `internal/device/mock/mock_port_test.go` | `mock` | 130 | Mock Port correctness (Read, Write, Close, ResetInputBuffer, SetReadTimeout, ForResponses) |

## Test Patterns

### Table-Driven Tests
Used extensively throughout. Pattern:
```go
tests := []struct {
    name    string
    input   string
    want    map[string]string
    wantErr bool
}{...}
for _, testCase := range tests {
    t.Run(testCase.name, func(t *testing.T) { ... })
}
```
Variable name: `testCase` (not `tc` or `tt`).

### Mock Port Pattern
`internal/device/mock/mock_port.go` provides a testify mock implementing `device.Port`:

**Quick setup with `ForResponses`:**
```go
port := devicemock.ForResponses("+++\r\nOK")  // queues sequential Read responses
conn := NewSerialConn(port)
```

**Fine-grained control:**
```go
port := new(devicemock.Port)
port.On("ResetInputBuffer").Return(nil)
port.On("Write", tmock.Anything).Return(0, nil)
port.On("Read", tmock.Anything).Return([]byte("+++"), nil).Once()
```

### TUI Testing Pattern
Tests use helper constructors:
- `testModel()` - disconnected model with default state
- `connectedModel()` - connected model with all fields enabled

Tests exercise the BubbleTea `Update()` method directly with typed messages:
```go
newM, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
got := newM.(model)
assert.True(t, got.connecting)
```

### Assertion Style
- `require.NoError`/`require.Error` for fatal checks (test stops on failure)
- `assert.Equal`/`assert.Contains` for non-fatal checks
- `assert.NotPanics` for safety checks
- `port.AssertExpectations(t)` to verify mock call counts

## CI Integration

`.github/workflows/build.yml` runs pre-commit checks on PRs to `master`:
- **Pre-commit hooks** via `pre-commit/action@v3.0.1`
- **golangci-lint** v2.11.4
- No explicit `go test` step in CI (tests run via pre-commit hooks)
- Build matrix: linux/amd64, darwin/amd64, darwin/arm64, windows/amd64

## Coverage Gaps

### Not Tested
- **No integration tests** with real serial devices (all tests use mock port)
- **No TUI end-to-end rendering tests** (View() output checked for content, not pixel-perfect layout)
- **No serial timeout behavior tests** (`sendAndRead` deadline path untested)
- **No concurrent access tests** (sessionMu locking correctness not verified under contention)
- **No ANSI escape handling edge cases** beyond basic `stripAnsi`/`truncateToWidth`/`skipWidth`

### Partially Tested
- Error paths in `exitAT` called within `SetParam`/`ReadAllParamsAndVersion` (error is silently ignored with `_ = exitAT(conn)`)
- `sendAndRead` partial response on timeout (returns data without error)
- Dropdown overlay positioning in `View()` (tested for non-empty, not for correctness)

## Running Tests

```bash
# All tests
go test ./...

# Verbose with coverage
go test -v -cover ./...

# Specific package
go test -v ./internal/device/...
go test -v ./internal/tui/...

# Run specific test
go test -v -run TestParseALLP ./internal/device/
```
