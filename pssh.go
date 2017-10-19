package main

import (
	"flag"
	"io/ioutil"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/kountable/pssh/commands"
	"github.com/kountable/pssh/parameterstore"
)

func main() {
	shell := ishell.New()
	ps := parameterstore.NewParameterStore()
	commands.Init(shell, ps)

	_fn := flag.String("file", "", "Read commands from file")
	flag.Parse()
	fn := *_fn
	if fn != "" {
		commandsFromFile(shell, fn)
	} else if len(flag.Args()) > 1 {
		shell.Process(flag.Args()...)
	} else {
		shell.Println("Browse the EC2 Parameter Store")
		shell.Run()
	}
	shell.Close()
}

func commandsFromFile(shell *ishell.Shell, fn string) {
	content, err := ioutil.ReadFile(fn)
	if err != nil {
		shell.Println("Error reading from file.", err)
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		options := strings.Split(line, " ")
		shell.Process(options...)
	}
}
