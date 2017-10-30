package cmd

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/kountable/pssh/display"
	"github.com/spf13/cobra"
)

var (
	getParametereHeader = []string{"Parameter", "Type", "Value"}
)

var getOptions = &GetParametersInput{}

type GetParametersInput struct {
	SSMInput ssm.GetParametersInput
}

func RunGet(o *GetParametersInput) error {
	resp, err := ps.GetParameters(&o.SSMInput)
	if err != nil {
		return err
	}

	var data [][]string

	for _, p := range resp {
		var pName, pType, pValue string

		if p.Name != nil {
			pName = *p.Name
		}
		if p.Type != nil {
			pType = *p.Type
		}
		if p.Value != nil {
			pValue = *p.Value
		}

		data = append(data, []string{pName, pType, pValue})
	}

	display.RenderTable(getParametereHeader, data)

	return nil
}

func NewGetparameterCommand() *cobra.Command {

	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "get [NAMES] (list) [-d|--with-decryption Return decrypted secure string value.]",
		Long:  `Get one or more parameters.`,
		Run: func(cmd *cobra.Command, args []string) {

			var names []*string

			for _, v := range args {
				names = append(names, aws.String(v))
			}

			getOptions.SSMInput.Names = names

			if err := getOptions.SSMInput.Validate(); err != nil {
				ExitWithError(ExitInvalidInput, err)
			}

			if err := RunGet(getOptions); err != nil {
				ExitWithError(ExitError, err)
			}
		},
	}

	getOptions.SSMInput.WithDecryption = aws.Bool(false)

	getCmd.Flags().BoolVarP(getOptions.SSMInput.WithDecryption, "with-decryption", "d", false, "Return decrypted values for secure string parameters.")

	return getCmd
}
