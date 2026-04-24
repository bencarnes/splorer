package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/app"
	"github.com/bjcarnes/splorer/internal/shellinit"
)

func main() {
	// Subcommand: `splorer init <shell>` prints the shell wrapper function
	// to stdout. Handled before flag parsing so `init` isn't confused with
	// a flag argument.
	if len(os.Args) >= 2 && os.Args[1] == "init" {
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: splorer init <bash|zsh|powershell>")
			os.Exit(2)
		}
		script, err := shellinit.Script(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(script)
		return
	}

	var cdFile string
	flag.StringVar(&cdFile, "cd-file", "",
		"on exit, write the final navigated directory to this file (used by the shell wrapper)")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine working directory:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app.New(cwd))
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Hand the final navigated directory back to the shell wrapper. A child
	// process can't change its parent shell's cwd directly; the wrapper
	// reads this file and cd's on our behalf.
	if cdFile != "" {
		if am, ok := finalModel.(app.Model); ok {
			if wErr := os.WriteFile(cdFile, []byte(am.CWD()), 0644); wErr != nil {
				fmt.Fprintln(os.Stderr, "cannot write --cd-file:", wErr)
			}
		}
	}
}
