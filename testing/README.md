# Pantry Test Suite

Integration tests for the pantry CLI. Exercises all subcommands with a temporary pantry.

## Run

From project root:

```bash
./testing/test-pantry.sh
```

Or with a pre-built binary:

```bash
PANTRY_BIN=./pantry ./testing/test-pantry.sh
```

## What it tests

- **init** — Creates shelves, index.db, config.yaml
- **store** — Basic store, with details, with related-files, validation (requires --title/--what)
- **search** — Keyword search, limit, result content
- **list** — Recent items, limit
- **retrieve** — By full ID, short ID, details content
- **remove** — Delete item, verify it's gone from search
- **sessions** — List session files
- **config** — Show config, init --force
- **reindex** — Rebuild vector index (FTS-only when embeddings unavailable)
- **setup/uninstall** — Unknown agent fails, cursor with --config-dir
- **mcp** — Server starts
- **help** — Root, store, search

## Notes

- Uses `PANTRY_HOME` in a temp dir; never touches `~/.pantry`
- Config overrides embedding to `http://127.0.0.1:19999` so tests run without Ollama (FTS-only path)
- Vector search (sqlite-vec) is enabled via `github.com/asg017/sqlite-vec-go-bindings/ncruces`
- Setup/uninstall use `--config-dir` to avoid modifying real agent configs
