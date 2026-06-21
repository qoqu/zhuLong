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
