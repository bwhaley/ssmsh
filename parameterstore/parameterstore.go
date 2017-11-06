package parameterstore

import (
	"errors"
	"fmt"
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
	Confirm bool   // TODO Prompt for confirmation to delete or overwrite
	Cwd     string // The current working directory in the hierarchy
	Decrypt bool   // Decrypt values retrieved from Get
	Key     string // The KMS key to use for SecureString parameters
	Client  ssmiface.SSMAPI
}

// NewParameterStore initializes a ParameterStore with default values
func (ps *ParameterStore) NewParameterStore() error {
	sess := session.Must(session.NewSession())
	ps.Confirm = false
	ps.Cwd = Delimiter
	ps.Decrypt = false
	ps.Client = ssm.New(sess)
	_, err := ps.List(Delimiter, false)
	if err != nil {
		return err
	}
	return nil
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
		return errors.New("No such path")
	}
	return nil
}

// List displays the parameters in a given path
// Behavior is vaguely similar to UNIX ls
func (ps *ParameterStore) List(path string, recurse bool) (r []string, err error) {
	var pathParam string
	path = fqp(path, ps.Cwd)
	param, err := ps.Get([]string{path})
	if err != nil {
		return nil, err
	}
	if len(param) == 1 {
		pathParam = aws.StringValue(param[0].Name)
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
	if !recurse {
		r = cull(r, path)
	}
	if pathParam != "" {
		r = append(r, pathParam)
	}
	return r, nil
}

// Delete removes one or more parameters
func (ps *ParameterStore) Remove(params []string, recurse bool) (err error) {
	var parametersToDelete []string
	for _, p := range params {
		if ps.isParameter(p) {
			parametersToDelete = append(parametersToDelete, p)
		} else if ps.isPath(p) {
			if recurse {
				additionalParams := &ssm.GetParametersByPathInput{
					Path:      aws.String(p),
					Recursive: aws.Bool(true),
				}
				for {
					resp, err := ps.Client.GetParametersByPath(additionalParams)
					if err != nil {
						return err
					}
					for _, r := range resp.Parameters {
						parametersToDelete = append(parametersToDelete, aws.StringValue(r.Name))
					}
					if aws.StringValue(resp.NextToken) == "" {
						break
					}
					additionalParams.NextToken = resp.NextToken
				}
			} else {
				return fmt.Errorf("Tried to delete path %s but recursive not requested.", p)
			}
		} else {
			return fmt.Errorf("No path or parameter %s was found, aborting", p)
		}
	}
	return ps.Delete(parametersToDelete)
}

func (ps *ParameterStore) Delete(params []string) (err error) {
	const maxParams = 10
	var invalidParams []string
	var arrayEnd int
	var deleteBatch []string
	for i := 0; i < len(params); i += maxParams {
		if len(params)-i < maxParams {
			arrayEnd = len(params)
		} else {
			arrayEnd = i + maxParams
		}
		deleteBatch = params[i:arrayEnd]
		ssmParams := &ssm.DeleteParametersInput{
			Names: ps.inputPaths(deleteBatch),
		}
		resp, err := ps.Client.DeleteParameters(ssmParams)
		if err != nil {
			return err
		}
		for _, r := range resp.InvalidParameters {
			invalidParams = append(invalidParams, aws.StringValue(r))
		}
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
func (ps *ParameterStore) Put(param *ssm.PutParameterInput) (resp *ssm.PutParameterOutput, err error) {
	resp, err = ps.Client.PutParameter(param)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// Move moves a parameter or path to another location
func (ps *ParameterStore) Move(src string, dst string) error {
	var err error
	if !ps.Decrypt {
		// Decryption required for copy
		ps.Decrypt = true
		defer func() {
			ps.Decrypt = false
		}()
	}
	err = ps.Copy(src, dst, true)
	if err != nil {
		return err
	}
	err = ps.Remove([]string{src}, true)
	if err != nil {
		return err
	}
	return nil
}

// Copy duplicates a parameter from src to dst
func (ps *ParameterStore) Copy(src string, dst string, recurse bool) error {
	if !ps.Decrypt {
		// Decryption required for copy
		ps.Decrypt = true
		defer func() {
			ps.Decrypt = false
		}()
	}
	src = fqp(src, ps.Cwd)
	dst = fqp(dst, ps.Cwd)
	if ps.isParameter(src) {
		if ps.isPath(dst) {
			_dst := strings.Split(src, Delimiter)
			dst = dst + Delimiter + _dst[len(_dst)-1]
			fmt.Println(dst)
		}
		return ps.copyParameter(src, dst)
	} else if ps.isPath(src) {
		if recurse {
			return ps.copyPath(src, dst)
		}
		return errors.New(src + " is a path")
	}
	return errors.New("Invalid source " + src)
}

func (ps *ParameterStore) copyPath(srcPath string, dstPath string) error {
	params := &ssm.GetParametersByPathInput{
		Path:      aws.String(srcPath),
		Recursive: aws.Bool(true),
	}
	getResp, err := ps.Client.GetParametersByPath(params)
	if err != nil {
		return err
	}
	var srcName string
	var dstName string
	for _, r := range getResp.Parameters {
		srcName = aws.StringValue(r.Name)
		dstName = strings.Join([]string{dstPath, srcName[len(srcPath)+1:]}, Delimiter)
		err = ps.copyParameter(srcName, dstName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *ParameterStore) copyParameter(src string, dst string) error {
	if !ps.isParameter(src) {
		return errors.New("source must be a parameter")
	}
	pHist, err := ps.GetHistory(src)
	if err != nil {
		return err
	}
	pLatest := pHist[len(pHist)-1]
	putParamInput := &ssm.PutParameterInput{
		Name:           aws.String(dst),
		Type:           pLatest.Type,
		Value:          pLatest.Value,
		KeyId:          pLatest.KeyId,
		Description:    pLatest.Description,
		AllowedPattern: pLatest.AllowedPattern,
		Overwrite:      aws.Bool(true), // TODO Prompt for overwrite
	}
	_, err = ps.Put(putParamInput)
	if err != nil {
		return err
	}
	return nil
}

// inputPaths cleans a list of parameter paths and returns strings
// suitable for use as ssm.Parameters
func (ps *ParameterStore) inputPaths(paths []string) []*string {
	var _paths []*string
	for i, p := range paths {
		paths[i] = fqp(p, ps.Cwd)
		_paths = append(_paths, aws.String(paths[i]))
	}
	return _paths
}

// fqp cleans a provided path
// relative paths are prefixed with cwd
// TODO Support regex or globbing
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
