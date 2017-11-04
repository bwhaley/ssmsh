package commands

import (
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/kountable/pssh/parameterstore"
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
... value="Queen"
... type="String"
... description="Queen of the Seven Kingdoms"
... key=arn:aws:kms:us-west-2:012345678901:key/321ec4ec-ed00-427f-9729-748ba2254794
... overwrite=true
... pattern=[A-z]+
...
`

var putParamInput ssm.PutParameterInput

// Add or update parameters
func put(c *ishell.Context) {
	var err error
	var resp *ssm.PutParameterOutput
	putParamInput = ssm.PutParameterInput{}
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
	resp, err = ps.Put(&putParamInput)
	if err != nil {
		shell.Println("Error: ", err)
	} else {
		version := strconv.Itoa(int(aws.Int64Value(resp.Version)))
		if version != "" {
			shell.Println("Put version " + version)
		}
	}
}

func multiLinePut() bool {
	// Set the prompt explicitly rather than use SetMultiPrompt
	// due to the unexpected 2nd line behavior
	shell.SetPrompt("... ")
	defer setPrompt(ps.Cwd)
	shell.Println("Input options. End with a blank line.")
	shell.ReadMultiLinesFunc(putOptions)
	if putParamInput == (ssm.PutParameterInput{}) {
		return false
	}
	return true
}

func inlinePut(options []string) bool {
	for _, p := range options {
		if putOptions(p) == false {
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
		shell.Println("Invalid input.")
		shell.Println(putUsage)
		return false
	}
	field := strings.ToLower(paramOption[0])
	val := strings.Join(paramOption[1:], "=")
	if validate(field, val) {
		return true
	}
	return false
}

func validate(f string, v string) bool {
	m := map[string]func(string) bool{
		"type":        validateType,
		"name":        validateName,
		"value":       validateValue,
		"description": validateDescription,
		"key":         validateKey,
		"pattern":     validatePattern,
		"overwrite":   validateOverwrite,
	}
	if validator, ok := m[strings.ToLower(f)]; ok {
		if validator(v) {
			return true
		}
	}
	shell.Println("Input error.")
	shell.Println(putUsage)
	putParamInput = ssm.PutParameterInput{}
	return false
}

func validateType(s string) bool {
	validTypes := []string{"String", "StringList", "SecureString"}
	for i := 0; i < len(validTypes); i++ {
		if strings.EqualFold(s, validTypes[i]) {
			putParamInput.Type = aws.String(validTypes[i])
			return true
		}
	}
	shell.Println("Invalid type " + s)
	return false
}

func validateValue(s string) bool {
	putParamInput.Value = aws.String(s)
	return true
}

func validateName(s string) bool {
	if strings.HasPrefix(s, parameterstore.Delimiter) {
		putParamInput.Name = aws.String(s)
	} else {
		putParamInput.Name = aws.String(ps.Cwd + parameterstore.Delimiter + s)
	}
	return true
}

func validateDescription(s string) bool {
	putParamInput.Description = aws.String(s)
	return true
}

func validateKey(s string) bool {
	putParamInput.KeyId = aws.String(s)
	return true
}

func validatePattern(s string) bool {
	putParamInput.AllowedPattern = aws.String(s)
	return true
}

func validateOverwrite(s string) bool {
	overwrite, err := strconv.ParseBool(s)
	if err != nil {
		shell.Println("overwrite must be true or false")
		return false
	}
	putParamInput.Overwrite = aws.Bool(overwrite)
	return true
}
