package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func dockerCommands(spec *config.DockerSpec, vars map[string]string) []string {
	if spec == nil {
		return nil
	}
	cmds := []string{}
	if spec.Build != nil {
		cmds = append(cmds, dockerBuildCommand(spec.Build, vars))
	}
	if spec.Pull != nil && spec.Pull.Tag != "" {
		tag := expandVars(spec.Pull.Tag, vars)
		cmds = append(cmds, fmt.Sprintf("docker pull %s", tag))
	}
	if spec.Push != nil && spec.Push.Tag != "" {
		tag := expandVars(spec.Push.Tag, vars)
		cmds = append(cmds, fmt.Sprintf("docker push %s", tag))
	}
	return cmds
}

func dockerBuildCommand(build *config.DockerBuild, vars map[string]string) string {
	context := build.Context
	if strings.TrimSpace(context) == "" {
		context = "."
	}
	context = expandVars(context, vars)

	args := []string{"docker", "build"}
	if build.Dockerfile != "" {
		args = append(args, "-f", expandVars(build.Dockerfile, vars))
	}
	if build.Platform != "" {
		args = append(args, "--platform", expandVars(build.Platform, vars))
	}
	if build.Target != "" {
		args = append(args, "--target", expandVars(build.Target, vars))
	}
	if build.Tag != "" {
		args = append(args, "-t", expandVars(build.Tag, vars))
	}
	if len(build.Args) > 0 {
		keys := make([]string, 0, len(build.Args))
		for key := range build.Args {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			val := expandVars(build.Args[key], vars)
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, val))
		}
	}
	args = append(args, context)
	return strings.Join(args, " ")
}
