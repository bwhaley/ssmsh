package cmd

import (
	"fmt"
	"strings"

	prompt "github.com/c-bata/go-prompt"
	"github.com/kountable/pssh/parameterstore"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var RootCmd = &cobra.Command{
	Use:   "pssh",
	Short: "Command line interface to work with AWS parameters store",
	Long:  `Command line interface to work with AWS parameters store`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please input pssh commands")

		defer fmt.Println("\U0001f984")

		p := prompt.New(
			executor,
			completer,
			prompt.OptionTitle("pssh interactive client"),
			prompt.OptionPrefix(">>"),
		)

		p.Run()

	},
}

var commandFlagSuggestions = make(map[string][]prompt.Suggest)
var commandMap = make(map[string][]prompt.Suggest)
var promptSuggestions = []prompt.Suggest{}

// var globalOptions = &GlobalOptions{}
// type GlobaOptions struct {
// 	AwsRegion string
// }

var ps *parameterstore.ParameterStore

func init() {

	ps = parameterstore.NewParameterStore()

	RootCmd.AddCommand(NewGetparameterCommand())
	RootCmd.AddCommand(NewPutParameterCommand())

	//TODO: This is an example of the global flag, we will need global flag for AWS region and some toher parameters
	// RootCmd.PersistentFlags().StringVarP(&globalOptions.AwsRegion, "region", "r", "", "aws region")

	initializePrompt()
}

func initPrompt() {
}

func executor(s string) {
	cmd := strings.TrimSpace(s)

	if cmd == "" {
		return
	}

	exitCommands := map[string]struct{}{
		"exit":  struct{}{},
		"close": struct{}{},
		"quit":  struct{}{},
	}

	if _, ok := exitCommands[cmd]; ok {
		fmt.Println("Bye!")
		ExitWithError(ExitSuccess, nil)
		return
	}

}

func completer(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}
	args := strings.Split(d.TextBeforeCursor(), " ")
	w := d.GetWordBeforeCursor()

	// If PIPE is in text before the cursor, returns empty suggestions.
	for i := range args {
		if args[i] == "|" {
			return []prompt.Suggest{}
		}
	}

	// If word before the cursor starts with "-", returns CLI flag options.
	if strings.HasPrefix(w, "-") {
		return optionCompleter(args)
	}

	return argumentsCompleter(excludeOptions(args))
}

func initializePrompt() {
	// Get the commands from Cobra to build the prompt suggestions
	for _, command := range RootCmd.Commands() {
		// Get the flags for each of the commands in order to enable option completion
		var flagSuggestions = []prompt.Suggest{}

		command.Flags().VisitAll(func(flag *pflag.Flag) {
			flagSuggestions = append(flagSuggestions, prompt.Suggest{
				Text:        "--" + flag.Name,
				Description: flag.Usage,
			})
		})

		commandFlagSuggestions[command.Name()] = flagSuggestions

		var promptSuggestion = prompt.Suggest{
			Text:        command.Name(),
			Description: command.Short,
		}
		promptSuggestions = append(promptSuggestions, promptSuggestion)
	}
}

func optionCompleter(args []string) []prompt.Suggest {
	l := len(args)
	var flagSuggestions []prompt.Suggest

	command := args[0]
	flagSuggestions = commandFlagSuggestions[command]

	return prompt.FilterContains(flagSuggestions, strings.TrimLeft(args[l-1], "-"), true)
}

func argumentsCompleter(args []string) []prompt.Suggest {
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(promptSuggestions, args[0], true)
	}
	first := args[0]
	return prompt.FilterHasPrefix(promptSuggestions, first, true)
}

func excludeOptions(args []string) []string {
	ret := make([]string, 0, len(args))
	for i := range args {
		if !strings.HasPrefix(args[i], "-") {
			ret = append(ret, args[i])
		}
	}
	return ret
}
