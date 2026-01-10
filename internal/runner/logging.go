package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

type logger struct {
	out io.Writer
	err io.Writer
	mu  sync.Mutex
}

func newLogger(out, err io.Writer) *logger {
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	return &logger{out: out, err: err}
}

func (l *logger) Printf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.err, format, args...)
}

type prefixWriter struct {
	prefix      string
	out         io.Writer
	mu          *sync.Mutex
	atLineStart bool
}

func newPrefixWriter(prefix string, out io.Writer, mu *sync.Mutex) *prefixWriter {
	return &prefixWriter{prefix: prefix, out: out, mu: mu, atLineStart: true}
}

func (w *prefixWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		idx := bytes.IndexByte(p, '\n')
		chunk := p
		if idx >= 0 {
			chunk = p[:idx+1]
		}

		if w.atLineStart {
			combined := make([]byte, 0, len(w.prefix)+len(chunk))
			combined = append(combined, w.prefix...)
			combined = append(combined, chunk...)
			w.mu.Lock()
			_, err := w.out.Write(combined)
			w.mu.Unlock()
			if err != nil {
				return total, err
			}
		} else {
			w.mu.Lock()
			_, err := w.out.Write(chunk)
			w.mu.Unlock()
			if err != nil {
				return total, err
			}
		}

		total += len(chunk)
		p = p[len(chunk):]
		if len(chunk) > 0 && chunk[len(chunk)-1] == '\n' {
			w.atLineStart = true
		} else {
			w.atLineStart = false
		}
	}
	return total, nil
}
