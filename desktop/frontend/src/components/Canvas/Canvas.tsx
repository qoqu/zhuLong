// 主画布组件
import { useState, useCallback, useRef } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  ReactFlowProvider,
  useReactFlow,
  type Node,
  type Edge,
  type Connection,
  type NodeChange,
  type EdgeChange,
} from 'reactflow';
import 'reactflow/dist/style.css';

import { useCanvasStore } from '../../stores/canvasStore';
import { CanvasNodeType } from '../../types/canvas';
import { PlanNode } from './nodes/PlanNode';
import { StepNode } from './nodes/StepNode';
import { TextNode } from './nodes/TextNode';
import { ImageNode } from './nodes/ImageNode';
import { VideoNode } from './nodes/VideoNode';
import { ConfigNode } from './nodes/ConfigNode';
import { BoardNode } from './nodes/BoardNode';
import { CanvasToolbar } from './CanvasToolbar';
import { NodePanel } from './NodePanel';
import { AssetPanel } from './AssetPanel';
import { CanvasAssistant } from './CanvasAssistant';
import { VersionPanel } from './VersionPanel';
import { ExportPanel } from './ExportPanel';
import { CollaborationPanel } from './CollaborationPanel';
import { PerformancePanel } from './PerformancePanel';
import { OfflinePanel } from './OfflinePanel';
import { AIPanel } from './AIPanel';
import { PluginPanel } from './PluginPanel';
import { ChapterGraphPanel } from './ChapterGraphPanel';
import { ProductionPanel } from './ProductionPanel';

// 注册自定义节点类型
const nodeTypes = {
  plan: PlanNode,
  step: StepNode,
  text: TextNode,
  image: ImageNode,
  video: VideoNode,
  config: ConfigNode,
  board: BoardNode,
};

// 画布内部组件
function CanvasInner() {
  const reactFlowInstance = useReactFlow();
  const wrapperRef = useRef<HTMLDivElement>(null);

  // 面板状态
  const [showNodePanel, setShowNodePanel] = useState(false);
  const [showAssetPanel, setShowAssetPanel] = useState(false);
  const [showAssistant, setShowAssistant] = useState(false);
  const [showVersionPanel, setShowVersionPanel] = useState(false);
  const [showExportPanel, setShowExportPanel] = useState(false);
  const [showCollaborationPanel, setShowCollaborationPanel] = useState(false);
  const [showPerformancePanel, setShowPerformancePanel] = useState(false);
  const [showOfflinePanel, setShowOfflinePanel] = useState(false);
  const [showAIPanel, setShowAIPanel] = useState(false);
  const [showPluginPanel, setShowPluginPanel] = useState(false);
  const [showChapterGraphPanel, setShowChapterGraphPanel] = useState(false);
  const [showProductionPanel, setShowProductionPanel] = useState(false);

  const {
    nodes,
    edges,
    selectedNodeIds,
    addNode,
    updateNode,
    deleteNode,
    deleteNodes,
    addEdge,
    deleteEdge,
    selectNodes,
    deselectAll,
    undo,
    redo,
  } = useCanvasStore();

  // 转换为ReactFlow格式
  const flowNodes: Node[] = nodes.map((node) => ({
    id: node.id,
    type: node.type,
    position: node.position,
    data: node.data,
    selected: selectedNodeIds.includes(node.id),
  }));

  const flowEdges: Edge[] = edges.map((edge) => ({
    id: edge.id,
    source: edge.source,
    target: edge.target,
    sourceHandle: edge.sourceHandle || undefined,
    targetHandle: edge.targetHandle || undefined,
  }));

  // 节点变化处理
  const onNodesChange = useCallback(
    (changes: NodeChange[]) => {
      for (const change of changes) {
        if (change.type === 'position' && change.position) {
          updateNode(change.id, { position: change.position });
        } else if (change.type === 'select') {
          if (change.selected) {
            selectNodes([...selectedNodeIds, change.id]);
          } else {
            selectNodes(selectedNodeIds.filter((id) => id !== change.id));
          }
        } else if (change.type === 'remove') {
          deleteNode(change.id);
        }
      }
    },
    [selectedNodeIds, updateNode, selectNodes, deleteNode]
  );

  // 连线变化处理
  const onEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      for (const change of changes) {
        if (change.type === 'remove') {
          deleteEdge(change.id);
        }
      }
    },
    [deleteEdge]
  );

  // 连线创建处理
  const onConnect = useCallback(
    (connection: Connection) => {
      if (connection.source && connection.target) {
        addEdge({
          id: `edge-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
          source: connection.source,
          target: connection.target,
          sourceHandle: connection.sourceHandle || undefined,
          targetHandle: connection.targetHandle || undefined,
        });
      }
    },
    [addEdge]
  );

  // 画布点击处理
  const onPaneClick = useCallback(() => {
    deselectAll();
  }, [deselectAll]);

  // 拖拽放置处理
  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const type = event.dataTransfer.getData('application/reactflow');
      if (!type) return;

      const position = reactFlowInstance.screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const nodeId = `node-${Date.now()}-${Math.random()
        .toString(36)
        .substr(2, 9)}`;
      const nodeType = type as CanvasNodeType;

      const newNode = {
        id: nodeId,
        type: nodeType,
        position,
        size: { width: 200, height: 100 },
        data: {
          id: nodeId,
          type: nodeType,
          title: type,
          status: 'idle' as const,
        } as any,
      };

      addNode(newNode);
    },
    [reactFlowInstance, addNode]
  );

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  // 键盘快捷键
  const onKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey) {
        if (event.key === 'z') {
          event.preventDefault();
          if (event.shiftKey) {
            redo();
          } else {
            undo();
          }
        }
      }

      if (event.key === 'Delete' || event.key === 'Backspace') {
        if (selectedNodeIds.length > 0) {
          deleteNodes(selectedNodeIds);
        }
      }
    },
    [selectedNodeIds, deleteNodes, undo, redo]
  );

  return (
    <div
      ref={wrapperRef}
      className="canvas-wrapper"
      onKeyDown={onKeyDown}
      tabIndex={0}
    >
      {/* 工具栏 */}
      <CanvasToolbar
        onToggleNodePanel={() => setShowNodePanel(!showNodePanel)}
        onToggleAssetPanel={() => setShowAssetPanel(!showAssetPanel)}
        onToggleAssistant={() => setShowAssistant(!showAssistant)}
        onToggleVersionPanel={() => setShowVersionPanel(!showVersionPanel)}
        onToggleExportPanel={() => setShowExportPanel(!showExportPanel)}
        onToggleCollaborationPanel={() => setShowCollaborationPanel(!showCollaborationPanel)}
        onTogglePerformancePanel={() => setShowPerformancePanel(!showPerformancePanel)}
        onToggleOfflinePanel={() => setShowOfflinePanel(!showOfflinePanel)}
        onToggleAIPanel={() => setShowAIPanel(!showAIPanel)}
        onTogglePluginPanel={() => setShowPluginPanel(!showPluginPanel)}
        onToggleChapterGraphPanel={() => setShowChapterGraphPanel(!showChapterGraphPanel)}
        onToggleProductionPanel={() => setShowProductionPanel(!showProductionPanel)}
        showNodePanel={showNodePanel}
        showAssetPanel={showAssetPanel}
        showAssistant={showAssistant}
        showVersionPanel={showVersionPanel}
        showExportPanel={showExportPanel}
        showCollaborationPanel={showCollaborationPanel}
        showPerformancePanel={showPerformancePanel}
        showOfflinePanel={showOfflinePanel}
        showAIPanel={showAIPanel}
        showPluginPanel={showPluginPanel}
        showChapterGraphPanel={showChapterGraphPanel}
        showProductionPanel={showProductionPanel}
      />

      {/* 节点面板 */}
      {showNodePanel && (
        <NodePanel onClose={() => setShowNodePanel(false)} />
      )}

      {/* 资产面板 */}
      {showAssetPanel && (
        <AssetPanel onClose={() => setShowAssetPanel(false)} />
      )}

      {/* 画布助手 */}
      {showAssistant && (
        <CanvasAssistant onClose={() => setShowAssistant(false)} />
      )}

      {/* 版本管理面板 */}
      {showVersionPanel && (
        <VersionPanel onClose={() => setShowVersionPanel(false)} />
      )}

      {/* 导出面板 */}
      {showExportPanel && (
        <ExportPanel onClose={() => setShowExportPanel(false)} />
      )}

      {/* 协作面板 */}
      {showCollaborationPanel && (
        <CollaborationPanel onClose={() => setShowCollaborationPanel(false)} />
      )}

      {/* 性能面板 */}
      {showPerformancePanel && (
        <PerformancePanel onClose={() => setShowPerformancePanel(false)} />
      )}

      {/* 离线面板 */}
      {showOfflinePanel && (
        <OfflinePanel onClose={() => setShowOfflinePanel(false)} />
      )}

      {/* AI面板 */}
      {showAIPanel && (
        <AIPanel onClose={() => setShowAIPanel(false)} />
      )}

      {/* 插件面板 */}
      {showPluginPanel && (
        <PluginPanel onClose={() => setShowPluginPanel(false)} />
      )}

      {/* 章节事件图谱面板 */}
      {showChapterGraphPanel && (
        <ChapterGraphPanel onClose={() => setShowChapterGraphPanel(false)} />
      )}

      {/* 内容生产面板 */}
      {showProductionPanel && (
        <ProductionPanel onClose={() => setShowProductionPanel(false)} />
      )}

      {/* React Flow 画布 */}
      <ReactFlow
        nodes={flowNodes}
        edges={flowEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onPaneClick={onPaneClick}
        onDrop={onDrop}
        onDragOver={onDragOver}
        nodeTypes={nodeTypes}
        fitView
        snapToGrid
        snapGrid={[15, 15]}
        defaultEdgeOptions={{
          type: 'smoothstep',
          animated: true,
        }}
      >
        <Background />
        <Controls />
        <MiniMap />
      </ReactFlow>
    </div>
  );
}

// 主画布组件
export function Canvas() {
  return (
    <ReactFlowProvider>
      <CanvasInner />
    </ReactFlowProvider>
  );
}
