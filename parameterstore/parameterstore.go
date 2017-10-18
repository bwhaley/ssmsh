package parameterstore

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Delimiter is the parameter path separator character
const Delimiter = "/"

// ParameterStore represents the current state and preferences of the shell
type ParameterStore struct {
	Confirm   bool   // TODO Prompt for confirmation to delete or overwrite
	Cwd       string // The current working directory in the hierarchy
	Decrypt   bool   // Decrypt values retrieved from Get
	Key       string // The KMS key to use for SecureString parameters
	Overwrite bool   // Overwrite parameters with Put, Mv or Cp
	Recurse   bool   // List parameters recursively
}

var ssmsvc *ssm.SSM

// NewParameterStore initializes a ParameterStore with default values
func NewParameterStore() *ParameterStore {
	ps := new(ParameterStore)
	ps.Confirm = false
	ps.Cwd = Delimiter
	ps.Decrypt = false
	ps.Overwrite = false
	ps.Recurse = false
	ssmsvc = ssm.New(session.New())
	return ps
}

// SetCwd sets the current working dir within the parameter store
func (ps *ParameterStore) SetCwd(path string) error {
	path = fqp(path, ps.Cwd)
	if parameterPathExists(path) {
		ps.Cwd = path
	} else {
		return errors.New("No such path ")
	}
	return nil
}

// List displays the parameters in a given path
// if recursive listing is requested, displays recursive listing of subdirectories in the provided path
// if recursive is not enabled, behavior is similar to UNIX ls
func (ps *ParameterStore) List(path string) (r []string, err error) {
	path = fqp(path, ps.Cwd)
	params := &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	for {
		resp, err := ssmsvc.GetParametersByPath(params)
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

// Rm removes one or more parameters
func (ps *ParameterStore) Rm(params []string) error {
	var err error
	ssmParams := &ssm.DeleteParametersInput{
		Names: ps.inputPaths(params),
	}
	resp, err := ssmsvc.DeleteParameters(ssmParams)
	fmt.Println(resp)
	if err != nil {
		return err
	}
	return nil
}

// GetHistory gets the details of a slice of parameters
func (ps *ParameterStore) GetHistory(param string) (r []ssm.ParameterHistory, err error) {
	history := &ssm.GetParameterHistoryInput{
		Name:           aws.String(fqp(param, ps.Cwd)),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	for {
		resp, err := ssmsvc.GetParameterHistory(history)
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

// Get retrieves parameters
func (ps *ParameterStore) Get(params []string) (r []ssm.Parameter, err error) {
	ssmParams := &ssm.GetParametersInput{
		Names:          ps.inputPaths(params),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	resp, err := ssmsvc.GetParameters(ssmParams)
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
	_, err := ssmsvc.PutParameter(param)
	if err != nil {
		return err
	}
	return nil
}

// Cp copies a paramter from src to dest
func (ps *ParameterStore) Cp(src string, dest string) (err error) {
	return nil
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

// parameterPathExists checks for the existence of at least one key under path
func parameterPathExists(path string) bool {
	params := &ssm.GetParametersByPathInput{
		Path:      aws.String(path),
		Recursive: aws.Bool(true),
	}
	resp, _ := ssmsvc.GetParametersByPath(params)
	if len(resp.Parameters) > 0 {
		return true
	}
	return false
}

// cull removes all but the top level results (relative to Cwd) from a list of paths
func cull(paths []string, relative string) (culled []string) {
	var r []string
	for _, p := range paths {
		if relative == Delimiter {
			p = p[1:]
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
