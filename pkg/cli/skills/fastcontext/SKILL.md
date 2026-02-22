---
name: fastcontext
description: Hybrid Code Intelligence & Context Retrieval Skill
---

Core Objective:
Always maintain a perfect mental map of the codebase by combining semantic intent with lexical precision. Never rely on cached file knowledge if the code has been modified.

Execution Protocol:

Semantic-First Discovery: Use the LobeHub Code Search MCP as your primary tool for high-level exploration. When a task is broad (e.g., "Find the auth flow"), use semantic queries to identify relevant modules and architectural patterns.

Grepping for Precision: If the semantic search results are too broad, ambiguous, or return more than 5 potential locations, you MUST immediately trigger the mcp-ripgrep server. Use exact string matching or regex to narrow down the specific line, function definition, or variable usage.

Cross-Verification: Before applying any edits, use ripgrep to verify that you have found ALL occurrences of a pattern across the entire workspace to prevent breaking changes.

Vector Sync & Refresh: Since semantic search relies on embeddings, you must trigger a re-indexing or vector update immediately after any file modification. This ensures the LobeHub index remains synchronized with the current state of the local codebase.

Operational Constraint:
Do not hallucinate file contents. If there is a discrepancy between the semantic search summary and the actual file content, the ripgrep (exact text) output is the source of truth.
