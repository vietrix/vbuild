package runner

import (
	"encoding/json"
	"os"
	"sort"
	"time"
)

type summaryPayload struct {
	TotalDuration string        `json:"total_duration"`
	Ok            int           `json:"ok"`
	Failed        int           `json:"failed"`
	Skipped       int           `json:"skipped"`
	UpToDate      int           `json:"up_to_date"`
	Tasks         []summaryTask `json:"tasks"`
}

type summaryTask struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func (r *Runner) writeSummaryJSON(summary summaryPayload) {
	if r.opts.SummaryPath == "" {
		return
	}
	var out *os.File
	if r.opts.SummaryPath == "-" {
		out = os.Stdout
	} else {
		file, err := os.OpenFile(r.resolvePath(r.opts.SummaryPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			r.log.Errorf("summary: %v\n", err)
			return
		}
		defer file.Close()
		out = file
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		r.log.Errorf("summary: %v\n", err)
	}
}

func buildSummary(plan *taskPlan, results map[string]taskResult, total time.Duration) summaryPayload {
	okCount := 0
	failCount := 0
	skipCount := 0
	upToDateCount := 0

	tasks := make([]summaryTask, 0, len(plan.order))
	for _, name := range plan.order {
		result, ok := results[name]
		if !ok {
			skipCount++
			tasks = append(tasks, summaryTask{Name: name, Status: string(statusSkipped)})
			continue
		}
		entry := summaryTask{
			Name:     name,
			Status:   string(result.status),
			Duration: formatDuration(result.duration),
		}
		if result.reason != "" {
			entry.Reason = result.reason
		}
		switch result.status {
		case statusFailed:
			failCount++
		case statusUpToDate:
			upToDateCount++
		case statusSkipped, statusCanceled:
			skipCount++
		default:
			okCount++
		}
		tasks = append(tasks, entry)
	}

	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Name < tasks[j].Name
	})

	return summaryPayload{
		TotalDuration: formatDuration(total),
		Ok:            okCount,
		Failed:        failCount,
		Skipped:       skipCount,
		UpToDate:      upToDateCount,
		Tasks:         tasks,
	}
}
