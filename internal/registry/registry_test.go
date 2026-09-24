package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func seed(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(e Entry) {
		t.Helper()
		if err := Write(dir, e); err != nil {
			t.Fatal(err)
		}
	}
	write(Entry{Name: "research-assistant", Version: "0.1.0", SourceRepo: "https://example.com/r", Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ManifestPath: "agentpack.yaml", Description: "summarizes papers", Harness: "claude-code", Tools: []string{"web-search", "github"}})
	write(Entry{Name: "code-helper", Version: "0.2.0", SourceRepo: "https://example.com/c", Commit: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ManifestPath: "agentpack.yaml", Description: "writes code", Harness: "opencode", Tools: []string{"github"}})
	// Legacy entry without discovery metadata must still validate.
	legacy := "name: old-timer\nversion: 0.0.1\nsource_repo: https://example.com/o\ncommit: cccccccccccccccccccccccccccccccccccccccc\nmanifest_path: agentpack.yaml\n"
	if err := os.WriteFile(filepath.Join(dir, "old-timer.yaml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestListSorted(t *testing.T) {
	entries, err := List(seed(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[0].Name != "code-helper" || entries[2].Name != "research-assistant" {
		t.Fatalf("unexpected list: %+v", entries)
	}
}

func TestSearchRanking(t *testing.T) {
	dir := seed(t)
	m, err := Search(dir, "summarizes papers")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 || m[0].Entry.Name != "research-assistant" {
		t.Fatalf("expected research-assistant, got %+v", m)
	}
	m, err = Search(dir, "github")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 github hits, got %+v", m)
	}
	m, err = Search(dir, "opencode")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 || m[0].Entry.Name != "code-helper" {
		t.Fatalf("expected code-helper via harness, got %+v", m)
	}
	m, err = Search(dir, "no-such-thing-zzz")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("expected no hits, got %+v", m)
	}
	all, err := Search(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("empty query should return all, got %+v", all)
	}
}
