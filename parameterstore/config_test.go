package parameterstore

import (
	"context"
	"github.com/bwhaley/ssmsh/config"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPrecedence(t *testing.T) {
	t.Setenv("AWS_PROFILE", "env")
	t.Setenv("AWS_REGION", "env-region")
	var cfg config.Config
	cfg.Default.Profile = "file"
	cfg.Default.Region = "file-region"
	var p ParameterStore
	p.SetDefaults(cfg)
	if p.Profile != "env" || p.Region != "env-region" {
		t.Fatal(p)
	}
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_REGION", "")
	p.SetDefaults(cfg)
	if p.Profile != "file" || p.Region != "file-region" {
		t.Fatal(p)
	}
	p.SetDefaults(config.Config{})
	if p.Profile != "" {
		t.Fatal("default profile overrides environment credentials")
	}
}
func TestSwitchClearsAllClients(t *testing.T) {
	d := t.TempDir()
	file := filepath.Join(d, "config")
	if err := os.WriteFile(file, []byte("[profile next]\nregion=us-west-2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", file)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(d, "missing"))
	p, _ := fixture()
	p.Clients["other"] = &fakeSSM{}
	if err := p.Switch(context.Background(), "us-east-1", "next"); err != nil {
		t.Fatal(err)
	}
	if len(p.Clients) != 1 || p.Profile != "next" || p.Clients["other"] != nil {
		t.Fatal("stale clients retained")
	}
}
