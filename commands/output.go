package commands

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// These output records are owned by ssmsh, independent of SDK serialization.
// Capitalized JSON field names preserve the existing command-line contract.
type Parameter struct {
	ARN              string
	DataType         string
	LastModifiedDate *time.Time
	Name             string
	Selector         string
	SourceResult     string
	Type             string
	Value            string
	Version          int64
}
type ParameterPolicy struct{ PolicyStatus, PolicyText, PolicyType string }
type ParameterHistory struct {
	AllowedPattern   string
	DataType         string
	Description      string
	KeyId            string
	Labels           []string
	LastModifiedDate *time.Time
	LastModifiedUser string
	Name             string
	Policies         []ParameterPolicy
	Tier             string
	Type             string
	Value            string
	Version          int64
}

func printParameters(params []types.Parameter) error {
	records := make([]Parameter, 0, len(params))
	for _, p := range params {
		if cfg.Default.Output == "value" {
			shell.Println(aws.ToString(p.Value))
			continue
		}
		records = append(records, Parameter{ARN: aws.ToString(p.ARN), DataType: aws.ToString(p.DataType), LastModifiedDate: p.LastModifiedDate, Name: aws.ToString(p.Name), Selector: aws.ToString(p.Selector), SourceResult: aws.ToString(p.SourceResult), Type: string(p.Type), Value: aws.ToString(p.Value), Version: p.Version})
	}
	if cfg.Default.Output == "value" {
		return nil
	}
	if cfg.Default.Output == "json" {
		return printJSON(records)
	}
	for _, p := range records {
		shell.Printf("%s\t%s\t%s\n", p.Name, p.Type, p.Value)
	}
	return nil
}
func printHistory(params []types.ParameterHistory) error {
	records := make([]ParameterHistory, 0, len(params))
	for _, p := range params {
		if cfg.Default.Output == "value" {
			shell.Println(aws.ToString(p.Value))
			continue
		}
		policies := []ParameterPolicy{}
		for _, v := range p.Policies {
			policies = append(policies, ParameterPolicy{
				PolicyStatus: aws.ToString(v.PolicyStatus),
				PolicyText:   aws.ToString(v.PolicyText),
				PolicyType:   aws.ToString(v.PolicyType),
			})
		}
		labels := append([]string{}, p.Labels...)
		records = append(records, ParameterHistory{AllowedPattern: aws.ToString(p.AllowedPattern), DataType: aws.ToString(p.DataType), Description: aws.ToString(p.Description), KeyId: aws.ToString(p.KeyId), Labels: labels, LastModifiedDate: p.LastModifiedDate, LastModifiedUser: aws.ToString(p.LastModifiedUser), Name: aws.ToString(p.Name), Policies: policies, Tier: string(p.Tier), Type: string(p.Type), Value: aws.ToString(p.Value), Version: p.Version})
	}
	if cfg.Default.Output == "value" {
		return nil
	}
	if cfg.Default.Output == "json" {
		return printJSON(records)
	}
	for _, p := range records {
		shell.Printf("%s\t%d\t%s\n", p.Name, p.Version, p.Value)
	}
	return nil
}
