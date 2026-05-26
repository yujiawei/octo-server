package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitHubStatusNode 调用 GitHub Statuses API 设置 commit status。
//
// config（所有字段引擎已经把 {{...}} 模板渲染过）:
//
//	token:       string   # 必填；personal access token / app installation token
//	repo:        string   # owner/repo
//	sha:         string   # 必填；commit SHA
//	state:       string   # pending / success / failure / error
//	context:     string   # status 名（默认 "default"）
//	description: string   # 简短描述
//	target_url:  string   # 详情页（可空）
//	api_base:    string   # 可选；默认 https://api.github.com
//
// output:
//
//	status: int           # GitHub 响应 HTTP 状态码（成功是 201）
//	state:  string        # 回写的 state（方便下游引用）
//	url:    string        # 调用的 API URL（不含 token）
//	body:   string        # 响应体原文（便于调试 / 失败排错）
type GitHubStatusNode struct {
	client *http.Client
}

// NewGitHubStatusNode 构造一个 github_status runner，传 nil 使用默认（10s 超时）的 http.Client。
func NewGitHubStatusNode(client *http.Client) *GitHubStatusNode {
	if client == nil {
		client = &http.Client{Timeout: githubStatusDefaultTimeout}
	}
	return &GitHubStatusNode{client: client}
}

// Type 返回 "github_status"
func (g *GitHubStatusNode) Type() string { return "github_status" }

const githubStatusDefaultTimeout = 10 * time.Second

// validGitHubStatusStates 是 GitHub Statuses API 接受的 state 集合。
var validGitHubStatusStates = map[string]bool{
	"pending": true,
	"success": true,
	"failure": true,
	"error":   true,
}

// Run 调用 POST /repos/{owner}/{repo}/statuses/{sha}
func (g *GitHubStatusNode) Run(ctx context.Context, cfg map[string]any) (*Result, error) {
	token, _ := cfg["token"].(string)
	if token == "" {
		return nil, errors.New("github_status node: token is required")
	}
	repo, _ := cfg["repo"].(string)
	if repo == "" {
		return nil, errors.New("github_status node: repo is required")
	}
	if !strings.Contains(repo, "/") {
		return nil, fmt.Errorf("github_status node: repo must be owner/repo, got %q", repo)
	}
	sha, _ := cfg["sha"].(string)
	if sha == "" {
		return nil, errors.New("github_status node: sha is required")
	}
	state, _ := cfg["state"].(string)
	if state == "" {
		return nil, errors.New("github_status node: state is required")
	}
	if !validGitHubStatusStates[state] {
		return nil, fmt.Errorf("github_status node: invalid state %q (must be pending/success/failure/error)", state)
	}

	apiBase, _ := cfg["api_base"].(string)
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	apiBase = strings.TrimRight(apiBase, "/")
	url := fmt.Sprintf("%s/repos/%s/statuses/%s", apiBase, repo, sha)

	payload := map[string]any{"state": state}
	if ctxName, ok := cfg["context"].(string); ok && ctxName != "" {
		payload["context"] = ctxName
	}
	if desc, ok := cfg["description"].(string); ok && desc != "" {
		payload["description"] = desc
	}
	if target, ok := cfg["target_url"].(string); ok && target != "" {
		payload["target_url"] = target
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("github_status node: marshal payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, githubStatusDefaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("github_status node: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github_status node: do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github_status node: read response: %w", err)
	}

	out := map[string]any{
		"status": resp.StatusCode,
		"state":  state,
		"url":    url,
		"body":   string(respBody),
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &Result{Output: out}, fmt.Errorf("github_status node: GitHub API returned %d: %s", resp.StatusCode, truncate(string(respBody), 256))
	}
	return &Result{Output: out}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
