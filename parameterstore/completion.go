package parameterstore

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// CompletionNames returns metadata only. Incomplete results are discarded so a
// partial page cannot make an ambiguous prefix look like a unique match.
func (ps *ParameterStore) CompletionNames(ctx context.Context, prefix, region string) ([]string, error) {
	client, err := ps.InitClient(ctx, region)
	if err != nil {
		return nil, err
	}
	input := &ssm.DescribeParametersInput{MaxResults: aws.Int32(50)}
	input.ParameterFilters = []types.ParameterStringFilter{{Key: aws.String("Name"), Option: aws.String("BeginsWith"), Values: []string{prefix}}}
	pager := ssm.NewDescribeParametersPaginator(client, input)
	names := make(map[string]bool)
	tokens := make(map[string]bool)
	for pages := 0; pager.HasMorePages(); pages++ {
		if pages == 10 {
			return nil, fmt.Errorf("too many completion results; type a longer path prefix")
		}
		out, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range out.Parameters {
			if p.Name != nil {
				names[*p.Name] = true
			}
		}
		if token := aws.ToString(out.NextToken); token != "" {
			if tokens[token] {
				return nil, fmt.Errorf("completion pagination did not advance")
			}
			tokens[token] = true
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
