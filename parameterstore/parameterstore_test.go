package parameterstore_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/kountable/pssh/parameterstore"
)

type mockedSSM struct {
	ssmiface.SSMAPI
	GetParametersByPathResp ssm.GetParametersByPathOutput
	GetParameterResp        ssm.GetParameterOutput
	GetParametersResp       ssm.GetParametersOutput
}

func (m mockedSSM) GetParametersByPath(in *ssm.GetParametersByPathInput) (*ssm.GetParametersByPathOutput, error) {
	return &m.GetParametersByPathResp, nil
}

func (m mockedSSM) GetParameter(in *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	return &m.GetParameterResp, nil
}

func (m mockedSSM) GetParameters(in *ssm.GetParametersInput) (*ssm.GetParametersOutput, error) {
	return &m.GetParametersResp, nil
}

func TestList(t *testing.T) {
	cases := []struct {
		Query                   string
		GetParametersByPathResp ssm.GetParametersByPathOutput
		GetParameterResp        ssm.GetParameterOutput
		GetParametersResp       ssm.GetParametersOutput
		Expected                []string
	}{
		{
			Query: "/dev/db/username",
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("/dev/db/username"),
						Type:  aws.String("String"),
						Value: aws.String("someusername"),
					},
				},
				NextToken: aws.String(""),
			},
			Expected: []string{
				"/dev/db/username",
			},
			GetParametersResp: ssm.GetParametersOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("/dev/db/username"),
						Type:  aws.String("String"),
						Value: aws.String("someusername"),
					},
				},
			},
		}, {
			Query: "/",
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("root"),
						Type:  aws.String("String"),
						Value: aws.String("A root parameter"),
					},
				},
				NextToken: aws.String(""),
			},
			Expected: []string{
				"root",
			},
			GetParametersResp: ssm.GetParametersOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("root"),
						Type:  aws.String("String"),
						Value: aws.String("A root parameter"),
					},
				},
			},
		},
		{
			Query: "/dev/db",
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("/dev/db/name"),
						Type:  aws.String("String"),
						Value: aws.String("mydb"),
					},
					{
						Name:  aws.String("/dev/db/username"),
						Type:  aws.String("String"),
						Value: aws.String("myusername"),
					},
					{
						Name:  aws.String("/dev/db/password"),
						Type:  aws.String("SecureString"),
						Value: aws.String("mypassword"),
					},
				},
				NextToken: aws.String(""),
			},
			Expected: []string{
				"name",
				"username",
				"password",
			},
			GetParametersResp: ssm.GetParametersOutput{
				InvalidParameters: []*string{
					aws.String("/dev/db/name"),
					aws.String("String"),
					aws.String("mydb"),
				},
			},
		},
	}

	for _, c := range cases {
		var p parameterstore.ParameterStore
		p.NewParameterStore()
		p.Client = mockedSSM{
			GetParametersByPathResp: c.GetParametersByPathResp,
			GetParametersResp:       c.GetParametersResp,
		}
		p.Cwd = parameterstore.Delimiter
		resp, err := p.List(c.Query)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		if !equal(resp, c.Expected) {
			msg := fmt.Errorf("expected %v, got %v", c.Expected, resp)
			t.Fatal(msg)
		}
	}
}

func equal(first []string, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for i := 0; i < len(first); i++ {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}
