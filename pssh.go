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
	var ps parameterstore.ParameterStore
	ps.NewParameterStore()
	commands.Init(shell, &ps)

	_fn := flag.String("file", "", "Read commands from file")
	flag.Parse()

	fn := *_fn
	if fn != "" {
		processFromFile(shell, fn)
	} else if len(flag.Args()) > 1 {
		shell.Process(flag.Args()...)
	} else {
		shell.Run()
	}
	shell.Close()
}

func processFromFile(shell *ishell.Shell, fn string) {
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
