package flow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Mininglamp-OSS/octo-server/modules/flow/nodes"
	"github.com/gocraft/dbr/v2"
	"github.com/gocraft/dbr/v2/dialect"
)

func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	conn := &dbr.Connection{DB: rawDB, EventReceiver: &dbr.NullEventReceiver{}, Dialect: dialect.MySQL}
	sess := conn.NewSession(nil)
	return NewDB(sess), mock
}

func TestEngine_RunSimpleScriptFlow(t *testing.T) {
	db, mock := newMockDB(t)
	// 简单：所有 query/exec 都放行；不验证具体语句（QueryMatcherRegexp + ExpectAny 风格）
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	for i := 0; i < 20; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	eng := NewEngine(db, nodes.DefaultRegistry(), nil)

	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script", Config: map[string]any{
				"code":  `return { greeting: "hello " + input.who };`,
				"input": map[string]any{"who": "{{trigger.input.who}}"},
			}},
			{ID: "b", Type: "script", Config: map[string]any{
				"code":  `return { upper: input.s.toUpperCase() };`,
				"input": map[string]any{"s": "{{a.output.greeting}}"},
			}},
		},
		Edges: []EdgeDef{{From: "a", To: "b"}},
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "f1", Definition: string(defJSON), Status: FlowStatusActive}

	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{
		Type:  TriggerTypeManual,
		Input: map[string]any{"who": "world"},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	// 等待异步完成
	deadline := time.Now().Add(3 * time.Second)
	var final *ExecutionContext
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		// 读 in-memory exec.Context（engine 会更新）
		ec, _ := exec.DecodeContext()
		if ec != nil && len(ec.Nodes) == 2 && ec.Nodes["b"].Status == NodeStatusSuccess {
			final = ec
			break
		}
	}
	if final == nil {
		t.Fatalf("script flow did not finish: %+v", exec)
	}
	got := final.Nodes["b"].Output["upper"]
	if got != "HELLO WORLD" {
		t.Fatalf("got upper=%v", got)
	}
}

func TestEngine_ConditionBranches(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 30; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	eng := NewEngine(db, nodes.DefaultRegistry(), nil)
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "classify", Type: "condition", Config: map[string]any{
				"expression": "{{trigger.input.kind}}",
				"branches": []any{
					map[string]any{"value": "small"},
					map[string]any{"value": "big"},
				},
			}},
			{ID: "small_path", Type: "script", Config: map[string]any{"code": `return { hit: "small" };`}},
			{ID: "big_path", Type: "script", Config: map[string]any{"code": `return { hit: "big" };`}},
		},
		Edges: []EdgeDef{
			{From: "classify", To: "small_path", Branch: "small"},
			{From: "classify", To: "big_path", Branch: "big"},
		},
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "fcond", Definition: string(defJSON), Status: FlowStatusActive}

	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{
		Type:  TriggerTypeManual,
		Input: map[string]any{"kind": "big"},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		ec, _ := exec.DecodeContext()
		if ec == nil {
			continue
		}
		if ec.Nodes["big_path"].Status == NodeStatusSuccess {
			// small_path 应未执行
			if _, ran := ec.Nodes["small_path"]; ran {
				t.Fatalf("small_path should not run, got: %+v", ec.Nodes["small_path"])
			}
			return
		}
	}
	ec, _ := exec.DecodeContext()
	t.Fatalf("did not finish: %+v", ec)
}

func TestEngine_HTTPNodeIntegrates(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 30; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"echo":"x"}`))
	}))
	defer srv.Close()

	eng := NewEngine(db, nodes.DefaultRegistry(), nil)
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "fetch", Type: "http", Config: map[string]any{
				"method": "GET",
				"url":    srv.URL,
			}},
		},
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "fhttp", Definition: string(defJSON), Status: FlowStatusActive}
	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{Type: TriggerTypeManual})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		ec, _ := exec.DecodeContext()
		if ec != nil && ec.Nodes["fetch"].Status == NodeStatusSuccess {
			jv, _ := ec.Nodes["fetch"].Output["json"].(map[string]any)
			if jv["ok"] != true {
				t.Fatalf("json=%#v", jv)
			}
			return
		}
	}
	t.Fatalf("http flow did not finish")
}

// blockingNode 是一个测试节点，会阻塞直到 ctx 被取消，然后返回 ctx.Err()
// （模拟真实节点在执行中被取消的行为，例如 HTTP 请求遇到 context.Canceled）。
type blockingNode struct {
	started chan struct{}
	once    sync.Once
}

func (b *blockingNode) Type() string { return "block" }
func (b *blockingNode) Run(ctx context.Context, _ map[string]any) (*nodes.Result, error) {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestEngine_CancelDoesNotOverwriteToFailed 验证 PR#9 review bug 1：
// 当 CancelExecution 把状态置为 cancelled 后，执行 goroutine 不应再走
// finishFailed 把状态覆写为 failed。
func TestEngine_CancelDoesNotOverwriteToFailed(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 30; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	bn := &blockingNode{started: make(chan struct{})}
	reg := nodes.NewRegistry()
	reg.Register(bn)
	eng := NewEngine(db, reg, nil)

	def := &Definition{
		Nodes: []NodeDef{{ID: "block", Type: "block", Config: map[string]any{}}},
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "fcancel", Definition: string(defJSON), Status: FlowStatusActive}

	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{Type: TriggerTypeManual})
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	// 等节点开始执行
	select {
	case <-bn.started:
	case <-time.After(2 * time.Second):
		t.Fatalf("node never started")
	}

	if err := eng.CancelExecution(exec.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// 等 goroutine 落地：cancel map 中条目被清除
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		eng.mu.Lock()
		_, stillRunning := eng.cancel[exec.ID]
		eng.mu.Unlock()
		if !stillRunning {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 关键断言：goroutine 不应把内存中的 exec.Status 覆写为 failed
	// （没有 fix 时 finishFailed 会把它置为 failed，覆盖 CancelExecution
	//  写入 DB 的 cancelled）。
	if exec.Status == ExecutionStatusFailed {
		t.Fatalf("regression: cancel was overwritten to failed; got status=%s", exec.Status)
	}
}
