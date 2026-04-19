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

		// Single session: +++ -> AT+ALLP? -> AT+VER -> AT+EXIT
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
