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
	GetParametersByPathNext ssm.GetParametersByPathOutput
	GetParameterResp        ssm.GetParameterOutput
	GetParameterHistoryResp ssm.GetParameterHistoryOutput
	GetParametersResp       ssm.GetParametersOutput
	DeleteParametersResp    ssm.DeleteParametersOutput
}

func (m mockedSSM) GetParametersByPath(in *ssm.GetParametersByPathInput) (*ssm.GetParametersByPathOutput, error) {
	if aws.StringValue(in.NextToken) != "" {
		return &m.GetParametersByPathNext, nil
	}
	return &m.GetParametersByPathResp, nil
}

func (m mockedSSM) DeleteParameters(in *ssm.DeleteParametersInput) (*ssm.DeleteParametersOutput, error) {
	return &m.DeleteParametersResp, nil
}

func (m mockedSSM) GetParameter(in *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	return &m.GetParameterResp, nil
}

func (m mockedSSM) GetParameterHistory(in *ssm.GetParameterHistoryInput) (*ssm.GetParameterHistoryOutput, error) {
	return &m.GetParameterHistoryResp, nil
}

func (m mockedSSM) GetParameters(in *ssm.GetParametersInput) (*ssm.GetParametersOutput, error) {
	return &m.GetParametersResp, nil
}

func TestCwd(t *testing.T) {
	cases := []struct {
		GetParametersByPathResp ssm.GetParametersByPathOutput
		Path                    string
		Expected                string
	}{
		{
			Path:     "/",
			Expected: "/",
		},
		{
			Path: "/dev/db/../..//prod",
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("/prod/db/username"),
						Type:  aws.String("String"),
						Value: aws.String("someusername"),
					},
				},
				NextToken: aws.String(""),
			},
			Expected: "/prod",
		},
	}

	var p parameterstore.ParameterStore
	for _, c := range cases {
		p.NewParameterStore()
		p.Cwd = parameterstore.Delimiter
		p.Client = mockedSSM{
			GetParametersByPathResp: c.GetParametersByPathResp,
		}
		err := p.SetCwd(c.Path)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		if p.Cwd != c.Expected {
			msg := fmt.Errorf("expected %v, got %v", c.Expected, p.Cwd)
			t.Fatal(msg)
		}
	}

	p.NewParameterStore()
	p.Cwd = parameterstore.Delimiter
	testDir := "/nodir"
	err := p.SetCwd(testDir)
	if err == nil {
		msg := fmt.Errorf("Expected error for dir %s, got cwd %s ", testDir, p.Cwd)
		t.Fatal(msg)
	}
}

func TestDelete(t *testing.T) {
	testParams := []string{
		"/dev/db/username",
		"/dev/db/password",
		"/dev/db/foobar",
	}
	deleteParametersOutput := ssm.DeleteParametersOutput{
		DeletedParameters: []*string{
			aws.String("/dev/db/username"),
			aws.String("/dev/db/password"),
		},
		InvalidParameters: []*string{
			aws.String("/dev/db/foobar"),
		},
	}

	var p parameterstore.ParameterStore
	p.NewParameterStore()
	p.Client = mockedSSM{
		DeleteParametersResp: deleteParametersOutput,
	}
	err := p.Delete(testParams)
	if err == nil {
		msg := fmt.Errorf("Expected error for param %s, got %s ", testParams[2], err)
		t.Fatal(msg)
	}
}

func TestGetHistory(t *testing.T) {
	testParam := "/dev/db/username"
	getHistoryOutput := ssm.GetParameterHistoryOutput{
		Parameters: []*ssm.ParameterHistory{
			{
				Name: aws.String("/dev/db/username"),
			},
			{
				Name: aws.String("/dev/db/username"),
			},
		},
		NextToken: aws.String(""),
	}
	var p parameterstore.ParameterStore
	p.NewParameterStore()
	p.Client = mockedSSM{
		GetParameterHistoryResp: getHistoryOutput,
	}
	resp, err := p.GetHistory(testParam)
	if err != nil {
		msg := fmt.Errorf("Unexpected error %s", err)
		t.Fatal(msg)
	}
	if len(resp) != 2 {
		msg := fmt.Errorf("Expected history of length 2, got %s", resp)
		t.Fatal(msg)
	}
}

func TestList(t *testing.T) {
	cases := []struct {
		Query                   string
		GetParametersByPathResp ssm.GetParametersByPathOutput
		GetParametersResp       ssm.GetParametersOutput
		GetParametersByPathNext ssm.GetParametersByPathOutput
		Expected                []string
		Recurse                 bool
	}{
		{
			Query:   "/dev/db/username",
			Recurse: false,
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
			Query:   "/",
			Recurse: false,
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
			Query:   "/dev/db",
			Recurse: false,
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
		{
			Query:   "/dev/db",
			Recurse: false,
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
				NextToken: aws.String("A1B2C3D4"),
			},
			GetParametersByPathNext: ssm.GetParametersByPathOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("/dev/db/test/port"),
						Type:  aws.String("String"),
						Value: aws.String("3306"),
					},
				},
				NextToken: aws.String(""),
			},
			Expected: []string{
				"name",
				"username",
				"password",
				"test/",
			},
			GetParametersResp: ssm.GetParametersOutput{
				InvalidParameters: []*string{
					aws.String("/dev/db/name"),
					aws.String("String"),
					aws.String("mydb"),
				},
			},
		},
		{
			Query:   "/dev/db",
			Recurse: true,
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
				NextToken: aws.String("A1B2C3D4"),
			},
			GetParametersByPathNext: ssm.GetParametersByPathOutput{
				Parameters: []*ssm.Parameter{
					{
						Name:  aws.String("/dev/db/test/port"),
						Type:  aws.String("String"),
						Value: aws.String("3306"),
					},
				},
				NextToken: aws.String(""),
			},
			Expected: []string{
				"/dev/db/name",
				"/dev/db/username",
				"/dev/db/password",
				"/dev/db/test/port",
			},
			GetParametersResp: ssm.GetParametersOutput{
				InvalidParameters: []*string{
					aws.String("/dev/db/name"),
					aws.String("String"),
					aws.String("mydb"),
				},
			},
		}}

	for _, c := range cases {
		var p parameterstore.ParameterStore
		p.NewParameterStore()
		p.Client = mockedSSM{
			GetParametersByPathResp: c.GetParametersByPathResp,
			GetParametersByPathNext: c.GetParametersByPathNext,
			GetParametersResp:       c.GetParametersResp,
		}
		p.Cwd = parameterstore.Delimiter
		if c.Recurse {
			p.Recurse = true
		} else {
			p.Recurse = false
		}
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
