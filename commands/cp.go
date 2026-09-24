package commands

import (
	"fmt"
	"strings"
)

const cpUsage = "cp [-rR] [--dry-run] [key=DESTINATION_KEY] src dest"

func cp(c *Context) error { return copyOrMove(c, false) }
func copyOrMove(c *Context, move bool) error {
	args, recursive := checkRecursion(c.Args)
	oldDry, oldKey := ps.DryRun, ps.Key
	defer func() { ps.DryRun, ps.Key = oldDry, oldKey }()
	var paths []string
	for _, arg := range args {
		if arg == "--dry-run" {
			ps.DryRun = true
		} else if strings.HasPrefix(arg, "key=") {
			ps.Key = strings.TrimPrefix(arg, "key=")
			if ps.Key == "" {
				return fmt.Errorf("destination key cannot be empty")
			}
		} else {
			paths = append(paths, arg)
		}
	}
	if len(paths) != 2 {
		return fmt.Errorf("expected source and destination; usage: %s", cpUsage)
	}
	src, err := parsePath(paths[0])
	if err != nil {
		return err
	}
	dst, err := parsePath(paths[1])
	if err != nil {
		return err
	}
	if move {
		err = ps.Move(commandContext, src, dst)
	} else {
		err = ps.Copy(commandContext, src, dst, recursive)
	}
	if err == nil && ps.DryRun && !oldDry {
		err = printJSON(ps.Actions)
	}
	return err
}
