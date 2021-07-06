#!/usr/bin/env bash

set -euo pipefail

_version=v$(plutil -convert json info.plist -r -o - | jq .version -r)

if [ -z "${_version}" ]; then
  echo "Could not detect version"
  exit 1
fi

_gooses=(darwin)
_goarches=(arm64 amd64)

echo "Prep to build for OSes ${_gooses[*]} with arches ${_goarches[*]}"

for _goos in "${_gooses[@]}"; do
  for _goarch in "${_goarches[@]}"; do
    echo -n "Building for OS ${_goos} arch ${_goarch}... "
    GOOS=${_goos} GOARCH=${_goarch} CGO_ENABLED=1 go build --tags fts5 -o run ./app
    echo "built"

    _zipname="CraftDocs_SearchIndex_${_version}_${_goos}_${_goarch}.alfredworkflow"

    echo -n "Packing files into ${_zipname}... "
    zip -rq "${_zipname}" icon.png info.plist run
    echo "packed"
  done
done