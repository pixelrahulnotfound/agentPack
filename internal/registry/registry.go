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
	"time"

	"gopkg.in/yaml.v3"
)

var shaRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Entry is one published agent pointer.
type Entry struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	SourceRepo   string `yaml:"source_repo"`
	Commit       string `yaml:"commit"`
	ManifestPath string `yaml:"manifest_path"`
	UpdatedAt    string `yaml:"updated_at"`
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

// IndexPath returns the file for name inside the registry dir.
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
