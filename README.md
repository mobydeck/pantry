# Pantry

Local note storage for coding agents. Your agent keeps notes on decisions, bugs, and context across sessions — no cloud, no API keys required, no cost.

## Features

- **Works with multiple agents** — Claude Code, Cursor, Codex, OpenCode, RooCode. One command sets up MCP config for your agent.
- **MCP native** — Runs as an MCP server exposing `pantry_store`, `pantry_search`, and `pantry_context` as tools.
- **Local-first** — Everything stays on your machine. Notes are stored as Markdown in `~/.pantry/shelves/`, readable in Obsidian or any editor.
- **Zero idle cost** — No background processes, no daemon, no RAM overhead. The MCP server only runs when the agent starts it.
- **Hybrid search** — FTS5 keyword search works out of the box. Add Ollama, OpenAI, or OpenRouter for semantic vector search.
- **Secret redaction** — 3-layer redaction strips API keys, passwords, and credentials before anything hits disk.
- **Cross-agent** — Notes stored by one agent are searchable by all agents. One pantry, many agents.

## Install

### Download a binary (recommended)

1. Go to the [Releases](../../releases) page and download the binary for your platform:

   | Platform | File |
   |----------|------|
   | macOS (Apple Silicon) | `pantry-darwin-arm64` |
   | macOS (Intel) | `pantry-darwin-amd64` |
   | Linux x86-64 | `pantry-linux-amd64` |
   | Linux ARM64 | `pantry-linux-arm64` |
   | Windows x86-64 | `pantry-windows-amd64.exe` |

2. Make it executable and move it to your PATH (macOS/Linux):

   ```bash
   chmod +x pantry-darwin-arm64
   mv pantry-darwin-arm64 /usr/local/bin/pantry
   ```

3. On macOS you may need to allow the binary in **System Settings → Privacy & Security** the first time you run it.

### Initialize

```bash
pantry init
```

### Connect your agent

```bash
pantry setup claude-code   # or: cursor, codex, opencode, roocode
```

This writes the MCP server entry into your agent's config file. Restart the agent and pantry will be available as a tool.

Run `pantry doctor` to verify everything is working.

### Tell your agent to use Pantry

MCP registration makes the tools available, but your agent also needs instructions to actually use them. The `setup` command installs a skill file automatically for agents that support it (Claude Code, Cursor, Codex). For other agents — or if you prefer to use a project-level rules file — add the following to your `AGENTS.md`, `.rules`, `CLAUDE.md`, or equivalent:

```markdown
## Pantry — persistent notes

You have access to a persistent note storage system via the `pantry` MCP tools.

**Session start — MANDATORY**: Before doing any work, retrieve notes from previous sessions:
- Call `pantry_context` to get recent notes for this project
- If the request relates to a specific topic, also call `pantry_search` with relevant terms

**Session end — MANDATORY**: After any task that involved changes, decisions, bugs, or learnings, call `pantry_store` with:
- `title`: short descriptive title
- `what`: what happened or was decided
- `why`: reasoning behind it
- `impact`: what changed
- `category`: one of `decision`, `pattern`, `bug`, `context`, `learning`
- `details`: full context for a future agent with no prior knowledge

Do not skip either step. Notes are how context survives across sessions.
```

## Semantic search (optional)

Keyword search (FTS5) works with no extra setup. To also enable semantic vector search, configure an embedding provider in `~/.pantry/config.yaml`:

**Ollama (local, free):**
```yaml
embedding:
  provider: ollama
  model: nomic-embed-text
  base_url: http://localhost:11434
```
Install [Ollama](https://ollama.com), then: `ollama pull nomic-embed-text`

**OpenAI:**
```yaml
embedding:
  provider: openai
  model: text-embedding-3-small
  api_key: sk-...
```

**OpenRouter:**
```yaml
embedding:
  provider: openrouter
  model: openai/text-embedding-3-small
  api_key: sk-or-...
```

After changing providers, rebuild the vector index:
```bash
pantry reindex
```

## Environment variables

All config file values can be overridden with environment variables. They take precedence over `~/.pantry/config.yaml` and are useful when the MCP host injects secrets into the environment instead of writing them to disk.

| Variable | Description | Example |
|----------|-------------|---------|
| `PANTRY_HOME` | Override pantry home directory | `/data/pantry` |
| `PANTRY_EMBEDDING_PROVIDER` | Embedding provider | `ollama`, `openai`, `openrouter` |
| `PANTRY_EMBEDDING_MODEL` | Embedding model name | `text-embedding-3-small` |
| `PANTRY_EMBEDDING_API_KEY` | API key for the embedding provider | `sk-...` |
| `PANTRY_EMBEDDING_BASE_URL` | Base URL for the embedding API | `http://localhost:11434` |
| `PANTRY_CONTEXT_SEMANTIC` | Semantic search mode | `auto`, `always`, `never` |

### Examples

Use OpenAI embeddings without putting the key in the config file:

```bash
PANTRY_EMBEDDING_PROVIDER=openai \
PANTRY_EMBEDDING_MODEL=text-embedding-3-small \
PANTRY_EMBEDDING_API_KEY=sk-... \
pantry search "rate limiting"
```

Point a second pantry instance at a different directory (useful for testing or per-workspace isolation):

```bash
PANTRY_HOME=/tmp/pantry-test pantry init
PANTRY_HOME=/tmp/pantry-test pantry store -t "test note" -w "testing" -y "because"
```

Pass the API key through the MCP server config so it is injected at launch time rather than stored on disk. Example for Claude Code (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "pantry": {
      "command": "pantry",
      "args": ["mcp"],
      "env": {
        "PANTRY_EMBEDDING_PROVIDER": "openai",
        "PANTRY_EMBEDDING_MODEL": "text-embedding-3-small",
        "PANTRY_EMBEDDING_API_KEY": "sk-..."
      }
    }
  }
}
```

Disable semantic search entirely for a single invocation (falls back to FTS5 keyword search):

```bash
PANTRY_CONTEXT_SEMANTIC=never pantry search "connection pool"
```

## Commands

```
pantry init                  Initialize pantry (~/.pantry)
pantry doctor                Check health and capabilities
pantry store                 Store a note
pantry search <query>        Search notes
pantry retrieve <id>         Show full note details
pantry list                  List recent notes
pantry remove <id>           Delete a note
pantry notes                 List daily note files (alias: log)
pantry config                Show current configuration
pantry config init           Generate a starter config.yaml
pantry setup <agent>         Configure MCP for an agent
pantry uninstall <agent>     Remove agent MCP config
pantry reindex               Rebuild vector search index
pantry version               Print version
```

## Storing notes manually

```bash
pantry store \
  -t "Switched to JWT auth" \
  -w "Replaced session cookies with JWT" \
  -y "Needed stateless auth for API" \
  -i "All endpoints now require Bearer token" \
  -g "auth,jwt" \
  -c "decision"
```

## Flag reference

`pantry store`:

| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-t` | Title (required) |
| `--what` | `-w` | What happened or was learned (required) |
| `--why` | `-y` | Why it matters |
| `--impact` | `-i` | Impact or consequences |
| `--tags` | `-g` | Comma-separated tags |
| `--category` | `-c` | `decision`, `pattern`, `bug`, `context`, `learning` |
| `--details` | `-d` | Extended details |
| `--source` | `-s` | Source agent identifier |
| `--project` | `-p` | Project name (defaults to current directory) |

`pantry list` / `pantry search` / `pantry notes`:

| Flag | Short | Description |
|------|-------|-------------|
| `--project` | `-p` | Filter to current project |
| `--limit` | `-n` | Maximum results |
| `--source` | `-s` | Filter by source agent |
| `--query` | `-q` | Text filter (list only) |

## Under the hood

### CGO-free, pure Go

Pantry is built without CGO. SQLite runs as a WebAssembly module inside the process via [wazero](https://github.com/tetratelabs/wazero) — a zero-dependency, pure-Go WASM runtime. This means:

- **No C compiler needed** — `go build` just works, no `gcc`, `musl`, or `zig` required
- **True static binaries** — the distributed binaries have no shared library dependencies (`ldd` shows nothing)
- **Cross-compilation is trivial** — all five platform targets (`GOOS`/`GOARCH`) build from a single `go build` invocation with `CGO_ENABLED=0`
- **Reproducible builds** — no C toolchain version drift

The tradeoff: first query of a session pays a one-time ~10 ms WASM compilation cost. Subsequent queries are fast.

### SQLite extensions

Two SQLite extensions are compiled into the binary as embedded WASM blobs:

**[sqlite-vec](https://github.com/asg017/sqlite-vec)** — vector similarity search. Pantry uses it to store note embeddings as 768- or 1536-dimensional `float32` vectors in a `vec0` virtual table, then retrieves the nearest neighbours with a single SQL query:

```sql
SELECT note_id, distance
FROM vec_notes
WHERE embedding MATCH ?
ORDER BY distance
LIMIT 20
```

The extension is loaded at connection open time via `sqlite3_load_extension` equivalent in the WASM host.

**FTS5** — SQLite's built-in full-text search virtual table. Notes are indexed in an `fts_notes` shadow table using the `porter` tokenizer (English stemming). FTS5 handles keyword search when no embedding provider is configured, and also runs alongside vector search as a hybrid fallback.

### Storage layout

Notes live in `~/.pantry/`:

```
~/.pantry/
  config.yaml          # embedding provider, model, API key
  pantry.db            # SQLite database (WAL mode)
  shelves/
    project/
      YYYY-MM-DD.md    # daily Markdown files — human-readable, Obsidian-compatible
```

The SQLite database holds structured note data and search indexes. The Markdown files in `shelves/` are append-only daily logs — they're the canonical human-readable view and survive even if the database is deleted (run `pantry reindex` to rebuild from them).

### GORM + vendored gormlite

The ORM layer uses [GORM](https://gorm.io) with a vendored copy of [gormlite](https://github.com/ncruces/go-sqlite3/tree/main/gormlite) — the SQLite GORM dialector from the same `ncruces/go-sqlite3` ecosystem. It's vendored (at `internal/gormlite/`) rather than imported as a module because the gormlite sub-module is versioned independently from the parent `go-sqlite3` package, and the sqlite-vec WASM binary constrains the host runtime to `go-sqlite3 v0.23.x`. Vendoring decouples dialector quality from host runtime version.

### Dependency count

The binary embeds everything it needs. Runtime dependencies: zero. The full `go.mod` direct dependency list:

| Package | Role |
|---------|------|
| `ncruces/go-sqlite3` | SQLite via WASM/wazero |
| `asg017/sqlite-vec-go-bindings` | Vector search extension (WASM blob) |
| `tetratelabs/wazero` | Pure-Go WebAssembly runtime |
| `gorm.io/gorm` | ORM |
| `modelcontextprotocol/go-sdk` | MCP server |
| `openai/openai-go` | OpenAI/OpenRouter embedding API |
| `spf13/cobra` | CLI |
| `google/uuid` | Note IDs |
| `go.yaml.in/yaml/v3` | Config parsing |

## License

MIT
