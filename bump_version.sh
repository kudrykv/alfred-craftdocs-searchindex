#!/usr/bin/env bash

set -euo pipefail

_current_version=$(plutil -convert json info.plist -r -o - | jq .version -r)
if [ -z "${_current_version}" ]; then
  echo 'Could not get the version from info.plist'
  exit 1
fi

_major=$(echo "${_current_version}" | cut -d. -f1)
_minor=$(echo "${_current_version}" | cut -d. -f2)
_patch=$(echo "${_current_version}" | cut -d. -f3)

case "${1:-}" in
patch)
  _patch=$((_patch + 1))
  ;;

minor)
  _minor=$((_minor + 1))
  _patch=0
  ;;

major)
  _major=$((_major+1))
  _minor=0
  _patch=0
  ;;

*)
  echo "Specify bump step: major, minor, or patch"
  exit 1
  ;;
esac

_new_version=${_major}.${_minor}.${_patch}

echo "Current version is ${_current_version}"
echo "Changing it to ${_new_version}"

plutil -replace version -string "${_new_version}" info.plist

git add info.plist
git commit -m "update version from ${_current_version} to ${_new_version}"
git tag -a "v${_new_version}" -m "${_new_version}"