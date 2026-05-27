// Package nodes 实现 Octo Flow 的节点类型：script / http / condition / ...
//
// 每个节点实现 Runner 接口。引擎在执行节点前把 NodeDef.Config 中的
// {{...}} 模板引用渲染为实际值，然后传给 Runner.Run。
package nodes

import (
	"context"
	"errors"
	"sync"
)

// Result 是节点执行结果。
//
// Output 会被写回 ExecutionContext.Nodes[nodeID].Output，下游节点可通过
// {{node_id.output.field}} 引用。
//
// NextBranches 仅对 condition 节点有意义：指定下一步应当走哪条 branch
// 名（与 EdgeDef.Branch 匹配）。空表示走所有未指定 branch 的出边。
type Result struct {
	Output       map[string]any
	NextBranches []string
}

// Runner 是节点执行器。
//
// rendered 是已经把 {{...}} 模板渲染过的 config，节点实现可以直接读字段。
type Runner interface {
	Type() string
	Run(ctx context.Context, rendered map[string]any) (*Result, error)
}

// Registry 是节点类型注册表
type Registry struct {
	mu     sync.RWMutex
	byType map[string]Runner
}

// NewRegistry 创建一个空 registry
func NewRegistry() *Registry {
	return &Registry{byType: map[string]Runner{}}
}

// Register 注册节点类型
func (r *Registry) Register(runner Runner) {
	if runner == nil || runner.Type() == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byType[runner.Type()] = runner
}

// Get 按 type 取 runner
func (r *Registry) Get(t string) (Runner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rn, ok := r.byType[t]
	return rn, ok
}

// ErrUnknownType 表示未注册的节点类型
var ErrUnknownType = errors.New("unknown node type")

// ExecContextKey 是引擎注入到 rendered config 中的执行上下文快照键名。
// script 节点会从 rendered config 中提取这个键，把它绑成 JS 全局
// `context`，并从 rendered config 中删除该键，避免污染 input。
//
// 该键以双下划线包围，且不会出现在任何用户级 schema 中，因此对外的
// `input.*` 视图保持不变。
const ExecContextKey = "__exec_context__"

// DefaultRegistry 返回内置节点 registry（script / http / condition）
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewScriptNode())
	r.Register(NewHTTPNode(nil))
	r.Register(NewConditionNode())
	return r
}
