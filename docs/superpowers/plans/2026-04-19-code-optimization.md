# Code Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve code quality and runtime efficiency of lora-config-SX1262 while preserving all serial timing constants.

**Architecture:** Split the monolithic `model.go` into 6 focused files by concern. Extract a `withATSession` helper to eliminate duplicated AT session boilerplate and ignored errors. Fix `sendAndRead` to report partial timeouts and yield on empty reads.

**Tech Stack:** Go 1.26, BubbleTea, lipgloss, testify

---

## File Structure

### Files to create (extracted from `internal/tui/model.go`):
- `internal/tui/navigation.go` — Focus navigation helpers
- `internal/tui/keys.go` — Key event handling
- `internal/tui/commands.go` — BubbleTea async commands
- `internal/tui/view.go` — View rendering and styles
- `internal/tui/ansi.go` — ANSI string manipulation utilities

### Files to modify:
- `internal/tui/model.go` — Reduce to model struct, types, messages, InitialModel, Init, Update
- `internal/device/serial.go` — Fix sendAndRead error handling and polling
- `internal/device/at.go` — Extract withATSession, fix error handling
- `internal/device/serial_test.go` — Update tests for new error behaviors
- `internal/device/at_test.go` — Add withATSession tests

---

### Task 1: Extract ANSI utilities to `ansi.go`

**Files:**
- Create: `internal/tui/ansi.go`
- Modify: `internal/tui/model.go` (remove lines 611-700)

- [ ] **Step 1: Create `internal/tui/ansi.go`**

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// overlayString places overlay on top of base at the given column offset.
func overlayString(base, overlay string, col int) string {
	baseRunes := []rune(stripAnsi(base))
	baseWidth := len(baseRunes)

	if baseWidth < col {
		base += strings.Repeat(" ", col-baseWidth)
	}

	lines := base
	overlayWidth := lipgloss.Width(overlay)

	baseBefore := truncateToWidth(lines, col)
	baseAfter := skipWidth(lines, col+overlayWidth)

	return baseBefore + overlay + baseAfter
}

func truncateToWidth(str string, width int) string {
	if width <= 0 {
		return ""
	}
	var result strings.Builder
	visWidth := 0
	inEsc := false
	for _, char := range str {
		if char == '\x1b' {
			inEsc = true
			result.WriteRune(char)
			continue
		}
		if inEsc {
			result.WriteRune(char)
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
				inEsc = false
			}
			continue
		}
		if visWidth >= width {
			break
		}
		result.WriteRune(char)
		visWidth++
	}
	return result.String()
}

func skipWidth(str string, width int) string {
	if width <= 0 {
		return str
	}
	visWidth := 0
	inEsc := false
	for idx, char := range str {
		if char == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
				inEsc = false
			}
			continue
		}
		if visWidth >= width {
			return str[idx:]
		}
		visWidth++
	}
	return ""
}

func stripAnsi(str string) string {
	var result strings.Builder
	inEsc := false
	for _, char := range str {
		if char == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
				inEsc = false
			}
			continue
		}
		result.WriteRune(char)
	}
	return result.String()
}
```

- [ ] **Step 2: Remove ANSI functions from `model.go`**

Delete `overlayString`, `truncateToWidth`, `skipWidth`, and `stripAnsi` functions from `internal/tui/model.go` (lines 611-700). Also remove the `lipgloss` import if it is no longer used in `model.go` after all extractions are complete (it will still be needed until Task 5 extracts the view).

- [ ] **Step 3: Run tests to verify nothing broke**

Run: `go test -v -count=1 ./internal/tui/...`
Expected: All tests pass. These functions are package-private and tests are in the same package.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/ansi.go internal/tui/model.go
git commit -m "refactor(tui): extract ANSI string utilities to ansi.go"
```

---

### Task 2: Extract navigation helpers to `navigation.go`

**Files:**
- Create: `internal/tui/navigation.go`
- Modify: `internal/tui/model.go` (remove lines 384-557)

- [ ] **Step 1: Create `internal/tui/navigation.go`**

```go
package tui

// updateFocus syncs the focused state across the device input and all fields.
func (m *model) updateFocus() {
	m.deviceInput.Blur()
	for i := range m.fields {
		m.fields[i].Focused = false
	}

	switch m.focusIndex {
	case focusDevice:
		m.deviceInput.Focus()
	default:
		if m.focusIndex >= 0 && m.focusIndex < len(m.fields) {
			m.fields[m.focusIndex].Focused = true
		}
	}
}

func (m *model) focusNext() {
	switch m.focusIndex {
	case focusDevice:
		m.focusIndex = focusConnect
	case focusConnect:
		if m.connected && len(m.fields) > 0 {
			m.focusIndex = 0
		} else {
			m.focusIndex = focusDevice
		}
	case focusRestore:
		m.focusIndex = focusReboot
	case focusReboot:
		if m.connected {
			m.focusIndex = focusConnect
		} else {
			m.focusIndex = focusDevice
		}
	default:
		m.focusIndex++
		if m.focusIndex >= len(m.fields) {
			m.focusIndex = focusRestore
		}
	}
}

func (m *model) focusPrev() {
	switch m.focusIndex {
	case focusDevice:
		m.focusIndex = focusReboot
	case focusConnect:
		if m.connected {
			m.focusIndex = focusReboot
		} else {
			m.focusIndex = focusDevice
		}
	case focusRestore:
		if m.connected && len(m.fields) > 0 {
			m.focusIndex = len(m.fields) - 1
		} else {
			m.focusIndex = focusConnect
		}
	case focusReboot:
		m.focusIndex = focusRestore
	default:
		m.focusIndex--
		if m.focusIndex < 0 {
			m.focusIndex = focusConnect
		}
	}
}

func (m *model) focusNextInColumn() {
	switch m.focusIndex {
	case focusRestore:
		m.focusIndex = focusReboot
		return
	case focusReboot:
		return
	case focusDevice, focusConnect:
		if m.connected && len(m.fields) > 0 {
			m.focusIndex = 0
		}
		return
	}
	col, row := m.fieldPosition(m.focusIndex)
	var colList []int
	if col == 0 {
		colList = m.leftCol
	} else {
		colList = m.rightCol
	}
	if row+1 < len(colList) {
		m.focusIndex = colList[row+1]
	} else {
		m.focusIndex = focusRestore
	}
}

func (m *model) focusPrevInColumn() {
	switch m.focusIndex {
	case focusRestore:
		if len(m.leftCol) > 0 {
			m.focusIndex = m.leftCol[len(m.leftCol)-1]
		}
		return
	case focusReboot:
		m.focusIndex = focusRestore
		return
	case focusDevice, focusConnect:
		return
	}
	col, row := m.fieldPosition(m.focusIndex)
	var colList []int
	if col == 0 {
		colList = m.leftCol
	} else {
		colList = m.rightCol
	}
	if row-1 >= 0 {
		m.focusIndex = colList[row-1]
	} else {
		m.focusIndex = focusConnect
	}
}

func (m *model) focusSwitchColumn() {
	if m.focusIndex < 0 {
		return
	}
	col, row := m.fieldPosition(m.focusIndex)
	var targetCol []int
	if col == 0 {
		targetCol = m.rightCol
	} else {
		targetCol = m.leftCol
	}
	if row < len(targetCol) {
		m.focusIndex = targetCol[row]
	} else if len(targetCol) > 0 {
		m.focusIndex = targetCol[len(targetCol)-1]
	}
}

func (m model) fieldPosition(idx int) (col, row int) {
	for r, i := range m.leftCol {
		if i == idx {
			return 0, r
		}
	}
	for r, i := range m.rightCol {
		if i == idx {
			return 1, r
		}
	}
	return 0, 0
}

func (m model) openDropdownIndex() int {
	for i, f := range m.fields {
		if f.Open {
			return i
		}
	}
	return -1
}

func (m model) editingFieldIndex() int {
	for i, f := range m.fields {
		if f.Editing {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Remove navigation functions from `model.go`**

Delete `updateFocus`, `focusNext`, `focusPrev`, `focusNextInColumn`, `focusPrevInColumn`, `focusSwitchColumn`, `fieldPosition`, `openDropdownIndex`, and `editingFieldIndex` from `internal/tui/model.go` (lines 384-557).

- [ ] **Step 3: Run tests**

Run: `go test -v -count=1 ./internal/tui/...`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/navigation.go internal/tui/model.go
git commit -m "refactor(tui): extract navigation helpers to navigation.go"
```

---

### Task 3: Extract key handling to `keys.go`

**Files:**
- Create: `internal/tui/keys.go`
- Modify: `internal/tui/model.go` (remove lines 201-380)

- [ ] **Step 1: Create `internal/tui/keys.go`**

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		if m.conn != nil {
			m.conn.Close()
		}
		return m, tea.Quit
	}

	// If a numeric field is being edited, handle it
	if idx := m.editingFieldIndex(); idx >= 0 {
		return m.handleNumInputKey(idx, msg)
	}

	// If a dropdown is open, handle it
	if idx := m.openDropdownIndex(); idx >= 0 {
		return m.handleDropdownKey(idx, key)
	}

	// q to quit only when not in any input mode
	if key == "q" && m.focusIndex != focusDevice {
		if m.conn != nil {
			m.conn.Close()
		}
		return m, tea.Quit
	}

	// Handle device text input
	if m.focusIndex == focusDevice {
		switch key {
		case "tab", "enter":
			m.focusIndex = focusConnect
			m.updateFocus()
			return m, nil
		case "shift+tab":
			m.focusIndex = focusReboot
			m.updateFocus()
			return m, nil
		default:
			var cmd tea.Cmd
			m.deviceInput, cmd = m.deviceInput.Update(msg)
			return m, cmd
		}
	}

	switch key {
	case "tab":
		m.focusNext()
		m.updateFocus()
	case "shift+tab":
		m.focusPrev()
		m.updateFocus()
	case "enter":
		return m.handleEnter()
	case "up":
		m.focusPrevInColumn()
		m.updateFocus()
	case "down":
		m.focusNextInColumn()
		m.updateFocus()
	case "left", "right":
		m.focusSwitchColumn()
		m.updateFocus()
	case "d":
		if m.connected {
			return m, func() tea.Msg { return disconnectMsg{} }
		}
	}

	return m, nil
}

func (m model) handleNumInputKey(fieldIdx int, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	field := &m.fields[fieldIdx]
	key := msg.String()

	switch key {
	case "enter":
		// Validate and submit
		val, ok := field.ValidateNumInput()
		if !ok {
			m.statusMsg = fmt.Sprintf("%s: must be %d-%d", field.Label, field.Min, field.Max)
			field.Status = StatusError
			field.NumInput.SetValue(field.LastValue) // revert
			field.Editing = false
			field.NumInput.Blur()
			return m, nil
		}
		field.NumInput.SetValue(val)
		field.Editing = false
		field.NumInput.Blur()
		if m.conn != nil && val != field.LastValue {
			field.LastValue = val
			m.statusMsg = fmt.Sprintf("Setting %s = %s...", field.ATCmd, val)
			return m, m.setParamCmd(fieldIdx)
		}
		field.LastValue = val
		return m, nil
	case "esc":
		// Cancel editing, revert
		field.NumInput.SetValue(field.LastValue)
		field.Editing = false
		field.NumInput.Blur()
		return m, nil
	default:
		// Forward to text input (only allow digits)
		var cmd tea.Cmd
		field.NumInput, cmd = field.NumInput.Update(msg)
		return m, cmd
	}
}

func (m model) handleDropdownKey(fieldIdx int, key string) (tea.Model, tea.Cmd) {
	field := &m.fields[fieldIdx]

	switch key {
	case "up", "k":
		field.MoveUp()
	case "down", "j":
		field.MoveDown()
	case "enter":
		prevValue := field.LastValue
		field.Open = false
		newValue := field.SelectedValue()
		field.LastValue = newValue
		// Only send AT command if value actually changed
		if m.conn != nil && newValue != prevValue {
			m.statusMsg = fmt.Sprintf("Setting %s = %s...", field.ATCmd, field.SelectedDisplay())
			return m, m.setParamCmd(fieldIdx)
		}
	case "esc":
		// Revert to previous selection
		field.SetByValue(field.LastValue)
		field.Open = false
	}
	return m, nil
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.focusIndex {
	case focusConnect:
		if m.connected {
			return m, func() tea.Msg { return disconnectMsg{} }
		}
		if m.connecting {
			return m, nil
		}
		m.connecting = true
		m.statusMsg = "Connecting..."
		return m, m.connectCmd()
	case focusRestore:
		if m.conn != nil {
			m.statusMsg = "Restoring..."
			return m, m.restoreCmd()
		}
	case focusReboot:
		if m.conn != nil {
			m.statusMsg = "Rebooting..."
			return m, m.rebootCmd()
		}
	default:
		if m.focusIndex >= 0 && m.focusIndex < len(m.fields) {
			field := &m.fields[m.focusIndex]
			if !field.Disabled {
				field.Status = StatusNormal
				if field.IsNumInput {
					// Start editing numeric field
					field.Editing = true
					field.LastValue = field.NumInput.Value()
					cmd := field.NumInput.Focus()
					m.statusMsg = fmt.Sprintf("Editing %s (%d-%d), Enter to confirm, Esc to cancel", field.Label, field.Min, field.Max)
					return m, cmd
				}
				field.ToggleOpen()
			}
		}
	}
	return m, nil
}
```

- [ ] **Step 2: Remove key handling functions from `model.go`**

Delete `handleKey`, `handleNumInputKey`, `handleDropdownKey`, and `handleEnter` from `internal/tui/model.go` (lines 201-380).

- [ ] **Step 3: Run tests**

Run: `go test -v -count=1 ./internal/tui/...`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/keys.go internal/tui/model.go
git commit -m "refactor(tui): extract key handling to keys.go"
```

---

### Task 4: Extract commands to `commands.go`

**Files:**
- Create: `internal/tui/commands.go`
- Modify: `internal/tui/model.go` (remove lines 559-609)

- [ ] **Step 1: Create `internal/tui/commands.go`**

```go
package tui

import (
	"lora-config-SX1262/internal/device"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) connectCmd() tea.Cmd {
	devicePath := m.deviceInput.Value()
	return func() tea.Msg {
		conn, err := device.OpenSerial(devicePath, 115200)
		if err != nil {
			return connectResultMsg{err: err}
		}

		// Single session: +++ → AT+ALLP? → AT+VER → AT+EXIT
		params, version, err := device.ReadAllParamsAndVersion(conn)
		if err != nil {
			conn.Close()
			return connectResultMsg{err: err}
		}

		return connectResultMsg{
			conn:    conn,
			params:  params,
			version: version,
		}
	}
}

func (m *model) setParamCmd(fieldIdx int) tea.Cmd {
	field := m.fields[fieldIdx]
	conn := m.conn
	return func() tea.Msg {
		err := device.SetParam(conn, field.ATCmd, field.SelectedValue())
		return paramResultMsg{
			fieldIndex: fieldIdx,
			ok:         err == nil,
			err:        err,
		}
	}
}

func (m model) restoreCmd() tea.Cmd {
	conn := m.conn
	return func() tea.Msg {
		return restoreResultMsg{err: device.Restore(conn)}
	}
}

func (m model) rebootCmd() tea.Cmd {
	conn := m.conn
	return func() tea.Msg {
		return rebootResultMsg{err: device.Reboot(conn)}
	}
}
```

- [ ] **Step 2: Remove command functions from `model.go`**

Delete `connectCmd`, `setParamCmd`, `restoreCmd`, and `rebootCmd` from `internal/tui/model.go` (lines 559-609). Remove the `"lora-config-SX1262/internal/device"` import from `model.go` if it is no longer used there (it will still be needed for the message types that reference `*device.SerialConn`).

- [ ] **Step 3: Run tests**

Run: `go test -v -count=1 ./internal/tui/...`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/commands.go internal/tui/model.go
git commit -m "refactor(tui): extract BubbleTea commands to commands.go"
```

---

### Task 5: Extract view and styles to `view.go`

**Files:**
- Create: `internal/tui/view.go`
- Modify: `internal/tui/model.go` (remove lines 702-898, style vars)

- [ ] **Step 1: Create `internal/tui/view.go`**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginBottom(1)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)

	buttonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 2)

	buttonFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 2)

	buttonConnectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("2")).
				Foreground(lipgloss.Color("2")).
				Padding(0, 2)

	buttonDisabledStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Foreground(lipgloss.Color("238")).
				Padding(0, 2)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
)

func (m model) View() string {
	var view strings.Builder

	// Title
	view.WriteString(titleStyle.Render("  LoRa Configurator for SX1262"))
	view.WriteString("\n\n")

	// Device row
	deviceLabel := labelStyle.Render("Device")
	deviceField := m.deviceInput.View()

	var connectBtn string
	switch {
	case m.connecting:
		connectBtn = buttonStyle.Render("Connecting...")
	case m.connected:
		if m.focusIndex == focusConnect {
			connectBtn = buttonFocusedStyle.Render("Disconnect")
		} else {
			connectBtn = buttonConnectedStyle.Render("Connected ✓")
		}
	case m.focusIndex == focusConnect:
		connectBtn = buttonFocusedStyle.Render("Connect")
	default:
		connectBtn = buttonStyle.Render("Connect")
	}

	deviceRow := lipgloss.JoinHorizontal(lipgloss.Center, deviceLabel, deviceField, "  ", connectBtn)
	view.WriteString(deviceRow)
	view.WriteString("\n")

	// Separator
	sep := separatorStyle.Render(strings.Repeat("─", 70))
	view.WriteString(sep)
	view.WriteString("\n\n")

	// Two-column parameter fields
	maxRows := len(m.leftCol)
	if len(m.rightCol) > maxRows {
		maxRows = len(m.rightCol)
	}

	colGap := "    "

	openRow := -1
	openCol := -1
	if idx := m.openDropdownIndex(); idx >= 0 {
		openCol, openRow = m.fieldPosition(idx)
	}

	var gridLines []string
	for row := range maxRows {
		var leftView, rightView string

		if row < len(m.leftCol) {
			leftView = m.fields[m.leftCol[row]].ViewClosed()
		} else {
			leftView = strings.Repeat(" ", 35)
		}

		if row < len(m.rightCol) {
			rightView = m.fields[m.rightCol[row]].ViewClosed()
		}

		rowStr := lipgloss.JoinHorizontal(lipgloss.Top, leftView, colGap, rightView)
		gridLines = append(gridLines, strings.Split(rowStr, "\n")...)
	}

	// Overlay dropdown if open
	if openRow >= 0 {
		var openField *Field
		if openCol == 0 && openRow < len(m.leftCol) {
			openField = &m.fields[m.leftCol[openRow]]
		} else if openCol == 1 && openRow < len(m.rightCol) {
			openField = &m.fields[m.rightCol[openRow]]
		}

		if openField != nil {
			dropdown := openField.RenderDropdown()
			dropdownLines := strings.Split(dropdown, "\n")

			colOffset := 15
			if openCol == 1 {
				leftWidth := lipgloss.Width(m.fields[m.leftCol[0]].ViewClosed())
				colOffset = leftWidth + len(colGap)
			}

			startGridLine := openRow*3 + 3

			for i, dLine := range dropdownLines {
				targetGridLine := startGridLine + i
				if targetGridLine < len(gridLines) {
					gridLines[targetGridLine] = overlayString(gridLines[targetGridLine], dLine, colOffset)
				} else {
					padded := strings.Repeat(" ", colOffset) + dLine
					gridLines = append(gridLines, padded)
				}
			}
		}
	}

	for _, gl := range gridLines {
		view.WriteString(gl)
		view.WriteString("\n")
	}

	view.WriteString("\n")

	// Bottom buttons row
	var restoreBtn, rebootBtn string

	switch {
	case !m.connected:
		restoreBtn = buttonDisabledStyle.Render("Restore")
		rebootBtn = buttonDisabledStyle.Render("Reboot")
	case m.focusIndex == focusRestore:
		restoreBtn = buttonFocusedStyle.Render("Restore")
		rebootBtn = buttonStyle.Render("Reboot")
	case m.focusIndex == focusReboot:
		restoreBtn = buttonStyle.Render("Restore")
		rebootBtn = buttonFocusedStyle.Render("Reboot")
	default:
		restoreBtn = buttonStyle.Render("Restore")
		rebootBtn = buttonStyle.Render("Reboot")
	}

	versionText := ""
	if m.version != "" {
		versionText = versionStyle.Render(fmt.Sprintf("  Firmware: %s", m.version))
	}

	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Center,
		"  ", restoreBtn, "  ", rebootBtn, versionText,
	)
	view.WriteString(buttonsRow)
	view.WriteString("\n")

	// Status bar
	view.WriteString(statusBarStyle.Render(fmt.Sprintf("  %s", m.statusMsg)))
	view.WriteString("\n")

	// Help
	help := "Tab/Shift+Tab: navigate • Enter: select • ↑↓: move • ←→: switch column • Click: select • q: quit"
	view.WriteString(helpStyle.Render("  " + help))
	view.WriteString("\n\n")

	buildInfo := fmt.Sprintf("  TAG: %s @%s (%s)", m.tag, m.commit, m.buildDate)
	view.WriteString(helpStyle.Render(buildInfo))

	return view.String()
}
```

- [ ] **Step 2: Remove View() and style vars from `model.go`**

Delete all style variable declarations (`titleStyle` through `helpStyle`) and the `View()` method from `internal/tui/model.go` (lines 702-898). Clean up unused imports in `model.go` — remove `"strings"`, `"github.com/charmbracelet/lipgloss"`, and `"fmt"` if they are no longer used. Keep the `"fmt"` import if `Update` still uses `fmt.Sprintf` for status messages.

- [ ] **Step 3: Run tests**

Run: `go test -v -count=1 ./internal/tui/...`
Expected: All tests pass.

- [ ] **Step 4: Verify `model.go` is clean**

Run: `wc -l internal/tui/model.go`
Expected: Approximately 100-200 lines containing only the model struct, message types, constants, `InitialModel`, `Init`, and `Update`.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/view.go internal/tui/model.go
git commit -m "refactor(tui): extract View and styles to view.go"
```

---

### Task 6: Fix `sendAndRead` — partial timeout error, yield, buffer reset

**Files:**
- Modify: `internal/device/serial.go:55-88`

- [ ] **Step 1: Write failing test for partial timeout error**

Add to `internal/device/serial_test.go`:

```go
func TestSendAndRead_PartialTimeout(t *testing.T) {
	// Port returns partial data but never OK/ERROR/+++
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(nil)
	port.On("Write", tmock.Anything).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("partial"), nil).Once()
	port.On("Read", tmock.Anything).Return(0, nil) // subsequent reads return nothing
	conn := NewSerialConn(port)

	resp, err := conn.sendAndRead("AT+TEST\r\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Contains(t, resp, "partial")
}
```

- [ ] **Step 2: Write failing test for ResetInputBuffer error**

Add to `internal/device/serial_test.go`:

```go
func TestSendAndRead_ResetBufferError(t *testing.T) {
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(errors.New("reset failed"))
	conn := NewSerialConn(port)

	_, err := conn.sendAndRead("AT+TEST\r\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reset buffer")
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test -v -run "TestSendAndRead_PartialTimeout|TestSendAndRead_ResetBufferError" ./internal/device/...`
Expected: Both tests FAIL — `sendAndRead` is not yet exported and current behavior returns nil error for partial data and ignores reset errors.

Note: `sendAndRead` is unexported. These tests are in `package device` so they can access it directly.

- [ ] **Step 4: Update `sendAndRead` in `serial.go`**

Replace the `sendAndRead` method (lines 56-88 of `internal/device/serial.go`) with:

```go
// sendAndRead writes a command and reads the response. NOT thread-safe — caller must hold session lock.
func (s *SerialConn) sendAndRead(cmd string) (string, error) {
	if err := s.port.ResetInputBuffer(); err != nil {
		return "", fmt.Errorf("reset buffer: %w", err)
	}

	_, err := s.port.Write([]byte(cmd))
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	var buf strings.Builder
	deadline := time.Now().Add(atTimeout)
	tmp := make([]byte, 256)

	for time.Now().Before(deadline) {
		n, err := s.port.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
			resp := buf.String()
			if strings.Contains(resp, "OK") || strings.Contains(resp, "ERROR") || strings.Contains(resp, "+++") {
				time.Sleep(interCmdDelay)
				return strings.TrimSpace(resp), nil
			}
		}
		if err != nil || n == 0 {
			time.Sleep(time.Millisecond) // yield to prevent CPU spin
			continue
		}
	}

	resp := buf.String()
	if resp == "" {
		return "", fmt.Errorf("timeout: no response")
	}
	return strings.TrimSpace(resp), fmt.Errorf("timeout: incomplete response: %s", resp)
}
```

- [ ] **Step 5: Run the new tests to verify they pass**

Run: `go test -v -run "TestSendAndRead_PartialTimeout|TestSendAndRead_ResetBufferError" ./internal/device/...`
Expected: Both tests PASS.

- [ ] **Step 6: Run full test suite to check for regressions**

Run: `go test -v -count=1 ./...`
Expected: All tests pass. No existing test relies on partial-data-without-error behavior.

- [ ] **Step 7: Commit**

```bash
git add internal/device/serial.go internal/device/serial_test.go
git commit -m "fix(device): sendAndRead reports partial timeout, yields on empty reads, checks buffer reset"
```

---

### Task 7: Extract `withATSession` and fix error handling in `at.go`

**Files:**
- Modify: `internal/device/at.go`
- Modify: `internal/device/serial.go` (add `withATSession` method)

- [ ] **Step 1: Write failing test for `withATSession` — exit error surfaced**

Add to `internal/device/serial_test.go`:

```go
func TestWithATSession_ExitErrorSurfaced(t *testing.T) {
	// Enter succeeds, callback succeeds, exit fails
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(nil)
	port.On("Write", []byte("+++\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("+++"), nil).Once()
	port.On("Write", []byte("AT+EXIT\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("ERROR"), nil).Once()
	conn := NewSerialConn(port)

	err := conn.withATSession(func() error {
		return nil // callback succeeds
	})
	assert.ErrorContains(t, err, "exit AT mode")
}

func TestWithATSession_CallbackErrorPriority(t *testing.T) {
	// Enter succeeds, callback fails, exit also fails — callback error wins
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(nil)
	port.On("Write", []byte("+++\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("+++"), nil).Once()
	port.On("Write", []byte("AT+EXIT\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("OK"), nil).Once()
	conn := NewSerialConn(port)

	cbErr := errors.New("callback failed")
	err := conn.withATSession(func() error {
		return cbErr
	})
	assert.ErrorIs(t, err, cbErr)
}

func TestWithATSession_Success(t *testing.T) {
	port := devicemock.ForResponses("+++", "OK")
	conn := NewSerialConn(port)

	err := conn.withATSession(func() error {
		return nil
	})
	require.NoError(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v -run "TestWithATSession" ./internal/device/...`
Expected: FAIL — `withATSession` does not exist yet.

- [ ] **Step 3: Add `withATSession` to `serial.go`**

Add after the `sendAndRead` method in `internal/device/serial.go`:

```go
// withATSession acquires the session lock, enters AT mode, runs fn, exits AT mode.
// If fn returns an error, exitAT is still attempted and both errors are reported.
// Callback error takes priority over exit error.
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

- [ ] **Step 4: Run `withATSession` tests**

Run: `go test -v -run "TestWithATSession" ./internal/device/...`
Expected: All 3 tests PASS.

- [ ] **Step 5: Refactor `SetParam` to use `withATSession`**

Replace `SetParam` in `internal/device/at.go` (lines 56-76) with:

```go
// SetParam: full atomic session +++ → AT+CMD=value → AT+EXIT
func SetParam(conn *SerialConn, atCmd, value string) error {
	return conn.withATSession(func() error {
		cmd := fmt.Sprintf("AT+%s=%s\r\n", atCmd, value)
		resp, err := conn.sendAndRead(cmd)
		if err != nil {
			return fmt.Errorf("set %s=%s: %w", atCmd, value, err)
		}
		if !strings.Contains(resp, "OK") {
			return fmt.Errorf("set %s=%s: %s", atCmd, value, resp)
		}
		return nil
	})
}
```

- [ ] **Step 6: Refactor `ReadAllParamsAndVersion` to use `withATSession`**

Replace `ReadAllParamsAndVersion` in `internal/device/at.go` (lines 78-107) with:

```go
// ReadAllParamsAndVersion: single session +++ → AT+ALLP? → AT+VER → AT+EXIT
func ReadAllParamsAndVersion(conn *SerialConn) (map[string]string, string, error) {
	var params map[string]string
	var version string

	err := conn.withATSession(func() error {
		resp, err := conn.sendAndRead("AT+ALLP?\r\n")
		if err != nil {
			return fmt.Errorf("read ALLP: %w", err)
		}

		p, err := parseALLP(resp)
		if err != nil {
			return err
		}
		params = p

		// Read version in same session — non-fatal if it fails
		verResp, verErr := conn.sendAndRead("AT+VER\r\n")
		if verErr == nil {
			version = parseVersion(verResp)
		}

		return nil
	})

	if err != nil {
		return nil, "", err
	}
	return params, version, nil
}
```

- [ ] **Step 7: Fix `ResetInputBuffer` error in `enterAT`**

Replace the `enterAT` function in `internal/device/at.go` (lines 19-40) with:

```go
// enterAT sends +++ and expects echo, with retries and guard time.
// Caller must hold session lock.
func enterAT(conn *SerialConn) error {
	var lastErr error
	for attempt := range enterATRetries {
		if attempt > 0 {
			time.Sleep(preEnterDelay)
		}
		if err := conn.port.ResetInputBuffer(); err != nil {
			lastErr = fmt.Errorf("enter AT mode: reset buffer: %w", err)
			continue
		}
		time.Sleep(preEnterDelay)

		resp, err := conn.sendAndRead("+++\r\n")
		if err != nil {
			lastErr = fmt.Errorf("enter AT mode: %w", err)
			continue
		}
		if !strings.Contains(resp, "+++") {
			lastErr = fmt.Errorf("enter AT mode: unexpected response: %s", resp)
			continue
		}
		return nil
	}
	return lastErr
}
```

- [ ] **Step 8: Run full test suite**

Run: `go test -v -count=1 ./...`
Expected: All tests pass. The refactored functions have identical behavior to the originals, except errors are no longer silently ignored.

- [ ] **Step 9: Commit**

```bash
git add internal/device/serial.go internal/device/at.go internal/device/serial_test.go
git commit -m "refactor(device): extract withATSession, fix ignored errors in AT session lifecycle"
```

---

### Task 8: Add test for `enterAT` ResetInputBuffer error path

**Files:**
- Modify: `internal/device/serial_test.go`

- [ ] **Step 1: Write test for enterAT when ResetInputBuffer fails**

Add to `internal/device/serial_test.go`:

```go
func TestEnterAT_ResetBufferError(t *testing.T) {
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(errors.New("reset failed"))
	conn := NewSerialConn(port)

	err := enterAT(conn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reset buffer")
}
```

- [ ] **Step 2: Run the test**

Run: `go test -v -run "TestEnterAT_ResetBufferError" ./internal/device/...`
Expected: PASS (the fix from Task 7 Step 7 already handles this).

- [ ] **Step 3: Run full test suite one final time**

Run: `go test -v -count=1 ./...`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/device/serial_test.go
git commit -m "test(device): add enterAT ResetInputBuffer error path test"
```

---

### Task 9: Final verification

- [ ] **Step 1: Verify file structure**

Run: `ls -la internal/tui/*.go | grep -v _test`
Expected:
```
ansi.go
commands.go
field.go
keys.go
model.go
navigation.go
params.go
view.go
```

- [ ] **Step 2: Verify model.go size**

Run: `wc -l internal/tui/model.go`
Expected: Approximately 100-200 lines.

- [ ] **Step 3: Verify no timing constants changed**

Run: `grep -n "atTimeout\|postExitDelay\|preEnterDelay\|interCmdDelay\|enterATRetries" internal/device/at.go`
Expected output:
```
10:	atTimeout      = 3 * time.Second
11:	postExitDelay  = 1 * time.Second
12:	preEnterDelay  = 300 * time.Millisecond
13:	interCmdDelay  = 100 * time.Millisecond
14:	enterATRetries = 3
```

- [ ] **Step 4: Run full test suite with race detector**

Run: `go test -race -count=1 ./...`
Expected: All tests pass, no data races.

- [ ] **Step 5: Run linter**

Run: `golangci-lint run ./...`
Expected: No new warnings.

- [ ] **Step 6: Commit any final cleanups if needed**

If the linter or race detector surfaced issues, fix and commit:
```bash
git add -A
git commit -m "chore: final cleanup after optimization refactor"
```
