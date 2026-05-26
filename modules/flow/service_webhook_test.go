package flow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Mininglamp-OSS/octo-server/modules/flow/nodes"
)

// TestService_HandleWebhookByFlowID 验证按 flow id 寻址的 webhook：
//   - flow active + 含 webhook trigger 时启动一次 execution
//   - request body / headers 正确写入 TriggerData，并能被 script 节点读取
func TestService_HandleWebhookByFlowID(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)

	// 这个 flow 跑一个 script 节点，把 trigger.payload.hello 当作输入再原样输出。
	def := &Definition{
		Triggers: []TriggerDef{
			{ID: "wh", Type: TriggerTypeWebhook, Config: map[string]any{"path": "demo"}},
		},
		Nodes: []NodeDef{
			{ID: "echo", Type: "script", Config: map[string]any{
				"code":  `return { got: input.msg, ua: input.ua };`,
				"input": map[string]any{"msg": "{{trigger.payload.hello}}", "ua": "{{trigger.headers.User-Agent}}"},
			}},
		},
	}
	defJSON, _ := json.Marshal(def)

	// GetFlow → 一行
	flowRows := sqlmock.NewRows([]string{
		"id", "space_id", "name", "description", "definition", "version",
		"status", "created_by", "created_at", "updated_at",
	}).AddRow("f-wh", "s1", "demo", "", string(defJSON), 1,
		FlowStatusActive, "u1", time.Now(), time.Now())
	mock.ExpectQuery(`(?i)SELECT \* FROM flows`).WillReturnRows(flowRows)

	// ListTriggersByFlow → 一条 webhook trigger
	cfgJSON, _ := json.Marshal(map[string]any{"path": "demo"})
	trigRows := sqlmock.NewRows([]string{
		"id", "flow_id", "type", "config", "webhook_path", "status", "created_at", "updated_at",
	}).AddRow("t-wh", "f-wh", TriggerTypeWebhook, string(cfgJSON), "demo", "active",
		time.Now(), time.Now())
	mock.ExpectQuery(`(?i)SELECT \* FROM flow_triggers`).WillReturnRows(trigRows)

	// 后续 engine 的所有写入一律放行
	for i := 0; i < 30; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	eng := NewEngine(db, nodes.DefaultRegistry(), nil)
	svc, err := NewService(db, eng, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	body := []byte(`{"hello":"world"}`)
	headers := map[string]string{"User-Agent": "test-suite", "Content-Type": "application/json"}
	exec, err := svc.HandleWebhookByFlowID(context.Background(), "f-wh", body, headers)
	if err != nil {
		t.Fatalf("HandleWebhookByFlowID: %v", err)
	}
	if exec == nil || exec.ID == "" {
		t.Fatalf("expected execution, got %+v", exec)
	}
	if exec.TriggerID != "t-wh" {
		t.Fatalf("expected trigger_id=t-wh, got %q", exec.TriggerID)
	}

	// 等异步跑完
	deadline := time.Now().Add(3 * time.Second)
	var final *ExecutionContext
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		ec, _ := exec.DecodeContext()
		if ec != nil && ec.Nodes["echo"].Status == NodeStatusSuccess {
			final = ec
			break
		}
	}
	if final == nil {
		ec, _ := exec.DecodeContext()
		t.Fatalf("execution did not succeed: status=%s nodes=%+v", exec.Status, ec)
	}
	if got := final.Nodes["echo"].Output["got"]; got != "world" {
		t.Fatalf("expected got=world, got %v", got)
	}
	if got := final.Nodes["echo"].Output["ua"]; got != "test-suite" {
		t.Fatalf("expected ua=test-suite, got %v", got)
	}

	// triggerData 中 payload + headers 都应保留
	if final.Trigger.Type != TriggerTypeWebhook {
		t.Fatalf("trigger.type=%q", final.Trigger.Type)
	}
	if final.Trigger.Payload["hello"] != "world" {
		t.Fatalf("trigger.payload=%+v", final.Trigger.Payload)
	}
	if final.Trigger.Headers["User-Agent"] != "test-suite" {
		t.Fatalf("trigger.headers=%+v", final.Trigger.Headers)
	}
}

// TestService_HandleWebhookByFlowID_NotActive 验证 flow 非 active 时拒绝
func TestService_HandleWebhookByFlowID_NotActive(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)

	def := &Definition{
		Triggers: []TriggerDef{{ID: "wh", Type: TriggerTypeWebhook, Config: map[string]any{"path": "demo"}}},
		Nodes:    []NodeDef{{ID: "n", Type: "script", Config: map[string]any{"code": `return {};`}}},
	}
	defJSON, _ := json.Marshal(def)
	mock.ExpectQuery(`(?i)SELECT \* FROM flows`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "space_id", "name", "description", "definition", "version",
			"status", "created_by", "created_at", "updated_at",
		}).AddRow("f1", "", "", "", string(defJSON), 1,
			FlowStatusDraft, "", time.Now(), time.Now()),
	)

	eng := NewEngine(db, nodes.DefaultRegistry(), nil)
	svc, _ := NewService(db, eng, nil)
	_, err := svc.HandleWebhookByFlowID(context.Background(), "f1", []byte(`{}`), nil)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected not active error, got %v", err)
	}
}

// TestService_HandleWebhookByFlowID_NoWebhookTrigger 验证 flow 不含 webhook trigger 时拒绝
func TestService_HandleWebhookByFlowID_NoWebhookTrigger(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)

	def := &Definition{
		Triggers: []TriggerDef{{ID: "m", Type: TriggerTypeManual}},
		Nodes:    []NodeDef{{ID: "n", Type: "script", Config: map[string]any{"code": `return {};`}}},
	}
	defJSON, _ := json.Marshal(def)
	mock.ExpectQuery(`(?i)SELECT \* FROM flows`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "space_id", "name", "description", "definition", "version",
			"status", "created_by", "created_at", "updated_at",
		}).AddRow("f2", "", "", "", string(defJSON), 1,
			FlowStatusActive, "", time.Now(), time.Now()),
	)

	eng := NewEngine(db, nodes.DefaultRegistry(), nil)
	svc, _ := NewService(db, eng, nil)
	_, err := svc.HandleWebhookByFlowID(context.Background(), "f2", []byte(`{}`), nil)
	if err == nil || !strings.Contains(err.Error(), "no webhook trigger") {
		t.Fatalf("expected no webhook trigger error, got %v", err)
	}
}

// TestService_HasWebhookTrigger 简单检查 helper
func TestService_HasWebhookTrigger(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	def := &Definition{
		Triggers: []TriggerDef{{ID: "wh", Type: TriggerTypeWebhook}},
	}
	defJSON, _ := json.Marshal(def)
	mock.ExpectQuery(`(?i)SELECT \* FROM flows`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "space_id", "name", "description", "definition", "version",
			"status", "created_by", "created_at", "updated_at",
		}).AddRow("f3", "", "", "", string(defJSON), 1,
			FlowStatusActive, "", time.Now(), time.Now()),
	)
	eng := NewEngine(db, nodes.DefaultRegistry(), nil)
	svc, _ := NewService(db, eng, nil)
	ok, err := svc.HasWebhookTrigger("f3")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok {
		t.Fatalf("expected true")
	}
}
