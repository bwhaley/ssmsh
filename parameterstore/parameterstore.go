package parameterstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	saws "github.com/bwhaley/ssmsh/aws"
	"github.com/bwhaley/ssmsh/config"
)

const Delimiter = "/"
const DefaultParameterType = "SecureString"

// API contains only the SSM operations used by this application.
type API interface {
	GetParameter(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParameters(context.Context, *ssm.GetParametersInput, ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
	GetParametersByPath(context.Context, *ssm.GetParametersByPathInput, ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
	GetParameterHistory(context.Context, *ssm.GetParameterHistoryInput, ...func(*ssm.Options)) (*ssm.GetParameterHistoryOutput, error)
	PutParameter(context.Context, *ssm.PutParameterInput, ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
	DeleteParameters(context.Context, *ssm.DeleteParametersInput, ...func(*ssm.Options)) (*ssm.DeleteParametersOutput, error)
	ListTagsForResource(context.Context, *ssm.ListTagsForResourceInput, ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error)
	AddTagsToResource(context.Context, *ssm.AddTagsToResourceInput, ...func(*ssm.Options)) (*ssm.AddTagsToResourceOutput, error)
}

type ParameterStore struct {
	Cwd, Type, Key, Region, Profile string
	Decrypt, Overwrite, DryRun      bool
	Clients                         map[string]API
	Actions                         []Action
}

type ParameterPath struct{ Name, Region string }
type Action struct {
	Operation   string
	Source      *ParameterPath `json:",omitempty"`
	Destination ParameterPath
}

func (ps *ParameterStore) SetDefaults(cfg config.Config) {
	ps.Cwd = Delimiter
	ps.Decrypt, ps.Overwrite = cfg.Default.Decrypt, cfg.Default.Overwrite
	ps.Key, ps.Type = cfg.Default.Key, cfg.Default.Type
	if ps.Type == "" {
		ps.Type = DefaultParameterType
	}
	ps.Profile = os.Getenv("AWS_PROFILE")
	if ps.Profile == "" {
		ps.Profile = cfg.Default.Profile
	}
	// Keep an empty profile to allow the SDK's environment/workload credential chain.
	ps.Region = os.Getenv("AWS_REGION")
	if ps.Region == "" {
		ps.Region = cfg.Default.Region
	}
}

func (ps *ParameterStore) InitClient(ctx context.Context, region string) (API, error) {
	if client := ps.Clients[region]; client != nil {
		return client, nil
	}
	cfg, err := saws.Load(ctx, region, ps.Profile)
	if err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		return nil, errors.New("no AWS region configured; set AWS_REGION, .ssmshrc region, or your AWS profile region")
	}
	if ps.Clients == nil {
		ps.Clients = make(map[string]API)
	}
	client := ssm.NewFromConfig(cfg)
	ps.Clients[region] = client
	if region == "" {
		ps.Region = cfg.Region
		ps.Clients[cfg.Region] = client
	}
	return client, nil
}

// Switch validates configuration before replacing the active client cache.
func (ps *ParameterStore) Switch(ctx context.Context, region, profile string) error {
	next := *ps
	next.Region, next.Profile, next.Clients = region, profile, nil
	if _, err := next.InitClient(ctx, region); err != nil {
		return err
	}
	*ps = next
	return nil
}

func (ps *ParameterStore) Resolve(name string) string {
	if strings.HasPrefix(name, Delimiter) {
		return path.Clean(name)
	}
	return path.Join(Delimiter, ps.Cwd, name)
}
func (ps *ParameterStore) normalize(p ParameterPath) ParameterPath {
	p.Name = ps.Resolve(p.Name)
	if p.Region == "" {
		p.Region = ps.Region
	}
	return p
}

func (ps *ParameterStore) parameter(ctx context.Context, p ParameterPath, decrypt bool) (*types.Parameter, error) {
	client, err := ps.InitClient(ctx, p.Region)
	if err != nil {
		return nil, err
	}
	out, err := client.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(p.Name), WithDecryption: aws.Bool(decrypt)})
	var missing *types.ParameterNotFound
	if errors.As(err, &missing) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return out.Parameter, nil
}

// descendants follows empty pages as well as populated ones and never decrypts values for listing.
func (ps *ParameterStore) descendants(ctx context.Context, p ParameterPath) ([]types.Parameter, error) {
	client, err := ps.InitClient(ctx, p.Region)
	if err != nil {
		return nil, err
	}
	pager := ssm.NewGetParametersByPathPaginator(client, &ssm.GetParametersByPathInput{Path: aws.String(p.Name), Recursive: aws.Bool(true), WithDecryption: aws.Bool(false)}, func(o *ssm.GetParametersByPathPaginatorOptions) { o.StopOnDuplicateToken = true })
	var result []types.Parameter
	for pager.HasMorePages() {
		out, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, out.Parameters...)
	}
	return result, nil
}

func (ps *ParameterStore) SetCwd(ctx context.Context, p ParameterPath) error {
	p = ps.normalize(p)
	if p.Name != Delimiter {
		params, err := ps.descendants(ctx, p)
		if err != nil {
			return err
		}
		if len(params) == 0 {
			return fmt.Errorf("no such path: %s", p.Name)
		}
	}
	ps.Cwd, ps.Region = p.Name, p.Region
	return nil
}

func (ps *ParameterStore) List(ctx context.Context, p ParameterPath, recursive bool) ([]string, error) {
	p = ps.normalize(p)
	params, err := ps.descendants(ctx, p)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool)
	for _, param := range params {
		name := aws.ToString(param.Name)
		if !recursive {
			name = strings.TrimPrefix(name, strings.TrimSuffix(p.Name, "/")+"/")
			if i := strings.IndexByte(name, '/'); i >= 0 {
				name = name[:i+1]
			}
		}
		names[name] = true
	}
	if p.Name != Delimiter {
		param, err := ps.parameter(ctx, p, false)
		if err != nil {
			return nil, err
		}
		if param != nil {
			names[aws.ToString(param.Name)] = true
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func (ps *ParameterStore) Get(ctx context.Context, names []string, region string) ([]types.Parameter, error) {
	client, err := ps.InitClient(ctx, region)
	if err != nil {
		return nil, err
	}
	result := make([]types.Parameter, 0, len(names))
	for start := 0; start < len(names); start += 10 {
		batch := make([]string, 0, 10)
		for _, name := range names[start:min(start+10, len(names))] {
			batch = append(batch, ps.Resolve(name))
		}
		out, err := client.GetParameters(ctx, &ssm.GetParametersInput{Names: batch, WithDecryption: aws.Bool(ps.Decrypt)})
		if err != nil {
			return nil, err
		}
		if len(out.InvalidParameters) != 0 {
			return nil, fmt.Errorf("parameters not found: %s", strings.Join(out.InvalidParameters, ", "))
		}
		result = append(result, out.Parameters...)
	}
	sort.Slice(result, func(i, j int) bool { return aws.ToString(result[i].Name) < aws.ToString(result[j].Name) })
	return result, nil
}

func (ps *ParameterStore) GetHistory(ctx context.Context, p ParameterPath) ([]types.ParameterHistory, error) {
	p = ps.normalize(p)
	client, err := ps.InitClient(ctx, p.Region)
	if err != nil {
		return nil, err
	}
	pager := ssm.NewGetParameterHistoryPaginator(client, &ssm.GetParameterHistoryInput{Name: aws.String(p.Name), WithDecryption: aws.Bool(ps.Decrypt)}, func(o *ssm.GetParameterHistoryPaginatorOptions) { o.StopOnDuplicateToken = true })
	result := []types.ParameterHistory{}
	for pager.HasMorePages() {
		out, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, out.Parameters...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Version < result[j].Version })
	return result, nil
}

func (ps *ParameterStore) Put(ctx context.Context, in *ssm.PutParameterInput, region string) (*ssm.PutParameterOutput, error) {
	if ps.DryRun {
		ps.Actions = append(ps.Actions, Action{Operation: "put", Destination: ParameterPath{aws.ToString(in.Name), region}})
		return &ssm.PutParameterOutput{}, nil
	}
	client, err := ps.InitClient(ctx, region)
	if err != nil {
		return nil, err
	}
	return client.PutParameter(ctx, in)
}

// collect snapshots the exact set of names before any mutation.
func (ps *ParameterStore) collect(ctx context.Context, p ParameterPath, recursive bool) ([]ParameterPath, error) {
	p = ps.normalize(p)
	var result []ParameterPath
	if p.Name != Delimiter {
		param, err := ps.parameter(ctx, p, false)
		if err != nil {
			return nil, err
		}
		if param != nil {
			result = append(result, p)
		}
	}
	if recursive || len(result) == 0 {
		params, err := ps.descendants(ctx, p)
		if err != nil {
			return nil, err
		}
		if !recursive && len(params) > 0 {
			return nil, fmt.Errorf("%s is a path; use -r", p.Name)
		}
		for _, param := range params {
			result = append(result, ParameterPath{aws.ToString(param.Name), p.Region})
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no path or parameter %s", p.Name)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (ps *ParameterStore) Remove(ctx context.Context, params []ParameterPath, recursive bool) error {
	var all []ParameterPath
	seen := map[ParameterPath]bool{}
	for _, p := range params {
		found, err := ps.collect(ctx, p, recursive)
		if err != nil {
			return err
		}
		for _, p := range found {
			if !seen[p] {
				seen[p] = true
				all = append(all, p)
			}
		}
	}
	return ps.delete(ctx, all)
}

func (ps *ParameterStore) delete(ctx context.Context, params []ParameterPath) error {
	// Sort for deterministic plans and contiguous regional batches.
	sort.Slice(params, func(i, j int) bool {
		if params[i].Region != params[j].Region {
			return params[i].Region < params[j].Region
		}
		return params[i].Name < params[j].Name
	})
	completed := 0
	for i := 0; i < len(params); {
		region := params[i].Region
		var names []string
		for i < len(params) && params[i].Region == region && len(names) < 10 {
			names = append(names, params[i].Name)
			ps.Actions = append(ps.Actions, Action{Operation: "delete", Destination: params[i]})
			i++
		}
		if ps.DryRun {
			continue
		}
		client, err := ps.InitClient(ctx, region)
		if err != nil {
			return err
		}
		out, err := client.DeleteParameters(ctx, &ssm.DeleteParametersInput{Names: names})
		if err != nil {
			return fmt.Errorf("delete failed in %s after %d confirmed deletions (current batch may be partially applied): %w", region, completed, err)
		}
		completed += len(out.DeletedParameters)
		if len(out.InvalidParameters) > 0 {
			return fmt.Errorf("deleted %d parameters; could not delete: %s", completed, strings.Join(out.InvalidParameters, ", "))
		}
	}
	return nil
}

func (ps *ParameterStore) Copy(ctx context.Context, src, dst ParameterPath, recursive bool) error {
	_, err := ps.copy(ctx, src, dst, recursive)
	return err
}
func (ps *ParameterStore) Move(ctx context.Context, src, dst ParameterPath) error {
	copied, err := ps.copy(ctx, src, dst, true)
	if err != nil {
		return err
	} // Never delete sources after an incomplete copy.
	if err := ps.delete(ctx, copied); err != nil {
		return fmt.Errorf("destinations copied; source cleanup incomplete: %w", err)
	}
	return nil
}

type copyItem struct {
	src, dst ParameterPath
	input    *ssm.PutParameterInput
	tags     []types.Tag
}

func (ps *ParameterStore) copy(ctx context.Context, src, dst ParameterPath, recursive bool) ([]ParameterPath, error) {
	src, dst = ps.normalize(src), ps.normalize(dst)
	if src.Region == dst.Region && (src.Name == dst.Name || strings.HasPrefix(dst.Name, strings.TrimSuffix(src.Name, "/")+"/")) {
		return nil, errors.New("destination must not be the source or inside the source hierarchy")
	}
	sources, err := ps.collect(ctx, src, recursive)
	if err != nil {
		return nil, err
	}
	dstParam, err := ps.parameter(ctx, dst, false)
	if err != nil {
		return nil, err
	}
	dstIsPath := dst.Name == Delimiter
	if dstParam == nil && !dstIsPath {
		children, err := ps.descendants(ctx, dst)
		if err != nil {
			return nil, err
		}
		dstIsPath = len(children) > 0
	}
	sourceIsTree := len(sources) > 1 || sources[0].Name != src.Name
	if sourceIsTree && dstParam != nil {
		return nil, errors.New("cannot copy a hierarchy onto a parameter")
	}
	var plan []copyItem
	sourceSet := map[ParameterPath]bool{}
	for _, p := range sources {
		sourceSet[p] = true
	}
	for _, p := range sources {
		target := dst
		if sourceIsTree {
			base := dst.Name
			if dstIsPath {
				base = path.Join(base, path.Base(src.Name))
			}
			relative := strings.TrimPrefix(p.Name, strings.TrimSuffix(src.Name, "/"))
			target.Name = path.Join(base, relative)
		} else if dstIsPath {
			target.Name = path.Join(dst.Name, path.Base(p.Name))
		}
		if sourceSet[target] {
			return nil, fmt.Errorf("destination %s overlaps a source", target.Name)
		}
		plan = append(plan, copyItem{src: p, dst: target})
	}
	// Read all metadata before writing, so read/permission failures cannot cause partial copies.
	for i := range plan {
		item := &plan[i]
		client, err := ps.InitClient(ctx, item.src.Region)
		if err != nil {
			return nil, err
		}
		temp := *ps
		temp.Decrypt = true
		history, err := temp.GetHistory(ctx, item.src)
		if err != nil {
			return nil, err
		}
		if len(history) == 0 {
			return nil, fmt.Errorf("empty history for %s", item.src.Name)
		}
		latest := history[len(history)-1]
		key := latest.KeyId
		if latest.Type == types.ParameterTypeSecureString {
			if ps.Key != "" {
				key = aws.String(ps.Key)
			} else if src.Region != dst.Region {
				return nil, errors.New("cross-region SecureString copy requires a destination key; use key=... or the key command")
			}
		}
		policies := []json.RawMessage{}
		for _, policy := range latest.Policies {
			if policy.PolicyText != nil {
				policies = append(policies, json.RawMessage(*policy.PolicyText))
			}
		}
		policyJSON, err := json.Marshal(policies)
		if err != nil {
			return nil, err
		}
		item.input = &ssm.PutParameterInput{Name: aws.String(item.dst.Name), Value: latest.Value, Type: latest.Type, KeyId: key, Description: latest.Description, AllowedPattern: latest.AllowedPattern, Tier: latest.Tier, DataType: latest.DataType, Overwrite: aws.Bool(ps.Overwrite)}
		if len(policies) > 0 {
			item.input.Policies = aws.String(string(policyJSON))
		}
		tags, err := client.ListTagsForResource(ctx, &ssm.ListTagsForResourceInput{ResourceType: types.ResourceTypeForTaggingParameter, ResourceId: aws.String(item.src.Name)})
		if err != nil {
			return nil, err
		}
		item.tags = tags.TagList
	}
	for i, item := range plan {
		ps.Actions = append(ps.Actions, Action{Operation: "copy", Source: &item.src, Destination: item.dst})
		if ps.DryRun {
			continue
		}
		client, err := ps.InitClient(ctx, item.dst.Region)
		if err != nil {
			return nil, err
		}
		_, err = client.PutParameter(ctx, item.input)
		if err != nil {
			return nil, fmt.Errorf("copy failed at %s:%s after %d completed copies; sources retained (failed write outcome may be unknown): %w", item.dst.Region, item.dst.Name, i, err)
		}
		if len(item.tags) > 0 {
			_, err = client.AddTagsToResource(ctx, &ssm.AddTagsToResourceInput{ResourceType: types.ResourceTypeForTaggingParameter, ResourceId: aws.String(item.dst.Name), Tags: item.tags})
			if err != nil {
				return nil, fmt.Errorf("value copied to %s but tags failed after %d completed copies; sources retained: %w", item.dst.Name, i, err)
			}
		}
	}
	return sources, nil
}
