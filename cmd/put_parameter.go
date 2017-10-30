package cmd

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/cobra"
)

var putOptions = &PutParameterInput{}

// var putParametereHeader = []string{"Status"}

type PutParameterInput struct {
	SSMInput ssm.PutParameterInput
}

func RunPut(p *PutParameterInput) error {

	err := ps.PutParameter(&p.SSMInput)
	if err != nil {
		ExitWithError(ExitError, err)
	}

	// TODO: API response from put request doesn't contain a new version,
	// but aws cli put-parameter returns a new version of the parameter -- :'(
	// display.RenderTable(putParametereHeader, [][]string{"Done"})

	return nil
}

func NewPutParameterCommand() *cobra.Command {

	var putCmd = &cobra.Command{
		Use:   "put",
		Short: "put [NAME] [-n|--name]  [-v|--value] [-t|--type] [-d|--description] [-k|--key-id] [-o|--overwrite] [-p|--allowed-pattern]",
		Long:  `Add or update one parameters to the system.`,
		Run: func(cmd *cobra.Command, args []string) {

			if err := putOptions.SSMInput.Validate(); err != nil {
				ExitWithError(ExitInvalidInput, err)
			}

			if err := RunPut(putOptions); err != nil {
				ExitWithError(ExitError, err)
			}
		},
	}

	putOptions.SSMInput.Name = aws.String("")
	putOptions.SSMInput.Value = aws.String("")
	putOptions.SSMInput.Type = aws.String("")
	putOptions.SSMInput.Description = aws.String("")
	putOptions.SSMInput.KeyId = aws.String("")
	putOptions.SSMInput.Overwrite = aws.Bool(false)
	putOptions.SSMInput.AllowedPattern = aws.String("")

	putCmd.Flags().StringVarP(putOptions.SSMInput.Name, "name", "n", "", "The name of the parameter that you want to add to the system.")
	putCmd.Flags().StringVarP(putOptions.SSMInput.Value, "value", "v", "", "The parameter value that you want to add to the system.")
	putCmd.Flags().StringVarP(putOptions.SSMInput.Type, "type", "t", "", "The type of parameter that you want to add to the system.")
	putCmd.Flags().StringVarP(putOptions.SSMInput.Description, "description", "d", "", "Information about the parameter that you want to add to the system.")
	putCmd.Flags().StringVarP(putOptions.SSMInput.KeyId, "key-id", "k", "", "The KMS Key ID that you want to use to encrypt a parameter when you choose the SecureString data type.")
	putCmd.Flags().BoolVarP(putOptions.SSMInput.Overwrite, "overwrite", "o", false, "Overwrite an existing parameter. If not specified, will default to 'false'.")
	putCmd.Flags().StringVarP(putOptions.SSMInput.AllowedPattern, "allowed-pattern", "p", "", "A regular expression used to validate the parameter value.")

	return putCmd
}
