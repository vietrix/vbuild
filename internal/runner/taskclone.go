package runner

import "github.com/vietrix/vbuild/internal/config"

func cloneTask(task *config.Task) *config.Task {
	if task == nil {
		return nil
	}
	out := *task
	out.Deps = append([]string(nil), task.Deps...)
	out.Needs = append([]string(nil), task.Needs...)
	out.DependsOn = append([]config.ConditionalDep(nil), task.DependsOn...)
	out.Run = append([]string(nil), task.Run...)
	out.Pre = append([]string(nil), task.Pre...)
	out.Post = append([]string(nil), task.Post...)
	out.Tags = append([]string(nil), task.Tags...)
	out.Secrets = append([]string(nil), task.Secrets...)
	out.Inputs = append([]string(nil), task.Inputs...)
	out.Outputs = append([]string(nil), task.Outputs...)
	out.OutputPaths = append([]string(nil), task.OutputPaths...)
	out.Watch = append([]string(nil), task.Watch...)
	out.Artifacts = append([]string(nil), task.Artifacts...)
	out.RetryOnExitCodes = append([]int(nil), task.RetryOnExitCodes...)
	out.RetryOnRegex = append([]string(nil), task.RetryOnRegex...)
	out.Fanout = task.Fanout
	out.Isolate = task.Isolate
	out.ContinueOnError = task.ContinueOnError
	out.AllowFailure = task.AllowFailure
	if task.Env != nil {
		out.Env = map[string]string{}
		for k, v := range task.Env {
			out.Env[k] = v
		}
	}
	if task.Vars != nil {
		out.Vars = map[string]string{}
		for k, v := range task.Vars {
			out.Vars[k] = v
		}
	}
	if task.Exports != nil {
		out.Exports = map[string]string{}
		for k, v := range task.Exports {
			out.Exports[k] = v
		}
	}
	if task.With != nil {
		out.With = map[string]string{}
		for k, v := range task.With {
			out.With[k] = v
		}
	}
	if task.Matrix != nil {
		out.Matrix = map[string][]string{}
		for k, v := range task.Matrix {
			out.Matrix[k] = append([]string(nil), v...)
		}
	}
	if task.Limits != nil {
		limits := *task.Limits
		out.Limits = &limits
	}
	if task.Remote != nil {
		remote := *task.Remote
		out.Remote = &remote
	}
	if task.Docker != nil {
		out.Docker = cloneDocker(task.Docker)
	}
	return &out
}

func cloneDocker(spec *config.DockerSpec) *config.DockerSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	if spec.Build != nil {
		build := *spec.Build
		if spec.Build.Args != nil {
			build.Args = map[string]string{}
			for k, v := range spec.Build.Args {
				build.Args[k] = v
			}
		}
		out.Build = &build
	}
	if spec.Push != nil {
		push := *spec.Push
		out.Push = &push
	}
	if spec.Pull != nil {
		pull := *spec.Pull
		out.Pull = &pull
	}
	return &out
}
