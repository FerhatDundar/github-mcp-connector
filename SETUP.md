# Setup guide — GitHub connector

Get a personal access token, build (or download) the server, wire it into
`.mcp.json`, then install it as a Cowork/Claude plugin.

---

## 1. Get a GitHub personal access token

1. Go to <https://github.com/settings/tokens> (fine-grained tokens are at
   <https://github.com/settings/personal-access-tokens/new>).
2. Create a token scoped to what you want the agent to touch:
   - **Classic token**: `repo` (full control of private/public repos) is
     the simplest broad scope; add `read:org` if you need to list org
     repos.
   - **Fine-grained token** (recommended): scope it to specific
     repositories, with Repository permissions **Contents**
     (read/write), **Issues** (read/write), **Pull requests**
     (read/write), and **Metadata** (read, mandatory).
3. Copy the token (`ghp_...` or `github_pat_...`) — you won't see it again.

Using GitHub Enterprise Server instead of github.com? Also note your API
base URL, e.g. `https://github.example.com/api/v3` — you'll set it as
`GITHUB_API_URL` below.

## 2. Get the server binary

**Option A — download a release (no Go needed):** grab
`github-mcp-connector-plugin-<version>-<os>-<arch>.zip` from the
[latest release](https://github.com/FerhatDundar/github-mcp-connector/releases/latest)
and unzip it — the `plugin/` folder inside is ready to install, skip to
step 4.

**Option B — build from source:** you need Go 1.25+ installed (`go
version` to check; get it from <https://go.dev/dl/> if not).

```bash
cd subprojects/github-mcp-connector/go-server
go mod tidy                          # fetches deps, writes go.sum
go build -o github-connector-server .
cp github-connector-server ../plugin/servers/go/
```

## 3. Configure `.mcp.json`

Edit `plugin/.mcp.json`:

```json
{
  "mcpServers": {
    "github": {
      "command": "${CLAUDE_PLUGIN_ROOT}/servers/go/github-connector-server",
      "env": {
        "GITHUB_TOKEN": "ghp_your_real_token_here",
        "GITHUB_API_URL": ""
      }
    }
  }
}
```

Leave `GITHUB_API_URL` empty for github.com. Keep this file out of any
shared/committed location — it now holds a real credential. (The
subproject's `.gitignore` already blocks `.env`/`*.token` files; the
`.mcp.json` inside the *installed* plugin lives in Cowork's plugin
directory, not the repo.)

You can sanity-check the token from a terminal before wiring it into the
plugin:

```bash
GITHUB_TOKEN=ghp_... ./go-server/github-connector-server --version
# then, separately, confirm the token itself works:
curl -s -H "Authorization: Bearer $GITHUB_TOKEN" https://api.github.com/user | head
```

## 4. Install the plugin and verify

1. In the Claude desktop app: **Settings → Capabilities** → add a plugin
   from a local folder, and point it at:
   `subprojects/github-mcp-connector/plugin/`
   (Same way `s3-connector`, `sqlite-connector`, and `discord-connector`
   were installed. A plugin can't be registered from inside a chat
   session — it has to be added here.)
2. Restart / reload so the new MCP server is picked up.
3. Back in a chat, verify:
   - Ask: **"Run github_whoami"** → should return your GitHub login.
   - Ask: **"List my repos"** → should return `github_list_repos` output.
   - Pick a repo you don't mind testing in and ask: **"Create an issue in
     it titled 'connector test'"**, then **"Get that issue"** to confirm
     it round-trips.

Done. From here you can tell the agent things like *"open a PR from
branch X into main"*, *"read the README from repo Y"*, or *"list open
issues labeled bug."*

### Testing without touching a real project

This connector was end-to-end verified against a dedicated, disposable
sandbox repo — [`github-mcp-connector-sandbox`](https://github.com/FerhatDundar/github-mcp-connector-sandbox) —
rather than a real project, so create/write/merge operations had nowhere
important to go wrong. If you want to exercise the write-heavy tools
(`github_create_repo`, `github_create_or_update_file`,
`github_merge_pull_request`, etc.) yourself before pointing this at
something that matters, spin up a similar throwaway public repo first.

---

## Troubleshooting

| Symptom | Likely cause / fix |
|---------|--------------------|
| `GITHUB_TOKEN is not set` at startup | `.mcp.json`'s `env.GITHUB_TOKEN` is empty or the plugin wasn't reloaded after editing it. |
| `GITHUB_TOKEN did not validate against the GitHub API` at startup | Token is malformed, expired, or revoked — generate a new one (step 1). |
| `401 Unauthorized` on a tool call | Same as above — the token stopped being valid after the server started. |
| `403 Forbidden` | Token doesn't have the scope/permission needed for that repo or operation (see step 1). |
| `403 ... rate limit reached` | You've hit GitHub's API rate limit — wait for it to reset (an hour for most authenticated limits). |
| `404 Not Found` | Repo/issue/PR/path doesn't exist, or the token can't see it (private repo without access). |
| `409 Conflict` on `github_create_or_update_file` | The file changed since you last read its `sha` — call `github_get_file_contents` again and retry with the fresh sha. |
| `422 Unprocessable Entity` | Validation error — duplicate name, invalid branch, or a merge conflict on `github_merge_pull_request`. |
| Tools don't appear at all | Binary not built/copied (step 2), or plugin not installed/reloaded (step 4). |
