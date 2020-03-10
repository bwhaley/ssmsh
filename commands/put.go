package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/bwhaley/ssmsh/parameterstore"
)

// TODO Inline syntax
const putUsage string = `usage: put <newline>
Create or update parameters. Enter one option per line, ending with a blank line, or all
options inline. All fields except name, value, and type are optional.
Example of inline put:
/> put name=/House/Targaryen/Daenerys value="Queen" type="String" description="Mother of dragons"
Example of multiline put:
/>put
... name=/House/Lannister/Cersei
... value=Queen
... type=String
... description=Queen of the Seven Kingdoms
... key=arn:aws:kms:us-west-2:012345678901:key/321examp-ed00-427f-9729-748ba2254794
... overwrite=true
... pattern=[A-z]+
... tier=advanced
... policies=[policy1, policy2]
...
/>
Use the policy command to create named policy objects. Tier defaults to standard unless policies are defined.
`

var putParamInput ssm.PutParameterInput
var putParamRegion string

// Add or update parameters
func put(c *ishell.Context) {
	var err error
	var resp *ssm.PutParameterOutput

	putParamInput = ssm.PutParameterInput{}
	err = setDefaults(&putParamInput)
	if err != nil {
		shell.Println(err)
		return
	}

	// Read args for values
	var r bool
	if len(c.Args) == 0 {
		r = multiLinePut()
	} else {
		r = inlinePut(c.Args)
	}
	if !r {
		return
	}

	if putParamInput.Name == nil ||
		putParamInput.Value == nil ||
		putParamInput.Type == nil {
		shell.Println("Error: name, type and value are required.")
		return
	}

	resp, err = ps.Put(&putParamInput, putParamRegion)
	if err != nil {
		shell.Println("Error: ", err)
	} else {
		version := strconv.Itoa(int(aws.Int64Value(resp.Version)))
		if version != "" {
			shell.Println("Put " + aws.StringValue(putParamInput.Name) + " version " + version)
		}
	}
}

// setDefaults sets parameter settings according to the defaults
func setDefaults(param *ssm.PutParameterInput) (err error) {
	param.SetOverwrite(ps.Overwrite)
	if ps.Key != "" {
		param.SetKeyId(ps.Key)
	}
	param.SetType(ps.Type)
	err = validateType(ps.Type)
	if err != nil {
		return err
	}
	putParamRegion = ps.Region
	return nil
}

func multiLinePut() bool {
	// Set the prompt explicitly rather than use SetMultiPrompt
	// due to the unexpected 2nd line behavior
	shell.SetPrompt("... ")
	defer setPrompt(ps.Cwd)

	shell.Println("Input options. End with a blank line.")
	str := shell.ReadMultiLinesFunc(putOptions)
	if str == "" {
		shell.Println("multiline input ended in empty string")
		return false
	}
	return true
}

func inlinePut(options []string) bool {
	for _, p := range options {
		if !putOptions(p) {
			return false
		}
	}
	return true
}

func putOptions(s string) bool {
	if s == "" {
		return false
	}
	paramOption := strings.Split(s, "=")
	if len(paramOption) < 2 {
		shell.Println("invalid input")
		shell.Println(putUsage)
		return false
	}
	field := strings.ToLower(paramOption[0])
	val := strings.Join(paramOption[1:], "=") // Handles the case where a value has an "=" character
	err := validate(field, val)
	if err != nil {
		shell.Println(err)
		return false
	}
	return true
}

func validate(f, v string) (err error) {
	m := map[string]func(string) error{
		"type":        validateType,
		"name":        validateName,
		"value":       validateValue,
		"description": validateDescription,
		"key":         validateKey,
		"pattern":     validatePattern,
		"overwrite":   validateOverwrite,
		"region":      validateRegion,
		"tier":        validateTier,
		"policies":    validatePolicies,
	}
	if validator, ok := m[strings.ToLower(f)]; ok {
		err = validator(v)
		if err != nil {
			// A validator failed so we need to reset the parameter input to an empty state
			putParamInput = ssm.PutParameterInput{}
			shell.Println(putUsage)
			return err
		}
	}
	return nil
}

func validateType(s string) (err error) {
	validTypes := []string{"String", "StringList", "SecureString"}
	for i := 0; i < len(validTypes); i++ {
		if strings.EqualFold(s, validTypes[i]) { // Case insensitive validation of type field
			putParamInput.Type = aws.String(validTypes[i])
			return nil
		}
	}
	return fmt.Errorf("Invalid type %s", s)
}

func validateValue(s string) (err error) {
	s = trimSpaces(s)
	putParamInput.Value = aws.String(s)
	return nil
}

// trimSpaces works around an issue in ishell where a space is added to the end of each line in a multiline value
// https://github.com/abiosoft/ishell/issues/132
func trimSpaces(s string) string {
	parts := strings.Split(s, "\n")
	for i := 0; i < len(parts)-1; i++ {
		size := len(parts[i])
		parts[i] = parts[i][:size-1]
	}
	return strings.Join(parts, "\n")
}

func validateName(s string) (err error) {
	if strings.HasPrefix(s, parameterstore.Delimiter) {
		putParamInput.SetName(s)
	} else {
		putParamInput.SetName(ps.Cwd + parameterstore.Delimiter + s)
	}
	return nil
}

func validateDescription(s string) (err error) {
	putParamInput.SetDescription(s)
	return nil
}

// TODO validate key
func validateKey(s string) (err error) {
	putParamInput.SetKeyId(s)
	return nil
}

// TODO validate pattern
func validatePattern(s string) (err error) {
	putParamInput.SetAllowedPattern(s)
	return nil
}

func validateOverwrite(s string) (err error) {
	overwrite, err := strconv.ParseBool(s)
	if err != nil {
		shell.Println("overwrite must be true or false")
		return err
	}
	putParamInput.SetOverwrite(overwrite)
	return nil
}

func validateRegion(s string) (err error) {
	putParamRegion = s
	return nil
}

const (
	StandardTier = "Standard"
	AdvancedTier = "Advanced"
)

func validateTier(s string) (err error) {
	if strings.ToLower(s) == StandardTier || strings.ToLower(s) == AdvancedTier {
		putParamInput.Tier = aws.String(strings.Title(s))
		return nil
	}
	return errors.New("tier must be standard or advanced")
}

func validatePolicies(s string) (err error) {
	var policySet []Policies
	re := regexp.MustCompile(`^\[([\w\s,]+)\]`)
	p := re.FindStringSubmatch(s)
	if len(p) != 2 {
		return fmt.Errorf("unable to validate policies %s", s)
	}
	namedPolicies := trim(strings.Split(p[1], ","))
	for _, p := range namedPolicies {
		policy, present := policies[p]
		if !present {
			return fmt.Errorf("policy %q does not exist. add it with the policy command", p)
		}
		if policy.expiration != (Expiration{}) {
			policySet = append(policySet, policy.expiration)
		}
		for i := range policy.expirationNotification {
			policySet = append(policySet, policy.expirationNotification[i])
		}
		for i := range policy.noChangeNotification {
			policySet = append(policySet, policy.noChangeNotification[i])
		}
	}
	policyBytes, err := json.Marshal(policySet)
	// shell.Printf("Policies: %v\n", string(policyBytes))
	putParamInput.Policies = aws.String(string(policyBytes))
	putParamInput.Tier = aws.String(AdvancedTier)
	return nil
}
