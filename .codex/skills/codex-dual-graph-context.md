# codex-dual-graph-context

Use this skill when Codex needs repository retrieval through the dual-graph MCP workflow.

## Rules

- Call `graph_continue` first before grep, bash exploration, or broad file reads.
- If `graph_continue` returns `needs_project=true`, call `graph_scan` with the current project directory.
- If `graph_continue` returns `skip=true`, do not do broad exploration; read only specific named files or ask a scoped question if necessary.
- Read recommended files with one `graph_read` call per file or `file::symbol` entry.
- Obey `confidence`, `max_supplementary_greps`, and `max_supplementary_files` caps strictly.
- Do not call `graph_retrieve` more than once per turn.
- After edits, call `graph_register_edit` with the changed files or symbols.

## Context Layering

When available, load only the relevant context layers:
- `.dual-graph-context/PROJECT_CONTEXT.md`
- `.dual-graph-context/SESSION_CONTEXT.md`
- only the relevant `.dual-graph-context/packs/*.md` files for the task

Ask only for missing context. Do not ask for full history or full context dumps.
