package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/local/swarm/dashboard/internal/tui"
)

func main() {
	if _, err := tea.NewProgram(tui.NewModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running dashboard:", err)
		os.Exit(1)
	}
}
