# vbuild

vbuild is a fast, minimal, cross-platform task runner that executes real shell commands, manages task dependencies, supports parallel execution, and streams stdout/stderr live. It is designed for CI/CD, local development, and production-grade workflows.

## Install

Installer scripts are hosted from the `scripts/` directory on GitHub.

### Linux/macOS

```sh
curl -fsSL https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.sh | sh
```

Pin a version:

```sh
VBUILD_VERSION=v1.2.3 curl -fsSL https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.ps1 | iex
```

Pin a version:

```powershell
$env:VBUILD_VERSION = "v1.2.3"; iwr -useb https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.ps1 | iex
```

## Usage

```sh
vbuild                  # run default task
vbuild build            # run build task + deps
vbuild help             # show help
vbuild list             # list tasks
vbuild --file other.yml # use alternate config
vbuild --dry-run        # print commands only
vbuild --dry-run=json   # emit JSON plan
vbuild --version        # print version
vbuild -V               # short version flag
vbuild update           # update to latest release
vbuild update --to v1.2.3
vbuild list --json      # list tasks as JSON
vbuild graph --format dot
vbuild watch <task>     # re-run on file changes
vbuild tag <tag>        # run tasks by tag
vbuild only-changed     # run tasks affected by git diff
vbuild inspect <task>   # resolved task details
vbuild shell <task>     # open shell with task env/workdir
vbuild doctor           # check tooling
vbuild clean            # remove .vbuild cache/artifacts
vbuild daemon start     # start daemon mode
vbuild init             # scaffold .vbuild.yml
vbuild lock             # write .vbuild.lock
```

Common flags:
- `--max-parallel` limit concurrent tasks
- `--continue-on-error` keep running independent tasks
- `--profile` print critical path
- `--explain` show skip reasons
- `--json` structured logs
- `--log-level` debug|info|warn|error
- `--timestamp` include timestamps in logs
- `--color` auto|always|never
- `--env-file` override .env path
- `--timeout` default task timeout
- `--timeline` write timeline JSON
- `--yes` auto-confirm prompts

## Configuration

vbuild reads `.vbuild.yml` from the project root (or `--file`).

```yaml
workflow: "Example Workflows"

vars:
  KEY: value

env:
  KEY: value

tasks:
  default:
    desc: run all
    deps: [fmt, test]
    run:
      - gofmt -w .
      - go test ./...

  build:
    desc: build binary
    env:
      GOOS: linux
      GOARCH: amd64
    run:
      - go build -o bin/app ./cmd/server

  dev:
    desc: run frontend + backend
    parallel: true
    run:
      - cd backend && go run main.go
      - cd frontend && npm run dev
```

Notes:
- `vars` can be referenced in commands as `{{VAR_NAME}}`.
- `vars` can be overridden via environment variables named `VBUILD_VAR_<NAME>`.
- `env` is merged: OS env -> global `env` -> task `env`.
- Config is validated on load with clear, path-specific errors.
- Dependencies are resolved with cycle detection.
- Tasks run as a DAG: independent tasks execute concurrently.
- `parallel: true` runs commands in a task concurrently with prefixed logs.
- A summary with per-task timing is printed after execution.

Additional config features:
- `templates` + `use`/`with` for reusable task blocks.
- `include` to merge additional YAML configs (local file or URL).
- `env_file` or `.env` for environment loading.
- `when` for conditional execution (`env.*`, `vars.*`, `os`, `arch`).
- `inputs` + `outputs` + `cache` (`mtime` or `sha256`) for incremental builds.
- `output_paths` to propagate produced outputs as downstream inputs.
- `matrix` for strategy expansion (creates task variants).
- `fanout` to split task commands into parallel DAG nodes.
- `workdir`, `shell`, `timeout`, `retries`, `backoff`, `continue_on_error`.
- per-command timeouts via `timeout=1m: <command>`.
- `depends_on` for conditional dependencies.
- `retry_on_exit_codes` and `retry_on_regex` for targeted retries.
- `allow_failure` to mark tasks as non-fatal.
- `exports` to export environment variables to downstream tasks.
- `limits` for CPU/memory caps (Unix shells).
- `remote` for SSH task execution.
- `isolate` to run tasks in a sandboxed workspace.
- `pre`/`post` hooks and `confirm` prompts.
- `watch` patterns for `vbuild watch`.
- `tags` for `vbuild tag <name>`.
- `artifacts` collection into `.vbuild/artifacts` (or `artifacts_dir`).
- `docker` helpers for build/push/pull tasks.
- `secrets` to mask log output (values pulled from env).
- `plugins` for task lifecycle hooks.
- `log_plugins` to receive JSON log events on stdin.
- `cache_remote` to store build cache in S3/GCS/MinIO.

## Self-update

`vbuild update` queries the GitHub Releases API for `vietrix/vbuild`, selects the correct asset for your OS/arch, downloads it, and replaces the current binary. If the release provides a `.sha256` checksum for the asset, vbuild verifies it before replacement and rolls back on failure. On Windows, replacement is deferred to a helper script because the running executable cannot be overwritten.

## Cross-platform behavior

- Windows uses PowerShell to execute commands.
- Linux/macOS uses `sh`.
- All stdout/stderr streams are live.
- Exit codes propagate directly for CI/CD.

## CI/CD integration

### GitHub Actions

```yaml
name: build
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22.x'
      - run: curl -fsSL https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.sh | sh
      - run: vbuild test
      - run: vbuild build
```

### Generic CI

```sh
curl -fsSL https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.sh | sh
vbuild --file .vbuild.yml test
```

## Release pipeline

Release builds are generated by GitHub Actions using the `vbuild release` task defined in `.vbuild.yml`. The pipeline:
- injects the version into the binary (`-X main.version=$VERSION`)
- builds deterministic artifacts for all supported OS/arch pairs
- generates per-asset `.sha256` checksum files for verified self-update
- names assets as `<os>-<arch>-vX.Y.Z` for versioned releases

See `.github/workflows/release.yml` for the exact steps.

Optional local release helpers:
- POSIX: `scripts/release.sh`
- Windows: `scripts/release.ps1`
Set `VERSION` in the environment to control the embedded version string.

## Examples

### Go project

```yaml
workflow: "Go build"

tasks:
  default:
    deps: [fmt, test, build]

  fmt:
    run:
      - gofmt -w .

  test:
    run:
      - go test ./...

  build:
    run:
      - go build -o bin/app ./cmd/app
```

### Non-Go (Node + Docker)

```yaml
workflow: "Web app"

tasks:
  default:
    deps: [install, lint, build, docker]

  install:
    run:
      - npm ci

  lint:
    run:
      - npm run lint

  build:
    run:
      - npm run build

  docker:
    run:
      - docker build -t myapp:latest .
```

### Templates + Matrix

```yaml
templates:
  go-build:
    run:
      - go build -o {{OUT}} {{PKG}}

tasks:
  build:
    use: go-build
    with:
      OUT: bin/app
      PKG: ./cmd/app

  test:
    matrix:
      GOOS: [linux, darwin]
      GOARCH: [amd64, arm64]
    env:
      GOOS: "{{GOOS}}"
      GOARCH: "{{GOARCH}}"
    run:
      - go test ./...
```

## Testing

Automated:

```sh
go test ./...
```

Manual:

```sh
vbuild list
vbuild --dry-run
vbuild build
```

Reproducibility:
- `go.mod` pins the Go toolchain version.
- Release builds use `-trimpath`, `-buildvcs=false`, and `CGO_ENABLED=0` for deterministic output.

## Lock file

Use `vbuild lock` to write `.vbuild.lock` with the current vbuild version. By default, vbuild will warn if the lock version differs from the running binary. Use `--strict-lock` to fail on mismatch or `--ignore-lock` to skip the check.

## Structured logging

Use `--json` for JSON logs and `--log-level` to control verbosity. When `parallel: true`, command output is prefixed (or tagged in JSON).

## Watch mode

`vbuild watch <task>` polls for file changes (using `watch` patterns or `inputs`) and re-runs the task. Use `--interval` and `--debounce` to tune polling.

## Only-changed mode

Use `vbuild only-changed` or the `--only-changed` flag to run tasks whose `inputs`/`watch` patterns match `git diff` results. Example:

```sh
vbuild only-changed --base origin/main
vbuild --only-changed --changed-base origin/main test
```

## Daemon mode

Daemon mode keeps a background process ready to execute tasks.

```sh
vbuild daemon start
vbuild daemon status
vbuild daemon run build
vbuild daemon stop
```

## Timeline traces

Use `--timeline path.json` to record task/command timings as JSON events.

## License

This repository uses a dual-license model:

- **Apache-2.0** for open-source, personal, educational, and internal use. See `LICENSE`.
- **Commercial License** required for commercial products, SaaS platforms, CI/CD services, resale, embedding into paid tools, or any offering that competes with vbuild. See `LICENSE-COMMERCIAL`.

For commercial licensing, contact the author.
