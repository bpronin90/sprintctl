# sprint.sh Migration Inventory

Source: `anime-streaming-tracker/sprint.sh` (1931 lines, Bash)
Target: `sprintctl` (Go CLI with adapter model)

Each section documents **what sprint.sh does**, then classifies behavior as:
- **CORE** — generic orchestration, belongs in sprintctl engine
- **ADAPTER** — anime-streaming-tracker-specific, belongs in adapter config
- **AMBIGUOUS** — needs an explicit design decision for sprintctl

---

## 1. Command Inputs and Modes

### Epic mode (default)
```
./sprint.sh <epic-issue-number>
./sprint.sh <epic-issue-number> --until 53
./sprint.sh <epic-issue-number> --phase 1
./sprint.sh <epic-issue-number> --phase 1-3
```

| Behavior | Classification | Notes |
|----------|---------------|-------|
| Positional arg = epic issue number | **CORE** | |
| `--until N` truncates queue at issue #N (inclusive) | **CORE** | |
| `--phase N` restricts to one phase section | **CORE** | Phase header format is adapter-specific (see §3) |
| `--phase N-M` restricts to phase range | **CORE** | |
| Validate epic is open; die if closed | **CORE** | |

### Solo mode
```
./sprint.sh --issue 53
./sprint.sh --issue 522,509
```

| Behavior | Classification | Notes |
|----------|---------------|-------|
| `--issue` / `--issues` bypasses epic entirely | **CORE** | |
| Comma-separated list, executed in given order | **CORE** | |
| No sprint branch created; uses current branch | **CORE** | |
| No epic body updates or recap | **CORE** | |
| `SOLO_MODE=true` flag gates epic-only logic | **CORE** | |

### Validation
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Must be on a non-main branch | **AMBIGUOUS** | Should sprintctl enforce this or leave it to adapter? The current check `die "Currently on 'main'"` is reasonable as core but the protected branch name could be configurable |
| Requires `gh`, `claude`, `gemini`, `codex`, `jq` on PATH | **ADAPTER** | Agent binaries are adapter-specific; core should only require `gh` and `jq` (or Go equivalents) |
| `gh auth status` check | **CORE** | |

---

## 2. Queue Selection Rules

| Behavior | Classification | Notes |
|----------|---------------|-------|
| Epic body fetched via `gh api` (not `gh issue view` — broken for this repo) | **ADAPTER** | The `gh issue view` workaround is anime-tracker-specific. Core should use API by default anyway |
| All `#NNN` references extracted from epic body in order | **CORE** | |
| Backtick-wrapped refs supported: `` `#NNN` `` | **CORE** | |
| Deduplication via `awk '!seen[$0]++'` | **CORE** | |
| `--until` truncates at matching issue; dies if not found | **CORE** | |
| Mid-sprint re-read of epic body (`check_for_new_issues`) adds newly-appeared issues to queue | **CORE** | Only in epic mode. Respects `--until` by skipping new issues if set |
| Re-queued issues from unblock/reconciliation appended to end | **CORE** | |

### Known edge case: accidental `#NNN` matches
The regex `grep -oP '#\K\d+'` matches ANY `#NNN` in the epic body — including casual mentions, markdown anchors, or unrelated references. The design doc explicitly calls this out as a failure mode to design against.

**AMBIGUOUS** — sprintctl should define whether child issues are identified by:
- (a) regex on `#NNN` in list items (current behavior, fragile)
- (b) a structured format like `- [ ] #NNN` (task list syntax)
- (c) a machine-readable section/tag in the epic body
- (d) GitHub sub-issues API (if available)

---

## 3. Epic Child / Phase Parsing Rules

### Phase parsing (lines 86–98, 530–545)
```
## Phase 1
- #51 — Set up schema (agent:claude)
- #52 — Build matching logic (agent:claude)

## Phase 2
- #53 — Wire up UI (agent:gemini)
```

| Behavior | Classification | Notes |
|----------|---------------|-------|
| Phase headers matched by regex: `^#{2,4}\s.*[Pp]hase\s+(\d+)` | **AMBIGUOUS** | The header format is a convention. Should sprintctl define a canonical format or let adapters specify a regex? |
| Content between phase headers extracted; `#NNN` grepped from it | **CORE** (mechanism) | |
| Phase range `N-M` iterates and concatenates | **CORE** | |
| Issues deduped across phases | **CORE** | |
| No phases → all `#NNN` refs extracted from full body | **CORE** | |
| Phase number is integer only (no names) | **AMBIGUOUS** | Could support named phases in sprintctl |

### Agent assignment (from labels)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| `agent:claude`, `agent:gemini`, `agent:codex` labels on child issues | **ADAPTER** | Label names and available agents are repo-specific |
| No agent label → skip issue, post warning comment | **CORE** (mechanism) / **ADAPTER** (label name) | |
| Agent determines which invocation wrapper is used | **CORE** | |

### Tier assignment (from labels)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| `tier:heavy` label → upgraded model/turns/reasoning | **ADAPTER** | Label name and tier definitions are repo-specific |
| Default tier if no tier label | **CORE** | |
| Tier affects model selection, turn limits, reasoning effort | **CORE** (mechanism) | |

---

## 4. Dependency Parsing and Ordering

### Dependency tags (parsed from child issue bodies)
| Tag | Direction | Classification |
|-----|-----------|---------------|
| `depends-on: #N` | Forward: this issue needs #N first | **CORE** |
| `unblocks: #N` | Reverse: #N needs this issue first (→ `dep_map[N] += this`) | **CORE** |

### Topological sort (`topo_sort_issues`, lines 243–344)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Iterative post-order DFS | **CORE** | |
| Cycle detection → `die` with cycle node identified | **CORE** | |
| Out-of-scope dependencies (not in sprint queue) → warn and ignore | **CORE** | |
| Always runs (even if no deps declared — no-op in that case) | **CORE** | |
| Each child issue body fetched via API to read dep tags | **CORE** | |
| Regex: `grep -oiP "depends-on:\s*#?\K\d+"` | **CORE** | Case-insensitive, optional `#` prefix |
| Regex: `grep -oiP "unblocks:\s*#?\K\d+"` | **CORE** | |

### Known edge cases
- Multi-dependency: `depends-on: #51, #52` is NOT supported — only one `#N` per line match. Each dep needs its own `depends-on:` line or appears on separate lines.
- **AMBIGUOUS**: Should sprintctl support comma-separated deps on one line?

---

## 5. Blocked/Unblocked Behavior

### Blocking during review (lines 1592–1654)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Reviewer outputs `VERDICT=BLOCKED` | **CORE** | |
| Reviewer includes `blocked-by: #N` or `blocked-by: none` | **CORE** | |
| Script posts structured comment with `blocked-by:` tag | **CORE** | |
| `blocked` label added to issue | **ADAPTER** (label name) / **CORE** (mechanism) | |
| Issue added to `BLOCKED_ISSUES` array, skipped for closeout | **CORE** | |
| Epic body patched with 🚫 symbol | **ADAPTER** | Emoji and body-patching convention |

### Unblock reconciliation on close (lines 927–999)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| After closing an issue, fetch all open issues with `blocked` label | **CORE** | |
| For each, find the LATEST comment containing `blocked-by:` tag | **CORE** | Uses newest-first ordering to avoid stale tag matches |
| If `blocked-by: #just_closed` → remove `blocked` label, post comment, re-queue | **CORE** | |
| Dedup against all tracking arrays before re-queue | **CORE** | |
| Epic body updated on re-queue | **ADAPTER** | |

### Parent/dependency reconciliation on close (lines 1002–1103)
Two-pass reconciliation:

| Pass | Behavior | Classification |
|------|----------|---------------|
| **A — outbound** | Read closed issue's body for `unblocks: #N` → reconcile those targets | **CORE** |
| **B — inbound** | Search API for open issues whose body contains `depends-on: #just_closed` | **CORE** |
| Both | Remove `blocked` label, post reconciliation comment, re-queue if not already seen | **CORE** |
| Both | Verify target is actually open before reconciling | **CORE** |

### Known edge cases
- **Stale blocked-by tags**: Fixed by using only the LATEST comment with `blocked-by:`. Earlier versions matched all comments and could false-fire on superseded blocks.
- **Unblock without reconciling parent**: The two-pass system (GAP 2 + GAP 3 fixes) addresses this but relies on both comment-thread tags AND body tags being correct.
- **AMBIGUOUS**: Should sprintctl use structured state (e.g., a run-state file or database) instead of comment-thread grep for block tracking?

---

## 6. Implementation / Review / Fixup Loop Semantics

### Implementation phase (lines 1335–1536)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Agent invoked with combined prompt: branch, issue body, thread context, file boundary rule, instructions | **CORE** (structure) |
| Thread context: issue body + last VERDICT comment + last N comments, deduped, char-capped | **CORE** |
| `MAX_THREAD_CHARS=6000`, `MAX_BODY_CHARS=4000`, `MAX_IMPL_SUMMARY_CHARS=3000` | **CORE** (configurable) |
| Agent instructed to post summary comment with `agent-summary: true` tag on last line | **CORE** | Machine-readable tag |
| Script detects if agent already posted summary (`agent_posted_summary`) → suppresses stdout fallback | **CORE** | Prevents duplicate summaries |
| If agent output is empty/error/too short (<20 chars) → skip posting | **CORE** | |
| If agent hit turn limit → retry with extended turns (continuation prompt) | **CORE** | |
| `post_diff_summary` always runs to capture actual git diff | **CORE** | |
| `IMPL_START_TIME` epoch captured for summary dedup window | **CORE** | |

### Anime-tracker-specific implementation prompts
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Claude/Codex: "Read docs/currentstate.md first" | **ADAPTER** | Repo-specific docs |
| Gemini: restricted to `frontend/**`, forbidden from `backend/**`, `docs/**`, etc. | **ADAPTER** | Repo-specific path constraints |
| `GH_CLI_WARNING` about broken `gh issue view` | **ADAPTER** | Repo-specific workaround |
| File boundary rule: "Only modify files explicitly listed in the issue brief" | **AMBIGUOUS** | Good default but adapter might want to relax it |

### Review phase (lines 1543–1698)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| `MAX_REVIEW_CYCLES=3` | **CORE** (configurable) |
| `ESCALATE_AFTER=2` cycles | **CORE** (configurable) |
| Cross-agent review: Codex reviews Claude/Gemini; Claude reviews Codex | **CORE** (mechanism) / **ADAPTER** (agent pairings) |
| Reviewer outputs `VERDICT=APPROVED`, `VERDICT=FEEDBACK`, or `VERDICT=BLOCKED` | **CORE** | |
| Review checklist includes: acceptance criteria, file scope, verification, complexity, Gemini path check | **ADAPTER** (checklist items) / **CORE** (review structure) |
| FEEDBACK → fixup loop; BLOCKED → block and skip; APPROVED → proceed to close | **CORE** | |

### Fixup loop (lines 1655–1697)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Tight context: last 2 comments, 4000 char cap | **CORE** | |
| Agent fixes exactly what was requested | **CORE** | |
| Fixup output posted as "Revision" comment with cycle number | **CORE** | |
| Uses fixup-tier model (between light and impl) | **CORE** (mechanism) | |

### Escalation (lines 1659–1664)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| If non-Claude agent fails `ESCALATE_AFTER` cycles → hand off to Claude | **ADAPTER** | Claude as escalation target is repo-specific |
| Posts escalation comment on issue | **CORE** (mechanism) | |

### Final review safety valve (lines 1700–1743)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| If max cycles hit without approval → one extra final review pass | **CORE** | |
| Final review can only output APPROVED or BLOCKED (no more FEEDBACK) | **CORE** | Prevents infinite loop |
| If still not approved → mark blocked with `blocked-by: none` | **CORE** | |

### Codex self-review gate (lines 1745–1803)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| After approval, if agent was Codex → one self-review pass | **ADAPTER** | Codex-specific trust level |
| If self-review finds a gap → Codex fixes, then Claude re-checks | **ADAPTER** | Agent-specific pairing |
| If Claude rejects self-fix → blocked | **CORE** (mechanism) | |

---

## 7. DB Verification Pause/Resume Behavior

### Triggering a DB block (lines 1596–1636)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Reviewer includes `db-verification-required: true` in output | **CORE** | Machine-readable tag |
| Optional `db-resume-agent: <agent>` tag specifies who resumes | **CORE** | |
| Script posts structured comment with both tags + `blocked-by: none` | **CORE** | |
| `blocked` label added | **ADAPTER** (label name) | |
| Terminal output shows human the required steps and resume command | **CORE** | |
| Issue added to `BLOCKED_ISSUES`, sprint continues to next issue | **CORE** | Does NOT exit the sprint |

### Detecting pending DB verification on re-entry (lines 1246–1333)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Fetch all comments desc, find latest `db-verification-required: true` and latest `db-verified: true` | **CORE** | |
| Compare positions (desc index): if `db-verified` is more recent → resolved | **CORE** | |
| If resolved AND `db-resume-agent` tag exists → continuation mode (agent verifies) | **CORE** | |
| If resolved AND no `db-resume-agent` → review mode (reviewer does light check) | **CORE** | |
| If NOT resolved → print steps to terminal, post reminder comment, skip issue | **CORE** | |

### Resume paths (lines 1374–1435)
| Path | Behavior | Classification |
|------|----------|---------------|
| **continuation** | Resume agent runs verification-only prompt, then normal review follows | **CORE** |
| **review** | Reviewer does light approval check (VERDICT=APPROVED or VERDICT=FEEDBACK) | **CORE** |

### Known edge cases
- Stale `db-verified: true` from a prior block cycle could match a newer block. Fixed by comparing comment indices (desc order).
- `db-resume-agent` tag is read from the block comment body, not from a separate source — if the block comment is edited, behavior changes.
- **AMBIGUOUS**: Should sprintctl use a structured state store for DB verification status instead of comment-thread position comparison?

---

## 8. Closeout Behavior

### Closing procedure (lines 1805–1856)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Reviewer runs closing procedure (light model) | **CORE** | |
| Update `docs/currentstate.md` if docs in scope | **ADAPTER** | Repo-specific docs |
| Update `docs/current_roadmap.md` if scope/progress changed | **ADAPTER** | Repo-specific docs |
| Stage ONLY files listed in issue brief + docs + napkin files | **CORE** (mechanism) / **ADAPTER** (allowed extras) |
| Explicit `git add` paths — NO `git add -A` | **CORE** | |
| `git diff --cached --name-only` verification before commit | **CORE** | |
| Commit message: `feat(#N): <issue title>` | **AMBIGUOUS** | Conventional commits format is reasonable as default but prefix should be configurable |
| `git push origin <sprint-branch>` | **CORE** | |
| `gh issue close` | **CORE** | |
| NO `closes`/`fixes`/`resolves` keywords in commit | **CORE** | Prevents accidental cross-issue closes |
| Always-allowed files: `.claude/napkin.md`, `.codex/napkin.md`, `.gemini/napkin.md` | **ADAPTER** | Agent-specific napkin files |

### Post-close actions
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Add to `COMPLETED_ISSUES` | **CORE** | |
| Update epic comment: "✅ Issue #N completed" | **CORE** (mechanism) | |
| Patch epic body: mark issue line with ✅ | **CORE** (mechanism) / **ADAPTER** (emoji convention) |
| `check_for_unblocked_issues` | **CORE** | |
| `check_for_parent_references` | **CORE** | |
| `check_for_new_issues` (epic mode only) | **CORE** | |

### Epic body status patching (`update_epic_body_status`, lines 1105–1152)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Regex matches `- #N` or `- \`#N\`` lines in epic body | **CORE** (mechanism) | |
| Prepends ✅ (completed) or 🚫 (blocked) | **AMBIGUOUS** | Symbol convention could be adapter-configurable |
| Strips prior symbol before adding new one (no double-marking) | **CORE** | |
| PATCH via `gh api` with JSON-escaped body | **CORE** | |
| No-op if no matching line found | **CORE** | |

---

## 9. Comment/Tag Conventions

### Machine-readable tags written by sprint.sh
| Tag | Where | Purpose | Classification |
|-----|-------|---------|---------------|
| `agent-summary: true` | Last line of agent summary comment | Dedup detection | **CORE** |
| `blocked-by: #N` or `blocked-by: none` | Block comments | Unblock reconciliation | **CORE** |
| `db-verification-required: true` | DB block comment | DB pause detection | **CORE** |
| `db-resume-agent: <agent>` | DB block comment | Resume routing | **CORE** |
| `db-verified: true` | Human confirmation comment | DB resume trigger | **CORE** |
| `VERDICT=APPROVED/FEEDBACK/BLOCKED` | Reviewer output (stdout, not always in comment) | Loop control | **CORE** |
| `SELF_REVIEW=PASS/FIXED` | Codex self-review output | Self-review gate | **ADAPTER** |
| `NO_UPDATE_NEEDED` | Brief check output | Brief update skip | **CORE** |

### Comment roles (implicit, not tagged)
| Role | When posted | By whom | Classification |
|------|------------|---------|---------------|
| Implementation summary | After impl | Agent or script fallback | **CORE** |
| Revision summary | After fixup | Script | **CORE** |
| Review/verdict | During review | Reviewer agent | **CORE** |
| Block notification | On BLOCKED verdict | Script | **CORE** |
| Unblock notification | On reconciliation | Script | **CORE** |
| DB verification pause | On DB block | Script | **CORE** |
| DB verification reminder | On re-entry to blocked DB issue | Script | **CORE** |
| Escalation notice | On agent escalation | Script | **CORE** |
| Epic progress update | After each issue start/complete/block | Script | **CORE** |
| Sprint recap | End of epic run | Codex | **ADAPTER** (Codex as recap agent) |

### AMBIGUOUS decisions needed
- Current tags are ad-hoc strings. Design doc proposes structured `comment-role:` and `sprint-run:` prefixes. Should all comments get a `comment-role` tag?
- Should `VERDICT=` be in the comment body (queryable) or only in stdout? Currently stdout-only for most cases.
- The `agent-summary: true` tag on last line is fragile — agents sometimes fail to include it. Should sprintctl use a different detection mechanism?

---

## 10. Branch/Worktree Rules

| Behavior | Classification | Notes |
|----------|---------------|-------|
| Must not be on `main` at script start | **CORE** / **AMBIGUOUS** (protected branch name configurable?) |
| Epic mode: sprint branch = `sprint/<slugified-epic-title>` | **CORE** (mechanism) / **ADAPTER** (naming pattern) |
| Slugification: lowercase, non-alnum → `-`, collapse, trim | **CORE** | |
| Branch created off current branch; falls back to checkout if exists | **CORE** | |
| Solo mode: stays on current branch, no new branch | **CORE** | |
| All agents instructed: "stay on `<sprint-branch>`, do NOT switch" | **CORE** | |
| Commits and pushes happen per-issue on the sprint branch | **CORE** | |
| No worktree support currently | **CORE** | Design doc lists worktree as a feature; sprint.sh doesn't use `git worktree` |

---

## 11. Model/Agent Configuration

### Agent invocation wrappers
| Wrapper | Model config | Classification |
|---------|-------------|---------------|
| `invoke_claude_impl` | Tier-dependent: sonnet-4-6 / opus-4-6, 15/25 turns | **ADAPTER** |
| `invoke_claude_light` | sonnet-4-6, 10 turns | **ADAPTER** |
| `invoke_claude_fixup` | sonnet-4-6, 15 turns | **ADAPTER** |
| `invoke_codex_impl` | Tier-dependent: gpt-5.4, medium/high reasoning | **ADAPTER** |
| `invoke_codex_light` | gpt-5.4, medium reasoning | **ADAPTER** |
| `invoke_codex_fixup` | gpt-5.4, medium reasoning | **ADAPTER** |
| `invoke_gemini` | gemini --yolo (no model config) | **ADAPTER** |

### Allowed tools (Claude)
```
--allowedTools "Bash(*) Read(*) Write(*) Edit(*) mcp__*"
```
**ADAPTER** — tool permissions are agent/repo-specific.

### Token/cost tracking (lines 560–612)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Running totals: cost USD, input/output/cache tokens (Claude), total tokens (Codex) | **CORE** |
| Claude usage parsed from `--output-format json` response | **CORE** (mechanism) / **ADAPTER** (Claude-specific JSON format) |
| Codex usage parsed from stdout `tokens used\n<number>` | **ADAPTER** | Codex-specific output format |
| Written to log file (not stdout, to avoid corrupting captured output) | **CORE** |
| Summary printed at sprint end | **CORE** |

---

## 12. Brief Update Check

| Behavior | Classification | Notes |
|----------|---------------|-------|
| Only in epic mode, after 3+ issues completed | **CORE** (configurable threshold) |
| Reviewer checks if prior work requires updating upcoming issue's brief | **CORE** |
| If yes: posts updated brief comment on issue | **CORE** |
| If no: outputs `NO_UPDATE_NEEDED` | **CORE** |

---

## 13. Recap Behavior

### Solo mode recap
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Terminal log only: completed, blocked, branch, token usage | **CORE** | |

### Epic mode recap (lines 1880–1912)
| Behavior | Classification | Notes |
|----------|---------------|-------|
| Codex posts recap comment on epic | **ADAPTER** (Codex as recap agent) |
| Codex updates `docs/currentstate.md` and `docs/current_roadmap.md` | **ADAPTER** |
| Codex commits and pushes doc updates | **ADAPTER** |
| If ALL issues completed, close epic; otherwise leave open | **CORE** |
| Partial run noted if `--until` or `--phase` was set | **CORE** |

---

## 14. Known Edge Cases and Failure Modes

| Issue | Current behavior | Severity | Classification |
|-------|-----------------|----------|---------------|
| Accidental `#NNN` in epic body | Extracted as child issue | **High** | **CORE** — needs structured format |
| Duplicate implementation summaries | `agent_posted_summary` dedup via `agent-summary: true` tag | Fixed | **CORE** |
| Duplicate review comments | Not explicitly prevented — relies on loop structure | **Medium** | **CORE** — design doc wants `comment-role` tags |
| Stale `blocked-by` comments causing false unblocks | Fixed: uses LATEST comment with tag (desc order) | Fixed | **CORE** |
| Unblock/follow-up closing without parent reconciliation | Fixed: two-pass reconciliation (GAP 2 + GAP 3) | Fixed | **CORE** |
| DB verification mixing block/resume semantics | Fixed: index-based comparison of block vs verified comments | Fixed | **CORE** |
| Shell function called before definition | Fixed: `parse_issues_from_epic` defined before first use (duplicated at line 70 and 514) | Fixed | N/A (Bash-specific) |
| Regex-based state inference | Pervasive — `blocked-by:`, `db-verified:`, `VERDICT=` all regex-parsed | **High** | **CORE** — design doc wants structured tags |
| Agent fails to include `agent-summary: true` | Fallback to posting stdout as summary | **Medium** | **CORE** |
| Agent posts empty/error output | `is_valid_output` check (>20 chars, no error patterns) | Fixed | **CORE** |
| Agent hits turn limit | Retry with extended turns + continuation prompt | Fixed | **CORE** |
| `gh issue view` broken on repo | Workaround: all reads use `gh api` | **Repo-specific** | **ADAPTER** |
| Comment body >64KB | Truncated with notice | Fixed | **CORE** |
| Issue body >4000 chars in prompt | `truncate_text` with head/tail split | Fixed | **CORE** |
| Multi-dependency on one line | Not supported — one dep per `depends-on:` line | **Low** | **AMBIGUOUS** |
| Comma in `--issue` arg with spaces | Not handled — relies on no-space comma separation | **Low** | **CORE** |
| `parse_issues_from_epic` duplicated (lines 70 and 514) | Bash sourcing order issue; both are identical | N/A | N/A (Bash-specific) |

---

## 16. Summary of Ambiguous Decisions for sprintctl

These need explicit design decisions before implementation:

1. **Epic child-issue format**: Regex `#NNN` extraction vs structured task-list syntax vs machine-readable section
2. **Phase header format**: Regex-matched `## Phase N` vs adapter-configurable pattern vs structured format
3. **Protected branch name**: Hardcoded `main` vs adapter-configurable
4. **Commit message format**: `feat(#N): title` vs adapter-configurable template
5. **Epic body status symbols**: ✅/🚫 vs adapter-configurable markers
6. **Block state tracking**: Comment-thread grep vs structured run-state store
7. **DB verification state**: Comment position comparison vs explicit state machine
8. **Multi-dep syntax**: One-per-line only vs comma-separated support
9. **File boundary rule**: Hard default vs adapter-configurable strictness
10. **Escalation target**: Always Claude vs adapter-configurable fallback agent
11. **Comment-role tags**: Ad-hoc tags vs design doc's `comment-role:` prefix system
12. **Sprint-run scoping**: No run ID currently vs design doc's `sprint-run: <id>` tagging
13. **Codex self-review gate**: Agent-specific trust level vs generic configurable post-approval gate
14. **Brief update threshold**: Hardcoded 3 issues vs configurable
15. **Recap agent**: Hardcoded Codex vs configurable
