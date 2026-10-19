#!/usr/bin/env bash
# Writes the Homebrew formula for a release to standard output.
#
# Usage: scripts/formula.sh VERSION CHECKSUMS
#
# VERSION is the version without "v", for example 0.3.0. CHECKSUMS is the
# checksums.txt file of the release.
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 VERSION CHECKSUMS" >&2
  exit 2
fi
version="$1"
sums="$2"
base="https://github.com/juthrbog/lazybw/releases/download/v${version}"

# sha prints the checksum of the archive for a platform, such as linux_amd64.
sha() {
  local file="lazybw_${version}_$1.tar.gz" sum
  sum="$(awk -v f="$file" '$2 == f { print $1 }' "$sums")"
  if [[ ! "$sum" =~ ^[0-9a-f]{64}$ ]]; then
    echo "no checksum for $file in $sums" >&2
    exit 1
  fi
  echo "$sum"
}

darwin_arm64="$(sha darwin_arm64)"
darwin_amd64="$(sha darwin_amd64)"
linux_arm64="$(sha linux_arm64)"
linux_amd64="$(sha linux_amd64)"

cat <<FORMULA
# Generated from the release's SHA-256 checksums. Do not edit manually.
class Lazybw < Formula
  desc "Terminal UI for the Bitwarden password manager"
  homepage "https://github.com/juthrbog/lazybw"
  version "${version}"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "${base}/lazybw_${version}_darwin_arm64.tar.gz"
      sha256 "${darwin_arm64}"
    end
    on_intel do
      url "${base}/lazybw_${version}_darwin_amd64.tar.gz"
      sha256 "${darwin_amd64}"
    end
  end

  on_linux do
    on_arm do
      url "${base}/lazybw_${version}_linux_arm64.tar.gz"
      sha256 "${linux_arm64}"
    end
    on_intel do
      url "${base}/lazybw_${version}_linux_amd64.tar.gz"
      sha256 "${linux_amd64}"
    end
  end

  def install
    bin.install "lazybw"
  end

  def caveats
    <<~EOS
      lazybw needs the Bitwarden CLI (bw) on your PATH.
      Install it with "brew install bitwarden-cli", or see
      https://bitwarden.com/help/cli/#download-and-install
      lazybw also needs a Nerd Font v3 and a terminal with OSC 52.
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/lazybw --version")
  end
end
FORMULA
