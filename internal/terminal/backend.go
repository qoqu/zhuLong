// 终端执行后端 - 策略模式（local/docker/ssh/singularity/modal/daytona）
package terminal

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BackendType 后端类型
type BackendType string

const (
	BackendLocal      BackendType = "local"       // 本地执行
	BackendDocker     BackendType = "docker"      // Docker容器
	BackendSSH        BackendType = "ssh"         // SSH远程
	BackendSingularity BackendType = "singularity" // Singularity容器
	BackendModal      BackendType = "modal"       // Modal云端
	BackendDaytona    BackendType = "daytona"     // Daytona云端
)

// Result 执行结果
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration string
	PID      int
}

// Backend 终端后端接口
type Backend interface {
	Type() BackendType
	Name() string
	Execute(ctx context.Context, command string, args []string, env map[string]string) (*Result, error)
	IsAvailable() bool
	Close() error
}

// ========== Local 后端 ==========

type LocalBackend struct{}

func NewLocalBackend() *LocalBackend { return &LocalBackend{} }

func (b *LocalBackend) Type() BackendType { return BackendLocal }
func (b *LocalBackend) Name() string      { return "Local" }

func (b *LocalBackend) Execute(ctx context.Context, command string, args []string, env map[string]string) (*Result, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	return &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		PID:      cmd.Process.Pid,
	}, nil
}

func (b *LocalBackend) IsAvailable() bool { return true }

func (b *LocalBackend) Close() error { return nil }

// ========== Docker 后端 ==========

type DockerBackend struct {
	image   string
	pull    bool
	network string
	volumes []string
}

func NewDockerBackend(image string) *DockerBackend {
	return &DockerBackend{
		image:   image,
		pull:    false,
		network: "none",
		volumes: []string{},
	}
}

func (b *DockerBackend) Type() BackendType { return BackendDocker }
func (b *DockerBackend) Name() string      { return fmt.Sprintf("Docker (%s)", b.image) }

func (b *DockerBackend) Execute(ctx context.Context, command string, args []string, env map[string]string) (*Result, error) {
	dockerArgs := []string{"run", "--rm", "-i"}

	// 安全配置
	dockerArgs = append(dockerArgs, "--cap-drop=ALL")
	dockerArgs = append(dockerArgs, "--security-opt=no-new-privileges")
	dockerArgs = append(dockerArgs, "--pids-limit=100")

	if b.network != "" {
		dockerArgs = append(dockerArgs, "--network", b.network)
	}
	for _, v := range b.volumes {
		dockerArgs = append(dockerArgs, "-v", v)
	}
	for k, v := range env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	dockerArgs = append(dockerArgs, b.image, command)
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("docker execute: %w", err)
		}
	}

	return &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

func (b *DockerBackend) IsAvailable() bool {
	cmd := exec.Command("docker", "--version")
	return cmd.Run() == nil
}

func (b *DockerBackend) Close() error { return nil }

// ========== SSH 后端 ==========

type SSHBackend struct {
	host      string
	port      int
	user      string
	keyPath   string
}

func NewSSHBackend(host, user string, port int) *SSHBackend {
	return &SSHBackend{
		host: host,
		user: user,
		port: port,
	}
}

func (b *SSHBackend) Type() BackendType { return BackendSSH }
func (b *SSHBackend) Name() string      { return fmt.Sprintf("SSH (%s@%s)", b.user, b.host) }

func (b *SSHBackend) Execute(ctx context.Context, command string, args []string, env map[string]string) (*Result, error) {
	sshArgs := []string{"-p", fmt.Sprintf("%d", b.port)}
	if b.keyPath != "" {
		sshArgs = append(sshArgs, "-i", b.keyPath)
	}

	// 构建远程命令
	var envStrs []string
	for k, v := range env {
		envStrs = append(envStrs, fmt.Sprintf("%s=%s", k, v))
	}
	envPrefix := strings.Join(envStrs, " ") + " "
	remoteCmd := envPrefix + command + " " + strings.Join(args, " ")

	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", b.user, b.host), remoteCmd)

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("ssh execute: %w", err)
		}
	}

	return &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

func (b *SSHBackend) IsAvailable() bool {
	return true // 延迟到执行时检查
}

func (b *SSHBackend) Close() error { return nil }

// ========== 管理器 ==========

// Manager 终端后端管理器
type Manager struct {
	backends map[BackendType]Backend
	current  BackendType
}

// NewManager 创建管理器
func NewManager() *Manager {
	m := &Manager{
		backends: make(map[BackendType]Backend),
		current:  BackendLocal,
	}
	// 默认注册local
	m.Register(NewLocalBackend())
	return m
}

// Register 注册后端
func (m *Manager) Register(b Backend) {
	m.backends[b.Type()] = b
}

// SetCurrent 切换当前后端
func (m *Manager) SetCurrent(t BackendType) error {
	b, ok := m.backends[t]
	if !ok {
		return fmt.Errorf("backend %s not registered", t)
	}
	if !b.IsAvailable() {
		return fmt.Errorf("backend %s is not available", t)
	}
	m.current = t
	return nil
}

// Execute 在当前后端执行命令
func (m *Manager) Execute(ctx context.Context, command string, args ...string) (*Result, error) {
	b, ok := m.backends[m.current]
	if !ok {
		return nil, fmt.Errorf("current backend %s not found", m.current)
	}
	return b.Execute(ctx, command, args, nil)
}

// ExecuteWithEnv 带环境变量执行
func (m *Manager) ExecuteWithEnv(ctx context.Context, command string, args []string, env map[string]string) (*Result, error) {
	b, ok := m.backends[m.current]
	if !ok {
		return nil, fmt.Errorf("current backend %s not found", m.current)
	}
	return b.Execute(ctx, command, args, env)
}

// Current 获取当前后端
func (m *Manager) Current() Backend {
	return m.backends[m.current]
}

// List 列出所有注册的后端
func (m *Manager) List() []Backend {
	var list []Backend
	for _, b := range m.backends {
		list = append(list, b)
	}
	return list
}
