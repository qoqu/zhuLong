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
export type RightPanelTab = 'overview' | 'files' | 'changes'
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
        }
      }
    }
    runtime?: WailsRuntime
  }
}
