package main

import (
	"github.com/kountable/pssh/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		cmd.ExitWithError(cmd.ExitError, err)
	}

	// TODO: Run prompt mode here... on or ROOT command
}
