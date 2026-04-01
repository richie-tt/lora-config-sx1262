package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseALLP(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "valid 15 fields",
			input: `+ALLP=7,0,1,22,0,0,1,10,10,0,0,0,"8N1",9600,0`,
			want: map[string]string{
				"SF": "7", "BW": "0", "CR": "1", "PWR": "22",
				"NETID": "0", "LBT": "0", "MODE": "1",
				"TXCH": "10", "RXCH": "10", "RSSI": "0",
				"ADDR": "0", "PORT": "0", "COMM": `"8N1"`,
				"BAUD": "9600", "KEY": "0",
			},
		},
		{
			name:  "with echo and OK",
			input: "AT+ALLP?\r\n+ALLP=7,0,1,22,0,0,1,10,10,0,0,0,\"8N1\",9600,0\r\nOK",
			want: map[string]string{
				"SF": "7", "BW": "0", "CR": "1", "PWR": "22",
				"NETID": "0", "LBT": "0", "MODE": "1",
				"TXCH": "10", "RXCH": "10", "RSSI": "0",
				"ADDR": "0", "PORT": "0", "COMM": `"8N1"`,
				"BAUD": "9600", "KEY": "0",
			},
		},
		{
			name:    "no ALLP marker",
			input:   "OK",
			wantErr: true,
		},
		{
			name:    "too few fields",
			input:   "+ALLP=7,0,1",
			wantErr: true,
		},
		{
			name:    "empty response",
			input:   "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := parseALLP(testCase.input)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestSplitALLP(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple comma separated",
			input: "7,0,1,22",
			want:  []string{"7", "0", "1", "22"},
		},
		{
			name:  "quoted field with comma inside",
			input: `7,"8N1",9600`,
			want:  []string{"7", `"8N1"`, "9600"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single value",
			input: "42",
			want:  []string{"42"},
		},
		{
			name:  "quoted field with embedded comma",
			input: `"a,b",c`,
			want:  []string{`"a,b"`, "c"},
		},
		{
			name:  "full ALLP response",
			input: `7,0,1,22,0,0,1,10,10,0,0,0,"8N1",9600,0`,
			want:  []string{"7", "0", "1", "22", "0", "0", "1", "10", "10", "0", "0", "0", `"8N1"`, "9600", "0"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := splitALLP(testCase.input)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with VER prefix",
			input: "AT+VER\r\n+VER=1.2.3\r\nOK",
			want:  "1.2.3",
		},
		{
			name:  "VER prefix only",
			input: "+VER=4.5.6",
			want:  "4.5.6",
		},
		{
			name:  "no VER prefix, echoed command stripped",
			input: "AT+VER\r\nSomeVersion\r\nOK",
			want:  "SomeVersion",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "  \r\n  ",
			want:  "",
		},
		{
			name:  "VER with trailing newline",
			input: "+VER=2.0.0\r\n",
			want:  "2.0.0",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, parseVersion(testCase.input))
		})
	}
}
