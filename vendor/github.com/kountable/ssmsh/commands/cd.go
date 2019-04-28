package commands

import (
	"github.com/abiosoft/ishell"
)

const cdUsage string = `
usage: cd path
Change your working directory within the parameter store.
Example:
/>cd /foo
/foo>
`

func cd(c *ishell.Context) {
	var err error
	if len(c.Args) == 0 {
		// noop
	} else if len(c.Args) == 1 {
		path := c.Args[0]
		err = ps.SetCwd(parsePath(path))
		if err != nil {
			shell.Println("Error:", err)
		} else {
			setPrompt(ps.Cwd)
		}
	} else {
		shell.Println("Incorrect number of arguments to cd command")
	}
}
