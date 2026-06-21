// Mock data for Zhulong UI browser fallback
import type { AgentInfo, ProjectInfo, RuntimeStats } from '../types'

export const mockAgents: AgentInfo[] = [
  { id: 'auto', name: '默认 Agent', model: 'deepseek-v4-flash', yolo: true },
]

export const mockProjects: ProjectInfo[] = [
  {
    id: 'global',
    name: 'Global',
    sessions: [],
  },
]

export const mockStats: RuntimeStats = {
  totalUsed: 0,
  totalLimit: 1_000_000,
  usagePercent: 0,
  prompt: 0,
  completion: 0,
  reasoning: 0,
  other: 0,
  elapsed: '0s',
  requestCount: 0,
  sessionTokens: 0,
  cacheHitRatio: 0.8,
  mainCost: 0,
  mainCount: 0,
  subCost: 0,
  subCount: 0,
  balance: '¥73.23',
  currentSession: 0,
  sessionCost: '$0.0000',
  model: 'deepseek-v4-flash',
  cacheHit: '未命中',
  avgHit: '未命中',
  thisTokens: 0,
  thisFee: '$0.0000',
  contextUsed: 0,
  compressPct: 80,
  remaining: '¥73.23',
}

export const exampleGoals = [
  { icon: '📊', text: { en: 'Analyze the codebase and suggest improvements', zh: '分析代码库并提出改进建议' } },
  { icon: '🔧', text: { en: 'Find and fix all TypeScript errors', zh: '查找并修复所有 TypeScript 错误' } },
  { icon: '🧪', text: { en: 'Write unit tests for the main module', zh: '为主模块编写单元测试' } },
  { icon: '🌐', text: { en: 'Search the web for latest AI news', zh: '搜索最新 AI 新闻' } },
]
