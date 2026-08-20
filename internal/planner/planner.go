package planner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/benpronin/sprintctl/internal/config"
)

type Options struct {
	EpicBodyPath string
	IssueDir     string
	Phase        string
	UntilIssue   int
}

type Plan struct {
	RunID          string
	Repo           string
	Adapter        string
	SourceEpicPath string
	SelectedIssues []int
	OrderedIssues  []PlannedIssue
	Warnings       []string
}

type PlannedIssue struct {
	IssueNumber     int
	Dependencies    []int
	Unblocks        []int
	DependencyState string
}

func Build(root config.Root, adapter config.Adapter, opts Options) (Plan, error) {
	epicBody, err := os.ReadFile(opts.EpicBodyPath)
	if err != nil {
		return Plan{}, fmt.Errorf("read epic body: %w", err)
	}

	selected, err := parseEpicIssues(string(epicBody), adapter, opts.Phase)
	if err != nil {
		return Plan{}, err
	}
	if opts.UntilIssue > 0 {
		selected, err = truncateAt(selected, opts.UntilIssue)
		if err != nil {
			return Plan{}, err
		}
	}

	graph := map[int][]int{}
	unblocksMap := map[int][]int{}
	warnings := []string{}

	selectedSet := make(map[int]struct{}, len(selected))
	for _, issue := range selected {
		selectedSet[issue] = struct{}{}
		graph[issue] = nil
	}

	for _, issue := range selected {
		bodyPath := filepath.Join(opts.IssueDir, fmt.Sprintf("%d.md", issue))
		bodyBytes, err := os.ReadFile(bodyPath)
		if err != nil {
			return Plan{}, fmt.Errorf("read issue body %d: %w", issue, err)
		}

		deps, revs, parseWarnings, err := parseDependencies(string(bodyBytes), adapter, selectedSet)
		if err != nil {
			return Plan{}, fmt.Errorf("parse dependencies for issue %d: %w", issue, err)
		}
		warnings = append(warnings, parseWarnings...)
		for _, reverseDependency := range graph[issue] {
			if !containsIssue(deps, reverseDependency) {
				deps = append(deps, reverseDependency)
			}
		}
		graph[issue] = deps
		unblocksMap[issue] = revs

		// Wire reverse edges: if this issue unblocks target, target depends on this issue
		for _, target := range revs {
			if !containsIssue(graph[target], issue) {
				graph[target] = append(graph[target], issue)
			}
		}
	}

	ordered, err := topoSort(selected, graph)
	if err != nil {
		return Plan{}, err
	}

	posMap := make(map[int]int, len(ordered))
	for i, n := range ordered {
		posMap[n] = i
	}

	planned := make([]PlannedIssue, 0, len(ordered))
	for i, issue := range ordered {
		deps := append([]int(nil), graph[issue]...)
		revs := append([]int(nil), unblocksMap[issue]...)
		sort.Ints(deps)
		sort.Ints(revs)
		state := "ready"
		for _, dep := range deps {
			if depPos, ok := posMap[dep]; ok && depPos >= i {
				state = "blocked"
				break
			}
		}
		planned = append(planned, PlannedIssue{
			IssueNumber:     issue,
			Dependencies:    deps,
			Unblocks:        revs,
			DependencyState: state,
		})
	}

	return Plan{
		RunID:          generateRunID(),
		Repo:           root.Repo.ID,
		Adapter:        adapter.Kind,
		SourceEpicPath: opts.EpicBodyPath,
		SelectedIssues: selected,
		OrderedIssues:  planned,
		Warnings:       warnings,
	}, nil
}

func (p Plan) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "sprintctl dry-run\n")
	fmt.Fprintf(&b, "run_id: %s\n", p.RunID)
	fmt.Fprintf(&b, "repo: %s\n", p.Repo)
	fmt.Fprintf(&b, "adapter: %s\n", p.Adapter)
	fmt.Fprintf(&b, "source_epic: %s\n", p.SourceEpicPath)
	fmt.Fprintf(&b, "selected_issue_count: %d\n", len(p.SelectedIssues))
	fmt.Fprintf(&b, "execution_queue:\n")

	for i, issue := range p.OrderedIssues {
		fmt.Fprintf(&b, "  %d. #%d", i+1, issue.IssueNumber)
		if len(issue.Dependencies) > 0 {
			fmt.Fprintf(&b, " depends_on=%s", formatInts(issue.Dependencies))
		}
		if len(issue.Unblocks) > 0 {
			fmt.Fprintf(&b, " unblocks=%s", formatInts(issue.Unblocks))
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(p.Warnings) > 0 {
		fmt.Fprintf(&b, "warnings:\n")
		for _, warning := range p.Warnings {
			fmt.Fprintf(&b, "  - %s\n", warning)
		}
	}

	return b.String()
}

func parseEpicIssues(body string, adapter config.Adapter, phaseSpec string) ([]int, error) {
	scope := body
	if phaseSpec != "" {
		phaseRange, err := parsePhaseRange(phaseSpec)
		if err != nil {
			return nil, err
		}
		phaseBody, err := extractPhaseBody(body, adapter.IssueParsing.PhaseHeaderPattern, phaseRange)
		if err != nil {
			return nil, err
		}
		scope = phaseBody
	}

	var issues []int
	seen := map[int]struct{}{}
	for _, pattern := range adapter.IssueParsing.EpicChildPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compile epic child pattern %q: %w", pattern, err)
		}
		matches := re.FindAllStringSubmatch(scope, -1)
		for _, match := range matches {
			issue, ok, err := captureIssueNumber(re, match, "issue")
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if _, exists := seen[issue]; exists {
				continue
			}
			seen[issue] = struct{}{}
			issues = append(issues, issue)
		}
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("no issue references matched adapter epic child patterns")
	}
	return issues, nil
}

type phaseRange struct {
	Start int
	End   int
}

func parsePhaseRange(spec string) (phaseRange, error) {
	if strings.Contains(spec, "-") {
		parts := strings.SplitN(spec, "-", 2)
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return phaseRange{}, fmt.Errorf("invalid phase range %q", spec)
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return phaseRange{}, fmt.Errorf("invalid phase range %q", spec)
		}
		if end < start {
			return phaseRange{}, fmt.Errorf("invalid phase range %q: end before start", spec)
		}
		return phaseRange{Start: start, End: end}, nil
	}

	value, err := strconv.Atoi(strings.TrimSpace(spec))
	if err != nil {
		return phaseRange{}, fmt.Errorf("invalid phase %q", spec)
	}
	return phaseRange{Start: value, End: value}, nil
}

func extractPhaseBody(body, pattern string, target phaseRange) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("phase filter requested but adapter does not define issue_parsing.phase_header_pattern")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("compile phase header pattern %q: %w", pattern, err)
	}

	lines := strings.Split(body, "\n")
	currentPhase := 0
	inTarget := false
	var out []string

	for _, line := range lines {
		match := re.FindStringSubmatch(line)
		if len(match) > 0 {
			phase, ok, err := captureIssueNumber(re, match, "phase")
			if err != nil {
				return "", err
			}
			if !ok {
				return "", fmt.Errorf("phase header pattern %q did not capture a phase number", pattern)
			}
			currentPhase = phase
			inTarget = currentPhase >= target.Start && currentPhase <= target.End
			continue
		}

		if inTarget {
			out = append(out, line)
		}
	}

	if len(out) == 0 {
		return "", fmt.Errorf("phase %d-%d matched no content", target.Start, target.End)
	}
	return strings.Join(out, "\n"), nil
}

func truncateAt(issues []int, until int) ([]int, error) {
	for i, issue := range issues {
		if issue == until {
			return issues[:i+1], nil
		}
	}
	return nil, fmt.Errorf("until issue #%d not found in selected queue", until)
}

func parseDependencies(body string, adapter config.Adapter, selectedSet map[int]struct{}) ([]int, []int, []string, error) {
	deps := []int{}
	unblocks := []int{}
	warnings := []string{}
	seenDeps := map[int]struct{}{}
	seenUnblocks := map[int]struct{}{}

	for _, pattern := range adapter.IssueParsing.DependencyPatterns.DependsOn {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("compile depends_on pattern %q: %w", pattern, err)
		}
		for _, match := range re.FindAllStringSubmatch(body, -1) {
			issue, ok, err := captureIssueNumber(re, match, "issue")
			if err != nil {
				return nil, nil, nil, err
			}
			if !ok {
				continue
			}
			if _, inScope := selectedSet[issue]; !inScope {
				warnings = append(warnings, fmt.Sprintf("ignoring out-of-scope dependency #%d", issue))
				continue
			}
			if _, exists := seenDeps[issue]; exists {
				continue
			}
			seenDeps[issue] = struct{}{}
			deps = append(deps, issue)
		}
	}

	for _, pattern := range adapter.IssueParsing.DependencyPatterns.Unblocks {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("compile unblocks pattern %q: %w", pattern, err)
		}
		for _, match := range re.FindAllStringSubmatch(body, -1) {
			issue, ok, err := captureIssueNumber(re, match, "issue")
			if err != nil {
				return nil, nil, nil, err
			}
			if !ok {
				continue
			}
			if _, inScope := selectedSet[issue]; !inScope {
				warnings = append(warnings, fmt.Sprintf("ignoring out-of-scope unblocks target #%d", issue))
				continue
			}
			if _, exists := seenUnblocks[issue]; exists {
				continue
			}
			seenUnblocks[issue] = struct{}{}
			unblocks = append(unblocks, issue)
		}
	}

	return deps, unblocks, warnings, nil
}

func topoSort(input []int, graph map[int][]int) ([]int, error) {
	const (
		visiting = 1
		done     = 2
	)

	state := map[int]int{}
	order := []int{}

	var visit func(int) error
	visit = func(node int) error {
		switch state[node] {
		case visiting:
			return fmt.Errorf("dependency cycle detected at issue #%d", node)
		case done:
			return nil
		}

		state[node] = visiting
		for _, dep := range graph[node] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[node] = done
		order = append(order, node)
		return nil
	}

	for _, node := range input {
		if err := visit(node); err != nil {
			return nil, err
		}
	}

	seen := map[int]struct{}{}
	unique := make([]int, 0, len(order))
	for _, issue := range order {
		if _, exists := seen[issue]; exists {
			continue
		}
		seen[issue] = struct{}{}
		unique = append(unique, issue)
	}

	return unique, nil
}

func captureIssueNumber(re *regexp.Regexp, match []string, preferredName string) (int, bool, error) {
	if idx := re.SubexpIndex(preferredName); idx > 0 && idx < len(match) {
		value := strings.TrimSpace(match[idx])
		if value == "" {
			return 0, false, nil
		}
		n, err := strconv.Atoi(value)
		if err != nil {
			return 0, false, fmt.Errorf("parse capture %q=%q: %w", preferredName, value, err)
		}
		return n, true, nil
	}

	for i := 1; i < len(match); i++ {
		value := strings.TrimSpace(match[i])
		if value == "" {
			continue
		}
		n, err := strconv.Atoi(value)
		if err == nil {
			return n, true, nil
		}
	}

	return 0, false, nil
}

func formatInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("#%d", value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func containsIssue(issues []int, target int) bool {
	for _, issue := range issues {
		if issue == target {
			return true
		}
	}
	return false
}

func generateRunID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("run-%s", hex.EncodeToString(b))
}
