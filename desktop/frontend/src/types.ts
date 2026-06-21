// Shared types for Zhulong desktop UI

export type AgentStatus =
  | 'idle'
  | 'planning'
  | 'executing'
  | 'reflecting'
  | 'replanning'
  | 'waiting_human'
  | 'done'
  | 'error'

export type Agent = {
  id: string
  name: string
  model: string
  yolo?: boolean
}

export type Session = {
  id: string
  agentId: string
  title: string
  messageCount: number
  toolCount: number
  updatedAt: string
  preview: string
  scope?: 'global' | string
  parentId?: string
}

export type Message = {
  id: string
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  timestamp: Date
  toolName?: string
  toolCount?: number
  toolCalls?: ToolCall[]
}

export type ToolCall = {
  id: string
  name: string
  args: string
  result?: string
  status: 'pending' | 'running' | 'completed' | 'failed'
}

export type LogEntry = {
  id: string
  time: string
  phase: string
  event: string
  detail?: string
}

export type PlanStep = {
  id: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'failed'
}

export type ExecutionMode = 'ask' | 'auto' | 'yolo'
export type InputMode = 'normal' | 'plan' | 'goal'
export type RightPanelTab = 'overview' | 'files' | 'changes'
export type SidebarView = 'projects' | 'robots' | 'history' | 'recycle' | 'settings'
export type Language = 'en' | 'zh'

export type ContextTokenBreakdown = {
  prompt: number
  completion: number
  reasoning: number
  other: number
}

export type RuntimeStats = {
  totalUsed: number
  totalLimit: number
  usagePercent: number
  breakdown: ContextTokenBreakdown
  elapsed: string
  requestCount: number
  sessionTokens: number
  cacheHitRatio: number
  mainCost: number
  mainCount: number
  subCost: number
  subCount: number
  balance: string
  currentSession: number
  sessionCost: string
}
