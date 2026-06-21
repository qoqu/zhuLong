// 画布状态管理
import { create } from 'zustand';
import type {
  CanvasNode,
  CanvasEdge,
  CanvasOp,
  CanvasSnapshot,
  Viewport,
  Asset,
  AssistantSession,
  AssistantMessage,
} from '../types/canvas';

// 画布状态接口
interface CanvasState {
  // 基础数据
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  viewport: Viewport;

  // 选择状态
  selectedNodeIds: string[];
  selectedEdgeIds: string[];

  // 历史记录（撤销/重做）
  past: CanvasSnapshot[];
  future: CanvasSnapshot[];

  // 资产
  assets: Asset[];

  // 助手会话
  assistantSessions: AssistantSession[];
  activeAssistantSessionId?: string;

  // 操作方法
  addNode: (node: CanvasNode) => void;
  updateNode: (id: string, patch: Partial<CanvasNode>) => void;
  deleteNode: (id: string) => void;
  deleteNodes: (ids: string[]) => void;

  addEdge: (edge: CanvasEdge) => void;
  deleteEdge: (id: string) => void;
  deleteEdges: (ids: string[]) => void;

  selectNodes: (ids: string[]) => void;
  deselectAll: () => void;

  setViewport: (viewport: Viewport) => void;
  fitView: () => void;
  zoomIn: () => void;
  zoomOut: () => void;

  // 批量操作
  applyOps: (ops: CanvasOp[]) => void;

  // 撤销/重做
  undo: () => void;
  redo: () => void;

  // 资产管理
  addAsset: (asset: Asset) => void;
  updateAsset: (id: string, patch: Partial<Asset>) => void;
  deleteAsset: (id: string) => void;

  // 助手会话
  addAssistantSession: (session: AssistantSession) => void;
  updateAssistantSession: (id: string, patch: Partial<AssistantSession>) => void;
  deleteAssistantSession: (id: string) => void;
  setActiveAssistantSession: (id: string) => void;
  addAssistantMessage: (sessionId: string, message: AssistantMessage) => void;

  // 快照
  getSnapshot: () => CanvasSnapshot;
  restoreSnapshot: (snapshot: CanvasSnapshot) => void;
}

// 创建画布状态store
export const useCanvasStore = create<CanvasState>((set, get) => ({
  // 初始状态
  nodes: [],
  edges: [],
  viewport: { x: 0, y: 0, zoom: 1 },
  selectedNodeIds: [],
  selectedEdgeIds: [],
  past: [],
  future: [],
  assets: [],
  assistantSessions: [],
  activeAssistantSessionId: undefined,

  // 节点操作
  addNode: (node) => {
    const state = get();
    const snapshot = getSnapshot(state);
    set({
      nodes: [...state.nodes, node],
      past: [...state.past, snapshot],
      future: [],
    });
  },

  updateNode: (id, patch) => {
    const state = get();
    const snapshot = getSnapshot(state);
    set({
      nodes: state.nodes.map((node) =>
        node.id === id ? { ...node, ...patch } : node
      ),
      past: [...state.past, snapshot],
      future: [],
    });
  },

  deleteNode: (id) => {
    const state = get();
    const snapshot = getSnapshot(state);
    set({
      nodes: state.nodes.filter((node) => node.id !== id),
      edges: state.edges.filter(
        (edge) => edge.source !== id && edge.target !== id
      ),
      selectedNodeIds: state.selectedNodeIds.filter((nodeId) => nodeId !== id),
      past: [...state.past, snapshot],
      future: [],
    });
  },

  deleteNodes: (ids) => {
    const state = get();
    const snapshot = getSnapshot(state);
    const idSet = new Set(ids);
    set({
      nodes: state.nodes.filter((node) => !idSet.has(node.id)),
      edges: state.edges.filter(
        (edge) => !idSet.has(edge.source) && !idSet.has(edge.target)
      ),
      selectedNodeIds: state.selectedNodeIds.filter(
        (nodeId) => !idSet.has(nodeId)
      ),
      past: [...state.past, snapshot],
      future: [],
    });
  },

  // 连线操作
  addEdge: (edge) => {
    const state = get();
    const snapshot = getSnapshot(state);
    set({
      edges: [...state.edges, edge],
      past: [...state.past, snapshot],
      future: [],
    });
  },

  deleteEdge: (id) => {
    const state = get();
    const snapshot = getSnapshot(state);
    set({
      edges: state.edges.filter((edge) => edge.id !== id),
      selectedEdgeIds: state.selectedEdgeIds.filter((edgeId) => edgeId !== id),
      past: [...state.past, snapshot],
      future: [],
    });
  },

  deleteEdges: (ids) => {
    const state = get();
    const snapshot = getSnapshot(state);
    const idSet = new Set(ids);
    set({
      edges: state.edges.filter((edge) => !idSet.has(edge.id)),
      selectedEdgeIds: state.selectedEdgeIds.filter(
        (edgeId) => !idSet.has(edgeId)
      ),
      past: [...state.past, snapshot],
      future: [],
    });
  },

  // 选择操作
  selectNodes: (ids) => {
    set({ selectedNodeIds: ids });
  },

  deselectAll: () => {
    set({ selectedNodeIds: [], selectedEdgeIds: [] });
  },

  // 视口操作
  setViewport: (viewport) => {
    set({ viewport });
  },

  fitView: () => {
    // TODO: 实现自适应视图
    set({ viewport: { x: 0, y: 0, zoom: 1 } });
  },

  zoomIn: () => {
    const state = get();
    set({
      viewport: {
        ...state.viewport,
        zoom: Math.min(state.viewport.zoom * 1.2, 3),
      },
    });
  },

  zoomOut: () => {
    const state = get();
    set({
      viewport: {
        ...state.viewport,
        zoom: Math.max(state.viewport.zoom / 1.2, 0.1),
      },
    });
  },

  // 批量操作
  applyOps: (ops) => {
    const state = get();
    const snapshot = getSnapshot(state);
    let currentNodes = [...state.nodes];
    let currentEdges = [...state.edges];

    for (const op of ops) {
      switch (op.type) {
        case 'add_node':
          currentNodes.push({
            id: `node-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            type: op.nodeType,
            data: {
              id: `node-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
              type: op.nodeType,
              title: op.metadata?.title || op.nodeType,
              status: 'idle',
              ...op.metadata,
            } as any,
            position: op.position,
            size: { width: 200, height: 100 },
          });
          break;

        case 'update_node':
          currentNodes = currentNodes.map((node) =>
            node.id === op.id
              ? { ...node, ...op.patch, data: { ...node.data, ...op.patch } } as CanvasNode
              : node
          );
          break;

        case 'delete_node':
          if (op.id) {
            currentNodes = currentNodes.filter((node) => node.id !== op.id);
            currentEdges = currentEdges.filter(
              (edge) => edge.source !== op.id && edge.target !== op.id
            );
          } else if (op.ids) {
            const idSet = new Set(op.ids);
            currentNodes = currentNodes.filter((node) => !idSet.has(node.id));
            currentEdges = currentEdges.filter(
              (edge) => !idSet.has(edge.source) && !idSet.has(edge.target)
            );
          }
          break;

        case 'connect_nodes':
          currentEdges.push({
            id: `edge-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            source: op.fromNodeId,
            target: op.toNodeId,
          });
          break;

        case 'disconnect_nodes':
          currentEdges = currentEdges.filter(
            (edge) =>
              !(edge.source === op.fromNodeId && edge.target === op.toNodeId)
          );
          break;

        case 'delete_connections':
          if (op.id) {
            currentEdges = currentEdges.filter((edge) => edge.id !== op.id);
          } else if (op.ids) {
            const idSet = new Set(op.ids);
            currentEdges = currentEdges.filter((edge) => !idSet.has(edge.id));
          } else if (op.all) {
            currentEdges = [];
          }
          break;

        case 'select_nodes':
          set({ selectedNodeIds: op.ids });
          break;

        case 'deselect_all':
          set({ selectedNodeIds: [], selectedEdgeIds: [] });
          break;

        case 'set_viewport':
          set({ viewport: op.viewport });
          break;

        // 其他操作...
      }
    }

    set({
      nodes: currentNodes,
      edges: currentEdges,
      past: [...state.past, snapshot],
      future: [],
    });
  },

  // 撤销/重做
  undo: () => {
    const state = get();
    if (state.past.length === 0) return;

    const previous = state.past[state.past.length - 1];
    const newPast = state.past.slice(0, -1);

    set({
      nodes: previous.nodes,
      edges: previous.edges,
      viewport: previous.viewport,
      past: newPast,
      future: [getSnapshot(state), ...state.future],
    });
  },

  redo: () => {
    const state = get();
    if (state.future.length === 0) return;

    const next = state.future[0];
    const newFuture = state.future.slice(1);

    set({
      nodes: next.nodes,
      edges: next.edges,
      viewport: next.viewport,
      past: [...state.past, getSnapshot(state)],
      future: newFuture,
    });
  },

  // 资产管理
  addAsset: (asset) => {
    const state = get();
    set({ assets: [...state.assets, asset] });
  },

  updateAsset: (id, patch) => {
    const state = get();
    set({
      assets: state.assets.map((asset) =>
        asset.id === id ? { ...asset, ...patch } : asset
      ),
    });
  },

  deleteAsset: (id) => {
    const state = get();
    set({ assets: state.assets.filter((asset) => asset.id !== id) });
  },

  // 助手会话
  addAssistantSession: (session) => {
    const state = get();
    set({
      assistantSessions: [...state.assistantSessions, session],
      activeAssistantSessionId: session.id,
    });
  },

  updateAssistantSession: (id, patch) => {
    const state = get();
    set({
      assistantSessions: state.assistantSessions.map((session) =>
        session.id === id ? { ...session, ...patch } : session
      ),
    });
  },

  deleteAssistantSession: (id) => {
    const state = get();
    const newSessions = state.assistantSessions.filter(
      (session) => session.id !== id
    );
    set({
      assistantSessions: newSessions,
      activeAssistantSessionId:
        state.activeAssistantSessionId === id
          ? newSessions[0]?.id
          : state.activeAssistantSessionId,
    });
  },

  setActiveAssistantSession: (id) => {
    set({ activeAssistantSessionId: id });
  },

  addAssistantMessage: (sessionId, message) => {
    const state = get();
    set({
      assistantSessions: state.assistantSessions.map((session) =>
        session.id === sessionId
          ? { ...session, messages: [...session.messages, message] }
          : session
      ),
    });
  },

  // 快照
  getSnapshot: () => {
    const state = get();
    return getSnapshot(state);
  },

  restoreSnapshot: (snapshot) => {
    const state = get();
    const currentSnapshot = getSnapshot(state);
    set({
      nodes: snapshot.nodes,
      edges: snapshot.edges,
      viewport: snapshot.viewport,
      past: [...state.past, currentSnapshot],
      future: [],
    });
  },
}));

// 辅助函数：获取快照
function getSnapshot(state: CanvasState): CanvasSnapshot {
  return {
    nodes: [...state.nodes],
    edges: [...state.edges],
    viewport: { ...state.viewport },
  };
}
