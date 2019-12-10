package commands

import (
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/bwhaley/ssmsh/parameterstore"
)

type fn func(*ishell.Context)

var shell *ishell.Shell
var ps *parameterstore.ParameterStore

// Init initializes the ssmsh subcommands
func Init(iShell *ishell.Shell, iPs *parameterstore.ParameterStore) {
	shell = iShell
	ps = iPs
	registerCommand("cd", "change your relative location within the parameter store", cd, cdUsage)
	registerCommand("cp", "copy source to dest", cp, cpUsage)
	registerCommand("decrypt", "toggle parameter decryption", decrypt, decryptUsage)
	registerCommand("get", "get parameters", get, getUsage)
	registerCommand("history", "get parameter history", history, historyUsage)
	registerCommand("key", "set the KMS key", key, keyUsage)
	registerCommand("ls", "list parameters", ls, lsUsage)
	registerCommand("mv", "move parameters", mv, mvUsage)
	registerCommand("policy", "create named parameter policy", policy, policyUsage)
	registerCommand("profile", "switch to a different AWS IAM profile", profile, profileUsage)
	registerCommand("put", "set parameter", put, putUsage)
	registerCommand("region", "change region", region, regionUsage)
	registerCommand("rm", "remove parameters", rm, rmUsage)
	setPrompt(parameterstore.Delimiter)
}

// registerCommand adds a command to the shell
func registerCommand(name string, helpText string, f fn, usageText string) {
	shell.AddCmd(&ishell.Cmd{
		Name:     name,
		Help:     helpText,
		LongHelp: usageText,
		Func:     f,
	})
}

// setPrompt configures the shell prompt
func setPrompt(prompt string) {
	shell.SetPrompt(prompt + ">")
}

// remove deletes an element from a slice of strings
func remove(slice []string, i int) []string {
	return append(slice[:i], slice[i+1:]...)
}

// checkRecursion searches a slice of strings for an element matching -r or -R
func checkRecursion(paths []string) ([]string, bool) {
	for i, p := range paths {
		if strings.EqualFold(p, "-r") {
			paths = remove(paths, i)
			return paths, true
		}
	}
	return paths, false
}

// parsePath determines whether a path includes a region
func parsePath(path string) (parameterPath parameterstore.ParameterPath) {
	var pathParts []string
	pathParts = strings.Split(path, ":")
	switch len(pathParts) {
	case 1:
		parameterPath.Name = pathParts[0]
		parameterPath.Region = ps.Region
	case 2:
		parameterPath.Region = pathParts[0]
		parameterPath.Name = pathParts[1]
	}
	ps.InitClient(parameterPath.Region)
	return parameterPath
}

func groupByRegion(params []parameterstore.ParameterPath) map[string][]string {
	paramsByRegion := make(map[string][]string)
	for _, p := range params {
		paramsByRegion[p.Region] = append(paramsByRegion[p.Region], p.Name)
	}
	return paramsByRegion
}

func trim(with []string) (without []string) {
	for i := range with {
		without = append(without, strings.TrimSpace(with[i]))
	}
	return without
}
