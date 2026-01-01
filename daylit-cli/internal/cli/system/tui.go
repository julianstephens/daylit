package system

import (
	"fmt"
	"os"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui"
)

type TuiCmd struct{}

func (c *TuiCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Perform automatic backup on TUI startup (after successful load)
	ctx.PerformAutomaticBackup()

	p := tea.NewProgram(tui.NewModel(ctx.Store, ctx.Scheduler), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	return nil
}
