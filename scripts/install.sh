#!/usr/bin/env sh
set -eu

VERSION="${VBUILD_VERSION:-latest}"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux) OS=linux ;;
  Darwin) OS=darwin ;;
  *)
    echo "unsupported OS: $OS" >&2
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)
    echo "unsupported arch: $ARCH" >&2
    exit 1
    ;;
esac

if [ "$VERSION" = "latest" ]; then
  SUFFIX="lastest"
elif [ "${VERSION%-lastest}" != "$VERSION" ]; then
  SUFFIX="lastest"
else
  SUFFIX="$VERSION"
fi

ASSET="$OS-$ARCH-$SUFFIX"

if [ "$VERSION" = "latest" ]; then
  URL="https://github.com/vietrix/vbuild/releases/latest/download/$ASSET"
else
  URL="https://github.com/vietrix/vbuild/releases/download/$VERSION/$ASSET"
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

BIN_PATH="$TMPDIR/vbuild"

download() {
  url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$BIN_PATH"
    return $?
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO "$BIN_PATH" "$url"
    return $?
  fi
  echo "curl or wget is required" >&2
  exit 1
}

fetch_latest_tag() {
  api="https://api.github.com/repos/vietrix/vbuild/releases/latest"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -H "User-Agent: vbuild-installer" "$api"
    return $?
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO- --header="User-Agent: vbuild-installer" "$api"
    return $?
  fi
  return 1
}

if ! download "$URL"; then
  if [ "$VERSION" = "latest" ]; then
    tag="$(fetch_latest_tag | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
    if [ -n "$tag" ]; then
      ASSET="$OS-$ARCH-$tag"
      URL="https://github.com/vietrix/vbuild/releases/download/$tag/$ASSET"
      download "$URL" || {
        echo "failed to download $URL" >&2
        exit 1
      }
    else
      echo "failed to resolve latest release tag" >&2
      exit 1
    fi
  else
    echo "failed to download $URL" >&2
    exit 1
  fi
fi

chmod 755 "$BIN_PATH"

if [ -w /usr/local/bin ]; then
  DEST="/usr/local/bin"
else
  DEST="$HOME/.local/bin"
  mkdir -p "$DEST"
fi

install -m 755 "$BIN_PATH" "$DEST/vbuild"

echo "vbuild installed to $DEST/vbuild"
if ! command -v vbuild >/dev/null 2>&1; then
  echo "make sure $DEST is in your PATH" >&2
fi
