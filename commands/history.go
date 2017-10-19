package commands

import (
	"strings"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/ryanuber/columnize"
)

const (
	historyHeader = `Type | Description | Value | Last Modified By | Last Modified Date | Pattern`
	historyUsage  = `
usage: history parameter
Display modification the history of a parameter.
`
)

// history prints the history of a parameter
func history(c *ishell.Context) {
	if len(c.Args) != 1 {
		shell.Println(historyUsage)
		return
	}
	resp, err := ps.GetHistory(c.Args[0])
	if err != nil {
		shell.Println("Error: ", err)
	} else {
		printParameterHistory(resp)
	}
}

func printParameterHistory(history []ssm.ParameterHistory) {
	results := []string{historyHeader}
	var val string
	var row []string
	for _, p := range history {
		if *p.Type == "SecureString" && !ps.Decrypt {
			val = "-"
		} else {
			val = aws.StringValue(p.Value)
		}
		row = []string{
			aws.StringValue(p.Type),
			aws.StringValue(p.Description),
			val,
			aws.StringValue(p.LastModifiedUser),
			p.LastModifiedDate.Format(time.RFC1123),
			aws.StringValue(p.AllowedPattern),
		}
		results = append(results, strings.Join(row, " | "))
	}
	output := columnize.SimpleFormat(results)
	shell.Println(output)
}
