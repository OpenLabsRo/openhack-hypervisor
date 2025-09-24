#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_FILE="${PROJECT_ROOT}/VERSION"

usage() {
	cat <<USAGE
Usage: ./BUILD [--output DIR]

Builds the OpenHack hypervisor binary for production. By default the executable
is written to the project root bin directory and named after the release
version. Supplying --output DIR places the executable in DIR instead (still
named <version>).
USAGE
}

OUTPUT_DIR=""

while [[ $# -gt 0 ]]; do
	case "$1" in
		--output)
			if [[ $# -lt 2 ]]; then
				echo "missing value for --output" >&2
				usage
				exit 1
			fi
			OUTPUT_DIR="$2"
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "unknown argument: $1" >&2
			usage
			exit 1
			;;
	esac
done

if [[ ! -f "${VERSION_FILE}" ]]; then
	echo "VERSION file not found at ${VERSION_FILE}" >&2
	exit 1
fi

VERSION="$(<"${VERSION_FILE}")"
VERSION="${VERSION//$'\r'/}"
VERSION="${VERSION## }"
VERSION="${VERSION%% }"

if [[ -z "${VERSION}" ]]; then
	echo "VERSION file is empty" >&2
	exit 1
fi

: "${GOOS:=linux}"
: "${GOARCH:=amd64}"
: "${CGO_ENABLED:=0}"

if [[ -z "${OUTPUT_DIR}" ]]; then
	OUTPUT_DIR="${PROJECT_ROOT}/bin"
fi

mkdir -p "${OUTPUT_DIR}"
ARTIFACT_PATH="${OUTPUT_DIR}/${VERSION}"

printf 'Building OpenHack hypervisor %s (%s/%s) -> %s\n' "${VERSION}" "${GOOS}" "${GOARCH}" "${ARTIFACT_PATH}"

go build \
	-trimpath \
	-ldflags "-s -w" \
	-o "${ARTIFACT_PATH}" \
	./cmd/server

printf 'Build complete: %s\n' "${ARTIFACT_PATH}"
