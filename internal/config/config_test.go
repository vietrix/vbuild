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
