# AgentPack Manifest Spec (v1 — frozen)

`agentpack.yaml` describes one runnable personal agent: who it is, what model/harness runs it, its prompt, tools, skills, permissions, and memory.

## Schema

```yaml
name: research-assistant        # required, ^[a-z0-9][a-z0-9-_]{1,63}$
version: 0.1.0                  # required, semver
description: Short human text   # optional
harness: claude-code            # required, one of: claude-code, opencode
model: claude-sonnet-4-5        # required, non-empty string
prompt: ./prompt.md             # required, path (relative to manifest dir) to prompt file
tools:                          # optional, default []
  - mcp: github                 # required, MCP server name, non-empty
    scope: [read]               # optional, list of capability strings
  - mcp: web-search
    scope: [search, fetch]
skills:                         # optional, list of local skill dir paths
  - ./skills/paper-summarizer
permissions:                    # optional
  filesystem: [./work]          # optional, list of allowed path prefixes
  network: [api.github.com]     # optional, list of allowed domains; [] = no network
  exec: false                   # optional, allow arbitrary exec in sandbox, default false
memory:                         # optional
  path: ./memory/               # where conversation memory is stored
  max_entries: 200              # cap, default 200
```

## Validation rules

1. File must parse as YAML and map to the schema above. Unknown top-level fields are rejected.
2. `name` must match `^[a-z0-9][a-z0-9-_]{1,63}$`.
3. `version` must be valid semver `MAJOR.MINOR.PATCH` (no `v` prefix, prerelease allowed).
4. `harness` must be one of `claude-code`, `opencode`.
5. `model` non-empty.
6. `prompt` must point to an existing file relative to the manifest directory.
7. Every `skills[]` entry must point to an existing directory containing a `SKILL.md`.
8. `permissions.filesystem[]` entries must be relative paths (no absolute paths, no `..` escapes).
9. `permissions.network[]` entries must be bare hostnames (no scheme, no path).
10. `memory.max_entries` must be > 0 when memory is set.

## Registry index entry (v1)

One YAML file per published agent at `registry-index/<name>.yaml`:

```yaml
name: research-assistant
version: 0.1.0
source_repo: https://github.com/example/research-assistant
commit: abc123def456abc123def456abc123def456abcd
manifest_path: examples/research-assistant/agentpack.yaml
updated_at: 2026-09-24T00:00:00Z
description: A personal research assistant that summarizes papers   # optional, from manifest
harness: claude-code                                               # optional, from manifest
tools: [web-search, github]                                        # optional, MCP names from manifest
```

`commit` must be a 40-char hex SHA. `description`/`harness`/`tools` are discovery metadata recorded by `agentpack push`; entries without them remain valid. Consumers `pull` by reading this file, cloning the source repo at the pinned commit, and reading `manifest_path`. `agentpack search` / `serve` rank over name (3 pts), description (2 pts), harness/tools (1 pt).

## AGENT.md (per-manifest, agent-readable)

Every publishable agent directory must contain an `AGENT.md` with: what the agent does, its tools/skills and scopes, its permission boundaries, how to invoke it (`agentpack run`), and its input/output contract. It must be derivable from `agentpack.yaml` alone — `agentpack init` generates a starter version.
