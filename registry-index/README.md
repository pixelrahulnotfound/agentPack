# registry-index (v1)

Static index: one YAML file per published agent, pointing at a source repo + pinned commit.

```yaml
name: research-assistant
version: 0.1.0
source_repo: https://github.com/example/research-assistant
commit: abc123def456abc123def456abc123def456abcd
manifest_path: examples/research-assistant/agentpack.yaml
updated_at: 2026-09-24T00:00:00Z
```

Publish with `agentpack push`; fetch with `agentpack pull`. In production this directory is a separate GitHub repo; here it is a local folder with the same format.
