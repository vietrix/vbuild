#!/usr/bin/env sh
set -eu

VERSION="${VERSION:-dev}"
want_latest=0
case "${VBUILD_RELEASE_LATEST:-}" in
  1|true|TRUE|yes|YES|on|ON) want_latest=1 ;;
esac

suffixes="$VERSION"
case "$VERSION" in
  latest) suffixes="lastest" ;;
  *-lastest) suffixes="lastest" ;;
esac
if [ "$want_latest" -eq 1 ] && [ "$suffixes" != "lastest" ]; then
  suffixes="$suffixes lastest"
fi

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"

mkdir -p "$DIST_DIR"
cd "$ROOT_DIR"

build() {
  os="$1"
  arch="$2"
  output="$3"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -buildvcs=false \
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

for suffix in $suffixes; do
  build linux amd64 "linux-amd64-$suffix"
  build linux arm64 "linux-arm64-$suffix"
  build darwin amd64 "darwin-amd64-$suffix"
  build darwin arm64 "darwin-arm64-$suffix"
  build windows amd64 "windows-amd64-$suffix.exe"

  checksum "linux-amd64-$suffix"
  checksum "linux-arm64-$suffix"
  checksum "darwin-amd64-$suffix"
  checksum "darwin-arm64-$suffix"
  checksum "windows-amd64-$suffix.exe"
done

echo "release artifacts in $DIST_DIR"
