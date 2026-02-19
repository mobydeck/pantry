# Pantry

Local note storage for coding agents. Your agent keeps notes on decisions, bugs, and context across sessions — no cloud, no API keys, no cost.

## Features

- **Works with multiple agents** — Claude Code, Cursor, Codex, OpenCode. One command sets up MCP config for your agent.
- **MCP native** — Runs as an MCP server exposing `pantry_store`, `pantry_search`, and `pantry_context` as tools.
- **Local-first** — Everything stays on your machine. Notes are stored as Markdown in `~/.pantry/shelves/`, readable in Obsidian or any editor.
- **Zero idle cost** — No background processes, no daemon, no RAM overhead. The MCP server only runs when the agent starts it.
- **Hybrid search** — FTS5 keyword search works out of the box. Add Ollama, OpenAI, or OpenRouter for semantic vector search.
- **Secret redaction** — 3-layer redaction strips API keys, passwords, and credentials before anything hits disk.
- **Cross-agent** — Notes stored by one agent are searchable in all agents. One pantry, many agents.

## Install

```bash
go build ./cmd/pantry   # from repo root
# or: go install <module-path>/cmd/pantry@latest when published
pantry init
pantry setup claude   # or: cursor, codex, opencode
```

## Usage

### Store a note

```bash
pantry store \
  -t "Switched to JWT auth" \
  -w "Replaced session cookies with JWT" \
  -y "Needed stateless auth for API" \
  -i "All endpoints now require Bearer token" \
  -g "auth,jwt" \
  -c "decision"

# Long form also works:
pantry store \
  --title "Switched to JWT auth" \
  --what "Replaced session cookies with JWT" \
  --why "Needed stateless auth for API" \
  --impact "All endpoints now require Bearer token" \
  --tags "auth,jwt" \
  --category "decision"
```

### Search notes

```bash
pantry search "authentication"
pantry search "authentication" -p        # filter to current project
pantry search "authentication" -n 10     # show up to 10 results
```

### Retrieve full details

```bash
pantry retrieve <id>
```

### List recent notes

```bash
pantry list
pantry list -p                           # filter to current project
pantry list -n 20                        # show up to 20 notes
pantry list -q "jwt"                     # filter by query
```

### Remove a note

```bash
pantry remove <id>
```

## Commands

- `pantry init` - Initialize the pantry
- `pantry store` - Store a note in the pantry
- `pantry search` - Search notes
- `pantry retrieve <id>` - Retrieve full details of a note
- `pantry list` - List recent notes
- `pantry remove <id>` - Remove a note from the pantry
- `pantry log` - List daily note logs
- `pantry config` - Show/manage configuration
- `pantry config init` - Generate a starter config.yaml
- `pantry setup <agent>` - Setup MCP for agents (claude, cursor, codex, opencode)
- `pantry uninstall <agent>` - Remove agent setup
- `pantry reindex` - Rebuild vector index
- `pantry mcp` - Start MCP server

## Flag aliases

`pantry store`:

| Flag | Alias | Description |
|------|-------|-------------|
| `--title` | `-t` | Title of the note (required) |
| `--what` | `-w` | What happened or was learned (required) |
| `--why` | `-y` | Why it matters |
| `--impact` | `-i` | Impact or consequences |
| `--tags` | `-g` | Comma-separated tags |
| `--category` | `-c` | Category (decision, pattern, bug, context, learning) |
| `--details` | `-d` | Extended details or context |
| `--source` | `-s` | Source agent identifier |
| `--project` | `-p` | Project name override |

`pantry list` / `pantry search` / `pantry log`:

| Flag | Alias | Description |
|------|-------|-------------|
| `--project` | `-p` | Filter to current project (list/search: boolean; log: string) |
| `--limit` | `-n` | Maximum number of results |
| `--source` | `-s` | Filter by source |
| `--query` | `-q` | Search query (list only) |

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
  topup_recent: true            # also include recent notes
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
