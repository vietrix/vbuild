package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vietrix/vbuild/internal/config"
	"github.com/vietrix/vbuild/internal/runner"
)

type Server struct {
	cfg       *config.Config
	opts      runner.Options
	version   string
	startedAt time.Time
	token     string
	addr      string
	mu        sync.Mutex
}

type RunRequest struct {
	Task    string   `json:"task,omitempty"`
	Targets []string `json:"targets,omitempty"`
	DryRun  bool     `json:"dry_run,omitempty"`
}

type RunResponse struct {
	Success    bool   `json:"success"`
	ExitCode   int    `json:"exit_code"`
	Error      string `json:"error,omitempty"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

type StatusResponse struct {
	Version   string `json:"version"`
	PID       int    `json:"pid"`
	Addr      string `json:"addr"`
	StartedAt string `json:"started_at"`
	UptimeMs  int64  `json:"uptime_ms"`
	TaskCount int    `json:"task_count"`
}

type DaemonInfo struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}

func Serve(ctx context.Context, cfg *config.Config, opts runner.Options, version, addr, token string, infoPath string) error {
	if addr == "" {
		addr = "127.0.0.1:8377"
	}
	server := &Server{
		cfg:       cfg,
		opts:      opts,
		version:   version,
		startedAt: time.Now(),
		token:     token,
		addr:      addr,
	}

	if err := server.writeInfo(infoPath); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", server.handleStatus)
	mux.HandleFunc("/tasks", server.handleTasks)
	mux.HandleFunc("/run", server.handleRun)
	mux.HandleFunc("/shutdown", server.handleShutdown)

	httpServer := &http.Server{Addr: addr, Handler: mux}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = httpServer.Close()
	}()

	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	_ = os.Remove(infoPath)
	return nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	payload := StatusResponse{
		Version:   s.version,
		PID:       os.Getpid(),
		Addr:      s.addr,
		StartedAt: s.startedAt.UTC().Format(time.RFC3339Nano),
		UptimeMs:  time.Since(s.startedAt).Milliseconds(),
		TaskCount: len(s.cfg.Tasks),
	}
	writeJSON(w, payload)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	buf := &bytes.Buffer{}
	runner.New(s.cfg, s.opts, buf, buf).ListTasksJSON(buf)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	targets := req.Targets
	if len(targets) == 0 {
		if req.Task != "" {
			targets = []string{req.Task}
		} else {
			targets = []string{"default"}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := s.opts
	opts.DryRun = req.DryRun
	instance := runner.New(s.cfg, opts, stdout, stderr)
	err := instance.RunTargets(targets)
	exitCode := runner.ExitCode(err)
	resp := RunResponse{
		Success:    err == nil,
		ExitCode:   exitCode,
		DurationMs: time.Since(start).Milliseconds(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}
	if err != nil {
		resp.Error = err.Error()
	}
	writeJSON(w, resp)
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{"status": "shutting down"})
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
}

func (s *Server) authorized(r *http.Request) bool {
	if s.token == "" {
		return true
	}
	return r.Header.Get("X-VBuild-Token") == s.token
}

func (s *Server) writeInfo(path string) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	info := DaemonInfo{Addr: s.addr, Token: s.token, PID: os.Getpid()}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func LoadInfo(path string) (*DaemonInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info DaemonInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func Client(addr, token string) *http.Client {
	return &http.Client{Timeout: 120 * time.Second}
}

func doRequest(method, url, token string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("X-VBuild-Token", token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("daemon request failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
