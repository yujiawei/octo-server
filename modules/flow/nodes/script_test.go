package nodes

import (
	"context"
	"testing"
)

func TestScriptNode_ReturnsObject(t *testing.T) {
	s := NewScriptNode()
	cfg := map[string]any{
		"runtime": "javascript",
		"code":    `return { sum: input.a + input.b };`,
		"input":   map[string]any{"a": 2, "b": 3},
	}
	res, err := s.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	v, ok := res.Output["sum"]
	if !ok {
		t.Fatalf("output missing sum: %#v", res.Output)
	}
	// goja exports number as int64 or float64; allow both
	switch x := v.(type) {
	case int64:
		if x != 5 {
			t.Fatalf("got %v", x)
		}
	case float64:
		if x != 5 {
			t.Fatalf("got %v", x)
		}
	default:
		t.Fatalf("unexpected type %T", v)
	}
}

func TestScriptNode_ReturnsScalar(t *testing.T) {
	s := NewScriptNode()
	cfg := map[string]any{
		"code": `return "hello";`,
	}
	res, err := s.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Output["result"] != "hello" {
		t.Fatalf("got %v", res.Output["result"])
	}
}

func TestScriptNode_ConsoleLog(t *testing.T) {
	s := NewScriptNode()
	cfg := map[string]any{
		"code": `console.log("hi"); return { ok: true };`,
	}
	res, err := s.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	logs, ok := res.Output["_logs"].([]string)
	if !ok || len(logs) != 1 || logs[0] != "hi" {
		t.Fatalf("logs: %#v", res.Output["_logs"])
	}
}

func TestScriptNode_RejectsUnknownRuntime(t *testing.T) {
	s := NewScriptNode()
	if _, err := s.Run(context.Background(), map[string]any{
		"runtime": "python", "code": "x",
	}); err == nil {
		t.Fatalf("expected err")
	}
}

func TestScriptNode_RequiresCode(t *testing.T) {
	s := NewScriptNode()
	if _, err := s.Run(context.Background(), map[string]any{}); err == nil {
		t.Fatalf("expected err")
	}
}

// TestScriptNode_ContextGlobal 验证 script 节点能访问引擎注入的
// `context` 全局——这是 GH-23 的回归保护：以前 `context` 未注入，
// 任何 `context.trigger.payload.*` 访问都会抛 ReferenceError。
func TestScriptNode_ContextGlobal(t *testing.T) {
	s := NewScriptNode()
	cfg := map[string]any{
		"code": `return {
			action: context.trigger.payload.action,
			exec_id: context.execution_id,
			flow_id: context.flow_id,
			from_node: context.nodes.upstream.output.greeting,
			from_vars: context.vars.region,
		};`,
		ExecContextKey: map[string]any{
			"execution_id": "exec-123",
			"flow_id":      "flow-abc",
			"trigger": map[string]any{
				"type": "github",
				"payload": map[string]any{
					"action": "opened",
				},
			},
			"nodes": map[string]any{
				"upstream": map[string]any{
					"status": "success",
					"output": map[string]any{"greeting": "hi"},
				},
			},
			"vars": map[string]any{"region": "us-east"},
		},
	}
	res, err := s.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := res.Output["action"]; got != "opened" {
		t.Fatalf("action: got %v, want opened (output=%#v)", got, res.Output)
	}
	if got := res.Output["exec_id"]; got != "exec-123" {
		t.Fatalf("exec_id: got %v", got)
	}
	if got := res.Output["flow_id"]; got != "flow-abc" {
		t.Fatalf("flow_id: got %v", got)
	}
	if got := res.Output["from_node"]; got != "hi" {
		t.Fatalf("from_node: got %v", got)
	}
	if got := res.Output["from_vars"]; got != "us-east" {
		t.Fatalf("from_vars: got %v", got)
	}
	// 关键约束：__exec_context__ 不能泄漏进 cfg["input"] 或被
	// 后续节点感知；script 节点内部已经从 cfg 中删除了它。
	if _, leaked := cfg[ExecContextKey]; leaked {
		t.Fatalf("__exec_context__ leaked back into cfg")
	}
}

// TestScriptNode_NoContextStillWorks 验证脚本不引用 context 时
// 旧行为不变（向后兼容）。
func TestScriptNode_NoContextStillWorks(t *testing.T) {
	s := NewScriptNode()
	cfg := map[string]any{
		"code":  `return { ok: input.x * 2 };`,
		"input": map[string]any{"x": 21},
	}
	res, err := s.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	switch v := res.Output["ok"].(type) {
	case int64:
		if v != 42 {
			t.Fatalf("got %v", v)
		}
	case float64:
		if v != 42 {
			t.Fatalf("got %v", v)
		}
	default:
		t.Fatalf("unexpected type %T", v)
	}
}

// TestScriptNode_ContextAbsentReferenceError 在没有 context 注入时，
// 引用 `context` 应产生 ReferenceError。这固定了「未注入即未定义」
// 的行为，使测试可以察觉将来引擎的回归（停止注入 context）。
func TestScriptNode_ContextAbsentReferenceError(t *testing.T) {
	s := NewScriptNode()
	cfg := map[string]any{
		"code": `return { x: context.trigger.payload.action };`,
	}
	if _, err := s.Run(context.Background(), cfg); err == nil {
		t.Fatalf("expected ReferenceError when context is not injected")
	}
}
