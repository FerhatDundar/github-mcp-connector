# Changelog

## 0.1.0 (2026-08-01)

Initial release.

### ✨ Features

- MCP server exposing 18 GitHub tools: `github_whoami`,
  `github_list_repos`, `github_get_repo`, `github_create_repo`,
  `github_list_branches`, `github_list_commits`,
  `github_get_file_contents`, `github_create_or_update_file`,
  `github_list_issues`, `github_get_issue`, `github_create_issue`,
  `github_update_issue`, `github_add_issue_comment`,
  `github_list_pull_requests`, `github_get_pull_request`,
  `github_create_pull_request`, `github_merge_pull_request`,
  `github_search_repositories`
- Auth via `GITHUB_TOKEN` (personal access token, classic or
  fine-grained); optional `GITHUB_API_URL` for GitHub Enterprise Server
- `markdown`/`json` response formats on every tool
- `--version` flag; ldflag-injected build version

### 🧪 Quality

- Unit tests, `go vet`, `gofmt`, `golangci-lint`, `govulncheck`, and
  CodeQL all wired into CI
- End-to-end verified against a real, dedicated GitHub sandbox repo
  ([github-mcp-connector-sandbox](https://github.com/FerhatDundar/github-mcp-connector-sandbox))
  during development — every tool exercised against the live API, not
  mocks

### 🤝 Project infrastructure

- Contribution guide, Code of Conduct, security policy, issue/PR templates
- Branch protection: all changes (including the maintainer's) land via
  reviewed, CI-green pull requests
- Automated semver releases via [release-please](https://github.com/googleapis/release-please),
  starting from this baseline
- Cross-platform (linux/darwin/windows × amd64/arm64) zipped plugin
  bundles attached to every release

---

*From here on, this file is maintained automatically by release-please
based on [Conventional Commits](https://www.conventionalcommits.org/).*
