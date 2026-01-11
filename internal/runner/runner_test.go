package runner

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/vietrix/vbuild/internal/config"
)

func TestBuildPlanOrder(t *testing.T) {
	cfg := &config.Config{
		Tasks: map[string]*config.Task{
			"default": {Deps: []string{"build", "test"}},
			"build":   {Deps: []string{"fmt"}},
			"test":    {},
			"fmt":     {},
		},
	}

	r := New(cfg, Options{DryRun: true}, io.Discard, io.Discard)
	plan, err := r.buildPlan([]string{"default"})
	if err != nil {
		t.Fatalf("buildPlan error: %v", err)
	}

	expected := []string{"fmt", "build", "test", "default"}
	if !reflect.DeepEqual(plan.order, expected) {
		t.Fatalf("expected order %v, got %v", expected, plan.order)
	}
}

func TestBuildPlanCycle(t *testing.T) {
	cfg := &config.Config{
		Tasks: map[string]*config.Task{
			"a": {Deps: []string{"b"}},
			"b": {Deps: []string{"a"}},
		},
	}

	r := New(cfg, Options{DryRun: true}, io.Discard, io.Discard)
	if _, err := r.buildPlan([]string{"a"}); err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestExpandVars(t *testing.T) {
	vars := map[string]string{"NAME": "vbuild"}
	got := expandVars("echo {{NAME}} {{MISSING}}", vars)
	expected := "echo vbuild {{MISSING}}"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestMergeEnv(t *testing.T) {
	base := []string{"A=1", "B=2"}
	merged := mergeEnv(base, map[string]string{"B": "3", "C": "4"})

	asMap := map[string]string{}
	for _, entry := range merged {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			asMap[parts[0]] = parts[1]
		}
	}

	expected := map[string]string{"A": "1", "B": "3", "C": "4"}
	if !reflect.DeepEqual(asMap, expected) {
		t.Fatalf("expected %v, got %v", expected, asMap)
	}
}
