package parameterstore

import (
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Get retrieves one or more parameters
func (ps *ParameterStore) GetParameters(c *ssm.GetParametersInput) ([]ssm.Parameter, error) {
	resp, err := ssmsvc.GetParameters(c)
	if err != nil {
		return nil, err
	}

	var params []ssm.Parameter

	for _, p := range resp.Parameters {
		params = append(params, *p)
	}

	return params, nil
}
