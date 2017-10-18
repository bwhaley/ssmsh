package main

import (
	"github.com/abiosoft/ishell"
	"github.com/kountable/pssh/commands"
	"github.com/kountable/pssh/parameterstore"
)

func main() {
	shell := ishell.New()
	shell.Println("Browse the EC2 Parameter Store")

	ps := parameterstore.NewParameterStore()
	commands.Init(shell, ps)

	shell.Run()
	shell.Close()
}
