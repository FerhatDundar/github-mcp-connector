package main

import (
	"context"
	"strings"
	"testing"
)

func TestFriendlyGitHubErrorNil(t *testing.T) {
	if err := friendlyGitHubError("do a thing", nil); err != nil {
		t.Errorf("expected nil error to stay nil, got %v", err)
	}
}

func TestFriendlyGitHubErrorWrapsMessage(t *testing.T) {
	base := errTest("boom")
	err := friendlyGitHubError("get repository \"x/y\"", base)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected wrapped error to retain original message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "get repository") {
		t.Errorf("expected wrapped error to include the action, got: %v", err)
	}
}

// errTest is a minimal error type so tests don't need to construct real
// go-github SDK error values for the plain-message fallback path.
type errTest string

func (e errTest) Error() string { return string(e) }

// The following tests exercise the input-validation guards on each core
// operation. All of them return before touching the (nil) ghClient, so
// they run without any network access or a real GitHub token.

func TestGetRepoRequiresOwnerAndRepo(t *testing.T) {
	if _, err := getRepo(context.Background(), "", "repo"); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := getRepo(context.Background(), "owner", ""); err == nil {
		t.Error("expected error for empty repo")
	}
}

func TestCreateRepoRequiresName(t *testing.T) {
	if _, err := createRepo(context.Background(), "", "", "desc", false, true); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestListBranchesRequiresOwnerAndRepo(t *testing.T) {
	if _, err := listBranches(context.Background(), "", "repo", 1, 30); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := listBranches(context.Background(), "owner", "", 1, 30); err == nil {
		t.Error("expected error for empty repo")
	}
}

func TestListCommitsRequiresOwnerAndRepo(t *testing.T) {
	if _, err := listCommits(context.Background(), "", "repo", "", "", 1, 30); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := listCommits(context.Background(), "owner", "", "", "", 1, 30); err == nil {
		t.Error("expected error for empty repo")
	}
}

func TestGetFileContentsRequiresOwnerRepoPath(t *testing.T) {
	ctx := context.Background()
	if _, err := getFileContents(ctx, "", "repo", "path", "", 0); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := getFileContents(ctx, "owner", "", "path", "", 0); err == nil {
		t.Error("expected error for empty repo")
	}
	if _, err := getFileContents(ctx, "owner", "repo", "", "", 0); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestCreateOrUpdateFileValidation(t *testing.T) {
	ctx := context.Background()
	if _, err := createOrUpdateFile(ctx, "", "repo", "path", "content", "msg", "", "", false); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := createOrUpdateFile(ctx, "owner", "repo", "path", "content", "", "", "", false); err == nil {
		t.Error("expected error for empty commit message")
	}
	huge := strings.Repeat("a", fileWriteLimit+1)
	if _, err := createOrUpdateFile(ctx, "owner", "repo", "path", huge, "msg", "", "", false); err == nil {
		t.Error("expected error for content over the size limit")
	}
	if _, err := createOrUpdateFile(ctx, "owner", "repo", "path", "not-base64!!", "msg", "", "", true); err == nil {
		t.Error("expected error for invalid base64 content when base64=true")
	}
}

func TestListIssuesRequiresOwnerAndRepo(t *testing.T) {
	if _, err := listIssues(context.Background(), "", "repo", "open", nil, 1, 30); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := listIssues(context.Background(), "owner", "", "open", nil, 1, 30); err == nil {
		t.Error("expected error for empty repo")
	}
}

func TestCreateIssueValidation(t *testing.T) {
	ctx := context.Background()
	if _, err := createIssue(ctx, "", "repo", "title", "", nil, nil); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := createIssue(ctx, "owner", "repo", "", "", nil, nil); err == nil {
		t.Error("expected error for empty title")
	}
}

func TestAddIssueCommentValidation(t *testing.T) {
	ctx := context.Background()
	if _, err := addIssueComment(ctx, "", "repo", 1, "body"); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := addIssueComment(ctx, "owner", "repo", 1, ""); err == nil {
		t.Error("expected error for empty body")
	}
}

func TestListPullRequestsRequiresOwnerAndRepo(t *testing.T) {
	if _, err := listPullRequests(context.Background(), "", "repo", "open", 1, 30); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := listPullRequests(context.Background(), "owner", "", "open", 1, 30); err == nil {
		t.Error("expected error for empty repo")
	}
}

func TestCreatePullRequestValidation(t *testing.T) {
	ctx := context.Background()
	if _, err := createPullRequest(ctx, "", "repo", "title", "head", "base", "", false); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := createPullRequest(ctx, "owner", "repo", "", "head", "base", "", false); err == nil {
		t.Error("expected error for empty title")
	}
	if _, err := createPullRequest(ctx, "owner", "repo", "title", "", "base", "", false); err == nil {
		t.Error("expected error for empty head")
	}
	if _, err := createPullRequest(ctx, "owner", "repo", "title", "head", "", "", false); err == nil {
		t.Error("expected error for empty base")
	}
}

func TestSearchRepositoriesRequiresQuery(t *testing.T) {
	if _, err := searchRepositories(context.Background(), "", "", "", 1, 30); err == nil {
		t.Error("expected error for empty query")
	}
}

func TestClampPerPage(t *testing.T) {
	if got := clampPerPage(0); got != defaultPerPage {
		t.Errorf("clampPerPage(0) = %d, want default %d", got, defaultPerPage)
	}
	if got := clampPerPage(500); got != maxPerPage {
		t.Errorf("clampPerPage(500) = %d, want capped %d", got, maxPerPage)
	}
	if got := clampPerPage(50); got != 50 {
		t.Errorf("clampPerPage(50) = %d, want 50", got)
	}
}

func TestRenderJSONRoundTrips(t *testing.T) {
	out := renderJSON(map[string]any{"repo": "x", "count": 3})
	if !strings.Contains(out, "\"repo\": \"x\"") {
		t.Errorf("expected rendered JSON to contain repo field, got: %s", out)
	}
}
