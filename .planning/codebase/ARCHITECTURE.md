# Architecture

**Analysis Date:** 2026-04-19

## Pattern Overview

**Overall:** Layered desktop TUI application with clear separation of concerns.

**Key Characteristics:**
- **TUI Layer**: BubbleTea-based state machine with lipgloss styling
- **Device Layer**: Serial protocol abstraction over AT command interface
- **Session-based Communication**: Atomic transactions with mutex-protected state
- **Unidirectional Data Flow**: Events bubble up from device layer to TUI for updates

## Layers

**Application (TUI) Layer:**
- Purpose: User interface state machine and event handling
- Location: `cmd/lora-config-sx1262/main.go`, `internal/tui/`
- Contains: Model state, views, key handling, field management
- Depends on: Device layer for serial communication
- Used by: Main entry point

**Device Communication Layer:**
- Purpose: Abstract serial port protocol and AT command handling
- Location: `internal/device/`
- Contains: AT protocol implementation, serial connection management, command parsing
- Depends on: `go.bug.st/serial` for low-level port I/O
- Used by: TUI layer for device interaction

**UI Components Layer:**
- Purpose: Reusable field rendering and parameter management
- Location: `internal/tui/params.go`, `internal/tui/field.go`
- Contains: Field definitions, parameter validation, dropdown/input rendering
- Depends on: lipgloss for styling
- Used by: Model for rendering and state

## Data Flow

**Connection Workflow:**

1. User enters device path and presses Connect
2. TUI calls `device.OpenSerial(path, baud)` asynchronously via command
3. Device layer opens serial port and creates `SerialConn` wrapper
4. TUI calls `device.ReadAllParamsAndVersion(conn)` in single AT session
5. Device sends `+++ → AT+ALLP? → AT+VER → AT+EXIT`
6. Response parsed into map of parameters and version string
7. TUI receives `connectResultMsg` with populated fields
8. Fields transition from disabled to enabled

**Parameter Update Workflow:**

1. User focuses field and presses Enter
2. For dropdown: TUI toggles `Open` state and allows navigation
3. For numeric input: TUI sets `Editing = true` and focuses text input
4. User confirms with Enter → field value captured
5. If value changed, TUI calls `device.SetParam(conn, atCmd, value)` asynchronously
6. Device layer executes atomic session: `+++ → AT+CMD=value → AT+EXIT`
7. Response checked for "OK"
8. TUI receives `paramResultMsg` with success/error status
9. Field border changes color (green=success, red=error) for visual feedback

**State Management:**

- **Focus State**: `focusIndex` tracks which element has keyboard focus (-1=device input, -2=connect, -3=restore, -4=reboot, 0..N=field index)
- **Field State**: Each field tracks `Selected` (dropdown index), `Value` (for numeric), `Status` (visual feedback), `Disabled` (availability)
- **Connection State**: `connected` boolean, `conn` pointer for active session
- **Layout State**: `leftCol` and `rightCol` arrays map field indices to column positions for navigation

## Key Abstractions

**Port Interface:**
- Purpose: Abstract serial port implementation for testability
- Examples: `internal/device/port.go`, `internal/device/mock/mock_port.go`
- Pattern: Interface with Read/Write/Close/ResetInputBuffer/SetReadTimeout methods

**SerialConn:**
- Purpose: Wraps Port with session-level locking
- Examples: `internal/device/serial.go`
- Pattern: Mutex-protected session transactions ensure commands complete atomically before next operation

**Field:**
- Purpose: Represents a single device parameter with UI state
- Examples: `internal/tui/field.go`
- Pattern: Handles both dropdown fields (Options array) and numeric input fields (Min/Max range)

**ParamDef:**
- Purpose: Static definition of device parameters and their UI representation
- Examples: `internal/tui/params.go` (allParams array)
- Pattern: Maps device parameters to UI fields with ALLP response ordering

**BubbleTea Model/Update/View:**
- Purpose: Event-driven TUI state machine
- Examples: `internal/tui/model.go`
- Pattern: Init → Update (handles messages) → View (renders) loop

## Entry Points

**Application Entry:**
- Location: `cmd/lora-config-sx1262/main.go`
- Triggers: Application startup
- Responsibilities: Creates BubbleTea program with initial model, runs TUI loop

**Initial Model:**
- Location: `internal/tui/model.go` - `InitialModel()` function
- Triggers: Application start
- Responsibilities: Initializes all fields, buttons, focus state; sets up deviceInput text field

**Key Message Handlers:**
- `Update()`: Main state machine processing all incoming messages (WindowSize, Connect, Param updates, Key presses)
- `handleKey()`: Keyboard input routing (tab navigation, enter confirmation, vim keys for dropdown)
- `handleEnter()`: Submit action routing (connect/disconnect, parameter edit, restore/reboot)

## Error Handling

**Strategy:** Synchronous error propagation with visual feedback.

**Patterns:**
- AT Command Errors: Response checked for "OK" or "ERROR" markers; on error, field border turns red
- Serial Port Errors: Connection failures show error in status bar, fields remain disabled
- Validation Errors: Numeric input range validation shows error message when value out of bounds
- Timeout Handling: 3-second AT command timeout with 3 retry attempts for entering AT mode
- Graceful Degradation: If version read fails during initial connection, parameters still populate (version omitted)

## Cross-Cutting Concerns

**Logging:** None - status messages displayed in TUI status bar at bottom

**Validation:**
- Numeric input: Range validation (Min/Max) performed before submission
- Dropdown: Selected index validated against Options array bounds
- AT responses: Checked for "OK" marker or specific response format (e.g., "+ALLP=")

**Authentication:** Not applicable - direct serial port communication with local device

**Session Management:** Mutex-protected window ensures only one AT session active at a time; sequence always `+++ → command → AT+EXIT`

**Timing:**
- Pre-enter delay: 300ms guard time before sending `+++`
- Post-exit delay: 1s wait after AT+EXIT before device returns to normal mode
- Inter-command delay: 100ms between commands within session
- Read timeout: 3 seconds per AT command with deadline-based read loop

---

*Architecture analysis: 2026-04-19*
