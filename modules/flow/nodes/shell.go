package nodes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"time"
)

// ShellNode 执行 shell 命令并返回 stdout/stderr/exit_code。
//
// config:
//
//	command: string                  # 必填；引擎已经把 {{...}} 模板渲染过
//	timeout: string                  # 默认 60s，最大 300s
//	env:     map[string]string       # 进程环境变量；不继承宿主进程
//
// output:
//
//	stdout:    string
//	stderr:    string
//	exit_code: int
//
// 行为约定：
//   - exit_code != 0 时，节点状态为 failed；但 stdout / stderr / exit_code 仍写入 output。
//   - 不继承宿主进程环境变量，env 完全显式声明（可为空）。
//   - 命令通过 `sh -c <command>` 执行，便于使用管道、重定向等 shell 语法。
type ShellNode struct{}

// NewShellNode 构造一个 shell runner
func NewShellNode() *ShellNode { return &ShellNode{} }

// Type 返回 "shell"
func (s *ShellNode) Type() string { return "shell" }

const (
	shellDefaultTimeout = 60 * time.Second
	shellMaxTimeout     = 300 * time.Second
)

// Run 执行命令。返回的 *Result 即使 err != nil 也不为 nil（保留 stdout/stderr/exit_code）。
// 调用方（引擎）需要在 err != nil 分支里也保留 result.Output。
func (s *ShellNode) Run(ctx context.Context, cfg map[string]any) (*Result, error) {
	command, _ := cfg["command"].(string)
	if command == "" {
		return nil, errors.New("shell node: command is required")
	}

	timeout := shellDefaultTimeout
	if ts, ok := cfg["timeout"].(string); ok && ts != "" {
		d, err := time.ParseDuration(ts)
		if err != nil {
			return nil, fmt.Errorf("shell node: parse timeout %q: %w", ts, err)
		}
		if d <= 0 {
			return nil, fmt.Errorf("shell node: timeout must be positive, got %s", d)
		}
		timeout = d
	}
	if timeout > shellMaxTimeout {
		timeout = shellMaxTimeout
	}

	envList, err := normalizeEnv(cfg["env"])
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "sh", "-c", command) // #nosec G204 — 这是节点的设计目的：执行用户配置的命令
	// 不继承宿主进程环境变量；env 为空即一个空 environment 传给子进程。
	if envList == nil {
		envList = []string{}
	}
	cmd.Env = envList

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()

	exitCode := 0
	if runErr != nil {
		// 优先识别超时：context 取消会导致进程被 SIGKILL，cmd.Run() 此时
		// 仍会返回 *exec.ExitError（exit code -1），但语义上是超时。
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("shell node: timeout after %s", timeout)
		}
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			exitCode = ee.ExitCode()
		} else {
			// 启动失败 / 找不到 sh / 其他系统错误
			return nil, fmt.Errorf("shell node: %w", runErr)
		}
	}

	output := map[string]any{
		"stdout":    stdoutBuf.String(),
		"stderr":    stderrBuf.String(),
		"exit_code": exitCode,
	}
	res := &Result{Output: output}

	if exitCode != 0 {
		// 引擎在 runErr != nil 时会把节点标记为 failed，但仍会保留 result.Output。
		return res, fmt.Errorf("shell node: command exited with code %d", exitCode)
	}
	return res, nil
}

// normalizeEnv 把 cfg["env"] 转换为 "KEY=VALUE" 列表。
// 输入支持 map[string]any / map[string]string / nil。键名按字典序排序，
// 让节点的执行结果在测试中可重现。
func normalizeEnv(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	pairs := map[string]string{}
	switch m := v.(type) {
	case map[string]any:
		for k, vv := range m {
			pairs[k] = fmt.Sprint(vv)
		}
	case map[string]string:
		for k, vv := range m {
			pairs[k] = vv
		}
	default:
		return nil, fmt.Errorf("shell node: env must be a map, got %T", v)
	}
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(pairs))
	for _, k := range keys {
		out = append(out, k+"="+pairs[k])
	}
	return out, nil
}
