package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) evaluateCanary(spec *config.CanarySpec, metrics map[string]float64) error {
	if spec == nil {
		return nil
	}
	if spec.Baseline == "" {
		return fmt.Errorf("canary baseline is empty")
	}
	path := r.resolvePath(spec.Baseline)
	format := metricsFormatFromPath(path)
	baseline, err := parseMetricsFile(path, format)
	if err != nil {
		return fmt.Errorf("canary baseline: %w", err)
	}
	violations := []string{}
	for name, rule := range spec.Rules {
		current, ok := metrics[name]
		if !ok {
			if spec.AllowMissing {
				continue
			}
			violations = append(violations, fmt.Sprintf("%s missing", name))
			continue
		}
		base, ok := baseline[name]
		if !ok {
			if spec.AllowMissing {
				continue
			}
			violations = append(violations, fmt.Sprintf("%s baseline missing", name))
			continue
		}
		if rule.Min != nil && current < *rule.Min {
			violations = append(violations, fmt.Sprintf("%s below min (%v < %v)", name, current, *rule.Min))
		}
		if rule.Max != nil && current > *rule.Max {
			violations = append(violations, fmt.Sprintf("%s above max (%v > %v)", name, current, *rule.Max))
		}
		if rule.MaxDelta != nil {
			diff := current - base
			if diff < 0 {
				diff = -diff
			}
			if diff > *rule.MaxDelta {
				violations = append(violations, fmt.Sprintf("%s delta %v > %v", name, diff, *rule.MaxDelta))
			}
		}
		if rule.MaxDeltaPct != nil && base != 0 {
			diff := (current - base) / base * 100
			if diff < 0 {
				diff = -diff
			}
			if diff > *rule.MaxDeltaPct {
				violations = append(violations, fmt.Sprintf("%s delta %0.2f%% > %0.2f%%", name, diff, *rule.MaxDeltaPct))
			}
		}
	}
	if len(violations) > 0 {
		return fmt.Errorf("canary failed: %s", strings.Join(violations, "; "))
	}
	return nil
}

func metricsFormatFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		return "csv"
	case ".txt", ".log":
		return "kv"
	default:
		return "json"
	}
}
