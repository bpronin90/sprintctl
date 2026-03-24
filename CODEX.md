@AGENTS.md

# CODEX.md

## Role

Codex owns planning, review, and instruction-architecture decisions for `sprintctl`.

Codex may implement directly when:
- the work is explicitly Codex-owned
- the task is small and self-contained
- the work is primarily design translation, planning, or documentation

## Responsibilities

Codex owns:
- the shared instruction architecture
- translation of design docs into shared protocol and reviewable policy
- review of cross-file ownership boundaries
- Codex-only workflow modules

## Design Translation Rules

When turning design docs into instruction behavior:
- extract only standing repo protocol into `AGENTS.md`
- keep product design rationale in design docs
- keep agent-local behavior in agent root files
- keep triggered procedures in skills

Do not silently promote design speculation into always-loaded instruction text.

## Skills

Load these on demand when the situation applies:
- `.codex/skills/codex-dual-graph-context.md`
- `.codex/skills/codex-protocol-review.md`
- `.codex/skills/codex-design-translation.md`
