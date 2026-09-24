package aws

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func environment(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(d, "credentials"))
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(d, "config"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	for _, key := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_ROLE_ARN"} {
		t.Setenv(key, "")
	}
	return d
}
func TestEnvironmentCredentials(t *testing.T) {
	environment(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_REGION", "us-east-1")
	cfg, err := Load(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil || creds.AccessKeyID != "test-key" || cfg.Region != "us-east-1" {
		t.Fatalf("region=%s err=%v", cfg.Region, err)
	}
}
func TestSharedProfile(t *testing.T) {
	d := environment(t)
	if err := os.WriteFile(filepath.Join(d, "config"), []byte("[profile dev]\nregion=us-west-2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "credentials"), []byte("[dev]\naws_access_key_id=profile-key\naws_secret_access_key=profile-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(context.Background(), "eu-west-1", "dev")
	if err != nil {
		t.Fatal(err)
	}
	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil || creds.AccessKeyID != "profile-key" || cfg.Region != "eu-west-1" {
		t.Fatalf("%s %v", cfg.Region, err)
	}
}
func TestRoleAndSSOConfig(t *testing.T) {
	d := environment(t)
	content := "[profile role]\nregion=us-east-1\nrole_arn=arn:aws:iam::123456789012:role/test\nsource_profile=base\n[profile base]\naws_access_key_id=test\naws_secret_access_key=test\n[profile sso]\nregion=us-west-2\nsso_session=test\nsso_account_id=123456789012\nsso_role_name=ReadOnly\n[sso-session test]\nsso_start_url=https://example.awsapps.com/start\nsso_region=us-east-1\nsso_registration_scopes=sso:account:access\n"
	if err := os.WriteFile(filepath.Join(d, "config"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"role", "sso"} {
		cfg, err := Load(context.Background(), "", profile)
		if err != nil || cfg.Credentials == nil || cfg.Region == "" {
			t.Fatalf("%s: %v", profile, err)
		}
	}
}
