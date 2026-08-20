package driver

import "testing"

func TestSandboxConfigDefaultsToAnIsolatedWorkspace(t *testing.T) {
	cfg := SandboxConfig{Image: "python:3.10-slim"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.WorkDir != "/workspace" {
		t.Fatalf("WorkDir = %q, want /workspace", cfg.WorkDir)
	}
	if cfg.EnableNetworking || cfg.NetworkPolicy.EnableInternet {
		t.Fatal("networking must remain disabled by default")
	}
}

func TestSandboxConfigRejectsNetworkAccessUntilPolicyExists(t *testing.T) {
	for _, cfg := range []SandboxConfig{
		{Image: "python:3.10-slim", EnableNetworking: true},
		{Image: "python:3.10-slim", NetworkPolicy: NetworkPolicy{EnableInternet: true}},
		{Image: "python:3.10-slim", AllowedHosts: []string{"example.com"}},
		{Image: "python:3.10-slim", NetworkPolicy: NetworkPolicy{AllowDomains: []string{"example.com"}}},
	} {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("Validate() succeeded for network-enabled config: %#v", cfg)
		}
	}
}

func TestSandboxConfigRejectsUnsafeWorkDirectories(t *testing.T) {
	for _, workDir := range []string{"relative", "/", "/tmp", "/output"} {
		cfg := SandboxConfig{Image: "python:3.10-slim", WorkDir: workDir}
		if err := cfg.Validate(); err == nil {
			t.Fatalf("Validate() succeeded for WorkDir %q", workDir)
		}
	}
}
