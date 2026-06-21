// 工作流模式（参考Harness-Starter）
// full/hotfix/tweak 模式切换
package workflow

import "fmt"

// Mode 工作流模式
type Mode string

const (
	ModeFull   Mode = "full"   // 完整检查
	ModeHotfix Mode = "hotfix" // 紧急修复
	ModeTweak  Mode = "tweak"  // 微调
)

// Stage 工作流阶段
type Stage string

const (
	StageDesign Stage = "design" // 设计阶段
	StageFix    Stage = "fix"    // 修复阶段
	StageTest   Stage = "test"   // 测试阶段
)

// Config 模式配置
type Config struct {
	Mode           Mode
	Stage          Stage
	ApprovalLevel  string  // 审批级别（strict/normal/relaxed）
	CheckIntensity float64 // 检查强度 0-1
	AutoFix        bool    // 是否自动修复
	MaxSteps       int     // 最大步数
}

// DefaultConfig 返回默认配置
func DefaultConfig(mode Mode) *Config {
	switch mode {
	case ModeFull:
		return &Config{
			Mode:           ModeFull,
			Stage:          StageDesign,
			ApprovalLevel:  "strict",
			CheckIntensity: 1.0,
			AutoFix:        false,
			MaxSteps:       20,
		}
	case ModeHotfix:
		return &Config{
			Mode:           ModeHotfix,
			Stage:          StageFix,
			ApprovalLevel:  "relaxed",
			CheckIntensity: 0.3,
			AutoFix:        true,
			MaxSteps:       5,
		}
	case ModeTweak:
		return &Config{
			Mode:           ModeTweak,
			Stage:          StageFix,
			ApprovalLevel:  "normal",
			CheckIntensity: 0.5,
			AutoFix:        true,
			MaxSteps:       10,
		}
	default:
		return DefaultConfig(ModeFull)
	}
}

// Workflow 工作流管理器
type Workflow struct {
	config *Config
}

// New 创建工作流管理器
func New(config *Config) *Workflow {
	if config == nil {
		config = DefaultConfig(ModeFull)
	}
	return &Workflow{config: config}
}

// SetMode 切换模式
func (w *Workflow) SetMode(mode Mode) {
	w.config = DefaultConfig(mode)
}

// SetStage 切换阶段
func (w *Workflow) SetStage(stage Stage) {
	w.config.Stage = stage
}

// Config 返回当前配置
func (w *Workflow) Config() *Config {
	return w.config
}

// String 返回模式描述
func (w *Workflow) String() string {
	return fmt.Sprintf("Mode: %s | Stage: %s | Approval: %s | Intensity: %.0f%% | AutoFix: %v",
		w.config.Mode, w.config.Stage, w.config.ApprovalLevel,
		w.config.CheckIntensity*100, w.config.AutoFix)
}
