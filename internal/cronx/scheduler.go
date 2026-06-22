// Cron高级特性 - 无Agent模式/wakeAgent门控/任务链
package cronx

import (
	"fmt"
	"strings"
)

// Job 高级任务
type Job struct {
	ID           string
	Prompt       string
	Schedule     string
	NoAgent      bool           // 无Agent模式：纯脚本，零Token消耗
	WakeGate     *WakeGate      // wakeAgent门控
	ContextFrom  string         // 任务链：从前一个任务注入上下文
	Deliver      string         // 投递目标：platform / broadcast / silent
	ScriptPath   string         // 脚本路径（noAgent=true时使用）
	Skills       []string       // 运行前加载的技能
	MaxRetries   int
	ProviderChain []string      // 提供者回退链
}

// WakeGate wakeAgent门控
type WakeGate struct {
	Type   string // file_change / flag / sql_count
	Config map[string]string
}

// WakeResult 门控检测结果
type WakeResult struct {
	WakeAgent bool
	Reason    string
}

// GateChecker 门控检查器
type GateChecker struct{}

// Check 执行门控检查
func (g *GateChecker) Check(gate *WakeGate) *WakeResult {
	if gate == nil {
		return &WakeResult{WakeAgent: true}
	}

	switch gate.Type {
	case "file_change":
		return g.checkFileChange(gate.Config)
	case "flag":
		return g.checkFlag(gate.Config)
	default:
		return &WakeResult{WakeAgent: true}
	}
}

func (g *GateChecker) checkFileChange(cfg map[string]string) *WakeResult {
	path := cfg["path"]
	if path == "" {
		return &WakeResult{WakeAgent: false, Reason: "no path configured"}
	}
	_ = path
	// 简化实现：始终返回true
	return &WakeResult{WakeAgent: true}
}

func (g *GateChecker) checkFlag(cfg map[string]string) *WakeResult {
	flag := cfg["flag"]
	_ = flag
	return &WakeResult{WakeAgent: true}
}

// Scheduler 高级调度器
type Scheduler struct {
	jobs     []Job
	checker  *GateChecker
}

func NewScheduler() *Scheduler {
	return &Scheduler{jobs: make([]Job, 0), checker: &GateChecker{}}
}

func (s *Scheduler) AddJob(job Job) { s.jobs = append(s.jobs, job) }

// GetDueJobs 获取到期的任务
func (s *Scheduler) GetDueJobs() []Job {
	var due []Job
	for _, j := range s.jobs {
		// 检查门控
		result := s.checker.Check(j.WakeGate)
		if result.WakeAgent {
			due = append(due, j)
		}
	}
	return due
}

// DescribeJob 描述任务
func DescribeJob(job Job) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", job.ID))

	if job.NoAgent {
		parts = append(parts, "⚡无Agent")
	}
	if job.WakeGate != nil {
		parts = append(parts, fmt.Sprintf("🚪%s门控", job.WakeGate.Type))
	}
	if job.ContextFrom != "" {
		parts = append(parts, fmt.Sprintf("🔗链:%s", job.ContextFrom))
	}
	if job.Deliver == "broadcast" {
		parts = append(parts, "📢广播")
	} else if job.Deliver == "silent" {
		parts = append(parts, "🔇静默")
	}

	return strings.Join(parts, " ")
}
