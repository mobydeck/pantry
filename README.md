# Pantry

Local memory for coding agents. Your agent remembers decisions, bugs, and context across sessions — no cloud, no API keys, no cost.

## Features

- **Works with multiple agents** — Claude Code, Cursor, Codex, OpenCode. One command sets up MCP config for your agent.
- **MCP native** — Runs as an MCP server exposing `pantry_store`, `pantry_search`, and `pantry_context` as tools.
- **Local-first** — Everything stays on your machine. Items are stored as Markdown in `~/.pantry/shelf/`, readable in Obsidian or any editor.
- **Zero idle cost** — No background processes, no daemon, no RAM overhead. The MCP server only runs when the agent starts it.
- **Hybrid search** — FTS5 keyword search works out of the box. Add Ollama, OpenAI, or OpenRouter for semantic vector search.
- **Secret redaction** — 3-layer redaction strips API keys, passwords, and credentials before anything hits disk.
- **Cross-agent** — Items saved by one agent are searchable in all agents. One vault, many agents.

## Install

```bash
go build ./cmd/pantry   # from repo root
# or: go install <module-path>/cmd/pantry@latest when published
pantry init
pantry setup claude   # or: cursor, codex, opencode
```

## Usage

### Store an item

```bash
pantry store \
  --title "Switched to JWT auth" \
  --what "Replaced session cookies with JWT" \
  --why "Needed stateless auth for API" \
  --impact "All endpoints now require Bearer token" \
  --tags "auth,jwt" \
  --category "decision"
```

### Search items

```bash
pantry search "authentication"
pantry search "authentication" --project
```

### Retrieve details

```bash
pantry retrieve <id>
```

### List recent items

```bash
pantry list
pantry list --project
```

### Remove an item

```bash
pantry remove <id>
```

## Commands

- `pantry init` - Initialize the pantry vault
- `pantry store` - Store an item in the pantry
- `pantry search` - Search pantry items
- `pantry retrieve <id>` - Retrieve full details of an item
- `pantry list` - List recent items
- `pantry remove <id>` - Remove an item from pantry
- `pantry sessions` - List session files
- `pantry config` - Show/manage configuration
- `pantry config init` - Generate a starter config.yaml
- `pantry setup <agent>` - Setup MCP for agents (claude, cursor, codex, opencode)
- `pantry uninstall <agent>` - Remove agent setup
- `pantry reindex` - Rebuild vector index
- `pantry mcp` - Start MCP server

## Configuration

Generate a starter config:

```bash
pantry config init
```

This creates `~/.pantry/config.yaml`:

```yaml
embedding:
  provider: ollama              # ollama | openai | openrouter
  model: nomic-embed-text
  base_url: http://localhost:11434
  # api_key: sk-...            # required for openai/openrouter

context:
  semantic: auto                # auto | always | never
  topup_recent: true            # also include recent items
```

Run `pantry reindex` after changing embedding providers to rebuild the vector index.

## Testing

See [TESTING.md](TESTING.md) for unit and integration tests. Quick run:

```bash
go test ./...
./testing/test-pantry.sh   # full CLI integration test
```

## License

MIT
