package commands

import (
	"github.com/abiosoft/ishell"
)

const rmUsage string = `
usage: rm parameter ...
Remove parameters. Separate multiple parameters with spaces. Parameters may be
absolute or relative.
Example:
/> rm /foo/bar baz
`

func rm(c *ishell.Context) {
	var err error
	if len(c.Args) >= 1 {
		err = ps.Rm(c.Args)
		if err != nil {
			shell.Println("Error: ", err)
		}
	} else {
		shell.Println(rmUsage, err)
	}
	return
}
