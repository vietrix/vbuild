package runner

import "time"

func (r *Runner) randomJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	r.randMu.Lock()
	defer r.randMu.Unlock()
	return time.Duration(r.rand.Int63n(int64(max)))
}
