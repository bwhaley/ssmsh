package parameterstore

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

// Delimiter is the parameter path separator character
const Delimiter = "/"

// ParameterStore represents the current state and preferences of the shell
type ParameterStore struct {
	Confirm   bool   // TODO Prompt for confirmation to delete or overwrite
	Cwd       string // The current working directory in the hierarchy
	Decrypt   bool   // Decrypt values retrieved from Get
	Key       string // The KMS key to use for SecureString parameters
	Overwrite bool   // Overwrite parameters with Put, Move or Copy
	Recurse   bool   // List parameters recursively
	Client    ssmiface.SSMAPI
}

// NewParameterStore initializes a ParameterStore with default values
func (ps *ParameterStore) NewParameterStore() {
	sess := session.Must(session.NewSession())
	ps.Confirm = false
	ps.Cwd = Delimiter
	ps.Decrypt = false
	ps.Overwrite = false
	ps.Recurse = false
	ps.Client = ssm.New(sess)
}

// SetCwd sets the current working dir within the parameter store
func (ps *ParameterStore) SetCwd(path string) error {
	if path == Delimiter {
		ps.Cwd = path
		return nil
	}
	path = fqp(path, ps.Cwd)
	if ps.isPath(path) {
		ps.Cwd = path
	} else {
		return errors.New("No such path ")
	}
	return nil
}

// List displays the parameters in a given path
// Behavior is vaguely similar to UNIX ls
func (ps *ParameterStore) List(path string) (r []string, err error) {
	path = fqp(path, ps.Cwd)
	param, err := ps.Get([]string{path})
	if err != nil {
		return nil, err
	}
	if len(param) == 1 {
		r = append(r, aws.StringValue(param[0].Name))
		return r, nil
	}
	params := &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	for {
		resp, err := ps.Client.GetParametersByPath(params)
		if err != nil {
			return nil, err
		}
		for _, p := range resp.Parameters {
			r = append(r, aws.StringValue(p.Name))
		}
		if aws.StringValue(resp.NextToken) == "" {
			break
		}
		params.NextToken = resp.NextToken
	}
	if ps.Recurse {
		return r, nil
	}
	return cull(r, path), nil
}

// Delete removes one or more parameters
func (ps *ParameterStore) Delete(params []string) error {
	var err error
	ssmParams := &ssm.DeleteParametersInput{
		Names: ps.inputPaths(params),
	}
	resp, err := ps.Client.DeleteParameters(ssmParams)
	if err != nil {
		return err
	}
	var invalidParams []string
	for _, r := range resp.InvalidParameters {
		invalidParams = append(invalidParams, aws.StringValue(r))
	}
	if len(invalidParams) > 0 {
		return errors.New("Could not delete invalid parameters " + strings.Join(invalidParams, ","))
	}
	return nil
}

// GetHistory returns the details and history of a parameter
func (ps *ParameterStore) GetHistory(param string) (r []ssm.ParameterHistory, err error) {
	history := &ssm.GetParameterHistoryInput{
		Name:           aws.String(fqp(param, ps.Cwd)),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	for {
		resp, err := ps.Client.GetParameterHistory(history)
		if err != nil {
			return nil, err
		}
		for _, p := range resp.Parameters {
			r = append(r, *p)
		}
		if aws.StringValue(resp.NextToken) == "" {
			break
		}
		history.NextToken = resp.NextToken
	}
	return r, nil
}

// Get retrieves one or more parameters
func (ps *ParameterStore) Get(params []string) (r []ssm.Parameter, err error) {
	ssmParams := &ssm.GetParametersInput{
		Names:          ps.inputPaths(params),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	resp, err := ps.Client.GetParameters(ssmParams)
	if err != nil {
		return nil, err
	}
	for _, p := range resp.Parameters {
		r = append(r, *p)
	}
	return r, nil
}

// Put creates or updates a parameter
func (ps *ParameterStore) Put(param *ssm.PutParameterInput) error {
	_, err := ps.Client.PutParameter(param)
	if err != nil {
		return err
	}
	return nil
}

// Copy duplicates a parameter from src to dest
func (ps *ParameterStore) Copy(src string, dest string) (err error) {
	if !ps.Decrypt {
		// Decryption required for copy
		ps.Decrypt = true
		defer func() {
			ps.Decrypt = false
		}()
	}
	src = fqp(src, ps.Cwd)
	dest = fqp(dest, ps.Cwd)
	if ps.isParameter(src) {
		return ps.copyParameter(src, dest)
	} else if ps.isPath(src) {
		if ps.Recurse {
			return ps.copyPath(src, dest)
		}
		return errors.New(src + " is a path")
	}
	return errors.New("Invalid source " + src)
}

func (ps *ParameterStore) copyPath(srcPath string, destPath string) (err error) {
	params := &ssm.GetParametersByPathInput{
		Path:      aws.String(srcPath),
		Recursive: aws.Bool(true),
	}
	resp, err := ps.Client.GetParametersByPath(params)
	if err != nil {
		return err
	}
	var srcName string
	var destName string
	for _, r := range resp.Parameters {
		srcName = aws.StringValue(r.Name)
		destName = strings.Join([]string{destPath, srcName[len(srcPath)+1:]}, Delimiter)
		err = ps.copyParameter(srcName, destName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *ParameterStore) copyParameter(src string, dest string) (err error) {
	if !ps.isParameter(src) {
		return errors.New("source must be a parameter")
	}
	pHist, err := ps.GetHistory(src)
	if err != nil {
		return nil
	}
	pLatest := pHist[len(pHist)-1]
	putParamInput := &ssm.PutParameterInput{
		Name:           aws.String(dest),
		Type:           pLatest.Type,
		Value:          pLatest.Value,
		KeyId:          pLatest.KeyId,
		Description:    pLatest.Description,
		AllowedPattern: pLatest.AllowedPattern,
		Overwrite:      aws.Bool(true),
	}
	return ps.Put(putParamInput)
}

// inputPaths cleans a list of parameter paths and returns a slice suitable for ssm inputs
func (ps *ParameterStore) inputPaths(paths []string) []*string {
	var _paths []*string
	for i, p := range paths {
		paths[i] = fqp(p, ps.Cwd)
		_paths = append(_paths, aws.String(paths[i]))
	}
	return _paths
}

// TODO Support regex
// fqp cleans a provided path
// relative paths are prefixed with cwd
func fqp(path string, cwd string) string {
	var dirtyPath string
	if strings.HasPrefix(path, Delimiter) {
		// fully qualified path
		dirtyPath = path
	} else {
		// relative to cwd
		dirtyPath = cwd + Delimiter + path
	}
	return filepath.Clean(dirtyPath)
}

// isParameter checks for the existence of a parameter
func (ps *ParameterStore) isParameter(param string) bool {
	p := &ssm.GetParameterInput{
		Name: aws.String(param),
	}
	_, err := ps.Client.GetParameter(p)
	if err != nil {
		return false
	}
	return true
}

// isPath checks for the existence of at least one key under path
func (ps *ParameterStore) isPath(path string) bool {
	var err error
	params := &ssm.GetParametersByPathInput{
		Path:      aws.String(path),
		Recursive: aws.Bool(true),
	}
	resp, err := ps.Client.GetParametersByPath(params)
	if err != nil {
		return false
	}
	if len(resp.Parameters) > 0 {
		return true
	}
	return false
}

// cull removes all but the top level results (relative to a provided path) from a list of paths
func cull(paths []string, relative string) (culled []string) {
	var r []string
	for _, p := range paths {
		if relative == Delimiter {
			// Parameters in the top level of the hierarchy are not prefixed with the delimiter
			// when returned from SSM API. Therefore we strip the first character except
			// for root-level parameters
			if string(p[0]) == Delimiter {
				p = p[1:]
			}
		} else {
			p = p[len(relative)+1:]
		}
		r = strings.Split(p, Delimiter)
		if len(r) > 1 {
			r[0] += Delimiter
		}
		culled = append(culled, r[0])
	}
	return uniq(culled)
}

// uniq removes duplicates
func uniq(input []string) (uniques []string) {
	m := make(map[string]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			uniques = append(uniques, val)
		}
	}
	return uniques
}
