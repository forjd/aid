#!/bin/sh

set -eu

usage() {
  cat >&2 <<EOF
Usage: $0 <tag> [--checksums <path>] [output-path]

Arguments:
  tag          The release tag (e.g. v0.4.5)
  --checksums  Optional path to the checksums.txt file downloaded from the release.
               Used to render a formula that installs the prebuilt binaries.
               When omitted, the script falls back to a build-from-source formula
               (previous behaviour) and <sha256> is read as the second argument.
  output-path  Optional path to write the formula to. If omitted, the formula is
               printed to stdout.

Examples:
  # Build-from-source (legacy): requires a source-archive sha256
  $0 v0.4.5 "<sha256>"             Formula/aid.rb

  # Prebuilt binaries (preferred): reads checksums.txt from the release
  $0 v0.4.5 --checksums checksums.txt Formula/aid.rb
EOF
}

if [ "$#" -lt 2 ]; then
  usage
  exit 1
fi

tag="$1"
shift

checksums_path=""
legacy_sha=""
output_path=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --checksums)
      [ "$#" -ge 2 ] || { usage; exit 1; }
      checksums_path="$2"
      shift 2
      ;;
    *)
      if [ -z "$legacy_sha" ] && [ -z "$checksums_path" ]; then
        legacy_sha="$1"
        shift
      else
        output_path="$1"
        shift
      fi
      ;;
  esac
done

extract_sha() {
  file="$1"
  archive_name="$2"
  awk -v name="$archive_name" '$2 == name { print $1 }' "$file"
}

render_legacy_formula() {
  url="https://github.com/forjd/aid/archive/refs/tags/${tag}.tar.gz"
  cat <<EOF
class Aid < Formula
  desc "Local memory for coding agents and developers working inside Git repositories"
  homepage "https://github.com/forjd/aid"
  url "${url}"
  sha256 "${legacy_sha}"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-trimpath", "-o", bin/"aid", "./cmd/aid"
  end

  test do
    assert_match "aid - local memory for coding agents and repos", shell_output("#{bin}/aid --help")
  end
end
EOF
}

render_binary_formula() {
  darwin_amd64="$(extract_sha "$checksums_path" "aid_darwin_amd64.tar.gz")"
  darwin_arm64="$(extract_sha "$checksums_path" "aid_darwin_arm64.tar.gz")"
  linux_amd64="$(extract_sha "$checksums_path" "aid_linux_amd64.tar.gz")"
  linux_arm64="$(extract_sha "$checksums_path" "aid_linux_arm64.tar.gz")"

  for sum in "$darwin_amd64" "$darwin_arm64" "$linux_amd64" "$linux_arm64"; do
    [ -n "$sum" ] || { echo "error: missing checksum entry in $checksums_path" >&2; exit 1; }
  done

  cat <<EOF
class Aid < Formula
  desc "Local memory for coding agents and developers working inside Git repositories"
  homepage "https://github.com/forjd/aid"
  version "${tag#v}"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/forjd/aid/releases/download/${tag}/aid_darwin_arm64.tar.gz"
      sha256 "${darwin_arm64}"
    end
    on_intel do
      url "https://github.com/forjd/aid/releases/download/${tag}/aid_darwin_amd64.tar.gz"
      sha256 "${darwin_amd64}"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/forjd/aid/releases/download/${tag}/aid_linux_arm64.tar.gz"
      sha256 "${linux_arm64}"
    end
    on_intel do
      url "https://github.com/forjd/aid/releases/download/${tag}/aid_linux_amd64.tar.gz"
      sha256 "${linux_amd64}"
    end
  end

  def install
    bin.install "aid"
  end

  test do
    assert_match "aid - local memory for coding agents and repos", shell_output("#{bin}/aid --help")
  end
end
EOF
}

if [ -n "$checksums_path" ]; then
  [ -f "$checksums_path" ] || { echo "error: checksums file not found: $checksums_path" >&2; exit 1; }
  render() { render_binary_formula; }
else
  [ -n "$legacy_sha" ] || { usage; exit 1; }
  render() { render_legacy_formula; }
fi

if [ -n "$output_path" ]; then
  mkdir -p "$(dirname "$output_path")"
  render > "$output_path"
else
  render
fi
