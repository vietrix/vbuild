package runner

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

type progressTracker struct {
	total    int
	out      io.Writer
	enabled  bool
	mu       sync.Mutex
	ok       int
	failed   int
	skipped  int
	upToDate int
	done     int
}

func newProgressTracker(total int, out io.Writer, enabled bool) *progressTracker {
	if !enabled || total == 0 {
		return &progressTracker{enabled: false}
	}
	if out == nil {
		out = os.Stderr
	}
	if !isTerminal(out) {
		return &progressTracker{enabled: false}
	}
	return &progressTracker{total: total, out: out, enabled: true}
}

func (p *progressTracker) Enabled() bool {
	return p.enabled
}

func (p *progressTracker) Update(result taskResult) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done++
	switch result.status {
	case statusFailed:
		p.failed++
	case statusUpToDate:
		p.upToDate++
	case statusSkipped, statusCanceled:
		p.skipped++
	default:
		p.ok++
	}
	p.renderLocked()
}

func (p *progressTracker) Render() {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.renderLocked()
}

func (p *progressTracker) Finish() {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintln(p.out)
}

func (p *progressTracker) renderLocked() {
	percent := 0
	if p.total > 0 {
		percent = int(float64(p.done) / float64(p.total) * 100.0)
	}
	fmt.Fprintf(p.out, "\r[%-3d%%] done=%d/%d ok=%d failed=%d skipped=%d up-to-date=%d",
		percent, p.done, p.total, p.ok, p.failed, p.skipped, p.upToDate)
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
