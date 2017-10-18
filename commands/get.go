package commands

import (
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/ryanuber/columnize"
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
			printParameters(resp)
		}
	} else {
		shell.Println(getUsage)
	}
}

func printParameters(paramList []ssm.Parameter) {
	results := []string{"Parameter | Type | Value"}
	var val string
	for _, p := range paramList {
		if *p.Type == "SecureString" && !ps.Decrypt {
			val = "-"
		} else {
			val = *p.Value
		}
		results = append(results, strings.Join([]string{*p.Name, *p.Type, val}, " | "))
	}
	output := columnize.SimpleFormat(results)
	shell.Println(output)
}
