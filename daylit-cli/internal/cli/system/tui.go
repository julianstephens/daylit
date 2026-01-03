package system

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui"
)

type TuiCmd struct{}

func (c *TuiCmd) Run(ctx *cli.Context) error {
	// Perform automatic backup on TUI startup (after successful load)
	ctx.PerformAutomaticBackup()

	p := tea.NewProgram(tui.NewModel(ctx.Store, ctx.Scheduler), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
