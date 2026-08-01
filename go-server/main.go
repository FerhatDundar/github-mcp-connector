package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z" by the
// release workflow. Local `go build` leaves it at "dev".
var version = "dev"

func fmtParam() mcp.ToolOption {
	return mcp.WithString("response_format", mcp.Enum("markdown", "json"), mcp.DefaultString("markdown"), mcp.Description("Output format: 'markdown' or 'json'"))
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println("github-connector-server " + version)
		return
	}

	ctx := context.Background()
	client, err := newGitHubClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "github-connector-server: %v\n", err)
		os.Exit(1)
	}
	ghClient = client

	s := server.NewMCPServer("github-mcp-connector", "0.1.0", server.WithToolCapabilities(false))

	s.AddTool(mcp.NewTool("github_whoami",
		mcp.WithDescription("Get the GitHub user authenticated by GITHUB_TOKEN. Call this first to confirm the token is working."),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := whoami(ctx)
		return resultOrError(data, err, getFormat(req), whoamiMD), nil
	})

	s.AddTool(mcp.NewTool("github_list_repos",
		mcp.WithDescription("List repositories for a user/org, or for the authenticated user if owner is left empty. Paginated via page/per_page."),
		mcp.WithString("owner", mcp.Description("Username or org to list repos for. Leave empty to list the authenticated user's own repos.")),
		mcp.WithString("type", mcp.Enum("all", "owner", "public", "private", "member"), mcp.DefaultString("all"), mcp.Description("Filter by repo type. 'public'/'private' only take effect when owner is empty (the authenticated user's own repos) — GitHub's per-user repos endpoint ignores them and only ever returns that user's public repos.")),
		mcp.WithString("sort", mcp.Enum("created", "updated", "pushed", "full_name"), mcp.DefaultString("full_name"), mcp.Description("Sort field.")),
		mcp.WithString("direction", mcp.Enum("asc", "desc"), mcp.Description("Sort direction.")),
		mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number, 1-based.")),
		mcp.WithNumber("per_page", mcp.DefaultNumber(30), mcp.Min(1), mcp.Max(100), mcp.Description("Results per page (max 100).")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner := req.GetString("owner", "")
		repoType := req.GetString("type", "all")
		sort := req.GetString("sort", "full_name")
		direction := req.GetString("direction", "")
		page := req.GetInt("page", 1)
		perPage := req.GetInt("per_page", defaultPerPage)
		data, err := listRepos(ctx, owner, repoType, sort, direction, page, perPage)
		return resultOrError(data, err, getFormat(req), listReposMD), nil
	})

	s.AddTool(mcp.NewTool("github_get_repo",
		mcp.WithDescription("Get details for one repository: description, default branch, stars, forks, open issues, language, URLs."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner (user or org).")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		data, err := getRepo(ctx, owner, repo)
		return resultOrError(data, err, getFormat(req), getRepoMD), nil
	})

	s.AddTool(mcp.NewTool("github_create_repo",
		mcp.WithDescription("Create a new GitHub repository, under an org (if org is set) or the authenticated user's account."),
		mcp.WithString("org", mcp.Description("Org to create the repo under. Leave empty to create under the authenticated user's account.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("description", mcp.Description("Short repository description.")),
		mcp.WithBoolean("private", mcp.DefaultBool(false), mcp.Description("Create as a private repository.")),
		mcp.WithBoolean("auto_init", mcp.DefaultBool(true), mcp.Description("Initialize with a README so the repo isn't empty.")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(false), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		org := req.GetString("org", "")
		description := req.GetString("description", "")
		private := req.GetBool("private", false)
		autoInit := req.GetBool("auto_init", true)
		data, err := createRepo(ctx, org, name, description, private, autoInit)
		return resultOrError(data, err, getFormat(req), createRepoMD), nil
	})

	s.AddTool(mcp.NewTool("github_list_branches",
		mcp.WithDescription("List branches in a repository. Paginated via page/per_page."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number, 1-based.")),
		mcp.WithNumber("per_page", mcp.DefaultNumber(30), mcp.Min(1), mcp.Max(100), mcp.Description("Results per page (max 100).")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		page := req.GetInt("page", 1)
		perPage := req.GetInt("per_page", defaultPerPage)
		data, err := listBranches(ctx, owner, repo, page, perPage)
		return resultOrError(data, err, getFormat(req), listBranchesMD), nil
	})

	s.AddTool(mcp.NewTool("github_list_commits",
		mcp.WithDescription("List commits in a repository, optionally filtered by branch/SHA and/or file path. Paginated via page/per_page."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("sha", mcp.Description("Branch name or commit SHA to start listing from. Defaults to the repo's default branch.")),
		mcp.WithString("path", mcp.Description("Only commits touching this file path.")),
		mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number, 1-based.")),
		mcp.WithNumber("per_page", mcp.DefaultNumber(30), mcp.Min(1), mcp.Max(100), mcp.Description("Results per page (max 100).")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sha := req.GetString("sha", "")
		path := req.GetString("path", "")
		page := req.GetInt("page", 1)
		perPage := req.GetInt("per_page", defaultPerPage)
		data, err := listCommits(ctx, owner, repo, sha, path, page, perPage)
		return resultOrError(data, err, getFormat(req), listCommitsMD), nil
	})

	registerFileTools(s)
	registerIssueTools(s)
	registerPullRequestTools(s)
	registerSearchTools(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "github-connector-server error: %v\n", err)
		os.Exit(1)
	}
}
