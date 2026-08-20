# sprintctl v0 Decisions

## Purpose

This document locks the minimum behavior decisions required to start implementing the first `sprintctl` prototype.

`v0` means:

- conservative scope
- dry-run first
- no GitHub write actions
- no agent execution
- no attempt to reach full `sprint.sh` parity yet

These decisions are derived from the design doc and the migration inventory. They are intended to prevent accidental reimplementation of Bash quirks as core product behavior.

## Decision Rules

- Prefer explicit structure over heuristic parsing.
- Prefer adapter-configurable formats over repo-specific hardcoding.
- Prefer local deterministic state over comment-thread inference.
- Defer anything not required for dry-run queue planning.

## Locked Decisions

### 1. Epic child issue format

Decision:
- Use adapter-configured extraction patterns.
- `anime-streaming-tracker` v0 will use task-list style issue lines such as `- [ ] #123`.
- Do not support arbitrary `#NNN` extraction from the full epic body as the default behavior.

Reason:
- The migration inventory identifies accidental `#NNN` matches as a high-severity failure mode.

Implication for v0:
- Dry-run queue building reads only adapter-approved child issue patterns.

### 2. Phase parsing

Decision:
- Phase filtering is supported in core.
- Phase header detection is adapter-configurable.
- `anime-streaming-tracker` v0 uses `^#{2,4}\s.*[Pp]hase\s+(\d+)`.

Reason:
- Phase filtering is generic, but header conventions are repo-specific.

Implication for v0:
- The adapter exposes phase header regex and child issue regex separately.

### 3. Protected branch handling

Decision:
- Protected branch names are config-driven, not hardcoded to `main`.
- v0 dry-run does not fail on branch state unless the command explicitly requires branch-sensitive behavior.

Reason:
- Branch protection is real workflow policy, but it does not block queue planning.

Implication for v0:
- Branch validation is deferred for execution commands.

### 4. Dependency syntax

Decision:
- Support one dependency reference per matched tag in v0.
- Support repeated lines such as multiple `depends-on:` entries.
- Do not support comma-separated dependencies on one line in v0.

Reason:
- This matches current behavior while keeping parsing simple and deterministic.

Implication for v0:
- The dry-run planner warns on unrecognized multi-dependency formats only if they become visible during implementation.

### 5. Block state tracking

Decision:
- Authoritative block state belongs in structured run state, not comment-thread grep.
- Historical comment parsing remains a migration concern, not the long-term state model.

Reason:
- The design doc explicitly pushes toward machine-readable state and away from regex inference.

Implication for v0:
- Dry-run can define the state model now, even before write actions exist.

### 6. DB verification state

Decision:
- DB verification is modeled as explicit run state, not by comparing comment positions.
- v0 dry-run only needs the state shape, not the runtime behavior.

Reason:
- The inventory shows the Bash implementation relies on fragile thread ordering.

Implication for v0:
- We preserve the concept in schemas and state docs, but do not implement pause/resume logic yet.

### 7. Comment tagging

Decision:
- Standardize on structured comment envelopes with explicit role and run identity.
- v0 adopts `comment_role` and `sprint_run_id` as the canonical fields.
- Legacy ad-hoc tags like `agent-summary: true` and bare `VERDICT=` are not the target protocol.

Reason:
- Duplicate and stale comment interpretation is one of the main reasons this tool exists.

Implication for v0:
- The dry-run CLI does not write comments, but the protocol is now considered fixed for future phases.

### 8. Sprint run scoping

Decision:
- All future tool-authored comments and state transitions are scoped by a `run_id`.

Reason:
- This is required for deterministic supersession and resume behavior.

Implication for v0:
- Dry-run output should include a generated run identifier shape, even if nothing is persisted yet.

### 9. Commit message format

Decision:
- Commit message templates are adapter-configurable.
- No default commit convention is required for v0.

Reason:
- Dry-run does not commit, so this should not become a premature core constraint.

Implication for v0:
- Leave commit templates out of the initial implementation path.

### 10. Epic body status markers

Decision:
- Epic progress marker style is adapter-configurable.
- v0 does not mutate epic bodies.

Reason:
- The marker convention is presentation policy, not queue-planning logic.

Implication for v0:
- No status patching logic is needed in the first CLI milestone.

### 11. File boundary rules

Decision:
- File-boundary restrictions belong to adapter prompts and execution policy, not dry-run planning.

Reason:
- This matters during implementation and review, not queue construction.

Implication for v0:
- Out of scope for the first CLI milestone.

### 12. Escalation targets and self-review gates

Decision:
- Agent pairings, escalation targets, and post-approval self-review are adapter policy.
- v0 does not implement them.

Reason:
- These are execution concerns and the current Bash behavior is tool-specific.

Implication for v0:
- Keep them out of core dry-run design.

## v0 Build Target

The first implementation target is a dry-run-only CLI that can:

- load root config
- load adapter config
- parse epic child issues using adapter rules
- optionally filter by phase
- parse dependency tags from issue bodies
- topologically order the queue
- print the exact planned queue and dependency status

It must not:

- invoke agents
- post comments
- mutate issue bodies
- close issues
- push branches

## Not Decided Yet

These remain intentionally open until execution work begins:

- final Go package boundaries
- exact CLI flags beyond dry-run needs
- whether run state is stored as JSON, YAML, or both
- whether GitHub access is only live API or can be replayed from fixtures
- how adapter hooks are represented in code

## Immediate Implementation Consequence

The next coding step is:

- scaffold `cmd/sprintctl`
- implement config and adapter loading
- implement epic parsing plus dependency resolution
- output a deterministic dry-run queue for `anime-streaming-tracker`
