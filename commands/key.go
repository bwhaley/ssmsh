package commands

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	saws "github.com/bwhaley/ssmsh/aws"
)

const keyUsage = "key [ARN|ID|alias/name]"

func key(c *Context) error {
	if len(c.Args) == 0 {
		shell.Println(ps.Key)
		return nil
	}
	if len(c.Args) != 1 {
		return fmt.Errorf("usage: %s", keyUsage)
	}
	cfg, err := saws.Load(commandContext, ps.Region, ps.Profile)
	if err != nil {
		return err
	}
	_, err = kms.NewFromConfig(cfg).DescribeKey(commandContext, &kms.DescribeKeyInput{KeyId: aws.String(c.Args[0])})
	if err != nil {
		return err
	}
	ps.Key = c.Args[0]
	return nil
}
