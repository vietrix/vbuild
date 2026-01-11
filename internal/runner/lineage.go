package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type lineageEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Kind      string `json:"kind"`
	Timestamp string `json:"timestamp"`
}

type lineageGraph struct {
	Edges []lineageEdge `json:"edges"`
}

func (r *Runner) recordLineage(taskName string, inputs []datasetEntry, outputs []datasetEntry, experimentID string) error {
	if taskName == "" {
		return nil
	}
	r.registryMu.Lock()
	defer r.registryMu.Unlock()
	graph, _ := r.loadLineage()
	now := time.Now().UTC().Format(time.RFC3339)
	taskNode := "task:" + taskName
	if experimentID != "" {
		graph.Edges = append(graph.Edges, lineageEdge{
			From:      taskNode,
			To:        "experiment:" + experimentID,
			Kind:      "task_experiment",
			Timestamp: now,
		})
	}
	for _, in := range inputs {
		graph.Edges = append(graph.Edges, lineageEdge{
			From:      "dataset:" + in.Name + "@" + in.Version,
			To:        taskNode,
			Kind:      "dataset_input",
			Timestamp: now,
		})
	}
	for _, out := range outputs {
		graph.Edges = append(graph.Edges, lineageEdge{
			From:      taskNode,
			To:        "dataset:" + out.Name + "@" + out.Version,
			Kind:      "dataset_output",
			Timestamp: now,
		})
	}
	return r.saveLineage(graph)
}

func (r *Runner) loadLineage() (lineageGraph, error) {
	graph := lineageGraph{Edges: []lineageEdge{}}
	path := filepath.Join(r.registryRoot(), "lineage.json")
	if err := readJSON(path, &graph); err != nil {
		if os.IsNotExist(err) {
			return graph, nil
		}
		return graph, err
	}
	return graph, nil
}

func (r *Runner) saveLineage(graph lineageGraph) error {
	path := filepath.Join(r.registryRoot(), "lineage.json")
	return writeJSONFile(path, graph)
}

func (r *Runner) Lineage(format string, out io.Writer) error {
	if out == nil {
		out = r.log.out
	}
	graph, err := r.loadLineage()
	if err != nil {
		return err
	}
	switch format {
	case "", "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(graph)
	case "dot":
		nodes := map[string]struct{}{}
		for _, edge := range graph.Edges {
			nodes[edge.From] = struct{}{}
			nodes[edge.To] = struct{}{}
		}
		list := make([]string, 0, len(nodes))
		for node := range nodes {
			list = append(list, node)
		}
		sort.Strings(list)
		if _, err := fmt.Fprintln(out, "digraph vbuild_lineage {"); err != nil {
			return err
		}
		for _, node := range list {
			if _, err := fmt.Fprintf(out, "  \"%s\";\n", node); err != nil {
				return err
			}
		}
		for _, edge := range graph.Edges {
			if _, err := fmt.Fprintf(out, "  \"%s\" -> \"%s\" [label=\"%s\"];\n", edge.From, edge.To, edge.Kind); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(out, "}")
		return err
	default:
		return fmt.Errorf("unsupported lineage format: %s", format)
	}
}
