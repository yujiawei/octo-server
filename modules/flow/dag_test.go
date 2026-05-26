package flow

import (
	"testing"
)

func TestBuildDAG_Simple(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "b", Type: "script"},
			{ID: "c", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}
	dag, err := BuildDAG(def)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := len(dag.TopoSort); got != 3 {
		t.Fatalf("topo size: %d", got)
	}
	// a 必须在 b 之前
	posA, posB, posC := -1, -1, -1
	for i, id := range dag.TopoSort {
		switch id {
		case "a":
			posA = i
		case "b":
			posB = i
		case "c":
			posC = i
		}
	}
	if !(posA < posB && posB < posC) {
		t.Fatalf("topo order wrong: %v", dag.TopoSort)
	}
	entries := dag.EntryNodes()
	if len(entries) != 1 || entries[0] != "a" {
		t.Fatalf("entry nodes wrong: %v", entries)
	}
}

func TestBuildDAG_DetectsCycle(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "b", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	}
	if _, err := BuildDAG(def); err == nil {
		t.Fatalf("expected cycle detection, got nil")
	}
}

func TestBuildDAG_DuplicateNode(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "a", Type: "script"},
		},
	}
	if _, err := BuildDAG(def); err == nil {
		t.Fatalf("expected duplicate id error")
	}
}

func TestBuildDAG_UnknownEdgeRef(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
		},
	}
	if _, err := BuildDAG(def); err == nil {
		t.Fatalf("expected unknown ref error")
	}
}

func TestBuildDAG_StartEnd(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "__start__", To: "a"},
			{From: "a", To: "__end__"},
		},
	}
	dag, err := BuildDAG(def)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if e := dag.EntryNodes(); len(e) != 1 || e[0] != "a" {
		t.Fatalf("entry nodes: %v", e)
	}
}

// TestDAG_Levels_Diamond 验证 A→B、A→C、B→D、C→D 的菱形被划分为
// [[A], [B, C], [D]]：A 先执行，B/C 并行，D 最后。
func TestDAG_Levels_Diamond(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "b", Type: "script"},
			{ID: "c", Type: "script"},
			{ID: "d", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}
	dag, err := BuildDAG(def)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	levels := dag.Levels()
	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %d: %v", len(levels), levels)
	}
	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Fatalf("level 0: %v", levels[0])
	}
	if len(levels[1]) != 2 || levels[1][0] != "b" || levels[1][1] != "c" {
		t.Fatalf("level 1: %v", levels[1])
	}
	if len(levels[2]) != 1 || levels[2][0] != "d" {
		t.Fatalf("level 2: %v", levels[2])
	}
}

// TestDAG_Levels_Linear 验证纯串行链路每层只一个节点。
func TestDAG_Levels_Linear(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "b", Type: "script"},
			{ID: "c", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}
	dag, err := BuildDAG(def)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	levels := dag.Levels()
	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %v", levels)
	}
	for i, want := range []string{"a", "b", "c"} {
		if len(levels[i]) != 1 || levels[i][0] != want {
			t.Fatalf("level %d: %v", i, levels[i])
		}
	}
}

// TestDAG_Levels_Independent 多入口独立分支：A 与 X 同处第 0 层，
// B 与 Y 同处第 1 层。
func TestDAG_Levels_Independent(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "b", Type: "script"},
			{ID: "x", Type: "script"},
			{ID: "y", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
			{From: "x", To: "y"},
		},
	}
	dag, err := BuildDAG(def)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	levels := dag.Levels()
	if len(levels) != 2 {
		t.Fatalf("expected 2 levels, got %v", levels)
	}
	if !equalSet(levels[0], []string{"a", "x"}) {
		t.Fatalf("level 0: %v", levels[0])
	}
	if !equalSet(levels[1], []string{"b", "y"}) {
		t.Fatalf("level 1: %v", levels[1])
	}
}

// TestDAG_Levels_LongerPathDeterminesLevel 验证当一个节点同时被
// 短路径和长路径指向时，其层级由最长前驱路径决定。
//
//	A → B → D
//	A → D
//
// D 应该排在 B 之后（第 2 层），而非与 B 同层。
func TestDAG_Levels_LongerPathDeterminesLevel(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "b", Type: "script"},
			{ID: "d", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "a", To: "b"},
			{From: "a", To: "d"},
			{From: "b", To: "d"},
		},
	}
	dag, err := BuildDAG(def)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	levels := dag.Levels()
	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %v", levels)
	}
	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Fatalf("level 0: %v", levels[0])
	}
	if len(levels[1]) != 1 || levels[1][0] != "b" {
		t.Fatalf("level 1: %v", levels[1])
	}
	if len(levels[2]) != 1 || levels[2][0] != "d" {
		t.Fatalf("level 2: %v", levels[2])
	}
}

func TestDAG_Levels_StartEndIgnored(t *testing.T) {
	def := &Definition{
		Nodes: []NodeDef{
			{ID: "a", Type: "script"},
			{ID: "b", Type: "script"},
		},
		Edges: []EdgeDef{
			{From: "__start__", To: "a"},
			{From: "a", To: "b"},
			{From: "b", To: "__end__"},
		},
	}
	dag, err := BuildDAG(def)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	levels := dag.Levels()
	if len(levels) != 2 {
		t.Fatalf("levels: %v", levels)
	}
	if levels[0][0] != "a" || levels[1][0] != "b" {
		t.Fatalf("levels: %v", levels)
	}
}

func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}
