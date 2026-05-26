package flow

import "testing"

// TestLookupHeader_CaseInsensitive 验证 PR#9 review bug 2：
// api.go 的 handleWebhook 用 http.Header 构建 headers map，key 是
// CanonicalMIMEHeaderKey 形式（如 X-Hub-Signature-256）。当用户在
// trigger config 里配小写 header 名时，service 层必须做规范化才能命中。
func TestLookupHeader_CaseInsensitive(t *testing.T) {
	// 模拟 api.go 构建的 headers map
	headers := map[string]string{
		"X-Hub-Signature-256": "sha256=abc",
		"Content-Type":        "application/json",
	}
	cases := []struct {
		name string
		key  string
		want string
	}{
		{"canonical", "X-Hub-Signature-256", "sha256=abc"},
		{"all-lowercase", "x-hub-signature-256", "sha256=abc"},
		{"all-uppercase", "X-HUB-SIGNATURE-256", "sha256=abc"},
		{"mixed", "x-Hub-signature-256", "sha256=abc"},
		{"empty-name", "", ""},
		{"missing", "X-Missing", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lookupHeader(headers, tc.key)
			if got != tc.want {
				t.Fatalf("lookupHeader(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

// 兜底：如果某个调用方传入未规范化的 map（例如手工构造），也能命中。
func TestLookupHeader_NonCanonicalMap(t *testing.T) {
	headers := map[string]string{"x-hub-signature-256": "sha256=xyz"}
	if got := lookupHeader(headers, "x-hub-signature-256"); got != "sha256=xyz" {
		t.Fatalf("got %q", got)
	}
}
