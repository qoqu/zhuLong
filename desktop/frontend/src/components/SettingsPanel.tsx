import { useState, useEffect } from 'react'
import type { Language, ExecutionMode } from '../types'
import { resolveLanguage } from '../i18n'

const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null

type SettingsTab = 'general' | 'model' | 'bots' | 'mcp' | 'memory' | 'hooks' | 'permissions' | 'sandbox' | 'network' | 'appearance' | 'updates'

interface SettingsPanelProps {
  language: Language
  darkMode: boolean
  executionMode: ExecutionMode
  model: string
  desktopStyle?: 'classic' | 'workspace'
  onLanguageChange: (l: Language) => void
  onToggleDarkMode: () => void
  onModelChange: (m: string) => void
  onExecutionModeChange: (m: ExecutionMode) => void
  onDesktopStyleChange?: (s: 'classic' | 'workspace') => void
  onClose: () => void
}

// ─── Bot 类型定义 ───
type BotPlatform = 'qq' | 'feishu' | 'lark' | 'wechat' | 'telegram' | 'discord' | 'slack' | 'webhook' | 'github' | 'gitlab'

interface BotConfig {
  platform: BotPlatform
  name: string
  enabled: boolean
  connected: boolean
  config: Record<string, string>
}

// ─── Model Provider 类型 ───
interface ModelInfo {
  id: string
  name: string
  enabled: boolean
}

interface Provider {
  id: string
  name: string
  type: 'official' | 'custom'
  source: 'builtin' | 'user'
  keySet: boolean
  description: string
  apiType: string
  baseUrl: string
  apiKeyEnv: string
  models: ModelInfo[]
}

// ─── LocalStorage Keys ───
const LS_PREFIX = 'zhulong-settings_'

function loadSetting<T>(key: string, fallback: T): T {
  try {
    const v = localStorage.getItem(LS_PREFIX + key)
    return v ? JSON.parse(v) : fallback
  } catch { return fallback }
}
function saveSetting(key: string, value: unknown) {
  try { localStorage.setItem(LS_PREFIX + key, JSON.stringify(value)) } catch {}
}

export function SettingsPanel(props: SettingsPanelProps) {
  const isZh = resolveLanguage(props.language) === 'zh'
  const [tab, setTab] = useState<SettingsTab>(loadSetting('lastTab', 'general') as SettingsTab)

  useEffect(() => { saveSetting('lastTab', tab) }, [tab])

  return (
    <div className="settings-overlay">
      <div className="settings-dialog">
        <div className="settings-dialog__header">
          <h2 className="settings-dialog__title">{isZh ? '设置' : 'Settings'}</h2>
          <button className="settings-dialog__close" onClick={props.onClose}>✕</button>
        </div>

        <div className="settings-dialog__body">
          <div className="settings-dialog__sidebar">
            <NavBtn tab={tab} target='general' label={isZh ? '通用' : 'General'} desc={isZh ? '语言、主题和默认设置' : 'Language, theme & defaults'} onClick={setTab} />
            <NavBtn tab={tab} target='model' label={isZh ? '模型' : 'Model'} desc={props.model} onClick={setTab} />
            <NavBtn tab={tab} target='bots' label={isZh ? '机器人' : 'Bots'} desc={isZh ? 'IM Bot 渠道' : 'IM Bot channels'} onClick={setTab} />
            <NavBtn tab={tab} target='mcp' label={isZh ? 'MCP 与工具' : 'MCP & Tools'} desc={isZh ? 'MCP 服务器' : 'MCP Servers'} onClick={setTab} />
            <NavBtn tab={tab} target='memory' label={isZh ? '记忆' : 'Memory'} desc={isZh ? '项目与全局' : 'Project & Global'} onClick={setTab} />
            <NavBtn tab={tab} target='hooks' label={isZh ? 'Hooks' : 'Hooks'} desc={isZh ? 'Shell 自动化' : 'Shell Automation'} onClick={setTab} />
            <NavBtn tab={tab} target='permissions' label={isZh ? '权限' : 'Permissions'} desc={isZh ? '写操作前询问' : 'Write before asking'} onClick={setTab} />
            <NavBtn tab={tab} target='sandbox' label={isZh ? '沙箱' : 'Sandbox'} desc={isZh ? '强制隔离' : 'Force Isolation'} onClick={setTab} />
            <NavBtn tab={tab} target='network' label={isZh ? '网络' : 'Network'} desc={isZh ? '自动' : 'Auto'} onClick={setTab} />
            <NavBtn tab={tab} target='appearance' label={isZh ? '外观' : 'Appearance'} desc={isZh ? '主题 · 强调色 · 字号 · 字体' : 'Theme · Accent · Size · Font'} onClick={setTab} />
            <NavBtn tab={tab} target='updates' label={isZh ? '更新' : 'Updates'} desc={isZh ? '版本 · 配置' : 'Version · Config'} onClick={setTab} />
          </div>

          <div className="settings-dialog__content">
            {tab === 'general' && <GeneralSettings {...props} />}
            {tab === 'model' && <ModelSettings {...props} />}
            {tab === 'bots' && <BotsSettings language={props.language} />}
            {tab === 'mcp' && <McpSettings language={props.language} />}
            {tab === 'memory' && <MemorySettings language={props.language} />}
            {tab === 'hooks' && <HooksSettings language={props.language} />}
            {tab === 'permissions' && <PermissionsSettings language={props.language} />}
            {tab === 'sandbox' && <SandboxSettings language={props.language} />}
            {tab === 'network' && <NetworkSettings language={props.language} />}
            {tab === 'appearance' && <AppearanceSettings language={props.language} darkMode={props.darkMode} onToggleDarkMode={props.onToggleDarkMode} />}
            {tab === 'updates' && <UpdatesSettings language={props.language} />}
          </div>
        </div>
      </div>
    </div>
  )
}

function NavBtn({ tab, target, label, desc, onClick }: { tab: string; target: string; label: string; desc: string; onClick: (t: SettingsTab) => void }) {
  return (
    <button
      className={`settings-dialog__nav ${tab === target ? 'active' : ''}`}
      onClick={() => onClick(target as SettingsTab)}
    >
      <span className="settings-dialog__nav-title">{label}</span>
      <span className="settings-dialog__nav-desc">{desc}</span>
    </button>
  )
}

// ════════════════════════════════════════
// General Settings — 全部功能化
// ══════════════════════════════════════

function GeneralSettings(props: SettingsPanelProps) {
  const isZh = resolveLanguage(props.language) === 'zh'

  // ── 持久化状态 ──
  const [desktopStyle, setDesktopStyle] = useState<'classic' | 'workspace'>(() => loadSetting('desktopStyle', props.desktopStyle || 'classic'))
  const [closeBehavior, setCloseBehavior] = useState(() => loadSetting('closeBehavior', 'background') as string)
  const [sessionDisplay, setSessionDisplay] = useState(() => loadSetting('sessionDisplay', 'standard') as string)
  const [autoPlan, setAutoPlan] = useState(() => loadSetting('autoPlan', 'off') as string)
  const [sound, setSound] = useState(() => loadSetting('sound', 'off') as string)

  function persistAndSet<T>(key: string, setter: (v: T) => void, val: T) {
    setter(val)
    saveSetting(key, val)
  }

  function handleDesktopStyleChange(val: 'classic' | 'workspace') {
    persistAndSet('desktopStyle', setDesktopStyle, val)
    if (props.onDesktopStyleChange) props.onDesktopStyleChange(val)
  }

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZh ? '通用' : 'General'}</h3>
      <p className="settings-page__desc">{isZh ? '语言、主题、关闭行为等全局偏好。' : 'Global preferences: language, theme, close behavior, etc.'}</p>

      <div className="settings-group">
        <div className="settings-group__title">{isZh ? '外观与语言' : 'Appearance & Language'}</div>

        {/* 语言 */}
        <Row label={isZh ? '语言' : 'Language'}>
          <OptionPills options={[
            { value: 'auto', label: isZh ? '自动(跟随系统)' : 'Auto(System)' },
            { value: 'zh', label: '中文' },
            { value: 'en', label: 'English' },
          ]} selected={props.language} onChange={(v) => props.onLanguageChange(v as Language)} />
        </Row>

        {/* 主题 */}
        <Row label={isZh ? '主题' : 'Theme'}>
          <OptionPills options={[
            { value: 'light', label: isZh ? '浅色' : 'Light' },
            { value: 'dark', label: isZh ? '深色' : 'Dark' },
          ]} selected={props.darkMode ? 'dark' : 'light'} onChange={(v) => {
            if ((v === 'dark') !== props.darkMode) props.onToggleDarkMode()
          }} />
        </Row>

        {/* 桌面风格 → 联动 App */}
        <Row label={isZh ? '桌面风格' : 'Style'} hint={isZh ? '工作台：树形项目视图；经典：搜索+扁平列表' : 'Workspace: tree view; Classic: search + flat list'}>
          <OptionPills options={[
            { value: 'classic', label: isZnZz(isZh, '经典', 'Classic') },
            { value: 'workspace', label: isZnZz(isZh, '工作台', 'Workspace') },
          ]} selected={desktopStyle} onChange={(v) => handleDesktopStyleChange(v as 'classic' | 'workspace')} />
        </Row>
      </div>

      <div className="settings-group">
        <div className="settings-group__title">{isZh ? '行为' : 'Behavior'}</div>

        <Row label={isZh ? '关闭窗口时' : 'On Close'}>
          <OptionPills options={[
            { value: 'background', label: isZh ? '保持后台' : 'Background' },
            { value: 'quit', label: isZh ? '退出' : 'Quit' },
          ]} selected={closeBehavior} onChange={(v) => persistAndSet('closeBehavior', setCloseBehavior, v)} />
        </Row>

        <Row label={isZh ? '会话展示' : 'Session Display'}>
          <OptionPills options={[
            { value: 'standard', label: isZh ? '标准' : 'Standard' },
            { value: 'compact', label: isZh ? '紧凑' : 'Compact' },
          ]} selected={sessionDisplay} onChange={(v) => persistAndSet('sessionDisplay', setSessionDisplay, v)} />
        </Row>

        <Row label={isZh ? '自动规划' : 'Auto Plan'} hint={isZh ? 'Agent 启动时是否自动生成执行计划' : 'Whether agent auto-generates execution plan on start'}>
          <OptionPills options={[
            { value: 'on', label: isZh ? '开启' : 'On' },
            { value: 'off', label: isZh ? '关闭' : 'Off' },
          ]} selected={autoPlan} onChange={(v) => persistAndSet('autoPlan', setAutoPlan, v)} />
        </Row>
      </div>

      <div className="settings-group">
        <div className="settings-group__title">{isZh ? '通知与界面' : 'Notification & UI'}</div>

        <Row label={isZh ? '声音' : 'Sound'}>
          <select className="settings-select" value={sound} onChange={(e) => persistAndSet('sound', setSound, e.target.value)}>
            <option value="off">{isZh ? '全部关闭' : 'All Off'}</option>
            <option value="notify">{isZh ? '仅通知' : 'Notifications Only'}</option>
            <option value="all">{isZh ? '全部开启' : 'All On'}</option>
          </select>
        </Row>
      </div>

      {/* 备份控制 */}
      <BackupControl language={props.language} />
    </div>
  )
}

// ─── 备份控制组件 ───
function BackupControl({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null
  const [mode, setMode] = useState<string>(() => loadJSON('zhulong-backup-mode', 'immediate'))
  const [onFail, setOnFail] = useState<boolean>(() => loadJSON('zhulong-backup-onfail', false))
  const [backing, setBacking] = useState(false)

  useEffect(() => {
    saveJSON('zhulong-backup-mode', mode)
    if (backend) try { backend.SetBackupMode(mode) } catch {}
  }, [mode])
  useEffect(() => { saveJSON('zhulong-backup-onfail', onFail) }, [onFail])

  const handleBackup = async () => {
    if (!backend) return
    setBacking(true)
    try {
      await backend.TriggerBackup()
    } catch {}
    setBacking(false)
  }

  return (
    <div className="settings-group">
      <div className="settings-group__title">{isZnZz(isZh, '备份', 'Backup')}</div>
      <Row label={isZnZz(isZh, '备份模式', 'Backup Mode')}>
        <select className="settings-select" value={mode}
          onChange={async (e) => {
            setMode(e.target.value)
            if (backend) try { await backend.SetBackupMode(e.target.value, onFail) } catch {}
          }}>
          <option value="immediate">{isZnZz(isZh, '即时备份（每轮结束自动触发）', 'Immediate (auto after each turn)')}</option>
          <option value="on-completion">{isZnZz(isZh, '完成备份（任务结束时触发）', 'On Completion (after task)')}</option>
          <option value="off">{isZnZz(isZh, '关闭', 'Off')}</option>
        </select>
      </Row>
      <Row label={isZnZz(isZh, '失败时备份', 'Backup on Failure')} hint={isZnZz(isZh, '执行失败时自动保存快照，便于问题排查。', 'Auto-save snapshot on failure for debugging.')}>
        <OptionPills options={[
          { value: 'on', label: isZh ? '开启' : 'On' },
          { value: 'off', label: isZh ? '关闭' : 'Off' },
        ]} selected={onFail ? 'on' : 'off'} onChange={async (v) => {
          const val = v === 'on'
          setOnFail(val)
          if (backend) try { await backend.SetBackupMode(mode, val) } catch {}
        }} />
      </Row>
      <Row label={isZnZz(isZh, '手动备份', 'Manual Backup')}>
        <button className="settings-btn settings-btn--outline settings-btn--sm" onClick={handleBackup} disabled={backing}>
          {backing ? (isZh ? '备份中...' : 'Backing up...') : (isZh ? '创建备份' : 'Create Backup')}
        </button>
      </Row>
    </div>
  )
}

// ════════════════════════════════════════
// Model Settings — 参考 Reasonix 细化
// ════════════════════════════════════════

const DS_MODELS = [
  { id: 'deepseek-v4-flash', name: 'deepseek-v4-flash', provider: 'DeepSeek Official', providerShort: (isZh: boolean) => isZh ? 'DeepSeek 官方' : 'DeepSeek Official' },
  { id: 'deepseek-v4-pro', name: 'deepseek-v4-pro', provider: 'DeepSeek Official', providerShort: (isZn: boolean) => isZn ? 'DeepSeek 官方' : 'DeepSeek Official' },
]

// ─── 默认供应商（内置 DeepSeek）───
const DEFAULT_PROVIDERS: Provider[] = [
  {
    id: 'deepseek-official',
    name: 'DeepSeek',
    type: 'official',
    source: 'builtin',
    keySet: true,
    description: 'DeepSeek 官方 OpenAI-compatible 接入',
    apiType: 'openai',
    baseUrl: 'https://api.deepseek.com',
    apiKeyEnv: 'DEEPSEEK_API_KEY',
    models: [
      { id: 'deepseek-v4-flash', name: 'deepseek-v4-flash', enabled: true },
      { id: 'deepseek-v4-pro', name: 'deepseek-v4-pro', enabled: true },
    ],
  },
]

// ─── 预置模型库（用户添加新 Provider 时可选择）───
const PRESET_MODELS: Record<string, { name: string; apiType: string; baseUrl: string; apiKeyEnv: string; models: { id: string; name: string }[] }> = {
  'openai': {
    name: 'OpenAI',
    apiType: 'openai',
    baseUrl: 'https://api.openai.com/v1',
    apiKeyEnv: 'OPENAI_API_KEY',
    models: [
      { id: 'gpt-4o', name: 'GPT-4o' },
      { id: 'gpt-4o-mini', name: 'GPT-4o Mini' },
      { id: 'gpt-4-turbo', name: 'GPT-4 Turbo' },
      { id: 'o1', name: 'o1' },
      { id: 'o1-mini', name: 'o1 Mini' },
      { id: 'o3-mini', name: 'o3 Mini' },
    ],
  },
  'anthropic': {
    name: 'Anthropic',
    apiType: 'anthropic',
    baseUrl: 'https://api.anthropic.com/v1',
    apiKeyEnv: 'ANTHROPIC_API_KEY',
    models: [
      { id: 'claude-opus-4-20250514', name: 'Claude Opus 4' },
      { id: 'claude-sonnet-4-20250514', name: 'Claude Sonnet 4' },
      { id: 'claude-3-5-sonnet-20241022', name: 'Claude 3.5 Sonnet' },
      { id: 'claude-3-5-haiku-20241022', name: 'Claude 3.5 Haiku' },
    ],
  },
  'google': {
    name: 'Google',
    apiType: 'openai',
    baseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai',
    apiKeyEnv: 'GOOGLE_API_KEY',
    models: [
      { id: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro' },
      { id: 'gemini-2.5-flash', name: 'Gemini 2.5 Flash' },
      { id: 'gemini-2.0-flash', name: 'Gemini 2.0 Flash' },
    ],
  },
  'deepseek-official': {
    name: 'DeepSeek',
    apiType: 'openai',
    baseUrl: 'https://api.deepseek.com',
    apiKeyEnv: 'DEEPSEEK_API_KEY',
    models: [
      { id: 'deepseek-v4-flash', name: 'deepseek-v4-flash' },
      { id: 'deepseek-v4-pro', name: 'deepseek-v4-pro' },
      { id: 'deepseek-reasoner', name: 'deepseek-reasoner' },
    ],
  },
  'openrouter': {
    name: 'OpenRouter',
    apiType: 'openai',
    baseUrl: 'https://openrouter.ai/api/v1',
    apiKeyEnv: 'OPENROUTER_API_KEY',
    models: [
      { id: 'anthropic/claude-opus-4', name: 'Claude Opus 4 (via OpenRouter)' },
      { id: 'openai/gpt-4o', name: 'GPT-4o (via OpenRouter)' },
      { id: 'google/gemini-2.5-pro', name: 'Gemini 2.5 Pro (via OpenRouter)' },
      { id: 'meta-llama/llama-4-maverick', name: 'Llama 4 Maverick (via OpenRouter)' },
    ],
  },
  'custom': {
    name: '自定义',
    apiType: 'openai',
    baseUrl: '',
    apiKeyEnv: '',
    models: [],
  },
}

function ModelSettings(props: SettingsPanelProps) {
  const isZh = resolveLanguage(props.language) === 'zh'

  // ── 子 Tab 状态 ──
  type ModelSubTab = 'usage' | 'access'
  const [subTab, setSubTab] = useState<ModelSubTab>('usage')

  // ── 持久化模型配置 ──
  const [planningModel, setPlanningModel] = useState<string>(() => loadSetting('planningModel', 'same'))
  const [subAgentModel, setSubAgentModel] = useState<string>(() => loadSetting('subAgentModel', 'same'))
  const [subAgentEffort, setSubAgentEffort] = useState<string>(() => loadSetting('subAgentEffort', 'auto'))
  const [maxExecRounds, setMaxExecRounds] = useState<string>(() => loadSetting('maxExecRounds', '0'))
  const [maxPlanRounds, setMaxPlanRounds] = useState<string>(() => loadSetting('maxPlanRounds', '12'))
  const [coldStartTrim, setColdStartTrim] = useState<string>(() => loadSetting('coldStartTrim', 'on'))
  const [thinkingLang, setThinkingLang] = useState<string>(() => loadSetting('thinkingLang', 'auto'))

  // ── Provider 状态 ──
  const [providers, setProviders] = useState<Provider[]>(() => {
    try {
      const saved = localStorage.getItem('zhulong-model-providers')
      return saved ? JSON.parse(saved) : DEFAULT_PROVIDERS
    } catch { return DEFAULT_PROVIDERS }
  })
  const [showAddModal, setShowAddModal] = useState(false)
  const [addPreset, setAddPreset] = useState<string>('openai')
  const [addCustom, setAddCustom] = useState({ name: '', baseUrl: '', apiKeyEnv: '', modelsText: '' })

  const saveProviders = (ps: Provider[]) => {
    setProviders(ps)
    localStorage.setItem('zhulong-model-providers', JSON.stringify(ps))
  }

  const handleAddProvider = () => {
    const preset = PRESET_MODELS[addPreset]
    if (!preset) return

    const newId = `${addPreset}-${Date.now()}`
    const models: ModelInfo[] = []

    if (addPreset === 'custom') {
      // 自定义：解析用户输入的模型列表
      const lines = addCustom.modelsText.split('\n').filter(l => l.trim())
      for (const line of lines) {
        const [id, name] = line.split(',').map(s => s.trim())
        if (id) models.push({ id, name: name || id, enabled: true })
      }
      if (models.length === 0) return
    } else {
      for (const m of preset.models) {
        models.push({ id: m.id, name: m.name, enabled: true })
      }
    }

    const newProvider: Provider = {
      id: newId,
      name: addPreset === 'custom' ? addCustom.name : preset.name,
      type: 'official',
      source: 'user',
      keySet: false,
      description: addPreset === 'custom'
        ? `${addCustom.name} — ${addCustom.baseUrl}`
        : `${preset.name} OpenAI-compatible 接入`,
      apiType: preset.apiType,
      baseUrl: addPreset === 'custom' ? addCustom.baseUrl : preset.baseUrl,
      apiKeyEnv: addPreset === 'custom' ? addCustom.apiKeyEnv : preset.apiKeyEnv,
      models,
    }

    saveProviders([...providers, newProvider])
    setShowAddModal(false)
    setAddCustom({ name: '', baseUrl: '', apiKeyEnv: '', modelsText: '' })
  }

  const toggleModel = (providerId: string, modelId: string) => {
    saveProviders(providers.map(p =>
      p.id !== providerId ? p : {
        ...p,
        models: p.models.map(m =>
          m.id !== modelId ? m : { ...m, enabled: !m.enabled }
        )
      }
    ))
  }

  const removeProvider = (id: string) => {
    if (!isZh && !window.confirm('Remove this provider?')) return
    if (isZh && !window.confirm('\u786e\u5b9a\u8981\u79fb\u9664\u6b64\u4f9b\u5e94\u554f\uff1f')) return
    saveProviders(providers.filter(p => p.id !== id))
  }

  // 同步到后端
  async function ps<T extends string | number>(k: string, s: (v: T) => void, v: T) {
    s(v)
    saveSetting(k, v)
    if (backend) {
      try {
        await backend.SetConfigField(k, v)
      } catch (e) { console.error('Failed to sync config:', e) }
    }
  }

  const currentProvider = DS_MODELS.find(m => m.id === props.model)?.provider || ''

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, '\u6a21\u578b', 'Model')}</h3>
      <p className="settings-page__desc">{isZnZz(isZh, '\u9ed8\u8ba4\u6a21\u578b\u3001\u89c4\u5212\u6a21\u578b\u3001\u8fd0\u884c\u4e0a9\u9650\u4e0e\u63a5\u5165\u6982\u89c8\u3002', 'Default model, planning model, runtime limits & provider overview.')}</p>

      {/* ═══ 子 Tab 栏 ═══ */}
      <div className="model-subtabs">
        <button
          className={`model-subtab ${subTab === 'usage' ? 'active' : ''}`}
          onClick={() => setSubTab('usage')}
        >
          {isZnZz(isZh, '\u4f7f\u7528', 'Usage')}
        </button>
        <button
          className={`model-subtab ${subTab === 'access' ? 'active' : ''}`}
          onClick={() => setSubTab('access')}
        >
          {isZnZz(isZh, '\u63a5\u5165', 'Access')}
        </button>
      </div>

      {/* ═══ 使用 Tab ═══ */}
      {subTab === 'usage' && (
        <>
          {/* 使用模型 */}
          <div className="settings-card">
            <div className="settings-card__title">{isZnZz(isZh, '\u4f7f\u7528\u6a21\u578b', 'Usage Model')}</div>

            <Row label={isZnZz(isZh, '\u9ed8\u8ba4\u6a21\u578b', 'Default Model')} hint={currentProvider}>
              <ModelSelector
                value={props.model}
                options={DS_MODELS}
                isZh={isZh}
                onChange={(v) => props.onModelChange(v)}
              />
            </Row>

            <Row label={isZnZz(isZh, '\u72ec\u7acb\u89c4\u5212\u6a21\u578b', 'Planning Model')} hint={isZnZz(isZh, '\u72ec\u7acb\u89c4\u5212\u9636\u6bb5\u4f7f\u7528\u7684\u6a21\u578b', 'Model used in planning phase')}>
              <ModelSelector
                value={planningModel}
                options={[{ id: 'same', name: '', provider: '', providerShort: _ => isZnZz(isZh, '\u4f7f\u7528\u5f53\u524d\u6a21\u578b\uff08\u5355\u6a21\u578b\uff09', 'Use current (single)'), isSame: true }, ...DS_MODELS]}
                isZh={isZh}
                onChange={(v) => ps('planningModel', setPlanningModel, v)}
              />
            </Row>

            <Row label={isZnZz(isZh, '\u5b50\u4ee3\u7406\u6a21\u578b', 'Sub-agent Model')}>
              <ModelSelector
                value={subAgentModel}
                options={[{ id: 'same', name: '', provider: '', providerShort: _ => isZnZz(isZh, '\u540c\u9ed8\u8ba4', 'Same as default'), isSame: true }, ...DS_MODELS]}
                isZh={isZh}
                onChange={(v) => ps('subAgentModel', setSubAgentModel, v)}
              />
            </Row>

            <Row label={isZnZz(isZh, '\u5b50\u4ee3\u7406 effort', 'Sub-agent Effort')} hint={isZnZz(isZh, '\u4f5c\u7528\u4e8e task \u548c runAs=subagent skills\uff0c\u5de5\u5177\u8c03\u7528...', 'For task and runAs=subagent skills, tool calls...')}>
              <select className="settings-select settings-select--wide" value={subAgentEffort}
                onChange={(e) => ps('subAgentEffort', setSubAgentEffort, e.target.value)}>
                <option value="auto">auto ({isZnZz(isZh, '\u6a21\u578b\u670d\u52a1\u9ed8\u8ba4', 'model default')})</option>
                <option value="low">low</option>
                <option value="medium">medium</option>
                <option value="high">high</option>
              </select>
            </Row>
          </div>

          {/* Agent 运行参数 */}
          <div className="settings-card">
            <div className="settings-card__title">{isZnZz(isZh, 'Agent \u8fd0\u884c', 'Agent Runtime')}</div>
            <div className="settings-card__hint">{isZnZz(isZh, '\u4f5c\u4e3a\u5168\u5c40\u9ed8\u8ba4\u503c\uff0c\u9879\u76ee\u91cc\u7684\u914d\u7f6e\u53ef\u8986\u76d6\u30020 \u8868\u793a\u4e0d9650\u3002', 'Global defaults; project-level config can override. 0 = unlimited.')}</div>

            <Row label={isZnZz(isZh, '\u6267\u884c\u8f6e\u6570\u4e0a9\u9650', 'Max Exec Rounds')} hint={isZnZz(isZh, '\u9650\u5236\u6bcf\u6b21\u56de\u590d\u6700\u591a\u8c03\u7528\u591a\u5c11\u8f6e\u5de5\u5177', 'Limit max tool calls per response')}>
              <OptionPills options={[
                { value: '10', label: '10' }, { value: '25', label: '25' },
                { value: '50', label: '50' }, { value: '0', label: isZnZz(isZh, '\u4e0d9650', '') },
              ]} selected={maxExecRounds} onChange={(v) => ps('maxExecRounds', setMaxExecRounds, v)} />
            </Row>

            <Row label={isZnZz(isZh, '\u89c4\u5212\u8f6e\u6570\u4e0a9\u9650', 'Max Plan Rounds')} hint={isZnZz(isZh, '\u9009\u62e9\u72ec\u7acb\u89c4\u5212\u6a21\u578b\u540e\uff0c\u6b64\u4e0a9\u9650\u624d\u4f1a\u751f\u6548\u3002', 'Only effective when using separate planning model.')}>
              <OptionPills options={[
                { value: '6', label: '6' }, { value: '12', label: '12' },
                { value: '25', label: '25' }, { value: '0', label: isZnZz(isZh, '\u4e0d9650', '') },
              ]} selected={maxPlanRounds} onChange={(v) => ps('maxPlanRounds', setMaxPlanRounds, v)} />
            </Row>

            <Row label={isZnZz(isZh, '\u51b7\u5428\u52a8\u7cbe\u7b80', 'Cold Start Trim')} hint={isZnZz(isZh, '\u91cd\u5f00\u8fc7\u671f\u7684\u4f1a\u8bdd\u65f6\u7cbe\u7b80\u65e7\u5de5\u5177\u7ed3\u679c\uff0c\u964d\u4f4e\u7eed\u804a\u6210\u672c', 'Trim old tool results on session reopen to reduce cost')}>
              <OptionPills options={[
                { value: 'on', label: isZnZz(isZh, '\u5f00', 'On') },
                { value: 'off', label: isZnZz(isZh, '\u5173', 'Off') },
              ]} selected={coldStartTrim} onChange={(v) => ps('coldStartTrim', setColdStartTrim, v)} />
            </Row>

            <Row label={isZnZz(isZh, '\u601d\u8003\u8bed\u8a00', 'Thinking Lang')} hint={isZnZz(isZh, '\u53ea\u5f71\u54cd\u53ef\u89c1\u601d\u8003\u8fc7\u7a0b\u3002\u81ea\u52a8\u8ddf9\u5bf9\u8bdd\u8bed\u8a00\u3002', 'Only affects visible thinking. Auto-follows conversation lang.')}>
              <OptionPills options={[
                { value: 'auto', label: isZnZz(isZh, '\u81ea\u52a8', 'Auto') },
                { value: 'zh', label: '\u4e2d\u6587' },
                { value: 'en', label: 'EN' },
              ]} selected={thinkingLang} onChange={(v) => ps('thinkingLang', setThinkingLang, v)} />
            </Row>
          </div>
        </>
      )}

      {/* ═══ 接入 Tab ═══ */}
      {subTab === 'access' && (
        <>
          <div className="model-access-header">
            <span>{isZnZz(isZh, '\u4f9b\u5e94\u5546\u63a5\u5165', 'Provider Access')}</span>
            <button className="settings-btn settings-btn--outline" onClick={() => setShowAddModal(true)}>
              + {isZnZz(isZh, '\u6dfb\u52a0\u6a21\u578b\u670d\u52a1', 'Add Provider')}
            </button>
          </div>
          <div className="model-access-desc">
            {isZnZz(isZh,
              '\u6dfb\u52a0\u5b98\u65b9\u6216\u81ea\u5b9a\u4e49\u4f9b\u5e94\u5546\u540e\uff0c\u624d\u4f1a\u51fa\u73b0\u5728\u8fd9\u91cc\u3002\u4f1a\u8bdd\u6a21\u578b\u5217\u8868\u53ea\u663e\u793a\u5df2\u4fdd\u5b58\u7684\u542f\u7528\u6a21\u578b\u3002',
              'After adding official or custom providers, they appear here. Session model list shows only enabled models.'
            )}
          </div>

          {providers.length === 0 ? (
            <div className="settings-empty-state">
              <div style={{ fontSize: 32, marginBottom: 8 }}>\U0001f50c</div>
              <div>{isZnZz(isZh, '\u6682\u65e0\u5df2\u63a5\u5165\u7684\u6a21\u578b\u4f9b\u5e94\u5546', 'No model providers connected yet')}</div>
            </div>
          ) : (
            <div className="settings-providers-list">
              {providers.map(provider => (
                <ProviderCard
                  key={provider.id}
                  provider={provider}
                  isZh={isZh}
                  onToggleModel={(mid) => toggleModel(provider.id, mid)}
                  onRemove={() => removeProvider(provider.id)}
                />
              ))}
            </div>
          )}

          {/* ═══ 添加供应商弹窗 ═══ */}
          {showAddModal && (
            <div className="settings-add-modal" style={{
              position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)',
              display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9999,
            }}>
              <div style={{
                background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 12,
                padding: 24, width: 480, maxHeight: '80vh', overflow: 'auto',
              }}>
                <h3 style={{ margin: '0 0 16px', fontSize: 16 }}>{isZh ? '添加模型服务' : 'Add Model Provider'}</h3>

                <div style={{ marginBottom: 12 }}>
                  <label style={{ display: 'block', fontSize: 13, color: 'var(--fg-faint)', marginBottom: 4 }}>
                    {isZh ? '选择预设' : 'Preset'}
                  </label>
                  <select className="settings-select settings-select--wide" value={addPreset}
                    onChange={e => setAddPreset(e.target.value)}>
                    {Object.entries(PRESET_MODELS).map(([k, v]) => (
                      <option key={k} value={k}>{v.name}</option>
                    ))}
                  </select>
                </div>

                {addPreset === 'custom' && (
                  <>
                    <div style={{ marginBottom: 12 }}>
                      <label style={{ display: 'block', fontSize: 13, color: 'var(--fg-faint)', marginBottom: 4 }}>
                        {isZh ? '服务名称' : 'Name'}
                      </label>
                      <input className="settings-input" value={addCustom.name}
                        onChange={e => setAddCustom(p => ({ ...p, name: e.target.value }))}
                        placeholder="My LLM" />
                    </div>
                    <div style={{ marginBottom: 12 }}>
                      <label style={{ display: 'block', fontSize: 13, color: 'var(--fg-faint)', marginBottom: 4 }}>
                        {isZh ? 'API Base URL' : 'API Base URL'}
                      </label>
                      <input className="settings-input" value={addCustom.baseUrl}
                        onChange={e => setAddCustom(p => ({ ...p, baseUrl: e.target.value }))}
                        placeholder="https://api.example.com/v1" />
                    </div>
                    <div style={{ marginBottom: 12 }}>
                      <label style={{ display: 'block', fontSize: 13, color: 'var(--fg-faint)', marginBottom: 4 }}>
                        {isZh ? '环境变量名（存放 API Key）' : 'Env var for API Key'}
                      </label>
                      <input className="settings-input" value={addCustom.apiKeyEnv}
                        onChange={e => setAddCustom(p => ({ ...p, apiKeyEnv: e.target.value }))}
                        placeholder="MY_API_KEY" />
                    </div>
                    <div style={{ marginBottom: 12 }}>
                      <label style={{ display: 'block', fontSize: 13, color: 'var(--fg-faint)', marginBottom: 4 }}>
                        {isZh ? '模型列表（每行一个：model-id,显示名称）' : 'Models (one per line: model-id,display-name)'}
                      </label>
                      <textarea className="settings-input settings-textarea" rows={4}
                        value={addCustom.modelsText}
                        onChange={e => setAddCustom(p => ({ ...p, modelsText: e.target.value }))}
                        placeholder={"gpt-4o,GPT-4o\ngpt-4o-mini,GPT-4o Mini"} style={{ fontFamily: 'monospace', fontSize: 12 }} />
                    </div>
                  </>
                )}

                {addPreset !== 'custom' && PRESET_MODELS[addPreset] && (
                  <div style={{ marginBottom: 12, padding: 12, background: 'var(--bg-soft)', borderRadius: 8, fontSize: 13 }}>
                    <div style={{ color: 'var(--fg-faint)', marginBottom: 4 }}>{isZh ? '将添加以下模型：' : 'Models to add:'}</div>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                      {PRESET_MODELS[addPreset].models.map(m => (
                        <span key={m.id} style={{ padding: '2px 8px', background: 'var(--bg)', borderRadius: 4, fontSize: 12 }}>{m.name}</span>
                      ))}
                    </div>
                    <div style={{ marginTop: 8, color: 'var(--fg-faint)', fontSize: 12 }}>
                      {isZh ? '添加后请在系统环境变量中配置' : 'After adding, set the env variable:'} <code>{PRESET_MODELS[addPreset].apiKeyEnv}</code>
                    </div>
                  </div>
                )}

                <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 16 }}>
                  <button className="settings-btn settings-btn--ghost" onClick={() => setShowAddModal(false)}>
                    {isZh ? '取消' : 'Cancel'}
                  </button>
                  <button className="settings-btn settings-btn--primary" onClick={handleAddProvider}>
                    {isZh ? '添加' : 'Add'}
                  </button>
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}

// 模型下拉选择器（带搜索和分组）
function ModelSelector({ value, options, isZh, onChange }: {
  value: string
  options: { id: string; name: string; provider: string; providerShort: (b: boolean) => string; isSame?: boolean }[]
  isZh: boolean
  onChange: (v: string) => void
}) {
  return (
    <select className="settings-select settings-select--wide" value={value}
      onChange={(e) => onChange(e.target.value)}>
      {options.map(m => (
        <option key={m.id} value={m.id}>
          {m.isSame ? m.providerShort(isZh) : `${m.name}${m.provider ? `  ·  ${m.providerShort(isZh)}` : ''}`}
        </option>
      ))}
    </select>
  )
}

// ════════════════════════════════════════
// Provider Card — 供应商接入卡片
// ════════════════════════════════════════

function ProviderCard({ provider, isZh, onToggleModel, onRemove }: {
  provider: Provider
  isZh: boolean
  onToggleModel: (modelId: string) => void
  onRemove: () => void
}) {
  const enabledCount = provider.models.filter(m => m.enabled).length

  return (
    <div className="settings-provider-card">
      {/* Header row */}
      <div className="settings-provider-card__header">
        <div className="settings-provider-card__name-row">
          <span className="settings-provider-card__name">{provider.name}</span>
          <span className={`settings-provider-badge settings-provider-badge--${provider.type}`}>
            {isZnZz(isZh,
              provider.type === 'official' ? '官方' : '自定义',
              provider.type === 'official' ? 'Official' : 'Custom',
            )}
          </span>
          {provider.source === 'builtin' && (
            <span className="settings-provider-badge settings-provider-badge--builtin">内置</span>
          )}
          {provider.keySet ? (
            <span className="settings-provider-badge settings-provider-badge--ok">
              {isZnZz(isZh, '已设密钥', 'Key Set')}
            </span>
          ) : (
            <span className="settings-provider-badge" style={{ background: 'var(--danger-bg, #fee)', color: 'var(--danger, #c33)' }}>
              {isZnZz(isZh, '未设密钥', 'No Key')}
            </span>
          )}
        </div>
        <div className="settings-provider-card__actions">
          {provider.source !== 'builtin' && (
            <>
              <button className="settings-btn settings-btn--ghost" style={{ fontSize: 12 }}>
                {isZnZz(isZh, '配置', 'Config')}
              </button>
              <button className="settings-btn settings-btn--ghost" style={{ fontSize: 12 }}>
                {isZnZz(isZh, '刷新模型', 'Refresh')}
              </button>
            </>
          )}
          {provider.source !== 'builtin' && (
            <button className="settings-btn settings-btn--ghost settings-btn--danger" style={{ fontSize: 12 }} onClick={onRemove}>
              {isZnZz(isZh, '移除', 'Remove')}
            </button>
          )}
        </div>
      </div>

      {/* Description */}
      <div className="settings-provider-card__desc">{provider.description}</div>

      {/* API info */}
      <div className="settings-provider-card__meta">
        <span>{provider.apiType}</span>
        <span>·</span>
        <span>{provider.baseUrl}</span>
        <span>·</span>
        <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{provider.apiKeyEnv}</span>
      </div>

      {/* Model tags */}
      <div className="settings-provider-card__models">
        <span className="settings-provider-card__models-label">
          {isZnZz(isZh, '已启用模型', 'Enabled Models')} ({enabledCount}/{provider.models.length})
        </span>
        <div className="settings-provider-card__model-tags">
          {provider.models.map(m => (
            <button
              key={m.id}
              className={`settings-model-tag ${m.enabled ? 'enabled' : ''}`}
              onClick={() => onToggleModel(m.id)}
            >
              [{m.id}]
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// Bots Settings
// ════════════════════════════════════════

const BOT_PLATFORMS: { id: BotPlatform; labelZn: string; labelEn: string; icon: string; descZn: string; descEn: string }[] = [
  { id: 'qq', labelZn: 'QQ', labelEn: 'QQ', icon: '🐧', descZn: 'QQ 机器人平台', descEn: 'QQ Bot Platform' },
  { id: 'feishu', labelZn: '飞书', labelEn: 'Feishu', icon: '🚀', descZn: '飞书/Lark 开放平台', descEn: 'Feishu/Lark Open Platform' },
  { id: 'lark', labelZn: 'Lark', labelEn: 'Lark', icon: '🔷', descZn: 'Lark 国际版', descEn: 'Lark International' },
  { id: 'wechat', labelZn: '微信', labelEn: 'WeChat', icon: '💬', descZn: '微信机器人/企业微信', descEn: 'WeChat / WeCom Bot' },
  { id: 'telegram', labelZn: 'Telegram', labelEn: 'Telegram', icon: '✈️', descZn: 'Telegram Bot API', descEn: 'Telegram Bot API' },
  { id: 'discord', labelZn: 'Discord', labelEn: 'Discord', icon: '🎮', descZn: 'Discord Bot', descEn: 'Discord Bot' },
  { id: 'slack', labelZn: 'Slack', labelEn: 'Slack', icon: '💼', descZn: 'Slack Bot', descEn: 'Slack Bot' },
  { id: 'webhook', labelZn: 'Webhook', labelEn: 'Webhook', icon: '🪝', descZn: '通用 Webhook 接入', descEn: 'Generic Webhook Endpoint' },
  { id: 'github', labelZn: 'GitHub', labelEn: 'GitHub', icon: '🐙', descZn: 'GitHub App / Bot', descEn: 'GitHub App / Bot' },
  { id: 'gitlab', labelZn: 'GitLab', labelEn: 'GitLab', icon: '🦊', descZn: 'GitLab Bot / Webhook', descEn: 'GitLab Bot / Webhook' },
]

function BotsSettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null
  const [selectedBot, setSelectedBot] = useState<BotPlatform | null>(null)
  const [botConfigs, setBotConfigs] = useState<Record<BotPlatform, BotConfig>>(() => {
    const saved = loadSetting<Record<string, BotConfig>>('botConfigs', {})
    const initial: Record<string, BotConfig> = {}
    BOT_PLATFORMS.forEach(p => {
      initial[p.id] = saved[p.id] || { platform: p.id, name: p.labelZn, enabled: false, connected: false, config: {} }
    })
    return initial as Record<BotPlatform, BotConfig>
  })

  useEffect(() => { saveSetting('botConfigs', botConfigs) }, [botConfigs])

  const imPlatforms: BotPlatform[] = ['qq', 'feishu', 'lark', 'wechat']
  const otherPlatforms: BotPlatform[] = ['telegram', 'discord', 'slack', 'webhook', 'github', 'gitlab']
  const currentBot = selectedBot ? botConfigs[selectedBot] : null
  const platformInfo = selectedBot ? BOT_PLATFORMS.find(p => p.id === selectedBot) : null
  const connectedCount = Object.values(botConfigs).filter(b => b.connected).length

  const handleToggleBot = (platform: BotPlatform) => {
    setBotConfigs(prev => ({ ...prev, [platform]: { ...prev[platform], enabled: !prev[platform].enabled } }))
  }

  const handleSaveBot = async () => {
    if (!selectedBot) return
    const cfg = botConfigs[selectedBot]
    const name = selectedBot

    // Update local state
    setBotConfigs(prev => ({ ...prev, [selectedBot]: { ...prev[selectedBot], connected: true } }))

    // Connect backend
    if (backend) {
      try {
        const token = cfg.config.botToken || ''
        const appId = cfg.config.appId || ''
        const appSecret = cfg.config.appSecret || ''
        const webhook = cfg.config.url || ''

        await backend.BotConnect(
          name,
          selectedBot,
          token,
          appId,
          appSecret,
          webhook
        )
      } catch (e) {
        console.error('Bot connect failed:', e)
        // Revert state on failure
        setBotConfigs(prev => ({ ...prev, [selectedBot]: { ...prev[selectedBot], connected: false } }))
      }
    }
  }

  const handleConfigChange = (key: string, value: string) => {
    if (!selectedBot) return
    setBotConfigs(prev => ({
      ...prev,
      [selectedBot]: { ...prev[selectedBot], config: { ...prev[selectedBot].config, [key]: value } }
    }))
  }

  const getConfigFields = (platform: BotPlatform): { key: string; labelZn: string; labelEn: string; type?: 'text' | 'password'; placeholder?: string }[] => {
    switch (platform) {
      case 'qq': return [{ key: 'appId', labelZn: 'App ID', labelEn: 'App ID' }, { key: 'appSecret', labelZn: 'App Secret', labelEn: 'App Secret', type: 'password' }]
      case 'feishu': case 'lark': return [{ key: 'appId', labelZn: 'App ID', labelEn: 'App ID' }, { key: 'appSecret', labelZn: 'App Secret', labelEn: 'App Secret', type: 'password' }]
      case 'wechat': return [{ key: 'appId', labelZn: 'AppID', labelEn: 'AppID' }, { key: 'appSecret', labelZn: 'App Secret', labelEn: 'App Secret', type: 'password' }, { key: 'token', labelZn: 'Token', labelEn: 'Token' }, { key: 'encodingAESKey', labelZn: 'EncodingAESKey', labelEn: 'AES Key', type: 'password' }]
      case 'telegram': return [{ key: 'botToken', labelZn: 'Bot Token', labelEn: 'Bot Token', type: 'password', placeholder: '@BotFather' }]
      case 'discord': return [{ key: 'botToken', labelZn: 'Bot Token', labelEn: 'Bot Token', type: 'password' }, { key: 'guildId', labelZn: 'Guild ID', labelEn: 'Guild ID' }]
      case 'slack': return [{ key: 'botToken', labelZn: 'Bot Token', labelEn: 'Bot Token', type: 'password', placeholder: 'xoxb-' }, { key: 'signingSecret', labelZn: 'Signing Secret', labelEn: 'Signing Secret', type: 'password' }]
      case 'webhook': return [{ key: 'url', labelZn: 'URL', labelEn: 'URL' }, { key: 'secret', labelZn: 'Secret', labelEn: 'Secret', type: 'password' }]
      case 'github': return [{ key: 'appId', labelZn: 'App ID', labelEn: 'App ID' }, { key: 'privateKey', labelZn: 'Private Key', labelEn: 'Private Key', type: 'password' }, { key: 'webhookSecret', labelZn: 'Webhook Secret', labelEn: 'Webhook Secret', type: 'password' }]
      case 'gitlab': return [{ key: 'url', labelZn: 'URL', labelEn: 'URL' }, { key: 'token', labelZn: 'PAT', labelEn: 'PAT', type: 'password' }, { key: 'secret', labelZn: 'Secret', labelEn: 'Secret', type: 'password' }]
      default: return []
    }
  }

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZh ? '机器人' : 'Bots'}</h3>
      <p className="settings-page__desc">{isZh ? '配置 IM Bot 渠道，所有配置本地持久化。' : 'Configure IM bot channels. All configs persisted locally.'}</p>

      <div className="settings-bots-status">
        <span>{isZh ? '已连接' : 'Connected'} <strong>{connectedCount}</strong> / {BOT_PLATFORMS.length}</span>
      </div>

      {connectedCount === 0 && (
        <div className="settings-empty-state"><span>{isZh ? '尚未连接任何 Bot' : 'No bots connected yet'}</span></div>
      )}

      {/* IM Bot */}
      <div className="settings-group">
        <div className="settings-group__title">{isZh ? 'IM Bot' : 'IM Bot'}</div>
        <div className="settings-bot-platforms">
          {imPlatforms.map(p => {
            const info = BOT_PLATFORMS.find(x => x.id === p)!
            const cfg = botConfigs[p]
            return (
              <button key={p} className={`settings-bot-chip ${selectedBot === p ? 'active' : ''} ${cfg.connected ? 'connected' : ''}`}
                onClick={() => setSelectedBot(selectedBot === p ? null : p)}>
                <span className="settings-bot-chip__icon">{info.icon}</span>
                <span>{isZh ? info.labelZn : info.labelEn}</span>
                {cfg.connected && <span className="settings-bot-chip__dot" />}
              </button>
            )
          })}
        </div>

        {currentBot && platformInfo && (
          <div className="settings-bot-config">
            <div className="settings-bot-config__header">
              <h4><span>{platformInfo.icon}</span><span>{isZnZz(isZh, platformInfo.labelZn, platformInfo.labelEn)}</span></h4>
              <span className={`settings-bot-config__badge ${currentBot.connected ? 'settings-bot-config__badge--ok' : ''}`}>
                {currentBot.connected ? (isZnZz(isZh, '已连接', 'Connected')) : (isZnZz(isZh, '未配置', 'Not configured'))}
              </span>
            </div>
            <div className="settings-bot-form">
              {getConfigFields(selectedBot!).map(f => (
                <div key={f.key} className="settings-bot-field">
                  <label className="settings-bot-field__label">{isZnZz(isZh, f.labelZn, f.labelEn)}</label>
                  <input type={f.type || 'text'} className="settings-input"
                    placeholder={f.placeholder || ''}
                    value={currentBot.config[f.key] || ''}
                    onChange={(e) => handleConfigChange(f.key, e.target.value)}
                  />
                </div>
              ))}
              <button className="settings-btn settings-btn--primary" onClick={handleSaveBot}>
                {isZnZz(isZh, '保存并启用', 'Save & Enable')}
              </button>
              <div className="settings-bot-warning"><span>⚠</span><span>{isZnZz(isZh, '启用前请在对应平台添加白名单', 'Add whitelist before enabling')}</span></div>
            </div>
          </div>
        )}
      </div>

      {/* 其他渠道 */}
      <div className="settings-group">
        <div className="settings-group__title">{isZh ? '其他渠道' : 'Other Channels'}</div>
        <div className="settings-bot-list">
          {otherPlatforms.map(p => {
            const info = BOT_PLATFORMS.find(x => x.id === p)!
            const cfg = botConfigs[p]
            return (
              <div key={p} className={`settings-bot-item ${selectedBot === p ? 'active' : ''}`} onClick={() => setSelectedBot(selectedBot === p ? null : p)}>
                <div className="settings-bot-item__left">
                  <span className="settings-bot-item__icon">{info.icon}</span>
                  <div className="settings-bot-item__info">
                    <span className="settings-bot-item__name">{isZnZz(isZh, info.labelZn, info.labelEn)}</span>
                    <span className="settings-bot-item__desc">{isZnZz(isZh, info.descZn, info.descEn)}</span>
                  </div>
                </div>
                <ToggleSwitch checked={cfg.enabled} onChange={() => handleToggleBot(p)} onClick={(e) => e.stopPropagation()} />
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// MCP Settings — MCP 服务器管理
// ════════════════════════════════════════

interface McpServer {
  id: string
  name: string
  type: 'stdio' | 'http'
  command?: string       // stdio: 命令路径
  args?: string[]        // stdio: 参数
  env?: Record<string, string>  // stdio: 环境变量
  url?: string           // http: 端点 URL
  headers?: Record<string, string> // http: 自定义头
  enabled: boolean
  status: 'connected' | 'disconnected' | 'error' | 'unknown'
  toolCount: number
}

function McpSettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null

  const [servers, setServers] = useState<McpServer[]>(() => {
    try {
      const saved = localStorage.getItem('zhulong-mcp-servers')
      return saved ? JSON.parse(saved) : []
    } catch { return [] }
  })

  const [showAdd, setShowAdd] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)

  useEffect(() => { localStorage.setItem('zhulong-mcp-servers', JSON.stringify(servers)) }, [servers])

  const addServer = (server: Omit<McpServer, 'id' | 'status' | 'toolCount'>) => {
    setServers(prev => [...prev, { ...server, id: crypto.randomUUID(), status: 'unknown' as const, toolCount: 0 }])
    setShowAdd(false)
  }

  const removeServer = async (id: string) => {
    if (!isZh && !window.confirm('Remove this MCP server?')) return
    if (isZh && !window.confirm('\u786e\u5b9a\u8981\u79fb\u9664\u6b64 MCP \u670d\u52a1\u5668\uff1f')) return
    const server = servers.find(s => s.id === id)
    if (server && backend) {
      try { await backend.MCPDisconnectServer(server.name) } catch {}
    }
    setServers(prev => prev.filter(s => s.id !== id))
  }

  const toggleServer = async (id: string) => {
    const server = servers.find(s => s.id === id)
    if (!server) return

    if (!server.enabled) {
      // Enable and connect
      setServers(prev => prev.map(s => s.id !== id ? s : { ...s, enabled: true, status: 'connected' as const }))
      if (backend) {
        try {
          await backend.MCPConnectServer(
            server.name,
            server.type,
            server.command || '',
            server.args || [],
            server.url || ''
          )
          // Get tool count
          const tools = await backend.MCPListTools()
          const serverTools = tools?.[server.name]
          setServers(prev => prev.map(s => s.id !== id ? s : {
            ...s,
            status: 'connected' as const,
            toolCount: serverTools?.length || 0
          }))
        } catch (e) {
          setServers(prev => prev.map(s => s.id !== id ? s : { ...s, status: 'error' as const }))
        }
      }
    } else {
      // Disable and disconnect
      if (backend) {
        try { await backend.MCPDisconnectServer(server.name) } catch {}
      }
      setServers(prev => prev.map(s => s.id !== id ? s : { ...s, enabled: false, status: 'disconnected' as const, toolCount: 0 }))
    }
  }

  const connectedCount = servers.filter(s => s.status === 'connected').length

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, 'MCP 与工具', 'MCP & Tools')}</h3>
      <p className="settings-page__desc">{isZnZz(isZh, '管理 MCP 服务器和外部工具，为 Agent 扩展能力。', 'Manage MCP servers and external tools to extend agent capabilities.')}</p>

      {/* Header */}
      <div className="mcp-header">
        <div className="mcp-status">
          <span className="mcp-status__label">{isZnZz(isZh, '已连接', 'Connected')}</span>
          <span className="mcp-status__count">{connectedCount}</span>
          <span className="mcp-status__sep">/</span>
          <span className="mcp-status__total">{servers.length}</span>
        </div>
        <button className="settings-btn settings-btn--primary" onClick={() => setShowAdd(true)}>
          + {isZnZz(isZh, '添加服务器', 'Add Server')}
        </button>
      </div>

      {/* Server List */}
      {servers.length === 0 ? (
        <div className="settings-empty-state mcp-empty">
          <div className="mcp-empty__icon">🔌</div>
          <div className="mcp-empty__title">{isZnZz(isZh, '暂无 MCP 服务器', 'No MCP Servers')}</div>
          <div className="mcp-empty__desc">
            {isZnZz(isZh,
              '点击上方「添加服务器」配置 MCP 服务端，为烛龙 Agent 接入外部工具和数据源。',
              'Click "Add Server" above to configure an MCP endpoint and extend your agent with external tools.'
            )}
          </div>
          <div className="mcp-empty__hints">
            <span className="mcp-hint">{isZnZz(isZh, '支持 stdio（本地进程）和 http（远程服务）两种传输方式。', 'Supports both stdio (local process) and http (remote service) transports.')}</span>
          </div>
        </div>
      ) : (
        <div className="mcp-server-list">
          {servers.map(server => (
            <div key={server.id} className={`mcp-server-card ${!server.enabled ? 'mcp-server-card--disabled' : ''}`}>
              <div className="mcp-server-card__header">
                <div className="mcp-server-card__info">
                  <span className={`mcp-server-card__status-dot mcp-server-card__status-dot--${server.status}`} />
                  <span className="mcp-server-card__name">{server.name}</span>
                  <span className={`mcp-server-badge mcp-server-badge--${server.type}`}>
                    {server.type === 'stdio' ? 'stdio' : 'http'}
                  </span>
                </div>
                <div className="mcp-server-card__actions">
                  <ToggleSwitch checked={server.enabled} onChange={() => toggleServer(server.id)} />
                  <button className="settings-btn settings-btn--ghost settings-btn--danger-sm"
                    onClick={() => removeServer(server.id)}>
                    ✕
                  </button>
                </div>
              </div>
              <div className="mcp-server-card__body">
                {server.type === 'stdio' ? (
                  <div className="mcp-server-card__detail">
                    <code>{server.command}{server.args?.length ? ` ${server.args.join(' ')}` : ''}</code>
                  </div>
                ) : (
                  <div className="mcp-server-card__detail">
                    <code>{server.url}</code>
                  </div>
                )}
                <div className="mcp-server-card__meta">
                  <span className={`mcp-server-card__status-text mcp-server-card__status-text--${server.status}`}>
                    {server.status === 'connected' && isZnZz(isZh, '已连接', 'Connected')}
                    {server.status === 'disconnected' && isZnZz(isZh, '未连接', 'Disconnected')}
                    {server.status === 'error' && isZnZz(isZh, '错误', 'Error')}
                    {server.status === 'unknown' && isZnZz(isZh, '未检测', 'Unknown')}
                  </span>
                  {server.toolCount > 0 && (
                    <span className="mcp-server-card__tools">
                      {server.toolCount} {isZnZz(isZh, '个工具', 'tools')}
                    </span>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 添加/编辑弹窗 */}
      {(showAdd || editingId) && (
        <McpServerForm
          isZh={isZh}
          server={editingId ? servers.find(s => s.id === editingId) ?? null : null}
          onSave={(srv) => {
            if (editingId) {
              setServers(prev => prev.map(s => s.id === editingId ? { ...s, ...srv } : s))
              setEditingId(null)
            } else {
              addServer(srv)
            }
          }}
          onClose={() => { setShowAdd(false); setEditingId(null) }}
        />
      )}

      {/* ═══ Dashboard 控制 ═══ */}
      <DashboardControl language={language} />
    </div>
  )
}

// ─── Dashboard 控制组件 ───
function DashboardControl({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null
  const [port, setPort] = useState<number>(() => loadJSON('zhulong-dashboard-port', 7788))
  const [running, setRunning] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    saveJSON('zhulong-dashboard-port', port)
    if (backend) try { backend.SetDashboardPort(port) } catch {}
  }, [port])

  const handleStart = async () => {
    if (!backend) return
    setLoading(true)
    try {
      await backend.SetDashboardPort(port)
      const err = await backend.StartDashboard()
      if (!err) setRunning(true)
    } catch {}
    setLoading(false)
  }

  const handleStop = async () => {
    if (!backend) return
    setLoading(true)
    try {
      const err = await backend.StopDashboard()
      if (!err) setRunning(false)
    } catch {}
    setLoading(false)
  }

  return (
    <div className="settings-section" style={{ marginTop: 20 }}>
      <h4 className="settings-section__title">{isZnZz(isZh, 'Dashboard 仪表盘', 'Dashboard')}</h4>
      <p className="settings-section__desc">
        {isZnZz(isZh, '启动本地 Web 仪表盘，通过浏览器监控 Agent 运行状态。', 'Start a local web dashboard to monitor agent status via browser.')}
      </p>
      <div className="settings-row">
        <span className="settings-row__label">{isZnZz(isZh, '端口', 'Port')}</span>
        <input className="settings-input settings-input--xs" type="number" value={port}
          onChange={e => setPort(parseInt(e.target.value) || 7788)} />
      </div>
      <div className="settings-row">
        <span className="settings-row__label">{isZnZz(isZh, '状态', 'Status')}</span>
        <span style={{ color: running ? '#22c55e' : '#888', fontSize: 13 }}>
          {running ? (isZh ? '运行中' : 'Running') : (isZh ? '已停止' : 'Stopped')}
        </span>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
          {running ? (
            <button className="settings-btn settings-btn--ghost settings-btn--sm" onClick={handleStop} disabled={loading}>
              {isZnZz(isZh, '停止', 'Stop')}
            </button>
          ) : (
            <button className="settings-btn settings-btn--primary settings-btn--sm" onClick={handleStart} disabled={loading}>
              {isZnZz(isZh, '启动', 'Start')}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

// ─── MCP 服务器表单弹窗 ───
function McpServerForm({ isZh, server, onSave, onClose }: {
  isZh: boolean
  server: McpServer | null
  onSave: (s: Omit<McpServer, 'id' | 'status' | 'toolCount'>) => void
  onClose: () => void
}) {
  const [name, setName] = useState(server?.name ?? '')
  const [type, setType] = useState<'stdio' | 'http'>(server?.type ?? 'stdio')
  const [command, setCommand] = useState(server?.command ?? '')
  const [args, setArgs] = useState(server?.args?.join('\n') ?? '')
  const [url, setUrl] = useState(server?.url ?? '')
  const [enabled, setEnabled] = useState(server?.enabled ?? true)

  const handleSave = () => {
    if (!name.trim()) return
    if (type === 'stdio') {
      onSave({ name: name.trim(), type: 'stdio', command: command.trim(), args: args.split('\n').map(a => a.trim()).filter(Boolean), enabled })
    } else {
      if (!url.trim()) return
      onSave({ name: name.trim(), type: 'http', url: url.trim(), enabled })
    }
  }

  return (
    <div className="mcp-form-overlay" onClick={onClose}>
      <div className="mcp-form-dialog" onClick={e => e.stopPropagation()}>
        <div className="mcp-form-dialog__header">
          <h4>{server ? isZnZz(isZh, '编辑服务器', 'Edit Server') : isZnZz(isZh, '添加 MCP 服务器', 'Add MCP Server')}</h4>
          <button className="settings-dialog__close" onClick={onClose}>✕</button>
        </div>

        <div className="mcp-form-dialog__body">
          {/* 名称 */}
          <div className="settings-bot-field">
            <label className="settings-bot-field__label">{isZnZz(isZh, '名称', 'Name')}</label>
            <input type="text" className="settings-input" value={name}
              onChange={e => setName(e.target.value)}
              placeholder={isZnZz(isZh, '例如：filesystem、github', 'e.g. filesystem, github')}
            />
          </div>

          {/* 传输方式 */}
          <div className="settings-bot-field">
            <label className="settings-bot-field__label">{isZnZz(isZh, '传输方式', 'Transport')}</label>
            <OptionPills options={[
              { value: 'stdio', label: 'stdio' },
              { value: 'http', label: 'HTTP / SSE' },
            ]} selected={type} onChange={(v) => setType(v as 'stdio' | 'http')} />
          </div>

          {type === 'stdio' ? (
            <>
              <div className="settings-bot-field">
                <label className="settings-bot-field__label">{isZnZz(isZh, '命令', 'Command')}</label>
                <input type="text" className="settings-input" value={command} onChange={e => setCommand(e.target.value)}
                  placeholder="npx"
                />
              </div>
              <div className="settings-bot-field">
                <label className="settings-bot-field__label">{isZnZz(isZh, '参数（每行一个）', 'Arguments (one per line)')}</label>
                <textarea className="settings-input settings-textarea" rows={3}
                  value={args} onChange={e => setArgs(e.target.value)}
                  placeholder="-y&#10;@modelcontextprotocol/server-filesystem&#10;/path/to/allowed/dir"
                />
              </div>
            </>
          ) : (
            <div className="settings-bot-field">
              <label className="settings-bot-field__label">{isZnZz(isZh, 'URL', 'Endpoint URL')}</label>
              <input type="text" className="settings-input" value={url} onChange={e => setUrl(e.target.value)}
                placeholder="https://your-mcp-server.example.com/sse"
              />
            </div>
          )}

          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 16 }}>
            <ToggleSwitch checked={enabled} onChange={() => setEnabled(!enabled)} />
            <span style={{ fontSize: 13, color: 'var(--fg-soft)' }}>{isZnZz(isZh, '启用后自动连接', 'Auto-connect on enable')}</span>
          </div>
        </div>

        <div className="mcp-form-dialog__footer">
          <button className="settings-btn settings-btn--ghost" onClick={onClose}>
            {isZnZz(isZh, '取消', 'Cancel')}
          </button>
          <button className="settings-btn settings-btn--primary" onClick={handleSave}
            disabled={!name.trim() || (type === 'http' && !url.trim()) || (type === 'stdio' && !command.trim())}>
            {server ? isZnZz(isZh, '保存修改', 'Save Changes') : isZnZz(isZh, '添加', 'Add')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// Memory Settings — 记忆管理（参考 Reasonix）
// ════════════════════════════════════════

// ─── 记忆数据类型 ───
interface MemoryFact {
  name: string
  title?: string
  description: string
  type: 'user' | 'feedback' | 'project' | 'reference'
  body: string
  createdAt: string
}

interface MemoryDoc {
  path: string
  scope: 'user' | 'project' | 'local'
  body: string
  updatedAt: string
}

interface MemoryArchive extends MemoryFact {
  archivedAt: string
}

interface MemorySuggestion {
  id: string
  name: string
  title: string
  description: string
  type: string
  body: string
  reason: string
  evidence: string[]
}

function loadJSON<T>(key: string, fallback: T): T {
  try { const v = localStorage.getItem(key); return v ? JSON.parse(v) : fallback } catch { return fallback }
}
function saveJSON(key: string, value: unknown) { try { localStorage.setItem(key, JSON.stringify(value)) } catch {} }

function MemorySettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const [subTab, setSubTab] = useState<'saved' | 'archived' | 'docs' | 'suggestions'>('saved')
  const [showStoragePath, setShowStoragePath] = useState(false)
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null

  // ── 数据状态（从后端加载）──
  const [facts, setFacts] = useState<MemoryFact[]>([])
  const [archives, setArchives] = useState<MemoryArchive[]>([])
  const [docs, setDocs] = useState<MemoryDoc[]>([])
  const [storageDir, setStorageDir] = useState('')
  const [loading, setLoading] = useState(true)
  const [suggestions, setSuggestions] = useState<MemorySuggestion[] | null>(null)
  const [autoSuggest, setAutoSuggest] = useState(() => loadJSON('zhulong-auto-suggest', false))

  // ── 搜索/过滤 ──
  const [query, setQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('all')

  // ── 编辑状态 ──
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editDraft, setEditDraft] = useState('')
  const [addFormOpen, setAddFormOpen] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newBody, setNewBody] = useState('')
  const [newType, setNewType] = useState<'user' | 'feedback' | 'project' | 'reference'>('user')

  // ── 指令文件编辑 ──
  const [docEditPath, setDocEditPath] = useState<string | null>(null)
  const [docDraft, setDocDraft] = useState('')

  useEffect(() => { saveJSON('zhulong-auto-suggest', autoSuggest) }, [autoSuggest])

  // ── 从后端加载记忆数据 ──
  const loadMemory = async () => {
    if (!backend) { setLoading(false); return }
    try {
      const view = await backend.ListMemory()
      if (view) {
        setFacts(view.facts || [])
        setArchives(view.archives || [])
        setDocs(view.docs || [])
        setStorageDir(view.storeDir || '')
      }
    } catch (e) { console.error('Failed to load memory:', e) }
    setLoading(false)
  }

  useEffect(() => { loadMemory() }, [])

  // ── 过滤后的记忆列表 ──
  const filteredFacts = (() => {
    let list = facts
    if (typeFilter !== 'all') list = list.filter(f => f.type === typeFilter)
    if (query.trim()) {
      const q = query.toLowerCase()
      list = list.filter(f =>
        (f.title || '').toLowerCase().includes(q) ||
        f.name.toLowerCase().includes(q) ||
        f.description.toLowerCase().includes(q) ||
        f.body.toLowerCase().includes(q)
      )
    }
    return list
  })()

  const savedCount = facts.length
  const archivedCount = archives.length
  const docsCount = docs.length

  // ── CRUD 操作（调用后端 API）──
  const handleAddMemory = async () => {
    if (!newBody.trim()) return
    if (!backend) return
    try {
      const fact = await backend.Remember(
        `mem-${Date.now()}`,
        newTitle.trim() || '',
        newDesc.trim(),
        newType,
        newBody.trim()
      )
      if (fact) {
        setFacts(prev => [fact, ...prev])
        setNewTitle(''); setNewDesc(''); setNewBody(''); setAddFormOpen(false)
      }
    } catch (e) { console.error('Remember failed:', e) }
  }

  const handleArchive = async (name: string) => {
    if (!isZh && !window.confirm('Archive this memory?')) return
    if (isZh && !window.confirm('\u786e\u5b9a\u8981\u5f52\u6866\u6b64\u8bb0\u5fc6\uff1f')) return
    if (!backend) return
    try {
      await backend.Forget(name)
      // 重新加载以同步状态
      await loadMemory()
    } catch (e) { console.error('Forget failed:', e) }
  }

  const handleDelete = async (name: string) => {
    if (!isZh && !window.confirm('Delete this memory?')) return
    if (isZh && !window.confirm('\u786e\u5b9a\u8981\u5220\u9664\u6b64\u8bb0\u5fc6\uff1f')) return
    if (!backend) return
    try {
      await backend.DeleteMemory(name)
      setFacts(prev => prev.filter(f => f.name !== name))
    } catch (e) { console.error('DeleteMemory failed:', e) }
  }

  const handleRestore = async (name: string) => {
    if (!backend) return
    try {
      await backend.RestoreMemory(name)
      await loadMemory()
    } catch (e) { console.error('RestoreMemory failed:', e) }
  }

  const handleSaveEdit = async () => {
    if (!editingId || !editDraft.trim()) return
    if (!backend) return
    const fact = facts.find(f => f.name === editingId)
    if (!fact) return
    try {
      const updated = await backend.Remember(
        fact.name,
        fact.title || '',
        fact.description,
        fact.type,
        editDraft.trim()
      )
      if (updated) {
        setFacts(prev => prev.map(f => f.name === editingId ? updated : f))
      }
    } catch (e) { console.error('Remember (edit) failed:', e) }
    setEditingId(null); setEditDraft('')
  }

  const handleSaveDoc = async () => {
    if (!docEditPath || docDraft === null) return
    if (!backend) return
    try {
      const doc = await backend.SaveDoc(docEditPath, 'project', docDraft)
      if (doc) {
        setDocs(prev => {
          const idx = prev.findIndex(d => d.path === docEditPath)
          if (idx >= 0) return prev.map((d, i) => i === idx ? doc : d)
          return [...prev, doc]
        })
      }
    } catch (e) { console.error('SaveDoc failed:', e) }
    setDocEditPath(null); setDocDraft('')
  }

  const handleDeleteDoc = async (path: string) => {
    if (!isZh && !window.confirm('Delete this instruction file?')) return
    if (isZh && !window.confirm('\u786e\u5b9a\u8981\u522a\u9664\u6b64\u6307\u4ee4\u6587\u4ef6\uff1f')) return
    if (!backend) return
    try {
      await backend.DeleteDoc(path)
      setDocs(prev => prev.filter(d => d.path !== path))
    } catch (e) { console.error('DeleteDoc failed:', e) }
  }

  // ── 扫描建议 ──
  const handleScanSuggestions = () => {
    const mockSuggestions: MemorySuggestion[] = []
    if (facts.length > 0) {
      const userFacts = facts.filter(f => f.type === 'user').slice(0, 2)
      userFacts.forEach(f => {
        mockSuggestions.push({
          id: `sug-${Date.now()}-${f.name}`,
          name: f.name + '-refined',
          title: f.title || f.name,
          description: isZh ? '从现有记忆提炼' : 'Refined from existing',
          type: f.type,
          body: f.body,
          reason: isZh ? '此条目在多次对话中被引用，建议保留为全局偏好。' : 'Referenced multiple times.',
          evidence: [isZnZz(isZh, '\u8fd1\u671f3\u6b21\u5bf9\u8bdd\u4e2d\u5747\u6d89\u53ca\u6b64\u504f\u597d', 'Referenced in last 3 sessions')]
        })
      })
    }
    setSuggestions(mockSuggestions.length > 0 ? mockSuggestions : [{
      id: `sug-${Date.now()}`,
      name: 'example-preference',
      title: isZh ? '示例：用户偏好中文回复' : 'Example: Prefers Chinese replies',
      description: '',
      type: 'user',
      body: isZh ? '用户偏好使用中文进行交流，所有回复应使用简体中文。' : 'User prefers communication in Simplified Chinese.',
      reason: isZh ? '扫描发现用户持续使用中文交互。' : 'Detected consistent Chinese usage.',
      evidence: []
    }])
  }

  const handleAcceptSuggestion = async (sug: MemorySuggestion) => {
    if (!backend) return
    try {
      const fact = await backend.Remember(
        sug.name,
        sug.title,
        sug.description,
        (['user','feedback','project','reference'] as const).includes(sug.type as any) ? sug.type : 'user',
        sug.body
      )
      if (fact) {
        setFacts(prev => [fact, ...prev])
        setSuggestions(prev => prev?.filter(s => s.id !== sug.id) ?? null)
      }
    } catch (e) { console.error('AcceptSuggestion failed:', e) }
  }

  if (loading) {
    return <div className="settings-page"><p style={{ color: '#999', textAlign: 'center', padding: 40 }}>{isZh ? '加载中...' : 'Loading...'}</p></div>
  }

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, '\u8bb0\u5fc6', 'Memory')}</h3>
      <p className="settings-page__desc">
        {isZnZz(isZh, '\u7ba1\u7406 \u70ed\u9f99 \u957f\u671f\u53c2\u8003\u504f\u597d\uff0c\u9879\u76ee\u7ea6\u5b9a\u548c\u6307\u4ee4\u6587\u4ef6\u3002', 'Manage Zhulong long-term preferences, conventions and instruction files.')}
      </p>

      {/* ═══ 状态栏 ═══ */}
      <div className="memory-stats-bar">
        <span>{savedCount} {isZnZz(isZh, '\u6761\u5df2\u4fdd\u5b58', 'saved')}</span>
        <span>·</span>
        <span>{archivedCount} {isZnZz(isZh, '\u6761\u5f52\u6863', 'archived')}</span>
        <span>·</span>
        <span>{docsCount} {isZnZz(isZh, '\u4e2a\u6307\u4ee4\u6587\u4ef6', 'instruction files')}</span>
        <span>·</span>
        <span>{isZnZz(isZh, '\u5f53\u524d\u5de5\u4f5c\u533a', 'current workspace')}</span>
        <button className="memory-storage-btn" onClick={() => setShowStoragePath(!showStoragePath)}>
          {isZnZz(isZh, showStoragePath ? '\u9690\u85cf\u4f4d\u7f6e' : '\u5b58\u50a8\u4f4d\u7f6e', showStoragePath ? 'Hide Path' : 'Storage Location')}
        </button>
      </div>

      {/* 存储路径展开 */}
      {showStoragePath && (
        <div className="memory-path-display">
          <span className="memory-path-label">{isZnZz(isZh, '\u5b58\u50a8\u76ee\u5f55', 'Store Dir')}</span>
          <code className="memory-path-value">{storageDir}</code>
        </div>
      )}

      {/* ═══ 子 Tab 栏 ═══ */}
      <div className="memory-tabs-row">
        <button className={`memory-tab ${subTab === 'saved' ? 'active' : ''}`} onClick={() => setSubTab('saved')}>
          {isZnZz(isZh, '\u5df2\u4fdd\u5b58\u7684\u8bb0\u5fc6', 'Saved Memories')}
          {savedCount > 0 && <span className="memory-tab-count">{savedCount}</span>}
        </button>
        <button className={`memory-tab ${subTab === 'archived' ? 'active' : ''}`} onClick={() => setSubTab('archived')}>
          {isZnZz(isZh, '\u5f52\u6863\u7684\u8bb0\u5fc6', 'Archived')}
          {archivedCount > 0 && <span className="memory-tab-count">{archivedCount}</span>}
        </button>
        <button className={`memory-tab ${subTab === 'docs' ? 'active' : ''}`} onClick={() => setSubTab('docs')}>
          {isZnZz(isZh, '\u6307\u4ee4\u6587\u4ef6', 'Instruction Files')}
          {docsCount > 0 && <span className="memory-tab-count">{docsCount}</span>}
        </button>
        <button className={`memory-tab memory-tab--suggest ${subTab === 'suggestions' ? 'active' : ''}`} onClick={() => setSubTab('suggestions')}>
          ✨ {isZnZz(isZh, '\u6a21\u9009\u5efa\u8bae', 'Model Suggestions')}
        </button>
      </div>

      {/* ═══ 已保存的记忆 ═══ */}
      {subTab === 'saved' && (
        <>
          <div className="memory-section-header">
            <div>
              <h4 className="memory-section-title">{isZnZz(isZh, '\u5df2\u4fdd\u5b58\u7684\u8bb0\u5fc6', 'Saved Memories')}</h4>
              <p className="memory-section-desc">
                {isZnZz(isZh, '\u8fd9\u4e9b\u662f\u5e38\u7528\u504f\u597d\u3001\u5f15\u7528\u6587\u4ef6\u3001\u51fd\u6570\u6216\u56e2\u961f\u89c4\u5219\u3002\u53ef\u901a\u8fc7\u9a8c\u8bc1\u3002', 'Common preferences, references, function or team rules. Verifiable.')}
              </p>
            </div>
            <button className="settings-btn settings-btn--primary" onClick={() => setAddFormOpen(!addFormOpen)}>
              + {isZnZz(isZh, '\u6dfb\u52a0\u8bb0\u5fc6', 'Add Memory')}
            </button>
          </div>

          {/* 搜索栏 */}
          <div className="memory-toolbar">
            <div className="memory-search">
              <span className="memory-search-icon">🔍</span>
              <input className="memory-search-input"
                value={query}
                onChange={e => setQuery(e.target.value)}
                placeholder={isZnZz(isZh, '\u641c\u7d22\u6807\u9898\u3001slug\u3001\u63cf\u8ff0\u6216\u6b63\u6587...', 'Search title, slug, desc or content...')}
              />
            </div>
            <div className="memory-filter">
              {[
                { value: 'all', label: isZnZz(isZh, '\u5168\u90e8', 'All') },
                { value: 'feedback', label: isZnZz(isZh, '\u53cd\u9988', 'Feedback') },
                { value: 'project', label: isZnZz(isZh, '\u9879\u76ee\u7ea6\u5b9a', 'Project') },
                { value: 'reference', label: isZnZz(isZh, '\u5f15\u7528', 'Reference') },
                { value: 'user', label: isZnZz(isZh, '\u5168\u5c40\u504f\u597d', 'Global') },
              ].map(f => (
                <button key={f.value}
                  className={`memory-filter-item ${typeFilter === f.value ? 'on' : ''}`}
                  onClick={() => setTypeFilter(f.value)}>{f.label}</button>
              ))}
            </div>
          </div>

          {/* 添加表单 */}
          {addFormOpen && (
            <div className="memory-add-form">
              <div className="memory-form-field">
                <label>{isZnZz(isZh, '\u6807\u9898', 'Title')} ({isZnZz(isZh, '可选', 'optional')})</label>
                <input className="settings-input" value={newTitle} onChange={e => setNewTitle(e.target.value)}
                  placeholder={isZnZz(isZh, 'e.g. \u7528\u6237\u504f\u597d\u4e2d\u6587', 'e.g. User prefers Chinese')} />
              </div>
              <div className="memory-form-field">
                <label>{isZnZz(isZh, '\u63cf\u8ff0', 'Description')}</label>
                <input className="settings-input" value={newDesc} onChange={e => setNewDesc(e.target.value)}
                  placeholder={isZnZz(isZh, '\u77ed\u8bed\u63cf\u8ff0', 'Short description')} />
              </div>
              <div className="memory-form-field">
                <label>{isZnZz(isZh, '\u7c7b\u578b', 'Type')}</label>
                <OptionPills options={[
                  { value: 'user', label: isZnZz(isZh, '\u5168\u5c40\u504f\u597d', 'Global') },
                  { value: 'feedback', label: isZnZz(isZh, '\u53cd\u9988', 'Feedback') },
                  { value: 'project', label: isZnZz(isZh, '\u9879\u76ee\u7ea6\u5b9a', 'Project') },
                  { value: 'reference', label: isZnZz(isZh, '\u5f15\u7528', 'Reference') },
                ]} selected={newType} onChange={(v) => setNewType(v as any)} />
              </div>
              <div className="memory-form-field">
                <label>{isZnZz(isZh, '\u5185\u5bb9', 'Content')} *</label>
                <textarea className="settings-input settings-textarea" rows={4} value={newBody}
                  onChange={e => setNewBody(e.target.value)}
                  placeholder={isZnZz(isZh, '\u8bb0\u5fc6\u6b63\u6587...', 'Memory content...')} />
              </div>
              <div className="memory-form-actions">
                <button className="settings-btn settings-btn--ghost" onClick={() => setAddFormOpen(false)}>
                  {isZnZz(isZh, '\u53d6\u6d88', 'Cancel')}
                </button>
                <button className="settings-btn settings-btn--primary" onClick={handleAddMemory} disabled={!newBody.trim()}>
                  {isZnZz(isZh, '\u4fdd\u5b58', 'Save')}
                </button>
              </div>
            </div>
          )}

          {/* 记忆列表 */}
          {filteredFacts.length === 0 ? (
            <div className="memory-empty">
              <span className="memory-empty-icon">🧠</span>
              <span className="memory-empty-title">
                {query ? (isZnZz(isZh, '\u6ca1\u6709\u5339\u914d\u7684\u8bb0\u5fc6', 'No matching memories')) : (isZnZz(isZh, '\u8fd8\u6ca1\u6709\u4fdd\u5b58\u7684\u8bb0\u5fc6', 'No saved memories yet'))}
              </span>
              <span className="memory-empty-desc">
                {query ? '' : (isZnZz(isZh, '\u70b9\u51fb\u201c\u6dfb\u52a0\u8bb0\u5fc6\u201d\u521b\u5efa\u7b2c\u4e00\u6761\u3002', 'Click "Add Memory" to create your first.'))}
              </span>
            </div>
          ) : (
            <div className="memory-list">
              {filteredFacts.map(fact => (
                <div key={fact.name} className={`mem-fact mem-fact--${fact.type}`}>
                  <div className="mem-fact__summary" onClick={() => setExpandedId(expandedId === fact.name ? null : fact.name)}>
                    <span className="mem-fact__expand">{'▸'}</span>
                    <div className="mem-fact__main">
                      <div className="mem-fact__header">
                        <span className="mem-fact__title">{fact.title || fact.name}</span>
                        <span className={`mem-fact__type-badge mem-type--${fact.type}`}>{
                          fact.type === 'user' && isZnZz(isZh, '\u5168\u5c40\u504f\u597d', 'Global')
                          || fact.type === 'feedback' && isZnZz(isZh, '\u53cd\u9988', 'Feedback')
                          || fact.type === 'project' && isZnZz(isZh, '\u9879\u76ee\u7ea6\u5b9a', 'Project')
                          || isZnZz(isZh, '\u5f15\u7528', 'Reference')
                        }</span>
                      </div>
                      <div className="mem-fact__meta">
                        <code className="mem-fact__slug">{fact.name}</code>
                      </div>
                      {fact.description && (
                        <div className="mem-fact__desc">{fact.description}</div>
                      )}
                    </div>
                    <div className="mem-fact__actions">
                      <button className="settings-btn settings-btn--ghost settings-btn--sm"
                        onClick={(e) => { e.stopPropagation(); handleArchive(fact.name) }}>
                        {isZnZz(isZh, '\u5f52\u6863', 'Archive')}
                      </button>
                      <button className="settings-btn settings-btn--ghost settings-btn--danger-sm"
                        onClick={(e) => { e.stopPropagation(); handleDelete(fact.name) }}>✕</button>
                    </div>
                  </div>

                  {/* 展开内容 */}
                  {expandedId === fact.name && (
                    <div className="mem-fact__body-panel">
                      {editingId === fact.name ? (
                        <div className="mem-fact__edit">
                          <textarea className="settings-input settings-textarea" rows={6}
                            value={editDraft} onChange={e => setEditDraft(e.target.value)} />
                          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                            <button className="settings-btn settings-btn--primary" onClick={handleSaveEdit}>
                              {isZnZz(isZh, '\u4fdd\u5b58', 'Save')}
                            </button>
                            <button className="settings-btn settings-btn--ghost"
                              onClick={() => { setEditingId(null); setEditDraft('') }}>
                              {isZnZz(isZh, '\u53d6\u6d88', 'Cancel')}
                            </button>
                          </div>
                        </div>
                      ) : (
                        <>
                          <pre className="mem-fact__body">{fact.body}</pre>
                          <button className="settings-btn settings-btn--ghost settings-btn--sm"
                            onClick={() => { setEditingId(fact.name); setEditDraft(fact.body) }}>
                            ✎ {isZnZz(isZh, '\u7f16\u8f91', 'Edit')}
                          </button>
                        </>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* ═══ 归档的记忆 ═══ */}
      {subTab === 'archived' && (
        <>
          <div className="memory-section-header">
            <div>
              <h4 className="memory-section-title">{isZnZz(isZh, '\u5f52\u6863\u7684\u8bb0\u5fc6', 'Archived Memories')}</h4>
              <p className="memory-section-desc">
                {isZnZz(isZh, '\u4ec5\u7528\u4e8e\u907f\u514d\u6d4b\u91cf\u3002\u5f52\u6863\u8bb0\u5fc6\u4e0d\u4f1a\u4f5c\u4e3a active memory \u88ab\u52a0\u8f7d\u68c0\u7d22\u3002',
                  'Only for avoidance. Archived memories are not loaded as active memory for retrieval.')}
              </p>
            </div>
          </div>

          {archives.length === 0 ? (
            <div className="memory-empty memory-empty--archived">
              <h4 className="memory-empty-title">{isZnZz(isZh, '\u8fd8\u6ca1\u6709\u5f52\u6863\u8bb0\u5fc6', 'No Archived Memories')}</h4>
              <p className="memory-empty-desc">
                {isZnZz(isZh, '\u8bb0\u5fc6\u88ab\u5f52\u6863\u540e\u4f1a\u505c\u6b62\u751f\u6548\uff0c\u4f46\u4ecd\u4f1a\u4fdd\u7559\u5728\u8fd9\u91cd\u7528\u4e8e\u5ba1\u8ba1\u548c\u6062\u590d\u3002',
                  'Archived memories stop being effective but remain here for audit and restore.')}
              </p>
            </div>
          ) : (
            <div className="memory-list">
              {archives.map(arch => (
                <div key={`${arch.name}-${arch.archivedAt}`} className="mem-fact mem-fact--archived">
                  <div className="mem-fact__summary">
                    <span className="mem-fact__expand">▸</span>
                    <div className="mem-fact__main">
                      <div className="mem-fact__header">
                        <span className="mem-fact__title">{arch.title || arch.name}</span>
                        <span className="mem-fact__type-badge mem-type--archived">
                          {isZnZz(isZh, '\u5df2\u5f52\u6863', 'Archived')}
                        </span>
                      </div>
                      <div className="mem-fact__desc">{arch.description}</div>
                      <code className="mem-fact__slug">{arch.archivedAt.slice(0, 19)}</code>
                    </div>
                    <div className="mem-fact__actions">
                      <button className="settings-btn settings-btn--primary settings-btn--sm"
                        onClick={() => handleRestore(arch.name)}>
                        ↩ {isZnZz(isZh, '\u6062\u590d', 'Restore')}
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* ═══ 指令文件 ═══ */}
      {subTab === 'docs' && (
        <>
          <div className="memory-section-header">
            <div>
              <h4 className="memory-section-title">{isZnZz(isZh, '\u6307\u4ee4\u6587\u4ef6', 'Instruction Files')}</h4>
              <p className="memory-section-desc">
                {isZnZz(isZh, '\u8fd9\u4e9b Markdown \u6587\u4ef6\u4f1a\u4f1c\u4e3a\u7cfb\u7edf\u4e0a\u4e0b\u6587\uff0c\u9002\u5408\u4fdd\u5b58\u957f\u671f\u89c4\u5219\u548c\u9879\u76ee\u7ea6\u5b9a\u3002',
                  'These Markdown files serve as system context, suitable for long-term rules and project conventions.')}
              </p>
            </div>
          </div>

          {docsCount === 0 ? (
            <div className="memory-empty">
              <span className="memory-empty-icon">📄</span>
              <p className="memory-empty-desc">
                {isZnZz(isZh, '\u672a\u627e\u5230 ZHULONG.md\uff0c\u53ef\u5728\u4e0a\u65b9\u5feb\u901f\u6dfb\u52a0\u4e00\u6761\u3002',
                  'No ZHULONG.md found. You can quickly add one above.')}
              </p>
              <button className="settings-btn settings-btn--primary" onClick={() => {
                setDocEditPath('ZHULONG.md'); setDocDraft(`# ZHULONG\n\n## 项目约定\n\n- 使用 Go 语言开发\n- 遵循 DeepSeek prefix-cache 最佳实践\n`)
              }}>
                + {isZnZz(isZh, '\u521b\u5efa ZHULONG.md', 'Create ZHULONG.md')}
              </button>
            </div>
          ) : (
            <div className="memory-docs-list">
              {docs.map(doc => (
                <div key={doc.path} className="mem-doc">
                  {docEditPath === doc.path ? (
                    <div className="mem-doc__edit">
                      <div className="mem-doc__edit-header">
                        <code>{doc.path}</code>
                        <div className="mem-doc__edit-actions">
                          <button className="settings-btn settings-btn--ghost settings-btn--sm"
                            onClick={() => { setDocEditPath(null); setDocDraft('') }}>
                            {isZnZz(isZh, '\u53d6\u6d88', 'Cancel')}
                          </button>
                          <button className="settings-btn settings-btn--primary settings-btn--sm" onClick={handleSaveDoc}>
                            {isZnZz(isZh, '\u4fdd\u5b58', 'Save')}
                          </button>
                        </div>
                      </div>
                      <textarea className="settings-input settings-textarea mem-doc-editor" rows={12}
                        value={docDraft} onChange={e => setDocDraft(e.target.value)} />
                    </div>
                  ) : (
                    <>
                      <div className="mem-doc__head" onClick={() => { setDocEditPath(doc.path); setDocDraft(doc.body) }}>
                        <span className="mem-doc__name">{doc.path}</span>
                        <span className="mem-doc__scope">{`${
                          doc.scope === 'user' ? (isZh ? '\u7528\u6237' : 'User')
                          : doc.scope === 'project' ? (isZh ? '\u9879\u76ee' : 'Project')
                          : (isZh ? '\u672c\u5730' : 'Local')
                        }`}</span>
                        <span className="mem-doc__time">{doc.updatedAt?.slice(0, 10)}</span>
                      </div>
                      <div className="mem-doc__body">
                        <pre>{doc.body.slice(0, 200)}{doc.body.length > 200 ? '...' : ''}</pre>
                      </div>
                      <div className="mem-doc__actions">
                        <button className="settings-btn settings-btn--ghost settings-btn--sm"
                          onClick={() => { setDocEditPath(doc.path); setDocDraft(doc.body) }}>
                          ✎ {isZnZz(isZh, '\u7f16\u8f91', 'Edit')}
                        </button>
                        <button className="settings-btn settings-btn--ghost settings-btn--danger-sm"
                          onClick={() => handleDeleteDoc(doc.path)}>✕</button>
                      </div>
                    </>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* ═══ 模选建议 ═══ */}
      {subTab === 'suggestions' && (
        <>
          <div className="memory-section-header">
            <div>
              <h4 className="memory-section-title">{isZnZz(isZh, '\u6a21\u9009\u5efa\u8bae', 'Model Suggestions')}</h4>
              <p className="memory-section-desc">
                {isZnZz(isZh, '\u4ece\u8fd1\u671f\u672c\u5730\u5386\u53f2\u4e2d\u63d0\u53d6\u5019\u9009\uff1b\u786e\u8ba4\u540e\u4f1a\u5165\u8bb0\u5fc6\u6216 Skill\u3002', 'Extract candidates from recent local history; confirm to register as memory or Skill.')}
              </p>
            </div>
            <button className="settings-btn settings-btn--outline" onClick={handleScanSuggestions}>
              🔄 {isZnZz(isZh, '\u624b\u52a8\u626b\u63cf\u5386\u53f2', 'Manual Scan History')}
            </button>
          </div>

          {/* 自动生成开关 */}
          <div className="memory-suggest-options">
            <div className="memory-suggest-opt">
              <h5>{isZnZz(isZh, '\u81ea\u52a8\u751f\u6210\u6a21\u9009', 'Auto-generate Suggestions')}</h5>
              <ToggleSwitch checked={autoSuggest} onChange={() => setAutoSuggest(!autoSuggest)} />
            </div>
            <p className="memory-suggest-hint">
              {isZnZz(isZh, '\u5f00\u542f\u540e\u8fdb\u5165\u5efa\u8bae\u4f1a\u81ea\u52a8\u626b\u63cf\u8fd1\u671f\u5386\u53f2\uff1b\u786e\u8ba4\u540c\u610f\u52a8\u4f5c\u540e\u624d\u4f1a\u4fdd\u5b58\u3002', 'When enabled, suggestions auto-scan recent history on entry; confirmed actions only persist after approval.')}
            </p>
          </div>

          {/* 建议列表 */}
          {!suggestions ? (
            <div className="memory-empty">
              <h4 className="memory-empty-title">{isZnZz(isZh, '\u4ece\u5386\u53f2\u751f\u6210\u6a21\u9009', 'Generate from History')}</h4>
              <p className="memory-empty-desc">
                {isZnZz(isZh, '\u626b\u63cf\u8fd1\u671f\u4f1a\u8bdd\uff0c\u627e\u51fa\u53ef\u80fd\u503c\u5f97\u4fdd\u5b58\u7684\u8bb0\u5fc6\u548c\u91cd\u590d\u51fa\u73b0\u7684\u5de5\u4f5c\u6d41\u3002\u7ed3\u679c\u53ea\u4f5c\u4e3a\u5355\u7a97\u5c55\u793a\u3002',
                  'Scan recent conversations to find memories worth keeping and recurring workflows. Results shown as single-pane preview.')}
              </p>
              <button className="settings-btn settings-btn--primary" onClick={handleScanSuggestions}>
                ✨ {isZnZz(isZh, '\u624b\u52a8\u626b\u63cf\u5386\u53f2', 'Manual Scan History')}
              </button>
            </div>
          ) : (
            <div className="memory-suggestions-list">
              {suggestions.map(sug => (
                <div key={sug.id} className="mem-suggestion">
                  <div className="mem-suggestion__header">
                    <span className="mem-suggestion__title">{sug.title}</span>
                    <span className={`mem-fact__type-badge mem-type--${sug.type}`}>{sug.type}</span>
                  </div>
                  {sug.reason && <p className="mem-suggestion__reason">{sug.reason}</p>}
                  <pre className="mem-suggestion__body">{sug.body}</pre>
                  {sug.evidence?.length > 0 && (
                    <ul className="mem-suggestion__evidence">
                      {sug.evidence.map((ev, i) => <li key={i}>{ev}</li>)}
                    </ul>
                  )}
                  <div className="mem-suggestion__actions">
                    <button className="settings-btn settings-btn--primary settings-btn--sm"
                      onClick={() => handleAcceptSuggestion(sug)}>
                      ✓ {isZh ? '接受' : 'Accept'}
                    </button>
                    <button className="settings-btn settings-btn--ghost settings-btn--sm"
                      onClick={() => setSuggestions(prev => prev?.filter(s => s.id !== sug.id) ?? null)}>
                      ✕ {isZh ? '忽略' : 'Dismiss'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}

// ════════════════════════════════════════
// Hooks Settings — Shell 自动化
// ════════════════════════════════════════

function HooksSettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null
  const [scope, setScope] = useState<'global' | 'project'>('global')
  const [hooksJson, setHooksJson] = useState<string>(() =>
    loadJSON('zhulong-hooks-json', '{\n  "hooks": {}\n}')
  )
  const [configPath] = useState('~/.zhulong/settings.json')

  useEffect(() => {
    saveJSON('zhulong-hooks-json', hooksJson)
    // 同步到后端
    if (backend) {
      try {
        backend.SetConfigField('hooks', JSON.parse(hooksJson))
      } catch {}
    }
  }, [hooksJson])

  const handleFormat = () => {
    try { setHooksJson(JSON.stringify(JSON.parse(hooksJson), null, 2)) } catch {}
  }

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">Hooks</h3>
      <p className="settings-page__desc">
        {isZnZz(isZh, '配置围绕对话、工具调用、压缩和会话生命周期执行的 shell hooks。',
          'Configure shell hooks executed around conversations, tool calls, compression and session lifecycle.')}
      </p>

      {/* 配置范围 */}
      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '配置范围', 'Configuration Scope')}</h4>
        <p className="settings-section__desc">
          {isZnZz(isZh, 'Hooks 会执行 shell 命令。全局 hooks 属于你本人；项目 hooks 必须显式信任后才会加载。',
            'Hooks execute shell commands. Global hooks belong to you; project hooks must be explicitly trusted before loading.')}
        </p>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, '范围', 'Scope')}</span>
          <select className="settings-select" value={scope} onChange={e => setScope(e.target.value as any)}>
            <option value="global">{isZnZz(isZh, '全局', 'Global')}</option>
            <option value="project">{isZnZz(isZh, '项目', 'Project')}</option>
          </select>
        </div>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, '设置文件', 'Config File')}</span>
          <div className="settings-row__value">
            <span className="settings-row__path">{configPath}</span>
            <button className="settings-btn settings-btn--ghost settings-btn--sm" onClick={() => navigator.clipboard.writeText(configPath)}>
              {isZnZz(isZh, '复制路径', 'Copy Path')}
            </button>
          </div>
        </div>
      </div>

      {/* JSON 配置 */}
      <div className="settings-section">
        <div className="settings-section__header">
          <h4 className="settings-section__title">Hooks</h4>
          <span className="settings-section__hint">
            {isZnZz(isZh, '保存为全局配置。重启烛龙后加载新的 hooks。',
              'Saved as global config. Restart Zhulong to load new hooks.')}
          </span>
        </div>
        <p className="settings-section__desc">
          {isZnZz(isZh, 'JSON 配置 — 直接编辑 settings.json 中的 {"hooks": [...]}，也支持粘贴带 event 字段的 hook 数组；保存前会校验并格式化。',
            'JSON Config — Edit {"hooks": [...]} in settings.json directly, or paste hook arrays with event field; validates and formats before saving.')}
        </p>
        <div className="hooks-json-actions">
          <button className="settings-btn settings-btn--ghost settings-btn--sm" onClick={() => navigator.clipboard.writeText(hooksJson)}>
            {isZnZz(isZh, '复制 JSON', 'Copy JSON')}
          </button>
          <button className="settings-btn settings-btn--ghost settings-btn--sm" onClick={() => navigator.clipboard.readText().then(setHooksJson)}>
            {isZnZz(isZh, '粘贴 JSON', 'Paste JSON')}
          </button>
          <button className="settings-btn settings-btn--ghost settings-btn--sm" onClick={handleFormat}>
            {isZnZz(isZh, '格式化 JSON', 'Format JSON')}
          </button>
          <button className="settings-btn settings-btn--primary settings-btn--sm" onClick={handleFormat}>
            {isZnZz(isZh, '保存', 'Save')}
          </button>
        </div>
        <textarea className="settings-input settings-textarea hooks-json-editor" rows={14}
          value={hooksJson} onChange={e => setHooksJson(e.target.value)}
          spellCheck={false} />
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// Permissions Settings — 权限管理
// ════════════════════════════════════════

function PermissionsSettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null
  const [mode, setMode] = useState<string>(() => loadJSON('zhulong-perm-mode', 'ask'))
  const [denyRules, setDenyRules] = useState<string[]>(() => loadJSON('zhulong-perm-deny', []))
  const [askRules, setAskRules] = useState<string[]>(() => loadJSON('zhulong-perm-ask', []))
  const [allowRules, setAllowRules] = useState<string[]>(() => loadJSON('zhulong-perm-allow', []))

  useEffect(() => {
    saveJSON('zhulong-perm-mode', mode)
    if (backend) try { backend.SetConfigField('permMode', mode) } catch {}
  }, [mode])
  useEffect(() => {
    saveJSON('zhulong-perm-deny', denyRules)
    if (backend) try { backend.SetConfigField('permDeny', denyRules) } catch {}
  }, [denyRules])
  useEffect(() => {
    saveJSON('zhulong-perm-ask', askRules)
    if (backend) try { backend.SetConfigField('permAsk', askRules) } catch {}
  }, [askRules])
  useEffect(() => {
    saveJSON('zhulong-perm-allow', allowRules)
    if (backend) try { backend.SetConfigField('permAllow', allowRules) } catch {}
  }, [allowRules])

  const addRule = (list: string, rule: string) => {
    if (!rule.trim()) return
    if (list === 'deny') setDenyRules(prev => [...prev, rule.trim()])
    else if (list === 'ask') setAskRules(prev => [...prev, rule.trim()])
    else setAllowRules(prev => [...prev, rule.trim()])
  }
  const removeRule = (list: string, idx: number) => {
    if (list === 'deny') setDenyRules(prev => prev.filter((_, i) => i !== idx))
    else if (list === 'ask') setAskRules(prev => prev.filter((_, i) => i !== idx))
    else setAllowRules(prev => prev.filter((_, i) => i !== idx))
  }

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, '权限', 'Permissions')}</h3>
      <p className="settings-page__desc">
        {isZnZz(isZh, '写操作模式与细粒度工具权限规则。', 'Write operation mode and fine-grained tool permission rules.')}
      </p>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '权限', 'Permissions')}</h4>
        <p className="settings-section__desc">
          {isZnZz(isZh, '控制写文件、执行命令等写操作默认如何处理。',
            'Control how write operations like file writing and command execution are handled by default.')}
        </p>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, '写操作模式', 'Write Mode')}</span>
          <select className="settings-select" value={mode} onChange={e => setMode(e.target.value)}>
            <option value="ask">{isZnZz(isZh, 'ask（写操作前询问）', 'ask (ask before write)')}</option>
            <option value="allow">{isZnZz(isZh, 'allow（自动允许）', 'allow (auto allow)')}</option>
            <option value="deny">{isZnZz(isZh, 'deny（自动拒绝）', 'deny (auto deny)')}</option>
          </select>
        </div>
      </div>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '细粒度规则', 'Fine-Grained Rules')}</h4>
        <p className="settings-section__desc">
          {isZnZz(isZh, '规则格式：ToolName 或 ToolName(glob)，优先级：deny > ask > allow。',
            'Rule format: ToolName or ToolName(glob). Priority: deny > ask > allow.')}
        </p>
        <div className="perm-rules-grid">
          <RuleList label={isZnZz(isZh, '阻止', 'Deny')} hint={isZnZz(isZh, '始终阻止匹配的工具或路径。', 'Always block matching tools or paths.')}
            rules={denyRules} onAdd={(r) => addRule('deny', r)} onRemove={(i) => removeRule('deny', i)} isZh={isZh} placeholder={isZnZz(isZh, '添加 deny 规则...', 'Add deny rule...')} />
          <RuleList label={isZnZz(isZh, '询问', 'Ask')} hint={isZnZz(isZh, '运行前先询问确认。', 'Ask for confirmation before running.')}
            rules={askRules} onAdd={(r) => addRule('ask', r)} onRemove={(i) => removeRule('ask', i)} isZh={isZh} placeholder={isZnZz(isZh, '添加 ask 规则...', 'Add ask rule...')} />
          <RuleList label={isZnZz(isZh, '允许', 'Allow')} hint={isZnZz(isZh, '自动放行匹配项。', 'Auto-allow matching items.')}
            rules={allowRules} onAdd={(r) => addRule('allow', r)} onRemove={(i) => removeRule('allow', i)} isZh={isZh} placeholder={isZnZz(isZh, '添加 allow 规则...', 'Add allow rule...')} />
        </div>
      </div>
    </div>
  )
}

function RuleList({ label, hint, rules, onAdd, onRemove, isZh, placeholder }: {
  label: string; hint: string; rules: string[];
  onAdd: (rule: string) => void; onRemove: (idx: number) => void;
  isZh: boolean; placeholder: string
}) {
  const [input, setInput] = useState('')
  return (
    <div className="perm-rule-list">
      <div className="perm-rule-list__header">
        <h5 className="perm-rule-list__title">{label}</h5>
        <p className="perm-rule-list__hint">{hint}</p>
      </div>
      <div className="perm-rule-list__items">
        {rules.length === 0 ? (
          <span className="perm-rule-list__empty">{isZh ? '无' : 'None'}</span>
        ) : rules.map((r, i) => (
          <span key={`${r}-${i}`} className="perm-rule-chip">
            <span>{r}</span>
            <button className="perm-rule-chip__del" onClick={() => onRemove(i)}>✕</button>
          </span>
        ))}
      </div>
      <div className="perm-rule-list__add">
        <input className="settings-input" value={input} onChange={e => setInput(e.target.value)}
          placeholder={placeholder}
          onKeyDown={e => { if (e.key === 'Enter') { onAdd(input); setInput('') } }} />
        <button className="settings-btn settings-btn--ghost settings-btn--sm"
          onClick={() => { onAdd(input); setInput('') }}>{isZh ? '添加' : 'Add'}</button>
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// Sandbox Settings — 沙箱配置
// ════════════════════════════════════════

function SandboxSettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null
  const [shell, setShell] = useState<string>(() => loadJSON('zhulong-sandbox-shell', 'auto'))
  const [bash, setBash] = useState<string>(() => loadJSON('zhulong-sandbox-bash', 'enforce'))
  const [network, setNetwork] = useState<boolean>(() => loadJSON('zhulong-sandbox-network', true))
  const [workspaceRoot, setWorkspaceRoot] = useState<string>(() => loadJSON('zhulong-sandbox-root', ''))
  const [allowWrite, setAllowWrite] = useState<string[]>(() => loadJSON('zhulong-sandbox-allowwrite', []))

  useEffect(() => {
    saveJSON('zhulong-sandbox-shell', shell)
    if (backend) try { backend.SetConfigField('shell', shell) } catch {}
  }, [shell])
  useEffect(() => {
    saveJSON('zhulong-sandbox-bash', bash)
    if (backend) try { backend.SetConfigField('sandboxBash', bash) } catch {}
  }, [bash])
  useEffect(() => {
    saveJSON('zhulong-sandbox-network', network)
    if (backend) try { backend.SetConfigField('sandboxNetwork', network) } catch {}
  }, [network])
  useEffect(() => {
    saveJSON('zhulong-sandbox-root', workspaceRoot)
    if (backend) try { backend.SetConfigField('monitorPath', workspaceRoot) } catch {}
  }, [workspaceRoot])
  useEffect(() => {
    saveJSON('zhulong-sandbox-allowwrite', allowWrite)
    if (backend) try { backend.SetConfigField('allowWrite', allowWrite) } catch {}
  }, [allowWrite])

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, '沙箱', 'Sandbox')}</h3>
      <p className="settings-page__desc">
        {isZnZz(isZh, 'Bash 沙箱、网络出口与工作区根目录。', 'Bash sandbox, network egress and workspace root.')}
      </p>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '沙箱与工作区', 'Sandbox & Workspace')}</h4>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, 'Shell 解释器', 'Shell Interpreter')}</span>
          <select className="settings-select" value={shell} onChange={e => setShell(e.target.value)}>
            <option value="auto">{isZnZz(isZh, '自动（优先 bash，Windows 无 bash 时用 PowerShell）', 'Auto (prefer bash, fallback to PowerShell on Windows)')}</option>
            <option value="bash">Bash</option>
            <option value="powershell">PowerShell</option>
          </select>
        </div>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, 'Bash 沙箱', 'Bash Sandbox')}</span>
          <select className="settings-select" value={bash} onChange={e => setBash(e.target.value)}>
            <option value="enforce">{isZnZz(isZh, 'enforce（隔离 bash）', 'enforce (isolated bash)')}</option>
            <option value="off">{isZnZz(isZh, 'off（关闭）', 'off (disabled)')}</option>
          </select>
        </div>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, '允许沙箱内 bash 访问网络', 'Allow sandboxed bash network access')}</span>
          <label className="settings-toggle">
            <input type="checkbox" checked={network} onChange={e => setNetwork(e.target.checked)} />
            <span className="settings-toggle__track" />
          </label>
        </div>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, '工作区根目录', 'Workspace Root')}</span>
          <input className="settings-input" value={workspaceRoot} onChange={e => setWorkspaceRoot(e.target.value)}
            placeholder={isZnZz(isZh, '（默认：当前目录）', '(default: current directory)')} />
        </div>
        <div className="settings-row settings-row--top">
          <span className="settings-row__label">{isZnZz(isZh, '允许写入路径', 'Allow Write Paths')}</span>
        </div>
        {allowWrite.length === 0 && <span className="perm-rule-list__empty">{isZh ? '无' : 'None'}</span>}
        {allowWrite.map((p, i) => (
          <span key={`${p}-${i}`} className="perm-rule-chip">
            <span>{p}</span>
            <button className="perm-rule-chip__del" onClick={() => setAllowWrite(prev => prev.filter((_, j) => j !== i))}>✕</button>
          </span>
        ))}
        <div className="perm-rule-list__add">
          <input className="settings-input" placeholder={isZnZz(isZh, '添加 allow_write 规则...', 'Add allow_write rule...')}
            onKeyDown={e => {
              if (e.key === 'Enter' && e.currentTarget.value.trim()) {
                setAllowWrite(prev => [...prev, e.currentTarget.value.trim()])
                e.currentTarget.value = ''
              }
            }} />
          <button className="settings-btn settings-btn--ghost settings-btn--sm"
            onClick={(e) => { const inp = (e.currentTarget.parentElement as HTMLElement).querySelector('input') as HTMLInputElement; if (inp?.value.trim()) { setAllowWrite(prev => [...prev, inp.value.trim()]); inp.value = '' } }}>
            {isZh ? '添加' : 'Add'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// Network Settings — 网络配置
// ════════════════════════════════════════

function NetworkSettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const backend = typeof window !== 'undefined' && (window as any).go?.main?.App ? (window as any).go.main.App : null
  const [proxyMode, setProxyMode] = useState<string>(() => loadJSON('zhulong-proxy-mode', 'auto'))
  const [proxyType, setProxyType] = useState<string>(() => loadJSON('zhulong-proxy-type', 'socks5'))
  const [proxyServer, setProxyServer] = useState<string>(() => loadJSON('zhulong-proxy-server', ''))
  const [proxyPort, setProxyPort] = useState<string>(() => loadJSON('zhulong-proxy-port', '7890'))
  const [proxyUrl, setProxyUrl] = useState<string>(() => loadJSON('zhulong-proxy-url', ''))
  const [noProxy, setNoProxy] = useState<string>(() => loadJSON('zhulong-no-proxy', 'localhost,127.0.0.1'))

  useEffect(() => {
    saveJSON('zhulong-proxy-mode', proxyMode)
    if (backend) try { backend.SetConfigField('proxyMode', proxyMode) } catch {}
  }, [proxyMode])
  useEffect(() => { saveJSON('zhulong-proxy-type', proxyType) }, [proxyType])
  useEffect(() => {
    saveJSON('zhulong-proxy-server', proxyServer)
    if (backend && proxyServer) try { backend.SetConfigField('proxyUrl', `${proxyType}://${proxyServer}:${proxyPort}`) } catch {}
  }, [proxyServer, proxyType, proxyPort])
  useEffect(() => { saveJSON('zhulong-proxy-port', proxyPort) }, [proxyPort])
  useEffect(() => {
    saveJSON('zhulong-proxy-url', proxyUrl)
    if (backend) try { backend.SetConfigField('proxyUrl', proxyUrl) } catch {}
  }, [proxyUrl])
  useEffect(() => {
    saveJSON('zhulong-no-proxy', noProxy)
    if (backend) try { backend.SetConfigField('noProxy', noProxy) } catch {}
  }, [noProxy])
  useEffect(() => { saveJSON('zhulong-no-proxy', noProxy) }, [noProxy])

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, '网络', 'Network')}</h3>
      <p className="settings-page__desc">
        {isZnZz(isZh, '代理与网络配置。', 'Proxy and network configuration.')}
      </p>

      <div className="settings-section">
        <div className="settings-section__header">
          <h4 className="settings-section__title">{isZnZz(isZh, '网络', 'Network')}</h4>
        </div>
        <div className="settings-row">
          <span className="settings-row__label">{isZnZz(isZh, '代理模式', 'Proxy Mode')}</span>
          <div className="settings-segmented">
            {(['auto', 'custom', 'off'] as const).map(m => (
              <button key={m} className={`settings-segmented__item ${proxyMode === m ? 'active' : ''}`}
                onClick={() => setProxyMode(m)}>
                {isZnZz(isZh, m === 'auto' ? '自动' : m === 'custom' ? '自定义' : '关闭', m)}
              </button>
            ))}
          </div>
        </div>

        {proxyMode === 'custom' && (
          <>
            <div className="settings-row">
              <span className="settings-row__label">{isZnZz(isZh, '代理类型', 'Proxy Type')}</span>
              <div className="settings-segmented">
                {(['http', 'https', 'socks5', 'socks5h'] as const).map(t => (
                  <button key={t} className={`settings-segmented__item ${proxyType === t ? 'active' : ''}`}
                    onClick={() => setProxyType(t)}>
                    {t.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>
            <div className="settings-row">
              <span className="settings-row__label">{isZnZz(isZh, '代理服务器', 'Server')}</span>
              <div className="settings-inline-group">
                <input className="settings-input settings-input--sm" value={proxyServer}
                  onChange={e => setProxyServer(e.target.value)} placeholder="127.0.0.1" />
                <span className="settings-inline-sep">:</span>
                <input className="settings-input settings-input--xs" value={proxyPort}
                  onChange={e => setProxyPort(e.target.value)} placeholder="7890" />
              </div>
            </div>
            <div className="settings-row">
              <span className="settings-row__label">{isZnZz(isZh, '代理 URL', 'Proxy URL')}</span>
              <input className="settings-input" value={proxyUrl}
                onChange={e => setProxyUrl(e.target.value)} placeholder="socks5://127.0.0.1:7890" />
            </div>
            <div className="settings-row">
              <span className="settings-row__label">{isZnZz(isZh, '不代理', 'No Proxy')}</span>
              <input className="settings-input" value={noProxy}
                onChange={e => setNoProxy(e.target.value)} placeholder="localhost,127.0.0.1" />
            </div>
          </>
        )}
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// Appearance Settings — 外观
// ════════════════════════════════════════

function AppearanceSettings({ language, darkMode, onToggleDarkMode }: { language: Language; darkMode: boolean; onToggleDarkMode: () => void }) {
  const isZh = resolveLanguage(language) === 'zh'
  const [theme, setTheme] = useState<string>(() => loadJSON('zhulong-theme', 'auto'))
  const [themeStyle, setThemeStyle] = useState<string>(() => loadJSON('zhulong-theme-style', 'amber'))
  const [textSize, setTextSize] = useState<string>(() => loadJSON('zhulong-text-size', 'default'))
  const [fontFamily, setFontFamily] = useState<string>(() => loadJSON('zhulong-font-family', 'system'))
  const [monoFont, setMonoFont] = useState<string>(() => loadJSON('zhulong-mono-font', 'system'))

  useEffect(() => { saveJSON('zhulong-theme', theme) }, [theme])
  useEffect(() => { saveJSON('zhulong-theme-style', themeStyle) }, [themeStyle])
  useEffect(() => { saveJSON('zhulong-text-size', textSize) }, [textSize])
  useEffect(() => { saveJSON('zhulong-font-family', fontFamily) }, [fontFamily])
  useEffect(() => { saveJSON('zhulong-mono-font', monoFont) }, [monoFont])

  useEffect(() => { saveJSON('zhulong-theme', theme) }, [theme])
  useEffect(() => { saveJSON('zhulong-theme-style', themeStyle) }, [themeStyle])
  useEffect(() => { saveJSON('zhulong-text-size', textSize) }, [textSize])
  useEffect(() => { saveJSON('zhulong-font-family', fontFamily) }, [fontFamily])
  useEffect(() => { saveJSON('zhulong-mono-font', monoFont) }, [monoFont])

  const themeStyles = [
    { key: 'graphite', name: 'Graphite', zh: '石墨', note: '利落', colors: ['#1a1a1e', '#2a2a2e', '#e87a3f', '#333'], desc: isZh ? '纸面白配石墨文字与橙色强调，利落、克制，贴近编解器工作台。' : 'White paper with graphite text and orange accent, clean and restrained.' },
    { key: 'aurora', name: 'Aurora', zh: '柔雾极光', note: '温润', colors: ['#1a1a2e', '#2a2a3e', '#a855f7', '#444'], desc: isZh ? '柔雾底色融合极光蓝绿，半透明面板与弹性圆角，轻盈，有呼吸感。' : 'Mist base with aurora blue-green, translucent panels and rounded corners.' },
    { key: 'slate', name: 'Slate', zh: '精炼', note: '原生', colors: ['#1e1e22', '#2e2e32', '#3b82f6', '#3a3a3e'], desc: isZh ? '冷灰工作台配品蓝边，发丝边清晰，适合高密度扫描与专业操作。' : 'Cold grey with blue edges, hairline borders, for high-density scanning.' },
    { key: 'carbon', name: 'Carbon', zh: '深邃', note: '高级', colors: ['#0d0d0d', '#1a1a1a', '#22c55e', '#252525'], desc: isZh ? '暖碳黑与米灰表面配青绿强调，质感更厚，对比更足，适合长时间专注。' : 'Warm carbon black with green accent, richer texture for long focus.' },
    { key: 'nocturne', name: 'Nocturne', zh: '柔和', note: '呼吸', colors: ['#12121a', '#1e1e2a', '#ec4899', '#2a2a35'], desc: isZh ? '柔紫夜色和云白表面配大圆角留白，阅读更安静，节奏更舒缓。' : 'Soft purple night and cloud white with large rounded corners.' },
    { key: 'amber', name: 'Amber', zh: '琥珀', note: '暖阳', colors: ['#1a1510', '#2a2518', '#f59e0b', '#332a1a'], desc: isZh ? '暖橙强调色，明亮亲和(含深色变体)。' : 'Warm orange accent, bright and friendly (with dark variant).' },
  ]

  const textSizes = [
    { key: 'small', label: isZh ? '小' : 'S' },
    { key: 'default', label: isZh ? '默认' : 'Default' },
    { key: 'large', label: isZh ? '大' : 'L' },
    { key: 'xlarge', label: isZh ? '特大' : 'XL' },
    { key: 'xxlarge', label: isZh ? '超大' : 'XXL' },
  ]

  const fontOptions = [
    { key: 'system', label: isZh ? '系统默认' : 'System' },
    { key: 'yahei', label: '微软雅黑' },
    { key: 'cascadia', label: 'Cascadia' },
    { key: 'custom', label: isZh ? '自定义' : 'Custom' },
  ]

  const monoOptions = [
    { key: 'system', label: isZh ? '系统等宽' : 'System Mono' },
    { key: 'cascadia', label: 'Cascadia Code' },
    { key: 'custom', label: isZh ? '自定义' : 'Custom' },
  ]

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, '外观', 'Appearance')}</h3>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '主题', 'Theme')}</h4>
        <div className="settings-segmented settings-segmented--right">
          {(['auto', 'light', 'dark'] as const).map(t => (
            <button key={t} className={`settings-segmented__item ${theme === t ? 'active' : ''}`}
              onClick={() => { setTheme(t); if (t === 'dark' && !darkMode) onToggleDarkMode(); if (t === 'light' && darkMode) onToggleDarkMode() }}>
              {isZnZz(isZh, t === 'auto' ? '自动' : t === 'light' ? '浅色' : '深色', t)}
            </button>
          ))}
        </div>
      </div>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '视觉风格', 'Visual Style')}</h4>
        <div className="theme-card-grid">
          {themeStyles.map(s => (
            <button key={s.key} className={`theme-card ${themeStyle === s.key ? 'active' : ''}`}
              onClick={() => setThemeStyle(s.key)}>
              <div className="theme-card__header">
                <span className="theme-card__name">{s.name}</span>
                <span className="theme-card__zh">{s.zh}</span>
                {themeStyle === s.key && <span className="theme-card__check">✓</span>}
              </div>
              <div className="theme-card__note">{s.note}</div>
              <div className="theme-card__swatches">
                {s.colors.map((c, i) => <span key={i} className="theme-card__swatch" style={{ background: c }} />)}
              </div>
              <p className="theme-card__desc">{s.desc}</p>
            </button>
          ))}
        </div>
      </div>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '界面字号', 'Text Size')}</h4>
        <div className="settings-segmented">
          {textSizes.map(s => (
            <button key={s.key} className={`settings-segmented__item ${textSize === s.key ? 'active' : ''}`}
              onClick={() => setTextSize(s.key)}>
              {s.label}
            </button>
          ))}
        </div>
      </div>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '界面字体', 'Font Family')}</h4>
        <div className="settings-segmented">
          {fontOptions.map(f => (
            <button key={f.key} className={`settings-segmented__item ${fontFamily === f.key ? 'active' : ''}`}
              onClick={() => setFontFamily(f.key)}>
              {f.label}
            </button>
          ))}
        </div>
      </div>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '等宽字体', 'Mono Font')}</h4>
        <div className="settings-segmented">
          {monoOptions.map(f => (
            <button key={f.key} className={`settings-segmented__item ${monoFont === f.key ? 'active' : ''}`}
              onClick={() => setMonoFont(f.key)}>
              {f.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

// ════════════════════════════════════════
// Updates Settings — 更新
// ════════════════════════════════════════

function UpdatesSettings({ language }: { language: Language }) {
  const isZh = resolveLanguage(language) === 'zh'
  const [checkUpdates, setCheckUpdates] = useState<boolean>(() => loadJSON('zhulong-check-updates', true))
  const [telemetry, setTelemetry] = useState<boolean>(() => loadJSON('zhulong-telemetry', false))
  const [metrics, setMetrics] = useState<boolean>(() => loadJSON('zhulong-metrics', false))
  const [version] = useState('0.1.0-alpha')
  const [checking, setChecking] = useState(false)

  useEffect(() => { saveJSON('zhulong-check-updates', checkUpdates) }, [checkUpdates])
  useEffect(() => { saveJSON('zhulong-telemetry', telemetry) }, [telemetry])
  useEffect(() => { saveJSON('zhulong-metrics', metrics) }, [metrics])

  const handleCheck = () => {
    setChecking(true)
    setTimeout(() => setChecking(false), 1500)
  }

  return (
    <div className="settings-page">
      <h3 className="settings-page__title">{isZnZz(isZh, '更新', 'Updates')}</h3>
      <p className="settings-page__desc">
        {isZnZz(isZh, '检查软件更新并查看版本信息。', 'Check for software updates and view version info.')}
      </p>

      <div className="settings-section">
        <h4 className="settings-section__title">{isZnZz(isZh, '软件更新', 'Software Updates')}</h4>

        <div className="settings-row">
          <div className="settings-row__text">
            <span className="settings-row__label">{isZnZz(isZh, '启动时检测新版本', 'Check for updates on startup')}</span>
            <span className="settings-row__desc">{isZnZz(isZh, '关闭后，启动时不会自动检查更新；你仍可在此页手动检查。',
              'When off, no auto-check on startup; you can still check manually from this page.')}</span>
          </div>
          <div className="settings-segmented settings-segmented--sm">
            <button className={`settings-segmented__item ${checkUpdates ? 'active' : ''}`} onClick={() => setCheckUpdates(true)}>
              {isZnZz(isZh, '开启', 'On')}
            </button>
            <button className={`settings-segmented__item ${!checkUpdates ? 'active' : ''}`} onClick={() => setCheckUpdates(false)}>
              {isZnZz(isZh, '关闭', 'Off')}
            </button>
          </div>
        </div>

        <div className="settings-row">
          <div className="settings-row__text">
            <span className="settings-row__label">{isZnZz(isZh, '匿名启动统计', 'Anonymous Telemetry')}</span>
            <span className="settings-row__desc">{isZnZz(isZh, '启动时发送匿名的安装 ID、版本号和操作系统用于统计活跃安装量。',
              'Sends anonymous install ID, version and OS on startup for active install counting.')}</span>
          </div>
          <div className="settings-segmented settings-segmented--sm">
            <button className={`settings-segmented__item ${telemetry ? 'active' : ''}`} onClick={() => setTelemetry(true)}>
              {isZnZz(isZh, '开启', 'On')}
            </button>
            <button className={`settings-segmented__item ${!telemetry ? 'active' : ''}`} onClick={() => setTelemetry(false)}>
              {isZnZz(isZh, '关闭', 'Off')}
            </button>
          </div>
        </div>

        <div className="settings-row">
          <div className="settings-row__text">
            <span className="settings-row__label">{isZnZz(isZh, '共享聚合质量指标', 'Share Aggregate Metrics')}</span>
            <span className="settings-row__desc">{isZnZz(isZh, '发送匿名的轮次结束统计（结束原因、缓存命中率、错误与工具失败的类别），用于跨版本发现问题。',
              'Sends anonymous turn-end stats (end reason, cache hit, errors) for cross-version issue finding.')}</span>
          </div>
          <div className="settings-segmented settings-segmented--sm">
            <button className={`settings-segmented__item ${metrics ? 'active' : ''}`} onClick={() => setMetrics(true)}>
              {isZnZz(isZh, '开启', 'On')}
            </button>
            <button className={`settings-segmented__item ${!metrics ? 'active' : ''}`} onClick={() => setMetrics(false)}>
              {isZnZz(isZh, '关闭', 'Off')}
            </button>
          </div>
        </div>
      </div>

      <div className="settings-section">
        <div className="settings-row">
          <span className="settings-row__label">
            {isZnZz(isZh, '当前版本', 'Current Version')}: <strong>{version}</strong>
          </span>
          <button className="settings-btn settings-btn--outline settings-btn--sm" onClick={handleCheck} disabled={checking}>
            {checking ? (isZh ? '检查中...' : 'Checking...') : (isZh ? '检查更新' : 'Check for Updates')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ════════════════════════════════════════

/** 简单的双语辅助函数 */
function isZnZz(isZh: boolean, zn: string, en: string): string {
  return isZh ? zn : en
}

function Row({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="settings-row">
      <div className="settings-row__label">{label}</div>
      {hint && <div className="settings-row__hint">{hint}</div>}
      <div className="settings-row__control">{children}</div>
    </div>
  )
}

function OptionPills(props: { options: { value: string; label: string }[]; selected: string; onChange: (v: string) => void }) {
  return (
    <div className="option-pills">
      {props.options.map(o => (
        <button key={o.value} className={`option-pill ${o.value === props.selected ? 'active' : ''}`}
          onClick={() => props.onChange(o.value)}>{o.label}</button>
      ))}
    </div>
  )
}

function ToggleSwitch(props: { checked: boolean; onChange: () => void; onClick?: (e: React.MouseEvent) => void }) {
  return (
    <button className={`settings-toggle ${props.checked ? 'on' : 'off'}`}
      onClick={props.onClick || props.onChange} role="switch" aria-checked={props.checked}>
      <span className="settings-toggle__thumb" />
    </button>
  )
}
