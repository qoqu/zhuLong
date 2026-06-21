// Mock data for Zhulong UI browser fallback
import type { AgentInfo, ProjectInfo, RuntimeStats } from '../types'

export const mockAgents: AgentInfo[] = [
  { id: 'auto', name: '自适应游…', model: 'deepseek-v4-flash', yolo: true },
  { id: 'umit', name: 'umit', model: 'deepseek-v4-flash', yolo: true },
  { id: 'mnemonic', name: 'mnemon…', model: 'deepseek-v4-flash', yolo: true },
]

const sessionTitles = [
  ['自适应游工作室', '24 轮 · 5 工具 · 5天前'],
  ['mnemonic全笔记2.mcp', '195 轮 · 4 工具 · 4天前'],
  ['umit', '5 轮 · 6 工具 · 5天前'],
  ['V2构建', '25 轮 · 6 工具 · 6天前'],
  ['热点创作工作流—今天有什么…', '41 轮 · 6 工具 · 6天前'],
  ['2自滚小说工作室', '100 轮 · 5 工具 · 5天前'],
  ['创世日记引擎', '144 轮 · 6 工具 · 6天前'],
  ['1自滚小说作家接手', '13 轮 · 6 工具 · 6天前'],
  ['UUMit 接单工作', '22 轮 · 6 工具 · 6天前'],
  ['reasonix-buddy', '4 轮 · 6 工具 · 6天前'],
]

export const mockProjects: ProjectInfo[] = [
  {
    id: 'global',
    name: 'Global',
    sessions: sessionTitles.map(([title, meta], i) => {
      const [mc, tc, u] = meta.split(' · ')
      return {
        id: 's' + (i + 1),
        title: title!,
        agentId: i === 1 ? 'mnemonic' : i === 2 ? 'umit' : 'auto',
        projectId: 'global',
        messageCount: parseInt(mc!),
        toolCount: parseInt(tc!),
        updatedAt: u!,
        preview: title!,
      }
    }),
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
  elapsed: '15分52秒',
  requestCount: 147,
  sessionTokens: 53_740_187,
  cacheHitRatio: 0.8,
  mainCost: 0.4659,
  mainCount: 112,
  subCost: 0.0159,
  subCount: 35,
  balance: '¥73.23',
  currentSession: 193,
  sessionCost: '$0.4817',
  model: 'deepseek-v4-flash',
  cacheHit: '未命中',
  avgHit: '未命中',
  thisTokens: 0,
  thisFee: '$0.0000',
  contextUsed: 53_740_187,
  compressPct: 80,
  remaining: '¥73.23',
}

export const exampleGoals = [
  { icon: '📊', text: { en: 'Analyze the codebase and suggest improvements', zh: '分析代码库并提出改进建议' } },
  { icon: '🔧', text: { en: 'Find and fix all TypeScript errors', zh: '查找并修复所有 TypeScript 错误' } },
  { icon: '🧪', text: { en: 'Write unit tests for the main module', zh: '为主模块编写单元测试' } },
  { icon: '🌐', text: { en: 'Search the web for latest AI news', zh: '搜索最新 AI 新闻' } },
]
