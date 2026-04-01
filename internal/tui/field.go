package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// FieldStatus represents the visual state of a field after an operation.
type FieldStatus int

const (
	StatusNormal FieldStatus = iota
	StatusSuccess
	StatusError
)

// Field represents a single configurable parameter in the TUI.
type Field struct {
	Label     string
	Options   []Option
	Selected  int
	Status    FieldStatus
	Focused   bool
	Open      bool
	Disabled  bool
	ATCmd     string
	AllpIndex int

	scrollOffset int
	maxVisible   int

	IsNumInput bool
	Min        int
	Max        int
	NumInput   textinput.Model
	Editing    bool
	LastValue  string
}

func newField(param ParamDef) Field {
	field := Field{
		Label:      param.Label,
		Options:    param.Options,
		ATCmd:      param.ATCmd,
		AllpIndex:  param.AllpIndex,
		maxVisible: 8,
		IsNumInput: param.IsNumInput,
		Min:        param.Min,
		Max:        param.Max,
	}
	if param.IsNumInput {
		input := textinput.New()
		input.Width = 12
		input.CharLimit = len(fmt.Sprintf("%d", param.Max))
		input.Placeholder = fmt.Sprintf("%d-%d", param.Min, param.Max)
		input.SetValue("0")
		field.NumInput = input
	}
	return field
}

func (f *Field) SelectedValue() string {
	if f.IsNumInput {
		return f.NumInput.Value()
	}
	if f.Selected >= 0 && f.Selected < len(f.Options) {
		return f.Options[f.Selected].Value
	}
	return ""
}

func (f *Field) SelectedDisplay() string {
	if f.IsNumInput {
		return f.NumInput.Value()
	}
	if f.Selected >= 0 && f.Selected < len(f.Options) {
		return f.Options[f.Selected].Display
	}
	return "---"
}

func (f *Field) SetByValue(value string) {
	if f.IsNumInput {
		f.NumInput.SetValue(value)
		f.LastValue = value
		return
	}
	for i, o := range f.Options {
		if o.Value == value {
			f.Selected = i
			f.LastValue = value
			return
		}
	}
}

func (f *Field) ValidateNumInput() (string, bool) {
	val := strings.TrimSpace(f.NumInput.Value())
	if val == "" {
		return "", false
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		return "", false
	}
	if num < f.Min || num > f.Max {
		return "", false
	}
	return fmt.Sprintf("%d", num), true
}

func (f *Field) MoveUp() {
	if f.Selected > 0 {
		f.Selected--
		if f.Selected < f.scrollOffset {
			f.scrollOffset = f.Selected
		}
	}
}

func (f *Field) MoveDown() {
	if f.Selected < len(f.Options)-1 {
		f.Selected++
		if f.Selected >= f.scrollOffset+f.maxVisible {
			f.scrollOffset = f.Selected - f.maxVisible + 1
		}
	}
}

func (f *Field) ToggleOpen() {
	f.Open = !f.Open
	if f.Open {
		f.scrollOffset = f.Selected - f.maxVisible/2
		if f.scrollOffset < 0 {
			f.scrollOffset = 0
		}
		maxOff := len(f.Options) - f.maxVisible
		if maxOff < 0 {
			maxOff = 0
		}
		if f.scrollOffset > maxOff {
			f.scrollOffset = maxOff
		}
	}
}

var (
	labelStyle = lipgloss.NewStyle().
			Width(15).
			Align(lipgloss.Right).
			PaddingRight(1)

	fieldNormalStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Width(16).
				Padding(0, 1)

	fieldFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Width(16).
				Padding(0, 1)

	fieldSuccessStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("2")).
				Width(16).
				Padding(0, 1)

	fieldErrorStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("1")).
			Width(16).
			Padding(0, 1)

	// Focused + status: purple border with colored text
	fieldFocusedSuccessStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("63")).
					Foreground(lipgloss.Color("2")).
					Width(16).
					Padding(0, 1)

	fieldFocusedErrorStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Foreground(lipgloss.Color("1")).
				Width(16).
				Padding(0, 1)

	fieldDisabledStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Width(16).
				Padding(0, 1).
				Foreground(lipgloss.Color("238"))

	dropdownStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Width(18)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("63")).
				Width(16)

	normalItemStyle = lipgloss.NewStyle().
			Width(16)
)

func (f *Field) ViewClosed() string {
	label := labelStyle.Render(f.Label)

	var style lipgloss.Style
	var display string

	switch {
	case f.Disabled:
		style = fieldDisabledStyle
	case f.IsNumInput && f.Editing:
		style = fieldFocusedStyle
		return lipgloss.JoinHorizontal(lipgloss.Center, label, style.Render(f.NumInput.View()))
	case f.Focused && f.Status == StatusSuccess:
		style = fieldFocusedSuccessStyle
	case f.Focused && f.Status == StatusError:
		style = fieldFocusedErrorStyle
	case f.Focused:
		style = fieldFocusedStyle
	case f.Status == StatusSuccess:
		style = fieldSuccessStyle
	case f.Status == StatusError:
		style = fieldErrorStyle
	default:
		style = fieldNormalStyle
	}

	if f.IsNumInput {
		display = f.NumInput.Value()
		if f.Focused && !f.Editing {
			display += " ✎"
		}
	} else {
		display = f.SelectedDisplay()
		if f.Focused && f.Open {
			display += " ▲"
		} else {
			display += " ▼"
		}
	}

	if f.Disabled {
		display = "---"
	}

	field := style.Render(display)
	return lipgloss.JoinHorizontal(lipgloss.Center, label, field)
}

func (f *Field) RenderDropdown() string {
	visible := f.maxVisible
	if len(f.Options) < visible {
		visible = len(f.Options)
	}

	var items strings.Builder
	end := f.scrollOffset + visible
	if end > len(f.Options) {
		end = len(f.Options)
	}

	for idx := f.scrollOffset; idx < end; idx++ {
		opt := f.Options[idx]
		line := fmt.Sprintf(" %s", opt.Display)
		if idx == f.Selected {
			items.WriteString(selectedItemStyle.Render(line))
		} else {
			items.WriteString(normalItemStyle.Render(line))
		}
		if idx < end-1 {
			items.WriteString("\n")
		}
	}

	prefix := ""
	suffix := ""
	if f.scrollOffset > 0 {
		prefix = "  ↑\n"
	}
	if end < len(f.Options) {
		suffix = "\n  ↓"
	}

	return dropdownStyle.Render(prefix + items.String() + suffix)
}
