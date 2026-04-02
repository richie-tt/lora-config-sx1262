package main

import (
	"fmt"
	"lora-config-SX1262/internal/tui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	Tag       = "dev"
	Commit    = "0000000"
	BuildDate = "00-00-0000 00:00"
)

func main() {
	p := tea.NewProgram(tui.InitialModel(Tag, Commit, BuildDate), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
