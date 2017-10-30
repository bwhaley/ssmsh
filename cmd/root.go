package cmd

import (
	"fmt"

	"github.com/kountable/pssh/parameterstore"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "pssh",
	Short: "Command line interface to work with AWS parameters store",
	Long:  `Command line interface to work with AWS parameters store`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Root command....")
		// TODO: Start interactive mode here
	},
}

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
}
