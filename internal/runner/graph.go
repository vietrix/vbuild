package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

type graphNode struct {
	Name string `json:"name"`
}

type graphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type graphPayload struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

func (r *Runner) Graph(targets []string, format string, out io.Writer) error {
	if out == nil {
		out = r.log.out
	}
	if len(targets) == 0 {
		targets = r.allTaskNames()
	}
	plan, err := r.buildPlan(targets)
	if err != nil {
		return err
	}
	switch format {
	case "", "dot":
		return writeDOT(plan, out)
	case "json":
		return writeJSON(plan, out)
	default:
		return fmt.Errorf("unsupported graph format: %s", format)
	}
}

func (r *Runner) allTaskNames() []string {
	names := make([]string, 0, len(r.cfg.Tasks))
	for name := range r.cfg.Tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeDOT(plan *taskPlan, out io.Writer) error {
	if _, err := fmt.Fprintln(out, "digraph vbuild {"); err != nil {
		return err
	}
	for name := range plan.tasks {
		if _, err := fmt.Fprintf(out, "  \"%s\";\n", name); err != nil {
			return err
		}
	}
	for task, deps := range plan.deps {
		for _, dep := range deps {
			if _, err := fmt.Fprintf(out, "  \"%s\" -> \"%s\";\n", dep, task); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(out, "}")
	return err
}

func writeJSON(plan *taskPlan, out io.Writer) error {
	nodes := make([]graphNode, 0, len(plan.tasks))
	for name := range plan.tasks {
		nodes = append(nodes, graphNode{Name: name})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	edges := []graphEdge{}
	for task, deps := range plan.deps {
		for _, dep := range deps {
			edges = append(edges, graphEdge{From: dep, To: task})
		}
	}
	payload := graphPayload{Nodes: nodes, Edges: edges}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
