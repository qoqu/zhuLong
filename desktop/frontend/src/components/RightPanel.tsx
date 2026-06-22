import { useState, useEffect } from 'react'
import type { RightPanelTab, RuntimeStats, Language, FileChange, TreeNode, MemoryState, LearningState, ModuleState } from '../types'
import { useT } from '../i18n'

interface RightPanelProps {
  language: Language
  tab: RightPanelTab
  onChangeTab: (t: RightPanelTab) => void
  stats: RuntimeStats
  files: string[]
  changes: FileChange[]
  memoryState?: MemoryState | null
  learningState?: LearningState | null
  moduleState?: ModuleState | null
}

const backend = typeof window !== 'undefined' && (window as any).go ? (window as any).go.main.App : null

export function RightPanel(props: RightPanelProps) {
  const t = useT(props.language)
  return (
    <aside className="right-panel">
      <div className="right-panel__tabs">
        <button
          className={`right-panel__tab ${props.tab === 'overview' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('overview')}
        >
          <span className="right-panel__tab-icon">≣</span>
          {t.overview}
        </button>
        <button
          className={`right-panel__tab ${props.tab === 'files' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('files')}
        >
          <span className="right-panel__tab-icon">📄</span>
          {t.files}
        </button>
        <button
          className={`right-panel__tab ${props.tab === 'changes' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('changes')}
        >
          <span className="right-panel__tab-icon">↻</span>
          {t.changes}
        </button>
        <button
          className={`right-panel__tab ${props.tab === 'memory' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('memory')}
        >
          <span className="right-panel__tab-icon">🧠</span>
          {t.memory || 'Memory'}
        </button>
        <button
          className={`right-panel__tab ${props.tab === 'learning' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('learning')}
        >
          <span className="right-panel__tab-icon">📚</span>
          {t.learning || 'Learning'}
        </button>
        <button
          className={`right-panel__tab ${props.tab === 'modules' ? 'active' : ''}`}
          onClick={() => props.onChangeTab('modules')}
        >
          <span className="right-panel__tab-icon">⚙️</span>
          {t.modules || 'Modules'}
        </button>
      </div>

      <div className="right-panel__content">
        {props.tab === 'overview' && <OverviewTab language={props.language} stats={props.stats} />}
        {props.tab === 'files' && <FilesTab language={props.language} files={props.files} />}
        {props.tab === 'changes' && <ChangesTab language={props.language} changes={props.changes} />}
        {props.tab === 'memory' && <MemoryTab language={props.language} memoryState={props.memoryState} />}
        {props.tab === 'learning' && <LearningTab language={props.language} learningState={props.learningState} />}
        {props.tab === 'modules' && <ModulesTab language={props.language} moduleState={props.moduleState} />}
      </div>
    </aside>
  )
}

function OverviewTab(props: { language: Language; stats: RuntimeStats }) {
  const t = useT(props.language)
  const s = props.stats
  const fmt = (n: number) => n.toLocaleString()
  return (
    <div className="overview-tab">
      {/* Context window donut */}
      <section className="overview-section">
        <div className="overview-section__header">
          <h3>{t.contextWindow}</h3>
          <span className="overview-section__hint">{t.currentWindowUsage}</span>
        </div>
        <Donut percent={s.usagePercent} used={s.totalUsed} total={s.totalLimit} />
        <ul className="breakdown">
          <li>
            <span className="dot dot--prompt" />
            <span className="label">{t.prompt}</span>
            <span className="value">{fmt(s.prompt)}</span>
          </li>
          <li>
            <span className="dot dot--completion" />
            <span className="label">{t.completion}</span>
            <span className="value">{fmt(s.completion)}</span>
          </li>
          <li>
            <span className="dot dot--reasoning" />
            <span className="label">{t.reasoning}</span>
            <span className="value">{fmt(s.reasoning)}</span>
          </li>
          <li>
            <span className="dot dot--other" />
            <span className="label">{t.other}</span>
            <span className="value">{fmt(s.other)}</span>
          </li>
        </ul>
        <div className="total-line">
          <span>{t.total}</span>
          <span>
            {fmt(s.totalUsed)} / {fmt(s.totalLimit)}
          </span>
        </div>
      </section>

      {/* Runtime */}
      <section className="overview-section">
        <h3>{t.runtime}</h3>
        <div className="runtime-grid">
          <div className="runtime-cell">
            <div className="runtime-cell__label">{t.elapsed}</div>
            <div className="runtime-cell__value">{s.elapsed}</div>
          </div>
          <div className="runtime-cell">
            <div className="runtime-cell__label">{t.requests}</div>
            <div className="runtime-cell__value">{s.requestCount}</div>
          </div>
        </div>
        <div className="runtime-row">
          <span>{t.sessionTokens}</span>
          <span>{fmt(s.sessionTokens)}</span>
        </div>
        <div className="runtime-row">
          <span>{t.cacheBalance}</span>
          <span>{(s.cacheHitRatio * 100).toFixed(0)}%</span>
        </div>
      </section>

      {/* Cost */}
      <section className="overview-section">
        <h3>{t.cost}</h3>
        <div className="cost-grid">
          <div className="cost-cell">
            <div className="cost-cell__label">{t.cacheBalance}</div>
            <div className="cost-cell__value">—</div>
          </div>
          <div className="cost-cell">
            <div className="cost-cell__label">{t.sessionCost}</div>
            <div className="cost-cell__value">{s.sessionCost}</div>
          </div>
        </div>
        <div className="cost-row">
          <span>{t.mainModel}</span>
          <span>
            ${s.mainCost.toFixed(4)}{' '}
            <span className="muted">/ {s.mainCount} 次</span>
          </span>
        </div>
        <div className="cost-row">
          <span>{t.subModel}</span>
          <span>
            ${s.subCost.toFixed(4)}{' '}
            <span className="muted">/ {s.subCount} 次</span>
          </span>
        </div>
      </section>
    </div>
  )
}

function Donut(props: { percent: number; used: number; total: number }) {
  const size = 160
  const stroke = 12
  const r = (size - stroke) / 2
  const c = 2 * Math.PI * r
  const offset = c * (1 - Math.min(props.percent, 1))
  return (
    <div className="donut">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle cx={size / 2} cy={size / 2} r={r} stroke="var(--bg-soft)" strokeWidth={stroke} fill="none" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke="var(--accent)"
          strokeWidth={stroke}
          fill="none"
          strokeDasharray={c}
          strokeDashoffset={offset}
          strokeLinecap="round"
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
        />
        <text x="50%" y="48%" textAnchor="middle" className="donut__big">
          {props.used >= 1000 ? Math.round(props.used / 1000) + 'k' : props.used}
        </text>
        <text x="50%" y="62%" textAnchor="middle" className="donut__small">
          / {Math.round(props.total / 1000)}k tokens
        </text>
      </svg>
      <div className="donut__percent">{props.percent.toFixed(1)}%</div>
    </div>
  )
}

function FilesTab(props: { language: Language; files: string[] }) {
  const t = useT(props.language)
  const [tree, setTree] = useState<TreeNode | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      try {
        if (backend) {
          const result = await backend.ListWorkspaceTree('.', 4)
          if (!cancelled) setTree(result)
        }
      } catch {}
      setLoading(false)
    }
    load()
    return () => { cancelled = true }
  }, [])

  if (loading) {
    return (
      <div className="panel-placeholder">
        <p>⏳</p>
        <p>{props.language === 'zh' ? '加载中...' : 'Loading...'}</p>
      </div>
    )
  }

  if (!tree) {
    // Browser fallback
    if (props.files.length === 0) {
      return (
        <div className="panel-placeholder">
          <p>📄</p>
          <p>{t.noFiles}</p>
        </div>
      )
    }
    return (
      <ul className="file-list">
        {props.files.map((f, i) => (
          <li key={i} className="file-list__item">📄 {f}</li>
        ))}
      </ul>
    )
  }

  return (
    <div className="file-tree">
      <FileTreeNode node={tree} depth={0} />
    </div>
  )
}

function FileTreeNode(props: { node: TreeNode; depth: number }) {
  const { node, depth } = props
  const [expanded, setExpanded] = useState(depth < 2)

  if (node.isDir) {
    return (
      <div className="file-tree__node">
        <button
          className="file-tree__dir"
          style={{ paddingLeft: depth * 16 }}
          onClick={() => setExpanded(!expanded)}
        >
          <span className="file-tree__arrow">{expanded ? '▾' : '▸'}</span>
          <span className="file-tree__icon">📁</span>
          <span className="file-tree__name">{node.name}</span>
        </button>
        {expanded && node.children?.map((child, i) => (
          <FileTreeNode key={child.path || i} node={child} depth={depth + 1} />
        ))}
      </div>
    )
  }

  return (
    <div className="file-tree__node">
      <div className="file-tree__file" style={{ paddingLeft: depth * 16 + 18 }}>
        <span className="file-tree__icon">{fileIcon(node.name)}</span>
        <span className="file-tree__name">{node.name}</span>
        {node.size !== undefined && node.size > 0 && (
          <span className="file-tree__size">{formatSize(node.size)}</span>
        )}
      </div>
    </div>
  )
}

function fileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  switch (ext) {
    case 'go': return '🐹'
    case 'ts': case 'tsx': return '🔷'
    case 'js': case 'jsx': return '🟡'
    case 'json': return '📋'
    case 'md': return '📝'
    case 'css': return '🎨'
    case 'html': return '🌐'
    case 'yaml': case 'yml': return '⚙️'
    case 'toml': return '⚙️'
    case 'gitignore': return '🔒'
    case 'mod': case 'sum': return '📦'
    case 'png': case 'jpg': case 'svg': return '🖼️'
    default: return '📄'
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + 'K'
  return (bytes / (1024 * 1024)).toFixed(1) + 'M'
}

function ChangesTab(props: { language: Language; changes: FileChange[] }) {
  const t = useT(props.language)
  if (props.changes.length === 0) {
    return (
      <div className="panel-placeholder">
        <p>↻</p>
        <p>{t.noChanges}</p>
      </div>
    )
  }
  return (
    <ul className="change-list">
      {props.changes.map((c, i) => (
        <li key={i} className={`change-list__item change-list__item--${c.kind}`}>
          <span className="change-list__kind">{c.kind === 'created' ? '+' : c.kind === 'deleted' ? '−' : '✎'}</span>
          <span className="change-list__path">{c.path}</span>
          <span className="change-list__time">{c.time}</span>
        </li>
      ))}
    </ul>
  )
}

// Memory Tab - 三层记忆状态
function MemoryTab(props: { language: Language; memoryState?: MemoryState | null }) {
  const t = useT(props.language)
  const memory = props.memoryState
  
  if (!memory) {
    return (
      <div className="panel-placeholder">
        <p>🧠</p>
        <p>{props.language === 'zh' ? '暂无记忆数据' : 'No memory data'}</p>
      </div>
    )
  }
  
  return (
    <div className="memory-tab">
      <section className="overview-section">
        <h3>{props.language === 'zh' ? '情景记忆 (Episodic)' : 'Episodic Memory'}</h3>
        <div className="memory-stats">
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '记忆数量' : 'Count'}</span>
            <span className="value">{memory.episodic.count}</span>
          </div>
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '总Token' : 'Total Tokens'}</span>
            <span className="value">{memory.episodic.totalTokens.toLocaleString()}</span>
          </div>
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '最后更新' : 'Last Updated'}</span>
            <span className="value">{memory.episodic.lastUpdated}</span>
          </div>
        </div>
      </section>
      
      <section className="overview-section">
        <h3>{props.language === 'zh' ? '语义记忆 (Semantic)' : 'Semantic Memory'}</h3>
        <div className="memory-stats">
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '概念数量' : 'Concepts'}</span>
            <span className="value">{memory.semantic.count}</span>
          </div>
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '分类' : 'Categories'}</span>
            <span className="value">{memory.semantic.categories.join(', ')}</span>
          </div>
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '最后更新' : 'Last Updated'}</span>
            <span className="value">{memory.semantic.lastUpdated}</span>
          </div>
        </div>
      </section>
      
      <section className="overview-section">
        <h3>{props.language === 'zh' ? '程序记忆 (Procedural)' : 'Procedural Memory'}</h3>
        <div className="memory-stats">
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '策略数量' : 'Strategies'}</span>
            <span className="value">{memory.procedural.count}</span>
          </div>
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '成功率' : 'Success Rate'}</span>
            <span className="value">{(memory.procedural.successRate * 100).toFixed(1)}%</span>
          </div>
          <div className="memory-stat-item">
            <span className="label">{props.language === 'zh' ? '最后更新' : 'Last Updated'}</span>
            <span className="value">{memory.procedural.lastUpdated}</span>
          </div>
        </div>
      </section>
    </div>
  )
}

// Learning Tab - 学习状态
function LearningTab(props: { language: Language; learningState?: LearningState | null }) {
  const t = useT(props.language)
  const learning = props.learningState
  
  if (!learning) {
    return (
      <div className="panel-placeholder">
        <p>📚</p>
        <p>{props.language === 'zh' ? '暂无学习数据' : 'No learning data'}</p>
      </div>
    )
  }
  
  return (
    <div className="learning-tab">
      <section className="overview-section">
        <h3>{props.language === 'zh' ? '认知模型' : 'Cognitive Model'}</h3>
        <div className="learning-stats">
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '已更新' : 'Updated'}</span>
            <span className="value">{learning.cognitiveModel.updated ? (props.language === 'zh' ? '是' : 'Yes') : (props.language === 'zh' ? '否' : 'No')}</span>
          </div>
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '置信度' : 'Confidence'}</span>
            <span className="value">{(learning.cognitiveModel.confidence * 100).toFixed(1)}%</span>
          </div>
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '最后更新' : 'Last Update'}</span>
            <span className="value">{learning.cognitiveModel.lastUpdate}</span>
          </div>
        </div>
      </section>
      
      <section className="overview-section">
        <h3>{props.language === 'zh' ? '多样性' : 'Diversity'}</h3>
        <div className="learning-stats">
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '多样性分数' : 'Diversity Score'}</span>
            <span className="value">{(learning.diversity.score * 100).toFixed(1)}%</span>
          </div>
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '策略数量' : 'Strategies'}</span>
            <span className="value">{learning.diversity.strategies.length}</span>
          </div>
        </div>
      </section>
      
      <section className="overview-section">
        <h3>{props.language === 'zh' ? '探索与利用' : 'Exploration vs Exploitation'}</h3>
        <div className="learning-stats">
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '探索率' : 'Exploration Rate'}</span>
            <span className="value">{(learning.explorationRate * 100).toFixed(1)}%</span>
          </div>
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '利用率' : 'Utilization Rate'}</span>
            <span className="value">{(learning.utilizationRate * 100).toFixed(1)}%</span>
          </div>
          <div className="learning-stat-item">
            <span className="label">{props.language === 'zh' ? '成功模式' : 'Success Patterns'}</span>
            <span className="value">{learning.successPatterns}</span>
          </div>
        </div>
      </section>
    </div>
  )
}

// Modules Tab - 模块状态
function ModulesTab(props: { language: Language; moduleState?: ModuleState | null }) {
  const t = useT(props.language)
  const modules = props.moduleState
  
  if (!modules) {
    return (
      <div className="panel-placeholder">
        <p>⚙️</p>
        <p>{props.language === 'zh' ? '暂无模块数据' : 'No module data'}</p>
      </div>
    )
  }
  
  const statusColor = (status: string) => {
    switch (status) {
      case 'active': return 'var(--ok)'
      case 'idle': return 'var(--fg-faint)'
      case 'error': return 'var(--err)'
      default: return 'var(--fg-faint)'
    }
  }
  
  const statusLabel = (status: string) => {
    if (props.language === 'en') {
      return status.charAt(0).toUpperCase() + status.slice(1)
    }
    switch (status) {
      case 'active': return '活跃'
      case 'idle': return '空闲'
      case 'error': return '错误'
      default: return status
    }
  }
  
  return (
    <div className="modules-tab">
      <section className="overview-section">
        <h3>P0 {props.language === 'zh' ? '核心模块' : 'Core Modules'}</h3>
        <div className="module-list">
          <ModuleItem name="Controller" module={modules.controller} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Planner" module={modules.planner} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Executor" module={modules.executor} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Reflector" module={modules.reflector} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Memory" module={modules.memory} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Compressor" module={modules.compressor} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Checkpoint" module={modules.checkpoint} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Budget" module={modules.budget} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Trace" module={modules.trace} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Human" module={modules.human} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="Tools" module={modules.tools} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
          <ModuleItem name="DeepSeek" module={modules.deepseek} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />
        </div>
      </section>
      
      {(modules.stagnation || modules.exploration || modules.stability || modules.information || modules.synergetics || modules.learningModule) && (
        <section className="overview-section">
          <h3>P1 {props.language === 'zh' ? '核心增强' : 'Core Enhancement'}</h3>
          <div className="module-list">
            {modules.stagnation && <ModuleItem name="Stagnation" module={modules.stagnation} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.exploration && <ModuleItem name="Exploration" module={modules.exploration} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.stability && <ModuleItem name="Stability" module={modules.stability} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.information && <ModuleItem name="Information" module={modules.information} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.synergetics && <ModuleItem name="Synergetics" module={modules.synergetics} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.learningModule && <ModuleItem name="Learning" module={modules.learningModule} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
          </div>
        </section>
      )}
      
      {(modules.altPlanner || modules.envMonitor || modules.noiseHandler || modules.redundancy) && (
        <section className="overview-section">
          <h3>P2 {props.language === 'zh' ? '扩展模块' : 'Extension Modules'}</h3>
          <div className="module-list">
            {modules.altPlanner && <ModuleItem name="AltPlanner" module={modules.altPlanner} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.envMonitor && <ModuleItem name="EnvMonitor" module={modules.envMonitor} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.noiseHandler && <ModuleItem name="NoiseHandler" module={modules.noiseHandler} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.redundancy && <ModuleItem name="Redundancy" module={modules.redundancy} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
          </div>
        </section>
      )}
      
      {(modules.i18n || modules.plugins || modules.dashboard || modules.models || modules.backup) && (
        <section className="overview-section">
          <h3>P3 {props.language === 'zh' ? '扩展功能' : 'Extended Features'}</h3>
          <div className="module-list">
            {modules.i18n && <ModuleItem name="i18n" module={modules.i18n} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.plugins && <ModuleItem name="Plugins" module={modules.plugins} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.dashboard && <ModuleItem name="Dashboard" module={modules.dashboard} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.models && <ModuleItem name="Models" module={modules.models} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
            {modules.backup && <ModuleItem name="Backup" module={modules.backup} statusColor={statusColor} statusLabel={statusLabel} language={props.language} />}
          </div>
        </section>
      )}
    </div>
  )
}

function ModuleItem(props: { 
  name: string
  module: any
  statusColor: (status: string) => string
  statusLabel: (status: string) => string
  language: Language
}) {
  const mod = props.module
  return (
    <div className="module-item">
      <div className="module-item__header">
        <span className="module-item__name">{props.name}</span>
        <span className="module-item__status" style={{ color: props.statusColor(mod.status) }}>
          {props.statusLabel(mod.status)}
        </span>
      </div>
      <div className="module-item__details">
        {mod.fsmState && <span>{props.language === 'zh' ? 'FSM' : 'FSM'}: {mod.fsmState}</span>}
        {mod.lastPlan && <span>{props.language === 'zh' ? '最后规划' : 'Last Plan'}: {mod.lastPlan}</span>}
        {mod.toolsLoaded !== undefined && <span>{props.language === 'zh' ? '工具' : 'Tools'}: {mod.toolsLoaded}</span>}
        {mod.lastReflection && <span>{props.language === 'zh' ? '最后反省' : 'Last Reflection'}: {mod.lastReflection}</span>}
        {mod.compactionEnabled !== undefined && <span>{props.language === 'zh' ? '压缩' : 'Compaction'}: {mod.compactionEnabled ? (props.language === 'zh' ? '开启' : 'On') : (props.language === 'zh' ? '关闭' : 'Off')}</span>}
        {mod.compressThreshold !== undefined && <span>{props.language === 'zh' ? '压缩阈值' : 'Compress Threshold'}: {mod.compressThreshold}%</span>}
        {mod.warningLevel && <span>{props.language === 'zh' ? '警告级别' : 'Warning Level'}: {mod.warningLevel}</span>}
        {mod.traceEnabled !== undefined && <span>{props.language === 'zh' ? 'Trace' : 'Trace'}: {mod.traceEnabled ? (props.language === 'zh' ? '开启' : 'On') : (props.language === 'zh' ? '关闭' : 'Off')}</span>}
        {mod.approvalPending !== undefined && <span>{props.language === 'zh' ? '审批待定' : 'Approval Pending'}: {mod.approvalPending ? (props.language === 'zh' ? '是' : 'Yes') : (props.language === 'zh' ? '否' : 'No')}</span>}
        {mod.mcpConnected !== undefined && <span>MCP: {mod.mcpConnected ? (props.language === 'zh' ? '已连接' : 'Connected') : (props.language === 'zh' ? '未连接' : 'Disconnected')}</span>}
        {mod.cacheHitRate !== undefined && <span>{props.language === 'zh' ? '缓存命中率' : 'Cache Hit Rate'}: {(mod.cacheHitRate * 100).toFixed(1)}%</span>}
      </div>
    </div>
  )
}
