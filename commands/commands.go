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
