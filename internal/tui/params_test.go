package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRangeOptions(t *testing.T) {
	tests := []struct {
		name   string
		min    int
		max    int
		suffix string
		want   int
	}{
		{"SF range", 7, 12, "", 6},
		{"power range", 10, 22, "dBm", 13},
		{"single value", 5, 5, "", 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			opts := rangeOptions(testCase.min, testCase.max, testCase.suffix)
			require.Len(t, opts, testCase.want)

			assert.Equal(t, fmt.Sprintf("%d", testCase.min), opts[0].Value)
			assert.Equal(t, fmt.Sprintf("%d", testCase.max), opts[len(opts)-1].Value)

			for _, opt := range opts {
				if testCase.suffix != "" {
					assert.Equal(t, opt.Value+testCase.suffix, opt.Display)
				} else {
					assert.Equal(t, opt.Value, opt.Display)
				}
			}
		})
	}
}

func TestAllParamsConsistency(t *testing.T) {
	require.NotEmpty(t, allParams)

	seenATCmd := make(map[string]bool)
	seenAllpIndex := make(map[int]bool)

	for idx, param := range allParams {
		assert.NotEmpty(t, param.Label, "param[%d]: empty label", idx)
		assert.NotEmpty(t, param.ATCmd, "param[%d]: empty ATCmd", idx)

		assert.False(t, seenATCmd[param.ATCmd], "param[%d]: duplicate ATCmd %q", idx, param.ATCmd)
		seenATCmd[param.ATCmd] = true

		assert.False(t, seenAllpIndex[param.AllpIndex], "param[%d]: duplicate AllpIndex %d", idx, param.AllpIndex)
		seenAllpIndex[param.AllpIndex] = true

		if param.IsNumInput {
			assert.LessOrEqual(t, param.Min, param.Max, "param[%d] %s", idx, param.ATCmd)
		} else {
			assert.NotEmpty(t, param.Options, "param[%d] %s: dropdown with no options", idx, param.ATCmd)
		}
	}
}
