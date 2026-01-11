package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) prepareSchedulerCommand(command string, sched *config.SchedulerSpec, shell string) (string, func(), error) {
	if sched == nil {
		return command, func() {}, nil
	}
	kind := strings.ToLower(strings.TrimSpace(sched.Type))
	switch kind {
	case "slurm":
		return buildSlurmCommand(command, sched, shell), func() {}, nil
	case "pbs":
		cmd, cleanup, err := r.buildPBSCommand(command, sched, shell)
		return cmd, cleanup, err
	default:
		return "", func() {}, fmt.Errorf("scheduler: unsupported type %s", sched.Type)
	}
}

func buildSlurmCommand(command string, sched *config.SchedulerSpec, shell string) string {
	args := []string{"sbatch", "--wait", "--parsable", "--export=ALL"}
	if sched.Queue != "" {
		args = append(args, "--partition", sched.Queue)
	}
	if sched.Account != "" {
		args = append(args, "--account", sched.Account)
	}
	if sched.Time != "" {
		args = append(args, "--time", sched.Time)
	}
	if sched.Nodes > 0 {
		args = append(args, "--nodes", fmt.Sprintf("%d", sched.Nodes))
	}
	if sched.GPUs > 0 {
		args = append(args, "--gpus", fmt.Sprintf("%d", sched.GPUs))
	}
	if sched.CPUs > 0 {
		args = append(args, "--cpus-per-task", fmt.Sprintf("%d", sched.CPUs))
	}
	if sched.Memory != "" {
		args = append(args, "--mem", sched.Memory)
	}
	args = append(args, sched.Args...)
	quoted := shellQuote(command)
	if shellKind(shell) == "powershell" {
		quoted = powershellQuote(command)
	}
	args = append(args, "--wrap", quoted)
	return strings.Join(args, " ")
}

func (r *Runner) buildPBSCommand(command string, sched *config.SchedulerSpec, shell string) (string, func(), error) {
	tmpDir := filepath.Join(r.configRoot, ".vbuild", "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", func() {}, err
	}
	file, err := os.CreateTemp(tmpDir, "pbs-*.sh")
	if err != nil {
		return "", func() {}, err
	}
	script := "#!/bin/sh\n"
	script += "cd \"$PBS_O_WORKDIR\"\n"
	script += command + "\n"
	if _, err := file.WriteString(script); err != nil {
		_ = file.Close()
		return "", func() {}, err
	}
	_ = file.Close()
	args := []string{"qsub", "-V"}
	if sched.Queue != "" {
		args = append(args, "-q", sched.Queue)
	}
	if sched.Account != "" {
		args = append(args, "-A", sched.Account)
	}
	resources := []string{}
	if sched.Nodes > 0 {
		resources = append(resources, fmt.Sprintf("nodes=%d", sched.Nodes))
	}
	if sched.CPUs > 0 {
		resources = append(resources, fmt.Sprintf("ncpus=%d", sched.CPUs))
	}
	if sched.GPUs > 0 {
		resources = append(resources, fmt.Sprintf("ngpus=%d", sched.GPUs))
	}
	if sched.Memory != "" {
		resources = append(resources, fmt.Sprintf("mem=%s", sched.Memory))
	}
	if sched.Time != "" {
		resources = append(resources, fmt.Sprintf("walltime=%s", sched.Time))
	}
	if len(resources) > 0 {
		args = append(args, "-l", strings.Join(resources, ","))
	}
	args = append(args, sched.Args...)
	args = append(args, file.Name())
	return strings.Join(args, " "), func() { _ = os.Remove(file.Name()) }, nil
}
