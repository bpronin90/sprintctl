# sprintctl Schema Draft

## Purpose

This document defines the first-pass schema for `sprintctl` phase 1:

- config schema
- adapter contract
- structured comment protocol
- run-state model

This is intentionally narrow. It establishes stable shapes for validation and implementation without locking the project into a final storage layer.

## Design Goals

- Keep core orchestration generic.
- Push repo-specific behavior into adapter configuration.
- Make comment and run state machine-readable.
- Preserve enough structure to support dry-run, resume, review supersession, and blocked/unblocked reconciliation.

## Top-Level Artifacts

The initial schema is split into four logical artifacts:

1. `SprintctlConfig`
2. `AdapterConfig`
3. `CommentEnvelope`
4. `RunState`

## `SprintctlConfig`

This is the root configuration loaded by the CLI.

```yaml
version: v1
repo:
  id: anime-streaming-tracker
  owner: bpronin90
  name: anime-streaming-tracker
  default_branch: main
github:
  host: github.com
  api_base_url: https://api.github.com
execution:
  worktree_root: .worktrees
  default_base_branch: main
  max_fixup_passes: 2
  reviewers_required: 1
adapter:
  kind: anime-streaming-tracker
  config_path: adapters/anime-streaming-tracker/config.yaml
comments:
  tag_prefix: sprintctl
  allow_supersede: true
logging:
  run_log_dir: .sprintctl/runs
  emit_json: true
```

### Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `version` | string | yes | Schema version. Start with `v1`. |
| `repo.id` | string | yes | Stable internal identifier for the target repo. |
| `repo.owner` | string | yes | GitHub owner or org. |
| `repo.name` | string | yes | GitHub repository name. |
| `repo.default_branch` | string | yes | Default branch for issue work. |
| `github.host` | string | no | Defaults to `github.com`. |
| `github.api_base_url` | string | no | Override for GitHub Enterprise. |
| `execution.worktree_root` | string | no | Root directory for worktrees. |
| `execution.default_base_branch` | string | no | Branch used when no override is given. |
| `execution.max_fixup_passes` | integer | no | Upper bound for review/fixup loops. |
| `execution.reviewers_required` | integer | no | Minimum successful review count. |
| `adapter.kind` | string | yes | Adapter identifier. |
| `adapter.config_path` | string | yes | Path to repo-specific adapter config. |
| `comments.tag_prefix` | string | no | Marker prefix used in structured comments. |
| `comments.allow_supersede` | boolean | no | Enables review and summary supersession. |
| `logging.run_log_dir` | string | no | Directory for run artifacts. |
| `logging.emit_json` | boolean | no | Emit machine-readable logs. |

## `AdapterConfig`

The adapter owns repo-specific policy and parsing.

```yaml
version: v1
kind: anime-streaming-tracker
issue_parsing:
  epic_child_patterns:
    - '(?m)^- \\[ \\] #(?P<issue>[0-9]+)'
  dependency_patterns:
    depends_on:
      - '(?i)depends-on:\\s*#(?P<issue>[0-9]+)'
    unblocks:
      - '(?i)unblocks:\\s*#(?P<issue>[0-9]+)'
labels:
  blocked: blocked
  agent_prefix: agent:
  tier_prefix: tier:
comments:
  required_roles:
    - implementation-summary
    - review
    - blocked
    - db-verification
    - closeout
docs_hooks:
  roadmap_path: docs/current_roadmap.md
  currentstate_path: docs/currentstate.md
closeout:
  close_issue_on_success: true
  revisit_parent_on_close: true
branching:
  issue_branch_template: issue/{{issue_number}}-{{slug}}
  epic_branch_template: epic/{{issue_number}}-{{slug}}
```

### Adapter responsibilities

- issue parsing rules
- label mapping
- branch naming strategy
- docs update hooks
- DB verification wording and state behavior
- closeout policy

### Boundaries

The adapter must not define:

- generic dependency resolution algorithm
- core review loop semantics
- GitHub transport implementation
- run log storage format

## `CommentEnvelope`

Every tool-authored comment should embed a structured envelope. The visible body can remain human-readable, but the envelope must be parseable.

```yaml
comment_role: review
sprint_run_id: run_2026_03_23T13_15_04Z_ab12cd
issue_number: 482
attempt: 1
status: approved
supersedes_comment_id: 1234567890
created_by: sprintctl
created_at: 2026-03-23T13:15:04Z
payload:
  reviewer: codex-review
  summary: No blocking findings.
```

### Required envelope fields

| Field | Type | Notes |
| --- | --- | --- |
| `comment_role` | enum | One of `implementation-summary`, `review`, `blocked`, `db-verification`, `closeout`, `resume`. |
| `sprint_run_id` | string | Correlates all comments written during one run. |
| `issue_number` | integer | GitHub issue number. |
| `attempt` | integer | Monotonic per role within a run. |
| `status` | string | Role-specific state such as `active`, `superseded`, `approved`, `changes-requested`, `blocked`, `resolved`. |
| `supersedes_comment_id` | integer/null | Previous comment replaced by this one. |
| `created_by` | string | Agent or tool identity. |
| `created_at` | RFC3339 timestamp | Creation time. |
| `payload` | object | Role-specific structured data. |

### Role-specific payloads

`implementation-summary`

- `head_branch`
- `commit_sha`
- `files_changed`
- `summary`
- `tests`

`review`

- `reviewer`
- `decision`
- `findings`
- `follow_up_required`

`blocked`

- `reason`
- `blocking_issue_numbers`
- `resume_hint`

`db-verification`

- `environment`
- `required_checks`
- `operator_action`

`closeout`

- `resolution`
- `closed_issue_numbers`
- `parent_reconciliation`

## `RunState`

`RunState` is the machine-readable state of one orchestration run.

```yaml
run_id: run_2026_03_23T13_15_04Z_ab12cd
status: running
mode: epic
target:
  epic_issue_number: 455
queue:
  - issue_number: 482
    status: in_review
    dependency_state: ready
    branch: issue/482-review-supersession
    worktree_path: .worktrees/482
    attempts:
      implementation: 1
      review: 1
      fixup: 0
active_issue_number: 482
blocked:
  issue_numbers: []
started_at: 2026-03-23T13:15:04Z
updated_at: 2026-03-23T13:19:18Z
```

### Core fields

| Field | Type | Notes |
| --- | --- | --- |
| `run_id` | string | Unique run identifier. |
| `status` | enum | `planned`, `running`, `paused`, `blocked`, `failed`, `completed`. |
| `mode` | enum | `epic`, `issues`, `resume`, `dry-run`. |
| `target` | object | Original selection input. |
| `queue` | array of `IssueRunState` | Ordered execution plan. |
| `active_issue_number` | integer/null | Current issue being processed. |
| `blocked.issue_numbers` | array[int] | Issues currently blocked during this run. |
| `started_at` | timestamp | Run start time. |
| `updated_at` | timestamp | Last mutation time. |

### `IssueRunState`

| Field | Type | Notes |
| --- | --- | --- |
| `issue_number` | integer | GitHub issue number. |
| `status` | enum | `queued`, `implementing`, `in_review`, `fixup`, `blocked`, `awaiting-db-verification`, `done`, `failed`, `skipped`. |
| `dependency_state` | enum | `ready`, `waiting`, `external`, `cycle`. |
| `branch` | string | Working branch for this issue. |
| `worktree_path` | string/null | Worktree path when enabled. |
| `attempts.implementation` | integer | Implementation attempts. |
| `attempts.review` | integer | Review passes. |
| `attempts.fixup` | integer | Fixup passes. |
| `comment_ids` | object | Role-to-latest-comment lookup. |
| `blocked_by` | array[int] | Blocking issue numbers. |
| `result` | object/null | Final closeout summary. |

## Validation Rules

- `version` must be `v1` for the first release line.
- `adapter.kind` in `SprintctlConfig` must match `kind` in `AdapterConfig`.
- `execution.max_fixup_passes` must be `>= 0`.
- `execution.reviewers_required` must be `>= 0`.
- `CommentEnvelope.attempt` must be `>= 1`.
- A new `review` comment with `status != superseded` must supersede any earlier active review in the same run.
- `IssueRunState.status=done` requires a non-null `result`.
- `IssueRunState.status=blocked` requires at least one `blocked_by` entry or an active `blocked` comment.

## Storage Guidance

- Config should live in YAML because adapters will edit it directly.
- Validation should use JSON Schema generated or mirrored from the Go structs.
- Run state can be stored as JSON under `.sprintctl/runs/<run-id>.json`.
- Comment envelopes should be renderable as front matter or fenced metadata blocks inside GitHub comments.

## Initial Implementation Mapping

These are the first Go structs implied by this draft:

- `pkg/config.Config`
- `pkg/config.AdapterRef`
- `pkg/adapter.Config`
- `pkg/comments.Envelope`
- `pkg/runs.State`
- `pkg/runs.IssueState`

## Open Questions

- Whether multi-reviewer flows need distinct reviewer identities in core v1 or only in payload.
- Whether adapter parsing rules should allow regex only or support pluggable parsers.
- Whether run state should include token and cost tracking in the core model or only in logs.
