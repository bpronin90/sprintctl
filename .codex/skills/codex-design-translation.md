# codex-design-translation

Use this skill when Codex is translating design docs or migration analysis into repo instruction architecture.

## Translation Rules

- Extract protocol from design docs; do not copy rationale verbatim into root instruction files.
- Keep product decisions in design docs.
- Promote only stable, operationally relevant semantics into `AGENTS.md`.
- Keep adapter-specific behavior out of shared protocol unless the repo explicitly treats it as shared.

## Source Priority

For `sprintctl`, prefer this order when translating:
1. `docs/v0-decisions.md`
2. `docs/sprintctl_design.md`
3. `docs/migration-inventory.md`

Use the design docs to decide what the protocol should be, not to create long always-loaded prompts.
