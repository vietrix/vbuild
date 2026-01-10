#!/usr/bin/env sh
set -eu

VERSION="${VERSION:-dev}"

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"

mkdir -p "$DIST_DIR"
cd "$ROOT_DIR"

build() {
  os="$1"
  arch="$2"
  output="$3"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -buildvcs=false -buildid= \
      -ldflags "-s -w -X main.version=$VERSION" \
      -o "$DIST_DIR/$output" ./cmd/vbuild
}

checksum() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$DIST_DIR" && sha256sum "$file" > "$file.sha256")
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    (cd "$DIST_DIR" && shasum -a 256 "$file" > "$file.sha256")
    return
  fi
  echo "sha256sum or shasum is required" >&2
  exit 1
}

build linux amd64 linux-amd64
build linux arm64 linux-arm64
build darwin amd64 darwin-amd64
build darwin arm64 darwin-arm64
build windows amd64 windows-amd64.exe

checksum linux-amd64
checksum linux-arm64
checksum darwin-amd64
checksum darwin-arm64
checksum windows-amd64.exe

echo "release artifacts in $DIST_DIR"
