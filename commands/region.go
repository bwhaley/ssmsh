package commands

import (
	"fmt"
)

const regionUsage = "region [name]"

func region(c *Context) error {
	if len(c.Args) == 0 {
		shell.Println(ps.Region)
		return nil
	}
	if len(c.Args) != 1 {
		return fmt.Errorf("usage: %s", regionUsage)
	}
	return ps.Switch(commandContext, c.Args[0], ps.Profile)
}
