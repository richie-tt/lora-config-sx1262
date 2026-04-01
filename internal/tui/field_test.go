package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeDropdownField() Field {
	return newField(ParamDef{
		Label:     "Test",
		ATCmd:     "TEST",
		AllpIndex: 0,
		Options: []Option{
			{"0", "Zero"},
			{"1", "One"},
			{"2", "Two"},
			{"3", "Three"},
			{"4", "Four"},
		},
	})
}

func makeNumField() Field {
	return newField(ParamDef{
		Label:      "NumTest",
		ATCmd:      "NUM",
		AllpIndex:  1,
		IsNumInput: true,
		Min:        0,
		Max:        255,
	})
}

func TestNewField_Dropdown(t *testing.T) {
	field := makeDropdownField()

	assert.Equal(t, "Test", field.Label)
	assert.Equal(t, "TEST", field.ATCmd)
	assert.False(t, field.IsNumInput)
	assert.Len(t, field.Options, 5)
	assert.Equal(t, 0, field.Selected)
}

func TestNewField_NumInput(t *testing.T) {
	field := makeNumField()

	assert.True(t, field.IsNumInput)
	assert.Equal(t, 0, field.Min)
	assert.Equal(t, 255, field.Max)
	assert.Equal(t, "0", field.NumInput.Value())
}

func TestSelectedValue_Dropdown(t *testing.T) {
	field := makeDropdownField()
	assert.Equal(t, "0", field.SelectedValue())

	field.Selected = 2
	assert.Equal(t, "2", field.SelectedValue())
}

func TestSelectedValue_OutOfBounds(t *testing.T) {
	field := makeDropdownField()

	field.Selected = -1
	assert.Empty(t, field.SelectedValue())

	field.Selected = 100
	assert.Empty(t, field.SelectedValue())
}

func TestSelectedValue_NumInput(t *testing.T) {
	field := makeNumField()
	field.NumInput.SetValue("42")
	assert.Equal(t, "42", field.SelectedValue())
}

func TestSelectedDisplay_Dropdown(t *testing.T) {
	field := makeDropdownField()
	assert.Equal(t, "Zero", field.SelectedDisplay())

	field.Selected = 3
	assert.Equal(t, "Three", field.SelectedDisplay())
}

func TestSelectedDisplay_OutOfBounds(t *testing.T) {
	field := makeDropdownField()
	field.Selected = -1
	assert.Equal(t, "---", field.SelectedDisplay())
}

func TestSelectedDisplay_NumInput(t *testing.T) {
	field := makeNumField()
	field.NumInput.SetValue("123")
	assert.Equal(t, "123", field.SelectedDisplay())
}

func TestSetByValue_Dropdown(t *testing.T) {
	field := makeDropdownField()

	field.SetByValue("3")
	assert.Equal(t, 3, field.Selected)
	assert.Equal(t, "3", field.LastValue)

	// Non-existent value - should not change
	field.SetByValue("99")
	assert.Equal(t, 3, field.Selected)
}

func TestSetByValue_NumInput(t *testing.T) {
	field := makeNumField()
	field.SetByValue("128")
	assert.Equal(t, "128", field.NumInput.Value())
	assert.Equal(t, "128", field.LastValue)
}

func TestValidateNumInput(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		min     int
		max     int
		wantVal string
		wantOK  bool
	}{
		{"valid", "42", 0, 255, "42", true},
		{"min boundary", "0", 0, 255, "0", true},
		{"max boundary", "255", 0, 255, "255", true},
		{"below min", "-1", 0, 255, "", false},
		{"above max", "256", 0, 255, "", false},
		{"empty", "", 0, 255, "", false},
		{"not a number", "abc", 0, 255, "", false},
		{"whitespace around", " 42", 0, 255, "42", true},
		{"leading zeros normalized", "007", 0, 255, "7", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			field := makeNumField()
			field.Min = testCase.min
			field.Max = testCase.max
			field.NumInput.SetValue(testCase.value)

			val, ok := field.ValidateNumInput()
			assert.Equal(t, testCase.wantOK, ok)
			assert.Equal(t, testCase.wantVal, val)
		})
	}
}

func TestMoveUp(t *testing.T) {
	field := makeDropdownField()
	field.Selected = 2

	field.MoveUp()
	assert.Equal(t, 1, field.Selected)

	field.MoveUp()
	assert.Equal(t, 0, field.Selected)

	// Can't go below 0
	field.MoveUp()
	assert.Equal(t, 0, field.Selected)
}

func TestMoveDown(t *testing.T) {
	field := makeDropdownField()
	field.Selected = 3

	field.MoveDown()
	assert.Equal(t, 4, field.Selected)

	// Can't go past last
	field.MoveDown()
	assert.Equal(t, 4, field.Selected)
}

func TestMoveDown_ScrollOffset(t *testing.T) {
	field := makeDropdownField()
	field.maxVisible = 3
	field.Selected = 0
	field.scrollOffset = 0

	field.MoveDown() // 1
	field.MoveDown() // 2
	field.MoveDown() // 3 → should scroll
	assert.Equal(t, 1, field.scrollOffset)
}

func TestMoveUp_ScrollOffset(t *testing.T) {
	field := makeDropdownField()
	field.maxVisible = 3
	field.Selected = 3
	field.scrollOffset = 2

	field.MoveUp() // 2
	field.MoveUp() // 1 → should scroll back
	assert.Equal(t, 1, field.scrollOffset)
}

func TestToggleOpen(t *testing.T) {
	field := makeDropdownField()
	assert.False(t, field.Open)

	field.ToggleOpen()
	assert.True(t, field.Open)

	field.ToggleOpen()
	assert.False(t, field.Open)
}

func TestToggleOpen_ScrollPosition(t *testing.T) {
	field := makeDropdownField()
	field.Selected = 3
	field.maxVisible = 3

	field.ToggleOpen()
	assert.GreaterOrEqual(t, field.scrollOffset, 0)
	assert.LessOrEqual(t, field.scrollOffset, len(field.Options)-field.maxVisible)
}

func TestToggleOpen_FewOptions(t *testing.T) {
	field := newField(ParamDef{
		Label: "Small", ATCmd: "SM", AllpIndex: 0,
		Options: []Option{{"0", "A"}, {"1", "B"}},
	})
	field.Selected = 1

	field.ToggleOpen()
	assert.Equal(t, 0, field.scrollOffset)
}

func TestRenderDropdown(t *testing.T) {
	field := makeDropdownField()
	field.Open = true
	field.Selected = 2
	field.scrollOffset = 0

	dropdown := field.RenderDropdown()
	require.NotEmpty(t, dropdown)
}

func TestRenderDropdown_WithScroll(t *testing.T) {
	field := makeDropdownField()
	field.maxVisible = 2
	field.Open = true
	field.Selected = 3
	field.scrollOffset = 2

	dropdown := field.RenderDropdown()
	assert.Contains(t, dropdown, "↑")
	assert.Contains(t, dropdown, "↓")
}

func TestViewClosed_States(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		field := makeDropdownField()
		field.Disabled = true
		assert.Contains(t, field.ViewClosed(), "---")
	})

	t.Run("focused success", func(t *testing.T) {
		field := makeDropdownField()
		field.Focused = true
		field.Status = StatusSuccess
		assert.NotEmpty(t, field.ViewClosed())
	})

	t.Run("focused error", func(t *testing.T) {
		field := makeDropdownField()
		field.Focused = true
		field.Status = StatusError
		assert.NotEmpty(t, field.ViewClosed())
	})

	t.Run("success without focus", func(t *testing.T) {
		field := makeDropdownField()
		field.Status = StatusSuccess
		assert.NotEmpty(t, field.ViewClosed())
	})

	t.Run("error without focus", func(t *testing.T) {
		field := makeDropdownField()
		field.Status = StatusError
		assert.NotEmpty(t, field.ViewClosed())
	})

	t.Run("num input editing", func(t *testing.T) {
		field := makeNumField()
		field.Focused = true
		field.Editing = true
		assert.NotEmpty(t, field.ViewClosed())
	})

	t.Run("num input focused not editing", func(t *testing.T) {
		field := makeNumField()
		field.Focused = true
		field.Editing = false
		assert.Contains(t, field.ViewClosed(), "✎")
	})

	t.Run("dropdown focused open", func(t *testing.T) {
		field := makeDropdownField()
		field.Focused = true
		field.Open = true
		assert.Contains(t, field.ViewClosed(), "▲")
	})

	t.Run("dropdown focused closed", func(t *testing.T) {
		field := makeDropdownField()
		field.Focused = true
		view := field.ViewClosed()
		stripped := strings.ReplaceAll(view, " ", "")
		assert.Contains(t, stripped, "▼")
	})
}
