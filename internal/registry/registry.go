// Package registry implements registry v1: a directory (or a git checkout
// of a GitHub repo) acting as an index, one YAML file per published agent
// pointing at a source repo + pinned commit.
package registry

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var shaRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Entry is one published agent pointer. Description, Harness and Tools are
// optional discovery metadata recorded by `push`; older entries without them
// remain valid.
type Entry struct {
	Name         string   `yaml:"name" json:"name"`
	Version      string   `yaml:"version" json:"version"`
	SourceRepo   string   `yaml:"source_repo" json:"source_repo"`
	Commit       string   `yaml:"commit" json:"commit"`
	ManifestPath string   `yaml:"manifest_path" json:"manifest_path"`
	UpdatedAt    string   `yaml:"updated_at" json:"updated_at"`
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	Harness      string   `yaml:"harness,omitempty" json:"harness,omitempty"`
	Tools        []string `yaml:"tools,omitempty" json:"tools,omitempty"`
}

func (e *Entry) Validate() error {
	if e.Name == "" || e.Version == "" || e.SourceRepo == "" || e.ManifestPath == "" {
		return fmt.Errorf("name, version, source_repo and manifest_path are required")
	}
	if !shaRe.MatchString(e.Commit) {
		return fmt.Errorf("commit must be a 40-char hex SHA, got %q", e.Commit)
	}
	return nil
}

// List returns all index entries sorted by name. A missing registry dir
// yields an empty list, not an error.
func List(registryDir string) ([]Entry, error) {
	files, err := filepath.Glob(filepath.Join(registryDir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, f := range files {
		if strings.HasSuffix(f, "README.yaml") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var e Entry
		if err := yaml.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f, err)
		}
		if err := e.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Match is one search hit with its relevance score.
type Match struct {
	Entry Entry    `json:"entry"`
	Score int      `json:"score"`
	Why   []string `json:"why"`
}

// Search ranks entries against a free-text query. Name hits score 3,
// description hits 2, harness/tool hits 1. Empty query returns everything
// (score 0). Results are sorted by score desc, then name.
func Search(registryDir, query string) ([]Match, error) {
	entries, err := List(registryDir)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		matches := make([]Match, 0, len(entries))
		for _, e := range entries {
			matches = append(matches, Match{Entry: e})
		}
		return matches, nil
	}
	terms := strings.Fields(q)
	var matches []Match
	for _, e := range entries {
		m := Match{Entry: e}
		name := strings.ToLower(e.Name)
		desc := strings.ToLower(e.Description)
		harness := strings.ToLower(e.Harness)
		tools := strings.ToLower(strings.Join(e.Tools, " "))
		for _, t := range terms {
			switch {
			case strings.Contains(name, t):
				m.Score += 3
				m.Why = append(m.Why, "name")
			case strings.Contains(desc, t):
				m.Score += 2
				m.Why = append(m.Why, "description")
			case strings.Contains(harness, t) || strings.Contains(tools, t):
				m.Score += 1
				m.Why = append(m.Why, "capability")
			}
		}
		if m.Score > 0 {
			matches = append(matches, m)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Entry.Name < matches[j].Entry.Name
	})
	return matches, nil
}
func IndexPath(registryDir, name string) string {
	return filepath.Join(registryDir, name+".yaml")
}

// Read loads one index entry.
func Read(registryDir, name string) (*Entry, error) {
	data, err := os.ReadFile(IndexPath(registryDir, name))
	if err != nil {
		return nil, fmt.Errorf("agent %q not found in registry %s: %w", name, registryDir, err)
	}
	var e Entry
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("parse registry entry: %w", err)
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return &e, nil
}

// Write publishes/updates one index entry.
func Write(registryDir string, e Entry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if e.UpdatedAt == "" {
		e.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(&e)
	if err != nil {
		return err
	}
	return os.WriteFile(IndexPath(registryDir, e.Name), data, 0o644)
}

// HeadCommit resolves the current HEAD SHA of sourceRepo (local path or URL).
// For local paths it runs `git rev-parse HEAD`; for URLs it uses `git ls-remote`.
func HeadCommit(sourceRepo string) (string, error) {
	var cmd *exec.Cmd
	if _, err := os.Stat(sourceRepo); err == nil {
		cmd = exec.Command("git", "-C", sourceRepo, "rev-parse", "HEAD")
	} else {
		cmd = exec.Command("git", "ls-remote", sourceRepo, "HEAD")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve HEAD of %s: %s: %w", sourceRepo, string(out), err)
	}
	// ls-remote prints "<sha>\tHEAD\n"; rev-parse prints "<sha>\n".
	sha := string(out)
	if len(sha) > 40 {
		sha = sha[:40]
	} else {
		// trim newline for rev-parse
		for len(sha) > 0 && (sha[len(sha)-1] == '\n' || sha[len(sha)-1] == ' ' || sha[len(sha)-1] == '\t') {
			sha = sha[:len(sha)-1]
		}
	}
	if !shaRe.MatchString(sha) {
		return "", fmt.Errorf("could not resolve 40-char SHA from %q", string(out))
	}
	return sha, nil
}
