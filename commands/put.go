package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

const putUsage = "put name=PATH (value=VALUE|value-file=FILE|value-stdin=true) [type=String|SecureString|StringList] [key=KEY] [region=REGION] [overwrite=true] [tier=Standard|Advanced|Intelligent-Tiering] [policies=[NAME,...]]"

func put(c *Context) error {
	options := c.Args
	if len(options) == 0 {
		if !Interactive {
			return fmt.Errorf("usage: %s", putUsage)
		}
		shell.Println("Enter one name=value option per line; finish with an empty line.")
		for {
			if shell.readLine == nil {
				return fmt.Errorf("interactive input is unavailable")
			}
			line, err := shell.readLine()
			if err != nil {
				return err
			}
			if line == "" {
				break
			}
			options = append(options, line)
		}
	}
	in := &ssm.PutParameterInput{Type: types.ParameterType(ps.Type), Overwrite: aws.Bool(ps.Overwrite)}
	if ps.Key != "" {
		in.KeyId = aws.String(ps.Key)
	}
	region := ps.Region
	valueSet := false
	for _, option := range options {
		field, value, ok := strings.Cut(option, "=")
		if !ok {
			return fmt.Errorf("expected name=value option")
		}
		field = strings.ToLower(strings.TrimSpace(field))
		switch field {
		case "name":
			if value == "" {
				return fmt.Errorf("name cannot be empty")
			}
			in.Name = aws.String(ps.Resolve(value))
		case "value", "value-file", "value-stdin":
			if valueSet {
				return fmt.Errorf("specify exactly one value source")
			}
			valueSet = true
			switch field {
			case "value-file":
				data, err := os.ReadFile(value)
				if err != nil {
					return err
				}
				value = string(data)
			case "value-stdin":
				if value != "true" {
					return fmt.Errorf("value-stdin must be true")
				}
				if BatchInput {
					return fmt.Errorf("value-stdin cannot share stdin with batch commands")
				}
				data, err := io.ReadAll(SecretInput)
				if err != nil {
					return err
				}
				value = string(data)
			}
			in.Value = aws.String(value)
		case "type":
			switch strings.ToLower(value) {
			case "string":
				in.Type = types.ParameterTypeString
			case "securestring":
				in.Type = types.ParameterTypeSecureString
			case "stringlist":
				in.Type = types.ParameterTypeStringList
			default:
				return fmt.Errorf("invalid parameter type")
			}
		case "description":
			in.Description = aws.String(value)
		case "key":
			in.KeyId = aws.String(value)
		case "pattern":
			in.AllowedPattern = aws.String(value)
		case "region":
			if value == "" {
				return fmt.Errorf("region cannot be empty")
			}
			region = value
		case "overwrite":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("overwrite must be boolean")
			}
			in.Overwrite = aws.Bool(b)
		case "tier":
			tier, err := parseTier(value)
			if err != nil {
				return err
			}
			in.Tier = tier
		case "policies":
			policy, err := policyJSON(value)
			if err != nil {
				return err
			}
			in.Policies = aws.String(policy)
		default:
			return fmt.Errorf("unknown put option %q", field)
		}
	}
	if in.Name == nil || in.Value == nil {
		return fmt.Errorf("name and a value source are required")
	}
	if in.Policies != nil {
		in.Tier = types.ParameterTierAdvanced
	}
	switch in.Type {
	case types.ParameterTypeString, types.ParameterTypeSecureString, types.ParameterTypeStringList:
	default:
		return fmt.Errorf("invalid default parameter type")
	}
	out, err := ps.Put(commandContext, in, region)
	if err != nil {
		return err
	}
	if !ps.DryRun {
		shell.Printf("Put %s version %d\n", aws.ToString(in.Name), out.Version)
	}
	return nil
}
func parseTier(value string) (types.ParameterTier, error) {
	for _, tier := range []types.ParameterTier{types.ParameterTierStandard, types.ParameterTierAdvanced, types.ParameterTierIntelligentTiering} {
		if strings.EqualFold(value, string(tier)) {
			return tier, nil
		}
	}
	return "", fmt.Errorf("tier must be Standard, Advanced or Intelligent-Tiering")
}
func policyJSON(value string) (string, error) {
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return "", fmt.Errorf("policies must be [name,...]")
	}
	result := []Policies{}
	for _, name := range trim(strings.Split(value[1:len(value)-1], ",")) {
		p, ok := policies[name]
		if !ok {
			return "", fmt.Errorf("unknown policy %q", name)
		}
		if p.expiration != (Expiration{}) {
			result = append(result, p.expiration)
		}
		for _, v := range p.expirationNotification {
			result = append(result, v)
		}
		for _, v := range p.noChangeNotification {
			result = append(result, v)
		}
	}
	b, err := json.Marshal(result)
	return string(b), err
}
