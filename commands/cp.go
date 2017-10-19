package commands

import (
	"github.com/abiosoft/ishell"
)

const cpUsage string = `
cp usage: cp [-rR] src dest
Copy a parameter from src to dest.
  -r Copy parameters recursively
`

func cp(c *ishell.Context) {
	paths := c.Args
	for i, p := range paths {
		if p == "-R" {
			if !ps.Recurse {
				ps.Recurse = true
				defer func() {
					ps.Recurse = false // recursive listing is per invocation
				}()
			}
			paths = remove(paths, i)
		}
	}
	err := ps.Copy(c.Args[0], c.Args[1])
	if err != nil {
		shell.Println(err)
	}
}
