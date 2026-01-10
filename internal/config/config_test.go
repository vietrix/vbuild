package config

import "testing"

func TestValidateEmptyTasks(t *testing.T) {
	cfg := &Config{}
	cfg.normalize()
	if err := cfg.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateUnknownDep(t *testing.T) {
	cfg := &Config{
		Tasks: map[string]*Task{
			"build": {Deps: []string{"missing"}, Run: []string{"echo build"}},
		},
	}
	cfg.normalize()
	if err := cfg.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateNoopTask(t *testing.T) {
	cfg := &Config{
		Tasks: map[string]*Task{
			"noop": {},
		},
	}
	cfg.normalize()
	if err := cfg.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEnvOverridesVars(t *testing.T) {
	t.Setenv("VBUILD_VAR_VERSION", "v9.9.9")
	cfg := &Config{
		Vars: map[string]string{"VERSION": "dev"},
	}
	cfg.normalize()
	cfg.applyEnvOverrides()
	if got := cfg.Vars["VERSION"]; got != "v9.9.9" {
		t.Fatalf("expected env override v9.9.9, got %q", got)
	}
}
