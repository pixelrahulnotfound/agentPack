// Command agentpack builds, runs, and shares versioned personal-agent manifests.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"agentpack/internal/diff"
	"agentpack/internal/manifest"
	"agentpack/internal/output"
	"agentpack/internal/registry"
	"agentpack/internal/sandbox"
)

const version = "0.1.0"

func usage() {
	fmt.Fprintf(os.Stderr, `agentpack %s — versioned personal-agent manifests

Usage:
  agentpack [--json] <command> [options]

Commands:
  init <dir> [--name NAME] [--harness HARNESS]
                               scaffold agentpack.yaml + prompt.md + AGENT.md + skills/
  run [--dir DIR] [--image IMG] [--dry-run] [-- CMD...]
                               validate manifest and launch harness in Docker sandbox
  pull NAME --registry DIR --out DIR
                               fetch agent from registry index at pinned commit
  push --dir DIR --registry DIR --source-repo URL
                               publish agent pointer to registry index
  diff <fileA> <fileB>          field-level manifest comparison
  search QUERY --registry DIR  rank agents in the registry by relevance
  serve --registry DIR --port PORT
                               run the live discovery HTTP API
  version                       print version

Global flags:
  --json    machine-readable output: {"ok":true,"data":...} on stdout,
            {"ok":false,"error":{"code","message","hint"}} on stderr

`, version)
}

func main() {
	// Global --json may appear before the subcommand.
	jsonMode := false
	args := os.Args[1:]
	filtered := []string{}
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
			continue
		}
		filtered = append(filtered, a)
	}
	output.JSONMode = jsonMode

	if len(filtered) == 0 || filtered[0] == "-h" || filtered[0] == "--help" || filtered[0] == "help" {
		usage()
		return
	}
	cmd, rest := filtered[0], filtered[1:]
	var err error
	switch cmd {
	case "version":
		output.Success(map[string]string{"version": version}, "agentpack "+version)
	case "init":
		err = cmdInit(rest)
	case "run":
		err = cmdRun(rest)
	case "pull":
		err = cmdPull(rest)
	case "push":
		err = cmdPush(rest)
	case "diff":
		err = cmdDiff(rest)
	case "search":
		err = cmdSearch(rest)
	case "serve":
		err = cmdServe(rest)
	default:
		output.Failure("unknown_command", "unknown command "+cmd, "run: agentpack --help")
		os.Exit(1)
	}
	if err != nil {
		// Subcommands already printed structured errors; ensure exit code.
		os.Exit(1)
	}
}

// splitMixed separates flag tokens (and their values) from positionals so
// flags work before or after positional args: `pull NAME --registry D` and
// `pull --registry D NAME` are equivalent. valueFlags names flags taking a value.
func splitMixed(args []string, valueFlags map[string]bool) (flagArgs, posArgs []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && a != "-" && a != "--" {
			name := strings.TrimLeft(a, "-")
			if eq := strings.Index(name, "="); eq >= 0 {
				name = name[:eq]
			}
			flagArgs = append(flagArgs, a)
			if eq := strings.Index(a, "="); eq < 0 && valueFlags[name] && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		if a == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		posArgs = append(posArgs, a)
	}
	return flagArgs, posArgs
}

// parseMixed parses flags regardless of position, returning positionals.
func parseMixed(fs *flag.FlagSet, args []string, valueFlags map[string]bool) []string {
	flagArgs, posArgs := splitMixed(args, valueFlags)
	_ = fs.Parse(flagArgs)
	return posArgs
}

// ---------- init ----------

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	name := fs.String("name", "my-agent", "agent name")
	harness := fs.String("harness", "claude-code", "target harness: claude-code or opencode")
	pos := parseMixed(fs, args, map[string]bool{"name": true, "harness": true})
	dir := "."
	if len(pos) > 0 {
		dir = pos[0]
	}
	if *harness != "claude-code" && *harness != "opencode" {
		output.Failure("bad_args", "invalid harness "+*harness, "supported harnesses: claude-code, opencode")
		return fmt.Errorf("args")
	}
	if err := os.MkdirAll(filepath.Join(dir, "skills", "starter"), 0o755); err != nil {
		output.Failure("init_failed", err.Error(), "")
		return err
	}
	mpath := filepath.Join(dir, "agentpack.yaml")
	if _, err := os.Stat(mpath); err == nil {
		output.Failure("init_failed", "agentpack.yaml already exists in "+dir, "run in an empty dir or remove it first")
		return fmt.Errorf("exists")
	}
	yman := fmt.Sprintf(`name: %s
version: 0.1.0
description: A personal research assistant
harness: %s
model: claude-sonnet-4-5
prompt: ./prompt.md
tools:
  - mcp: web-search
    scope: [search, fetch]
skills:
  - ./skills/starter
permissions:
  filesystem: [./work]
  network: [api.github.com]
  exec: false
memory:
  path: ./memory/
  max_entries: 200
`, *name, *harness)
	prompt := "# System prompt for " + *name + "\n\nYou are a helpful personal agent. Be concise, cite sources, and say when you don't know.\n"
	agentmd := fmt.Sprintf(`# %s — agent-readable card

- What: starter personal agent created by `+"`agentpack init`"+`.
- Harness: %s. Model: claude-sonnet-4-5.
- Tools: web-search (search, fetch). Skills: ./skills/starter.
- Permissions: filesystem [./work], network [api.github.com], exec false.
- Invoke: `+"`agentpack run --dir .`"+` from this directory.
- Input: a natural-language task on stdin or as CLI args. Output: concise markdown answer with sources.
- Manifest: ./agentpack.yaml is the source of truth; this file is derived from it.
`, *name, *harness)
	skill := "# Starter skill\n\nReplace this with reusable instructions. This dir must keep a SKILL.md (required by SPEC.md).\n"
	write := func(p, c string) error { return os.WriteFile(p, []byte(c), 0o644) }
	if err := write(mpath, yman); err != nil {
		output.Failure("init_failed", err.Error(), "")
		return err
	}
	_ = write(filepath.Join(dir, "prompt.md"), prompt)
	_ = write(filepath.Join(dir, "AGENT.md"), agentmd)
	_ = write(filepath.Join(dir, "skills", "starter", "SKILL.md"), skill)
	_ = os.MkdirAll(filepath.Join(dir, "work"), 0o755)
	output.Success(map[string]string{"dir": dir, "manifest": mpath},
		"initialized "+mpath)
	return nil
}

// ---------- run ----------

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	dir := fs.String("dir", ".", "agent directory containing agentpack.yaml")
	image := fs.String("image", "", "docker image for the harness (default: per-harness sandbox image)")
	dry := fs.Bool("dry-run", false, "validate + print docker command without running")
	pos := parseMixed(fs, args, map[string]bool{"dir": true, "image": true})
	extra := pos
	// "-- CMD" separator: flag package keeps "--" out; extra after is passthrough.
	mpath := filepath.Join(*dir, "agentpack.yaml")
	m, err := manifest.Load(mpath)
	if err != nil {
		output.Failure("invalid_manifest", err.Error(), "fix "+mpath+" per SPEC.md")
		return err
	}
	img := *image
	if img == "" {
		img = sandbox.DefaultImage(m.Harness)
	}
	if *dry {
		dargs := sandbox.RunArgs(*dir, img, extra)
		output.Success(map[string]any{"manifest": m.Name, "docker": append([]string{"docker"}, dargs...)},
			"dry-run ok: "+m.Name+" "+m.Version+" ("+m.Harness+"/"+m.Model+")\n$ docker "+strings.Join(dargs, " "))
		return nil
	}
	if err := sandbox.Run(*dir, img, extra); err != nil {
		output.Failure("sandbox_failed", err.Error(), "is docker running? try --dry-run to validate without docker")
		return err
	}
	output.Success(map[string]string{"agent": m.Name, "version": m.Version}, "ran "+m.Name)
	return nil
}

// ---------- diff ----------

func cmdDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	pos := parseMixed(fs, args, map[string]bool{})
	if len(pos) != 2 {
		output.Failure("bad_args", "diff needs exactly 2 manifest files", "usage: agentpack diff <fileA> <fileB>")
		return fmt.Errorf("args")
	}
	a, err := manifest.Load(pos[0])
	if err != nil {
		output.Failure("invalid_manifest", "fileA: "+err.Error(), "")
		return err
	}
	b, err := manifest.Load(pos[1])
	if err != nil {
		output.Failure("invalid_manifest", "fileB: "+err.Error(), "")
		return err
	}
	r := diff.Compare(a, b)
	if output.JSONMode {
		output.Success(r, "")
		return nil
	}
	if r.Equal {
		output.Success(r, "no changes ("+a.Name+" "+a.Version+")")
		return nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "diff %s: %s -> %s (%d changes)\n", r.Name, r.FromVer, r.ToVer, len(r.Changes))
	for _, c := range r.Changes {
		fmt.Fprintf(&sb, "  ~ %s: %s => %s\n", c.Field, c.From, c.To)
	}
	output.Success(r, sb.String())
	return nil
}

// ---------- push ----------

func cmdPush(args []string) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	dir := fs.String("dir", ".", "agent directory")
	reg := fs.String("registry", "registry-index", "registry index dir")
	source := fs.String("source-repo", "", "source repo URL or local path (required)")
	manifestPath := fs.String("manifest-path", "", "manifest path inside source repo (default: relative path of --dir)")
	_ = parseMixed(fs, args, map[string]bool{"dir": true, "registry": true, "source-repo": true, "manifest-path": true})
	if *source == "" {
		output.Failure("bad_args", "--source-repo is required", "example: agentpack push --dir . --registry ./registry-index --source-repo https://github.com/you/my-agent")
		return fmt.Errorf("args")
	}
	m, err := manifest.Load(filepath.Join(*dir, "agentpack.yaml"))
	if err != nil {
		output.Failure("invalid_manifest", err.Error(), "")
		return err
	}
	commit, err := registry.HeadCommit(*source)
	if err != nil {
		output.Failure("push_failed", err.Error(), "is the repo URL/path correct and git installed?")
		return err
	}
	mp := *manifestPath
	if mp == "" {
		abs, _ := filepath.Abs(*dir)
		_ = abs
		mp = "agentpack.yaml"
		// Keep it simple and predictable for v1: manifest at agent root.
	}
	tools := make([]string, 0, len(m.Tools))
	for _, t := range m.Tools {
		tools = append(tools, t.MCP)
	}
	e := registry.Entry{Name: m.Name, Version: m.Version, SourceRepo: *source, Commit: commit, ManifestPath: mp,
		Description: m.Description, Harness: m.Harness, Tools: tools}
	if err := registry.Write(*reg, e); err != nil {
		output.Failure("push_failed", err.Error(), "")
		return err
	}
	output.Success(e, fmt.Sprintf("pushed %s %s @ %s to %s", e.Name, e.Version, commit[:12], *reg))
	return nil
}

// ---------- pull ----------

func cmdPull(args []string) error {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	reg := fs.String("registry", "registry-index", "registry index dir")
	out := fs.String("out", "", "output dir (default: ./<name>)")
	pos := parseMixed(fs, args, map[string]bool{"registry": true, "out": true})
	if len(pos) != 1 {
		output.Failure("bad_args", "pull needs exactly one agent name", "usage: agentpack pull NAME --registry DIR --out DIR")
		return fmt.Errorf("args")
	}
	name := pos[0]
	e, err := registry.Read(*reg, name)
	if err != nil {
		output.Failure("not_found", err.Error(), "check the name and --registry dir")
		return err
	}
	dest := *out
	if dest == "" {
		dest = filepath.Join(".", name)
	}
	tmp, err := os.MkdirTemp("", "agentpack-pull-*")
	if err != nil {
		output.Failure("pull_failed", err.Error(), "")
		return err
	}
	defer os.RemoveAll(tmp)
	clone := exec.Command("git", "clone", "--quiet", e.SourceRepo, tmp+"/src")
	if out, err := clone.CombinedOutput(); err != nil {
		output.Failure("pull_failed", "git clone failed: "+string(out), "check source_repo URL and network")
		return err
	}
	checkout := exec.Command("git", "-C", tmp+"/src", "checkout", "--quiet", e.Commit)
	if out, err := checkout.CombinedOutput(); err != nil {
		output.Failure("pull_failed", "git checkout "+e.Commit[:12]+" failed: "+string(out), "the pinned commit may be missing")
		return err
	}
	// manifest_path is relative to repo root; the agent dir is its parent.
	srcManifest := filepath.Join(tmp, "src", e.ManifestPath)
	srcDir := filepath.Dir(srcManifest)
	if _, err := os.Stat(srcManifest); err != nil {
		output.Failure("pull_failed", "manifest_path not found at pinned commit: "+e.ManifestPath, "")
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		output.Failure("pull_failed", err.Error(), "")
		return err
	}
	cp := exec.Command("cp", "-a", srcDir+"/.", dest)
	if out, err := cp.CombinedOutput(); err != nil {
		output.Failure("pull_failed", "copy failed: "+string(out), "")
		return err
	}
	// Validate what we pulled.
	if _, err := manifest.Load(filepath.Join(dest, "agentpack.yaml")); err != nil {
		output.Failure("pull_failed", "pulled agent failed validation: "+err.Error(), "")
		return err
	}
	// Emit AGENT.md note: ensure one exists (older packs may lack it).
	agentMD := filepath.Join(dest, "AGENT.md")
	if _, err := os.Stat(agentMD); err != nil {
		_ = os.WriteFile(agentMD, []byte("# "+e.Name+"\n\nPulled from "+e.SourceRepo+" @ "+e.Commit+". See agentpack.yaml.\n"), 0o644)
	}
	data, _ := yaml.Marshal(e)
	_ = data
	output.Success(map[string]string{"name": e.Name, "version": e.Version, "commit": e.Commit, "out": dest},
		fmt.Sprintf("pulled %s %s @ %s -> %s", e.Name, e.Version, e.Commit[:12], dest))
	return nil
}

// ---------- search ----------

func cmdSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	reg := fs.String("registry", "registry-index", "registry index dir")
	limit := fs.Int("limit", 10, "max results")
	pos := parseMixed(fs, args, map[string]bool{"registry": true, "limit": true})
	query := strings.Join(pos, " ")
	matches, err := registry.Search(*reg, query)
	if err != nil {
		output.Failure("search_failed", err.Error(), "check --registry dir")
		return err
	}
	if *limit >= 0 && len(matches) > *limit {
		matches = matches[:*limit]
	}
	if output.JSONMode {
		output.Success(matches, "")
		return nil
	}
	if len(matches) == 0 {
		output.Success(matches, "no agents match "+strconv.Quote(query))
		return nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d result(s) for %s:\n", len(matches), strconv.Quote(query))
	for _, m := range matches {
		fmt.Fprintf(&sb, "  %s %s [%s] score=%d (%s)\n",
			m.Entry.Name, m.Entry.Version, m.Entry.Harness, m.Score, strings.Join(m.Why, ","))
		if m.Entry.Description != "" {
			fmt.Fprintf(&sb, "    %s\n", m.Entry.Description)
		}
	}
	output.Success(matches, sb.String())
	return nil
}
