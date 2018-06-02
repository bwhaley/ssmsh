package commands

import (
	"github.com/abiosoft/ishell"
	"github.com/kountable/ssmsh/parameterstore"
)

const getUsage string = `
get usage: get parameter ...
Get one or more parameters.
`

// Get parameters
func get(c *ishell.Context) {
	if len(c.Args) >= 1 {
		var params []parameterstore.ParameterPath
		for _, p := range c.Args {
			params = append(params, parsePath(p))
		}
		paramsByRegion := groupByRegion(params)
		for region, params := range paramsByRegion {
			resp, err := ps.Get(params, region)
			if err != nil {
				shell.Println("Error: ", err)
			} else {
				if len(resp) >= 1 {
					shell.Printf("%+v\n", resp)
				}
			}
		}
	} else {
		shell.Println(getUsage)
	}
}
