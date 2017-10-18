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
	//var err error
	shell.Println("Not yet implemented")
	/*
		if len(c.Args) == 2 {
			src := c.Args[0]
			dest := c.Args[1]
			err = ps.Cp(src, dest)
		} else if len(c.Args) == 3 {
			switch c.Args[0] {
			case "-r":
				err = recursiveCopy(c.Args[1], c.Args[2])
			default:
				shell.Println(cpUsage, err)
			}
		} else {
			shell.Println(cpUsage, err)
		}
		if err != nil {
			shell.Println("Error: ", err)
		}
	*/
}

func recursiveCopy(src string, dest string) error {
	return nil
}
