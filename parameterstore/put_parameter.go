package parameterstore

import "github.com/aws/aws-sdk-go/service/ssm"

// TODO: Do we need this method wrapper at all?

// Put creates or updates a parameter
func (ps *ParameterStore) PutParameter(param *ssm.PutParameterInput) error {
	_, err := ssmsvc.PutParameter(param)

	return err
}
