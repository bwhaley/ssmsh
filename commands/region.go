package commands

import (
	"github.com/abiosoft/ishell"
)

const regionUsage string = `
usage: region region
Update your region.
Example:
region us-west-2
`

func region(c *ishell.Context) {
	if len(c.Args) == 0 {
		if ps.Region != "" {
			shell.Println(ps.Region)
		}
	} else if len(c.Args) == 1 {
		ps.Region = c.Args[0]
		err := ps.NewParameterStore()
		if err != nil {
			shell.Printf("Error: %s", err)
		}
	}
}
