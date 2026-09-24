package commands

import (
	"fmt"
)

const historyUsage = "history [region:]parameter"

func history(c *Context) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("usage: %s", historyUsage)
	}
	p, err := parsePath(c.Args[0])
	if err != nil {
		return err
	}
	out, err := ps.GetHistory(commandContext, p)
	if err != nil {
		return err
	}
	return printHistory(out)
}
