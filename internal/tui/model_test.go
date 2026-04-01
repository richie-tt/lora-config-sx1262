package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

func testModel() model {
	return InitialModel().(model)
}

func connectedModel() model {
	mdl := testModel()
	mdl.connected = true
	for idx := range mdl.fields {
		mdl.fields[idx].Disabled = false
	}
	return mdl
}

// --- Navigation tests ---

func TestFocusNext_Disconnected(t *testing.T) {
	mdl := testModel()
	require.Equal(t, focusDevice, mdl.focusIndex)

	mdl.focusNext()
	assert.Equal(t, focusConnect, mdl.focusIndex)

	mdl.focusNext()
	assert.Equal(t, focusDevice, mdl.focusIndex)
}

func TestFocusNext_Connected(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusConnect

	mdl.focusNext()
	assert.Equal(t, 0, mdl.focusIndex)

	for range len(mdl.fields) - 1 {
		mdl.focusNext()
	}
	assert.Equal(t, len(mdl.fields)-1, mdl.focusIndex)

	mdl.focusNext()
	assert.Equal(t, focusRestore, mdl.focusIndex)

	mdl.focusNext()
	assert.Equal(t, focusReboot, mdl.focusIndex)

	mdl.focusNext()
	assert.Equal(t, focusConnect, mdl.focusIndex)
}

func TestFocusPrev_Connected(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusConnect

	mdl.focusPrev()
	assert.Equal(t, focusReboot, mdl.focusIndex)

	mdl.focusPrev()
	assert.Equal(t, focusRestore, mdl.focusIndex)

	mdl.focusPrev()
	assert.Equal(t, len(mdl.fields)-1, mdl.focusIndex)
}

func TestFocusPrev_Disconnected(t *testing.T) {
	mdl := testModel()

	mdl.focusPrev()
	assert.Equal(t, focusReboot, mdl.focusIndex)

	mdl.focusPrev()
	assert.Equal(t, focusRestore, mdl.focusIndex)

	mdl.focusPrev()
	assert.Equal(t, focusConnect, mdl.focusIndex)

	mdl.focusPrev()
	assert.Equal(t, focusDevice, mdl.focusIndex)
}

func TestFocusNextInColumn(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = mdl.leftCol[0]

	mdl.focusNextInColumn()
	assert.Equal(t, mdl.leftCol[1], mdl.focusIndex)

	for range len(mdl.leftCol) - 2 {
		mdl.focusNextInColumn()
	}
	assert.Equal(t, mdl.leftCol[len(mdl.leftCol)-1], mdl.focusIndex)

	mdl.focusNextInColumn()
	assert.Equal(t, focusRestore, mdl.focusIndex)

	mdl.focusNextInColumn()
	assert.Equal(t, focusReboot, mdl.focusIndex)

	mdl.focusNextInColumn()
	assert.Equal(t, focusReboot, mdl.focusIndex, "reboot should stay")
}

func TestFocusPrevInColumn(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = mdl.leftCol[2]

	mdl.focusPrevInColumn()
	assert.Equal(t, mdl.leftCol[1], mdl.focusIndex)

	mdl.focusPrevInColumn()
	assert.Equal(t, mdl.leftCol[0], mdl.focusIndex)

	mdl.focusPrevInColumn()
	assert.Equal(t, focusConnect, mdl.focusIndex)

	mdl.focusPrevInColumn()
	assert.Equal(t, focusConnect, mdl.focusIndex, "connect should stay")
}

func TestFocusPrevInColumn_Restore(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusRestore

	mdl.focusPrevInColumn()
	assert.Equal(t, mdl.leftCol[len(mdl.leftCol)-1], mdl.focusIndex)
}

func TestFocusPrevInColumn_RebootToRestore(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusReboot

	mdl.focusPrevInColumn()
	assert.Equal(t, focusRestore, mdl.focusIndex)
}

func TestFocusSwitchColumn(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = mdl.leftCol[0]

	mdl.focusSwitchColumn()
	assert.Equal(t, mdl.rightCol[0], mdl.focusIndex)

	mdl.focusSwitchColumn()
	assert.Equal(t, mdl.leftCol[0], mdl.focusIndex)
}

func TestFocusSwitchColumn_NoOpWhenNegative(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusDevice

	mdl.focusSwitchColumn()
	assert.Equal(t, focusDevice, mdl.focusIndex)
}

func TestFocusSwitchColumn_ClampToLastRow(t *testing.T) {
	mdl := connectedModel()

	longerCol := mdl.leftCol
	shorterCol := mdl.rightCol
	if len(mdl.rightCol) > len(mdl.leftCol) {
		longerCol = mdl.rightCol
		shorterCol = mdl.leftCol
	}

	if len(longerCol) > len(shorterCol) {
		mdl.focusIndex = longerCol[len(longerCol)-1]
		mdl.focusSwitchColumn()
		assert.Equal(t, shorterCol[len(shorterCol)-1], mdl.focusIndex)
	}
}

func TestFieldPosition(t *testing.T) {
	mdl := connectedModel()

	col, row := mdl.fieldPosition(mdl.leftCol[0])
	assert.Equal(t, 0, col)
	assert.Equal(t, 0, row)

	col, row = mdl.fieldPosition(mdl.rightCol[0])
	assert.Equal(t, 1, col)
	assert.Equal(t, 0, row)
}

func TestOpenDropdownIndex(t *testing.T) {
	mdl := connectedModel()
	assert.Equal(t, -1, mdl.openDropdownIndex())

	mdl.fields[2].Open = true
	assert.Equal(t, 2, mdl.openDropdownIndex())
}

func TestEditingFieldIndex(t *testing.T) {
	mdl := connectedModel()
	assert.Equal(t, -1, mdl.editingFieldIndex())

	for idx, field := range mdl.fields {
		if field.IsNumInput {
			mdl.fields[idx].Editing = true
			assert.Equal(t, idx, mdl.editingFieldIndex())
			break
		}
	}
}

// --- Update message tests ---

func TestUpdateWindowSize(t *testing.T) {
	mdl := testModel()
	newM, cmd := mdl.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := newM.(model)

	assert.Equal(t, 120, got.width)
	assert.Equal(t, 40, got.height)
	assert.Nil(t, cmd)
}

func TestUpdateConnectResult_Success(t *testing.T) {
	mdl := testModel()
	mdl.connecting = true

	params := map[string]string{
		"SF": "7", "BW": "0", "CR": "1", "PWR": "22",
		"NETID": "0", "LBT": "0", "MODE": "1",
		"TXCH": "10", "RXCH": "10", "RSSI": "0",
		"ADDR": "0", "PORT": "0", "COMM": `"8N1"`,
		"BAUD": "9600", "KEY": "0",
	}
	newM, _ := mdl.Update(connectResultMsg{params: params, version: "1.0.0"})
	got := newM.(model)

	assert.True(t, got.connected)
	assert.False(t, got.connecting)
	assert.Equal(t, "1.0.0", got.version)
	for idx, field := range got.fields {
		assert.False(t, field.Disabled, "field[%d] should not be disabled", idx)
	}
}

func TestUpdateConnectResult_Error(t *testing.T) {
	mdl := testModel()
	mdl.connecting = true

	newM, _ := mdl.Update(connectResultMsg{err: errTest})
	got := newM.(model)

	assert.False(t, got.connected)
	assert.False(t, got.connecting)
	assert.Contains(t, got.statusMsg, "Connection failed")
}

func TestUpdateDisconnect(t *testing.T) {
	mdl := connectedModel()

	newM, _ := mdl.Update(disconnectMsg{})
	got := newM.(model)

	assert.False(t, got.connected)
	for idx, field := range got.fields {
		assert.True(t, field.Disabled, "field[%d] should be disabled", idx)
	}
	assert.Equal(t, focusDevice, got.focusIndex)
}

func TestUpdateParamResult_Success(t *testing.T) {
	mdl := connectedModel()
	newM, _ := mdl.Update(paramResultMsg{fieldIndex: 0, ok: true})
	got := newM.(model)

	assert.Equal(t, StatusSuccess, got.fields[0].Status)
}

func TestUpdateParamResult_Error(t *testing.T) {
	mdl := connectedModel()
	newM, _ := mdl.Update(paramResultMsg{fieldIndex: 1, ok: false, err: errTest})
	got := newM.(model)

	assert.Equal(t, StatusError, got.fields[1].Status)
}

func TestUpdateParamResult_OutOfBounds(t *testing.T) {
	mdl := connectedModel()
	assert.NotPanics(t, func() {
		mdl.Update(paramResultMsg{fieldIndex: 999, ok: true}) //nolint:errcheck
	})
}

func TestUpdateRestoreResult(t *testing.T) {
	mdl := connectedModel()

	newM, _ := mdl.Update(restoreResultMsg{err: nil})
	got := newM.(model)
	assert.Contains(t, got.statusMsg, "Factory restore")

	newM, _ = mdl.Update(restoreResultMsg{err: errTest})
	got = newM.(model)
	assert.Contains(t, got.statusMsg, "Restore failed")
}

func TestUpdateRebootResult(t *testing.T) {
	mdl := connectedModel()

	newM, _ := mdl.Update(rebootResultMsg{err: nil})
	got := newM.(model)
	assert.Contains(t, got.statusMsg, "Reboot command sent")

	newM, _ = mdl.Update(rebootResultMsg{err: errTest})
	got = newM.(model)
	assert.Contains(t, got.statusMsg, "Reboot failed")
}

// --- Key handling tests ---

func TestKeyCtrlC(t *testing.T) {
	mdl := connectedModel()
	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd)
}

func TestKeyQ_Quit(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = 0
	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd)
}

func TestKeyQ_NoQuitInDeviceInput(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = focusDevice
	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		msg := cmd()
		assert.NotEqual(t, tea.Quit(), msg)
	}
}

func TestKeyTab_DeviceToConnect(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = focusDevice

	newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := newM.(model)
	assert.Equal(t, focusConnect, got.focusIndex)
}

func TestKeyShiftTab_DeviceToReboot(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = focusDevice

	newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	got := newM.(model)
	assert.Equal(t, focusReboot, got.focusIndex)
}

func TestKeyEnter_Connect(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = focusConnect

	newM, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := newM.(model)

	assert.True(t, got.connecting)
	assert.NotNil(t, cmd)
}

func TestKeyEnter_Disconnect(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusConnect

	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
}

func TestKeyEnter_AlreadyConnecting(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = focusConnect
	mdl.connecting = true

	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestKeyTab_Navigation(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = 0
	mdl.updateFocus()

	newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := newM.(model)
	assert.Equal(t, 1, got.focusIndex)
}

func TestKeyUpDown_ColumnNavigation(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = mdl.leftCol[0]
	mdl.updateFocus()

	newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyDown})
	got := newM.(model)
	assert.Equal(t, mdl.leftCol[1], got.focusIndex)

	newM, _ = got.Update(tea.KeyMsg{Type: tea.KeyUp})
	got = newM.(model)
	assert.Equal(t, mdl.leftCol[0], got.focusIndex)
}

func TestKeyLeftRight_SwitchColumn(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = mdl.leftCol[0]
	mdl.updateFocus()

	newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyRight})
	got := newM.(model)
	assert.Equal(t, mdl.rightCol[0], got.focusIndex)

	newM, _ = got.Update(tea.KeyMsg{Type: tea.KeyLeft})
	got = newM.(model)
	assert.Equal(t, mdl.leftCol[0], got.focusIndex)
}

func TestKeyEnter_OpenDropdown(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if !field.IsNumInput {
			mdl.focusIndex = idx
			mdl.updateFocus()

			newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := newM.(model)
			assert.True(t, got.fields[idx].Open)
			return
		}
	}
	t.Skip("no dropdown fields found")
}

func TestKeyEnter_StartNumEditing(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if field.IsNumInput {
			mdl.focusIndex = idx
			mdl.updateFocus()

			newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := newM.(model)
			assert.True(t, got.fields[idx].Editing)
			return
		}
	}
	t.Skip("no numeric fields found")
}

func TestKeyD_Disconnect(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = 0
	mdl.updateFocus()

	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.NotNil(t, cmd)
}

func TestKeyD_NotConnected(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = focusConnect

	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Nil(t, cmd)
}

func TestDropdownNavigation(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if !field.IsNumInput {
			mdl.fields[idx].Open = true
			mdl.focusIndex = idx
			mdl.updateFocus()

			newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyDown})
			got := newM.(model)
			assert.Equal(t, field.Selected+1, got.fields[idx].Selected)

			newM, _ = got.Update(tea.KeyMsg{Type: tea.KeyUp})
			got = newM.(model)
			assert.Equal(t, field.Selected, got.fields[idx].Selected)

			newM, _ = got.Update(tea.KeyMsg{Type: tea.KeyEscape})
			got = newM.(model)
			assert.False(t, got.fields[idx].Open)
			return
		}
	}
}

func TestDropdownJK(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if !field.IsNumInput {
			mdl.fields[idx].Open = true
			mdl.focusIndex = idx

			newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
			got := newM.(model)
			assert.Equal(t, 1, got.fields[idx].Selected)

			newM, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
			got = newM.(model)
			assert.Equal(t, 0, got.fields[idx].Selected)
			return
		}
	}
}

func TestDropdownEnter_NoChange(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if !field.IsNumInput {
			mdl.fields[idx].Open = true
			mdl.fields[idx].LastValue = field.Options[0].Value
			mdl.fields[idx].Selected = 0
			mdl.focusIndex = idx

			newM, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := newM.(model)

			assert.False(t, got.fields[idx].Open)
			assert.Nil(t, cmd, "should not send AT command when value unchanged")
			return
		}
	}
}

func TestNumInputEsc(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if field.IsNumInput {
			mdl.fields[idx].Editing = true
			mdl.fields[idx].LastValue = "42"
			mdl.fields[idx].NumInput.SetValue("99")
			mdl.fields[idx].NumInput.Focus()
			mdl.focusIndex = idx

			newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyEscape})
			got := newM.(model)

			assert.False(t, got.fields[idx].Editing)
			assert.Equal(t, "42", got.fields[idx].NumInput.Value())
			return
		}
	}
}

func TestNumInputEnter_Valid(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if field.IsNumInput {
			mdl.fields[idx].Editing = true
			mdl.fields[idx].LastValue = "0"
			mdl.fields[idx].NumInput.SetValue("42")
			mdl.fields[idx].NumInput.Focus()
			mdl.focusIndex = idx

			newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := newM.(model)

			assert.False(t, got.fields[idx].Editing)
			assert.Equal(t, "42", got.fields[idx].NumInput.Value())
			return
		}
	}
}

func TestNumInputEnter_Invalid(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if field.IsNumInput {
			mdl.fields[idx].Editing = true
			mdl.fields[idx].LastValue = "10"
			mdl.fields[idx].NumInput.SetValue("abc")
			mdl.fields[idx].NumInput.Focus()
			mdl.focusIndex = idx

			newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := newM.(model)

			assert.False(t, got.fields[idx].Editing)
			assert.Equal(t, StatusError, got.fields[idx].Status)
			assert.Equal(t, "10", got.fields[idx].NumInput.Value())
			return
		}
	}
}

func TestNumInputEnter_SameValue(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if field.IsNumInput {
			mdl.fields[idx].Editing = true
			mdl.fields[idx].LastValue = "42"
			mdl.fields[idx].NumInput.SetValue("42")
			mdl.fields[idx].NumInput.Focus()
			mdl.focusIndex = idx

			newM, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := newM.(model)

			assert.False(t, got.fields[idx].Editing)
			assert.Nil(t, cmd, "should not send AT command when value unchanged")
			return
		}
	}
}

// --- String helper tests ---

func TestStripAnsi(t *testing.T) {
	assert.Equal(t, "hello", stripAnsi("hello"))
	assert.Equal(t, "red", stripAnsi("\x1b[31mred\x1b[0m"))
	assert.Equal(t, "bold green", stripAnsi("\x1b[1m\x1b[32mbold green\x1b[0m"))
	assert.Empty(t, stripAnsi(""))
}

func TestTruncateToWidth(t *testing.T) {
	assert.Equal(t, "hello", truncateToWidth("hello", 10))
	assert.Equal(t, "hel", truncateToWidth("hello", 3))
	assert.Empty(t, truncateToWidth("hello", 0))
	assert.Empty(t, truncateToWidth("hello", -1))
	assert.Equal(t, "hello", truncateToWidth("hello", 5))
	assert.Equal(t, "\x1b[31mhel", truncateToWidth("\x1b[31mhello\x1b[0m", 3))
}

func TestSkipWidth(t *testing.T) {
	assert.Equal(t, "hello", skipWidth("hello", 0))
	assert.Equal(t, "lo", skipWidth("hello", 3))
	assert.Empty(t, skipWidth("hello", 5))
	assert.Empty(t, skipWidth("hello", 10))
	assert.Equal(t, "hello", skipWidth("hello", -1))
}

func TestSkipWidth_WithAnsi(t *testing.T) {
	got := skipWidth("\x1b[31mhello\x1b[0m", 3)
	assert.True(t, strings.HasPrefix(got, "lo"))
}

func TestOverlayString(t *testing.T) {
	assert.Contains(t, overlayString("abcdefghij", "XY", 3), "XY")
	assert.Contains(t, overlayString("abcdefghij", "XY", 0), "XY")
	assert.Contains(t, overlayString("abc", "XY", 5), "XY")
}

// --- View tests ---

func TestView_Renders(t *testing.T) {
	mdl := testModel()
	view := mdl.View()

	assert.Contains(t, view, "LoRa Configurator")
	assert.Contains(t, view, "Connect")
	assert.Contains(t, view, "Restore")
	assert.Contains(t, view, "Reboot")
}

func TestView_Connected(t *testing.T) {
	mdl := connectedModel()
	mdl.version = "2.0.0"
	assert.Contains(t, mdl.View(), "2.0.0")
}

func TestView_Connecting(t *testing.T) {
	mdl := testModel()
	mdl.connecting = true
	assert.Contains(t, mdl.View(), "Connecting")
}

func TestView_WithDropdownOpen(t *testing.T) {
	mdl := connectedModel()
	for idx, field := range mdl.fields {
		if !field.IsNumInput {
			mdl.fields[idx].Open = true
			mdl.fields[idx].Focused = true
			mdl.focusIndex = idx
			break
		}
	}
	assert.NotEmpty(t, mdl.View())
}

func TestView_FocusedButtons(t *testing.T) {
	mdl := connectedModel()

	mdl.focusIndex = focusRestore
	assert.NotPanics(t, func() { mdl.View() })

	mdl.focusIndex = focusReboot
	assert.NotPanics(t, func() { mdl.View() })
}

func TestInit(t *testing.T) {
	mdl := testModel()
	assert.NotNil(t, mdl.Init())
}

// --- Additional navigation edge cases ---

func TestFocusNextInColumn_FromDeviceConnect(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusDevice

	mdl.focusNextInColumn()
	assert.Equal(t, 0, mdl.focusIndex)
}

func TestFocusNextInColumn_FromConnect(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusConnect

	mdl.focusNextInColumn()
	assert.Equal(t, 0, mdl.focusIndex)
}

func TestFocusPrev_Field0ToConnect(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = 0

	mdl.focusPrev()
	assert.Equal(t, focusConnect, mdl.focusIndex)
}

func TestFocusNext_RebootDisconnected(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = focusReboot

	mdl.focusNext()
	assert.Equal(t, focusDevice, mdl.focusIndex)
}

func TestHandleEnter_RestoreNoConn(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusRestore
	mdl.conn = nil

	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestHandleEnter_RebootNoConn(t *testing.T) {
	mdl := connectedModel()
	mdl.focusIndex = focusReboot
	mdl.conn = nil

	_, cmd := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestHandleEnter_DisabledField(t *testing.T) {
	mdl := testModel()
	mdl.focusIndex = 0

	newM, _ := mdl.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := newM.(model)
	assert.False(t, got.fields[0].Open)
}
