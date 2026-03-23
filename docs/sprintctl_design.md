# sprintctl Design Doc

## Summary

`sprintctl` should be extracted from the current repo-local sprint/orchestration workflow into its own standalone tool and repository.

This should not be a direct publication of the current `sprint.sh`. The goal is to extract the orchestration model behind it into a reusable CLI with:

- a generic execution engine
- explicit repo adapters
- structured comment/state handling
- deterministic dependency and closeout behavior
- a migration path for `anime-streaming-tracker` as the first adopter

Implementation language recommendation: **Go**.

## Problem

The current sprint system already acts like a standalone tool, but it is embedded in one repo and mixed together with repo-specific workflow assumptions.

Today it handles:

- epic and issue queue selection
- dependency sorting
- branch/worktree orchestration
- implementation/review/fixup loops
- blocked/unblocked reconciliation
- DB verification pauses/resume
- token/cost logging
- issue closeout and partial epic updates

The problem is that those generic orchestration concerns are coupled to anime-tracker-specific assumptions:

- epic body formatting conventions
- repo-specific labels (`agent:*`, `tier:*`, `blocked`)
- repo-specific docs hooks (`docs/current_roadmap.md`, `docs/currentstate.md`)
- repo-specific closeout semantics
- repo-specific branch naming patterns
- repo-specific comment/state conventions
- shell/regex parsing that is increasingly fragile

This makes the current system:

- hard to reason about
- hard to test
- hard to share externally
- easy to break with small workflow/body-format changes
- too repo-specific to be a clean reusable tool

## Goal

Build a standalone orchestration tool that:

1. owns the generic sprint/issue execution engine
2. moves repo-specific behavior into adapters/config
3. uses structured comment/state tags instead of heuristic thread inference wherever possible
4. supports `anime-streaming-tracker` as the first adapter without baking anime-specific workflow into the core
5. is clean enough for another team like Graperoot to evaluate and adapt

## Non-Goals

- Do not ship the current Bash script unchanged as the product.
- Do not make `anime-streaming-tracker` workflow assumptions part of the core by default.
- Do not try to solve every multi-agent workflow in v1.
- Do not turn this into a full project-management platform.
- Do not require adopters to use the same docs/update conventions as this repo.

## Design Principles

- Core orchestration must be generic.
- Repo workflow policy must be adapter-driven.
- Comment/state handling must be machine-readable.
- One run should produce one authoritative implementation summary and one authoritative review outcome per issue.
- Dependency resolution must be deterministic.
- Dry-run visibility must be first-class.
- Failure modes should be inspectable without reading shell logs line-by-line.

## Why Go

`sprintctl` should be built in **Go**.

Reasons:

- good fit for a standalone CLI
- easy binary distribution
- strong support for structured state, config parsing, and GitHub API clients
- much easier long-term maintenance than large Bash
- easier unit/integration testing for queueing, reconciliation, and comment parsing
- lower runtime friction for adopters than a Python dependency stack

Python would be acceptable for speed of iteration, but Bash should not remain the long-term implementation language.

## High-Level Architecture

### 1. Core engine

Reusable responsibilities:

- select execution target: epic or explicit issue list
- build execution queue
- resolve dependency order
- invoke implementation/review/fixup agents
- manage blocked/unblocked/requeue flow
- manage run-scoped comment/state markers
- manage branch/worktree flow
- manage closeout transitions
- emit logs and run summaries

### 2. Repo adapter layer

Repo-specific responsibilities:

- epic child-issue parsing rules
- issue brief conventions
- label names
- tier behavior
- branch naming strategy
- docs hooks
- board/project hooks
- DB verification workflow wording
- closeout policy

### 3. Prompt/template layer

Move prompts out of code into templates:

- implementation prompt
- review prompt
- fixup prompt
- DB verification prompt
- closeout prompt
- recap prompt

### 4. Structured comment protocol

Comments created or recognized by the tool should include explicit markers such as:

- `comment-role: implementation-summary`
- `comment-role: review`
- `comment-role: blocked`
- `comment-role: db-verification`
- `comment-role: closeout`
- `sprint-run: <id>`

This is required to eliminate ambiguity around duplicate summaries, duplicate reviews, stale blocked comments, and resume behavior.

## Recommended Repository Layout

```text
sprintctl/
  cmd/sprintctl/
  internal/github/
  internal/queue/
  internal/deps/
  internal/agents/
  internal/review/
  internal/git/
  internal/comments/
  internal/runs/
  internal/logging/
  pkg/config/
  adapters/
    anime-streaming-tracker/
      config.yaml
      prompts/
      hooks/
  examples/
  docs/
```

## Core Feature Set

### Queue selection

- run by epic
- run by explicit issue list
- stop after issue
- stop after phase
- dry-run queue preview

### Dependency handling

- parse `depends-on`
- parse `unblocks`
- topological ordering
- cycle detection
- out-of-scope dependency warnings

### Agent orchestration

- implementation agent selection
- review agent selection
- fixup loops
- escalation rules
- run-scoped state tracking

### Comment/state handling

- run-scoped tags
- role-scoped tags
- implementation summary dedupe
- review supersession
- blocked/resume semantics
- DB verification semantics

### Git/GitHub integration

- branch creation / reuse
- worktree support
- issue comments
- issue state updates
- label updates
- optional epic body updates

### Logging

- human-readable run log
- machine-readable run log
- token/cost accounting
- per-issue execution summary

## Anime-Tracker Adapter Requirements

The `anime-streaming-tracker` adapter should define:

### Issue parsing rules

- how epic child issues are identified
- how phase sections are interpreted
- how dependency tags are read

### Labels

- `agent:*`
- `tier:*`
- `blocked`
- any repo-specific planning labels

### Comment protocol

- required summary tags
- required review tags
- DB verification tags
- supersession expectations

### Docs hooks

- when to update `docs/current_roadmap.md`
- when to update `docs/currentstate.md`
- when docs should be left untouched

### Closeout policy

- what counts as closure
- what gets committed
- how parent issues/epics are revisited

## Current Failure Modes To Design Against

The extracted tool must explicitly prevent:

- epic parsing based on accidental `#NNN` mentions
- duplicate implementation summaries in one run
- duplicate review comments in one run without supersession
- stale blocked comments causing false requeues
- unblock/follow-up issues closing without reconciling the parent blocked issue
- DB verification flows mixing incompatible block/resume semantics
- shell-order bugs like calling a function before definition
- regex-based state inference where structured tags should exist

## Migration Plan

### Phase 0: freeze and document current behavior

Deliverables:

- current `sprint.sh` feature inventory
- generic vs repo-specific split
- current comment/tag taxonomy
- known failure modes
- current dependency semantics

This is the first artifact to show external reviewers.

### Phase 1: define the tool contract

Deliverables:

- config schema
- adapter interface
- structured comment protocol
- run-state model
- command surface proposal

### Phase 2: build core CLI skeleton

Deliverables:

- epic parsing
- explicit issue-list parsing
- dependency graph
- dry-run output
- adapter loading

### Phase 3: add GitHub and agent orchestration

Deliverables:

- issue fetch/update/comment support
- implementation/review loop
- branch/worktree logic
- run tagging

### Phase 4: add blocking/reconciliation flows

Deliverables:

- blocked/unblocked logic
- DB verification pause/resume
- review supersession
- authoritative per-run comment behavior

### Phase 5: anime-tracker adapter parity

Deliverables:

- this repo can run through `sprintctl`
- old `sprint.sh` can be retired or reduced to a thin wrapper

### Phase 6: externalization

Deliverables:

- adapter docs
- example config
- limitations
- adoption notes for another repo/team

## Minimum Acceptance Criteria

1. Core orchestration logic is no longer embedded in a repo-local Bash script.
2. Repo-specific workflow logic is externalized into an adapter/config layer.
3. Comment/state handling is structured enough to eliminate duplicate-summary ambiguity.
4. Dependency ordering and blocked/unblocked reconciliation are deterministic and testable.
5. `anime-streaming-tracker` can run through the extracted tool at feature parity for the currently used workflow.
6. Another team can inspect the repo and understand how to adapt it without reading anime-tracker internals.

## Deliverables

Required:

- standalone repo
- design doc
- config schema
- comment protocol spec
- working CLI prototype
- `anime-streaming-tracker` adapter
- migration notes from `sprint.sh` to `sprintctl`

Nice to have:

- dry-run visualization of queue/dependencies
- replayable run logs
- debug command for “why was this issue queued/blocked/requeued?”
- synthetic test harness for comment/reconciliation flows

## First Build Scope

The first build of `sprintctl` should be intentionally narrow:

- epic parsing
- explicit issue-list execution
- dependency sorting
- implementation/review/fixup loop
- blocked/unblocked reconciliation
- run-scoped comment roles
- anime-tracker adapter only

Do not try to support arbitrary repo behaviors on day one.

## Immediate Next Step

When spinning out the new repo, the first engineering artifact should be:

- this design doc
- plus a config schema draft for the adapter model

The first code milestone after repo creation should be a **dry-run-only** `sprintctl` that can:

- load an adapter
- parse an epic
- resolve dependencies
- print the exact queue it would execute

That gives you a stable foundation before agent execution and GitHub write actions are reintroduced.
