package commands

import (
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/kountable/ssmsh/parameterstore"
)

type fn func(*ishell.Context)

var shell *ishell.Shell
var ps *parameterstore.ParameterStore

// Init initializes the ssmsh subcommands
func Init(_shell *ishell.Shell, _ps *parameterstore.ParameterStore) {
	shell = _shell
	ps = _ps
	registerCommand("cd", "change your relative location within the parameter store", cd, cdUsage)
	registerCommand("cp", "copy source to dest", cp, cpUsage)
	registerCommand("decrypt", "toggle parameter decryption", decrypt, decryptUsage)
	registerCommand("get", "get parameters", get, getUsage)
	registerCommand("history", "get parameter history", history, historyUsage)
	registerCommand("ls", "list parameters", ls, lsUsage)
	registerCommand("mv", "move parameters", mv, mvUsage)
	registerCommand("put", "set parameter", put, putUsage)
	registerCommand("region", "change region", region, regionUsage)
	registerCommand("rm", "remove parameters", rm, rmUsage)
	setPrompt(parameterstore.Delimiter)
}

func registerCommand(name string, helpText string, f fn, usageText string) {
	shell.AddCmd(&ishell.Cmd{
		Name:     name,
		Help:     helpText,
		LongHelp: usageText,
		Func:     f,
	})
}

func setPrompt(prompt string) {
	shell.SetPrompt(prompt + ">")
}

func remove(slice []string, i int) []string {
	return append(slice[:i], slice[i+1:]...)
}

func checkRecursion(paths []string) ([]string, bool) {
	for i, p := range paths {
		if strings.EqualFold(p, "-r") {
			paths = remove(paths, i)
			return paths, true
		}
	}
	return paths, false
}

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
