# sprintctl

`v0` currently includes a local dry-run planner prototype in Go.

Planned usage:

```bash
go run ./cmd/sprintctl \
  -config examples/anime-streaming-tracker.config.yaml \
  -epic-body examples/anime-streaming-tracker.epic.md \
  -issue-dir examples/issues
```

Optional filters:

```bash
go run ./cmd/sprintctl \
  -config examples/anime-streaming-tracker.config.yaml \
  -epic-body examples/anime-streaming-tracker.epic.md \
  -issue-dir examples/issues \
  -phase 1-2 \
  -until 53
```

Current scope:

- loads root config and adapter config
- parses epic child issues from local markdown
- applies optional phase filtering
- reads local issue bodies
- resolves dependencies
- prints a deterministic dry-run queue

Current non-goals:

- GitHub API access
- branch/worktree orchestration
- comment writing
- agent execution
