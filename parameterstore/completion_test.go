package parameterstore

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type completionAPI struct {
	API
	calls    int
	describe func(context.Context, *ssm.DescribeParametersInput) (*ssm.DescribeParametersOutput, error)
}

func (f *completionAPI) DescribeParameters(ctx context.Context, in *ssm.DescribeParametersInput, _ ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
	f.calls++
	return f.describe(ctx, in)
}
func TestCompletionNamesPagination(t *testing.T) {
	f := &completionAPI{}
	f.describe = func(_ context.Context, in *ssm.DescribeParametersInput) (*ssm.DescribeParametersOutput, error) {
		filter := in.ParameterFilters[0]
		if aws.ToInt32(in.MaxResults) != 50 || aws.ToString(filter.Key) != "Name" || aws.ToString(filter.Option) != "BeginsWith" || !reflect.DeepEqual(filter.Values, []string{"/app/"}) {
			t.Fatalf("unexpected input: %+v", in)
		}
		if in.NextToken == nil {
			return &ssm.DescribeParametersOutput{NextToken: aws.String("next")}, nil
		}
		if *in.NextToken != "next" {
			t.Fatal("wrong pagination token")
		}
		return &ssm.DescribeParametersOutput{Parameters: []types.ParameterMetadata{{Name: aws.String("/app/z")}, {Name: aws.String("/app/a")}, {Name: aws.String("/app/a")}}}, nil
	}
	ps := &ParameterStore{Clients: map[string]API{"r": f}}
	got, err := ps.CompletionNames(context.Background(), "/app/", "r")
	if err != nil || !reflect.DeepEqual(got, []string{"/app/a", "/app/z"}) || f.calls != 2 {
		t.Fatalf("got %v, %v; calls %d", got, err, f.calls)
	}
}
func TestCompletionNamesDiscardsIncompleteResults(t *testing.T) {
	for _, mode := range []string{"denied", "cancelled", "repeated", "limit"} {
		t.Run(mode, func(t *testing.T) {
			f := &completionAPI{}
			f.describe = func(ctx context.Context, _ *ssm.DescribeParametersInput) (*ssm.DescribeParametersOutput, error) {
				if mode == "denied" {
					return nil, errors.New("access denied")
				}
				if mode == "cancelled" {
					return nil, ctx.Err()
				}
				token := "same"
				if mode == "limit" {
					token = string(rune('a' + f.calls))
				}
				return &ssm.DescribeParametersOutput{Parameters: []types.ParameterMetadata{{Name: aws.String("/only")}}, NextToken: aws.String(token)}, nil
			}
			ps := &ParameterStore{Clients: map[string]API{"r": f}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			got, err := ps.CompletionNames(ctx, "/", "r")
			if err == nil || got != nil {
				t.Fatalf("incomplete result accepted: %v, %v", got, err)
			}
			if mode == "limit" && f.calls != 10 {
				t.Fatalf("calls = %d", f.calls)
			}
		})
	}
}
