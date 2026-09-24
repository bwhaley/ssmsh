package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestBatchStopsAndRedacts(t *testing.T) {
	calls := 0
	err := processData(strings.NewReader(" # comment\nput name=a value=secret\nget a\n"), func(args []string) error { calls++; return errors.New("denied") })
	if calls != 1 || err == nil || !strings.Contains(err.Error(), "line 2") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("%d %v", calls, err)
	}
	err = processData(strings.NewReader("put value='secret"), func([]string) error { return nil })
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
}
func TestBatchQuotes(t *testing.T) {
	var got []string
	err := processData(strings.NewReader("put name=/a value=\"two words\"\n"), func(args []string) error { got = args; return nil })
	if err != nil || !reflect.DeepEqual(got, []string{"put", "name=/a", "value=two words"}) {
		t.Fatalf("%v %v", got, err)
	}
}
func TestRunLocalCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_PROFILE", "")
	for _, tc := range []struct {
		args   []string
		status int
		want   string
	}{{[]string{"-version"}, 0, "Version"}, {[]string{"profile"}, 0, "default"}, {[]string{"help"}, 0, "Commands"}, {[]string{"mv"}, 1, "source"}, {[]string{"-file", "/nonexistent/ssmsh-batch"}, 1, "ssmsh-batch"}, {[]string{"-config", "/nonexistent/ssmsh-config"}, 1, "ssmsh-config"}, {[]string{"-output", "bad", "profile"}, 2, "output"}} {
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		code := run(tc.args, strings.NewReader(""), out, errOut)
		if code != tc.status || !strings.Contains(out.String()+errOut.String(), tc.want) {
			t.Errorf("%v: code=%d out=%q err=%q", tc.args, code, out, errOut)
		}
	}
}
