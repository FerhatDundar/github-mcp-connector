#!/usr/bin/env bash
# Renders a fresh server.json for the MCP Registry from a set of already-
# built release zip assets. Used by the release pipeline (and callable by
# hand) so server.json's version/identifiers/hashes always match exactly
# what's attached to the GitHub release being published.
#
# Usage: render-server-json.sh <version> <tag> <asset-dir>
#   version    e.g. 0.1.0  (no leading v)
#   tag        e.g. v0.1.0 (matches the GitHub release tag)
#   asset-dir  directory containing the
#              github-mcp-connector-plugin-<tag>-<goos>-<goarch>.zip files
#
# Prints the rendered server.json to stdout.
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <version> <tag> <asset-dir>" >&2
  exit 1
fi

VERSION="$1"
TAG="$2"
ASSET_DIR="$3"
REPO="FerhatDundar/github-mcp-connector"
PLATFORMS=(darwin-arm64 darwin-amd64 linux-amd64 linux-arm64 windows-amd64 windows-arm64)

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
command -v shasum >/dev/null && HASH_CMD=(shasum -a 256) || HASH_CMD=(sha256sum)

env_vars='[
  {"name":"GITHUB_TOKEN","description":"GitHub personal access token (classic or fine-grained)","isRequired":true,"isSecret":true,"format":"string"},
  {"name":"GITHUB_API_URL","description":"GitHub Enterprise Server API base URL. Leave unset for github.com.","format":"string"}
]'

packages="[]"
for plat in "${PLATFORMS[@]}"; do
  file="$ASSET_DIR/github-mcp-connector-plugin-${TAG}-${plat}.zip"
  if [ ! -f "$file" ]; then
    echo "missing release asset: $file" >&2
    exit 1
  fi
  sha=$("${HASH_CMD[@]}" "$file" | awk '{print $1}')
  url="https://github.com/${REPO}/releases/download/${TAG}/github-mcp-connector-plugin-${TAG}-${plat}.zip"

  pkg=$(jq -n \
    --arg url "$url" \
    --arg version "$VERSION" \
    --arg sha "$sha" \
    --argjson envVars "$env_vars" \
    '{
      registryType: "mcpb",
      identifier: $url,
      version: $version,
      fileSha256: $sha,
      transport: { type: "stdio" },
      environmentVariables: $envVars
    }')
  packages=$(jq --argjson p "$pkg" '. + [$p]' <<<"$packages")
done

jq -n \
  --arg version "$VERSION" \
  --argjson packages "$packages" \
  '{
    "$schema": "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json",
    name: "io.github.FerhatDundar/github-mcp-connector",
    description: "MCP server for GitHub: repos, files, issues, PRs, search. Single Go binary.",
    repository: { url: "https://github.com/FerhatDundar/github-mcp-connector", source: "github" },
    version: $version,
    websiteUrl: "https://github.com/FerhatDundar/github-mcp-connector#readme",
    packages: $packages
  }'
