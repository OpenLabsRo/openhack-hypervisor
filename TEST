#!/bin/bash
usage() {
	cat <<USAGE
Usage: ./TEST [--env-root PATH] [--app-version VERSION]

Forwards optional arguments to the Go test suite, which applies its own
fallbacks when values are omitted.
USAGE
}

if [[ $# -gt 0 ]]; then
	case "$1" in
		-h|--help)
			usage
			exit 0
			;;
	esac
fi

go_cmd=(go test ./test/... -v -count=1)

if (( $# )); then
	go_cmd+=(-args)
	go_cmd+=("$@")
fi

"${go_cmd[@]}"
