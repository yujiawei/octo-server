package flow

import (
	"errors"
	"fmt"
	"sort"
)

// DAG 表示已解析的 flow 图
type DAG struct {
	Nodes    map[string]NodeDef
	Edges    []EdgeDef
	Outgoing map[string][]EdgeDef // nodeID -> 出边
	Incoming map[string][]EdgeDef // nodeID -> 入边
	TopoSort []string             // 拓扑序（不含 __start__/__end__）
}

// BuildDAG 解析 definition 并构建 DAG，做基本校验（节点 id 唯一、边引用合法、无环）
func BuildDAG(def *Definition) (*DAG, error) {
	if def == nil {
		return nil, errors.New("definition is nil")
	}
	d := &DAG{
		Nodes:    map[string]NodeDef{},
		Edges:    def.Edges,
		Outgoing: map[string][]EdgeDef{},
		Incoming: map[string][]EdgeDef{},
	}
	for _, n := range def.Nodes {
		if n.ID == "" {
			return nil, errors.New("node missing id")
		}
		if _, dup := d.Nodes[n.ID]; dup {
			return nil, fmt.Errorf("duplicate node id: %s", n.ID)
		}
		if n.Type == "" {
			return nil, fmt.Errorf("node %s missing type", n.ID)
		}
		d.Nodes[n.ID] = n
	}
	for _, e := range def.Edges {
		// __start__ / __end__ 是允许的占位符
		if e.From != "__start__" && e.From != "" {
			if _, ok := d.Nodes[e.From]; !ok {
				return nil, fmt.Errorf("edge.from references unknown node: %s", e.From)
			}
		}
		if e.To != "__end__" && e.To != "" {
			if _, ok := d.Nodes[e.To]; !ok {
				return nil, fmt.Errorf("edge.to references unknown node: %s", e.To)
			}
		}
		d.Outgoing[e.From] = append(d.Outgoing[e.From], e)
		d.Incoming[e.To] = append(d.Incoming[e.To], e)
	}
	order, err := topoSort(d)
	if err != nil {
		return nil, err
	}
	d.TopoSort = order
	return d, nil
}

// EntryNodes 返回入度为 0 或仅由 __start__ 进入的节点
func (d *DAG) EntryNodes() []string {
	var entries []string
	for id := range d.Nodes {
		in := d.Incoming[id]
		if len(in) == 0 {
			entries = append(entries, id)
			continue
		}
		all := true
		for _, e := range in {
			if e.From != "__start__" && e.From != "" {
				all = false
				break
			}
		}
		if all {
			entries = append(entries, id)
		}
	}
	sort.Strings(entries)
	return entries
}

// Levels 把 DAG 按层划分。同一层中的节点没有相互依赖，可以并行执行。
//
// 返回值是按执行顺序排列的若干层，每层是该层节点 ID 的有序列表
// （字典序，保证稳定）。__start__ / __end__ 虚拟节点不参与分层。
//
// 示例：A→B、A→C、B→D、C→D 会得到 [[A], [B, C], [D]]。
//
// 一个节点的层级 = max(前驱层级) + 1（前驱不含 __start__）。空 DAG 返回 nil。
func (d *DAG) Levels() [][]string {
	indeg := make(map[string]int, len(d.Nodes))
	for id := range d.Nodes {
		indeg[id] = 0
	}
	for _, e := range d.Edges {
		if e.To == "__end__" || e.To == "" {
			continue
		}
		if e.From == "__start__" || e.From == "" {
			continue
		}
		if _, ok := d.Nodes[e.To]; ok {
			indeg[e.To]++
		}
	}

	var current []string
	for id, deg := range indeg {
		if deg == 0 {
			current = append(current, id)
		}
	}
	sort.Strings(current)

	var levels [][]string
	for len(current) > 0 {
		level := make([]string, len(current))
		copy(level, current)
		levels = append(levels, level)

		var next []string
		for _, n := range current {
			for _, e := range d.Outgoing[n] {
				if e.To == "__end__" || e.To == "" {
					continue
				}
				if _, ok := d.Nodes[e.To]; !ok {
					continue
				}
				indeg[e.To]--
				if indeg[e.To] == 0 {
					next = append(next, e.To)
				}
			}
		}
		sort.Strings(next)
		current = next
	}
	return levels
}

func topoSort(d *DAG) ([]string, error) {
	// Kahn's algorithm; 排除 __start__/__end__ 虚拟节点
	indeg := map[string]int{}
	for id := range d.Nodes {
		indeg[id] = 0
	}
	for _, e := range d.Edges {
		if e.To == "__end__" || e.To == "" {
			continue
		}
		if e.From == "__start__" || e.From == "" {
			continue
		}
		if _, ok := d.Nodes[e.To]; ok {
			indeg[e.To]++
		}
	}
	// 使用 slice + 排序保证稳定性
	var queue []string
	for id, deg := range indeg {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	var out []string
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		out = append(out, n)
		var nextBatch []string
		for _, e := range d.Outgoing[n] {
			if e.To == "__end__" || e.To == "" {
				continue
			}
			indeg[e.To]--
			if indeg[e.To] == 0 {
				nextBatch = append(nextBatch, e.To)
			}
		}
		sort.Strings(nextBatch)
		queue = append(queue, nextBatch...)
	}
	if len(out) != len(d.Nodes) {
		return nil, errors.New("cycle detected in flow graph")
	}
	return out, nil
}
