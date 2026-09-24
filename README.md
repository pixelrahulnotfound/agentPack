# AgentPack

Package a personal AI agent's **entire setup** — system prompt, MCP tool refs, skills, permission scopes, memory config — into one versioned, runnable manifest (`agentpack.yaml`), plus a CLI to build, run, and share it.

Agent-native by design: every part is usable by a human (CLI + README) and by a Personal Agent (`llms.txt`, per-manifest `AGENT.md`, `--json` structured output).

## Quick start

```bash
go build -o agentpack ./cmd/agentpack
./agentpack init ./my-agent --name my-agent
./agentpack run --dir ./my-agent --dry-run
./agentpack diff examples/research-assistant/agentpack.yaml my-agent/agentpack.yaml
./agentpack push --dir examples/research-assistant --registry ./registry-index --source-repo https://github.com/example/research-assistant
./agentpack pull research-assistant --registry ./registry-index --out ./pulled
./agentpack run --dir examples/research-assistant --dry-run
./agentpack --json diff <a> <b>   # machine-readable
```

## Layout

- `SPEC.md` — frozen manifest schema (read first).
- `cmd/agentpack/` — CLI (`init`, `run`, `pull`, `push`, `diff`, `--json`).
- `internal/manifest|sandbox|registry|diff/` — libraries.
- `examples/research-assistant/` — working example agent.
- `registry-index/` — v1 static index (one file per agent → source repo + pinned commit).
- `llms.txt` — agent-facing project description.

## Out of scope (v1)

Payments/x402, trust scoring, multi-harness abstraction, hosted registry/auth/UI, multi-agent compose, live search API. See `plan.md`.
