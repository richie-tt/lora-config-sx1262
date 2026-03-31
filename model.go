package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Focus targets
const (
	focusDevice  = -1
	focusConnect = -2
	focusRestore = -3
	focusReboot  = -4
)

// Messages
type connectResultMsg struct {
	conn    *SerialConn
	params  map[string]string
	version string
	err     error
}

type disconnectMsg struct{}

type paramResultMsg struct {
	fieldIndex int
	ok         bool
	err        error
}

type (
	restoreResultMsg struct{ err error }
	rebootResultMsg  struct{ err error }
)

type model struct {
	fields      []Field
	deviceInput textinput.Model
	focusIndex  int // -1=device, -2=connect, -3=restore, -4=reboot, 0..N=field index
	connected   bool
	connecting  bool
	conn        *SerialConn
	version     string
	statusMsg   string
	width       int
	height      int

	// Layout: left column field indices, right column field indices
	leftCol  []int
	rightCol []int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "/dev/ttyACM0"
	ti.SetValue("/dev/ttyACM0")
	ti.CharLimit = 64
	ti.Width = 30
	ti.Focus()

	fields := make([]Field, len(AllParams))
	for i, p := range AllParams {
		fields[i] = NewField(p)
		fields[i].Disabled = true
	}

	// Split into two columns: first 8 left, rest right
	left := make([]int, 0)
	right := make([]int, 0)
	half := (len(fields) + 1) / 2
	for i := range fields {
		if i < half {
			left = append(left, i)
		} else {
			right = append(right, i)
		}
	}

	return model{
		fields:      fields,
		deviceInput: ti,
		focusIndex:  focusDevice,
		leftCol:     left,
		rightCol:    right,
		statusMsg:   "Enter device path and press Connect",
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case connectResultMsg:
		m.connecting = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Connection failed: %v", msg.err)
			m.connected = false
			return m, nil
		}
		m.connected = true
		m.conn = msg.conn
		m.version = msg.version
		m.statusMsg = fmt.Sprintf("Connected to %s", m.deviceInput.Value())

		// Populate fields from params
		for i := range m.fields {
			m.fields[i].Disabled = false
			if val, ok := msg.params[m.fields[i].ATCmd]; ok {
				m.fields[i].SetByValue(val)
			}
		}
		m.focusIndex = 0
		m.updateFocus()
		return m, nil

	case disconnectMsg:
		if m.conn != nil {
			m.conn.Close()
			m.conn = nil
		}
		m.connected = false
		m.version = ""
		for i := range m.fields {
			m.fields[i].Disabled = true
			m.fields[i].Status = StatusNormal
		}
		m.statusMsg = "Disconnected"
		m.focusIndex = focusDevice
		m.updateFocus()
		return m, nil

	case paramResultMsg:
		if msg.fieldIndex >= 0 && msg.fieldIndex < len(m.fields) {
			if msg.ok {
				m.fields[msg.fieldIndex].Status = StatusSuccess
				m.statusMsg = fmt.Sprintf("Set %s = %s OK", m.fields[msg.fieldIndex].ATCmd, m.fields[msg.fieldIndex].SelectedDisplay())
			} else {
				m.fields[msg.fieldIndex].Status = StatusError
				m.statusMsg = fmt.Sprintf("Failed to set %s: %v", m.fields[msg.fieldIndex].ATCmd, msg.err)
			}
		}
		return m, nil

	case restoreResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Restore failed: %v", msg.err)
		} else {
			m.statusMsg = "Factory restore sent. Device may reboot."
		}
		return m, nil

	case rebootResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Reboot failed: %v", msg.err)
		} else {
			m.statusMsg = "Reboot command sent."
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

		// case tea.MouseMsg:
		// 	return m.handleMouse(msg)
	}

	// Forward non-key messages to active text inputs (for cursor blink)
	if m.focusIndex == focusDevice {
		var cmd tea.Cmd
		m.deviceInput, cmd = m.deviceInput.Update(msg)
		return m, cmd
	}

	if idx := m.editingFieldIndex(); idx >= 0 {
		var cmd tea.Cmd
		m.fields[idx].NumInput, cmd = m.fields[idx].NumInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c":
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
	f := &m.fields[fieldIdx]
	key := msg.String()

	switch key {
	case "enter":
		// Validate and submit
		val, ok := f.ValidateNumInput()
		if !ok {
			m.statusMsg = fmt.Sprintf("%s: must be %d-%d", f.Label, f.Min, f.Max)
			f.Status = StatusError
			f.NumInput.SetValue(f.LastValue) // revert
			f.Editing = false
			f.NumInput.Blur()
			return m, nil
		}
		f.NumInput.SetValue(val)
		f.Editing = false
		f.NumInput.Blur()
		if m.conn != nil && val != f.LastValue {
			f.LastValue = val
			m.statusMsg = fmt.Sprintf("Setting %s = %s...", f.ATCmd, val)
			return m, m.setParamCmd(fieldIdx)
		}
		f.LastValue = val
		return m, nil
	case "esc":
		// Cancel editing, revert
		f.NumInput.SetValue(f.LastValue)
		f.Editing = false
		f.NumInput.Blur()
		return m, nil
	default:
		// Forward to text input (only allow digits)
		var cmd tea.Cmd
		f.NumInput, cmd = f.NumInput.Update(msg)
		return m, cmd
	}
}

func (m model) handleDropdownKey(fieldIdx int, key string) (tea.Model, tea.Cmd) {
	f := &m.fields[fieldIdx]

	switch key {
	case "up", "k":
		f.MoveUp()
	case "down", "j":
		f.MoveDown()
	case "enter":
		prevValue := f.LastValue
		f.Open = false
		newValue := f.SelectedValue()
		f.LastValue = newValue
		// Only send AT command if value actually changed
		if m.conn != nil && newValue != prevValue {
			m.statusMsg = fmt.Sprintf("Setting %s = %s...", f.ATCmd, f.SelectedDisplay())
			return m, m.setParamCmd(fieldIdx)
		}
	case "esc":
		// Revert to previous selection
		f.SetByValue(f.LastValue)
		f.Open = false
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
			f := &m.fields[m.focusIndex]
			if !f.Disabled {
				f.Status = StatusNormal
				if f.IsNumInput {
					// Start editing numeric field
					f.Editing = true
					f.LastValue = f.NumInput.Value()
					cmd := f.NumInput.Focus()
					m.statusMsg = fmt.Sprintf("Editing %s (%d-%d), Enter to confirm, Esc to cancel", f.Label, f.Min, f.Max)
					return m, cmd
				}
				f.ToggleOpen()
			}
		}
	}
	return m, nil
}

// Navigation helpers

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
		// Last field in column → go to buttons
		m.focusIndex = focusRestore
	}
}

func (m *model) focusPrevInColumn() {
	switch m.focusIndex {
	case focusRestore:
		// Go back to last field in left column
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
		// First field in column → go to Connect
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

func (m model) anyDropdownOpen() bool {
	for _, f := range m.fields {
		if f.Open {
			return true
		}
	}
	return false
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

// Mouse handling
//
// Layout rows (fixed):
//   0-2: title area (3 lines: title + margin + blank)
//   3-5: device row (3 lines with border)
//   6-7: separator + blank
//   8+:  field grid, each field row = 3 lines (border+content+border)
//         row 0: lines 8-10, row 1: lines 11-13, etc.
//   after grid: blank, then buttons row (3 lines with border)

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionRelease {
		return m, nil
	}

	y := msg.Y
	x := msg.X

	// Close any open dropdown/editing
	for i := range m.fields {
		if m.fields[i].Open {
			m.fields[i].Open = false
		}
		if m.fields[i].Editing {
			m.fields[i].NumInput.SetValue(m.fields[i].LastValue)
			m.fields[i].Editing = false
			m.fields[i].NumInput.Blur()
		}
	}

	// Device row: lines 3-5
	if y >= 3 && y <= 5 {
		if x >= 50 {
			// Connect button area
			m.focusIndex = focusConnect
			m.updateFocus()
			return m.handleEnter()
		}
		if !m.connected {
			m.focusIndex = focusDevice
			m.updateFocus()
		}
		return m, nil
	}

	// Field grid starts at line 8, each row = 3 lines
	gridStart := 8
	maxRows := len(m.leftCol)
	if len(m.rightCol) > maxRows {
		maxRows = len(m.rightCol)
	}
	gridEnd := gridStart + maxRows*3

	if y >= gridStart && y < gridEnd {
		row := (y - gridStart) / 3

		// Determine column by x position
		var target int = -1
		if x < 36 {
			// Left column
			if row < len(m.leftCol) {
				target = m.leftCol[row]
			}
		} else {
			// Right column
			if row < len(m.rightCol) {
				target = m.rightCol[row]
			}
		}

		if target >= 0 && target < len(m.fields) {
			f := &m.fields[target]
			if !f.Disabled {
				m.focusIndex = target
				m.updateFocus()
				f.Status = StatusNormal
				if f.IsNumInput {
					f.Editing = true
					f.LastValue = f.NumInput.Value()
					cmd := f.NumInput.Focus()
					m.statusMsg = fmt.Sprintf("Editing %s (%d-%d), Enter to confirm, Esc to cancel", f.Label, f.Min, f.Max)
					return m, cmd
				}
				f.ToggleOpen()
			}
		}
		return m, nil
	}

	// Buttons row: after grid + 1 blank line
	btnStart := gridEnd + 1
	btnEnd := btnStart + 2
	if y >= btnStart && y <= btnEnd {
		if x < 20 {
			m.focusIndex = focusRestore
			m.updateFocus()
			return m.handleEnter()
		} else if x < 40 {
			m.focusIndex = focusReboot
			m.updateFocus()
			return m.handleEnter()
		}
		return m, nil
	}

	return m, nil
}

// Commands

func (m model) connectCmd() tea.Cmd {
	device := m.deviceInput.Value()
	return func() tea.Msg {
		conn, err := OpenSerial(device, 115200)
		if err != nil {
			return connectResultMsg{err: err}
		}

		// Single session: +++ → AT+ALLP? → AT+VER → AT+EXIT
		params, version, err := ReadAllParamsAndVersion(conn)
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
	f := m.fields[fieldIdx]
	conn := m.conn
	return func() tea.Msg {
		err := SetParam(conn, f.ATCmd, f.SelectedValue())
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
		return restoreResultMsg{err: Restore(conn)}
	}
}

func (m model) rebootCmd() tea.Cmd {
	conn := m.conn
	return func() tea.Msg {
		return rebootResultMsg{err: Reboot(conn)}
	}
}

// overlayString places overlay on top of base at the given column offset.
// It replaces characters in base with overlay content, preserving ANSI sequences.
func overlayString(base, overlay string, col int) string {
	// Simple approach: pad base to col, then append overlay, then append rest of base
	baseRunes := []rune(stripAnsi(base))
	baseWidth := len(baseRunes)

	// Ensure base is wide enough
	if baseWidth < col {
		base += strings.Repeat(" ", col-baseWidth)
	}

	// Split base into: before overlay, and after overlay
	// We work with raw strings and use visual width
	lines := base
	overlayWidth := lipgloss.Width(overlay)

	// Build result: take base up to col, add overlay, then rest of base after overlay
	baseBefore := truncateToWidth(lines, col)
	baseAfter := skipWidth(lines, col+overlayWidth)

	return baseBefore + overlay + baseAfter
}

// truncateToWidth returns the prefix of s up to the given visual width.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	var result strings.Builder
	w := 0
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			result.WriteRune(r)
			continue
		}
		if inEsc {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if w >= width {
			break
		}
		result.WriteRune(r)
		w++
	}
	return result.String()
}

// skipWidth skips the first `width` visible characters of s, returning the rest.
func skipWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	w := 0
	inEsc := false
	for i, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if w >= width {
			return s[i:]
		}
		w++
	}
	return ""
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// View

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
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("  LoRa Configurator for SX1262"))
	b.WriteString("\n\n")

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
	b.WriteString(deviceRow)
	b.WriteString("\n")

	// Separator
	sep := separatorStyle.Render(strings.Repeat("─", 70))
	b.WriteString(sep)
	b.WriteString("\n\n")

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
	for row := 0; row < maxRows; row++ {
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
		b.WriteString(gl)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Bottom buttons row
	var restoreBtn, rebootBtn string

	if !m.connected {
		restoreBtn = buttonDisabledStyle.Render("Restore")
		rebootBtn = buttonDisabledStyle.Render("Reboot")
	} else if m.focusIndex == focusRestore {
		restoreBtn = buttonFocusedStyle.Render("Restore")
		rebootBtn = buttonStyle.Render("Reboot")
	} else if m.focusIndex == focusReboot {
		restoreBtn = buttonStyle.Render("Restore")
		rebootBtn = buttonFocusedStyle.Render("Reboot")
	} else {
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
	b.WriteString(buttonsRow)
	b.WriteString("\n")

	// Status bar
	b.WriteString(statusBarStyle.Render(fmt.Sprintf("  %s", m.statusMsg)))
	b.WriteString("\n")

	// Help
	help := "Tab/Shift+Tab: navigate • Enter: select • ↑↓: move • ←→: switch column • Click: select • q: quit"
	b.WriteString(helpStyle.Render("  " + help))

	return b.String()
}
