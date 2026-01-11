package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (r *Runner) buildBaseEnv() []string {
	env := os.Environ()
	path := r.envFilePath()
	if path == "" {
		return env
	}
	values, err := parseEnvFile(path)
	if err != nil {
		r.log.Errorf("failed to load env file %s: %v\n", path, err)
		return env
	}
	return mergeEnv(env, values)
}

func (r *Runner) envFilePath() string {
	if r.opts.EnvFile != "" {
		return r.resolvePath(r.opts.EnvFile)
	}
	if r.cfg != nil && r.cfg.EnvFile != "" {
		return r.resolvePath(r.cfg.EnvFile)
	}
	defaultPath := r.resolvePath(".env")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	return ""
}

func (r *Runner) resolvePath(value string) string {
	return r.resolvePathWithRoot(r.configRoot, value)
}

func (r *Runner) resolvePathWithRoot(root, value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) || isURL(value) {
		return value
	}
	return filepath.Join(root, value)
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env line: %s", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid env line: %s", line)
		}
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func envListToMap(values []string) map[string]string {
	out := map[string]string{}
	for _, entry := range values {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

func envMapToList(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}

func isURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
