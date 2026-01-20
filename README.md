# vbuild

vbuild is a fast, minimal, cross-platform task runner that executes real shell commands, manages task dependencies, supports parallel execution, and streams stdout/stderr live. It is designed for CI/CD, local development, and production-grade workflows.

Docs (EN/VI): https://vietrix.github.io/vbuild/

## Install

Installer scripts are hosted from the `scripts/` directory on GitHub.

### Linux/macOS

```sh
curl -fsSL https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.sh | sh
```

Pin a version:

```sh
VBUILD_VERSION=v0.1.2 curl -fsSL https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.ps1 | iex
```

Pin a version:

```powershell
$env:VBUILD_VERSION = "v0.1.2"; iwr -useb https://raw.githubusercontent.com/vietrix/vbuild/main/scripts/install.ps1 | iex
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
vbuild script -- --flag # pass args to a task
vbuild --version        # print version
vbuild -V               # short version flag
vbuild update           # update to latest release
vbuild update --to v0.1.2
vbuild list --json      # list tasks as JSON
vbuild graph --format dot
vbuild watch <task>     # re-run on file changes
vbuild --since 2024-01-01T00:00:00Z
vbuild --until build
vbuild --reverse cleanup
vbuild --export-env --export-env-path .env
vbuild --print-vars build
vbuild --json-summary summary.json
vbuild --progress
vbuild tag <tag>        # run tasks by tag
vbuild only-changed     # run tasks affected by git diff
vbuild inspect <task>   # resolved task details
vbuild shell <task>     # open shell with task env/workdir
vbuild doctor           # check tooling
vbuild clean            # remove .vbuild cache/artifacts
vbuild dataset list     # list datasets
vbuild dataset show ds  # show dataset metadata
vbuild experiment list  # list experiments
vbuild experiment show id
vbuild lineage          # show dataset lineage
vbuild registry status  # show registry root
vbuild report --out report.json
vbuild daemon start     # start daemon mode
vbuild init             # scaffold .vbuild.yml
vbuild lock             # write .vbuild.lock
```

Common flags:
- `--max-parallel` limit concurrent tasks
- `--continue-on-error` keep running independent tasks
- `--fail-fast` stop immediately on first failure
- `--reverse` run in reverse dependency order
- `--until` stop at a target in topo order
- `--profile` print critical path
- `--progress` show progress meter
- `--explain` show skip reasons
- `--json` structured logs
- `--log-level` debug|info|warn|error
- `--timestamp` include timestamps in logs
- `--color` auto|always|never
- `--env-file` override .env path
- `--timeout` default task timeout
- `--timeline` write timeline JSON
- `--json-summary` write summary JSON
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
- `defaults` for timeout/shell/workdir/retries/backoff/jitter.
- `aliases` for alternate task names.
- `pass_args` to allow CLI args in tasks (`{{ARGS}}`, `{{ARG_0}}`, `{{ARGC}}`).
- `script` for single-command tasks with pass-through args and `{name}` placeholders.
- `fail_fast` to stop DAG execution on the first failure.
- `templates` + `use`/`with` for reusable task blocks.
- `include` to merge additional YAML configs (local file, URL, or glob).
- `env_file` or `.env` for environment loading.
- `when` for conditional execution (`env.*`, `vars.*`, `os`, `arch`).
- `only_on` for branch/tag matching.
- `inputs` + `outputs` + `cache` (`mtime` or `sha256`) for incremental builds.
- `output_paths` to propagate produced outputs as downstream inputs.
- `output` to export output variables to downstream tasks.
- `matrix` for strategy expansion (creates task variants).
- `fanout` to split task commands into parallel DAG nodes.
- `workdir`, `shell`, `timeout`, `retries`, `max_retries`, `backoff`, `jitter`, `continue_on_error`.
- per-command timeouts via `timeout=1m: <command>`.
- `depends_on` for conditional dependencies.
- `retry_on_exit_codes`, `retry_on_regex`, and `retry_on_signal` for targeted retries.
- `allow_failure` to mark tasks as non-fatal.
- `exports` to export environment variables to downstream tasks.
- `capture` to redirect stdout/stderr to files.
- `silent` to suppress command output for a task.
- `limits` for CPU/memory caps (Unix shells).
- `remote` for SSH task execution.
- `isolate` to run tasks in a sandboxed workspace.
- `pre`/`post` hooks and `confirm` prompts.
- `watch` patterns for `vbuild watch`.
- `tags` for `vbuild tag <name>`.
- `artifacts` collection into `.vbuild/artifacts` (or `artifacts_dir`).
- `artifacts_upload` to push artifacts to GitHub releases or S3.
- `docker` helpers for build/push/pull tasks.
- `secrets` to mask log output (values pulled from env).
- `plugins` for task lifecycle hooks.
- `log_plugins` to receive JSON log events on stdin.
- `cache_remote` to store build cache in S3/GCS/MinIO.
- `cache_remote` supports `AWS_PROFILE`/`GCP_PROFILE` and `AWS_SHARED_CREDENTIALS_FILE`.
- `if_missing` to skip tasks when outputs already exist.
- `require` to check binaries/versions before running.
- `seed`/`seed_env` for deterministic pipelines.
- `resources`/`group` for CPU/memory/GPU scheduling.
- `scheduler` for Slurm/PBS wrappers.
- `datasets` + `dataset_outputs` for registry-backed inputs/outputs.
- `split`, `validate`, and `stats` for data prep workflows.
- `metrics`, `benchmark`, and `canary` for quality gates.
- `experiment` tracking with a local registry.
- `snapshot` for environment/config capture.
- `checkpoint`, `model_card`, `notebook`, and `export` for ML artifacts.
- `offline` to enforce offline mode for model hubs.

## Script tasks

Shorthand for a single command that can receive CLI args.

```yaml
tasks:
  train:
    script: python train.py
```

Run with free-form flags:

```sh
vbuild train -- --seed 42
vbuild train --seed 42
```

Named placeholders:

```yaml
tasks:
  train:
    script: python train.py {dir} {out}
```

```sh
vbuild train --dir=data --out=dist
```

Placeholders accept letters, numbers, and `_`. Flags like `--data-dir` map to `{data_dir}`.
Flags passed as `--key=value` are also available as `{{KEY}}` / `{{VBUILD_ARG_KEY}}` (uppercased, `-` -> `_`).
`script` tasks accept args automatically (no need for `pass_args: true`).

## Aliases and namespaces

Tasks may define `aliases` for alternate names, and namespaced tasks like `frontend:build`. Running `vbuild frontend` executes all tasks with the `frontend:` prefix.

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

Optional: use `artifacts_upload` to push collected artifacts to a GitHub release or S3.

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

### Research workflow

```yaml
datasets:
  images:
    path: data/images
    version: 2024-01-01

tasks:
  train:
    datasets:
      - name: images
    metrics:
      regex: ["accuracy=(?P<value>[0-9\\.]+)"]
    checkpoint:
      paths: [checkpoints/*.pt]
      var: CHECKPOINT_PATH
    model_card:
      path: .vbuild/model_cards/train.md
    run:
      - python train.py --data {{DATASET_IMAGES_PATH}} --ckpt {{CHECKPOINT_PATH}}
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

Use `vbuild lock` to write `.vbuild.lock` with the current vbuild version and config hash. By default, vbuild will warn if the lock version or config hash differs from the running binary/config. Use `--strict-lock` to fail on mismatch or `--ignore-lock` to skip the check.

## Structured logging

Use `--json` for JSON logs and `--log-level` to control verbosity. When `parallel: true`, command output is prefixed (or tagged in JSON).

## Watch mode

`vbuild watch <task>` uses filesystem events when possible and falls back to polling (using `watch` patterns or `inputs`) to re-run the task. Use `--events`, `--poll`, `--interval`, and `--debounce` to tune behavior.

## Only-changed mode

Use `vbuild only-changed` or the `--only-changed` flag to run tasks whose `inputs`/`watch` patterns match `git diff` results. Example:

```sh
vbuild only-changed --base origin/main
vbuild --only-changed --changed-base origin/main test
```

Use `--since` to select tasks with inputs modified after a timestamp or duration:

```sh
vbuild --since 2024-01-01T00:00:00Z
vbuild --since 12h build
```

## Env export

Use `--export-env` to write the resolved environment for a task (default `default`) to `.env`, or set `--export-env-path` to choose a different location. Use `--print-vars` to print resolved variables for a task.

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

