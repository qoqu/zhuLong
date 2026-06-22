export type Language = 'en' | 'zh'

export type AgentStatus =
  | 'idle'
  | 'planning'
  | 'executing'
  | 'reflecting'
  | 'replanning'
  | 'waiting_human'
  | 'done'
  | 'error'

export type ExecutionMode = 'ask' | 'auto' | 'yolo'
export type InputMode = 'normal' | 'plan' | 'goal'
export type RightPanelTab = 'overview' | 'files' | 'changes' | 'memory' | 'learning' | 'modules'

export type SidebarView = 'projects' | 'agents' | 'history' | 'trash' | 'settings'

export interface AgentInfo {
  id: string
  name: string
  model: string
  yolo: boolean
}

export interface SessionInfo {
  id: string
  title: string
  agentId: string
  projectId: string
  messageCount: number
  toolCount: number
  updatedAt: string
  preview: string
}

export interface ProjectInfo {
  id: string
  name: string
  sessions: SessionInfo[]
  children?: ProjectInfo[]
}

export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  time: string
  toolName?: string
  toolCount?: number
}

export interface LogEntry {
  id: string
  time: string
  phase: string
  event: string
  detail?: string
}

export interface PlanStep {
  id: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'failed'
}

export interface RuntimeStats {
  totalUsed: number
  totalLimit: number
  usagePercent: number
  prompt: number
  completion: number
  reasoning: number
  other: number
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
  model: string
  cacheHit: string
  avgHit: string
  thisTokens: number
  thisFee: string
  contextUsed: number
  compressPct: number
  remaining: string
  // Enhanced UI fields (FSM state, budget info)
  fsmState?: string
  budgetUsed?: number
  budgetLimit?: number
  budgetWarning?: boolean
}

// Memory State - 三层记忆状态
export interface MemoryState {
  episodic: {
    count: number
    totalTokens: number
    lastUpdated: string
  }
  semantic: {
    count: number
    categories: string[]
    lastUpdated: string
  }
  procedural: {
    count: number
    successRate: number
    lastUpdated: string
  }
}

// Learning State - 学习状态
export interface LearningState {
  cognitiveModel: {
    updated: boolean
    lastUpdate: string
    confidence: number
  }
  diversity: {
    score: number
    strategies: string[]
  }
  explorationRate: number
  utilizationRate: number
  successPatterns: number
  lastLearning: string
}

// Module State - 模块状态
export interface ModuleState {
  // P0 modules
  controller: { status: 'active' | 'idle' | 'error'; fsmState: string }
  planner: { status: 'active' | 'idle' | 'error'; lastPlan: string }
  executor: { status: 'active' | 'idle' | 'error'; toolsLoaded: number }
  reflector: { status: 'active' | 'idle' | 'error'; lastReflection: string }
  memory: { status: 'active' | 'idle' | 'error'; compactionEnabled: boolean }
  compressor: { status: 'active' | 'idle' | 'error'; compressThreshold: number }
  checkpoint: { status: 'active' | 'idle' | 'error'; lastCheckpoint: string }
  budget: { status: 'active' | 'idle' | 'error'; warningLevel: 'ok' | 'warn' | 'critical' }
  trace: { status: 'active' | 'idle' | 'error'; traceEnabled: boolean }
  human: { status: 'active' | 'idle' | 'error'; approvalPending: boolean }
  tools: { status: 'active' | 'idle' | 'error'; mcpConnected: boolean }
  deepseek: { status: 'active' | 'idle' | 'error'; cacheHitRate: number }
  
  // P1 modules
  stagnation?: { status: 'active' | 'idle' | 'error'; detected: boolean }
  exploration?: { status: 'active' | 'idle' | 'error'; active: boolean }
  stability?: { status: 'active' | 'idle' | 'error'; oscillating: boolean }
  information?: { status: 'active' | 'idle' | 'error'; gain: number }
  synergetics?: { status: 'active' | 'idle' | 'error'; orderParameter: number }
  learningModule?: { status: 'active' | 'idle' | 'error'; learning: boolean }
  
  // P2 modules
  altPlanner?: { status: 'active' | 'idle' | 'error'; fallbackActive: boolean }
  envMonitor?: { status: 'active' | 'idle' | 'error'; monitoring: boolean }
  noiseHandler?: { status: 'active' | 'idle' | 'error'; filtering: boolean }
  redundancy?: { status: 'active' | 'idle' | 'error'; detected: boolean }
  
  // P3 modules
  i18n?: { status: 'active' | 'idle' | 'error'; language: string }
  plugins?: { status: 'active' | 'idle' | 'error'; loaded: number }
  dashboard?: { status: 'active' | 'idle' | 'error'; visible: boolean }
  models?: { status: 'active' | 'idle' | 'error'; poolSize: number }
  backup?: { status: 'active' | 'idle' | 'error'; lastBackup: string }
}

export interface FileChange {
  path: string
  kind: 'created' | 'modified' | 'deleted'
  time: string
}

export interface ApprovalRequest {
  id: string
  tool: string
  args: Record<string, any>
  risk: 'low' | 'medium' | 'high'
  reason: string
  createdAt: string
}

export interface SessionState {
  info: SessionInfo
  goal: string
  status: AgentStatus | string
  mode: ExecutionMode | string
  model: string
  messages: Message[]
  logs: LogEntry[]
  plan: PlanStep[]
  stats: RuntimeStats
  files: string[]
  changes: FileChange[]
  approval?: ApprovalRequest
  created: string
  updated: string
  // New fields for enhanced UI
  memoryState?: MemoryState
  learningState?: LearningState
  moduleState?: ModuleState
}

// Wails runtime types (minimal local copy)
export interface WailsRuntime {
  EventsOn(event: string, callback: (...data: any[]) => void): () => void
  EventsOff(event: string): void
}
declare global {
  interface Window {
    go?: {
      main: {
        App: {
          ListAgents(): Promise<AgentInfo[]>
          ListProjects(): Promise<ProjectInfo[]>
          SetActiveAgent(id: string): Promise<void>
          SetActiveSession(id: string): Promise<SessionState | null>
          GetSession(id: string): Promise<SessionState | null>
          NewSession(): Promise<SessionState>
          DeleteSession(id: string): Promise<void>
          RenameSession(id: string, title: string): Promise<void>
          SendMessage(sessionID: string, text: string): Promise<void>
          Stop(): Promise<void>
          Reset(): Promise<void>
          SetExecutionMode(mode: string): Promise<void>
          SetModel(model: string): Promise<void>
          RespondApproval(id: string, approved: boolean): Promise<void>
          ListWorkspaceTree(root: string, maxDepth: number): Promise<TreeNode>
        }
      }
    }
    runtime?: WailsRuntime
  }
}

export interface TreeNode {
  name: string
  path: string
  isDir: boolean
  children?: TreeNode[]
  size?: number
}
