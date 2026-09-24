package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Load uses the SDK credential chain, including SSO, roles and workload credentials.
func Load(ctx context.Context, region, profile string) (aws.Config, error) {
	var options []func(*config.LoadOptions) error
	if region != "" {
		options = append(options, config.WithRegion(region))
	}
	if profile != "" {
		options = append(options, config.WithSharedConfigProfile(profile))
	}
	return config.LoadDefaultConfig(ctx, options...)
}
