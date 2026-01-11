package runner

import (
	"fmt"
	"io"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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
	outputsMu  sync.Mutex
	outputs    map[string]map[string]string
	registryMu sync.Mutex
	trace      *traceRecorder
	remote     remoteCache
	logPlugins []logPlugin
	gitVars    map[string]string
	randMu     sync.Mutex
	rand       *rand.Rand
	resources  *resourceManager
}

func New(cfg *config.Config, opts Options, stdout, stderr io.Writer) *Runner {
	log := newLogger(stdout, stderr, opts.LogLevel, opts.JSON)
	log.SetFormat(opts.Timestamp, opts.Color)
	root := "."
	if cfg != nil && cfg.Path != "" {
		root = filepath.Dir(cfg.Path)
	}
	r := &Runner{
		cfg:        cfg,
		opts:       opts,
		log:        log,
		configRoot: root,
		exports:    map[string]string{},
		outputs:    map[string]map[string]string{},
	}
	r.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	r.baseEnv = r.buildBaseEnv()
	r.secrets = r.resolveSecrets(cfg.Secrets, r.baseEnv)
	r.log.SetSecrets(r.secrets)
	r.gitVars = gitMetadata(root)
	r.trace = newTraceRecorder(root, opts.TimelinePath)
	r.remote = newRemoteCache(cfg.CacheRemote, root, log)
	r.logPlugins = r.startLogPlugins(cfg.LogPlugins)
	r.resources = newResourceManager(cfg.Resources, root, log)
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
			if len(r.cfg.Tasks[name].Aliases) > 0 {
				aliasText := "aliases: " + strings.Join(r.cfg.Tasks[name].Aliases, ",")
				if desc == "" {
					desc = aliasText
				} else {
					desc = desc + " (" + aliasText + ")"
				}
			}
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

	r.exportsMu.Lock()
	r.exports = map[string]string{}
	r.exportsMu.Unlock()

	r.outputsMu.Lock()
	r.outputs = map[string]map[string]string{}
	r.outputsMu.Unlock()

	resolved, err := r.resolveTargets(targets)
	if err != nil {
		r.flushTrace()
		return err
	}

	plan, err := r.buildPlan(resolved)
	if err != nil {
		r.flushTrace()
		return err
	}
	if r.opts.Until != "" {
		if err := plan.PrefixUntil(r.opts.Until); err != nil {
			r.flushTrace()
			return err
		}
	}
	if r.opts.Reverse {
		if err := plan.Reverse(); err != nil {
			r.flushTrace()
			return err
		}
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
