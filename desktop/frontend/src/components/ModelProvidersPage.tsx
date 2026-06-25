import { useState } from 'react'
import type { Language } from '../types'
import { resolveLanguage } from '../i18n'

// ─── Types ───
interface ModelInfo {
  id: string
  name: string
  enabled: boolean
}

interface Provider {
  id: string
  name: string
  type: 'official' | 'custom'        // 官方 | 自定义
  source: 'builtin' | 'user'         // 内置 | 用户添加
  keySet: boolean                     // 是否已设置密钥
  description: string                 // e.g. "DeepSeek 官方 OpenAI-compatible 接入"
  apiType: string                     // openai / anthropic / ...
  baseUrl: string                     // API URL
  apiKeyEnv: string                   // 环境变量名
  models: ModelInfo[]
}

// ─── Default providers ───
const DEFAULT_PROVIDERS: Provider[] = [
  {
    id: 'deepseek',
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

// ─── Component Props ───
interface ModelProvidersPageProps {
  language: Language
  onBack?: () => void
}

export function ModelProvidersPage(props: ModelProvidersPageProps) {
  const isZh = resolveLanguage(props.language) === 'zh'

  const [tab, setTab] = useState<'use' | 'connect'>('connect')
  const [providers, setProviders] = useState<Provider[]>(() => {
    try {
      const saved = localStorage.getItem('zhulong-model-providers')
      return saved ? JSON.parse(saved) : DEFAULT_PROVIDERS
    } catch { return DEFAULT_PROVIDERS }
  })

  // Persist providers to localStorage
  const saveProviders = (ps: Provider[]) => {
    setProviders(ps)
    localStorage.setItem('zhulong-model-providers', JSON.stringify(ps))
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
    if (isZh && !window.confirm('确定要移除此供应商？')) return
    saveProviders(providers.filter(p => p.id !== id))
  }

  return (
    <div className="model-providers-page">
      {/* ══ Header ═══ */}
      <div className="model-providers__header">
        <h1 className="model-providers__title">{isZh ? '模型' : 'Models'}</h1>
        <p className="model-providers__desc">
          {isZh
            ? '默认模型、规划模型、运行上限与接入概览。'
            : 'Default models, planning models, runtime limits & provider overview.'
          }
        </p>
      </div>

      {/* ══ Tabs ═══ */}
      <div className="model-providers__tabs">
        <button
          className={`model-providers__tab ${tab === 'use' ? 'active' : ''}`}
          onClick={() => setTab('use')}
        >{isZh ? '使用' : 'Use'}</button>
        <button
          className={`model-providers__tab ${tab === 'connect' ? 'active' : ''}`}
          onClick={() => setTab('connect')}
        >{isZh ? '接入' : 'Connect'}</button>
      </div>

      {/* ══ Tab Content ═══ */}
      {tab === 'use' && (
        <div className="model-providers__tab-content">
          <div style={{ textAlign: 'center', padding: '60px 20px', color: 'var(--fg-faint)' }}>
            <div style={{ fontSize: 32, marginBottom: 12 }}>🤖</div>
            <div style={{ fontSize: 14 }}>{isZh ? '模型使用统计即将上线' : 'Model usage stats coming soon'}</div>
          </div>
        </div>
      )}

      {tab === 'connect' && (
        <div className="model-providers__tab-content">
          {/* Section header */}
          <div className="model-providers__section-header">
            <span className="model-providers__section-title">{isZh ? '供应商接入' : 'Provider Access'}</span>
            <button className="model-providers__add-btn">
              + {isZh ? '添加模型服务' : 'Add Provider'}
            </button>
          </div>

          {/* Hint text */}
          <p className="model-providers__hint">
            {isZh
              ? '添加官方或自定义供应商后，才会出现在这里。会话模型列表只显示已保存的启用模型。'
              : 'After adding official or custom providers, they will appear here. The session model list only shows enabled models.'
            }
          </p>

          {/* Provider cards */}
          {providers.length === 0 ? (
            <div className="model-providers__empty">
              <div style={{ fontSize: 32, marginBottom: 8 }}>🔌</div>
              <div>{isZh ? '暂无已接入的模型供应商' : 'No model providers connected yet'}</div>
              <div style={{ fontSize: 12, color: 'var(--fg-mute)', marginTop: 4 }}>
                {isZh ? '点击上方「添加模型服务」开始配置' : 'Click "Add Provider" above to get started'}
              </div>
            </div>
          ) : (
            <div className="model-providers__list">
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
        </div>
      )}
    </div>
  )
}

// ─── Provider Card Component ───
function ProviderCard({
  provider,
  isZh,
  onToggleModel,
  onRemove,
}: {
  provider: Provider
  isZh: boolean
  onToggleModel: (modelId: string) => void
  onRemove: () => void
}) {
  const enabledCount = provider.models.filter(m => m.enabled).length

  return (
    <div className="provider-card">
      {/* Card header row: Name + badges + actions */}
      <div className="provider-card__header">
        <div className="provider-card__name-row">
          <span className="provider-card__name">{provider.name}</span>
          <span className={`provider-card__badge provider-card__badge--${provider.type}`}>
            {provider.type === 'official' ? (isZh ? '官方' : 'Official') : (isZh ? '自定义' : 'Custom')}
          </span>
          <span className={`provider-card__badge provider-card__badge--${provider.source}`}>
            {provider.source === 'builtin' ? (isZh ? '内置' : 'Builtin') : (isZh ? '用户' : 'User')}
          </span>
          {provider.keySet && (
            <span className={`provider-card__badge provider-card__badge--key`}>
              {isZh ? '已设密钥' : 'Key Set'}
            </span>
          )}
        </div>
        <div className="provider-card__actions">
          <button className="provider-card__action-btn" title={isZh ? '配置' : 'Config'}>
            {isZh ? '配置' : 'Config'}
          </button>
          <button className="provider-card__action-btn" title={isZh ? '刷新模型' : 'Refresh Models'}>
            {isZh ? '刷新模型' : 'Refresh'}
          </button>
          {provider.source === 'user' && (
            <button className="provider-card__action-btn provider-card__action-btn--danger" onClick={onRemove} title={isZh ? '移除接入' : 'Remove'}>
              {isZh ? '移除接入' : 'Remove'}
            </button>
          )}
        </div>
      </div>

      {/* Description */}
      <div className="provider-card__desc">{provider.description || provider.baseUrl}</div>

      {/* API details row */}
      <div className="provider-card__api-row">
        <span className="provider-card__api-item">{provider.apiType}</span>
        <span className="provider-card__api-dot">·</span>
        <span className="provider-card__api-item provider-card__api-url">{provider.baseUrl}</span>
        <span className="provider-card__api-dot">·</span>
        <span className="provider-card__api-key">{provider.apiKeyEnv}</span>
      </div>

      {/* Enabled models tags */}
      {provider.models.length > 0 && (
        <div className="provider-card__models">
          <span className="provider-card__models-label">
            {isZh ? '已启用模型' : 'Enabled Models'} ({enabledCount}/{provider.models.length})
          </span>
          <div className="provider-card__models-tags">
            {provider.models.map(m => (
              <button
                key={m.id}
                className={`provider-card__model-tag ${m.enabled ? '' : 'disabled'}`}
                onClick={() => onToggleModel(m.id)}
                title={m.enabled ? (isZh ? '点击禁用' : 'Click to disable') : (isZh ? '点击启用' : 'Click to enable')}
              >
                {m.name}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
