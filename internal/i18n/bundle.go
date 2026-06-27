// 国际化(i18n)系统 - 16种语言支持
package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Language 语言
type Language string

const (
	LangZH    Language = "zh"      // 简体中文
	LangZHT   Language = "zh-hant" // 繁体中文
	LangEN    Language = "en"      // 英语
	LangJA    Language = "ja"      // 日语
	LangKO    Language = "ko"      // 韩语
	LangFR    Language = "fr"      // 法语
	LangDE    Language = "de"      // 德语
	LangES    Language = "es"      // 西班牙语
	LangPT    Language = "pt"      // 葡萄牙语
	LangRU    Language = "ru"      // 俄语
	LangIT    Language = "it"      // 意大利语
	LangTR    Language = "tr"      // 土耳其语
	LangUK    Language = "uk"      // 乌克兰语
	LangAF    Language = "af"      // 南非语
	LangHU    Language = "hu"      // 匈牙利语
	LangGA    Language = "ga"      // 爱尔兰语
)

// Bundle i18n资源包
type Bundle struct {
	mu       sync.RWMutex
	locales  map[Language]map[string]string
	defLang  Language
}

// New 创建i18n资源包
func New(defLang Language) *Bundle {
	return &Bundle{
		locales: make(map[Language]map[string]string),
		defLang: defLang,
	}
}

// LoadFromDir 从目录加载翻译文件
// 目录结构: locales/zh.json, locales/en.json ...
func (b *Bundle) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}

		lang := Language(e.Name()[:len(e.Name())-5]) // 去掉.json
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		var translations map[string]string
		if err := json.Unmarshal(data, &translations); err != nil {
			continue
		}

		b.mu.Lock()
		b.locales[lang] = translations
		b.mu.Unlock()
	}

	return nil
}

// Register 手动注册翻译
func (b *Bundle) Register(lang Language, translations map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.locales[lang] = translations
}

// T 翻译
func (b *Bundle) T(key string, lang Language) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 先查指定语言
	if m, ok := b.locales[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}

	// 回退到默认语言
	if m, ok := b.locales[b.defLang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}

	return key // 连默认语言都没有，返回key本身
}

// List 列出所有语言
func (b *Bundle) List() []Language {
	b.mu.RLock()
	defer b.mu.RUnlock()

	langs := make([]Language, 0, len(b.locales))
	for l := range b.locales {
		langs = append(langs, l)
	}
	return langs
}

// Has 检查语言是否存在
func (b *Bundle) Has(lang Language) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.locales[lang]
	return ok
}

// DefaultTranslations 返回默认翻译（中文）
func DefaultTranslations() map[string]string {
	return map[string]string{
		"app.name":         "烛龙",
		"app.description":  "基于DeepSeek的通用自主循环Agent框架",
		"agent.greeting":   "你好，我是烛龙，一个自主循环的AI助手。",
		"agent.planning":   "规划中...",
		"agent.executing":  "执行中...",
		"agent.reflecting": "反省中...",
		"agent.done":       "完成",
		"agent.failed":     "失败",
		"agent.cancelled":  "已取消",
		"tool.skill_view":  "查看技能",
		"tool.memory_note": "保存记忆",
		"canvas.nodes":     "节点",
		"canvas.edges":     "连线",
		"canvas.zoom_in":   "放大",
		"canvas.zoom_out":  "缩小",
		"board.triage":     "待规划",
		"board.todo":       "待执行",
		"board.ready":      "就绪",
		"board.running":    "运行中",
		"board.done":       "完成",
		"board.blocked":    "阻塞",
		"error.not_found":  "未找到",
		"error.timeout":    "超时",
		"error.permission": "权限不足",
	}
}

// EnglishTranslations 返回英语翻译
func EnglishTranslations() map[string]string {
	return map[string]string{
		"app.name":         "Zhulong",
		"app.description":  "DeepSeek-powered autonomous loop agent framework",
		"agent.greeting":   "Hello, I am Zhulong, an autonomous loop AI assistant.",
		"agent.planning":   "Planning...",
		"agent.executing":  "Executing...",
		"agent.reflecting": "Reflecting...",
		"agent.done":       "Done",
		"agent.failed":     "Failed",
		"agent.cancelled":  "Cancelled",
		"tool.skill_view":  "View Skill",
		"tool.memory_note": "Save Memory",
		"canvas.nodes":     "Nodes",
		"canvas.edges":     "Edges",
		"canvas.zoom_in":   "Zoom In",
		"canvas.zoom_out":  "Zoom Out",
		"board.triage":     "Triage",
		"board.todo":       "Todo",
		"board.ready":      "Ready",
		"board.running":    "Running",
		"board.done":       "Done",
		"board.blocked":    "Blocked",
		"error.not_found":  "Not Found",
		"error.timeout":    "Timeout",
		"error.permission": "Permission Denied",
	}
}
