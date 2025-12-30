package cli

import "fmt"

type InitCmd struct{}

func (c *InitCmd) Run(ctx *Context) error {
	if err := ctx.Store.Init(); err != nil {
		return err
	}
	fmt.Printf("Initialized daylit storage at: %s\n", ctx.Store.GetConfigPath())
	return nil
}
