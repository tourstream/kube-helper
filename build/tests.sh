#!/usr/bin/env bash

set -ex

PKGS=$(go list ./... | grep -v /vendor/ | grep -v mocks )
go vet $PKGS
go list -f '{{if len .TestGoFiles}}"go test -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' $PKGS | xargs -L 1 sh -c