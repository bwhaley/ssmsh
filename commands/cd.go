package commands

import (
	"fmt"
)

const cdUsage = "cd [region:]path"

func cd(c *Context) error {
	if len(c.Args) == 0 {
		return nil
	}
	if len(c.Args) != 1 {
		return fmt.Errorf("usage: %s", cdUsage)
	}
	p, err := parsePath(c.Args[0])
	if err != nil {
		return err
	}
	return ps.SetCwd(commandContext, p)
}
