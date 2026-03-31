package main

import "fmt"

type Option struct {
	Value   string // raw value sent to AT command
	Display string // human-readable label
}

type ParamDef struct {
	Label     string   // UI label
	ATCmd     string   // AT command name (e.g. "SF", "BW")
	Options   []Option // valid options (for dropdown fields)
	AllpIndex int      // position in ALLP response (0-based)
	// For numeric input fields (instead of dropdown)
	IsNumInput bool // true = text input with range validation
	Min        int  // minimum valid value
	Max        int  // maximum valid value
}

// ALLP order: SF,BW,CR,PWR,NETID,LBT,MODE,TXCH,RXCH,RSSI,ADDR,PORT,COMM,BAUD,KEY
var AllParams = []ParamDef{
	{
		Label: "Spread Factor", ATCmd: "SF", AllpIndex: 0,
		Options: rangeOptions(7, 12, ""),
	},
	{
		Label: "Bandwidth", ATCmd: "BW", AllpIndex: 1,
		Options: []Option{
			{"0", "125KHz"},
			{"1", "250KHz"},
			{"2", "500KHz"},
		},
	},
	{
		Label: "Code Rate", ATCmd: "CR", AllpIndex: 2,
		Options: []Option{
			{"1", "4/5"},
			{"2", "4/6"},
			{"3", "4/7"},
			{"4", "4/8"},
		},
	},
	{
		Label: "RF Power", ATCmd: "PWR", AllpIndex: 3,
		Options: rangeOptions(10, 22, "dBm"),
	},
	{
		Label: "Network ID", ATCmd: "NETID", AllpIndex: 4,
		IsNumInput: true, Min: 0, Max: 255,
	},
	{
		Label: "LBT", ATCmd: "LBT", AllpIndex: 5,
		Options: []Option{
			{"0", "Disabled"},
			{"1", "Enabled"},
		},
	},
	{
		Label: "Mode", ATCmd: "MODE", AllpIndex: 6,
		Options: []Option{
			{"1", "Stream"},
			{"2", "Packet"},
			{"3", "Relay"},
		},
	},
	{
		Label: "TX Channel", ATCmd: "TXCH", AllpIndex: 7,
		IsNumInput: true, Min: 0, Max: 80,
	},
	{
		Label: "RX Channel", ATCmd: "RXCH", AllpIndex: 8,
		IsNumInput: true, Min: 0, Max: 80,
	},
	{
		Label: "RSSI", ATCmd: "RSSI", AllpIndex: 9,
		Options: []Option{
			{"0", "Disabled"},
			{"1", "Enabled"},
		},
	},
	{
		Label: "Address", ATCmd: "ADDR", AllpIndex: 10,
		IsNumInput: true, Min: 0, Max: 65535,
	},
	{
		Label: "Port", ATCmd: "PORT", AllpIndex: 11,
		IsNumInput: true, Min: 0, Max: 65535,
	},
	{
		Label: "Baud Rate", ATCmd: "BAUD", AllpIndex: 13,
		Options: []Option{
			{"1200", "1200"},
			{"2400", "2400"},
			{"4800", "4800"},
			{"9600", "9600"},
			{"19200", "19200"},
			{"38400", "38400"},
			{"57600", "57600"},
			{"115200", "115200"},
		},
	},
	{
		Label: "Serial Cfg", ATCmd: "COMM", AllpIndex: 12,
		Options: []Option{
			{`"8N1"`, "8N1"},
			{`"8N2"`, "8N2"},
			{`"8E1"`, "8E1"},
			{`"8E2"`, "8E2"},
			{`"8O1"`, "8O1"},
			{`"8O2"`, "8O2"},
			{`"9N1"`, "9N1"},
			{`"9N2"`, "9N2"},
		},
	},
	{
		Label: "Key", ATCmd: "KEY", AllpIndex: 14,
		IsNumInput: true, Min: 0, Max: 65535,
	},
}

func rangeOptions(min, max int, suffix string) []Option {
	opts := make([]Option, 0, max-min+1)
	for i := min; i <= max; i++ {
		v := fmt.Sprintf("%d", i)
		d := v
		if suffix != "" {
			d = v + suffix
		}
		opts = append(opts, Option{Value: v, Display: d})
	}
	return opts
}

func FindOptionIndex(opts []Option, value string) int {
	for i, o := range opts {
		if o.Value == value {
			return i
		}
	}
	return 0
}
