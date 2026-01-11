package runner

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/vietrix/vbuild/internal/config"
)

type metricsCollector struct {
	mu     sync.Mutex
	regex  []*regexp.Regexp
	prefix string
	values map[string]float64
}

func newMetricsCollector(spec *config.MetricsSpec) (*metricsCollector, error) {
	if spec == nil || len(spec.Regex) == 0 {
		return nil, nil
	}
	regex := make([]*regexp.Regexp, 0, len(spec.Regex))
	for _, pattern := range spec.Regex {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("metrics regex: %w", err)
		}
		regex = append(regex, compiled)
	}
	return &metricsCollector{
		regex:  regex,
		prefix: spec.Prefix,
		values: map[string]float64{},
	}, nil
}

func (c *metricsCollector) consume(output string) {
	if c == nil || len(c.regex) == 0 || output == "" {
		return
	}
	for _, re := range c.regex {
		matches := re.FindAllStringSubmatch(output, -1)
		if len(matches) == 0 {
			continue
		}
		names := re.SubexpNames()
		for _, match := range matches {
			name, value := resolveMetricMatch(names, match)
			if name == "" || value == "" {
				continue
			}
			if c.prefix != "" {
				name = c.prefix + "." + name
			}
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				c.mu.Lock()
				c.values[name] = parsed
				c.mu.Unlock()
			}
		}
	}
}

func (c *metricsCollector) valuesMap() map[string]float64 {
	if c == nil {
		return map[string]float64{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]float64{}
	for k, v := range c.values {
		out[k] = v
	}
	return out
}

func (r *Runner) finalizeMetrics(task *config.Task, vars map[string]string, collector *metricsCollector) (map[string]float64, error) {
	values := map[string]float64{}
	if collector != nil {
		for k, v := range collector.valuesMap() {
			values[k] = v
		}
	}
	if task == nil || task.Metrics == nil || task.Metrics.File == "" {
		return values, nil
	}
	path := r.resolvePath(expandVars(task.Metrics.File, vars))
	if _, err := os.Stat(path); err == nil {
		parsed, err := parseMetricsFile(path, task.Metrics.Format)
		if err != nil {
			return values, err
		}
		for k, v := range parsed {
			values[k] = v
		}
		return values, nil
	}
	if len(values) > 0 {
		if err := writeMetricsFile(path, task.Metrics.Format, values); err != nil {
			return values, err
		}
	}
	return values, nil
}

func resolveMetricMatch(names []string, match []string) (string, string) {
	if len(match) == 0 {
		return "", ""
	}
	var name, value string
	if len(names) > 0 {
		for i, n := range names {
			if n == "name" && i < len(match) {
				name = match[i]
			}
			if n == "value" && i < len(match) {
				value = match[i]
			}
		}
		if name != "" && value != "" {
			return name, value
		}
	}
	if len(match) >= 3 {
		return match[1], match[2]
	}
	if len(match) >= 2 {
		return "metric", match[1]
	}
	return "", ""
}

func parseMetricsFile(path, format string) (map[string]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "json"
	}
	switch format {
	case "json":
		out := map[string]float64{}
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	case "csv":
		reader := csv.NewReader(strings.NewReader(string(data)))
		records, err := reader.ReadAll()
		if err != nil || len(records) < 2 {
			return nil, fmt.Errorf("metrics csv invalid")
		}
		headers := records[0]
		values := records[1]
		out := map[string]float64{}
		for i, key := range headers {
			if i >= len(values) {
				continue
			}
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(values[i]), 64); err == nil {
				out[strings.TrimSpace(key)] = parsed
			}
		}
		return out, nil
	case "kv":
		out := map[string]float64{}
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				parts = strings.SplitN(line, ":", 2)
			}
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				out[key] = parsed
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported metrics format: %s", format)
	}
}

func writeMetricsFile(path, format string, metrics map[string]float64) error {
	if len(metrics) == 0 {
		return nil
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "json"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	switch format {
	case "json":
		data, err := json.MarshalIndent(metrics, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0o644)
	case "kv":
		var builder strings.Builder
		keys := make([]string, 0, len(metrics))
		for key := range metrics {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			builder.WriteString(fmt.Sprintf("%s=%v\n", key, metrics[key]))
		}
		return os.WriteFile(path, []byte(builder.String()), 0o644)
	case "csv":
		keys := make([]string, 0, len(metrics))
		for key := range metrics {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var builder strings.Builder
		builder.WriteString(strings.Join(keys, ",") + "\n")
		values := make([]string, 0, len(keys))
		for _, key := range keys {
			values = append(values, fmt.Sprintf("%v", metrics[key]))
		}
		builder.WriteString(strings.Join(values, ",") + "\n")
		return os.WriteFile(path, []byte(builder.String()), 0o644)
	default:
		return fmt.Errorf("unsupported metrics format: %s", format)
	}
}

 
