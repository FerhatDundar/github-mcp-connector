package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/go-github/v79/github"
)

const (
	defaultPerPage = 30
	maxPerPage     = 100
	// defaultFileGetLimit caps how many bytes github_get_file_contents
	// returns inline by default, to avoid dumping huge files into a tool
	// response. Callers can raise it up to hardFileGetLimit via max_bytes.
	defaultFileGetLimit = 200_000
	hardFileGetLimit    = 5_000_000
	// fileWriteLimit caps how much content github_create_or_update_file
	// will accept in one call — this is a tool for small config/text
	// files, not bulk upload.
	fileWriteLimit = 5_000_000
)

func clampPerPage(v int) int {
	if v < 1 {
		return defaultPerPage
	}
	if v > maxPerPage {
		return maxPerPage
	}
	return v
}

// ---------------------------------------------------------------------
// Core operations — each returns a map[string]any for uniform rendering.
// ---------------------------------------------------------------------

func whoami(ctx context.Context) (map[string]any, error) {
	u, _, err := ghClient.Users.Get(ctx, "")
	if err != nil {
		return nil, friendlyGitHubError("get authenticated user", err)
	}
	return map[string]any{
		"login":        u.GetLogin(),
		"name":         u.GetName(),
		"html_url":     u.GetHTMLURL(),
		"company":      u.GetCompany(),
		"email":        u.GetEmail(),
		"public_repos": u.GetPublicRepos(),
		"followers":    u.GetFollowers(),
		"following":    u.GetFollowing(),
	}, nil
}

func whoamiMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Authenticated as `%v`\n\n", d["login"])
	fmt.Fprintf(&b, "- **Name**: %v\n", d["name"])
	fmt.Fprintf(&b, "- **Profile**: %v\n", d["html_url"])
	fmt.Fprintf(&b, "- **Public repos**: %v\n", d["public_repos"])
	fmt.Fprintf(&b, "- **Followers**: %v, **Following**: %v\n", d["followers"], d["following"])
	return b.String()
}

func listRepos(ctx context.Context, owner, repoType, sort, direction string, page, perPage int) (map[string]any, error) {
	listOpts := github.ListOptions{Page: page, PerPage: clampPerPage(perPage)}

	var (
		repos []*github.Repository
		resp  *github.Response
		err   error
	)
	if strings.TrimSpace(owner) == "" {
		repos, resp, err = ghClient.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{
			Type: repoType, Sort: sort, Direction: direction, ListOptions: listOpts,
		})
	} else {
		repos, resp, err = ghClient.Repositories.ListByUser(ctx, owner, &github.RepositoryListByUserOptions{
			Type: repoType, Sort: sort, Direction: direction, ListOptions: listOpts,
		})
	}
	if err != nil {
		return nil, friendlyGitHubError("list repositories", err)
	}
	return reposResult(repos, resp), nil
}

func reposResult(repos []*github.Repository, resp *github.Response) map[string]any {
	out := make([]map[string]any, 0, len(repos))
	for _, r := range repos {
		out = append(out, map[string]any{
			"full_name":      r.GetFullName(),
			"description":    r.GetDescription(),
			"private":        r.GetPrivate(),
			"fork":           r.GetFork(),
			"default_branch": r.GetDefaultBranch(),
			"stargazers":     r.GetStargazersCount(),
			"language":       r.GetLanguage(),
			"html_url":       r.GetHTMLURL(),
		})
	}
	result := map[string]any{"count": len(out), "repositories": out, "is_truncated": resp != nil && resp.NextPage != 0}
	if resp != nil && resp.NextPage != 0 {
		result["next_page"] = resp.NextPage
	}
	return result
}

func listReposMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Repositories\n\n%v found", d["count"])
	if t, _ := d["is_truncated"].(bool); t {
		fmt.Fprintf(&b, " (more available — pass `page=%v`)", d["next_page"])
	}
	b.WriteString(".\n\n")
	repos, _ := d["repositories"].([]map[string]any)
	if len(repos) == 0 {
		b.WriteString("_No repositories found._\n")
		return b.String()
	}
	b.WriteString("| Repo | Private | Default branch | Stars | Language |\n|------|---------|-----------------|-------|----------|\n")
	for _, r := range repos {
		fmt.Fprintf(&b, "| [%v](%v) | %v | %v | %v | %v |\n", r["full_name"], r["html_url"], r["private"], r["default_branch"], r["stargazers"], r["language"])
	}
	return b.String()
}

func getRepo(ctx context.Context, owner, repo string) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	r, _, err := ghClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("get repository %s/%s", owner, repo), err)
	}
	return map[string]any{
		"full_name":      r.GetFullName(),
		"description":    r.GetDescription(),
		"private":        r.GetPrivate(),
		"fork":           r.GetFork(),
		"default_branch": r.GetDefaultBranch(),
		"stargazers":     r.GetStargazersCount(),
		"forks":          r.GetForksCount(),
		"open_issues":    r.GetOpenIssuesCount(),
		"language":       r.GetLanguage(),
		"html_url":       r.GetHTMLURL(),
		"clone_url":      r.GetCloneURL(),
		"archived":       r.GetArchived(),
	}, nil
}

func getRepoMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %v\n\n%v\n\n", d["full_name"], d["description"])
	fmt.Fprintf(&b, "- **URL**: %v\n", d["html_url"])
	fmt.Fprintf(&b, "- **Default branch**: %v\n", d["default_branch"])
	fmt.Fprintf(&b, "- **Private**: %v, **Fork**: %v, **Archived**: %v\n", d["private"], d["fork"], d["archived"])
	fmt.Fprintf(&b, "- **Stars**: %v, **Forks**: %v, **Open issues**: %v\n", d["stargazers"], d["forks"], d["open_issues"])
	fmt.Fprintf(&b, "- **Language**: %v\n", d["language"])
	fmt.Fprintf(&b, "- **Clone URL**: %v\n", d["clone_url"])
	return b.String()
}

func createRepo(ctx context.Context, org, name, description string, private, autoInit bool) (map[string]any, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	repoReq := &github.Repository{
		Name:        github.Ptr(name),
		Description: github.Ptr(description),
		Private:     github.Ptr(private),
		AutoInit:    github.Ptr(autoInit),
	}
	r, _, err := ghClient.Repositories.Create(ctx, org, repoReq)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("create repository %q", name), err)
	}
	return map[string]any{
		"full_name": r.GetFullName(),
		"html_url":  r.GetHTMLURL(),
		"private":   r.GetPrivate(),
		"created":   true,
	}, nil
}

func createRepoMD(d map[string]any) string {
	return fmt.Sprintf("Created repository [%v](%v) (private=%v).\n", d["full_name"], d["html_url"], d["private"])
}

func listBranches(ctx context.Context, owner, repo string, page, perPage int) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	opts := &github.BranchListOptions{ListOptions: github.ListOptions{Page: page, PerPage: clampPerPage(perPage)}}
	branches, resp, err := ghClient.Repositories.ListBranches(ctx, owner, repo, opts)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("list branches for %s/%s", owner, repo), err)
	}
	out := make([]map[string]any, 0, len(branches))
	for _, br := range branches {
		out = append(out, map[string]any{
			"name":      br.GetName(),
			"protected": br.GetProtected(),
			"sha":       br.GetCommit().GetSHA(),
		})
	}
	result := map[string]any{"owner": owner, "repo": repo, "count": len(out), "branches": out, "is_truncated": resp != nil && resp.NextPage != 0}
	if resp != nil && resp.NextPage != 0 {
		result["next_page"] = resp.NextPage
	}
	return result, nil
}

func listBranchesMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Branches in `%v/%v`\n\n%v found.\n\n", d["owner"], d["repo"], d["count"])
	branches, _ := d["branches"].([]map[string]any)
	b.WriteString("| Name | Protected | SHA |\n|------|-----------|-----|\n")
	for _, br := range branches {
		fmt.Fprintf(&b, "| %v | %v | %v |\n", br["name"], br["protected"], br["sha"])
	}
	return b.String()
}

func listCommits(ctx context.Context, owner, repo, sha, path string, page, perPage int) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	opts := &github.CommitsListOptions{
		SHA:         sha,
		Path:        path,
		ListOptions: github.ListOptions{Page: page, PerPage: clampPerPage(perPage)},
	}
	commits, resp, err := ghClient.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("list commits for %s/%s", owner, repo), err)
	}
	out := make([]map[string]any, 0, len(commits))
	for _, c := range commits {
		out = append(out, map[string]any{
			"sha":      c.GetSHA(),
			"message":  firstLine(c.GetCommit().GetMessage()),
			"author":   c.GetCommit().GetAuthor().GetName(),
			"date":     c.GetCommit().GetAuthor().GetDate().Format("2006-01-02T15:04:05Z07:00"),
			"html_url": c.GetHTMLURL(),
		})
	}
	result := map[string]any{"owner": owner, "repo": repo, "count": len(out), "commits": out, "is_truncated": resp != nil && resp.NextPage != 0}
	if resp != nil && resp.NextPage != 0 {
		result["next_page"] = resp.NextPage
	}
	return result, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func listCommitsMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Commits in `%v/%v`\n\n%v found.\n\n", d["owner"], d["repo"], d["count"])
	commits, _ := d["commits"].([]map[string]any)
	b.WriteString("| SHA | Message | Author | Date |\n|-----|---------|--------|------|\n")
	for _, c := range commits {
		sha, _ := c["sha"].(string)
		if len(sha) > 7 {
			sha = sha[:7]
		}
		fmt.Fprintf(&b, "| [%v](%v) | %v | %v | %v |\n", sha, c["html_url"], c["message"], c["author"], c["date"])
	}
	return b.String()
}

func getFileContents(ctx context.Context, owner, repo, path, ref string, maxBytes int64) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" || strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("owner, repo, and path are all required")
	}
	if maxBytes <= 0 {
		maxBytes = defaultFileGetLimit
	}
	if maxBytes > hardFileGetLimit {
		maxBytes = hardFileGetLimit
	}
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	fileContent, dirContent, _, err := ghClient.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("get contents of %q in %s/%s", path, owner, repo), err)
	}

	if fileContent == nil {
		// path is a directory — return its listing instead of file content.
		entries := make([]map[string]any, 0, len(dirContent))
		for _, e := range dirContent {
			entries = append(entries, map[string]any{
				"name": e.GetName(),
				"path": e.GetPath(),
				"type": e.GetType(),
				"size": e.GetSize(),
				"sha":  e.GetSHA(),
			})
		}
		return map[string]any{
			"owner": owner, "repo": repo, "path": path, "is_directory": true,
			"count": len(entries), "entries": entries,
		}, nil
	}

	decoded, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("decoding contents of %q: %w", path, err)
	}
	data := []byte(decoded)
	totalSize := len(data)
	truncated := int64(len(data)) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}

	result := map[string]any{
		"owner": owner, "repo": repo, "path": path, "is_directory": false,
		"sha": fileContent.GetSHA(), "total_size": totalSize, "returned_size": len(data),
		"truncated": truncated,
	}
	if truncated {
		result["truncated_hint"] = "File is larger than the byte limit returned. Raise max_bytes (hard cap 5,000,000) to fetch more."
	}
	if utf8.Valid(data) {
		result["encoding"] = "text"
		result["content"] = string(data)
	} else {
		result["encoding"] = "base64"
		result["content"] = base64.StdEncoding.EncodeToString(data)
	}
	return result, nil
}

func getFileContentsMD(d map[string]any) string {
	var b strings.Builder
	if isDir, _ := d["is_directory"].(bool); isDir {
		fmt.Fprintf(&b, "# Directory `%v` in `%v/%v`\n\n%v entries.\n\n", d["path"], d["owner"], d["repo"], d["count"])
		entries, _ := d["entries"].([]map[string]any)
		b.WriteString("| Name | Type | Size |\n|------|------|------|\n")
		for _, e := range entries {
			fmt.Fprintf(&b, "| %v | %v | %v |\n", e["name"], e["type"], e["size"])
		}
		return b.String()
	}
	fmt.Fprintf(&b, "# `%v` in `%v/%v`\n\n", d["path"], d["owner"], d["repo"])
	fmt.Fprintf(&b, "- **SHA**: %v\n- **Size**: %v bytes (returned %v)\n- **Encoding**: %v\n", d["sha"], d["total_size"], d["returned_size"], d["encoding"])
	if hint, ok := d["truncated_hint"].(string); ok {
		fmt.Fprintf(&b, "- **Note**: %s\n", hint)
	}
	b.WriteString("\n---\n\n")
	fmt.Fprintf(&b, "%v\n", d["content"])
	return b.String()
}

func createOrUpdateFile(ctx context.Context, owner, repo, path, content, message, branch, sha string, base64Encoded bool) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" || strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("owner, repo, and path are all required")
	}
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("message (the commit message) is required")
	}
	if len(content) > fileWriteLimit {
		return nil, fmt.Errorf("content is %d bytes; this tool's limit is %d bytes for a single call", len(content), fileWriteLimit)
	}
	var body []byte
	if base64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("content is not valid base64: %w", err)
		}
		body = decoded
	} else {
		body = []byte(content)
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.Ptr(message),
		Content: body,
	}
	if strings.TrimSpace(branch) != "" {
		opts.Branch = github.Ptr(branch)
	}

	var (
		resp *github.RepositoryContentResponse
		err  error
	)
	if strings.TrimSpace(sha) != "" {
		opts.SHA = github.Ptr(sha)
		resp, _, err = ghClient.Repositories.UpdateFile(ctx, owner, repo, path, opts)
	} else {
		resp, _, err = ghClient.Repositories.CreateFile(ctx, owner, repo, path, opts)
	}
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("write %q in %s/%s", path, owner, repo), err)
	}
	return map[string]any{
		"owner": owner, "repo": repo, "path": path,
		"sha":        resp.GetContent().GetSHA(),
		"commit_sha": resp.GetSHA(),
		"html_url":   resp.GetContent().GetHTMLURL(),
	}, nil
}

func createOrUpdateFileMD(d map[string]any) string {
	return fmt.Sprintf("Wrote [`%v`](%v) in `%v/%v` (blob sha `%v`, commit `%v`).\n", d["path"], d["html_url"], d["owner"], d["repo"], d["sha"], d["commit_sha"])
}

func listIssues(ctx context.Context, owner, repo, state string, labels []string, page, perPage int) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	opts := &github.IssueListByRepoOptions{
		State:       state,
		Labels:      labels,
		ListOptions: github.ListOptions{Page: page, PerPage: clampPerPage(perPage)},
	}
	issues, resp, err := ghClient.Issues.ListByRepo(ctx, owner, repo, opts)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("list issues for %s/%s", owner, repo), err)
	}
	out := make([]map[string]any, 0, len(issues))
	for _, i := range issues {
		if i.IsPullRequest() {
			continue // the issues API includes PRs; keep this tool issues-only
		}
		out = append(out, issueSummary(i))
	}
	result := map[string]any{"owner": owner, "repo": repo, "count": len(out), "issues": out, "is_truncated": resp != nil && resp.NextPage != 0}
	if resp != nil && resp.NextPage != 0 {
		result["next_page"] = resp.NextPage
	}
	return result, nil
}

func issueSummary(i *github.Issue) map[string]any {
	labels := make([]string, 0, len(i.Labels))
	for _, l := range i.Labels {
		labels = append(labels, l.GetName())
	}
	return map[string]any{
		"number":   i.GetNumber(),
		"title":    i.GetTitle(),
		"state":    i.GetState(),
		"user":     i.GetUser().GetLogin(),
		"labels":   labels,
		"comments": i.GetComments(),
		"html_url": i.GetHTMLURL(),
	}
}

func listIssuesMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Issues in `%v/%v`\n\n%v found.\n\n", d["owner"], d["repo"], d["count"])
	issues, _ := d["issues"].([]map[string]any)
	b.WriteString("| # | Title | State | Author | Comments |\n|---|-------|-------|--------|----------|\n")
	for _, i := range issues {
		fmt.Fprintf(&b, "| [%v](%v) | %v | %v | %v | %v |\n", i["number"], i["html_url"], i["title"], i["state"], i["user"], i["comments"])
	}
	return b.String()
}

func getIssue(ctx context.Context, owner, repo string, number int) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	i, _, err := ghClient.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("get issue #%d in %s/%s", number, owner, repo), err)
	}
	d := issueSummary(i)
	d["body"] = i.GetBody()
	return d, nil
}

func getIssueMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Issue #%v: %v\n\n", d["number"], d["title"])
	fmt.Fprintf(&b, "- **State**: %v\n- **Author**: %v\n- **Labels**: %v\n- **Comments**: %v\n- **URL**: %v\n\n", d["state"], d["user"], d["labels"], d["comments"], d["html_url"])
	fmt.Fprintf(&b, "---\n\n%v\n", d["body"])
	return b.String()
}

func createIssue(ctx context.Context, owner, repo, title, body string, labels, assignees []string) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	req := &github.IssueRequest{Title: github.Ptr(title)}
	if body != "" {
		req.Body = github.Ptr(body)
	}
	if len(labels) > 0 {
		req.Labels = &labels
	}
	if len(assignees) > 0 {
		req.Assignees = &assignees
	}
	i, _, err := ghClient.Issues.Create(ctx, owner, repo, req)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("create issue in %s/%s", owner, repo), err)
	}
	return issueSummary(i), nil
}

func createIssueMD(d map[string]any) string {
	return fmt.Sprintf("Created issue [#%v: %v](%v).\n", d["number"], d["title"], d["html_url"])
}

func updateIssue(ctx context.Context, owner, repo string, number int, title, body, state string, labels []string) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	req := &github.IssueRequest{}
	if title != "" {
		req.Title = github.Ptr(title)
	}
	if body != "" {
		req.Body = github.Ptr(body)
	}
	if state != "" {
		req.State = github.Ptr(state)
	}
	if labels != nil {
		req.Labels = &labels
	}
	i, _, err := ghClient.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("update issue #%d in %s/%s", number, owner, repo), err)
	}
	return issueSummary(i), nil
}

func updateIssueMD(d map[string]any) string {
	return fmt.Sprintf("Updated issue [#%v: %v](%v) — state: %v.\n", d["number"], d["title"], d["html_url"], d["state"])
}

func addIssueComment(ctx context.Context, owner, repo string, number int, body string) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("body is required")
	}
	c, _, err := ghClient.Issues.CreateComment(ctx, owner, repo, number, &github.IssueComment{Body: github.Ptr(body)})
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("comment on issue #%d in %s/%s", number, owner, repo), err)
	}
	return map[string]any{
		"issue_number": number, "comment_id": c.GetID(), "html_url": c.GetHTMLURL(), "user": c.GetUser().GetLogin(),
	}, nil
}

func addIssueCommentMD(d map[string]any) string {
	return fmt.Sprintf("Added [comment](%v) to issue #%v.\n", d["html_url"], d["issue_number"])
}

func listPullRequests(ctx context.Context, owner, repo, state string, page, perPage int) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	opts := &github.PullRequestListOptions{
		State:       state,
		ListOptions: github.ListOptions{Page: page, PerPage: clampPerPage(perPage)},
	}
	prs, resp, err := ghClient.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("list pull requests for %s/%s", owner, repo), err)
	}
	out := make([]map[string]any, 0, len(prs))
	for _, p := range prs {
		out = append(out, pullRequestSummary(p))
	}
	result := map[string]any{"owner": owner, "repo": repo, "count": len(out), "pull_requests": out, "is_truncated": resp != nil && resp.NextPage != 0}
	if resp != nil && resp.NextPage != 0 {
		result["next_page"] = resp.NextPage
	}
	return result, nil
}

func pullRequestSummary(p *github.PullRequest) map[string]any {
	return map[string]any{
		"number":   p.GetNumber(),
		"title":    p.GetTitle(),
		"state":    p.GetState(),
		"user":     p.GetUser().GetLogin(),
		"head":     p.GetHead().GetRef(),
		"base":     p.GetBase().GetRef(),
		"draft":    p.GetDraft(),
		"merged":   p.GetMerged(),
		"html_url": p.GetHTMLURL(),
	}
}

func listPullRequestsMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Pull requests in `%v/%v`\n\n%v found.\n\n", d["owner"], d["repo"], d["count"])
	prs, _ := d["pull_requests"].([]map[string]any)
	b.WriteString("| # | Title | State | Head → Base | Author |\n|---|-------|-------|--------------|--------|\n")
	for _, p := range prs {
		fmt.Fprintf(&b, "| [%v](%v) | %v | %v | %v → %v | %v |\n", p["number"], p["html_url"], p["title"], p["state"], p["head"], p["base"], p["user"])
	}
	return b.String()
}

func getPullRequest(ctx context.Context, owner, repo string, number int) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	p, _, err := ghClient.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("get pull request #%d in %s/%s", number, owner, repo), err)
	}
	d := pullRequestSummary(p)
	d["body"] = p.GetBody()
	d["mergeable"] = p.GetMergeable()
	d["additions"] = p.GetAdditions()
	d["deletions"] = p.GetDeletions()
	d["changed_files"] = p.GetChangedFiles()
	return d, nil
}

func getPullRequestMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# PR #%v: %v\n\n", d["number"], d["title"])
	fmt.Fprintf(&b, "- **State**: %v (merged: %v, draft: %v)\n", d["state"], d["merged"], d["draft"])
	fmt.Fprintf(&b, "- **%v → %v**, author %v\n", d["head"], d["base"], d["user"])
	fmt.Fprintf(&b, "- **Changes**: +%v/-%v across %v files\n", d["additions"], d["deletions"], d["changed_files"])
	fmt.Fprintf(&b, "- **Mergeable**: %v\n- **URL**: %v\n\n", d["mergeable"], d["html_url"])
	fmt.Fprintf(&b, "---\n\n%v\n", d["body"])
	return b.String()
}

func createPullRequest(ctx context.Context, owner, repo, title, head, base, body string, draft bool) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	if strings.TrimSpace(title) == "" || strings.TrimSpace(head) == "" || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("title, head, and base are all required")
	}
	req := &github.NewPullRequest{
		Title: github.Ptr(title),
		Head:  github.Ptr(head),
		Base:  github.Ptr(base),
		Draft: github.Ptr(draft),
	}
	if body != "" {
		req.Body = github.Ptr(body)
	}
	p, _, err := ghClient.PullRequests.Create(ctx, owner, repo, req)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("create pull request in %s/%s", owner, repo), err)
	}
	return pullRequestSummary(p), nil
}

func createPullRequestMD(d map[string]any) string {
	return fmt.Sprintf("Created PR [#%v: %v](%v) (%v → %v).\n", d["number"], d["title"], d["html_url"], d["head"], d["base"])
}

func mergePullRequest(ctx context.Context, owner, repo string, number int, commitMessage, mergeMethod string) (map[string]any, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("owner and repo are both required")
	}
	opts := &github.PullRequestOptions{MergeMethod: mergeMethod}
	res, _, err := ghClient.PullRequests.Merge(ctx, owner, repo, number, commitMessage, opts)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("merge pull request #%d in %s/%s", number, owner, repo), err)
	}
	return map[string]any{
		"number":  number,
		"merged":  res.GetMerged(),
		"sha":     res.GetSHA(),
		"message": res.GetMessage(),
	}, nil
}

func mergePullRequestMD(d map[string]any) string {
	return fmt.Sprintf("PR #%v merge: %v (sha `%v`) — %v\n", d["number"], d["merged"], d["sha"], d["message"])
}

func searchRepositories(ctx context.Context, query, sort, order string, page, perPage int) (map[string]any, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required, e.g. \"language:go stars:>1000\"")
	}
	opts := &github.SearchOptions{
		Sort:        sort,
		Order:       order,
		ListOptions: github.ListOptions{Page: page, PerPage: clampPerPage(perPage)},
	}
	res, resp, err := ghClient.Search.Repositories(ctx, query, opts)
	if err != nil {
		return nil, friendlyGitHubError(fmt.Sprintf("search repositories for %q", query), err)
	}
	result := reposResult(res.Repositories, resp)
	result["total_count"] = res.GetTotal()
	result["query"] = query
	return result, nil
}

func searchRepositoriesMD(d map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Search: `%v`\n\n%v total match(es), showing %v.\n\n", d["query"], d["total_count"], d["count"])
	repos, _ := d["repositories"].([]map[string]any)
	b.WriteString("| Repo | Stars | Language |\n|------|-------|----------|\n")
	for _, r := range repos {
		fmt.Fprintf(&b, "| [%v](%v) | %v | %v |\n", r["full_name"], r["html_url"], r["stargazers"], r["language"])
	}
	return b.String()
}
