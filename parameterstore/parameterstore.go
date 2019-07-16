package parameterstore

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

// Delimiter is the parameter path separator character
const Delimiter = "/"

// ParameterStore represents the current state and preferences of the shell
type ParameterStore struct {
	Confirm bool                       // TODO Prompt for confirmation to delete or overwrite
	Cwd     string                     // The current working directory in the hierarchy
	Decrypt bool                       // Decrypt values retrieved from Get
	Key     string                     // The KMS key to use for SecureString parameters
	Region  string                     // AWS region on which to operate
	Profile string                     // Profile to use from .aws/[config|credentials]
	Clients map[string]ssmiface.SSMAPI // per-region SSM clients
}

func newSession(region, profile string) *session.Session {
	return session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
				Config: aws.Config{
					Region:      aws.String(region),
					Credentials: credentials.NewSharedCredentials("", profile),
				},
			},
		),
	)
}

// NewParameterStore initializes a ParameterStore with default values
func (ps *ParameterStore) NewParameterStore() error {
	ps.Confirm = false
	ps.Cwd = Delimiter
	ps.Decrypt = false
	ps.Clients = make(map[string]ssmiface.SSMAPI)
	if ps.Profile == "" {
		ps.Profile = "default"
	}
	ps.Clients[ps.Region] = ssm.New(newSession(ps.Region, ps.Profile))

	// Check for a non-existent parameter to validate credentials & permissions
	_, err := ps.Get([]string{Delimiter}, ps.Region)
	if err != nil {
		return err
	}
	return nil
}

// InitClient initializes an SSM client in a given region
func (ps *ParameterStore) InitClient(region string) {
	if _, ok := ps.Clients[region]; !ok {
		ps.Clients[region] = ssm.New(newSession(region, ps.Profile))
	}
}

// ParameterPath abstracts a parameter to include some metadata
type ParameterPath struct {
	Name   string
	Region string
}

// SetCwd sets the current working dir within the parameter store
func (ps *ParameterStore) SetCwd(path ParameterPath) error {
	if path.Name == Delimiter {
		ps.Cwd = Delimiter
		return nil
	}
	path.Name = fqp(path.Name, ps.Cwd)
	if ps.isPath(path) {
		ps.Cwd = path.Name
	} else {
		return errors.New("No such path")
	}
	return nil
}

// ListResult contains the results and error message from a call to List()
type ListResult struct {
	Result []string
	Error  error
}

// List displays the parameters in a given path
// Behavior is vaguely similar to UNIX ls
func (ps *ParameterStore) List(ppath ParameterPath, recurse bool, lr chan ListResult, quit chan bool) {
	path := ppath.Name
	region := ppath.Region
	// Check for parameters under this path
	path = fqp(path, ps.Cwd)
	results := []string{}
	params := &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	for {
		select {
		case <-quit:
			return
		default:
		}
		resp, err := ps.Clients[region].GetParametersByPath(params)
		if err != nil {
			lr <- ListResult{nil, err}
		}
		for _, p := range resp.Parameters {
			results = append(results, aws.StringValue(p.Name))
		}
		if aws.StringValue(resp.NextToken) == "" {
			break
		}
		params.NextToken = resp.NextToken
	}
	if !recurse {
		results = cull(results, path)
	}

	// Check if this path is a parameter (could be both path & parameter)
	param, err := ps.Get([]string{path}, region)
	if err != nil {
		lr <- ListResult{nil, err}
		return
	}
	if len(param) == 1 {
		pathParam := aws.StringValue(param[0].Name)
		results = append(results, pathParam)
	}

	lr <- ListResult{results, nil}
}

// Remove removes one or more parameters
func (ps *ParameterStore) Remove(params []ParameterPath, recurse bool) (err error) {
	var parametersToDelete []ParameterPath
	for _, param := range params {
		param.Name = fqp(param.Name, ps.Cwd)
		if ps.isParameter(param) {
			parametersToDelete = append(parametersToDelete, param)
		} else if ps.isPath(param) {
			if recurse {
				err = ps.recursiveDelete(param)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("tried to delete path %s but recursive not requested", param.Name)
			}
		} else {
			return fmt.Errorf("No path or parameter %s was found, aborting", param.Name)
		}
	}
	return ps.deleteByRegion(parametersToDelete)
}

func (ps *ParameterStore) recursiveDelete(path ParameterPath) (err error) {
	var parametersToDelete []ParameterPath
	additionalParams := &ssm.GetParametersByPathInput{
		Path:      aws.String(path.Name),
		Recursive: aws.Bool(true),
	}
	for {
		resp, err := ps.Clients[path.Region].GetParametersByPath(additionalParams)
		if err != nil {
			return err
		}
		for _, r := range resp.Parameters {
			parametersToDelete = append(parametersToDelete, ParameterPath{
				Name:   aws.StringValue(r.Name),
				Region: path.Region,
			})
		}
		if aws.StringValue(resp.NextToken) == "" {
			break
		}
		additionalParams.NextToken = resp.NextToken
	}
	return ps.deleteByRegion(parametersToDelete)

}

// deleteByRegion groups parameters by region before calling delete()
func (ps *ParameterStore) deleteByRegion(params []ParameterPath) (err error) {
	const maxParams = 10
	paramsByRegion := make(map[string][]string)
	for _, p := range params {
		paramsByRegion[p.Region] = append(paramsByRegion[p.Region], p.Name)
	}
	for region, params := range paramsByRegion {
		err := ps.delete(params, region)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *ParameterStore) delete(params []string, region string) (err error) {
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
		resp, err := ps.Clients[region].DeleteParameters(ssmParams)
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

// GetHistory returns the parameter history
func (ps *ParameterStore) GetHistory(param ParameterPath) (r []ssm.ParameterHistory, err error) {
	history := &ssm.GetParameterHistoryInput{
		Name:           aws.String(fqp(param.Name, ps.Cwd)),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	for {
		resp, err := ps.Clients[param.Region].GetParameterHistory(history)
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
func (ps *ParameterStore) Get(params []string, region string) (r []ssm.Parameter, err error) {
	ssmParams := &ssm.GetParametersInput{
		Names:          ps.inputPaths(params),
		WithDecryption: aws.Bool(ps.Decrypt),
	}
	resp, err := ps.Clients[region].GetParameters(ssmParams)
	if err != nil {
		return nil, err
	}
	for _, p := range resp.Parameters {
		r = append(r, *p)
	}
	return r, nil
}

// Put creates or updates a parameter
func (ps *ParameterStore) Put(param *ssm.PutParameterInput, region string) (resp *ssm.PutParameterOutput, err error) {
	resp, err = ps.Clients[region].PutParameter(param)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// Move moves a parameter or path to another location
func (ps *ParameterStore) Move(src, dst ParameterPath) error {
	var err error
	err = ps.Copy(src, dst, true)
	if err != nil {
		return err
	}
	err = ps.Remove([]ParameterPath{src}, true)
	if err != nil {
		return err
	}
	return nil
}

// Copy duplicates a parameter from src to dst
func (ps *ParameterStore) Copy(src, dst ParameterPath, recurse bool) error {
	src.Name = fqp(src.Name, ps.Cwd)
	dst.Name = fqp(dst.Name, ps.Cwd)
	var srcIsParameter, dstIsParameter, srcIsPath, dstIsPath bool
	if !ps.Decrypt {
		// Decryption required for copy
		ps.Decrypt = true
		defer func() {
			ps.Decrypt = false
		}()
	}
	srcIsParameter = ps.isParameter(src)
	if !srcIsParameter {
		srcIsPath = ps.isPath(src)
	}
	dstIsParameter = ps.isParameter(dst)
	if !dstIsParameter {
		dstIsPath = ps.isPath(dst)
	}
	if srcIsParameter && !dstIsPath {
		return ps.copyParameter(src, dst)
	} else if srcIsParameter && dstIsPath {
		return ps.copyParameterToPath(src, dst)
	} else if srcIsPath && dstIsParameter {
		return fmt.Errorf("Cannot copy path (%s) to parameter (%s)", src, dst)
	} else if srcIsPath {
		if !recurse {
			return fmt.Errorf("%s and %s are both paths but recursion not requested. Use -R", src, dst)
		}
		if dstIsPath {
			return ps.copyPathToPath(false, src, dst)
		}
		return ps.copyPathToPath(true, src, dst)
	}
	return fmt.Errorf("%s is not a path or parameter", src)
}

func (ps *ParameterStore) copyParameter(src, dst ParameterPath) error {
	if !ps.isParameter(src) {
		return errors.New("source must be a parameter: " + src.Name)
	}
	pHist, err := ps.GetHistory(src)
	if err != nil {
		return err
	}
	pLatest := pHist[len(pHist)-1]
	if dst.Name == Delimiter {
		dst.Name = src.Name
	}
	putParamInput := &ssm.PutParameterInput{
		Name:           aws.String(dst.Name),
		Type:           pLatest.Type,
		Value:          pLatest.Value,
		KeyId:          pLatest.KeyId,
		Description:    pLatest.Description,
		AllowedPattern: pLatest.AllowedPattern,
		Overwrite:      aws.Bool(true), // TODO Prompt for overwrite
	}
	_, err = ps.Put(putParamInput, dst.Region)
	if err != nil {
		return err
	}
	return nil
}

func (ps *ParameterStore) copyParameterToPath(srcParam, dstPath ParameterPath) error {
	srcParamElements := strings.Split(srcParam.Name, Delimiter)
	dstPath.Name = dstPath.Name + Delimiter + srcParamElements[len(srcParamElements)-1]
	return ps.copyParameter(srcParam, dstPath)
}

func (ps *ParameterStore) copyPathToPath(newPath bool, srcPath, dstPath ParameterPath) error {
	/*
		1) Get all source parameters
		2) Map sources to destinations
		3) Create destinations
	*/
	params := &ssm.GetParametersByPathInput{
		Path:      aws.String(srcPath.Name),
		Recursive: aws.Bool(true),
	}
	for {
		resp, err := ps.Clients[srcPath.Region].GetParametersByPath(params)
		if err != nil {
			return err
		}
		paramMap := makeParameterMap(resp.Parameters, newPath, srcPath, dstPath)
		for src, dst := range paramMap {
			err = ps.copyParameter(src, dst)
			if err != nil {
				return err
			}
		}
		if aws.StringValue(resp.NextToken) == "" {
			break
		}
		params.NextToken = resp.NextToken
	}
	return nil
}

// makeParameterMap returns a map of source param name to dest param name
func makeParameterMap(params []*ssm.Parameter, newPath bool, srcPath, dstPath ParameterPath) (sourceToDst map[ParameterPath]ParameterPath) {
	/*
		sample input:
			params: [/House/Stark/JonSnow /House/Stark/Special/Bran]
			srcPath: /House/Stark
			dstPath: /House/Targaryen
	*/
	sourceToDst = make(map[ParameterPath]ParameterPath)
	for _, p := range params {
		srcParam := ParameterPath{
			Name:   aws.StringValue(p.Name),
			Region: srcPath.Region,
		}
		srcPathElements := strings.Split(srcPath.Name, Delimiter)
		srcBasePath := srcPathElements[len(srcPathElements)-1]
		dstParamName := string(srcParam.Name[len(srcPath.Name)+1:])

		if dstPath.Name == Delimiter {
			dstPath.Name = ""
		}

		var name string
		if newPath {
			name = strings.Join(
				[]string{dstPath.Name, dstParamName},
				Delimiter)
		} else {
			name = strings.Join(
				[]string{dstPath.Name, srcBasePath, dstParamName},
				Delimiter)
		}

		dstParam := ParameterPath{
			Name:   name,
			Region: dstPath.Region,
		}
		sourceToDst[srcParam] = dstParam
	}
	return sourceToDst
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
func (ps *ParameterStore) isParameter(param ParameterPath) bool {
	p := &ssm.GetParameterInput{
		Name: aws.String(param.Name),
	}
	_, err := ps.Clients[param.Region].GetParameter(p)
	if err != nil {
		return false
	}
	return true
}

// isPath checks for the existence of at least one key under path
func (ps *ParameterStore) isPath(path ParameterPath) bool {
	var err error
	params := &ssm.GetParametersByPathInput{
		Path:      aws.String(path.Name),
		Recursive: aws.Bool(true),
	}
	resp, err := ps.Clients[path.Region].GetParametersByPath(params)
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
