package runner

import (
	"fmt"
	"runtime"
	"strings"
)

func evalCondition(expr string, vars map[string]string, env map[string]string) (bool, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return true, nil
	}
	normalized := strings.ReplaceAll(trimmed, " and ", " && ")
	normalized = strings.ReplaceAll(normalized, " or ", " || ")

	orParts := splitCondition(normalized, "||")
	for _, part := range orParts {
		andParts := splitCondition(part, "&&")
		ok := true
		for _, term := range andParts {
			result, err := evalTerm(term, vars, env)
			if err != nil {
				return false, err
			}
			if !result {
				ok = false
				break
			}
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func splitCondition(expr, sep string) []string {
	parts := strings.Split(expr, sep)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func evalTerm(term string, vars map[string]string, env map[string]string) (bool, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return false, nil
	}
	negate := false
	if strings.HasPrefix(term, "!") {
		negate = true
		term = strings.TrimSpace(strings.TrimPrefix(term, "!"))
	}

	op := ""
	if strings.Contains(term, "!=") {
		op = "!="
	} else if strings.Contains(term, "==") {
		op = "=="
	}

	var result bool
	if op == "" {
		value, ok := resolveValue(term, vars, env)
		if !ok {
			return false, fmt.Errorf("unknown condition key: %s", term)
		}
		result = value != ""
	} else {
		parts := strings.SplitN(term, op, 2)
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid condition: %s", term)
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		right = trimQuotes(right)
		value, ok := resolveValue(left, vars, env)
		if !ok {
			return false, fmt.Errorf("unknown condition key: %s", left)
		}
		if op == "==" {
			result = value == right
		} else {
			result = value != right
		}
	}

	if negate {
		return !result, nil
	}
	return result, nil
}

func resolveValue(key string, vars map[string]string, env map[string]string) (string, bool) {
	switch {
	case key == "os":
		return runtime.GOOS, true
	case key == "arch":
		return runtime.GOARCH, true
	case strings.HasPrefix(key, "env."):
		return env[strings.TrimPrefix(key, "env.")], true
	case strings.HasPrefix(key, "vars."):
		return vars[strings.TrimPrefix(key, "vars.")], true
	case key == "true":
		return "true", true
	case key == "false":
		return "", true
	default:
		return "", false
	}
}

func trimQuotes(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
