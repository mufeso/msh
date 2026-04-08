#!/usr/bin/env bash
# release.sh VERSION - cross-compile msh for every supported platform.
#
# Produces dist/msh-VERSION-OS-ARCH and dist/SHA256SUMS.
set -euo pipefail

cd "$(dirname "$0")/.."
version="${1:?usage: scripts/release.sh VERSION}"
commit="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
pkg="example.com/msh/internal/buildinfo"
ldflags="-s -w -X ${pkg}.Version=${version} -X ${pkg}.Commit=${commit} -X ${pkg}.Date=${date}"

platforms=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64)

rm -rf dist
mkdir -p dist
for p in "${platforms[@]}"; do
	os="${p%/*}"
	arch="${p#*/}"
	out="dist/msh-${version}-${os}-${arch}"
	echo "==> ${out}"
	CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$ldflags" -o "$out" ./cmd/msh
done

cd dist
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum msh-* > SHA256SUMS
else
	shasum -a 256 msh-* > SHA256SUMS
fi
echo "==> dist/SHA256SUMS"
cat SHA256SUMS
