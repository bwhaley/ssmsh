package commands

import (
	"github.com/abiosoft/ishell"
	"github.com/kountable/ssmsh/parameterstore"
)

const rmUsage string = `
usage: rm -[r|R] parameter ...
Remove parameters. Separate multiple parameters with spaces. Parameters may be
absolute or relative.
-[r|R] Remove parameters recursively
Example usage:
/> rm /foo/bar /baz
/> rm -R /foo/
`

func rm(c *ishell.Context) {
	var err error
	var parameterPaths []parameterstore.ParameterPath
	paths, recurse := checkRecursion(c.Args)
	if len(paths) >= 1 {
		for _, p := range paths {
			parameterPaths = append(parameterPaths, parsePath(p))
		}
		err = ps.Remove(parameterPaths, recurse)
		if err != nil {
			shell.Println("Error: ", err)
		}
	} else {
		shell.Println(rmUsage, err)
	}
	return
}
