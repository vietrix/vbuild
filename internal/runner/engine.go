package runner

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"sync"

	"github.com/vietrix/vbuild/internal/config"
)

type Runner struct {
	cfg        *config.Config
	opts       Options
	log        *logger
	baseEnv    []string
	secrets    []string
	configRoot string
	exportsMu  sync.Mutex
	exports    map[string]string
	trace      *traceRecorder
	remote     remoteCache
	logPlugins []logPlugin
	gitVars    map[string]string
}

func New(cfg *config.Config, opts Options, stdout, stderr io.Writer) *Runner {
	log := newLogger(stdout, stderr, opts.LogLevel, opts.JSON)
	log.SetFormat(opts.Timestamp, opts.Color)
	root := "."
	if cfg != nil && cfg.Path != "" {
		root = filepath.Dir(cfg.Path)
	}
	r := &Runner{cfg: cfg, opts: opts, log: log, configRoot: root, exports: map[string]string{}}
	r.baseEnv = r.buildBaseEnv()
	r.secrets = r.resolveSecrets(cfg.Secrets, r.baseEnv)
	r.log.SetSecrets(r.secrets)
	r.gitVars = gitMetadata(root)
	r.trace = newTraceRecorder(root, opts.TimelinePath)
	r.remote = newRemoteCache(cfg.CacheRemote, root, log)
	r.logPlugins = r.startLogPlugins(cfg.LogPlugins)
	return r
}

func (r *Runner) ListTasks() {
	names := make([]string, 0, len(r.cfg.Tasks))
	for name := range r.cfg.Tasks {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		desc := ""
		if r.cfg.Tasks[name] != nil {
			desc = r.cfg.Tasks[name].Desc
		}
		r.log.Printf("%-16s %s\n", name, desc)
	}
}

func (r *Runner) Run(taskName string) error {
	return r.RunTargets([]string{taskName})
}

func (r *Runner) RunTargets(targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("task name is required")
	}

	defer r.closeLogPlugins()

	plan, err := r.buildPlan(targets)
	if err != nil {
		r.flushTrace()
		return err
	}

	if r.opts.DryRun {
		r.printDryRun(plan)
		r.flushTrace()
		return nil
	}

	err = r.executePlan(plan)
	r.flushTrace()
	return err
}
