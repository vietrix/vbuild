package config

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Workflow string            `yaml:"workflow"`
	Vars     map[string]string `yaml:"vars"`
	Env      map[string]string `yaml:"env"`
	Tasks    map[string]*Task  `yaml:"tasks"`
}

type Task struct {
	Desc     string            `yaml:"desc"`
	Deps     []string          `yaml:"deps"`
	Run      []string          `yaml:"run"`
	Env      map[string]string `yaml:"env"`
	Parallel bool              `yaml:"parallel"`
}

type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	var builder strings.Builder
	builder.WriteString("config validation failed:")
	for _, issue := range e.Issues {
		builder.WriteString("\n- ")
		builder.WriteString(issue)
	}
	return builder.String()
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	cfg.normalize()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) normalize() {
	if c.Tasks == nil {
		c.Tasks = map[string]*Task{}
	}
	if c.Env == nil {
		c.Env = map[string]string{}
	}
	if c.Vars == nil {
		c.Vars = map[string]string{}
	}
}

func (c *Config) validate() error {
	issues := []string{}

	validateKeyMap("vars", c.Vars, &issues)
	validateKeyMap("env", c.Env, &issues)

	if len(c.Tasks) == 0 {
		issues = append(issues, "tasks must not be empty")
	}

	taskNames := make([]string, 0, len(c.Tasks))
	for name := range c.Tasks {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	for _, name := range taskNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			issues = append(issues, "task name must not be empty")
			continue
		}
		task := c.Tasks[name]
		if task == nil {
			issues = append(issues, fmt.Sprintf("tasks.%s must be an object", name))
			continue
		}

		if len(task.Deps) == 0 && len(task.Run) == 0 {
			issues = append(issues, fmt.Sprintf("tasks.%s must define deps or run", name))
		}

		for i, dep := range task.Deps {
			depName := strings.TrimSpace(dep)
			if depName == "" {
				issues = append(issues, fmt.Sprintf("tasks.%s.deps[%d] must not be empty", name, i))
				continue
			}
			if _, ok := c.Tasks[depName]; !ok {
				issues = append(issues, fmt.Sprintf("tasks.%s.deps[%d] refers to unknown task %q", name, i, depName))
			}
		}

		for i, cmd := range task.Run {
			if strings.TrimSpace(cmd) == "" {
				issues = append(issues, fmt.Sprintf("tasks.%s.run[%d] must not be empty", name, i))
			}
		}

		validateKeyMap(fmt.Sprintf("tasks.%s.env", name), task.Env, &issues)
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

func validateKeyMap(path string, values map[string]string, issues *[]string) {
	if values == nil {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			*issues = append(*issues, fmt.Sprintf("%s key must not be empty", path))
		}
	}
}
