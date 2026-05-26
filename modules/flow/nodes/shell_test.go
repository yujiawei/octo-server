package nodes

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestShellNode_Echo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell node uses sh -c; skip on windows")
	}
	n := NewShellNode()
	res, err := n.Run(context.Background(), map[string]any{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Output["exit_code"].(int) != 0 {
		t.Fatalf("exit_code: %v", res.Output["exit_code"])
	}
	if !strings.Contains(res.Output["stdout"].(string), "hello") {
		t.Fatalf("stdout: %q", res.Output["stdout"])
	}
	if res.Output["stderr"].(string) != "" {
		t.Fatalf("stderr should be empty, got: %q", res.Output["stderr"])
	}
}

func TestShellNode_NonZeroExitFailsButPreservesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell node uses sh -c; skip on windows")
	}
	n := NewShellNode()
	res, err := n.Run(context.Background(), map[string]any{
		"command": "echo out; echo err 1>&2; exit 7",
	})
	if err == nil {
		t.Fatalf("expected error for non-zero exit code")
	}
	if res == nil {
		t.Fatalf("result must be non-nil so engine can preserve stdout/stderr")
	}
	if res.Output["exit_code"].(int) != 7 {
		t.Fatalf("exit_code: %v", res.Output["exit_code"])
	}
	if !strings.Contains(res.Output["stdout"].(string), "out") {
		t.Fatalf("stdout missing: %q", res.Output["stdout"])
	}
	if !strings.Contains(res.Output["stderr"].(string), "err") {
		t.Fatalf("stderr missing: %q", res.Output["stderr"])
	}
}

func TestShellNode_RequiresCommand(t *testing.T) {
	n := NewShellNode()
	if _, err := n.Run(context.Background(), map[string]any{}); err == nil {
		t.Fatalf("expected error for missing command")
	}
}

func TestShellNode_TimeoutKillsCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell node uses sh -c; skip on windows")
	}
	n := NewShellNode()
	_, err := n.Run(context.Background(), map[string]any{
		"command": "sleep 5",
		"timeout": "100ms",
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("error should mention timeout, got: %v", err)
	}
}

func TestShellNode_TimeoutCappedAt300s(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell node uses sh -c; skip on windows")
	}
	// 超过 300s 的配置应被 clamp 到 300s 上限；命令本身只跑 echo，
	// 应当快速成功。这个测试主要验证 parser 不会拒绝大值。
	n := NewShellNode()
	res, err := n.Run(context.Background(), map[string]any{
		"command": "echo ok",
		"timeout": "10m",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Output["exit_code"].(int) != 0 {
		t.Fatalf("exit_code: %v", res.Output["exit_code"])
	}
}

func TestShellNode_DoesNotInheritEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell node uses sh -c; skip on windows")
	}
	t.Setenv("OCTO_FLOW_SHELL_NODE_INHERIT_PROBE", "should-not-leak")
	n := NewShellNode()
	res, err := n.Run(context.Background(), map[string]any{
		// 父进程设置了 PROBE，但 cmd.Env 不应该继承它。
		"command": "echo \"probe=${OCTO_FLOW_SHELL_NODE_INHERIT_PROBE:-empty}\"",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(res.Output["stdout"].(string), "probe=empty") {
		t.Fatalf("stdout should not see host env, got: %q", res.Output["stdout"])
	}
}

func TestShellNode_UsesExplicitEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell node uses sh -c; skip on windows")
	}
	n := NewShellNode()
	res, err := n.Run(context.Background(), map[string]any{
		"command": "echo \"hello $NAME\"",
		"env":     map[string]any{"NAME": "world", "PATH": "/usr/bin:/bin"},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(res.Output["stdout"].(string), "hello world") {
		t.Fatalf("stdout: %q", res.Output["stdout"])
	}
}

func TestShellNode_BadTimeoutString(t *testing.T) {
	n := NewShellNode()
	if _, err := n.Run(context.Background(), map[string]any{
		"command": "echo hi",
		"timeout": "not-a-duration",
	}); err == nil {
		t.Fatalf("expected error for invalid timeout")
	}
}

func TestShellNode_BadEnvType(t *testing.T) {
	n := NewShellNode()
	if _, err := n.Run(context.Background(), map[string]any{
		"command": "echo hi",
		"env":     "not-a-map",
	}); err == nil {
		t.Fatalf("expected error for non-map env")
	}
}

func TestShellNode_Type(t *testing.T) {
	if NewShellNode().Type() != "shell" {
		t.Fatal("type should be shell")
	}
}
