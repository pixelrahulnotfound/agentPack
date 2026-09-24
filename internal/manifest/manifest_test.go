package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func writeAgent(t *testing.T, yml string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "prompt.md"), []byte("# p"), 0o644); err != nil {
		t.Fatal(err)
	}
	sk := filepath.Join(dir, "skills", "s")
	if err := os.MkdirAll(sk, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sk, "SKILL.md"), []byte("# s"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agentpack.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "agentpack.yaml")
}

func TestLoadValid(t *testing.T) {
	p := writeAgent(t, "name: demo-agent\nversion: 0.1.0\nharness: claude-code\nmodel: m\nprompt: ./prompt.md\nskills: [./skills/s]\nmemory:\n  path: ./memory/\n")
	m, err := Load(p)
	if err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if m.Name != "demo-agent" || m.Memory == nil || m.Memory.MaxEntries != 200 {
		t.Fatalf("unexpected manifest: %+v", m)
	}
}

func TestRejectUnknownField(t *testing.T) {
	p := writeAgent(t, "name: demo-agent\nversion: 0.1.0\nharness: claude-code\nmodel: m\nprompt: ./prompt.md\nzzz: 1\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestRejectBadNameAndVersion(t *testing.T) {
	p := writeAgent(t, "name: Bad_Name!\nversion: 0.1.0\nharness: claude-code\nmodel: m\nprompt: ./prompt.md\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected bad-name error")
	}
	p2 := writeAgent(t, "name: good-name\nversion: v1\nharness: claude-code\nmodel: m\nprompt: ./prompt.md\n")
	if _, err := Load(p2); err == nil {
		t.Fatal("expected bad-version error")
	}
}

func TestRejectMissingPrompt(t *testing.T) {
	dir := t.TempDir()
	mp := filepath.Join(dir, "agentpack.yaml")
	_ = os.WriteFile(mp, []byte("name: a1\nversion: 0.1.0\nharness: claude-code\nmodel: m\nprompt: ./nope.md\n"), 0o644)
	if _, err := Load(mp); err == nil {
		t.Fatal("expected missing-prompt error")
	}
}
