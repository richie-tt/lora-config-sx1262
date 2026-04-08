package tui

import (
	"fmt"
	"lora-config-SX1262/internal/device"
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
	conn    *device.SerialConn
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
	conn        *device.SerialConn
	version     string
	statusMsg   string
	tag         string
	commit      string
	buildDate   string
	width       int
	height      int

	// Layout: left column field indices, right column field indices
	leftCol  []int
	rightCol []int
}

// InitialModel returns the initial BubbleTea model for the application.
func InitialModel(tag, commit, buildDate string) tea.Model {
	deviceInput := textinput.New()
	deviceInput.Placeholder = "/dev/ttyACM0"
	deviceInput.SetValue("/dev/ttyACM0")
	deviceInput.CharLimit = 64
	deviceInput.Width = 30
	deviceInput.Focus()

	fields := make([]Field, len(allParams))
	for i, p := range allParams {
		fields[i] = newField(p)
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
		deviceInput: deviceInput,
		focusIndex:  focusDevice,
		leftCol:     left,
		rightCol:    right,
		statusMsg:   "Enter device path and press Connect",
		tag:         tag,
		commit:      commit,
		buildDate:   buildDate,
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

// Commands

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
