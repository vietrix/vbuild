package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

type logLevel int

const (
	levelDebug logLevel = iota
	levelInfo
	levelWarn
	levelError
)

type logger struct {
	out       io.Writer
	err       io.Writer
	mu        sync.Mutex
	level     logLevel
	json      bool
	timestamp bool
	colorMode string
	secrets   []string
	hooks     []io.Writer
}

func newLogger(out, err io.Writer, level string, jsonOutput bool) *logger {
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	return &logger{out: out, err: err, level: parseLogLevel(level), json: jsonOutput}
}

func (l *logger) SetSecrets(values []string) {
	l.secrets = append([]string(nil), values...)
}

func (l *logger) SetFormat(timestamp bool, colorMode string) {
	l.timestamp = timestamp
	l.colorMode = colorMode
}

func (l *logger) AddHook(out io.Writer) {
	if out == nil {
		return
	}
	l.hooks = append(l.hooks, out)
}

func (l *logger) Printf(format string, args ...interface{}) {
	l.log(levelInfo, format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.log(levelError, format, args...)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.log(levelDebug, format, args...)
}

func (l *logger) log(level logLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = maskString(msg, l.secrets)
	writer := l.out
	if level >= levelError {
		writer = l.err
	}
	if l.json {
		l.writeJSON(writer, map[string]interface{}{
			"ts":      time.Now().UTC().Format(time.RFC3339Nano),
			"level":   level.String(),
			"message": strings.TrimRight(msg, "\n"),
		})
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	prefix := l.formatPrefix(level)
	if prefix != "" {
		fmt.Fprint(writer, prefix)
	}
	fmt.Fprint(writer, msg)
	l.emitHooks(map[string]interface{}{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   level.String(),
		"message": strings.TrimRight(msg, "\n"),
	})
}

func (l *logger) commandWriter(prefix string, stream string, secrets []string) io.Writer {
	allSecrets := append([]string(nil), l.secrets...)
	allSecrets = append(allSecrets, secrets...)
	allSecrets = dedupeNonEmpty(allSecrets)
	if l.json {
		return newStreamWriter(l, stream, prefix, allSecrets)
	}
	var target io.Writer = &maskWriter{out: l.out, secrets: allSecrets}
	if stream == "stderr" {
		target = &maskWriter{out: l.err, secrets: allSecrets}
	}
	if prefix == "" {
		if len(l.hooks) == 0 {
			return target
		}
		return newMultiWriter(target, newHookWriter(l, stream, prefix, allSecrets))
	}
	prefixed := newPrefixWriter(prefix, target, &l.mu)
	if len(l.hooks) == 0 {
		return prefixed
	}
	return newMultiWriter(prefixed, newHookWriter(l, stream, prefix, allSecrets))
}

func (l *logger) writeJSON(out io.Writer, payload map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	enc := json.NewEncoder(out)
	_ = enc.Encode(payload)
	for _, hook := range l.hooks {
		_ = json.NewEncoder(hook).Encode(payload)
	}
}

func (l *logger) emitHooks(payload map[string]interface{}) {
	if len(l.hooks) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, hook := range l.hooks {
		_ = json.NewEncoder(hook).Encode(payload)
	}
}

func parseLogLevel(value string) logLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return levelDebug
	case "warn", "warning":
		return levelWarn
	case "error":
		return levelError
	default:
		return levelInfo
	}
}

func (l logLevel) String() string {
	switch l {
	case levelDebug:
		return "debug"
	case levelWarn:
		return "warn"
	case levelError:
		return "error"
	default:
		return "info"
	}
}

func (l *logger) formatPrefix(level logLevel) string {
	if !l.timestamp && l.colorMode == "" {
		return ""
	}
	ts := ""
	if l.timestamp {
		ts = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	}
	levelText := strings.ToUpper(level.String())
	if l.shouldColor() {
		levelText = colorize(level, levelText)
	}
	if ts == "" {
		return fmt.Sprintf("[%s] ", levelText)
	}
	return fmt.Sprintf("[%s %s] ", levelText, ts)
}

func (l *logger) shouldColor() bool {
	switch strings.ToLower(strings.TrimSpace(l.colorMode)) {
	case "always", "true", "yes", "on":
		return true
	case "never", "false", "no", "off":
		return false
	case "", "auto":
		return supportsColor(l.out)
	default:
		return supportsColor(l.out)
	}
}

func supportsColor(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func colorize(level logLevel, value string) string {
	switch level {
	case levelError:
		return "\x1b[31m" + value + "\x1b[0m"
	case levelWarn:
		return "\x1b[33m" + value + "\x1b[0m"
	case levelDebug:
		return "\x1b[36m" + value + "\x1b[0m"
	default:
		return "\x1b[32m" + value + "\x1b[0m"
	}
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

type maskWriter struct {
	out     io.Writer
	secrets []string
}

func (w *maskWriter) Write(p []byte) (int, error) {
	if len(w.secrets) == 0 {
		return w.out.Write(p)
	}
	masked := maskString(string(p), w.secrets)
	return w.out.Write([]byte(masked))
}

func maskString(value string, secrets []string) string {
	out := value
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		out = strings.ReplaceAll(out, secret, "***")
	}
	return out
}

type streamWriter struct {
	log     *logger
	stream  string
	prefix  string
	secrets []string
	buf     bytes.Buffer
}

func newStreamWriter(log *logger, stream, prefix string, secrets []string) *streamWriter {
	return &streamWriter{log: log, stream: stream, prefix: prefix, secrets: secrets}
}

func (w *streamWriter) Write(p []byte) (int, error) {
	total := len(p)
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(line)
			break
		}
		w.emit(line)
	}
	return total, nil
}

func (w *streamWriter) Flush() {
	if w.buf.Len() == 0 {
		return
	}
	w.emit(w.buf.String())
	w.buf.Reset()
}

func (w *streamWriter) emit(line string) {
	msg := strings.TrimRight(line, "\n")
	msg = maskString(msg, w.secrets)
	payload := map[string]interface{}{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   "info",
		"stream":  w.stream,
		"message": msg,
	}
	if w.prefix != "" {
		payload["prefix"] = strings.TrimSpace(w.prefix)
	}
	w.log.writeJSON(w.log.out, payload)
}

type hookWriter struct {
	log     *logger
	stream  string
	prefix  string
	secrets []string
	buf     bytes.Buffer
}

func newHookWriter(log *logger, stream, prefix string, secrets []string) *hookWriter {
	return &hookWriter{log: log, stream: stream, prefix: prefix, secrets: secrets}
}

func (w *hookWriter) Write(p []byte) (int, error) {
	total := len(p)
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(line)
			break
		}
		w.emit(line)
	}
	return total, nil
}

func (w *hookWriter) Flush() {
	if w.buf.Len() == 0 {
		return
	}
	w.emit(w.buf.String())
	w.buf.Reset()
}

func (w *hookWriter) emit(line string) {
	msg := strings.TrimRight(line, "\n")
	msg = maskString(msg, w.secrets)
	payload := map[string]interface{}{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   "info",
		"stream":  w.stream,
		"message": msg,
	}
	if w.prefix != "" {
		payload["prefix"] = strings.TrimSpace(w.prefix)
	}
	w.log.emitHooks(payload)
}

type multiWriter struct {
	writers []io.Writer
}

func newMultiWriter(writers ...io.Writer) io.Writer {
	out := []io.Writer{}
	for _, w := range writers {
		if w == nil {
			continue
		}
		out = append(out, w)
	}
	if len(out) == 1 {
		return out[0]
	}
	return &multiWriter{writers: out}
}

func (m *multiWriter) Write(p []byte) (int, error) {
	for _, w := range m.writers {
		if _, err := w.Write(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (m *multiWriter) Flush() {
	for _, w := range m.writers {
		if flusher, ok := w.(interface{ Flush() }); ok {
			flusher.Flush()
		}
	}
}
