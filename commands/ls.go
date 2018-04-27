package commands

import (
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/abiosoft/ishell"
	"github.com/kountable/ssmsh/parameterstore"
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
	if ps.Cwd == parameterstore.Delimiter {
		shell.Println("Warning: Listing a large number of parameters may take a long time.")
		shell.Println("Press ^C to interrupt.")
	}
	for _, p := range paths {
		pathList, err = list(p, recurse)
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

func list(path string, recurse bool) ([]string, error) {
	ch := make(chan parameterstore.ListResult, 0)

	go func() {
		ps.List(path, recurse, ch)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		<-sigs
	}()

	select {
	case result := <-ch:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Result, nil
	case <-sigs:
		return nil, nil
	}
}
