package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soarinferret/dns-tui/internal/config"
)

func TestLoad(t *testing.T) {
	content := `profiles:
  - name: "Test CF"
    provider: cloudflare
    credentials:
      api_token: "tok123"
  - name: "Test GD"
    provider: godaddy
    credentials:
      api_key: "key"
      api_secret: "secret"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}

	p := cfg.Profiles[0]
	if p.Name != "Test CF" {
		t.Errorf("expected name 'Test CF', got %q", p.Name)
	}
	if p.Provider != "cloudflare" {
		t.Errorf("expected provider 'cloudflare', got %q", p.Provider)
	}
	if p.Credentials["api_token"] != "tok123" {
		t.Errorf("expected api_token 'tok123', got %q", p.Credentials["api_token"])
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":\n  :\n    - [invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_EmptyProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte("profiles: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(cfg.Profiles))
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.yaml")

	cfg := &config.Config{
		Profiles: []config.Profile{
			{
				Name:     "My CF",
				Provider: "cloudflare",
				Credentials: map[string]string{
					"api_token": "tok",
				},
			},
		},
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].Name != "My CF" {
		t.Errorf("expected 'My CF', got %q", loaded.Profiles[0].Name)
	}
	if loaded.Profiles[0].Credentials["api_token"] != "tok" {
		t.Errorf("expected api_token 'tok', got %q", loaded.Profiles[0].Credentials["api_token"])
	}

	// Verify file permissions are restrictive (0600)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected file mode 0600, got %o", perm)
	}
}

func TestSave_Append(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &config.Config{
		Profiles: []config.Profile{
			{Name: "First", Provider: "cloudflare", Credentials: map[string]string{"api_token": "a"}},
		},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg.Profiles = append(cfg.Profiles, config.Profile{
		Name: "Second", Provider: "godaddy", Credentials: map[string]string{"api_key": "k", "api_secret": "s"},
	})
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(loaded.Profiles))
	}
}

func TestDefaultPath(t *testing.T) {
	p := config.DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}
	if filepath.Base(p) != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", filepath.Base(p))
	}
}
