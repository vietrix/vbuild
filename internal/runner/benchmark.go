package runner

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type benchmarkResult struct {
	Iterations int      `json:"iterations"`
	Warmup     int      `json:"warmup"`
	Durations  []string `json:"durations"`
	Min        string   `json:"min"`
	Max        string   `json:"max"`
	Mean       string   `json:"mean"`
	P50        string   `json:"p50"`
	P95        string   `json:"p95"`
}

func (r *Runner) runBenchmark(ctx context.Context, taskName string, commands []string, opts commandOptions, task *config.Task) (*benchmarkResult, error) {
	if task == nil || task.Benchmark == nil {
		return nil, nil
	}
	spec := task.Benchmark
	if spec.Iterations <= 0 {
		return nil, nil
	}
	durations := []time.Duration{}
	for i := 0; i < spec.Warmup; i++ {
		if opts.remote != nil && len(opts.remote.Hosts) > 0 {
			if err := r.runRemoteFanout(ctx, taskName, commands, opts, task); err != nil {
				return nil, err
			}
		} else if task.Parallel {
			if err := r.runParallel(ctx, taskName, commands, opts); err != nil {
				return nil, err
			}
		} else {
			if err := r.runSequential(ctx, taskName, commands, opts); err != nil {
				return nil, err
			}
		}
	}
	for i := 0; i < spec.Iterations; i++ {
		start := time.Now()
		if opts.remote != nil && len(opts.remote.Hosts) > 0 {
			if err := r.runRemoteFanout(ctx, taskName, commands, opts, task); err != nil {
				return nil, err
			}
		} else if task.Parallel {
			if err := r.runParallel(ctx, taskName, commands, opts); err != nil {
				return nil, err
			}
		} else {
			if err := r.runSequential(ctx, taskName, commands, opts); err != nil {
				return nil, err
			}
		}
		durations = append(durations, time.Since(start))
	}
	result := summarizeBenchmark(durations, spec.Iterations, spec.Warmup)
	if spec.Output != "" {
		path := r.resolvePath(spec.Output)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return result, err
		}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return result, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return result, err
		}
	}
	return result, nil
}

func summarizeBenchmark(durations []time.Duration, iterations, warmup int) *benchmarkResult {
	if len(durations) == 0 {
		return &benchmarkResult{Iterations: iterations, Warmup: warmup}
	}
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	min := sorted[0]
	max := sorted[len(sorted)-1]
	mean := time.Duration(0)
	for _, d := range durations {
		mean += d
	}
	mean = time.Duration(float64(mean) / float64(len(durations)))
	p50 := percentile(sorted, 0.5)
	p95 := percentile(sorted, 0.95)
	out := &benchmarkResult{
		Iterations: iterations,
		Warmup:     warmup,
		Durations:  formatDurations(durations),
		Min:        formatDuration(min),
		Max:        formatDuration(max),
		Mean:       formatDuration(mean),
		P50:        formatDuration(p50),
		P95:        formatDuration(p95),
	}
	return out
}

func percentile(sorted []time.Duration, pct float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if pct <= 0 {
		return sorted[0]
	}
	if pct >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := pct * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	weight := pos - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

func formatDurations(values []time.Duration) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, formatDuration(value))
	}
	return out
}
