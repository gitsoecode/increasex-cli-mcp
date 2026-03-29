# IncreaseX Spec

This repository implements `increasex`, a Go-based wrapper around Increase with:

- a human-friendly CLI
- a local MCP server over stdio
- one shared application core
- preview-first write flows

Durable design rules:

- Build against the official Increase Go client directly.
- Do not shell out to the official `increase` CLI.
- Keep CLI and MCP thin and route both through shared services.
- Mask sensitive data by default.
- Treat MCP tools as stateless except for short-lived confirmation tokens.
- Keep MCP tool schemas narrowly typed and task-oriented.
