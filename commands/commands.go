package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bwhaley/ssmsh/config"
	"github.com/bwhaley/ssmsh/parameterstore"
)

type Context struct {
	Args []string
}

type command struct {
	help    string
	usage   string
	handler func(*Context) error
}

type console struct {
	out      io.Writer
	readLine func() (string, error)
	err      error
}

func (c *console) Print(values ...any) {
	if c.err == nil {
		_, c.err = fmt.Fprint(c.out, values...)
	}
}

func (c *console) Println(values ...any) {
	if c.err == nil {
		_, c.err = fmt.Fprintln(c.out, values...)
	}
}

func (c *console) Printf(format string, values ...any) {
	if c.err == nil {
		_, c.err = fmt.Fprintf(c.out, format, values...)
	}
}

var (
	ErrExit = errors.New("exit requested")

	shell          *console
	ps             *parameterstore.ParameterStore
	cfg            *config.Config
	commandContext context.Context
	handlers       map[string]command
	Timeout                  = 2 * time.Minute
	SecretInput    io.Reader = os.Stdin
	// BatchInput prevents value-stdin from consuming the command stream.
	BatchInput  bool
	Interactive bool
)

func Init(out io.Writer, readLine func() (string, error), store *parameterstore.ParameterStore, configuration *config.Config) {
	shell = &console{out: out, readLine: readLine}
	ps, cfg = store, configuration
	handlers = make(map[string]command)
	policies = make(map[string]parameterPolicies)
	registerCommand("cd", "change parameter directory", cd, cdUsage)
	registerCommand("cp", "copy parameters", cp, cpUsage)
	registerCommand("decrypt", "set parameter decryption", decrypt, decryptUsage)
	registerCommand("exit", "exit the interactive shell", exit, "exit")
	registerCommand("get", "get parameters", get, getUsage)
	registerCommand("history", "get parameter history", history, historyUsage)
	registerCommand("key", "set the destination KMS key", key, keyUsage)
	registerCommand("ls", "list parameters", ls, lsUsage)
	registerCommand("mv", "move parameters", mv, mvUsage)
	registerCommand("policy", "create a named parameter policy", policy, policyUsage)
	registerCommand("profile", "switch AWS profile", profile, profileUsage)
	registerCommand("put", "set a parameter", put, putUsage)
	registerCommand("region", "switch AWS region", region, regionUsage)
	registerCommand("rm", "remove parameters", rm, rmUsage)
}

func exit(c *Context) error {
	if len(c.Args) != 0 {
		return fmt.Errorf("usage: exit")
	}
	return ErrExit
}

func registerCommand(name, help string, handler func(*Context) error, usage string) {
	handlers[name] = command{help: help, usage: usage, handler: handler}
}

// Execute is shared by the interactive shell, inline commands, and batch files.
func Execute(ctx context.Context, args []string) error {
	shell.err = nil
	if len(args) == 0 {
		return nil
	}
	if args[0] == "help" {
		return printHelp(args[1:])
	}
	entry, ok := handlers[args[0]]
	if !ok {
		return fmt.Errorf("unknown command %q", args[0])
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	commandContext = ctx
	ps.Actions = nil
	if err := entry.handler(&Context{Args: args[1:]}); err != nil {
		return err
	}
	if shell.err != nil {
		return shell.err
	}
	if ps.DryRun && len(ps.Actions) > 0 {
		return printJSON(ps.Actions)
	}
	return nil
}

func printHelp(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: help [command]")
	}
	if len(args) == 1 {
		entry, ok := handlers[args[0]]
		if !ok {
			return fmt.Errorf("unknown command %q", args[0])
		}
		shell.Println(entry.usage)
		return shell.err
	}
	names := make([]string, 0, len(handlers))
	for name := range handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	shell.Println("Commands:")
	for _, name := range names {
		shell.Printf("%-12s %s\n", name, handlers[name].help)
	}
	return shell.err
}

func Prompt() string {
	profile := ps.Profile
	if profile == "" {
		profile = "default"
	}
	return fmt.Sprintf("[%s@%s] %s> ", profile, ps.Region, ps.Cwd)
}

func checkRecursion(args []string) ([]string, bool) {
	var paths []string
	recursive := false
	for _, arg := range args {
		if strings.EqualFold(arg, "-r") {
			recursive = true
		} else {
			paths = append(paths, arg)
		}
	}
	return paths, recursive
}

func parsePath(value string) (parameterstore.ParameterPath, error) {
	if value == "" {
		return parameterstore.ParameterPath{}, fmt.Errorf("empty parameter path")
	}
	parts := strings.Split(value, ":")
	if len(parts) > 2 || (len(parts) == 2 && (parts[0] == "" || parts[1] == "")) {
		return parameterstore.ParameterPath{}, fmt.Errorf("invalid path %q; use [region:]path", value)
	}
	p := parameterstore.ParameterPath{Name: parts[0], Region: ps.Region}
	if len(parts) == 2 {
		p.Region, p.Name = parts[0], parts[1]
	}
	return p, nil
}

func trim(values []string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = strings.TrimSpace(value)
	}
	return result
}

func printJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return err
	}
	shell.Println(string(data))
	return shell.err
}
