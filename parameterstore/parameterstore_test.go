package parameterstore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type fakeSSM struct {
	API
	params        map[string]types.Parameter
	pages         []*ssm.GetParametersByPathOutput
	pageCalls     int
	history       []types.ParameterHistory
	tags          []types.Tag
	puts          []*ssm.PutParameterInput
	deletes       [][]string
	tagWrites     []types.Tag
	getBatchSizes []int
	failure       error
	failPutAt     int
	failTags      bool
	failDelete    bool
}

func (f *fakeSSM) GetParameter(ctx context.Context, in *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.failure != nil {
		return nil, f.failure
	}
	p, ok := f.params[aws.ToString(in.Name)]
	if !ok {
		return nil, &types.ParameterNotFound{}
	}
	return &ssm.GetParameterOutput{Parameter: &p}, nil
}
func (f *fakeSSM) GetParameters(ctx context.Context, in *ssm.GetParametersInput, _ ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.failure != nil {
		return nil, f.failure
	}
	f.getBatchSizes = append(f.getBatchSizes, len(in.Names))
	out := &ssm.GetParametersOutput{}
	for _, n := range in.Names {
		if p, ok := f.params[n]; ok {
			out.Parameters = append(out.Parameters, p)
		} else {
			out.InvalidParameters = append(out.InvalidParameters, n)
		}
	}
	return out, nil
}
func (f *fakeSSM) GetParametersByPath(ctx context.Context, in *ssm.GetParametersByPathInput, _ ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.failure != nil {
		return nil, f.failure
	}
	if len(f.pages) > 0 {
		index := 0
		if in.NextToken != nil {
			index = 1
		}
		f.pageCalls++
		return f.pages[index], nil
	}
	out := &ssm.GetParametersByPathOutput{}
	for n, p := range f.params {
		if strings.HasPrefix(n, strings.TrimSuffix(aws.ToString(in.Path), "/")+"/") {
			out.Parameters = append(out.Parameters, p)
		}
	}
	return out, nil
}
func (f *fakeSSM) GetParameterHistory(_ context.Context, in *ssm.GetParameterHistoryInput, _ ...func(*ssm.Options)) (*ssm.GetParameterHistoryOutput, error) {
	if f.history != nil {
		return &ssm.GetParameterHistoryOutput{Parameters: f.history}, nil
	}
	p := f.params[aws.ToString(in.Name)]
	return &ssm.GetParameterHistoryOutput{Parameters: []types.ParameterHistory{{Name: p.Name, Value: p.Value, Version: p.Version, Type: p.Type}}}, nil
}
func (f *fakeSSM) PutParameter(_ context.Context, in *ssm.PutParameterInput, _ ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	f.puts = append(f.puts, in)
	if len(f.puts) == f.failPutAt {
		return nil, errors.New("write failed")
	}
	f.params[aws.ToString(in.Name)] = types.Parameter{Name: in.Name, Value: in.Value, Type: in.Type, Version: 1}
	return &ssm.PutParameterOutput{Version: 1}, nil
}
func (f *fakeSSM) DeleteParameters(_ context.Context, in *ssm.DeleteParametersInput, _ ...func(*ssm.Options)) (*ssm.DeleteParametersOutput, error) {
	f.deletes = append(f.deletes, in.Names)
	if f.failDelete {
		return nil, errors.New("delete failed")
	}
	for _, n := range in.Names {
		delete(f.params, n)
	}
	return &ssm.DeleteParametersOutput{DeletedParameters: in.Names}, nil
}
func (f *fakeSSM) ListTagsForResource(context.Context, *ssm.ListTagsForResourceInput, ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error) {
	return &ssm.ListTagsForResourceOutput{TagList: f.tags}, nil
}
func (f *fakeSSM) AddTagsToResource(_ context.Context, in *ssm.AddTagsToResourceInput, _ ...func(*ssm.Options)) (*ssm.AddTagsToResourceOutput, error) {
	if f.failTags {
		return nil, errors.New("tags denied")
	}
	f.tagWrites = append(f.tagWrites, in.Tags...)
	return &ssm.AddTagsToResourceOutput{}, nil
}
func fixture(names ...string) (*ParameterStore, *fakeSSM) {
	f := &fakeSSM{params: map[string]types.Parameter{}}
	for _, n := range names {
		f.params[n] = types.Parameter{Name: aws.String(n), Value: aws.String("value"), Type: types.ParameterTypeString, Version: 1}
	}
	return &ParameterStore{Cwd: "/", Region: "r", Clients: map[string]API{"r": f}}, f
}
func pp(name string) ParameterPath { return ParameterPath{Name: name, Region: "r"} }

func TestListPaginationAndErrors(t *testing.T) {
	p, f := fixture()
	f.pages = []*ssm.GetParametersByPathOutput{{NextToken: aws.String("next")}, {Parameters: []types.Parameter{{Name: aws.String("/a/b/c")}, {Name: aws.String("/a/b/d")}}}}
	got, err := p.List(context.Background(), pp("/a"), false)
	if err != nil || !reflect.DeepEqual(got, []string{"b/"}) || f.pageCalls != 2 {
		t.Fatalf("%v %v calls=%d", got, err, f.pageCalls)
	}
	f.failure = errors.New("access denied")
	if _, err := p.List(context.Background(), pp("/a"), true); !errors.Is(err, f.failure) {
		t.Fatalf("lost API error: %v", err)
	}
}
func TestCancellation(t *testing.T) {
	p, _ := fixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.List(ctx, pp("/"), true); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}
func TestGetBatchingAndMissing(t *testing.T) {
	var names []string
	for i := 0; i < 23; i++ {
		names = append(names, fmt.Sprintf("/p%02d", i))
	}
	p, f := fixture(names...)
	got, err := p.Get(context.Background(), names, "r")
	if err != nil || len(got) != 23 || !reflect.DeepEqual(f.getBatchSizes, []int{10, 10, 3}) {
		t.Fatalf("%d %v %v", len(got), err, f.getBatchSizes)
	}
	if _, err := p.Get(context.Background(), []string{"/missing"}, "r"); err == nil {
		t.Fatal("missing parameter succeeded")
	}
}
func TestPaths(t *testing.T) {
	p, f := fixture("/a/b/c")
	p.Cwd = "/a/b"
	if got := p.Resolve("../d"); got != "/a/d" {
		t.Fatal(got)
	}
	if err := p.SetCwd(context.Background(), pp("/a")); err != nil || p.Cwd != "/a" {
		t.Fatalf("%s %v", p.Cwd, err)
	}
	f.failure = errors.New("permission denied")
	if err := p.SetCwd(context.Background(), pp("/x")); !errors.Is(err, f.failure) {
		t.Fatalf("%v", err)
	}
}
func TestCopyLatestMetadata(t *testing.T) {
	p, f := fixture("/source")
	f.history = []types.ParameterHistory{{Name: aws.String("/source"), Version: 9, Value: aws.String("latest"), Type: types.ParameterTypeSecureString, KeyId: aws.String("old-key"), Tier: types.ParameterTierAdvanced, DataType: aws.String("text"), Description: aws.String("description"), Policies: []types.ParameterInlinePolicy{{PolicyText: aws.String(`{"Type":"Expiration","Version":"1.0","Attributes":{"Timestamp":"2030-01-01T00:00:00Z"}}`)}}}, {Version: 1, Value: aws.String("old")}}
	f.tags = []types.Tag{{Key: aws.String("owner"), Value: aws.String("test")}}
	p.Key = "new-key"
	if err := p.Copy(context.Background(), pp("/source"), pp("/dest"), false); err != nil {
		t.Fatal(err)
	}
	in := f.puts[0]
	if aws.ToString(in.Value) != "latest" || aws.ToString(in.KeyId) != "new-key" || in.Tier != types.ParameterTierAdvanced || aws.ToString(in.DataType) != "text" || !strings.Contains(aws.ToString(in.Policies), "Expiration") || len(f.tagWrites) != 1 {
		t.Fatalf("metadata not preserved: %+v", in)
	}
}
func TestCopyTreeAndMove(t *testing.T) {
	p, f := fixture("/src/a", "/src/sub/b")
	if err := p.Move(context.Background(), pp("/src"), pp("/dst")); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"/dst/a", "/dst/sub/b"} {
		if _, ok := f.params[n]; !ok {
			t.Fatalf("missing %s", n)
		}
	}
	if _, ok := f.params["/src/a"]; ok {
		t.Fatal("source retained")
	}
}
func TestMoveNeverDeletesAfterCopyFailure(t *testing.T) {
	for _, tags := range []bool{false, true} {
		t.Run(fmt.Sprint(tags), func(t *testing.T) {
			p, f := fixture("/src/a", "/src/b")
			if tags {
				f.failTags = true
				f.tags = []types.Tag{{Key: aws.String("x"), Value: aws.String("y")}}
			} else {
				f.failPutAt = 2
			}
			err := p.Move(context.Background(), pp("/src"), pp("/dst"))
			if err == nil || !strings.Contains(err.Error(), "sources retained") || len(f.deletes) != 0 {
				t.Fatalf("unsafe move: %v %v", err, f.deletes)
			}
		})
	}
}
func TestMoveCleanupFailure(t *testing.T) {
	p, f := fixture("/src")
	f.failDelete = true
	err := p.Move(context.Background(), pp("/src"), pp("/dst"))
	if err == nil || !strings.Contains(err.Error(), "cleanup incomplete") {
		t.Fatal(err)
	}
}
func TestCopyOverlap(t *testing.T) {
	for _, dst := range []string{"/src", "/src/sub"} {
		p, f := fixture("/src/a")
		if err := p.Move(context.Background(), pp("/src"), pp(dst)); err == nil {
			t.Fatal("overlap accepted")
		}
		if len(f.puts)+len(f.deletes) != 0 {
			t.Fatal("overlap mutated data")
		}
	}
	p, f := fixture("/a/item")
	if err := p.Move(context.Background(), pp("/a/item"), pp("/a")); err == nil {
		t.Fatal("resolved self-move accepted")
	}
	if len(f.deletes) != 0 {
		t.Fatal("deleted source")
	}
}
func TestDryRun(t *testing.T) {
	p, f := fixture("/src/a", "/src/b")
	p.DryRun = true
	if err := p.Move(context.Background(), pp("/src"), pp("/dst")); err != nil {
		t.Fatal(err)
	}
	if len(f.puts)+len(f.deletes)+len(f.tagWrites) != 0 || len(p.Actions) != 4 {
		t.Fatalf("dry run: writes=%v deletes=%v actions=%v", f.puts, f.deletes, p.Actions)
	}
}
func TestCrossRegionKey(t *testing.T) {
	p, f := fixture("/src")
	dst := &fakeSSM{params: map[string]types.Parameter{}}
	p.Clients["other"] = dst
	f.history = []types.ParameterHistory{{Version: 1, Type: types.ParameterTypeSecureString, Value: aws.String("secret"), KeyId: aws.String("source-key")}}
	target := ParameterPath{Name: "/dst", Region: "other"}
	if err := p.Copy(context.Background(), pp("/src"), target, false); err == nil {
		t.Fatal("missing destination key accepted")
	}
	p.Key = "destination-key"
	if err := p.Copy(context.Background(), pp("/src"), target, false); err != nil {
		t.Fatal(err)
	}
	if aws.ToString(dst.puts[0].KeyId) != "destination-key" {
		t.Fatal("wrong key")
	}
}
func TestDeletePreflightAndBatching(t *testing.T) {
	var names []string
	for i := 0; i < 23; i++ {
		names = append(names, fmt.Sprintf("/src/%d", i))
	}
	p, f := fixture(names...)
	if err := p.Remove(context.Background(), []ParameterPath{pp("/src"), pp("/missing")}, true); err == nil {
		t.Fatal("missing target accepted")
	}
	if len(f.deletes) != 0 {
		t.Fatal("deleted before preflight finished")
	}
	if err := p.Remove(context.Background(), []ParameterPath{pp("/src")}, true); err != nil {
		t.Fatal(err)
	}
	if len(f.deletes) != 3 || len(f.deletes[0]) != 10 || len(f.deletes[2]) != 3 {
		t.Fatalf("%v", f.deletes)
	}
}
func TestPermissionErrorsPropagate(t *testing.T) {
	p, f := fixture()
	f.failure = errors.New("access denied")
	if err := p.Remove(context.Background(), []ParameterPath{pp("/src")}, true); !errors.Is(err, f.failure) {
		t.Fatalf("%v", err)
	}
}
