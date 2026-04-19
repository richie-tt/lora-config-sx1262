# Codebase Structure

**Analysis Date:** 2026-04-19

## Directory Layout

```
lora/
├── cmd/
│   └── lora-config-sx1262/     # Application entry point
│       └── main.go
├── internal/
│   ├── device/                 # Serial protocol and device communication
│   │   ├── at.go               # AT command protocol implementation
│   │   ├── at_test.go
│   │   ├── port.go             # Port interface abstraction
│   │   ├── serial.go           # SerialConn wrapper with locking
│   │   ├── serial_test.go
│   │   └── mock/
│   │       ├── mock_port.go    # Mock implementation for testing
│   │       └── mock_port_test.go
│   └── tui/                    # Terminal UI and state management
│       ├── model.go            # Main TUI state machine (897 lines)
│       ├── model_test.go
│       ├── field.go            # Field UI component
│       ├── field_test.go
│       ├── params.go           # Parameter definitions
│       └── params_test.go
├── assets/                     # Images and demo resources
│   ├── lora.png
│   └── demo.gif
├── .github/                    # GitHub workflows
├── .claude/                    # Claude agent configuration
├── .planning/
│   └── codebase/              # Generated codebase analysis (this directory)
├── go.mod                      # Module definition
├── go.sum                      # Dependency checksums
├── Makefile                    # Build automation
├── .pre-commit-config.yaml     # Pre-commit hooks
├── .golangci.yml               # Linter configuration
├── README.md                   # Project documentation
└── LICENSE                     # MIT License
```

## Directory Purposes

**cmd/lora-config-sx1262/:**
- Purpose: Application entry point - initializes BubbleTea TUI framework
- Contains: main function with hardcoded version info
- Key files: `main.go`

**internal/device/:**
- Purpose: Serial port communication layer - handles AT protocol
- Contains: Connection management, AT command parsing, session locking
- Key files: `serial.go` (SerialConn), `at.go` (protocol), `port.go` (interface)

**internal/device/mock/:**
- Purpose: Test doubles for serial port testing
- Contains: Mock port implementation satisfying Port interface
- Key files: `mock_port.go` (mock), `mock_port_test.go` (tests)

**internal/tui/:**
- Purpose: Terminal user interface - BubbleTea state machine and rendering
- Contains: Model state, field management, view rendering, event handlers
- Key files: `model.go` (main state machine, 897 lines), `field.go` (UI component), `params.go` (field definitions)

**assets/:**
- Purpose: Documentation and demo resources
- Contains: Project logo and animated demo GIF
- Key files: `lora.png`, `demo.gif`

## Key File Locations

**Entry Points:**
- `cmd/lora-config-sx1262/main.go`: Application startup - creates BubbleTea program, imports build info (Tag, Commit, BuildDate)

**Configuration:**
- `go.mod`: Module declaration and dependency versions
- `.golangci.yml`: Linter rules and settings
- `Makefile`: Build targets (implied by file existence)
- `.pre-commit-config.yaml`: Pre-commit hooks for code quality

**Core Logic:**

**Device Communication:**
- `internal/device/serial.go`: SerialConn struct, OpenSerial factory, Close, session locking
- `internal/device/at.go`: AT command protocol (enterAT, exitAT, SetParam, ReadAllParamsAndVersion, Restore, Reboot, parsing)
- `internal/device/port.go`: Port interface definition (Read, Write, Close, ResetInputBuffer, SetReadTimeout)

**TUI State & Rendering:**
- `internal/tui/model.go`: Main model struct, Init, Update (event handler), View (rendering), navigation, key handling, async commands
- `internal/tui/field.go`: Field component struct, rendering (ViewClosed, RenderDropdown), state (ToggleOpen, MoveUp, MoveDown)
- `internal/tui/params.go`: ParamDef array (allParams), field definitions, Option struct

**Testing:**
- `internal/device/at_test.go`: Parser tests for ALLP response format
- `internal/device/serial_test.go`: SerialConn send/read and session locking tests
- `internal/device/mock/mock_port_test.go`: Mock port behavior tests
- `internal/tui/model_test.go`: State machine, navigation, key handling tests (772 lines)
- `internal/tui/field_test.go`: Field rendering, dropdown behavior tests

## Naming Conventions

**Files:**
- `<subsystem>.go`: Core implementation (e.g., `serial.go`, `model.go`)
- `<subsystem>_test.go`: Test suite for subsystem
- `mock_<subsystem>.go`: Mock/test double implementation

**Packages:**
- `cmd/lora-config-sx1262`: Application package
- `internal/device`: Device communication package
- `internal/device/mock`: Mock port implementations
- `internal/tui`: Terminal UI package

**Functions:**
- `Init()`, `Update()`, `View()`: BubbleTea interface methods (PascalCase)
- `init<Action>()`, `handle<Input>()`: Helper methods (camelCase)
- `OpenSerial()`, `SetParam()`, `ReadAllParamsAndVersion()`: Public API (PascalCase)
- `enterAT()`, `exitAT()`, `sendAndRead()`: Internal protocol functions (camelCase)

**Types:**
- `SerialConn`, `Field`, `Model`, `ParamDef`, `Option`: Public structs (PascalCase)
- `connectResultMsg`, `paramResultMsg`, `disconnectMsg`: Message types (camelCase with Msg suffix)
- `focusDevice`, `focusConnect`, `focusRestore`: Focus constants (lowerCamelCase)

**Variables:**
- `m model`: Model receiver variable (short)
- `m.fields`, `m.focusIndex`, `m.connected`: State properties (lowerCamelCase)
- `conn *device.SerialConn`: Connection variable (short)
- Module-level: `allParams []ParamDef`, `atTimeout time.Duration` (lowerCamelCase/UPPER_SNAKE)

## Where to Add New Code

**New Feature (e.g., add new device parameter):**
- Parameter definition: Add entry to `internal/tui/params.go` - allParams array with new ParamDef
- AT protocol support: If command differs, add to `internal/device/at.go`
- UI testing: Add test case to `internal/tui/model_test.go`
- Integration test: Add to `internal/device/serial_test.go` if requires device protocol change

**New Component/Module:**
- Device feature: Create new file in `internal/device/` (e.g., `internal/device/firmware.go`)
- TUI view: Create new file in `internal/tui/` (e.g., `internal/tui/help.go`)
- Application command: Create new file in `cmd/lora-config-sx1262/` (unlikely for this TUI-only app)

**Utilities:**
- Shared parsing helpers: `internal/device/at.go` (for AT response parsing)
- Shared UI helpers: `internal/tui/field.go` or new `internal/tui/utils.go`
- String utilities: Top of file (e.g., ANSI string manipulation in `model.go`)

**Tests:**
- Unit tests: Co-located with source file as `*_test.go`
- Mock objects: `internal/device/mock/` directory
- Test fixtures: Inline in test functions or as test helper constants

## Special Directories

**internal/:**
- Purpose: Private packages not importable by external consumers
- Generated: No
- Committed: Yes - all source code

**internal/device/mock/:**
- Purpose: Isolate serial port tests from actual hardware
- Generated: No
- Committed: Yes

**assets/:**
- Purpose: Documentation resources (logo, demo)
- Generated: No
- Committed: Yes

**.planning/codebase/:**
- Purpose: Architecture and structure documentation
- Generated: Yes (by codebase mapper)
- Committed: Yes

**.github/:**
- Purpose: GitHub workflows (CI/CD, release builds)
- Generated: No
- Committed: Yes

**cmd/lora-config-sx1262/:**
- Purpose: Single executable entry point
- Generated: No
- Committed: Yes

## Import Organization

**Observed Import Order:**
1. Standard library (fmt, strings, time, sync, testing, etc.)
2. External packages (go.bug.st/serial, github.com/charmbracelet/*, github.com/stretchr/testify)
3. Internal packages (lora-config-SX1262/internal/*)

**Example from `internal/tui/model.go`:**
```go
import (
	"fmt"
	"lora-config-SX1262/internal/device"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
```

**No path aliases used** - imports reference package names directly.

---

*Structure analysis: 2026-04-19*
