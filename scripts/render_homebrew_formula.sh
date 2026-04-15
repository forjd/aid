#!/bin/sh

set -eu

usage() {
  printf '%s\n' "Usage: $0 <tag> <sha256> [output-path]" >&2
}

if [ "$#" -ne 2 ] && [ "$#" -ne 3 ]; then
  usage
  exit 1
fi

tag="$1"
sha256="$2"
output_path="${3:-}"
url="https://github.com/forjd/aid/archive/refs/tags/${tag}.tar.gz"

render_formula() {
  printf '%s\n' 'class Aid < Formula'
  printf '%s\n' '  desc "Local memory for coding agents and developers working inside Git repositories"'
  printf '%s\n' '  homepage "https://github.com/forjd/aid"'
  printf '  url "%s"\n' "$url"
  printf '  sha256 "%s"\n' "$sha256"
  printf '%s\n' '  license "MIT"'
  printf '%s\n' ''
  printf '%s\n' '  depends_on "go" => :build'
  printf '%s\n' ''
  printf '%s\n' '  def install'
  printf '%s\n' '    system "go", "build", "-trimpath", "-o", bin/"aid", "./cmd/aid"'
  printf '%s\n' '  end'
  printf '%s\n' ''
  printf '%s\n' '  test do'
  printf '%s\n' '    assert_match "aid - local memory for coding agents and repos", shell_output("#{bin}/aid --help")'
  printf '%s\n' '  end'
  printf '%s\n' 'end'
}

if [ -n "$output_path" ]; then
  mkdir -p "$(dirname "$output_path")"
  render_formula > "$output_path"
else
  render_formula
fi
