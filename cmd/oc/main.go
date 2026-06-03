package main

import (
	"fmt"
	"oc/internal/tui"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	m := tui.IntialModel()
	p := tea.NewProgram(m)
	tui.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
