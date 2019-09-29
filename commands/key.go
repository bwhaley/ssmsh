package commands

import (
	"fmt"

	"github.com/abiosoft/ishell"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	saws "github.com/bwhaley/ssmsh/aws"
)

const keyUsage string = `
key ARN|ID
Set the KMS key ARN (or ID) to use with SecureString parameters
`

func key(c *ishell.Context) {
	if len(c.Args) != 1 {
		shell.Println(keyUsage)
	}
	if err := checkKey(c.Args[0]); err != nil {
		shell.Println(err)
	}
	ps.Key = c.Args[0]
}

func checkKey(key string) (err error) {
	client := kms.New(saws.NewSession(ps.Region, ps.Profile))
	input := kms.ListKeysInput{}
	for {
		resp, err := client.ListKeys(&input)
		if err != nil {
			return err
		}
		for _, keyEntry := range resp.Keys {
			if aws.StringValue(keyEntry.KeyId) == key || aws.StringValue(keyEntry.KeyArn) == key {
				return nil
			}
		}
		if resp.NextMarker == nil {
			break
		}
		input.Marker = resp.NextMarker
	}
	return fmt.Errorf("key %s not found in this region", key)
}
