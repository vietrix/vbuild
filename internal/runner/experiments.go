package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type experimentRecord struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Task      string             `json:"task"`
	Status    string             `json:"status"`
	Start     string             `json:"start"`
	End       string             `json:"end"`
	Duration  string             `json:"duration"`
	Seed      int64              `json:"seed,omitempty"`
	RunDir    string             `json:"run_dir,omitempty"`
	Tags      []string           `json:"tags,omitempty"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
	Vars      map[string]string  `json:"vars,omitempty"`
	Env       map[string]string  `json:"env,omitempty"`
	Datasets  []datasetEntry     `json:"datasets,omitempty"`
	Outputs   map[string]string  `json:"outputs,omitempty"`
	Metrics   map[string]float64 `json:"metrics,omitempty"`
	Benchmark *benchmarkResult   `json:"benchmark,omitempty"`
}

type experimentIndex struct {
	Experiments []experimentRecord `json:"experiments"`
}

type experimentRun struct {
	record experimentRecord
}

func (r *Runner) experimentsRoot() string {
	dir := ".vbuild/experiments"
	if r.cfg != nil && r.cfg.Experiments != nil && r.cfg.Experiments.Dir != "" {
		dir = r.cfg.Experiments.Dir
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(r.configRoot, dir)
}

func (r *Runner) experimentEnabled(task *config.Task) bool {
	enabled := false
	if r.cfg != nil && r.cfg.Experiments != nil && r.cfg.Experiments.Enabled {
		enabled = true
	}
	if task != nil && task.Experiment != nil {
		enabled = true
		if task.Experiment.Record != nil {
			enabled = *task.Experiment.Record
		}
	}
	return enabled
}

func (r *Runner) startExperiment(taskName string, task *config.Task, vars, env map[string]string, seed int64, runDir string) *experimentRun {
	if !r.experimentEnabled(task) {
		return nil
	}
	id := fmt.Sprintf("%s-%d", sanitizePath(taskName), time.Now().UnixNano())
	name := taskName
	if task != nil && task.Experiment != nil && task.Experiment.Name != "" {
		name = task.Experiment.Name
	}
	record := experimentRecord{
		ID:       id,
		Name:     name,
		Task:     taskName,
		Status:   "running",
		Start:    time.Now().UTC().Format(time.RFC3339),
		Seed:     seed,
		RunDir:   runDir,
		Tags:     r.mergeExperimentTags(task),
		Metadata: r.mergeExperimentMetadata(task),
		Vars:     cloneStringMap(vars),
		Env:      cloneStringMap(env),
	}
	return &experimentRun{record: record}
}

func (r *Runner) finishExperiment(run *experimentRun, status string, duration time.Duration, datasets []datasetEntry, outputs map[string]string, metrics map[string]float64, benchmark *benchmarkResult) error {
	if run == nil {
		return nil
	}
	record := run.record
	record.Status = status
	record.End = time.Now().UTC().Format(time.RFC3339)
	record.Duration = formatDuration(duration)
	record.Datasets = datasets
	record.Outputs = outputs
	record.Metrics = metrics
	record.Benchmark = benchmark

	r.registryMu.Lock()
	defer r.registryMu.Unlock()
	if err := writeJSONFile(filepath.Join(r.experimentsRoot(), record.ID+".json"), record); err != nil {
		return err
	}
	index, _ := r.loadExperimentIndex()
	index.Experiments = append(index.Experiments, record)
	sort.Slice(index.Experiments, func(i, j int) bool {
		return index.Experiments[i].Start > index.Experiments[j].Start
	})
	return r.saveExperimentIndex(index)
}

func (r *Runner) loadExperimentIndex() (experimentIndex, error) {
	index := experimentIndex{Experiments: []experimentRecord{}}
	path := filepath.Join(r.experimentsRoot(), "index.json")
	if err := readJSON(path, &index); err != nil {
		if os.IsNotExist(err) {
			return index, nil
		}
		return index, err
	}
	return index, nil
}

func (r *Runner) saveExperimentIndex(index experimentIndex) error {
	path := filepath.Join(r.experimentsRoot(), "index.json")
	return writeJSONFile(path, index)
}

func (r *Runner) mergeExperimentTags(task *config.Task) []string {
	tags := []string{}
	if r.cfg != nil && r.cfg.Experiments != nil {
		tags = append(tags, r.cfg.Experiments.Tags...)
	}
	if task != nil && task.Experiment != nil {
		tags = append(tags, task.Experiment.Tags...)
	}
	return dedupeNonEmpty(tags)
}

func (r *Runner) mergeExperimentMetadata(task *config.Task) map[string]string {
	meta := map[string]string{}
	if r.cfg != nil && r.cfg.Experiments != nil {
		for k, v := range r.cfg.Experiments.Metadata {
			meta[k] = v
		}
	}
	if task != nil && task.Experiment != nil {
		for k, v := range task.Experiment.Metadata {
			meta[k] = v
		}
	}
	return meta
}

func (r *Runner) ListExperiments() ([]experimentRecord, error) {
	r.registryMu.Lock()
	defer r.registryMu.Unlock()
	index, err := r.loadExperimentIndex()
	if err != nil {
		return nil, err
	}
	return append([]experimentRecord(nil), index.Experiments...), nil
}

func (r *Runner) GetExperiment(id string) (experimentRecord, error) {
	path := filepath.Join(r.experimentsRoot(), id+".json")
	var record experimentRecord
	if err := readJSON(path, &record); err != nil {
		return experimentRecord{}, err
	}
	return record, nil
}
