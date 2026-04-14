#!/bin/sh

set -eu

REPO="${AID_REPO:-forjd/aid}"
BIN_NAME="aid"
BIN_DIR="${AID_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${AID_VERSION:-}"
TMPDIR=""

usage() {
  cat <<EOF
Install ${BIN_NAME} from a GitHub release.

Usage:
  install.sh [-b DIR] [-v VERSION]

Options:
  -b, --bin-dir DIR   Install into DIR instead of \$HOME/.local/bin
  -v, --version TAG   Install a specific release tag, for example v0.1.0
  -h, --help          Show this help

Environment:
  AID_INSTALL_DIR     Default install directory
  AID_VERSION         Default release tag
  AID_REPO            GitHub repo in owner/name form
EOF
}

log() {
  printf '%s\n' "$*" >&2
}

die() {
  log "error: $*"
  exit 1
}

cleanup() {
  if [ -n "${TMPDIR:-}" ] && [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}

trap cleanup EXIT INT TERM

while [ "$#" -gt 0 ]; do
  case "$1" in
    -b|--bin-dir)
      [ "$#" -ge 2 ] || die "missing value for $1"
      BIN_DIR="$2"
      shift 2
      ;;
    -v|--version)
      [ "$#" -ge 2 ] || die "missing value for $1"
      VERSION="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

download() {
  url="$1"
  dest="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
    return
  fi

  die "either curl or wget is required"
}

sha256_file() {
  file="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{ print $1 }'
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{ print $1 }'
    return
  fi

  die "sha256sum or shasum is required"
}

detect_os() {
  case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
    linux)
      printf 'linux'
      ;;
    darwin)
      printf 'darwin'
      ;;
    *)
      die "unsupported operating system: $(uname -s)"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'amd64'
      ;;
    aarch64|arm64)
      printf 'arm64'
      ;;
    *)
      die "unsupported architecture: $(uname -m)"
      ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
ARCHIVE_NAME="${BIN_NAME}_${OS}_${ARCH}.tar.gz"
CHECKSUM_NAME="checksums.txt"

if [ -n "$VERSION" ]; then
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
else
  BASE_URL="https://github.com/${REPO}/releases/latest/download"
fi

require_cmd awk
require_cmd find
require_cmd mkdir
require_cmd tar

TMPDIR="$(mktemp -d 2>/dev/null || mktemp -d -t "${BIN_NAME}-install")"
ARCHIVE_PATH="${TMPDIR}/${ARCHIVE_NAME}"
CHECKSUM_PATH="${TMPDIR}/${CHECKSUM_NAME}"
EXTRACT_DIR="${TMPDIR}/extract"

log "Downloading ${ARCHIVE_NAME}"
download "${BASE_URL}/${ARCHIVE_NAME}" "$ARCHIVE_PATH"

log "Downloading ${CHECKSUM_NAME}"
download "${BASE_URL}/${CHECKSUM_NAME}" "$CHECKSUM_PATH"

EXPECTED_SUM="$(awk -v file="$ARCHIVE_NAME" '$2 == file { print $1 }' "$CHECKSUM_PATH")"
[ -n "$EXPECTED_SUM" ] || die "no checksum entry found for ${ARCHIVE_NAME}"

ACTUAL_SUM="$(sha256_file "$ARCHIVE_PATH")"
[ "$EXPECTED_SUM" = "$ACTUAL_SUM" ] || die "checksum verification failed for ${ARCHIVE_NAME}"

mkdir -p "$EXTRACT_DIR"
tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"

BIN_PATH="$(find "$EXTRACT_DIR" -type f -name "$BIN_NAME" | head -n 1)"
[ -n "$BIN_PATH" ] || die "could not find ${BIN_NAME} inside ${ARCHIVE_NAME}"

mkdir -p "$BIN_DIR"
cp "$BIN_PATH" "${BIN_DIR}/${BIN_NAME}"
chmod 755 "${BIN_DIR}/${BIN_NAME}"

log "Installed ${BIN_NAME} to ${BIN_DIR}/${BIN_NAME}"

case ":$PATH:" in
  *":$BIN_DIR:"*)
    ;;
  *)
    log "Add ${BIN_DIR} to your PATH if it is not already available."
    ;;
esac
