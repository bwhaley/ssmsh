package commands

import (
	"fmt"

	"github.com/bwhaley/ssmsh/parameterstore"
)

const rmUsage = "rm [-rR] [--dry-run] [region:]path ..."

func rm(c *Context) error {
	args, recursive := checkRecursion(c.Args)
	old := ps.DryRun
	defer func() { ps.DryRun = old }()
	var paths []parameterstore.ParameterPath
	for _, arg := range args {
		if arg == "--dry-run" {
			ps.DryRun = true
			continue
		}
		p, err := parsePath(arg)
		if err != nil {
			return err
		}
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		return fmt.Errorf("usage: %s", rmUsage)
	}
	err := ps.Remove(commandContext, paths, recursive)
	if err == nil && ps.DryRun && !old {
		err = printJSON(ps.Actions)
	}
	return err
}
