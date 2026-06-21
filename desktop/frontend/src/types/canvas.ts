// 画布类型定义

// 节点类型枚举
export enum CanvasNodeType {
  // Agent循环节点
  Plan = 'plan',
  Step = 'step',
  Tool = 'tool',
  Result = 'result',

  // 多媒体节点
  Text = 'text',
  Image = 'image',
  Video = 'video',
  Audio = 'audio',
  Storyboard = 'storyboard',
  Config = 'config',

  // 资产节点
  Asset = 'asset',
  Reference = 'reference',
}

// 节点状态
export type NodeStatus = 'idle' | 'loading' | 'success' | 'error' | 'pending' | 'running' | 'completed' | 'failed';

// 基础节点数据
export interface BaseNodeData {
  id: string;
  type: CanvasNodeType;
  title: string;
  status: NodeStatus;
  metadata?: Record<string, any>;
}

// 文本节点数据
export interface TextNodeData extends BaseNodeData {
  type: CanvasNodeType.Text;
  content: string;
  prompt?: string;
  wordCount?: number;
}

// 图片节点数据
export interface ImageNodeData extends BaseNodeData {
  type: CanvasNodeType.Image;
  imageUrl?: string;
  prompt?: string;
  width?: number;
  height?: number;
  model?: string;
  aspect?: string;
  referenceIds?: string[];
}

// 视频节点数据
export interface VideoNodeData extends BaseNodeData {
  type: CanvasNodeType.Video;
  videoUrl?: string;
  prompt?: string;
  duration?: number;
  orientation?: 'landscape' | 'portrait';
  firstFrame?: string;
  lastFrame?: string;
  referenceIds?: string[];
}

// 音频节点数据
export interface AudioNodeData extends BaseNodeData {
  type: CanvasNodeType.Audio;
  audioUrl?: string;
  text?: string;
  voice?: string;
  speed?: number;
  format?: string;
}

// 分镜节点数据
export interface StoryboardNodeData extends BaseNodeData {
  type: CanvasNodeType.Storyboard;
  scenes: StoryboardScene[];
  script?: string;
}

export interface StoryboardScene {
  id: string;
  description: string;
  imageUrl?: string;
  duration?: number;
  cameraAngle?: string;
  dialogue?: string;
}

// 配置节点数据
export interface ConfigNodeData extends BaseNodeData {
  type: CanvasNodeType.Config;
  generationMode: 'text' | 'image' | 'video' | 'audio';
  model: string;
  params: Record<string, any>;
  prompt?: string;
  referenceIds?: string[];
}

// Plan节点数据
export interface PlanNodeData extends BaseNodeData {
  type: CanvasNodeType.Plan;
  goal: string;
  steps: number;
}

// Step节点数据
export interface StepNodeData extends BaseNodeData {
  type: CanvasNodeType.Step;
  stepId: string;
  description: string;
  tool: string;
  result?: string;
  tokensUsed?: number;
  duration?: string;
}

// Tool节点数据
export interface ToolNodeData extends BaseNodeData {
  type: CanvasNodeType.Tool;
  toolName: string;
  params: Record<string, any>;
  result?: string;
}

// Result节点数据
export interface ResultNodeData extends BaseNodeData {
  type: CanvasNodeType.Result;
  summary: string;
  findings: string[];
  suggestions: string[];
  confidence: number;
}

// 资产节点数据
export interface AssetNodeData extends BaseNodeData {
  type: CanvasNodeType.Asset;
  assetType: string;
  content?: string;
  url?: string;
  thumbnailUrl?: string;
}

// 参考节点数据
export interface ReferenceNodeData extends BaseNodeData {
  type: CanvasNodeType.Reference;
  referenceType: 'input' | 'output' | 'reference';
  targetId: string;
}

// 联合节点数据类型
export type CanvasNodeData =
  | TextNodeData
  | ImageNodeData
  | VideoNodeData
  | AudioNodeData
  | StoryboardNodeData
  | ConfigNodeData
  | PlanNodeData
  | StepNodeData
  | ToolNodeData
  | ResultNodeData
  | AssetNodeData
  | ReferenceNodeData;

// 画布操作类型
export type CanvasOp =
  // 节点操作
  | { type: 'add_node'; nodeType: CanvasNodeType; position: { x: number; y: number }; metadata?: Record<string, any> }
  | { type: 'update_node'; id: string; patch?: Partial<CanvasNodeData>; metadata?: Record<string, any> }
  | { type: 'delete_node'; id?: string; ids?: string[]; nodeType?: CanvasNodeType }
  | { type: 'move_node'; id: string; position: { x: number; y: number } }
  | { type: 'resize_node'; id: string; size: { width: number; height: number } }

  // 连线操作
  | { type: 'connect_nodes'; fromNodeId: string; toNodeId: string }
  | { type: 'disconnect_nodes'; fromNodeId: string; toNodeId: string }
  | { type: 'delete_connections'; id?: string; ids?: string[]; all?: boolean }

  // 选择操作
  | { type: 'select_nodes'; ids: string[] }
  | { type: 'deselect_all' }

  // 视图操作
  | { type: 'set_viewport'; viewport: { x: number; y: number; zoom: number } }
  | { type: 'fit_view' }
  | { type: 'zoom_in' }
  | { type: 'zoom_out' }

  // 分组操作
  | { type: 'group_nodes'; ids: string[]; groupId?: string }
  | { type: 'ungroup_nodes'; groupId: string }

  // 生成操作
  | { type: 'run_generation'; nodeId: string; mode?: string; prompt?: string }
  | { type: 'cancel_generation'; nodeId: string }

  // 资产操作
  | { type: 'create_asset'; assetType: string; input: any }
  | { type: 'update_asset'; id: string; patch: any }
  | { type: 'delete_asset'; id: string }

  // 批量操作
  | { type: 'batch_ops'; ops: CanvasOp[] };

// 画布快照
export interface CanvasSnapshot {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  viewport: Viewport;
}

// 画布节点
export interface CanvasNode {
  id: string;
  type: CanvasNodeType;
  data: CanvasNodeData;
  position: { x: number; y: number };
  size: { width: number; height: number };
}

// 画布连线
export interface CanvasEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
}

// 视口
export interface Viewport {
  x: number;
  y: number;
  zoom: number;
}

// 资产类型
export enum AssetType {
  Text = 'text',
  Image = 'image',
  Video = 'video',
  Audio = 'audio',
  Script = 'script',
  Storyboard = 'storyboard',
  Outline = 'outline',
  Character = 'character',
  Scene = 'scene',
  Prop = 'prop',
  Reference = 'reference',
  Template = 'template',
}

// 资产
export interface Asset {
  id: string;
  type: AssetType;
  name: string;
  description?: string;
  content?: string;
  url?: string;
  thumbnailUrl?: string;
  metadata?: Record<string, any>;
  projectId: string;
  createdAt: string;
  updatedAt: string;
  parentId?: string;
  deriveType?: 'variant' | 'version' | 'branch';
}

// 衍生资产
export interface DeriveAsset extends Asset {
  parentId: string;
  deriveType: 'variant' | 'version' | 'branch';
  prompt?: string;
  state: 'pending' | 'generating' | 'completed' | 'failed';
}

// 画布助手消息
export interface AssistantMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
  references?: ResourceReference[];
}

// 资源引用
export interface ResourceReference {
  type: 'node' | 'asset';
  id: string;
  title: string;
  content?: string;
  imageUrl?: string;
}

// 画布助手会话
export interface AssistantSession {
  id: string;
  title: string;
  messages: AssistantMessage[];
  createdAt: string;
  updatedAt: string;
}

// 画布项目
export interface CanvasProject {
  id: string;
  name: string;
  description?: string;
  assets: Asset[];
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  assistantSessions: AssistantSession[];
  activeAssistantSessionId?: string;
  createdAt: string;
  updatedAt: string;
}

// ========== 多Agent工作空间协作（参考TapCanvas） ==========

// Agent信息
export interface AgentInfo {
  id: string;
  name: string;
  role: string;
  status: 'idle' | 'busy' | 'error' | 'offline';
  capabilities: string[];
  currentTask?: string;
  lastActive: string;
}

// Agent消息
export interface AgentMessage {
  id: string;
  fromAgentId: string;
  toAgentId: string; // '*' 表示广播
  type: 'request' | 'response' | 'notification' | 'broadcast';
  protocol: string;
  payload: any;
  status: 'pending' | 'sent' | 'delivered' | 'read' | 'failed';
  timestamp: string;
  responseTo?: string;
}

// 工作空间移交请求
export interface WorkspaceHandoff {
  id: string;
  fromAgentId: string;
  toAgentId: string;
  assets: string[];       // 移交的资产ID
  context: string;        // 移交上下文
  status: 'pending' | 'accepted' | 'rejected' | 'completed';
  createdAt: string;
}

// Agent工作空间
export interface AgentWorkspace {
  id: string;
  name: string;
  description?: string;
  agents: AgentInfo[];
  messages: AgentMessage[];
  handoffs: WorkspaceHandoff[];
  sharedAssets: string[]; // 共享的资产ID
  ownerId: string;
  createdAt: string;
  updatedAt: string;
}

// ========== 章节事件图谱（参考Toonflow） ==========

// 章节事件
export interface ChapterEvent {
  id: string;
  chapterId: string;
  eventId: string;
  description: string;
  characters: string[];  // 涉及角色
  locations: string[];   // 涉及场景
  time: string;          // 时间点
  importance: 'low' | 'medium' | 'high';
  dependencies: string[]; // 依赖的其他事件
  metadata?: Record<string, any>;
}

// 章节
export interface Chapter {
  id: string;
  title: string;
  summary: string;
  events: string[]; // 事件ID列表
  characters: string[]; // 章节中的角色
  locations: string[];  // 章节中的场景
  metadata?: Record<string, any>;
}

// 事件关系
export interface EventRelationship {
  id: string;
  sourceEventId: string;
  targetEventId: string;
  type: 'causes' | 'follows' | 'contradicts' | 'supports' | 'references';
  description?: string;
}

// 章节事件图谱
export interface ChapterGraph {
  id: string;
  name: string;
  description?: string;
  chapters: Chapter[];
  events: ChapterEvent[];
  relationships: EventRelationship[];
  characters: CharacterInfo[];
  locations: LocationInfo[];
  createdAt: string;
  updatedAt: string;
}

// 角色信息
export interface CharacterInfo {
  id: string;
  name: string;
  description?: string;
  aliases: string[];
  relationships: CharacterRelationship[];
  appearances: string[]; // 出现的章节ID
  metadata?: Record<string, any>;
}

// 角色关系
export interface CharacterRelationship {
  characterId: string;
  relationship: string; // 如：朋友、敌人、家人等
  description?: string;
}

// 场景信息
export interface LocationInfo {
  id: string;
  name: string;
  description?: string;
  aliases: string[];
  appearances: string[]; // 出现的章节ID
  metadata?: Record<string, any>;
}

// ========== 五层内容生产架构（参考Toonflow） ==========

// 生产层级
export enum ProductionLayer {
  Import = 'import',       // 小说导入
  Parse = 'parse',         // 内容解析
  Character = 'character', // 角色生成
  Script = 'script',       // 剧本生成
  Storyboard = 'storyboard', // 分镜生成
  Video = 'video',         // 视频生成
}

// 生产状态
export type ProductionStatus = 'idle' | 'running' | 'completed' | 'failed' | 'paused';

// 生产管道
export interface ProductionPipeline {
  id: string;
  name: string;
  description?: string;
  layers: ProductionLayer[];
  currentLayer: ProductionLayer;
  status: ProductionStatus;
  progress: Record<ProductionLayer, number>; // 每层进度 0-100
  config: ProductionConfig;
  createdAt: string;
  updatedAt: string;
}

// 生产配置
export interface ProductionConfig {
  import?: ImportConfig;
  parse?: ParseConfig;
  character?: CharacterConfig;
  script?: ScriptConfig;
  storyboard?: StoryboardConfig;
  video?: VideoConfig;
}

// 导入配置
export interface ImportConfig {
  sourceType: 'file' | 'url' | 'text';
  source: string;
  format: 'txt' | 'docx' | 'pdf' | 'epub';
  encoding?: string;
}

// 解析配置
export interface ParseConfig {
  splitBy: 'chapter' | 'scene' | 'paragraph';
  minChapterLength?: number;
  maxChapterLength?: number;
  extractEvents: boolean;
  extractCharacters: boolean;
  extractLocations: boolean;
}

// 角色配置
export interface CharacterConfig {
  generatePortraits: boolean;
  portraitStyle?: string;
  extractRelationships: boolean;
  generateDescriptions: boolean;
}

// 剧本配置
export interface ScriptConfig {
  format: 'screenplay' | 'stage' | 'radio';
  includeDirections: boolean;
  includeDialogue: boolean;
  targetLength?: number;
}

// 分镜配置
export interface StoryboardConfig {
  style: 'realistic' | 'anime' | 'cartoon' | 'sketch';
  aspectRatio: '16:9' | '4:3' | '1:1' | '9:16';
  shotsPerScene?: number;
  includeDialogue: boolean;
  includeDirections: boolean;
}

// 视频配置
export interface VideoConfig {
  resolution: '720p' | '1080p' | '4k';
  fps: 24 | 30 | 60;
  style: string;
  duration?: number;
  includeAudio: boolean;
  audioType?: 'narration' | 'music' | 'both';
}

// 生产层级结果
export interface LayerResult {
  layer: ProductionLayer;
  status: ProductionStatus;
  progress: number;
  output?: any;
  error?: string;
  startedAt?: string;
  completedAt?: string;
}
