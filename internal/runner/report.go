package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type complianceReport struct {
	Generated   string           `json:"generated"`
	ConfigHash  string           `json:"config_hash,omitempty"`
	Registry    string           `json:"registry_root,omitempty"`
	Datasets    datasetRegistry  `json:"datasets"`
	Experiments experimentIndex  `json:"experiments"`
	Lineage     lineageGraph     `json:"lineage"`
	SBOMFiles   []string         `json:"sbom_files,omitempty"`
	Signatures  []string         `json:"signature_files,omitempty"`
}

func (r *Runner) Report(path string) error {
	report := complianceReport{
		Generated: time.Now().UTC().Format(time.RFC3339),
		Datasets:  datasetRegistry{Datasets: map[string][]datasetEntry{}},
		Experiments: experimentIndex{Experiments: []experimentRecord{}},
		Lineage: lineageGraph{Edges: []lineageEdge{}},
	}
	if r.cfg != nil {
		report.ConfigHash = r.cfg.Hash
	}
	report.Registry = r.registryRoot()
	if datasets, err := r.loadDatasetRegistry(); err == nil {
		report.Datasets = datasets
	}
	if experiments, err := r.loadExperimentIndex(); err == nil {
		report.Experiments = experiments
	}
	if lineage, err := r.loadLineage(); err == nil {
		report.Lineage = lineage
	}
	report.SBOMFiles = findFiles(filepath.Join(r.configRoot, ".vbuild", "sbom"))
	report.Signatures = findFiles(filepath.Join(r.configRoot, ".vbuild", "signatures"))
	if path == "" {
		path = "-"
	}
	if path == "-" {
		return writeJSONTo(os.Stdout, report)
	}
	path = r.resolvePath(path)
	return writeJSONFile(path, report)
}

func findFiles(root string) []string {
	out := []string{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		out = append(out, path)
		return nil
	})
	return out
}

func writeJSONTo(out *os.File, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	if err == nil {
		_, _ = out.Write([]byte("\n"))
	}
	return err
}
