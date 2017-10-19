package commands

import (
	"sort"

	"github.com/abiosoft/ishell"
)

const lsUsage string = `
ls [-r] path ...
Print the parameters in one or more paths.
-r List parameters recursively
`

func ls(c *ishell.Context) {
	var err error
	var pathList []string
	paths := c.Args
	for i, p := range paths {
		if p == "-R" {
			ps.Recurse = true
			defer func() {
				ps.Recurse = false // recursive listing is per invocation
			}()
			paths = remove(paths, i)
		}
	}
	// If no paths were provided, list the current directory
	if len(paths) == 0 {
		paths = append(paths, ps.Cwd)
	}
	for _, p := range paths {
		pathList, err = ps.List(p)
		if err != nil {
			shell.Println("Error: ", err)
			return
		}
		if len(paths) > 1 && len(pathList) != 0 {
			shell.Println(p + ":")
		}
		sort.Strings(pathList)
		for _, r := range pathList {
			shell.Println(r)
		}
	}
}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}
