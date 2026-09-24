// Package diff compares two agentpack.yaml manifests field by field.
package diff

import (
	"fmt"
	"sort"

	"agentpack/internal/manifest"
)

// Change is one field-level difference.
type Change struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// Result is the outcome of comparing two manifests.
type Result struct {
	Name    string   `json:"name"`
	FromVer string   `json:"from_version"`
	ToVer   string   `json:"to_version"`
	Changes []Change `json:"changes"`
	Equal   bool     `json:"equal"`
}

func str(v any) string { return fmt.Sprintf("%v", v) }

// Compare returns field-level changes between a and b.
func Compare(a, b *manifest.Manifest) Result {
	r := Result{Name: a.Name, FromVer: a.Version, ToVer: b.Version, Changes: []Change{}}
	add := func(field, from, to string) {
		if from != to {
			r.Changes = append(r.Changes, Change{Field: field, From: from, To: to})
		}
	}
	add("name", a.Name, b.Name)
	add("version", a.Version, b.Version)
	add("description", a.Description, b.Description)
	add("harness", a.Harness, b.Harness)
	add("model", a.Model, b.Model)
	add("prompt", a.Prompt, b.Prompt)
	add("tools", str(toolKeys(a)), str(toolKeys(b)))
	add("skills", str(a.Skills), str(b.Skills))
	add("permissions.filesystem", str(a.Permissions.Filesystem), str(b.Permissions.Filesystem))
	add("permissions.network", str(a.Permissions.Network), str(b.Permissions.Network))
	add("permissions.exec", str(a.Permissions.Exec), str(b.Permissions.Exec))
	add("memory", str(a.Memory), str(b.Memory))
	sort.Slice(r.Changes, func(i, j int) bool { return r.Changes[i].Field < r.Changes[j].Field })
	r.Equal = len(r.Changes) == 0
	return r
}

func toolKeys(m *manifest.Manifest) []string {
	out := make([]string, 0, len(m.Tools))
	for _, t := range m.Tools {
		out = append(out, t.MCP)
	}
	return out
}
