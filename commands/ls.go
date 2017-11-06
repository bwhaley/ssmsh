package commands

import (
	"sort"

	"github.com/abiosoft/ishell"
)

const lsUsage string = `
ls -[r|R] path ...
Print the parameters in one or more paths.
-[r|R] List parameters recursively
`

func ls(c *ishell.Context) {
	var err error
	var pathList []string
	paths, recurse := checkRecursion(c.Args)
	// If no paths were provided, list the current directory
	if len(paths) == 0 {
		paths = append(paths, ps.Cwd)
	}
	for _, p := range paths {
		pathList, err = ps.List(p, recurse)
		if err != nil {
			shell.Println("Error: ", err)
			return
		}
		if len(paths) > 1 && len(pathList) != 0 {
			shell.Println(p + ":")
		}
		sort.Strings(pathList)
		for _, r := range pathList {
			shell.Printf("%+s\n", r)
		}
	}
}
