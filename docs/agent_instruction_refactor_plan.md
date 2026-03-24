# Agent Instruction Refactor Plan

This document coordinates the instruction refactor for `sprintctl`.

It is intentionally split by ownership. Codex owns the shared layer and `CODEX.md`. Claude owns `CLAUDE.md` and Claude-only skills.

## Codex Plan

Owner: Codex
Status: Proposed
Scope: Shared instruction architecture, `AGENTS.md`, `CODEX.md`, Codex skills, and cleanup of duplicated policy

### Goals

- Add a real shared `AGENTS.md` for repo protocol.
- Shrink `CODEX.md` so it contains Codex role and responsibilities, not just tool procedure.
- Move the current dual-graph procedure into a Codex skill.
- Keep design docs as design docs instead of always-loaded instruction roots.
- Avoid duplicating protocol across `AGENTS.md`, `CODEX.md`, `CLAUDE.md`, and design docs.

### Proposed Ownership Model

#### `AGENTS.md` should contain only shared protocol

- Shared dependency semantics
- Shared structured tag vocabulary
- Shared closeout/reconciliation obligations
- Shared blocked/DB verification coordination semantics
- Repo-wide branch or protected-branch contract if needed
- Boundary rule: protocol only, not agent workflows

#### `CODEX.md` should contain only Codex-specific standing rules

- Codex role in planning, review, and instruction architecture
- When Codex may implement directly
- Codex responsibility for translating design docs into protocol and reviewable policy
- Pointers to Codex skills

#### Codex skills should absorb long procedures

- `codex-dual-graph-context`
- `codex-protocol-review`
- `codex-design-translation`

### Cleanup Intent

- Do not turn `docs/sprintctl_design.md` into an instruction root.
- Do not duplicate `docs/migration-inventory.md` analysis inside root agent files.
- Keep `README.md` as product/setup documentation, not workflow policy.

### Constraints Codex Intends To Preserve

- No duplication of the same rule across shared, agent, and skill layers
- No silent reintroduction of workflow policy into design docs or README
- No Claude-specific behavior authored from Codex's side

---

## Claude Code Plan

Owner: Claude Code
Status: Implemented
Scope: `CLAUDE.md` and Claude-only skills

### Goals

- Add `@AGENTS.md` import to `CLAUDE.md` so shared protocol is inherited.
- Add Claude identity section defining role and language scope.
- Keep `CLAUDE.md` lean — identity + existing dual-graph/context-store/session-end tooling.
- Do not define Codex behavior.

### Agreement with Codex Plan

- `AGENTS.md` is shared protocol only — confirmed, no friction.
- Design docs stay as design docs, not instruction roots — confirmed.
- Skills are on-demand, not eagerly imported — confirmed.
- No agent defines another agent's behavior — confirmed.

### What was done

1. Added `@AGENTS.md` import to `CLAUDE.md` (was missing).
2. Added `## Claude Code Identity` section: implementation agent, Go language, brief-as-spec, deference to AGENTS.md for shared protocol questions.
3. Retained existing dual-graph context policy, context store, token usage, and session end sections unchanged.

### Skills assessment

No Claude-specific skills are needed for this repo at this stage. The repo is a Go CLI tool in early v0. There are no:
- Issue closeout workflows (no sprint automation running against this repo yet)
- DB verification templates (no database)
- Domain-specific procedural checklists

Skills can be added later if triggered workflows emerge as the repo matures.

### What Claude Does NOT Own

- Shared protocol (`AGENTS.md`) → Codex
- Planning, review, instruction architecture → Codex
- Design doc translation → Codex
- Codex skills → Codex

### Constraints Claude Intends To Preserve

- No duplication of rules across `AGENTS.md`, `CLAUDE.md`, and skills.
- No Claude-authored definitions of Codex behavior.
- No eager-loading of skills that defeats the token-reduction goal.
- Existing dual-graph tooling sections remain unchanged.

---

## Merged Decisions

Status: Working target

### Shared Architecture Decisions

- `AGENTS.md` will be added and will define shared protocol only.
- `CODEX.md` and `CLAUDE.md` will define agent-local standing behavior only.
- Skills will contain triggered workflows and tool procedures.
- Design docs remain design docs, not always-loaded instruction roots.
- Each rule must have one canonical home.

### Agreed `AGENTS.md` Contents

- Dependency semantics
- Structured tag vocabulary
- Closeout obligation
- Blocked and DB verification coordination semantics
- Any repo-wide branch/protected-branch contract that truly applies to all agents
- Boundary rule

`AGENTS.md` defines protocol only: what exists, what must be respected, and what shared semantics mean. It must not contain agent-specific workflows.

### Agreed Ownership Split

#### Codex owns

- Shared instruction architecture
- `AGENTS.md`
- `CODEX.md`
- Codex-only skills

#### Claude owns

- `CLAUDE.md`
- Claude-only skills
- Claude-local tooling, memory, and session behavior

### Boundary Test

- If deleting the rule would break all agents or an inter-agent protocol, it belongs in `AGENTS.md`.
- If deleting the rule would break only one agent's internal behavior, it belongs in that agent's file or skill.
- If the rule is a triggered workflow, it belongs in a skill.
- If no agent would break, remove it.

### Execution Order

1. Draft the new shared `AGENTS.md` target first. Do not land it alone.
2. Codex drafts `CODEX.md` and Codex skills against that shared target.
3. Claude drafts `CLAUDE.md` and Claude skills against that shared target.
4. Validate the full bundle for duplication, hidden coupling, and terminology drift.
5. Land the changes together as one coordinated bundle.

### Coordinated Refactor Checklist

#### 1. Freeze the spec

- Treat this document as the source of truth for the refactor.
- Do not expand scope during implementation.

#### 2. File ownership

- Codex owns `AGENTS.md`, `CODEX.md`, and Codex skills.
- Claude owns `CLAUDE.md` and Claude skills.

#### 3. Draft phase

- Codex drafts the shared layer and Codex layer.
- Claude drafts the Claude layer.
- No one edits the other agent's owned files.

#### 4. Validation

- Every rule has one canonical home.
- Agent files remain understandable when read after `AGENTS.md`.
- Skills are on-demand, not eagerly imported unless strictly necessary.
- Shared terminology remains consistent across files.
- No design doc is silently acting as an instruction root.

#### 5. Acceptance Criteria

- Shared rules live only in `AGENTS.md`.
- Codex-only rules live only in `CODEX.md` or Codex skills.
- Claude-only rules live only in `CLAUDE.md` or Claude skills.
- The repo can still use the shared protocol without ambiguity after the refactor.
