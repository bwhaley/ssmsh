package commands

import (
	"github.com/abiosoft/ishell"
)

const getUsage string = `
get usage: get parameter ...
Get one or more parameters.
`

// Get parameters
func get(c *ishell.Context) {
	if len(c.Args) >= 1 {
		resp, err := ps.Get(c.Args)
		if err != nil {
			shell.Println("Error: ", err)
		} else {
			shell.Printf("%+v\n", resp)
		}
	} else {
		shell.Println(getUsage)
	}
}
