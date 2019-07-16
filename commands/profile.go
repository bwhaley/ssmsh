package commands

import "github.com/abiosoft/ishell"

const profileUsage string = `
profile name
Switch to the specified profile as listed in the .aws config or credentials file.
`

func profile(c *ishell.Context) {
	if len(c.Args) == 0 {
		if ps.Profile != "" {
			shell.Println(ps.Profile)
		}
	} else if len(c.Args) == 1 {
		ps.Profile = c.Args[0]
		ps.NewParameterStore()
	}
}
