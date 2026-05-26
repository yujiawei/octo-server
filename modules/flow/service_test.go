package flow

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

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

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	eng := NewEngine(db, nil, nil)
	svc, err := NewService(db, eng, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc, mock
}

func TestValidateDefinitionTriggers(t *testing.T) {
	cases := []struct {
		name    string
		def     *Definition
		wantErr bool
	}{
		{"nil", nil, false},
		{"no triggers", &Definition{}, false},
		{
			name: "valid cron 5-field",
			def: &Definition{Triggers: []TriggerDef{
				{ID: "t1", Type: TriggerTypeCron, Config: map[string]any{"expression": "*/1 * * * *"}},
			}},
		},
		{
			name: "valid cron with timezone",
			def: &Definition{Triggers: []TriggerDef{
				{ID: "t1", Type: TriggerTypeCron, Config: map[string]any{"expression": "0 0 * * *", "timezone": "Asia/Shanghai"}},
			}},
		},
		{
			name: "invalid cron expression",
			def: &Definition{Triggers: []TriggerDef{
				{ID: "t1", Type: TriggerTypeCron, Config: map[string]any{"expression": "not-cron"}},
			}},
			wantErr: true,
		},
		{
			name: "missing expression",
			def: &Definition{Triggers: []TriggerDef{
				{ID: "t1", Type: TriggerTypeCron, Config: map[string]any{}},
			}},
			wantErr: true,
		},
		{
			name: "invalid timezone",
			def: &Definition{Triggers: []TriggerDef{
				{ID: "t1", Type: TriggerTypeCron, Config: map[string]any{"expression": "* * * * *", "timezone": "Mars/Olympus"}},
			}},
			wantErr: true,
		},
		{
			name: "webhook ok at this stage",
			def: &Definition{Triggers: []TriggerDef{
				{ID: "t1", Type: TriggerTypeWebhook, Config: map[string]any{"path": "/hooks/foo"}},
			}},
		},
		{
			name: "unsupported type",
			def: &Definition{Triggers: []TriggerDef{
				{ID: "t1", Type: "telegram", Config: map[string]any{}},
			}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDefinitionTriggers(tc.def)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestService_NextTriggerAt_NoCron(t *testing.T) {
	svc, mock := newTestService(t)
	defer svc.Stop()
	// 列出 flow 的触发器 → 空
	mock.ExpectQuery(".*flow_triggers.*").
		WillReturnRows(sqlmock.NewRows([]string{}))
	if got := svc.NextTriggerAt("flow1"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
