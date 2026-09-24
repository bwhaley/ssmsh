package commands

import (
	"fmt"
)

const profileUsage = "profile [name]"

func profile(c *Context) error {
	if len(c.Args) == 0 {
		p := ps.Profile
		if p == "" {
			p = "default"
		}
		shell.Println(p)
		return nil
	}
	if len(c.Args) != 1 {
		return fmt.Errorf("usage: %s", profileUsage)
	}
	return ps.Switch(commandContext, ps.Region, c.Args[0])
}
