// Command github-connector-server is an MCP server that lets an agent work
// with GitHub: repositories, branches, file contents, commits, issues, and
// pull requests. Built with google/go-github and mark3labs/mcp-go,
// mirroring the layout and conventions of the sibling s3-mcp-connector,
// discord-mcp-connector, and sqlite-mcp-connector repos.
//
// Auth / config (all via environment variables, passed through by the
// plugin's .mcp.json):
//
//	GITHUB_TOKEN   — a GitHub personal access token (classic or
//	    fine-grained) with the scopes needed for the operations you use
//	    ("repo" and "read:org" for private/org access; "public_repo" is
//	    enough for public-only use). Create one at
//	    https://github.com/settings/tokens.
//	GITHUB_API_URL — optional. Point at a GitHub Enterprise Server API
//	    base URL, e.g. https://github.example.com/api/v3. Leave unset for
//	    github.com.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v79/github"
	"github.com/mark3labs/mcp-go/mcp"
)

var ghClient *github.Client

// newGitHubClient builds an authenticated GitHub API client from
// GITHUB_TOKEN (and optionally GITHUB_API_URL for Enterprise Server), and
// validates the token with one cheap read-only call before the server
// starts serving tools. A stale or revoked token won't surface as invalid
// until it's actually used against the API, so — unlike a purely local
// static-credential check — this makes one network call at startup on
// purpose (see references/auth-patterns.md).
func newGitHubClient(ctx context.Context) (*github.Client, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is not set — create a personal access token at https://github.com/settings/tokens and set it as GITHUB_TOKEN")
	}

	client := github.NewClient(nil).WithAuthToken(token)

	if base := strings.TrimSpace(os.Getenv("GITHUB_API_URL")); base != "" {
		var err error
		client, err = client.WithEnterpriseURLs(base, base)
		if err != nil {
			return nil, fmt.Errorf("invalid GITHUB_API_URL %q: %w", base, err)
		}
	}

	if _, _, err := client.Users.Get(ctx, ""); err != nil {
		return nil, fmt.Errorf("GITHUB_TOKEN did not validate against the GitHub API: %w", err)
	}

	return client, nil
}

// friendlyGitHubError adds actionable hints to common GitHub API failures
// instead of surfacing the raw SDK error string.
func friendlyGitHubError(action string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	var ge *github.ErrorResponse
	if errors.As(err, &ge) && ge.Response != nil {
		switch ge.Response.StatusCode {
		case 401:
			return fmt.Errorf("%s failed: 401 Unauthorized — GITHUB_TOKEN is missing, expired, or revoked. (%s)", action, msg)
		case 403:
			if ge.Response.Header.Get("X-RateLimit-Remaining") == "0" {
				return fmt.Errorf("%s failed: GitHub API rate limit reached — wait for the limit to reset and retry. (%s)", action, msg)
			}
			return fmt.Errorf("%s failed: 403 Forbidden — check GITHUB_TOKEN has the scopes required for this operation. (%s)", action, msg)
		case 404:
			return fmt.Errorf("%s failed: 404 Not Found — the repo/issue/PR/path doesn't exist, or the token can't see it. (%s)", action, msg)
		case 409:
			return fmt.Errorf("%s failed: 409 Conflict — the resource changed underneath this call (e.g. stale sha on a file update, or a merge conflict). Re-fetch and retry. (%s)", action, msg)
		case 422:
			return fmt.Errorf("%s failed: 422 Unprocessable Entity — check the arguments (e.g. duplicate name, invalid field, branch already exists). (%s)", action, msg)
		}
	}
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") {
		return fmt.Errorf("%s failed: could not reach the GitHub API — check network access and GITHUB_API_URL if set. (%s)", action, msg)
	}
	return fmt.Errorf("%s failed: %w", action, err)
}

// ---------------------------------------------------------------------
// Shared MCP infrastructure — identical across fd-mcp-builder connectors.
// ---------------------------------------------------------------------

func getFormat(req mcp.CallToolRequest) string {
	f := req.GetString("response_format", "markdown")
	if f != "json" {
		f = "markdown"
	}
	return f
}

func resultOrError(data map[string]any, err error, format string, mdFn func(map[string]any) string) *mcp.CallToolResult {
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err))
	}
	if format == "json" {
		return mcp.NewToolResultText(renderJSON(data))
	}
	return mcp.NewToolResultText(mdFn(data))
}

func renderJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(b)
}
