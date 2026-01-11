package runner

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/vietrix/vbuild/internal/config"
)

func matchOnlyOn(only *config.OnlyOn, gitVars map[string]string) bool {
	if only == nil {
		return true
	}
	branch := strings.TrimSpace(gitVars["GIT_BRANCH"])
	tag := strings.TrimSpace(gitVars["GIT_TAG"])
	branchMatch := matchAnyPattern(only.Branches, branch)
	tagMatch := matchAnyPattern(only.Tags, tag)
	if len(only.Branches) == 0 && len(only.Tags) == 0 {
		return true
	}
	if len(only.Tags) == 0 {
		return branchMatch
	}
	if len(only.Branches) == 0 {
		return tagMatch
	}
	return branchMatch || tagMatch
}

func matchAnyPattern(patterns []string, value string) bool {
	if value == "" || len(patterns) == 0 {
		return false
	}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if ok, _ := doublestar.Match(pattern, value); ok {
			return true
		}
	}
	return false
}
