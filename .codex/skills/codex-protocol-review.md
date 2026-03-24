# codex-protocol-review

Use this skill when Codex is reviewing shared protocol, tag semantics, or ownership boundaries.

## Review Checklist

- Confirm each rule has one canonical home.
- Confirm `AGENTS.md` contains only shared protocol.
- Confirm agent-local files contain only standing agent behavior.
- Confirm skills contain triggered workflows or tool procedures.
- Confirm design docs are not silently acting as always-loaded instruction roots.

## Drift Checks

- If a rule changes in one file, check whether it exists elsewhere and remove or update duplicates in the same change.
- If a rule affects all agents or inter-agent semantics, it belongs in `AGENTS.md`.
- If a rule affects only one agent's internal behavior, it belongs in that agent's file or skill.
- If a rule is only a procedure or template, it belongs in a skill.
