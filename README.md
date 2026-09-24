# AgentPack

**Docker Hub for personal AI agents.** AgentPack packages an agent's entire setup — system prompt, model, MCP tool references, skills, permission scopes, memory config — into one versioned, runnable manifest (`agentpack.yaml`), plus a CLI to build, run, and share it.

The problem it solves: today every AI agent is snowflake setup. Prompts live in gists, MCP configs in dotfiles, skills scattered across folders. Nothing is versioned, nothing is shareable, and nothing is runnable by someone else — human or agent. AgentPack is the packaging layer: versioned like code, sandboxed like containers, searchable like packages.

## Features

- **Versioned manifests** — `agentpack.yaml` with semver, validated against a frozen spec (`SPEC.md`). Unknown fields rejected, skill/prompt paths verified on load.
- **One-command runs** — `agentpack run` validates the manifest and launches the harness inside a Docker sandbox. `--dry-run` validates without Docker.
- **Two harnesses** — `claude-code` and `opencode`, each with its own default sandbox image (overridable per run).
- **Static registry + live discovery** — publish an agent pointer (source repo + pinned commit + discovery metadata) with `push`, fetch it reproducibly with `pull`, rank the index with `search`, or query it over HTTP with `serve` (`GET /agents`, `/agents/:name`, `/search?q=`).
- **Agent-native by design** — every surface is usable by humans *and* by autonomous agents: `llms.txt` at the repo root, an `AGENT.md` card next to every manifest, and `--json` structured output (`{"ok":true,"data":…}` / `{"ok":false,"error":{"code","message","hint"}}`) on all commands.

## Requirements

- Go 1.24 or later (to build the CLI)
- Docker (only for `run` without `--dry-run`; everything else works without it)
- Git (for `push` / `pull`, which pin and fetch source commits)

## Install

```bash
git clone https://github.com/pixelrahulnotfound/agentPack.git && cd agentPack
go build -o agentpack ./cmd/agentpack
```

## Quickstart

```bash
# Scaffold a new agent (harness: claude-code or opencode)
./agentpack init ./my-agent --name my-agent --harness opencode

# Validate it and see the exact docker command (no Docker needed)
./agentpack run --dir ./my-agent --dry-run

# Run it for real inside the sandbox
./agentpack run --dir ./my-agent

# Compare two versions field by field
./agentpack diff examples/research-assistant/agentpack.yaml ./my-agent/agentpack.yaml

# Publish to a registry (a folder, or a git checkout of the index repo)
./agentpack push --dir ./my-agent --registry ./registry-index \
  --source-repo https://github.com/you/my-agent

# Fetch it back, reproducibly, at the pinned commit
./agentpack pull my-agent --registry ./registry-index --out ./pulled

# Search the index / serve it as a live API
./agentpack search "research papers" --registry ./registry-index
./agentpack serve --registry ./registry-index --port 8080
curl "http://127.0.0.1:8080/search?q=papers"
```

Append `--json` to any command for machine-readable output aimed at agents.

## Command reference

| Command | What it does |
|---|---|
| `init <dir> [--name N] [--harness H]` | Scaffold `agentpack.yaml` + `prompt.md` + `AGENT.md` + starter skill |
| `run [--dir D] [--image I] [--dry-run] [-- CMD…]` | Validate and launch the harness in Docker (`/work` mount); `--dry-run` prints the docker command only |
| `diff <fileA> <fileB>` | Field-level manifest comparison (what changed between versions) |
| `push --dir D --registry R --source-repo URL` | Publish `name → repo + pinned HEAD commit + discovery metadata` into the index |
| `pull NAME --registry R --out D` | Clone source, checkout pinned commit, copy + validate the agent into `D` |
| `search QUERY --registry R [--limit N]` | Rank indexed agents (name 3 pts, description 2, harness/tools 1) |
| `serve --registry R --port P` | Live discovery HTTP API (JSON, same envelope as `--json`) |
| `version` | Print version |

## The manifest

`agentpack.yaml` is the source of truth for an agent. The full frozen schema lives in [`SPEC.md`](SPEC.md); a real example lives in [`examples/research-assistant/`](examples/research-assistant/):

```yaml
name: research-assistant
version: 0.1.0
harness: claude-code            # claude-code | opencode
model: claude-sonnet-4-5
prompt: ./prompt.md
tools:
  - mcp: web-search
    scope: [search, fetch]
skills:
  - ./skills/paper-summarizer
permissions:
  filesystem: [./work]
  network: [api.github.com, arxiv.org]
  exec: false
memory:
  path: ./memory/
  max_entries: 200
```

## Registry & discovery

The registry is deliberately boring: one YAML file per agent in a directory (use a GitHub repo as that directory in production). Each entry points at a source repo plus a pinned 40-char commit, with optional discovery metadata (`description`, `harness`, `tools`) recorded automatically by `push`:

```yaml
name: research-assistant
version: 0.1.0
source_repo: https://github.com/example/research-assistant
commit: cce02bfa9010efbfaf810e09583a084c26db2360
manifest_path: research-assistant/agentpack.yaml
```

`search` queries this index locally; `serve` exposes it as a live service so other agents can discover packs without the CLI.

## Project layout

- [`SPEC.md`](SPEC.md) — frozen manifest + registry schema. Read this before changing validation.
- [`llms.txt`](llms.txt) — agent-facing project description (how an autonomous agent uses this repo with no human help).
- `cmd/agentpack/` — CLI entrypoint (`main.go`, `serve.go`).
- `internal/manifest/` — parse + validate `agentpack.yaml`.
- `internal/sandbox/` — Docker wrapper (shells out to the Docker CLI; no custom runtime).
- `internal/registry/` — index read/write/list/ranked search.
- `internal/diff/` — manifest version comparison.
- `internal/output/` — human vs `--json` output envelopes.
- `examples/research-assistant/` — one real, working example agent.
- `registry-index/` — local folder standing in for the index repo.
- `tests/` — CLI smoke test (`bash tests/smoke.sh`).

## Development

```bash
go vet ./...        # static checks
go test ./...       # unit tests (manifest, diff, registry)
bash tests/smoke.sh # end-to-end CLI smoke test
```

## Roadmap (explicitly not built yet)

Payments / agent-to-agent transactions, behavioral trust scoring and verification pipelines, hosted registry with auth/UI, multi-agent orchestration, and broader harness abstractions. These are separated future phases — see the internal build plan (not shipped in this repo).
