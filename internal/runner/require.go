package runner

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type requirement struct {
	name    string
	op      string
	version string
}

func (r *Runner) checkRequirements(values []string) error {
	for _, entry := range values {
		req, err := parseRequirement(entry)
		if err != nil {
			return err
		}
		if err := checkRequirement(req); err != nil {
			return err
		}
	}
	return nil
}

func parseRequirement(value string) (requirement, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return requirement{}, fmt.Errorf("require entry is empty")
	}
	ops := []string{">=", "<=", "==", "!=", ">", "<", "@"}
	for _, op := range ops {
		if idx := strings.Index(value, op); idx > 0 {
			name := strings.TrimSpace(value[:idx])
			ver := strings.TrimSpace(value[idx+len(op):])
			if name == "" || ver == "" {
				return requirement{}, fmt.Errorf("require entry invalid: %s", value)
			}
			if op == "@" {
				op = ">="
			}
			return requirement{name: name, op: op, version: ver}, nil
		}
	}
	return requirement{name: value}, nil
}

func checkRequirement(req requirement) error {
	if req.name == "" {
		return errors.New("requirement name is empty")
	}
	if _, err := exec.LookPath(req.name); err != nil {
		return fmt.Errorf("require %s: not found in PATH", req.name)
	}
	if req.version == "" {
		return nil
	}
	found, err := readBinaryVersion(req.name)
	if err != nil {
		return fmt.Errorf("require %s: %w", req.name, err)
	}
	if !compareVersion(found, req.op, req.version) {
		return fmt.Errorf("require %s: version %s does not satisfy %s%s", req.name, found, req.op, req.version)
	}
	return nil
}

var versionPattern = regexp.MustCompile(`v?(\d+)\.(\d+)(?:\.(\d+))?`)

func readBinaryVersion(name string) (string, error) {
	output, err := exec.Command(name, "--version").CombinedOutput()
	if err != nil {
		output, err = exec.Command(name, "version").CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("version command failed")
		}
	}
	match := versionPattern.FindStringSubmatch(string(output))
	if len(match) == 0 {
		return "", fmt.Errorf("unable to parse version output")
	}
	parts := []string{match[1], match[2]}
	if match[3] != "" {
		parts = append(parts, match[3])
	} else {
		parts = append(parts, "0")
	}
	return strings.Join(parts, "."), nil
}

func compareVersion(found string, op string, required string) bool {
	left, ok := parseSemver(found)
	if !ok {
		return false
	}
	right, ok := parseSemver(required)
	if !ok {
		return false
	}
	cmp := left.compare(right)
	switch op {
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case "==":
		return cmp == 0
	case "!=":
		return cmp != 0
	default:
		return false
	}
}

type semver struct {
	major int
	minor int
	patch int
}

func parseSemver(value string) (semver, bool) {
	match := versionPattern.FindStringSubmatch(value)
	if len(match) == 0 {
		return semver{}, false
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch := 0
	if match[3] != "" {
		patch, _ = strconv.Atoi(match[3])
	}
	return semver{major: major, minor: minor, patch: patch}, true
}

func (s semver) compare(other semver) int {
	if s.major != other.major {
		if s.major > other.major {
			return 1
		}
		return -1
	}
	if s.minor != other.minor {
		if s.minor > other.minor {
			return 1
		}
		return -1
	}
	if s.patch != other.patch {
		if s.patch > other.patch {
			return 1
		}
		return -1
	}
	return 0
}
