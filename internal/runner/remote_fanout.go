package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) runRemoteFanout(ctx context.Context, taskName string, commands []string, opts commandOptions, task *config.Task) error {
	if opts.remote == nil || len(opts.remote.Hosts) == 0 {
		if task != nil && task.Parallel {
			return r.runParallel(ctx, taskName, commands, opts)
		}
		return r.runSequential(ctx, taskName, commands, opts)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, len(opts.remote.Hosts))
	for _, host := range opts.remote.Hosts {
		host := host
		wg.Add(1)
		go func() {
			defer wg.Done()
			hostOpts := opts
			remote := cloneRemote(opts.remote)
			remote.Host = host
			remote.Hosts = nil
			hostOpts.remote = remote
			hostName := fmt.Sprintf("%s@%s", taskName, host)
			var err error
			if task != nil && task.Parallel {
				err = r.runParallel(ctx, hostName, commands, hostOpts)
			} else {
				err = r.runSequential(ctx, hostName, commands, hostOpts)
			}
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		return err
	}
	return nil
}

func cloneRemote(spec *config.RemoteSpec) *config.RemoteSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Hosts = append([]string(nil), spec.Hosts...)
	if spec.Scheduler != nil {
		scheduler := *spec.Scheduler
		scheduler.Args = append([]string(nil), spec.Scheduler.Args...)
		out.Scheduler = &scheduler
	}
	return &out
}
