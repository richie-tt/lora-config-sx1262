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
