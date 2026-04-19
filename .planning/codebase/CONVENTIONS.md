# Coding Conventions

**Analysis Date:** 2026-04-19

## Naming Patterns

**Files:**
- Test files: `*_test.go` (e.g., `at_test.go`, `serial_test.go`)
- Mock files: placed in `internal/{package}/mock/` directory
- Package structure: lowercase, single word packages (`device`, `tui`)
- Main executable: `cmd/{app-name}/main.go`

**Functions:**
- Exported: PascalCase (e.g., `SetParam`, `ReadAllParamsAndVersion`, `OpenSerial`)
- Private: camelCase (e.g., `enterAT`, `exitAT`, `parseALLP`, `splitALLP`)
- Test functions: `Test{Function}_{Scenario}` format
  - Example: `TestSetParam_Success`, `TestEnterAT_UnexpectedResponse`, `TestRangeOptions`
- Table-driven tests use `testCase` variable name consistently

**Variables:**
- Local: camelCase (e.g., `resp`, `params`, `version`, `conn`, `port`)
- Constants: SCREAMING_SNAKE_CASE for exported, camelCase for private
  - Private constants example: `atTimeout`, `postExitDelay`, `preEnterDelay`
- Field names in structs: PascalCase (exported) or accessed via methods
- Loop variables: short names allowed (`i`, `idx`, `attempt`)
- Error variables: conventionally named (e.g., `err`, `lastErr`, `errWrite`)

**Types:**
- Structs: PascalCase and exported (e.g., `SerialConn`, `Field`, `ParamDef`, `Option`)
- Interfaces: PascalCase, usually single method or describing behavior (e.g., `Port` interface)
- Type aliases for message types: camelCase with `Msg` suffix (e.g., `connectResultMsg`, `paramResultMsg`)

## Code Style

**Formatting:**
- Tool: `gofmt` (via `.pre-commit-config.yaml`)
- Additional formatter: `gofumpt` configured in `.golangci.yml`
- Standard Go formatting: 4-space tabs, line length not strictly enforced
- All Go files processed by pre-commit hooks automatically

**Linting:**
- Tool: `golangci-lint` via `.golangci.yml`
- Version lock: configured in `.pre-commit-config.yaml` at v0.5.1
- Key linters enabled: errcheck, staticcheck, ineffassign, varnamelen, gosec, testifylint, gocritic, gocyclo

**Style Enforcements:**
- Unused variables cause lint errors (linter: `unused`)
- Variable name length validated (linter: `varnamelen`)
- Inefficient assignments flagged (linter: `ineffassign`)
- All errors must be checked explicitly (linter: `errcheck`)

## Import Organization

**Order:**
1. Standard library imports (e.g., `fmt`, `strings`, `sync`, `time`, `testing`)
2. Third-party imports (e.g., `github.com/charmbracelet/...`, `go.bug.st/serial`)
3. Local package imports (e.g., `lora-config-SX1262/internal/device`)

**Path Aliases:**
- No path aliases used in observed code
- Standard relative imports from module root: `lora-config-SX1262/internal/{package}`

**Package imports pattern:**
```go
import (
    "fmt"
    "strings"
    "sync"
    "time"

    "github.com/stretchr/testify/assert"
    "go.bug.st/serial"

    "lora-config-SX1262/internal/device"
)
```

## Error Handling

**Patterns:**
- Explicit error returns: functions that may fail return `(T, error)` or just `error`
- Error wrapping: use `fmt.Errorf("%w", err)` for context
  - Example: `fmt.Errorf("enter AT mode: %w", err)`, `fmt.Errorf("set %s=%s: %w", atCmd, value, err)`
- Error checking: immediate, inline checks before proceeding
  - Pattern: `if err != nil { return ... }`
- Ignore intentionally: use `_` to explicitly ignore return values
  - Example: `_ = exitAT(conn)` when error doesn't stop cleanup
- Last error tracking: retry loops track `lastErr` and return final error
  - See: `enterAT()` function retry pattern

**Error messages:**
- Prefix with context (e.g., "enter AT mode:", "parse ALLP:")
- Include actual values in errors for debugging (e.g., `"got %d"`, `"response: %s"`)
- No error shadowing: each branch handles its error

## Logging

**Framework:** `fmt` package only (no structured logging library)

**Patterns:**
- Errors sent to stderr: `fmt.Fprintf(os.Stderr, ...)`
- Example from main: `fmt.Fprintf(os.Stderr, "Error: %v\n", err)`
- No info/debug/warn levels observed - errors only
- Console output formatted for human readability

## Comments

**When to Comment:**
- Function-level comment for exported functions (Go convention)
- Complex parsing/algorithm explanation: inline comments
- AT protocol specifics: comments explain device behavior
- Locking semantics: explicitly documented (e.g., "Caller must hold session lock")

**Documentation Comments:**
- Start with function name and short description
- Example: `// enterAT sends +++ and expects echo, with retries and guard time.`
- Multi-paragraph comments for complex operations
- No JSDoc/TSDoc style (Go convention: plain English sentences)

**Internal Comments:**
- Explain "why" not "what" the code does
- Device protocol notes: `// Order: SF,BW,CR,PWR,NETID,LBT,MODE,TXCH,RXCH,RSSI,ADDR,PORT,COMM,BAUD,KEY`
- Sync annotations: `// Caller must hold session lock.`
- Status fields: `// -1=device, -2=connect, -3=restore, -4=reboot, 0..N=field index`

## Function Design

**Size:**
- Most functions 20-100 lines
- Complex functions documented with comments
- Focus on single responsibility

**Parameters:**
- Receiver pattern for methods: `func (s *SerialConn) MethodName()`
- Interface pointers for flexibility: functions take `Port` interface not concrete type
- Multiple return values for error handling
- Explicit locking parameters: functions receiving locked objects note in comment

**Return Values:**
- Error always last: `(result, error)` or just `error`
- Multiple success values: `(map, string, error)` for data + version + error
- Nil on error: return nil for complex types on error
- Use blank identifier `_` for intentional ignores

## Module Design

**Exports:**
- Only required public types exported (PascalCase)
- Private helpers in same package (camelCase)
- No package-level state except constants

**Barrel Files:**
- Not used in this codebase
- Each file contains focused functionality

**Package Organization:**
- `internal/device/`: Serial communication and AT protocol
- `internal/tui/`: Terminal UI models and rendering
- `cmd/lora-config-sx1262/`: Entry point only

**Struct Embedding:**
- Used minimally - `model` struct contains fields directly
- No behavior via embedding

---

*Convention analysis: 2026-04-19*
