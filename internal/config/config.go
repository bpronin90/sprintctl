package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Root struct {
	Version   string        `yaml:"version"`
	Repo      Repo          `yaml:"repo"`
	GitHub    GitHub        `yaml:"github"`
	Execution Execution     `yaml:"execution"`
	Adapter   AdapterRef    `yaml:"adapter"`
	Comments  Comments      `yaml:"comments"`
	Logging   Logging       `yaml:"logging"`
}

type Repo struct {
	ID            string `yaml:"id"`
	Owner         string `yaml:"owner"`
	Name          string `yaml:"name"`
	DefaultBranch string `yaml:"default_branch"`
}

type GitHub struct {
	Host       string `yaml:"host"`
	APIBaseURL string `yaml:"api_base_url"`
}

type Execution struct {
	WorktreeRoot      string `yaml:"worktree_root"`
	DefaultBaseBranch string `yaml:"default_base_branch"`
	MaxFixupPasses    int    `yaml:"max_fixup_passes"`
	ReviewersRequired int    `yaml:"reviewers_required"`
}

type AdapterRef struct {
	Kind       string `yaml:"kind"`
	ConfigPath string `yaml:"config_path"`
}

type Comments struct {
	TagPrefix      string `yaml:"tag_prefix"`
	AllowSupersede bool   `yaml:"allow_supersede"`
}

type Logging struct {
	RunLogDir string `yaml:"run_log_dir"`
	EmitJSON  bool   `yaml:"emit_json"`
}

type Adapter struct {
	Version      string       `yaml:"version"`
	Kind         string       `yaml:"kind"`
	IssueParsing IssueParsing `yaml:"issue_parsing"`
}

type IssueParsing struct {
	PhaseHeaderPattern string             `yaml:"phase_header_pattern"`
	EpicChildPatterns  []string           `yaml:"epic_child_patterns"`
	DependencyPatterns DependencyPatterns `yaml:"dependency_patterns"`
}

type DependencyPatterns struct {
	DependsOn []string `yaml:"depends_on"`
	Unblocks  []string `yaml:"unblocks"`
}

func Load(path string) (Root, error) {
	var cfg Root
	if err := loadYAML(path, &cfg); err != nil {
		return Root{}, fmt.Errorf("load root config %q: %w", path, err)
	}
	if cfg.Version == "" {
		return Root{}, fmt.Errorf("root config missing version")
	}
	if cfg.Adapter.ConfigPath == "" {
		return Root{}, fmt.Errorf("root config missing adapter.config_path")
	}
	return cfg, nil
}

func LoadAdapter(path string) (Adapter, error) {
	var cfg Adapter
	if err := loadYAML(path, &cfg); err != nil {
		return Adapter{}, fmt.Errorf("load adapter config %q: %w", path, err)
	}
	if cfg.Kind == "" {
		return Adapter{}, fmt.Errorf("adapter config missing kind")
	}
	if len(cfg.IssueParsing.EpicChildPatterns) == 0 {
		return Adapter{}, fmt.Errorf("adapter config missing issue_parsing.epic_child_patterns")
	}
	return cfg, nil
}

func loadYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}
