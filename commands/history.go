package commands

import (
	"github.com/abiosoft/ishell"
)

const (
	historyUsage = `
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
	resp, err := ps.GetHistory(parsePath(c.Args[0]))
	if err != nil {
		shell.Println("Error: ", err)
	} else {
		printResult(resp)
	}
}
