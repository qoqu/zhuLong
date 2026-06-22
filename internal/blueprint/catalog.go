// 自动化蓝图模板系统（参考Hermes blueprint_catalog + cron）
package blueprint

import (
	"fmt"
	"strings"
)

// Slot 蓝图插槽
type Slot struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`     // time/enum/text/weekdays
	Label    string   `yaml:"label"`
	Default  string   `yaml:"default"`
	Options  []string `yaml:"options,omitempty"`
	Required bool     `yaml:"required"`
	Desc     string   `yaml:"desc"`
}

// Blueprint 自动化蓝图
type Blueprint struct {
	Key              string   `yaml:"key"`
	Title            string   `yaml:"title"`
	Description      string   `yaml:"description"`
	Category         string   `yaml:"category"`          // daily/weekly/email/general
	ScheduleTemplate string   `yaml:"schedule_template"` // 带{slot}的cron表达式
	PromptTemplate   string   `yaml:"prompt_template"`   // 带{slot}的种子指令
	Slots            []Slot   `yaml:"slots"`
	Skills           []string `yaml:"skills,omitempty"`  // 运行前加载的技能
}

// FilledBlueprint 填充后的蓝图
type FilledBlueprint struct {
	Blueprint
	Schedule string
	Prompt   string
	Values   map[string]string
}

// Catalog 蓝图目录
type Catalog struct {
	blueprints map[string]*Blueprint
}

// New 创建蓝图目录
func New() *Catalog {
	c := &Catalog{blueprints: make(map[string]*Blueprint)}
	for _, bp := range defaultBlueprints() {
		c.blueprints[bp.Key] = bp
	}
	return c
}

// Get 获取蓝图
func (c *Catalog) Get(key string) (*Blueprint, bool) {
	bp, ok := c.blueprints[key]
	return bp, ok
}

// List 列出所有蓝图
func (c *Catalog) List() []*Blueprint {
	list := make([]*Blueprint, 0, len(c.blueprints))
	for _, bp := range c.blueprints {
		list = append(list, bp)
	}
	return list
}

// ListByCategory 按分类列出
func (c *Catalog) ListByCategory(category string) []*Blueprint {
	var list []*Blueprint
	for _, bp := range c.blueprints {
		if bp.Category == category {
			list = append(list, bp)
		}
	}
	return list
}

// Register 注册自定义蓝图
func (c *Catalog) Register(bp *Blueprint) {
	c.blueprints[bp.Key] = bp
}

// Fill 填充蓝图
func (c *Catalog) Fill(key string, values map[string]string) (*FilledBlueprint, error) {
	bp, ok := c.blueprints[key]
	if !ok {
		return nil, fmt.Errorf("blueprint not found: %s", key)
	}

	// 验证必需插槽
	for _, slot := range bp.Slots {
		if slot.Required {
			if _, ok := values[slot.Name]; !ok {
				return nil, fmt.Errorf("required slot %s is missing", slot.Name)
			}
		}
		if slot.Type == "enum" && values[slot.Name] != "" {
			valid := false
			for _, opt := range slot.Options {
				if values[slot.Name] == opt {
					valid = true
					break
				}
			}
			if !valid {
				return nil, fmt.Errorf("invalid value %s for slot %s, options: %v",
					values[slot.Name], slot.Name, slot.Options)
			}
		}
	}

	// 填充调度计划
	schedule := bp.ScheduleTemplate
	for k, v := range values {
		if v == "" {
			v = getDefault(bp, k)
		}
		schedule = strings.ReplaceAll(schedule, fmt.Sprintf("{%s}", k), v)
	}

	// 填充提示词
	prompt := bp.PromptTemplate
	for k, v := range values {
		if v == "" {
			v = getDefault(bp, k)
		}
		prompt = strings.ReplaceAll(prompt, fmt.Sprintf("{%s}", k), v)
	}

	return &FilledBlueprint{
		Blueprint: *bp,
		Schedule:  schedule,
		Prompt:    prompt,
		Values:    values,
	}, nil
}

func getDefault(bp *Blueprint, slotName string) string {
	for _, s := range bp.Slots {
		if s.Name == slotName {
			return s.Default
		}
	}
	return ""
}

// ========== 内置蓝图 ==========

func defaultBlueprints() []*Blueprint {
	return []*Blueprint{
		{
			Key: "morning-brief", Title: "晨间简报", Category: "daily",
			Description: "每天早上发送项目状态简报",
			ScheduleTemplate: "{minute} {hour} * * *",
			PromptTemplate: "请生成今天({date})的项目晨间简报，包含：\n1. 昨日完成的工作\n2. 今日计划\n3. 阻塞项",
			Slots: []Slot{
				{Name: "minute", Type: "time", Label: "分钟", Default: "0", Required: true},
				{Name: "hour", Type: "time", Label: "小时", Default: "9", Required: true},
				{Name: "date", Type: "text", Label: "日期", Default: "today", Desc: "可指定具体日期"},
			},
		},
		{
			Key: "code-health", Title: "代码健康扫描", Category: "daily",
			Description: "每日自动执行代码质量扫描",
			ScheduleTemplate: "{minute} {hour} * * 1-5",
			PromptTemplate: "请对项目进行代码健康扫描，检查：\n1. TODO/FIXME密度\n2. 测试覆盖率\n3. 依赖健康度\n4. 代码风格问题",
			Slots: []Slot{
				{Name: "minute", Type: "time", Label: "分钟", Default: "30", Required: true},
				{Name: "hour", Type: "time", Label: "小时", Default: "10", Required: true},
			},
			Skills: []string{"gc"},
		},
		{
			Key: "weekly-review", Title: "每周回顾", Category: "weekly",
			Description: "每周自动生成项目进展报告",
			ScheduleTemplate: "{minute} {hour} * * {dow}",
			PromptTemplate: "请生成本周({week})的项目回顾报告，分析：\n1. 完成了哪些工作\n2. 关键指标变化\n3. 下周规划建议",
			Slots: []Slot{
				{Name: "minute", Type: "time", Label: "分钟", Default: "0", Required: true},
				{Name: "hour", Type: "time", Label: "小时", Default: "17", Required: true},
				{Name: "dow", Type: "enum", Label: "星期", Default: "5", Options: []string{"0", "1", "2", "3", "4", "5", "6"}},
				{Name: "week", Type: "text", Label: "周数", Default: "this", Desc: "可指定具体周"},
			},
		},
		{
			Key: "dependency-check", Title: "依赖检查", Category: "weekly",
			Description: "每周检查依赖更新和安全漏洞",
			ScheduleTemplate: "{minute} {hour} * * {dow}",
			PromptTemplate: "请检查项目依赖状态：\n1. 检查go.mod依赖版本\n2. 检查已知安全漏洞\n3. 建议更新内容",
			Slots: []Slot{
				{Name: "minute", Type: "time", Default: "0"},
				{Name: "hour", Type: "time", Default: "14"},
				{Name: "dow", Type: "enum", Default: "1", Options: []string{"0", "1", "2", "3", "4", "5", "6"}},
			},
		},
		{
			Key: "test-runner", Title: "定时运行测试", Category: "general",
			Description: "按计划自动运行测试并报告结果",
			ScheduleTemplate: "{minute} {hour} * * *",
			PromptTemplate: "请运行 go test ./... 并报告：\n1. 通过的测试数\n2. 失败的测试数\n3. 失败详情",
			Slots: []Slot{
				{Name: "minute", Type: "time", Default: "0"},
				{Name: "hour", Type: "time", Default: "6"},
			},
		},
		{
			Key: "backup", Title: "自动备份", Category: "daily",
			Description: "每日自动备份项目关键文件",
			ScheduleTemplate: "{minute} {hour} * * *",
			PromptTemplate: "请执行项目备份：\n1. 备份config/目录\n2. 备份docs/目录\n3. 备份go.{mod,sum}\n4. 将备份打包到 backup/ 目录",
			Slots: []Slot{
				{Name: "minute", Type: "time", Default: "0"},
				{Name: "hour", Type: "time", Default: "3"},
			},
		},
		{
			Key: "report-gen", Title: "定时生成报告", Category: "general",
			Description: "按计划生成指定格式的报告",
			ScheduleTemplate: "{minute} {hour} * * {dow}",
			PromptTemplate: "请生成{type}报告：\n主题：{topic}\n格式：{format}",
			Slots: []Slot{
				{Name: "minute", Type: "time", Default: "0"},
				{Name: "hour", Type: "time", Default: "12"},
				{Name: "dow", Type: "enum", Default: "5", Options: []string{"0", "1", "2", "3", "4", "5", "6"}},
				{Name: "type", Type: "enum", Label: "报告类型", Default: "progress", Options: []string{"progress", "analysis", "summary"}},
				{Name: "topic", Type: "text", Label: "主题", Default: "project", Required: true},
				{Name: "format", Type: "enum", Label: "格式", Default: "markdown", Options: []string{"markdown", "html", "json"}},
			},
		},
	}
}

// ParseCron 解析cron表达式（简化版）
func ParseCron(expr string) (map[string]string, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("invalid cron expression: %s (need 5 fields)", expr)
	}

	return map[string]string{
		"minute": fields[0],
		"hour":   fields[1],
		"day":    fields[2],
		"month":  fields[3],
		"dow":    fields[4],
	}, nil
}

// DescribeSchedule 描述调度计划（人类可读）
func DescribeSchedule(schedule string) string {
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return schedule
	}

	minute, hour, day, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]

	if day == "*" && month == "*" && dow == "*" {
		if hour == "*" && minute == "*" {
			return "每分钟"
		}
		return fmt.Sprintf("每天 %s:%s", padZero(hour), padZero(minute))
	}
	if day == "*" && month == "*" && dow != "*" {
		days := map[string]string{"0": "周日", "1": "周一", "2": "周二", "3": "周三", "4": "周四", "5": "周五", "6": "周六"}
		d := days[dow]
		if d == "" {
			d = "周" + dow
		}
		return fmt.Sprintf("每%s %s:%s", d, padZero(hour), padZero(minute))
	}
	return schedule
}

func padZero(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}
