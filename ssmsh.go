package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/bwhaley/ssmsh/commands"
	"github.com/bwhaley/ssmsh/parameterstore"
	"github.com/mattn/go-shellwords"
)

var Version string

func main() {
	var version bool
	_fn := flag.String("file", "", "Read commands from file (use - for stdin)")
	flag.BoolVar(&version, "version", false, "Display the current version")
	flag.Parse()

	if version {
		fmt.Println("Version", Version)
		os.Exit(0)
	}

	shell := ishell.New()
	var ps parameterstore.ParameterStore
	err := ps.NewParameterStore()
	if err != nil {
		shell.Println("Error initializing session. Is your authentication correct?", err)
		os.Exit(1)
	}
	commands.Init(shell, &ps)

	fn := *_fn
	if fn == "-" {
		processStdin(shell, fn)
	} else if fn != "" {
		processFile(shell, fn)
	} else if len(flag.Args()) > 1 {
		shell.Process(flag.Args()...)
	} else {
		shell.Run()
	}
	shell.Close()
}

func processStdin(shell *ishell.Shell, fn string) {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		shell.Println("Error reading from stdin:", err)
		os.Exit(1)
	}
	processData(shell, string(data))
}

func processFile(shell *ishell.Shell, fn string) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		shell.Println("Error reading from file:", err)
	}
	processData(shell, string(data))
}

func processData(shell *ishell.Shell, data string) {
	lines := strings.Split(data, "\n")
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
