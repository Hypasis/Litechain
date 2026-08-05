#!/usr/bin/env bash
set -e

# Litechain Multi-Platform Release Generator
# Usage: ./scripts/release.sh [VERSION]
# Example: ./scripts/release.sh v2.0.0

VERSION="${1:-v2.0.0}"
VERSION_NO_V="${VERSION#v}"
DIST_DIR="dist/${VERSION}"
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS="-X github.com/Hypasis/Litechain/pkg/version.GitCommit=${GIT_COMMIT} -X github.com/Hypasis/Litechain/pkg/version.BuildTime=${BUILD_TIME} -X github.com/Hypasis/Litechain/pkg/version.Version=${VERSION_NO_V}"

echo "📦 Generating Litechain Release Assets for ${VERSION}..."
echo "   • Git Commit: ${GIT_COMMIT}"
echo "   • Build Time: ${BUILD_TIME}"

# Clean existing distribution directory
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/arm64"
    "darwin/amd64"
)

for TARGET in "${TARGETS[@]}"; do
    OS=$(echo "${TARGET}" | cut -d'/' -f1)
    ARCH=$(echo "${TARGET}" | cut -d'/' -f2)
    RELEASE_NAME="lightchain-${VERSION}-${OS}-${ARCH}"
    STAGE_DIR="${DIST_DIR}/${RELEASE_NAME}"

    echo "🔨 Compiling for ${OS}/${ARCH}..."
    mkdir -p "${STAGE_DIR}"

    # Build node and cli binaries
    GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "${STAGE_DIR}/lightchain" ./cmd/lightchain
    GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "${STAGE_DIR}/lightchain-cli" ./tools/lightchain-cli

    # Copy README & LICENSE
    cp README.md LICENSE "${STAGE_DIR}/"

    # Package tar.gz archive
    TARBALL="${DIST_DIR}/${RELEASE_NAME}.tar.gz"
    tar -czf "${TARBALL}" -C "${DIST_DIR}" "${RELEASE_NAME}"
    rm -rf "${STAGE_DIR}"

    echo "  ✅ Created ${TARBALL}"
done

# Generate SHA256 Checksums
echo "🔐 Generating SHA256 Checksums..."
cd "${DIST_DIR}"
shasum -a 256 *.tar.gz > checksums.txt
cd - > /dev/null

echo "🎉 Release ${VERSION} ready in ${DIST_DIR}/:"
ls -lh "${DIST_DIR}"
