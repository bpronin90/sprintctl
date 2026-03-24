# AGENTS.md

## Shared Protocol

This file defines shared protocol only: inter-agent semantics, shared tags, and repo-wide contracts.

Agent-specific behavior belongs in `CODEX.md`, `CLAUDE.md`, or agent-owned skills. Do not duplicate shared protocol into agent-local files.

## Dependency Semantics

Use these dependency tags in issue or artifact bodies when the workflow requires them:

- `depends-on: #N`
- `unblocks: #N`

These tags define dependency order and downstream reconciliation semantics.

## Structured Tags

These tags are shared coordination vocabulary. Agent-specific procedures for when to emit them belong in agent-owned files or skills.

- `blocked-by: #N|none` — identifies the active blocker for blocked work
- `db-verification-required: true` — human DB verification is required before the workflow can continue
- `db-resume-agent: <agent>` — names the agent that should resume after human DB verification
- `db-verified: true` — the requested DB step completed successfully
- `VERDICT=APPROVED` — review outcome is approval
- `VERDICT=FEEDBACK` — review found issues that must be addressed
- `VERDICT=BLOCKED` — workflow is blocked pending a named blocker or required action

## Closeout Obligation

If closing or completing item B affects item A through `unblocks:` or other explicit dependency semantics, closure is not complete until A is reconciled.

- Revisit the dependent item explicitly.
- Remove stale blocked state when applicable.
- Re-state whether the dependent item is now actionable or still blocked.

## Blocked and DB Verification Semantics

- Blocked state must be machine-readable.
- DB verification state must be machine-readable.
- Shared protocol defines the tag vocabulary; agent-owned files define the procedures.

## Branch / Execution Contract

- Do not assume `main` is always the protected branch; protected-branch handling is repo policy, not hardcoded naming.
- Shared execution semantics should stay adapter-driven where the product requires it.

## Boundary Rule

`AGENTS.md` defines protocol only: what exists, what tags mean, and what all agents must respect.

Design docs remain design docs. Triggered workflows belong in skills. Agent-local standing rules belong in agent root files.
