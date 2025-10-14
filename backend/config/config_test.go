package config

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to write temp config files
func writeTempConfig(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestValidate_ValidConfig(t *testing.T) {
	valid := `ami_enabled: true
ami_host: 127.0.0.1
nodes: [43732]
gamification:
  enabled: true
  tally_interval_minutes: 30
`
	p := writeTempConfig(t, "valid.yaml", valid)
	if err := Validate(p); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidate_TabsInConfig(t *testing.T) {
	tabbed := "gamification:\n\tenabled: true\n\trested_bonus:\n\t\tenabled: true\n"
	p := writeTempConfig(t, "tabs.yaml", tabbed)
	if err := Validate(p); err == nil {
		t.Fatalf("expected validation to fail due to tabs, but it passed")
	}
}

func TestValidate_MissingFile(t *testing.T) {
	if err := Validate("/path/does/not/exist.yaml"); err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}

func TestValidate_MalformedNodes(t *testing.T) {
	// nodes is present but malformed (map instead of list)
	bad := "nodes: { node_id: 123 }\n"
	p := writeTempConfig(t, "badnodes.yaml", bad)
	if err := Validate(p); err == nil {
		t.Fatalf("expected error for malformed nodes section, but got nil")
	}
}
