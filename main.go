package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/app"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine working directory:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app.New(cwd))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
