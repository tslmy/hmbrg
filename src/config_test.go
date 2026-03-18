package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromFileSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	data := []byte(`
endpoint = "http://localhost:8581"
username = "user"
password = "pass"
otp = "123456"
show_all = true
dump_accessories = true
timeout_seconds = 12
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigFromFile(path)
	if err != nil {
		t.Fatalf("LoadConfigFromFile: %v", err)
	}
	if cfg.Endpoint != "http://localhost:8581" {
		t.Fatalf("endpoint mismatch: %q", cfg.Endpoint)
	}
	if cfg.Username != "user" || cfg.Password != "pass" || cfg.OTP != "123456" {
		t.Fatalf("credentials mismatch: %+v", cfg)
	}
	if !cfg.ShowAll || !cfg.DumpAccessories || cfg.TimeoutSeconds != 12 {
		t.Fatalf("flags mismatch: %+v", cfg)
	}
}

func TestLoadConfigFromFileMissingRequired(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "missing endpoint",
			body: `
username = "user"
password = "pass"
`,
		},
		{
			name: "missing username",
			body: `
endpoint = "http://localhost:8581"
password = "pass"
`,
		},
		{
			name: "missing password",
			body: `
endpoint = "http://localhost:8581"
username = "user"
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}
			if _, err := LoadConfigFromFile(path); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}
