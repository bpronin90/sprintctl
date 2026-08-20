package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/benpronin/sprintctl/internal/config"
	"github.com/benpronin/sprintctl/internal/planner"
)

func main() {
	var (
		configPath   = flag.String("config", "", "path to sprintctl config yaml")
		epicBodyPath = flag.String("epic-body", "", "path to local epic body markdown")
		issueDir     = flag.String("issue-dir", "", "directory containing issue body markdown files named <issue>.md")
		phase        = flag.String("phase", "", "optional phase filter: N or N-M")
		until        = flag.Int("until", 0, "optional queue truncation at issue number (inclusive)")
	)

	flag.Parse()

	if err := run(*configPath, *epicBodyPath, *issueDir, *phase, *until); err != nil {
		fmt.Fprintf(os.Stderr, "sprintctl: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath, epicBodyPath, issueDir, phase string, until int) error {
	if configPath == "" {
		return fmt.Errorf("missing required -config")
	}
	if epicBodyPath == "" {
		return fmt.Errorf("missing required -epic-body")
	}
	if issueDir == "" {
		return fmt.Errorf("missing required -issue-dir")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	adapterCfg, err := config.LoadAdapter(cfg.Adapter.ConfigPath)
	if err != nil {
		return err
	}

	opts := planner.Options{
		EpicBodyPath: epicBodyPath,
		IssueDir:     issueDir,
		Phase:        phase,
		UntilIssue:   until,
	}

	plan, err := planner.Build(cfg, adapterCfg, opts)
	if err != nil {
		return err
	}

	fmt.Print(plan.Render())
	return nil
}
