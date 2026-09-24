# research-assistant — agent-readable card

- What: personal research assistant. Given a topic or paper, returns a concise, cited summary.
- Harness: claude-code. Model: claude-sonnet-4-5.
- Tools: web-search (search, fetch), github (read).
- Skills: ./skills/paper-summarizer (TL;DR, method, results, limits).
- Permissions: filesystem [./work], network [api.github.com, arxiv.org], exec false.
- Memory: ./memory/ (max 200 entries).
- Invoke: `agentpack run --dir examples/research-assistant` from the repo root.
- Input: natural-language research task (CLI args or stdin). Output: markdown with TL;DR + sources.
- Manifest: ./agentpack.yaml is the source of truth; this file is derived from it.
