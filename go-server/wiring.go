package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerFileTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("github_get_file_contents",
		mcp.WithDescription("Read a file's content from a repository, or list a directory's entries if path is a directory. Text files are returned as-is; binary files are base64-encoded. Output is capped by max_bytes (default 200,000, hard cap 5,000,000)."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory path, e.g. 'README.md' or 'src/'.")),
		mcp.WithString("ref", mcp.Description("Branch, tag, or commit SHA. Defaults to the repo's default branch.")),
		mcp.WithNumber("max_bytes", mcp.DefaultNumber(defaultFileGetLimit), mcp.Description("Max bytes to return for a file (hard cap 5,000,000).")),
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
		path, err := req.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ref := req.GetString("ref", "")
		maxBytes := req.GetInt("max_bytes", defaultFileGetLimit)
		data, err := getFileContents(ctx, owner, repo, path, ref, int64(maxBytes))
		return resultOrError(data, err, getFormat(req), getFileContentsMD), nil
	})

	s.AddTool(mcp.NewTool("github_create_or_update_file",
		mcp.WithDescription("Create a new file or update an existing one via a single commit. To update an existing file, pass its current blob sha (from github_get_file_contents) — omit sha to create a new file. Limit 5,000,000 bytes of content per call."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to write, e.g. 'docs/notes.md'.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("File content — plain text, or base64 if base64=true.")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Commit message.")),
		mcp.WithString("branch", mcp.Description("Branch to commit to. Defaults to the repo's default branch.")),
		mcp.WithString("sha", mcp.Description("Current blob SHA of the file being updated. Required to update; omit to create a new file.")),
		mcp.WithBoolean("base64", mcp.DefaultBool(false), mcp.Description("If true, decode 'content' from base64 before writing.")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		path, err := req.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		content, err := req.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		message, err := req.RequireString("message")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		branch := req.GetString("branch", "")
		sha := req.GetString("sha", "")
		base64Flag := req.GetBool("base64", false)
		data, err := createOrUpdateFile(ctx, owner, repo, path, content, message, branch, sha, base64Flag)
		return resultOrError(data, err, getFormat(req), createOrUpdateFileMD), nil
	})
}

func registerIssueTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("github_list_issues",
		mcp.WithDescription("List issues in a repository (pull requests are excluded — use github_list_pull_requests for those). Paginated via page/per_page."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("state", mcp.Enum("open", "closed", "all"), mcp.DefaultString("open"), mcp.Description("Filter by issue state.")),
		mcp.WithArray("labels", mcp.Description("Only issues with all of these labels."), mcp.Items(map[string]any{"type": "string"})),
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
		state := req.GetString("state", "open")
		labels := req.GetStringSlice("labels", nil)
		page := req.GetInt("page", 1)
		perPage := req.GetInt("per_page", defaultPerPage)
		data, err := listIssues(ctx, owner, repo, state, labels, page, perPage)
		return resultOrError(data, err, getFormat(req), listIssuesMD), nil
	})

	s.AddTool(mcp.NewTool("github_get_issue",
		mcp.WithDescription("Get one issue's full details, including body."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number.")),
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
		number, err := req.RequireInt("number")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		data, err := getIssue(ctx, owner, repo, number)
		return resultOrError(data, err, getFormat(req), getIssueMD), nil
	})

	s.AddTool(mcp.NewTool("github_create_issue",
		mcp.WithDescription("Create a new issue in a repository."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Issue title.")),
		mcp.WithString("body", mcp.Description("Issue body (Markdown).")),
		mcp.WithArray("labels", mcp.Description("Labels to apply."), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("assignees", mcp.Description("Usernames to assign."), mcp.Items(map[string]any{"type": "string"})),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(false), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := req.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body := req.GetString("body", "")
		labels := req.GetStringSlice("labels", nil)
		assignees := req.GetStringSlice("assignees", nil)
		data, err := createIssue(ctx, owner, repo, title, body, labels, assignees)
		return resultOrError(data, err, getFormat(req), createIssueMD), nil
	})

	s.AddTool(mcp.NewTool("github_update_issue",
		mcp.WithDescription("Update an issue's title, body, state, and/or labels. Only fields you pass are changed."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number.")),
		mcp.WithString("title", mcp.Description("New title.")),
		mcp.WithString("body", mcp.Description("New body (Markdown).")),
		mcp.WithString("state", mcp.Enum("open", "closed"), mcp.Description("New state.")),
		mcp.WithArray("labels", mcp.Description("Replace the issue's labels with this set."), mcp.Items(map[string]any{"type": "string"})),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		number, err := req.RequireInt("number")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title := req.GetString("title", "")
		body := req.GetString("body", "")
		state := req.GetString("state", "")
		labels := req.GetStringSlice("labels", nil)
		data, err := updateIssue(ctx, owner, repo, number, title, body, state, labels)
		return resultOrError(data, err, getFormat(req), updateIssueMD), nil
	})

	s.AddTool(mcp.NewTool("github_add_issue_comment",
		mcp.WithDescription("Add a comment to an issue or pull request (PRs are issues under the hood, so this works on both)."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue or pull request number.")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Comment body (Markdown).")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(false), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		number, err := req.RequireInt("number")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := req.RequireString("body")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		data, err := addIssueComment(ctx, owner, repo, number, body)
		return resultOrError(data, err, getFormat(req), addIssueCommentMD), nil
	})
}

func registerPullRequestTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("github_list_pull_requests",
		mcp.WithDescription("List pull requests in a repository. Paginated via page/per_page."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("state", mcp.Enum("open", "closed", "all"), mcp.DefaultString("open"), mcp.Description("Filter by PR state.")),
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
		state := req.GetString("state", "open")
		page := req.GetInt("page", 1)
		perPage := req.GetInt("per_page", defaultPerPage)
		data, err := listPullRequests(ctx, owner, repo, state, page, perPage)
		return resultOrError(data, err, getFormat(req), listPullRequestsMD), nil
	})

	s.AddTool(mcp.NewTool("github_get_pull_request",
		mcp.WithDescription("Get one pull request's full details: body, mergeability, diff stats."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Pull request number.")),
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
		number, err := req.RequireInt("number")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		data, err := getPullRequest(ctx, owner, repo, number)
		return resultOrError(data, err, getFormat(req), getPullRequestMD), nil
	})

	s.AddTool(mcp.NewTool("github_create_pull_request",
		mcp.WithDescription("Create a pull request from one branch into another."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Pull request title.")),
		mcp.WithString("head", mcp.Required(), mcp.Description("Branch with the changes, e.g. 'feature-x' or 'someuser:feature-x' for a fork.")),
		mcp.WithString("base", mcp.Required(), mcp.Description("Branch to merge into, e.g. 'main'.")),
		mcp.WithString("body", mcp.Description("Pull request description (Markdown).")),
		mcp.WithBoolean("draft", mcp.DefaultBool(false), mcp.Description("Open as a draft PR.")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(false), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := req.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		head, err := req.RequireString("head")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		base, err := req.RequireString("base")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body := req.GetString("body", "")
		draft := req.GetBool("draft", false)
		data, err := createPullRequest(ctx, owner, repo, title, head, base, body, draft)
		return resultOrError(data, err, getFormat(req), createPullRequestMD), nil
	})

	s.AddTool(mcp.NewTool("github_merge_pull_request",
		mcp.WithDescription("Merge a pull request. Irreversible via this tool (though the merge commit can be reverted separately)."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Pull request number.")),
		mcp.WithString("commit_message", mcp.Description("Extra detail appended to the automatic merge commit message.")),
		mcp.WithString("merge_method", mcp.Enum("merge", "squash", "rebase"), mcp.DefaultString("merge"), mcp.Description("Merge strategy.")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(true), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("owner")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo, err := req.RequireString("repo")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		number, err := req.RequireInt("number")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		commitMessage := req.GetString("commit_message", "")
		mergeMethod := req.GetString("merge_method", "merge")
		data, err := mergePullRequest(ctx, owner, repo, number, commitMessage, mergeMethod)
		return resultOrError(data, err, getFormat(req), mergePullRequestMD), nil
	})
}

func registerSearchTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("github_search_repositories",
		mcp.WithDescription("Search GitHub repositories using GitHub's search qualifiers, e.g. \"language:go stars:>1000 topic:mcp\". Paginated via page/per_page."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query with optional qualifiers (language:, stars:, org:, topic:, etc.).")),
		mcp.WithString("sort", mcp.Enum("stars", "forks", "help-wanted-issues", "updated"), mcp.Description("Sort field. Defaults to best-match relevance.")),
		mcp.WithString("order", mcp.Enum("asc", "desc"), mcp.Description("Sort order.")),
		mcp.WithNumber("page", mcp.DefaultNumber(1), mcp.Description("Page number, 1-based.")),
		mcp.WithNumber("per_page", mcp.DefaultNumber(30), mcp.Min(1), mcp.Max(100), mcp.Description("Results per page (max 100).")),
		fmtParam(),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sort := req.GetString("sort", "")
		order := req.GetString("order", "")
		page := req.GetInt("page", 1)
		perPage := req.GetInt("per_page", defaultPerPage)
		data, err := searchRepositories(ctx, query, sort, order, page, perPage)
		return resultOrError(data, err, getFormat(req), searchRepositoriesMD), nil
	})
}
