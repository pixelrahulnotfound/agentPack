// Package manifest parses and validates agentpack.yaml per SPEC.md.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]{1,63}$`)
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

type Tool struct {
	MCP   string   `yaml:"mcp"`
	Scope []string `yaml:"scope,omitempty"`
}

type Permissions struct {
	Filesystem []string `yaml:"filesystem,omitempty"`
	Network    []string `yaml:"network,omitempty"`
	Exec       bool     `yaml:"exec,omitempty"`
}

type Memory struct {
	Path       string `yaml:"path,omitempty"`
	MaxEntries int    `yaml:"max_entries,omitempty"`
}

type Manifest struct {
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version"`
	Description string      `yaml:"description,omitempty"`
	Harness     string      `yaml:"harness"`
	Model       string      `yaml:"model"`
	Prompt      string      `yaml:"prompt"`
	Tools       []Tool      `yaml:"tools,omitempty"`
	Skills      []string    `yaml:"skills,omitempty"`
	Permissions Permissions `yaml:"permissions,omitempty"`
	Memory      *Memory     `yaml:"memory,omitempty"`
}

// Load reads path, parses YAML (rejecting unknown top-level fields), and validates.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(false) // decode into struct first (forgiving)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	// Reject unknown top-level fields explicitly.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	allowed := map[string]bool{
		"name": true, "version": true, "description": true, "harness": true,
		"model": true, "prompt": true, "tools": true, "skills": true,
		"permissions": true, "memory": true,
	}
	for k := range raw {
		if !allowed[k] {
			return nil, fmt.Errorf("unknown field %q", k)
		}
	}
	base := filepath.Dir(path)
	if err := m.Validate(base); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks SPEC.md rules. base is the manifest dir for relative-path checks.
func (m *Manifest) Validate(base string) error {
	if !nameRe.MatchString(m.Name) {
		return fmt.Errorf("invalid name %q: must match ^[a-z0-9][a-z0-9-_]{1,63}$", m.Name)
	}
	if !semverRe.MatchString(m.Version) {
		return fmt.Errorf("invalid version %q: must be MAJOR.MINOR.PATCH semver", m.Version)
	}
	if m.Harness != "claude-code" && m.Harness != "opencode" {
		return fmt.Errorf("invalid harness %q: supported harnesses are \"claude-code\", \"opencode\"", m.Harness)
	}
	if strings.TrimSpace(m.Model) == "" {
		return fmt.Errorf("model must be non-empty")
	}
	if strings.TrimSpace(m.Prompt) == "" {
		return fmt.Errorf("prompt must be non-empty")
	}
	promptPath := m.Prompt
	if !filepath.IsAbs(promptPath) {
		promptPath = filepath.Join(base, m.Prompt)
	}
	if st, err := os.Stat(promptPath); err != nil || st.IsDir() {
		return fmt.Errorf("prompt file not found: %s", m.Prompt)
	}
	for _, t := range m.Tools {
		if strings.TrimSpace(t.MCP) == "" {
			return fmt.Errorf("tools[].mcp must be non-empty")
		}
	}
	for _, s := range m.Skills {
		p := s
		if !filepath.IsAbs(p) {
			p = filepath.Join(base, s)
		}
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			return fmt.Errorf("skill dir not found: %s", s)
		}
		if _, err := os.Stat(filepath.Join(p, "SKILL.md")); err != nil {
			return fmt.Errorf("skill %s missing SKILL.md", s)
		}
	}
	for _, f := range m.Permissions.Filesystem {
		if filepath.IsAbs(f) || f == ".." || strings.HasPrefix(f, "../") || strings.Contains(f, "../") {
			return fmt.Errorf("permissions.filesystem entries must be relative paths without escapes: %q", f)
		}
	}
	for _, h := range m.Permissions.Network {
		if strings.Contains(h, "://") || strings.Contains(h, "/") || strings.TrimSpace(h) == "" {
			return fmt.Errorf("permissions.network entries must be bare hostnames: %q", h)
		}
	}
	if m.Memory != nil && m.Memory.MaxEntries < 0 {
		return fmt.Errorf("memory.max_entries must be > 0")
	}
	if m.Memory != nil && m.Memory.MaxEntries == 0 {
		m.Memory.MaxEntries = 200
	}
	return nil
}
