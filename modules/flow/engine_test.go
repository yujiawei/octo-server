package flow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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
	if !waitEngineDone(eng, exec.ID, 3*time.Second) {
		t.Fatalf("script flow did not finish: %+v", exec)
	}
	final, _ := exec.DecodeContext()
	if final == nil || len(final.Nodes) != 2 || final.Nodes["b"].Status != NodeStatusSuccess {
		t.Fatalf("script flow did not finish: %+v", final)
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
	if !waitEngineDone(eng, exec.ID, 3*time.Second) {
		ec, _ := exec.DecodeContext()
		t.Fatalf("did not finish: %+v", ec)
	}
	ec, _ := exec.DecodeContext()
	if ec == nil || ec.Nodes["big_path"].Status != NodeStatusSuccess {
		t.Fatalf("big_path not success: %+v", ec)
	}
	if _, ran := ec.Nodes["small_path"]; ran {
		t.Fatalf("small_path should not run, got: %+v", ec.Nodes["small_path"])
	}
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
	if !waitEngineDone(eng, exec.ID, 3*time.Second) {
		t.Fatalf("http flow did not finish")
	}
	ec, _ := exec.DecodeContext()
	if ec == nil || ec.Nodes["fetch"].Status != NodeStatusSuccess {
		t.Fatalf("fetch not success: %+v", ec)
	}
	jv, _ := ec.Nodes["fetch"].Output["json"].(map[string]any)
	if jv["ok"] != true {
		t.Fatalf("json=%#v", jv)
	}
}

// sleepRunner 是测试用的节点 Runner：阻塞 sleep 指定时长后返回成功，
// 同时记录开始/结束时间戳，方便验证「同层并行」是否真的发生。
type sleepRunner struct {
	typeName string
	sleep    time.Duration
	startCnt *int32
	maxCnt   *int32
	endTimes chan time.Time
}

func (s *sleepRunner) Type() string { return s.typeName }

func (s *sleepRunner) Run(ctx context.Context, _ map[string]any) (*nodes.Result, error) {
	// 记录峰值并发
	cur := atomic.AddInt32(s.startCnt, 1)
	for {
		old := atomic.LoadInt32(s.maxCnt)
		if cur <= old {
			break
		}
		if atomic.CompareAndSwapInt32(s.maxCnt, old, cur) {
			break
		}
	}
	defer atomic.AddInt32(s.startCnt, -1)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(s.sleep):
	}
	if s.endTimes != nil {
		s.endTimes <- time.Now()
	}
	return &nodes.Result{Output: map[string]any{"ok": true}}, nil
}

// failRunner 故意失败，用于测试同层错误传播。
type failRunner struct {
	typeName string
	delay    time.Duration
}

func (f *failRunner) Type() string { return f.typeName }

func (f *failRunner) Run(ctx context.Context, _ map[string]any) (*nodes.Result, error) {
	if f.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(f.delay):
		}
	}
	return nil, errors.New("boom")
}

// TestEngine_ParallelLayer 验证两个独立节点真的并行执行：
// 总耗时应接近 max(t1, t2) 而不是 t1+t2，并且峰值并发应达到 2。
func TestEngine_ParallelLayer(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 30; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	reg := nodes.NewRegistry()
	var startCnt, maxCnt int32
	endTimes := make(chan time.Time, 8)
	reg.Register(&sleepRunner{typeName: "slow", sleep: 200 * time.Millisecond, startCnt: &startCnt, maxCnt: &maxCnt, endTimes: endTimes})

	eng := NewEngine(db, reg, nil)
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "slow"},
			{ID: "b", Type: "slow"},
		},
		// 没有边 → A 与 B 都是入口节点，同处第 0 层，应并行执行。
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "fparallel", Definition: string(defJSON), Status: FlowStatusActive}

	t0 := time.Now()
	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{Type: TriggerTypeManual})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !waitEngineDone(eng, exec.ID, 2*time.Second) {
		t.Fatalf("parallel flow did not finish")
	}
	ec, _ := exec.DecodeContext()
	if ec == nil ||
		ec.Nodes["a"].Status != NodeStatusSuccess ||
		ec.Nodes["b"].Status != NodeStatusSuccess {
		t.Fatalf("expected both nodes success, got: %+v", ec)
	}
	elapsed := time.Since(t0)
	// 串行至少 400ms（200ms × 2）；并行应明显小于 400ms。给 350ms 裕量。
	if elapsed >= 350*time.Millisecond {
		t.Fatalf("expected parallel execution, elapsed=%v >= 350ms", elapsed)
	}
	if got := atomic.LoadInt32(&maxCnt); got < 2 {
		t.Fatalf("expected concurrent peak >= 2, got %d", got)
	}
}

// TestEngine_ParallelDiamond 验证 A→B、A→C、B→D、C→D 菱形 DAG 中
// B 与 C 真的并行执行（峰值并发 2），且 D 的输入仍能正确读到 B/C 的结果。
func TestEngine_ParallelDiamond(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 50; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	reg := nodes.DefaultRegistry()
	var startCnt, maxCnt int32
	reg.Register(&sleepRunner{typeName: "slow", sleep: 150 * time.Millisecond, startCnt: &startCnt, maxCnt: &maxCnt})

	eng := NewEngine(db, reg, nil)
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "slow"},
			{ID: "b", Type: "slow"},
			{ID: "c", Type: "slow"},
			{ID: "d", Type: "script", Config: map[string]any{
				"code":  `return { fanin: input.b + ":" + input.c };`,
				"input": map[string]any{"b": "{{b.output.ok}}", "c": "{{c.output.ok}}"},
			}},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "fdiamond", Definition: string(defJSON), Status: FlowStatusActive}

	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{Type: TriggerTypeManual})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !waitEngineDone(eng, exec.ID, 3*time.Second) {
		t.Fatalf("diamond flow did not finish")
	}
	ec, _ := exec.DecodeContext()
	if ec == nil || ec.Nodes["d"].Status != NodeStatusSuccess {
		t.Fatalf("expected d=success, got %+v", ec)
	}
	if got := ec.Nodes["d"].Output["fanin"]; got != "true:true" {
		t.Fatalf("fanin=%v", got)
	}
	if got := atomic.LoadInt32(&maxCnt); got < 2 {
		t.Fatalf("expected B/C parallel peak >= 2, got %d", got)
	}
}

// TestEngine_ParallelConditionAndSibling 验证同层有 condition 节点和
// 普通节点时：condition 的下游 branch 选择在该层执行结束后才计算，
// 普通节点的下游也在同一时刻一起激活。
func TestEngine_ParallelConditionAndSibling(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 50; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	eng := NewEngine(db, nodes.DefaultRegistry(), nil)
	def := &Definition{
		Nodes: []NodeDef{
			// classify 与 sibling 同处第 0 层（都是入口节点）
			{ID: "classify", Type: "condition", Config: map[string]any{
				"expression": "{{trigger.input.kind}}",
				"branches": []any{
					map[string]any{"value": "small"},
					map[string]any{"value": "big"},
				},
			}},
			{ID: "sibling", Type: "script", Config: map[string]any{
				"code": `return { hit: "sibling" };`,
			}},
			{ID: "small_path", Type: "script", Config: map[string]any{"code": `return { hit: "small" };`}},
			{ID: "big_path", Type: "script", Config: map[string]any{"code": `return { hit: "big" };`}},
			{ID: "after_sibling", Type: "script", Config: map[string]any{"code": `return { hit: "after" };`}},
		},
		Edges: []EdgeDef{
			{From: "classify", To: "small_path", Branch: "small"},
			{From: "classify", To: "big_path", Branch: "big"},
			{From: "sibling", To: "after_sibling"},
		},
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "fcondpar", Definition: string(defJSON), Status: FlowStatusActive}

	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{
		Type:  TriggerTypeManual,
		Input: map[string]any{"kind": "big"},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !waitEngineDone(eng, exec.ID, 3*time.Second) {
		ec, _ := exec.DecodeContext()
		t.Fatalf("did not finish: %+v", ec)
	}
	ec, _ := exec.DecodeContext()
	if ec == nil {
		t.Fatalf("nil context")
	}
	if ec.Nodes["big_path"].Status != NodeStatusSuccess {
		t.Fatalf("big_path status=%v", ec.Nodes["big_path"].Status)
	}
	if ec.Nodes["after_sibling"].Status != NodeStatusSuccess {
		t.Fatalf("after_sibling status=%v", ec.Nodes["after_sibling"].Status)
	}
	if _, ran := ec.Nodes["small_path"]; ran {
		t.Fatalf("small_path should not run")
	}
	if ec.Nodes["sibling"].Status != NodeStatusSuccess {
		t.Fatalf("sibling status=%v", ec.Nodes["sibling"].Status)
	}
}

// TestEngine_ParallelFailureMarksFailed 验证同层中一个节点失败 →
// 整个 execution 标记 failed，且姐妹节点会因 errgroup ctx cancel 提前退出。
func TestEngine_ParallelFailureMarksFailed(t *testing.T) {
	db, mock := newMockDB(t)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 30; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	reg := nodes.NewRegistry()
	var startCnt, maxCnt int32
	// 兄弟节点 sleep 比失败节点更久，验证它会被 ctx 取消而提前返回
	reg.Register(&sleepRunner{typeName: "slow", sleep: 1 * time.Second, startCnt: &startCnt, maxCnt: &maxCnt})
	reg.Register(&failRunner{typeName: "fail", delay: 50 * time.Millisecond})

	eng := NewEngine(db, reg, nil)
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "slow"},
			{ID: "b", Type: "fail"},
		},
		// 同层入口节点：a 慢、b 快但失败 → b 失败后 a 的 ctx 应被 cancel
	}
	defJSON, _ := json.Marshal(def)
	flow := &Flow{ID: "ffail", Definition: string(defJSON), Status: FlowStatusActive}

	t0 := time.Now()
	exec, err := eng.StartExecution(context.Background(), flow, "", TriggerData{Type: TriggerTypeManual})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	// 等引擎 goroutine 真正退出（cancel 映射条目被 defer 删除时持有 eng.mu，
	// 我们这边再持锁观察 → 与 goroutine 退出建立 happens-before，保证后续读 exec 无 race）。
	if !waitEngineDone(eng, exec.ID, 3*time.Second) {
		t.Fatalf("engine goroutine did not exit in time")
	}
	if exec.Status != ExecutionStatusFailed {
		t.Fatalf("expected execution failed, got status=%q error=%q", exec.Status, exec.Error)
	}
	// 兄弟节点 a 的 sleep 是 1s，但因为 errgroup cancel，整体应远早于 1s 结束。
	if elapsed := time.Since(t0); elapsed >= 800*time.Millisecond {
		t.Fatalf("sibling cancel did not propagate, elapsed=%v", elapsed)
	}
}

// waitEngineDone 阻塞直到引擎上的 execution goroutine 退出（cancel 映射条目消失）。
// 持有 eng.mu 与引擎 goroutine 的 defer 建立 happens-before，
// 所以返回后读 exec.* 不会与引擎写竞态。
func waitEngineDone(eng *Engine, execID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		eng.mu.Lock()
		_, running := eng.cancel[execID]
		eng.mu.Unlock()
		if !running {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}
