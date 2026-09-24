package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMissingConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := ReadConfig(""); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("explicit missing config accepted")
	}
}
func TestConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(p, []byte("[default]\nregion=us-west-2\nprofile=dev\ndecrypt=true\noutput=json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := ReadConfig(p)
	if err != nil || c.Default.Region != "us-west-2" || !c.Default.Decrypt {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestConfigValidation(t *testing.T) {
	for _, contents := range []string{
		"region=us-east-1\n",
		"[other]\nregion=us-east-1\n",
		"[default]\ndecrypt=perhaps\n",
		"[default]\nunknown=value\n",
		"[default]\nnot-a-setting\n",
	} {
		path := filepath.Join(t.TempDir(), "config")
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadConfig(path); err == nil {
			t.Errorf("accepted invalid config %q", contents)
		}
	}
}
