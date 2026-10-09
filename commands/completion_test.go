package commands

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type completionAPI struct {
	*captureAPI
	names    []string
	calls    int
	prefixes []string
	err      error
}

func (f *completionAPI) DescribeParameters(ctx context.Context, in *ssm.DescribeParametersInput, _ ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
	f.calls++
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing lookup deadline")
	}
	f.prefixes = append(f.prefixes, in.ParameterFilters[0].Values[0])
	if f.err != nil {
		return nil, f.err
	}
	out := &ssm.DescribeParametersOutput{}
	for _, name := range f.names {
		if strings.HasPrefix(name, in.ParameterFilters[0].Values[0]) {
			out.Parameters = append(out.Parameters, types.ParameterMetadata{Name: aws.String(name)})
		}
	}
	return out, nil
}
func setupCompletion(t *testing.T) *completionAPI {
	_, base := setup(t)
	f := &completionAPI{captureAPI: base, names: []string{"/nonprod/password", "/nonprod/passthrough/key", "/nonprod/passthrough/other", "/nonprod/apple/key", "/nonprod/app/key", "/other/key"}}
	ps.Cwd = "/nonprod"
	ps.Clients["r"], ps.Clients["west"] = f, f
	return f
}
func TestCompletionCandidates(t *testing.T) {
	for _, tt := range []struct {
		line   string
		want   []string
		prefix string
	}{
		{"his", []string{"history"}, ""},
		{"help hi", []string{"history"}, ""},
		{"ls /nonprod/pas", []string{"/nonprod/passthrough/", "/nonprod/password"}, "/nonprod/pas"},
		{"get pas", []string{"passthrough/", "password"}, "/nonprod/pas"},
		{"cd pas", []string{"passthrough/"}, "/nonprod/pas"},
		{"ls app/", []string{"app/key"}, "/nonprod/app/"},
		{"ls ./pas", []string{"./passthrough/", "./password"}, "/nonprod/pas"},
		{"get ../other/", []string{"../other/key"}, "/other/"},
		{"ls west:pas", []string{"west:passthrough/", "west:password"}, "/nonprod/pas"},
		{"ls west:", []string{"west:app/", "west:apple/", "west:passthrough/", "west:password"}, "/nonprod/"},
		{"cp -r pas", []string{"passthrough/", "password"}, "/nonprod/pas"},
		{"cp key=alias/key source pas", []string{"passthrough/", "password"}, "/nonprod/pas"},
		{"put name=pas", []string{"name=passthrough/", "name=password"}, "/nonprod/pas"},
		{"put name=west:pas", nil, ""},
		{"put value=pas", nil, ""}, {"ls -", nil, ""}, {"profile pas", nil, ""},
		{"cd foo pas", nil, ""}, {"history foo pas", nil, ""}, {"cp a b pas", nil, ""},
		{"get 'pas", nil, ""}, {"get \"some path", nil, ""}, {"get :pas", nil, ""},
	} {
		t.Run(tt.line, func(t *testing.T) {
			f := setupCompletion(t)
			got, err := completionCandidates([]rune(tt.line), len([]rune(tt.line)))
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tt.want)
			}
			if tt.prefix == "" {
				if f.calls != 0 {
					t.Fatal("unexpected remote call")
				}
			} else if !reflect.DeepEqual(f.prefixes, []string{tt.prefix}) {
				t.Fatalf("prefixes %v", f.prefixes)
			}
		})
	}
}
func TestCompletionCache(t *testing.T) {
	f := setupCompletion(t)
	complete := func(line string) {
		t.Helper()
		if _, err := completionCandidates([]rune(line), len([]rune(line))); err != nil {
			t.Fatal(err)
		}
	}
	complete("ls pas")
	complete("ls pas")
	if f.calls != 1 {
		t.Fatal("cache missed")
	}
	complete("ls west:pas")
	ps.Profile = "other"
	complete("ls pas")
	ps.Cwd = "/other"
	complete("ls pas")
	if f.calls != 4 {
		t.Fatalf("cache crossed context: %d", f.calls)
	}
	for key, entry := range completionCache {
		entry.expires = time.Time{}
		completionCache[key] = entry
	}
	complete("ls pas")
	if f.calls != 5 {
		t.Fatal("expired cache used")
	}
}
func TestCompletionInvalidationAndErrors(t *testing.T) {
	for _, cmd := range []string{"put", "cp", "mv", "rm", "profile", "region"} {
		t.Run(cmd, func(t *testing.T) {
			f := setupCompletion(t)
			_, err := completionCandidates([]rune("ls pas"), 6)
			if err != nil {
				t.Fatal(err)
			}
			// Model a command that changes state and then fails partway through.
			handlers[cmd] = command{handler: func(*Context) error { return errors.New("partial failure") }}
			if Execute(context.Background(), []string{cmd}) == nil {
				t.Fatal("expected command error")
			}
			_, err = completionCandidates([]rune("ls pas"), 6)
			if err != nil {
				t.Fatal(err)
			}
			if f.calls != 2 {
				t.Fatal("cache survived mutation")
			}
		})
	}
	f := setupCompletion(t)
	f.err = errors.New("access denied")
	for range 2 {
		got, err := completionCandidates([]rune("ls pas"), 6)
		if err == nil || got != nil {
			t.Fatal("lookup error hidden")
		}
	}
	if f.calls != 1 {
		t.Fatal("repeated error was not briefly cached")
	}
}
func TestCompletionCursor(t *testing.T) {
	f := setupCompletion(t)
	for _, cursor := range []int{-1, 5, 100} {
		got, err := completionCandidates([]rune("get password"), cursor)
		if err != nil || len(got) != 0 {
			t.Fatalf("unsafe cursor %d: %v, %v", cursor, got, err)
		}
	}
	if f.calls != 0 {
		t.Fatal("unexpected lookup")
	}
}

func TestPutCompletionRegion(t *testing.T) {
	setupCompletion(t)
	remote := &completionAPI{names: []string{"/nonprod/remote"}}
	ps.Clients["west"] = remote
	for _, line := range []string{"put region=west name=rem", "put name=rem region=west"} {
		cursor := strings.Index(line, "name=rem") + len("name=rem")
		got, err := completionCandidates([]rune(line), cursor)
		if err != nil || !reflect.DeepEqual(got, []string{"name=remote"}) {
			t.Fatalf("got %v, %v", got, err)
		}
	}
	if remote.calls != 1 {
		t.Fatalf("remote calls = %d", remote.calls)
	}
}
