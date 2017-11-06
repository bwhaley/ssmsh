package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/kountable/pssh/commands"
	"github.com/kountable/pssh/parameterstore"
	"github.com/mattn/go-shellwords"
)

func main() {
	shell := ishell.New()
	var ps parameterstore.ParameterStore
	err := ps.NewParameterStore()
	if err != nil {
		shell.Println("Error initializing session. Is your authentication correct?", err)
		os.Exit(1)
	}
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
		if line == "" || string(line[0]) == "#" {
			continue
		}
		args, err := shellwords.Parse(line)
		if err != nil {
			msg := fmt.Errorf("Error parsing %s: %v", line, err)
			shell.Println(msg)
			os.Exit(1)
		}
		err = shell.Process(args...)
		if err != nil {
			msg := fmt.Errorf("Error executing %s: %v", line, err)
			shell.Println(msg)
			os.Exit(1)
		}
	}
}
