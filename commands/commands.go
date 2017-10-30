package commands

import (
	"github.com/abiosoft/ishell"
	"github.com/kountable/pssh/parameterstore"
)

type fn func(*ishell.Context)

var shell *ishell.Shell
var ps *parameterstore.ParameterStore

// Init initializes the pssh subcommands
func Init(_shell *ishell.Shell, _ps *parameterstore.ParameterStore) {
	shell = _shell
	ps = _ps
	registerCommand("cd", "change your relative location within the parameter store", cd, "")
	registerCommand("cp", "copy source to dest", cp, cpUsage)
	registerCommand("decrypt", "toggle parameter decryption", decrypt, decryptUsage)
	// registerCommand("get", "get parameters", get, getUsage)
	registerCommand("history", "toggle parameter history", history, historyUsage)
	registerCommand("ls", "list parameters", ls, "")
	registerCommand("mv", "not yet implemented", mv, "")
	registerCommand("put", "set parameter", put, putUsage)
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

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}
