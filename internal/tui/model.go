package tui

import (
	"fmt"
	"lora-config-SX1262/internal/device"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
