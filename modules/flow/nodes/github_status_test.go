package nodes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubStatusNode_Success(t *testing.T) {
	var (
		gotPath   string
		gotMethod string
		gotAuth   string
		gotAccept string
		gotBody   map[string]any
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_, _ = io.WriteString(w, `{"id":1,"state":"pending"}`)
	}))
	defer srv.Close()

	n := NewGitHubStatusNode(srv.Client())
	res, err := n.Run(context.Background(), map[string]any{
		"token":       "ghp_xxx",
		"repo":        "owner/repo",
		"sha":         "abc123",
		"state":       "pending",
		"context":     "code-review",
		"description": "ReviewBot reviewing...",
		"target_url":  "https://example.com/run/1",
		"api_base":    srv.URL,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method: %s", gotMethod)
	}
	if gotPath != "/repos/owner/repo/statuses/abc123" {
		t.Fatalf("path: %s", gotPath)
	}
	if gotAuth != "Bearer ghp_xxx" {
		t.Fatalf("auth: %s", gotAuth)
	}
	if !strings.Contains(gotAccept, "application/vnd.github") {
		t.Fatalf("accept: %s", gotAccept)
	}
	if gotBody["state"] != "pending" {
		t.Fatalf("body state: %#v", gotBody)
	}
	if gotBody["context"] != "code-review" {
		t.Fatalf("body context: %#v", gotBody)
	}
	if gotBody["description"] != "ReviewBot reviewing..." {
		t.Fatalf("body description: %#v", gotBody)
	}
	if gotBody["target_url"] != "https://example.com/run/1" {
		t.Fatalf("body target_url: %#v", gotBody)
	}
	if res.Output["status"].(int) != 201 {
		t.Fatalf("output status: %v", res.Output["status"])
	}
	if res.Output["state"].(string) != "pending" {
		t.Fatalf("output state: %v", res.Output["state"])
	}
}

func TestGitHubStatusNode_OmitEmptyOptionalFields(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(201)
	}))
	defer srv.Close()

	n := NewGitHubStatusNode(srv.Client())
	_, err := n.Run(context.Background(), map[string]any{
		"token":    "ghp_xxx",
		"repo":     "owner/repo",
		"sha":      "abc",
		"state":    "success",
		"api_base": srv.URL,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got["state"] != "success" {
		t.Fatalf("state: %#v", got)
	}
	if _, has := got["context"]; has {
		t.Fatalf("context should be omitted when empty: %#v", got)
	}
	if _, has := got["description"]; has {
		t.Fatalf("description should be omitted when empty")
	}
	if _, has := got["target_url"]; has {
		t.Fatalf("target_url should be omitted when empty")
	}
}

func TestGitHubStatusNode_GitHubErrorReturnsErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	}))
	defer srv.Close()

	n := NewGitHubStatusNode(srv.Client())
	res, err := n.Run(context.Background(), map[string]any{
		"token":    "tok",
		"repo":     "o/r",
		"sha":      "deadbeef",
		"state":    "error",
		"api_base": srv.URL,
	})
	if err == nil {
		t.Fatalf("expected error on 404")
	}
	if res == nil {
		t.Fatalf("result should still be returned with status / body for diagnostics")
	}
	if res.Output["status"].(int) != 404 {
		t.Fatalf("status: %v", res.Output["status"])
	}
	if !strings.Contains(res.Output["body"].(string), "Not Found") {
		t.Fatalf("body: %v", res.Output["body"])
	}
}

func TestGitHubStatusNode_InvalidState(t *testing.T) {
	n := NewGitHubStatusNode(nil)
	_, err := n.Run(context.Background(), map[string]any{
		"token": "tok",
		"repo":  "o/r",
		"sha":   "abc",
		"state": "in_progress",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid state") {
		t.Fatalf("expected invalid state error, got %v", err)
	}
}

func TestGitHubStatusNode_RequiredFields(t *testing.T) {
	n := NewGitHubStatusNode(nil)
	cases := []struct {
		name string
		cfg  map[string]any
	}{
		{"missing token", map[string]any{"repo": "o/r", "sha": "x", "state": "pending"}},
		{"missing repo", map[string]any{"token": "t", "sha": "x", "state": "pending"}},
		{"missing sha", map[string]any{"token": "t", "repo": "o/r", "state": "pending"}},
		{"missing state", map[string]any{"token": "t", "repo": "o/r", "sha": "x"}},
		{"bad repo", map[string]any{"token": "t", "repo": "bad", "sha": "x", "state": "pending"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := n.Run(context.Background(), tc.cfg); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestGitHubStatusNode_AllValidStates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	}))
	defer srv.Close()
	n := NewGitHubStatusNode(srv.Client())
	for _, s := range []string{"pending", "success", "failure", "error"} {
		if _, err := n.Run(context.Background(), map[string]any{
			"token": "t", "repo": "o/r", "sha": "x", "state": s, "api_base": srv.URL,
		}); err != nil {
			t.Fatalf("state %s: %v", s, err)
		}
	}
}

func TestGitHubStatusNode_Type(t *testing.T) {
	if NewGitHubStatusNode(nil).Type() != "github_status" {
		t.Fatal("type should be github_status")
	}
}

func TestGitHubStatusNode_DefaultRegistryRegistered(t *testing.T) {
	r := DefaultRegistry()
	if _, ok := r.Get("shell"); !ok {
		t.Fatalf("shell node should be registered in DefaultRegistry")
	}
	if _, ok := r.Get("github_status"); !ok {
		t.Fatalf("github_status node should be registered in DefaultRegistry")
	}
}
