package planner

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/benpronin/sprintctl/internal/config"
)

func testAdapter() config.Adapter {
	return config.Adapter{
		Kind: "test",
		IssueParsing: config.IssueParsing{
			PhaseHeaderPattern: `(?m)^## Phase (?P<phase>\d+)$`,
			EpicChildPatterns:  []string{`(?m)^- \[ \] #(?P<issue>\d+)`},
			DependencyPatterns: config.DependencyPatterns{
				DependsOn: []string{`(?m)^depends-on: #(?P<issue>\d+)$`},
				Unblocks:  []string{`(?m)^unblocks: #(?P<issue>\d+)$`},
			},
		},
	}
}

func TestParseEpicIssuesFiltersPhaseAndDeduplicates(t *testing.T) {
	body := "## Phase 1\n- [ ] #11 first\n- [ ] #11 duplicate\n## Phase 2\n- [ ] #12 second\n"

	issues, err := parseEpicIssues(body, testAdapter(), "1")
	if err != nil {
		t.Fatalf("parseEpicIssues() error = %v", err)
	}
	if want := []int{11}; !reflect.DeepEqual(issues, want) {
		t.Fatalf("parseEpicIssues() = %v, want %v", issues, want)
	}
}

func TestTopoSortOrdersDependenciesAndDetectsCycles(t *testing.T) {
	ordered, err := topoSort([]int{12, 11, 13}, map[int][]int{11: nil, 12: {11}, 13: {12}})
	if err != nil {
		t.Fatalf("topoSort() error = %v", err)
	}
	if want := []int{11, 12, 13}; !reflect.DeepEqual(ordered, want) {
		t.Fatalf("topoSort() = %v, want %v", ordered, want)
	}

	_, err = topoSort([]int{11, 12}, map[int][]int{11: {12}, 12: {11}})
	if err == nil || !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("topoSort() cycle error = %v", err)
	}
}

func TestBuildAppliesUnblocksAsReverseDependency(t *testing.T) {
	dir := t.TempDir()
	epicPath := filepath.Join(dir, "epic.md")
	issueDir := filepath.Join(dir, "issues")
	if err := os.Mkdir(issueDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, epicPath, "- [ ] #11 first\n- [ ] #12 second\n")
	writeTestFile(t, filepath.Join(issueDir, "11.md"), "unblocks: #12\n")
	writeTestFile(t, filepath.Join(issueDir, "12.md"), "body\n")

	plan, err := Build(config.Root{Repo: config.Repo{ID: "owner/repo"}}, testAdapter(), Options{
		EpicBodyPath: epicPath,
		IssueDir:     issueDir,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got := plan.OrderedIssues[1].Dependencies; !reflect.DeepEqual(got, []int{11}) {
		t.Fatalf("issue #12 dependencies = %v, want [11]", got)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
