#!/usr/bin/env bash
set -e

# Create a temp dir and clean it up on exit
TEMPDIR=`mktemp -d -t mtest-test.XXX`
trap "rm -rf $TEMPDIR" EXIT HUP INT QUIT TERM

# Build the Mtest binary for the tests
echo "--> Building mtest"
go build -o $TEMPDIR/mtest || exit 1

# Run the tests
echo "--> Running tests"
GOBIN="`which go`"
sudo -E PATH=$TEMPDIR:$PATH  -E GOPATH=$GOPATH \
    $GOBIN test ${GOTEST_FLAGS:--cover -timeout=900s} $($GOBIN list ./... | grep -v /vendor/)

