package runner

import (
	"io"
	"testing"

	"github.com/vietrix/vbuild/internal/config"
)

func TestBuildScriptCommandPassThrough(t *testing.T) {
	cfg := &config.Config{
		Tasks: map[string]*config.Task{
			"train": {Script: "python train.py", Shell: "sh"},
		},
	}
	r := New(cfg, Options{ArgsTarget: "train", Args: []string{"--seed=1", "data"}}, io.Discard, io.Discard)
	cmd, missing := r.buildScriptCommand("train", cfg.Tasks["train"])
	if len(missing) > 0 {
		t.Fatalf("unexpected missing args: %v", missing)
	}
	expected := "python train.py '--seed=1' 'data'"
	if cmd != expected {
		t.Fatalf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildScriptCommandPlaceholders(t *testing.T) {
	cfg := &config.Config{
		Tasks: map[string]*config.Task{
			"train": {Script: "python train.py {dir} {out}", Shell: "sh"},
		},
	}
	args := []string{"--dir", "data", "--out=dist", "--seed", "1"}
	r := New(cfg, Options{ArgsTarget: "train", Args: args}, io.Discard, io.Discard)
	cmd, missing := r.buildScriptCommand("train", cfg.Tasks["train"])
	if len(missing) > 0 {
		t.Fatalf("unexpected missing args: %v", missing)
	}
	expected := "python train.py 'data' 'dist' '--seed' '1'"
	if cmd != expected {
		t.Fatalf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildScriptCommandMissingArgs(t *testing.T) {
	cfg := &config.Config{
		Tasks: map[string]*config.Task{
			"train": {Script: "python train.py {dir} {out}", Shell: "sh"},
		},
	}
	r := New(cfg, Options{ArgsTarget: "train"}, io.Discard, io.Discard)
	_, missing := r.buildScriptCommand("train", cfg.Tasks["train"])
	if len(missing) == 0 {
		t.Fatal("expected missing args")
	}
}
