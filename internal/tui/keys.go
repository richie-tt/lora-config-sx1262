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
