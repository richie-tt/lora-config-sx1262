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
