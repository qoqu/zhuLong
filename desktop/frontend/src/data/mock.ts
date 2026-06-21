// Mock data for Zhulong UI prototype
import type { Agent, Message, LogEntry, PlanStep, RuntimeStats } from '../types'

export const mockAgents: Agent[] = [
  { id: 'auto', name: '自适应游...', model: 'deepseek-v4-flash', yolo: true },
  { id: 'umit', name: 'umit', model: 'deepseek-v4-flash', yolo: true },
  { id: 'mnemonic', name: 'mnemon...', model: 'deepseek-v4-flash', yolo: true },
]

export const mockProjects = [
  {
    id: 'global',
    name: 'Global',
    sessions: [
      {
        id: 's1',
        agentId: 'auto',
        title: '自适应游工作室',
        messageCount: 24,
        toolCount: 5,
        updatedAt: '5天前',
        preview: '自适应游工作室',
      },
      {
        id: 's2',
        agentId: 'auto',
        title: 'mnemonic全笔记2.mcp',
        messageCount: 195,
        toolCount: 4,
        updatedAt: '4天前',
        preview: 'mnemonic全笔记2.mcp',
      },
      {
        id: 's3',
        agentId: 'umit',
        title: 'umit',
        messageCount: 5,
        toolCount: 6,
        updatedAt: '5天前',
        preview: 'umit',
      },
      {
        id: 's4',
        agentId: 'auto',
        title: 'V2构建',
        messageCount: 25,
        toolCount: 6,
        updatedAt: '6天前',
        preview: 'V2构建',
      },
      {
        id: 's5',
        agentId: 'auto',
        title: '热点创作工作流—今天有什么...',
        messageCount: 41,
        toolCount: 6,
        updatedAt: '6天前',
        preview: '热点创作工作流—今天有什么...',
      },
      {
        id: 's6',
        agentId: 'auto',
        title: '2自滚小说工作室',
        messageCount: 100,
        toolCount: 5,
        updatedAt: '5天前',
        preview: '2自滚小说工作室',
      },
      {
        id: 's7',
        agentId: 'auto',
        title: '创世日记引擎',
        messageCount: 144,
        toolCount: 6,
        updatedAt: '6天前',
        preview: '创世日记引擎',
      },
      {
        id: 's8',
        agentId: 'auto',
        title: '1自滚小说作家接手',
        messageCount: 13,
        toolCount: 6,
        updatedAt: '6天前',
        preview: '1自滚小说作家接手',
      },
      {
        id: 's9',
        agentId: 'auto',
        title: 'UUMit 接单工作',
        messageCount: 22,
        toolCount: 6,
        updatedAt: '6天前',
        preview: 'UUMit 接单工作',
      },
      {
        id: 's10',
        agentId: 'auto',
        title: 'reasonix-buddy',
        messageCount: 4,
        toolCount: 6,
        updatedAt: '6天前',
        preview: 'reasonix-buddy',
      },
    ],
  },
]

export const mockMessages: Message[] = [
  {
    id: 'm0',
    role: 'system',
    content: '发现新版本: v1.10.0',
    timestamp: new Date(),
  },
  {
    id: 'm1',
    role: 'user',
    content: '这个项目使用说明？',
    timestamp: new Date(),
  },
  {
    id: 'm2',
    role: 'assistant',
    content: '`mnemonic` 是一个为研究/笔记工作流设计的全功能 mcp 项目。它包含：',
    timestamp: new Date(),
  },
  {
    id: 'm3',
    role: 'user',
    content: '推送了吗？',
    timestamp: new Date(),
    toolCount: 4,
  },
  {
    id: 'm4',
    role: 'user',
    content: '中文文档同步更新了吗？',
    timestamp: new Date(),
    toolCount: 4,
  },
  {
    id: 'm5',
    role: 'user',
    content: '现在我要迁移了，怎么使用？',
    timestamp: new Date(),
    toolCount: 9,
  },
  {
    id: 'm6',
    role: 'user',
    content: '怎么保证不会把同样的内容导入知识库？',
    timestamp: new Date(),
    toolCount: 18,
  },
  {
    id: 'm7',
    role: 'user',
    content: '之前已经导出的做了记录没？',
    timestamp: new Date(),
    toolCount: 2,
  },
  {
    id: 'm8',
    role: 'user',
    content: '导出的记录应该也属于迁移的数据库？防止到了新的设备上，恢复对话后又重新导出。',
    timestamp: new Date(),
    toolCount: 0,
  },
  {
    id: 'm9',
    role: 'user',
    content: '记住这个 The user wants me to remember this research about PlotPilot for future s...',
    timestamp: new Date(),
    toolCount: 1,
  },
  {
    id: 'm10',
    role: 'user',
    content: '手动记录的触发词是记住这个，但是为什么不是默认mnemonic',
    timestamp: new Date(),
    toolCount: 0,
  },
  {
    id: 'm11',
    role: 'user',
    content: '这应该是一个研发类的吧，去神经元吧？',
    timestamp: new Date(),
    toolCount: 7,
  },
  {
    id: 'm12',
    role: 'user',
    content: '原项目地址：https://github.com/thedotmack/claude-mem，仓库下载的资料"D:\\Desktop\\claude-mem-...',
    timestamp: new Date(),
    toolCount: 4,
  },
  {
    id: 'm13',
    role: 'user',
    content: '不对啊，我现在是在研究项目啊，你到底保存记忆啊。',
    timestamp: new Date(),
    toolCount: 0,
  },
  {
    id: 'm14',
    role: 'user',
    content: '1秒出来，不要直接抄代码，自己写，实现一致的功能。2秒出来，不要直接抄代码，自己写，实现一致的功能。3...',
    timestamp: new Date(),
    toolCount: 14,
  },
]

export const mockLogs: LogEntry[] = [
  { id: 'l1', time: '14:23:01', phase: 'plan', event: 'Plan created', detail: '4 steps' },
  { id: 'l2', time: '14:23:02', phase: 'exec', event: 'Step 1/4 done', detail: 'read_file' },
  { id: 'l3', time: '14:23:04', phase: 'exec', event: 'Step 2/4 done', detail: 'grep' },
  { id: 'l4', time: '14:23:06', phase: 'refl', event: 'Reflecting...', detail: 'progress=0.5' },
  { id: 'l5', time: '14:23:07', phase: 'exec', event: 'Step 3/4 done', detail: 'write_file' },
  { id: 'l6', time: '14:23:09', phase: 'exec', event: 'Step 4/4 done', detail: 'execute_command' },
  { id: 'l7', time: '14:23:10', phase: 'done', event: 'Goal achieved', detail: '4 steps, 6s' },
]

export const mockPlan: PlanStep[] = [
  { id: '1', description: 'Analyze the task and gather context', status: 'completed' },
  { id: '2', description: 'Read main.go to understand structure', status: 'completed' },
  { id: '3', description: 'Refactor to use approval engine', status: 'running' },
  { id: '4', description: 'Verify and run tests', status: 'pending' },
]

export const mockStats: RuntimeStats = {
  totalUsed: 0,
  totalLimit: 1000000,
  usagePercent: 0,
  breakdown: { prompt: 0, completion: 0, reasoning: 0, other: 0 },
  elapsed: '15分52秒',
  requestCount: 147,
  sessionTokens: 53740187,
  cacheHitRatio: 0.8,
  mainCost: 0.4659,
  mainCount: 112,
  subCost: 0.0159,
  subCount: 35,
  balance: '¥73.23',
  currentSession: 193,
  sessionCost: '¥0.4817',
}

export const exampleGoals = [
  { icon: '📊', text: { en: 'Analyze the codebase and suggest improvements', zh: '分析代码库并提出改进建议' } },
  { icon: '🔧', text: { en: 'Find and fix all TypeScript errors', zh: '查找并修复所有 TypeScript 错误' } },
  { icon: '🧪', text: { en: 'Write unit tests for the main module', zh: '为主模块编写单元测试' } },
  { icon: '🌐', text: { en: 'Search the web for latest AI news', zh: '搜索最新 AI 新闻' } },
]
