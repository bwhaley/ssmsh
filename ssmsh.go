package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwhaley/ssmsh/commands"
	"github.com/bwhaley/ssmsh/config"
	"github.com/bwhaley/ssmsh/parameterstore"
	"github.com/mattn/go-shellwords"
	"github.com/reeflective/readline"
)

var Version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("ssmsh", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configFile := flags.String("config", "", "Configuration file (default ~/.ssmshrc)")
	file := flags.String("file", "", "Read commands from file (- for stdin)")
	version := flags.Bool("version", false, "Print version")
	output := flags.String("output", "", "Output: text, json, or value")
	timeout := flags.Duration("timeout", 2*time.Minute, "Timeout per command")
	dryRun := flags.Bool("dry-run", false, "Print mutation plans without writing")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *version {
		if _, err := fmt.Fprintln(stdout, "Version", Version); err != nil {
			return 1
		}
		return 0
	}
	if *timeout <= 0 {
		_, _ = fmt.Fprintln(stderr, "timeout must be positive")
		return 2
	}
	if *file != "" && len(flags.Args()) > 0 {
		_, _ = fmt.Fprintln(stderr, "use either -file or an inline command")
		return 2
	}
	cfg, err := config.ReadConfig(*configFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if *output != "" {
		cfg.Default.Output = *output
	}
	switch cfg.Default.Output {
	case "", "text", "json", "value":
	default:
		_, _ = fmt.Fprintln(stderr, "output must be text, json, or value")
		return 2
	}
	var ps parameterstore.ParameterStore
	ps.SetDefaults(cfg)
	ps.DryRun = *dryRun
	// AWS clients load lazily so local commands and help work without credentials.
	commands.SecretInput = stdin
	commands.BatchInput = *file == "-"
	commands.Timeout = *timeout
	commands.Interactive = *file == "" && len(flags.Args()) == 0
	if commands.Interactive {
		return runInteractive(stdout, stderr, &ps, &cfg)
	}
	commands.Init(stdout, nil, &ps, &cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	execute := func(args []string) error { return commands.Execute(ctx, args) }
	if *file == "-" {
		err = processData(stdin, execute)
	} else if *file != "" {
		var input *os.File
		input, err = os.Open(*file)
		if err == nil {
			defer func() { _ = input.Close() }()
			err = processData(input, execute)
		}
	} else {
		err = execute(flags.Args())
	}
	if err != nil {
		if errors.Is(err, commands.ErrExit) {
			return 0
		}
		_, _ = fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	return 0
}

func runInteractive(stdout, stderr io.Writer, ps *parameterstore.ParameterStore, cfg *config.Config) int {
	shell := readline.NewShell()
	shell.Prompt.Primary(commands.Prompt)
	commands.Init(stdout, shell.Readline, ps, cfg)
	for {
		line, err := shell.Readline()
		if errors.Is(err, io.EOF) {
			return 0
		}
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		args, err := shellwords.Parse(line)
		if err == nil {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			err = commands.Execute(ctx, args)
			stop()
		}
		if errors.Is(err, commands.ErrExit) {
			return 0
		}
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
		}
	}
}
func processData(input io.Reader, execute func([]string) error) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		args, err := shellwords.Parse(line)
		// Never echo a command line: it may contain a secret value.
		if err != nil {
			return fmt.Errorf("line %d: invalid command syntax", lineNumber)
		}
		if err := execute(args); err != nil {
			return fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	return scanner.Err()
}
