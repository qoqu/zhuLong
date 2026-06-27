package pkg

// Integration 描述一个可选模块的集成点
type Integration struct {
	Name     string
	Enabled  bool
	Priority int // 越小越先加载
}

// DefaultIntegrations 返回所有可选模块的默认配置
// 这些模块已完整实现但未接入核心 Agent 循环，默认全部禁用
func DefaultIntegrations() []Integration {
	return []Integration{
		{Name: "state", Enabled: false, Priority: 10},    // 热/冷状态持久化
		{Name: "loop", Enabled: false, Priority: 11},     // 自治循环引擎
		{Name: "scheduler", Enabled: false, Priority: 20}, // 事件驱动调度器
		{Name: "cronx", Enabled: false, Priority: 21},    // 高级定时任务
		{Name: "voice", Enabled: false, Priority: 30},    // STT/TTS 语音引擎
		{Name: "board", Enabled: false, Priority: 31},    // 看板引擎（scheduler 依赖）
		{Name: "health", Enabled: false, Priority: 40},   // 健康检查
		{Name: "qa", Enabled: false, Priority: 41},       // 集成测试框架
		{Name: "hub", Enabled: false, Priority: 50},      // 技能市场
		{Name: "acp", Enabled: false, Priority: 60},      // IDE 集成（JSON-RPC）
		{Name: "upgrade", Enabled: false, Priority: 70},  // 版本升级检查
	}
}

// IntegrationMap 将 Integration 列表转换为 map，便于查找
func IntegrationMap(integrations []Integration) map[string]bool {
	m := make(map[string]bool, len(integrations))
	for _, i := range integrations {
		m[i.Name] = i.Enabled
	}
	return m
}
