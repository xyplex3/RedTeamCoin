#!/usr/bin/env bash

set -eu -o pipefail

if ! command -v gocritic &>/dev/null; then
	echo "gocritic not installed or available in the PATH" >&2
	echo "please check https://github.com/go-critic/go-critic" >&2
	exit 1
fi

GODIR=$(dirname "$1") # the pre commit passes the matched file (go.mod) to us
pushd "$GODIR" >/dev/null
gocritic check ./... 2>&1
popd >/dev/null
