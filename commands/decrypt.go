package commands

import (
	"fmt"
	"strconv"
)

const decryptUsage = "decrypt [true|false]"

func decrypt(c *Context) error {
	if len(c.Args) > 1 {
		return fmt.Errorf("usage: %s", decryptUsage)
	}
	if len(c.Args) == 1 {
		v, err := strconv.ParseBool(c.Args[0])
		if err != nil {
			return fmt.Errorf("decrypt requires true or false")
		}
		ps.Decrypt = v
	}
	shell.Println("Decrypt is", ps.Decrypt)
	return nil
}
