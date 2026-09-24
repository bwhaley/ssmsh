package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/bwhaley/ssmsh/config"
	"github.com/bwhaley/ssmsh/parameterstore"
)

type captureAPI struct {
	parameterstore.API
	input *ssm.PutParameterInput
}

func (f *captureAPI) GetParameter(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return nil, &types.ParameterNotFound{}
}

func (f *captureAPI) GetParametersByPath(context.Context, *ssm.GetParametersByPathInput, ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	return &ssm.GetParametersByPathOutput{}, nil
}

func (f *captureAPI) PutParameter(_ context.Context, in *ssm.PutParameterInput, _ ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	f.input = in
	return &ssm.PutParameterOutput{Version: 2}, nil
}
func setup(t *testing.T) (*bytes.Buffer, *captureAPI) {
	t.Helper()
	out := &bytes.Buffer{}
	f := &captureAPI{}
	store := &parameterstore.ParameterStore{Cwd: "/", Region: "r", Type: "SecureString", Clients: map[string]parameterstore.API{"r": f}}
	Init(out, nil, store, &config.Config{})
	Interactive = false
	BatchInput = false
	SecretInput = strings.NewReader("")
	return out, f
}
func TestValidation(t *testing.T) {
	setup(t)
	for _, args := range [][]string{{"mv"}, {"cp", "a"}, {"get"}, {"history"}, {"rm"}, {"region", "a", "b"}, {"profile", "a", "b"}, {"put", "name=/a", "value=x", "typo=true"}, {"put", "name=/a", "value=x", "value-file=x"}, {"policy", "p", "Expiration(Timestamp)"}, {"cd", "region:/a:bad"}} {
		if err := Execute(context.Background(), args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}
func TestPutPreservesFileAndStdin(t *testing.T) {
	_, f := setup(t)
	secret := "  first\n\nlast\n"
	file := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(file, []byte(secret), 0600); err != nil {
		t.Fatal(err)
	}
	for _, option := range []string{"value-file=" + file, "value-stdin=true"} {
		SecretInput = strings.NewReader(secret)
		if err := Execute(context.Background(), []string{"put", "name=a", option, "tier=advanced"}); err != nil {
			t.Fatal(err)
		}
		if aws.ToString(f.input.Value) != secret || aws.ToString(f.input.Name) != "/a" || f.input.Tier != types.ParameterTierAdvanced {
			t.Fatalf("bad input: %+v", f.input)
		}
	}
	BatchInput = true
	if err := Execute(context.Background(), []string{"put", "name=a", "value-stdin=true"}); err == nil {
		t.Fatal("shared stdin accepted")
	}
}
func TestTier(t *testing.T) {
	for _, v := range []string{"standard", "Advanced", "INTELLIGENT-TIERING"} {
		if _, err := parseTier(v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := parseTier("bad"); err == nil {
		t.Fatal("bad tier accepted")
	}
}
func TestJSONContractAndValue(t *testing.T) {
	out, _ := setup(t)
	cfg.Default.Output = "json"
	params := []types.Parameter{{Name: aws.String("/a"), Value: aws.String("v"), Type: types.ParameterTypeString, Version: 3}}
	if err := printParameters(params); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got[0]["Name"] != "/a" || got[0]["Version"] != float64(3) || len(got[0]) != 9 {
		t.Fatalf("%v", got)
	}
	out.Reset()
	cfg.Default.Output = "value"
	if err := printParameters(params); err != nil {
		t.Fatal(err)
	}
	if out.String() != "v\n" {
		t.Fatalf("%q", out.String())
	}
}
func TestDryPut(t *testing.T) {
	out, f := setup(t)
	ps.DryRun = true
	if err := Execute(context.Background(), []string{"put", "name=a", "value=secret"}); err != nil {
		t.Fatal(err)
	}
	if f.input != nil || strings.Contains(out.String(), "secret") || !strings.Contains(out.String(), "put") {
		t.Fatalf("%s", out)
	}
}

func TestSingleWordCommandAndExit(t *testing.T) {
	setup(t)
	if err := Execute(context.Background(), []string{"ls"}); err != nil {
		t.Fatal(err)
	}
	if err := Execute(context.Background(), []string{"help", "ls"}); err != nil {
		t.Fatal(err)
	}
	if err := Execute(context.Background(), []string{"exit"}); !errors.Is(err, ErrExit) {
		t.Fatalf("exit returned %v", err)
	}
}
