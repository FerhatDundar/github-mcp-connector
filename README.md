<div align="center">

# 🔌 github-mcp-connector

**Talk to GitHub from an MCP-speaking agent.**

[![CI](https://github.com/FerhatDundar/github-mcp-connector/actions/workflows/ci.yml/badge.svg)](https://github.com/FerhatDundar/github-mcp-connector/actions/workflows/ci.yml)
[![CodeQL](https://github.com/FerhatDundar/github-mcp-connector/actions/workflows/codeql.yml/badge.svg)](https://github.com/FerhatDundar/github-mcp-connector/actions/workflows/codeql.yml)
[![Latest release](https://img.shields.io/github/v/release/FerhatDundar/github-mcp-connector?color=blueviolet&label=release)](https://github.com/FerhatDundar/github-mcp-connector/releases/latest)
[![MCP Registry](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fregistry.modelcontextprotocol.io%2Fv0%2Fservers%3Fsearch%3Dgithub-mcp-connector&query=%24.servers%5B-1%3A%5D.server.version&label=MCP%20Registry&prefix=v&color=6A2FEE&logo=modelcontextprotocol)](https://registry.modelcontextprotocol.io/?q=github-mcp-connector)
[![Go Reference](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/license-MIT-yellow.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/protocol-MCP-orange)](https://modelcontextprotocol.io/)
[![Conventional Commits](https://img.shields.io/badge/commits-conventional-ff69b4)](https://www.conventionalcommits.org/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

---

A single static Go binary that speaks the [Model Context Protocol](https://modelcontextprotocol.io/)
and exposes 18 tools for working with GitHub: repositories, branches,
commits, file contents, issues, pull requests, and search.

No Python, no `uv`, no runtime dependency to install — just a binary and
an `.mcp.json`.

## ✨ Why this exists

> An agent that can only *talk about* your GitHub repos isn't that useful.
> This gives it hands: it can read and write files in a repo, open and
> triage issues, push a branch and open a PR, and merge it — all against
> the real GitHub API, with the same guardrails (size caps, friendly
> errors, destructive-action annotations) as the sibling connectors.

## 🧰 Tools

| Tool | What it does | Write? |
|---|---|:---:|
| `github_whoami` | Get the user authenticated by `GITHUB_TOKEN` | |
| `github_list_repos` | List repos for a user/org, or the authenticated user | |
| `github_get_repo` | Get one repository's details | |
| `github_create_repo` | Create a new repository | ✍️ |
| `github_list_branches` | List branches in a repository | |
| `github_list_commits` | List commits, optionally filtered by branch/path | |
| `github_get_file_contents` | Read a file's content, or list a directory | |
| `github_create_or_update_file` | Create/update a file via a commit (≤ 5 MB) | ✍️ |
| `github_list_issues` | List issues in a repository | |
| `github_get_issue` | Get one issue's full details | |
| `github_create_issue` | Create a new issue | ✍️ |
| `github_update_issue` | Update an issue's title/body/state/labels | ✍️ |
| `github_add_issue_comment` | Comment on an issue or PR | ✍️ |
| `github_list_pull_requests` | List pull requests | |
| `github_get_pull_request` | Get one PR's full details, incl. diff stats | |
| `github_create_pull_request` | Open a pull request | ✍️ |
| `github_merge_pull_request` | Merge a pull request | 🗑️ destructive |
| `github_search_repositories` | Search repositories with GitHub search qualifiers | |

Every tool accepts an optional `response_format`: `markdown` (default,
pretty tables for a chat UI) or `json` (for programmatic use).

## 🚀 Quickstart

**Fastest path:** grab a prebuilt bundle from the [latest release](https://github.com/FerhatDundar/github-mcp-connector/releases/latest) —
download `github-mcp-connector-plugin-<version>-<os>-<arch>.zip`, unzip it,
and point Cowork/Claude at the `plugin/` folder inside (see step 4 of
[SETUP.md](SETUP.md)). No Go toolchain required.

**From source:**

```bash
# 1. Build
cd go-server
go mod tidy
go build -o github-connector-server .
cp github-connector-server ../plugin/servers/go/

# 2. Set up auth — a personal access token, see SETUP.md
export GITHUB_TOKEN=ghp_...

# 3. Run
./go-server/github-connector-server   # serves MCP over stdio
```

Or `make build` — see the [Makefile](Makefile) for every shortcut
(`test`, `vet`, `fmt`, `lint`, `tidy`).

Full walkthrough — including wiring this up as a Claude/Cowork plugin — is
in **[SETUP.md](SETUP.md)**.

## 🔐 Configuration

Everything is environment variables, passed through by the plugin's
`.mcp.json`:

| Variable | Purpose | Default |
|---|---|---|
| `GITHUB_TOKEN` | Personal access token (classic or fine-grained). Needs `repo`/`public_repo` and `read:org` scopes depending on what you use it for. | — (required) |
| `GITHUB_API_URL` | GitHub Enterprise Server API base URL, e.g. `https://github.example.com/api/v3`. | *(unset — github.com)* |

## 🧪 Quality bar

This isn't a toy script — it's got the same checks you'd expect from a
production Go service:

- ✅ **Unit tests** for every input-validation path (`go test ./...`)
- ✅ **`go vet`** + **`gofmt`** clean
- ✅ **[golangci-lint](https://golangci-lint.run/)** (govet, staticcheck, errcheck, gosec, and more)
- ✅ **[govulncheck](https://go.dev/blog/vuln)** — no known vulnerabilities in the dependency graph
- ✅ **[CodeQL](https://codeql.github.com/)** static security analysis on every push
- ✅ **End-to-end verified** — every tool (repos, branches, commits, file
  read/write, issues, comments, PRs, merge, search) was exercised against
  a real, dedicated GitHub sandbox repo, not mocks
- ✅ **[Dependabot](.github/dependabot.yml)** keeps Go modules and Actions current

All of it runs in [CI](.github/workflows/ci.yml) on every push and PR.

## 🏷️ Releases & versioning

Versions follow [semver](https://semver.org/) and are cut automatically by
[release-please](https://github.com/googleapis/release-please) from
[Conventional Commits](https://www.conventionalcommits.org/) on `main`:

- `fix: ...` → patch (`v0.1.0` → `v0.1.1`)
- `feat: ...` → minor (`v0.1.1` → `v0.2.0`)
- `feat!: ...` / `BREAKING CHANGE:` footer → major (`v0.2.0` → `v1.0.0`)

Every merged PR updates a standing **"chore(main): release vX.Y.Z"** PR
with an auto-generated [CHANGELOG.md](CHANGELOG.md). Merging that PR:

1. tags the release and publishes it on GitHub
2. builds and attaches zipped, ready-to-install plugin bundles for
   linux/darwin/windows × amd64/arm64
3. regenerates `server.json` from those exact assets (fresh version +
   SHA-256 hashes) and publishes it to the
   [official MCP Registry](https://registry.modelcontextprotocol.io/) via
   `mcp-publisher`, authenticated with GitHub OIDC — no stored secrets

See [.github/workflows/release-please.yml](.github/workflows/release-please.yml)
and [.github/workflows/publish-mcp-registry.yml](.github/workflows/publish-mcp-registry.yml)
(also runnable by hand for an existing tag via `workflow_dispatch`).

## 📁 Layout

```
github-mcp-connector/
├── README.md                  ← you are here
├── SETUP.md                   ← step-by-step setup guide
├── CONTRIBUTING.md             ← how to contribute
├── CODE_OF_CONDUCT.md
├── SECURITY.md                 ← vulnerability reporting
├── CODEOWNERS
├── LICENSE                     ← MIT
├── Makefile                    ← build / test / lint shortcuts
├── .golangci.yml                ← lint rules
├── release-please-config.json  ← semver/changelog automation config
├── .release-please-manifest.json
├── server.json                  ← MCP Registry manifest (regenerated fresh per release by CI)
├── scripts/
│   └── render-server-json.sh    ← rebuilds server.json from a release's zip assets
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                     ← build, vet, test, lint, govulncheck
│   │   ├── codeql.yml                 ← security scanning
│   │   ├── pr-title.yml               ← Conventional Commits PR title check
│   │   ├── release-please.yml         ← version PRs, tagging, GitHub releases
│   │   ├── publish-mcp-registry.yml   ← publishes server.json to the MCP Registry
│   │   └── rebuild-release-assets.yml ← manual re-attach fallback
│   ├── ISSUE_TEMPLATE/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── dependabot.yml
├── go-server/                  ← the MCP server source
│   ├── main.go
│   ├── main_test.go
│   ├── go.mod / go.sum
│   └── README.md
└── plugin/                     ← installable Cowork/Claude plugin
    ├── .claude-plugin/plugin.json
    ├── .mcp.json                ← holds credentials locally — never commit real ones
    └── servers/go/              ← compiled binary goes here
```

## 🤝 Contributing

PRs and issues are very welcome — see **[CONTRIBUTING.md](CONTRIBUTING.md)**
for the full guide (setup, coding conventions, how to add a new tool) and
the **[Code of Conduct](CODE_OF_CONDUCT.md)**.

`main` is protected: every change, including the maintainer's, lands via
pull request with CI green. PR titles must follow
[Conventional Commits](https://www.conventionalcommits.org/) — that's what
drives the automatic versioning above.

Found a security issue? Please follow **[SECURITY.md](SECURITY.md)**
instead of opening a public issue.

## 📄 License

[MIT](LICENSE) © FerhatDundar
