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
		// Last field in column -> go to buttons
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
		// First field in column -> go to Connect
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
