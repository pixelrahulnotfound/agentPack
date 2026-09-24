package diff

import (
	"testing"

	"agentpack/internal/manifest"
)

func TestCompare(t *testing.T) {
	a := &manifest.Manifest{Name: "x", Version: "0.1.0", Harness: "claude-code", Model: "m1", Prompt: "./prompt.md"}
	b := &manifest.Manifest{Name: "x", Version: "0.2.0", Harness: "claude-code", Model: "m2", Prompt: "./prompt.md"}
	r := Compare(a, b)
	if r.Equal {
		t.Fatal("expected changes")
	}
	fields := map[string]bool{}
	for _, c := range r.Changes {
		fields[c.Field] = true
	}
	if !fields["version"] || !fields["model"] {
		t.Fatalf("missing expected changes: %+v", r.Changes)
	}
	if got := Compare(a, a); !got.Equal {
		t.Fatal("self-compare should be equal")
	}
}
