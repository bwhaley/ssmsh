package commands

import (
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/abiosoft/ishell"
	"github.com/bwhaley/ssmsh/parameterstore"
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)

	quit := make(chan bool)
	lr := make(chan parameterstore.ListResult, 0)
	go func() {
		parameterPath := parsePath(path)
		ps.List(parameterPath, recurse, lr, quit)
	}()

	select {
	case result := <-lr:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Result, nil
	case <-sigs:
		quit <- true
		signal.Stop(sigs)
		close(sigs)
		close(lr)
		return nil, nil
	}
}
