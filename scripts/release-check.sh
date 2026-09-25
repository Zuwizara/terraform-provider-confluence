#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

for required_file in LICENSE README.md CHANGELOG.md VERSION terraform-registry-manifest.json; do
  test -s "$required_file" || {
    echo "missing or empty release file: $required_file" >&2
    exit 1
  }
done

version=$(tr -d '[:space:]' < VERSION)
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]]; then
  echo "VERSION is not a semantic version: $version" >&2
  exit 1
fi

if [[ "${GITHUB_REF_TYPE:-}" == "tag" ]]; then
  tag_version=${GITHUB_REF_NAME#v}
  if [[ "$tag_version" != "$version" ]]; then
    echo "tag ${GITHUB_REF_NAME} does not match VERSION ($version)" >&2
    exit 1
  fi
fi

python3 - <<'PY'
import json

with open("terraform-registry-manifest.json", encoding="utf-8") as stream:
    manifest = json.load(stream)

if manifest.get("version") != 1:
    raise SystemExit("manifest version must be 1")
if "5.0" not in manifest.get("metadata", {}).get("protocol_versions", []):
    raise SystemExit("manifest must declare provider protocol 5.0")
PY

unformatted=$(find . -type f -name '*.go' -print0 | xargs -0 gofmt -l)
if [[ -n "$unformatted" ]]; then
  echo "Go files need formatting:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go mod tidy -diff
go test ./...
go vet ./...

if command -v terraform >/dev/null 2>&1; then
  terraform fmt -check -recursive
else
  echo "warning: terraform not found; skipped terraform fmt check" >&2
fi

if command -v goreleaser >/dev/null 2>&1; then
  goreleaser check
else
  echo "warning: goreleaser not found; skipped GoReleaser config validation" >&2
fi

echo "release checks passed for v$version"
