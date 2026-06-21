// 画布工具栏组件
import { useCanvasStore } from '../../stores/canvasStore';

interface CanvasToolbarProps {
  onToggleNodePanel: () => void;
  onToggleAssetPanel: () => void;
  onToggleAssistant: () => void;
  onToggleVersionPanel: () => void;
  onToggleExportPanel: () => void;
  onToggleCollaborationPanel: () => void;
  onTogglePerformancePanel: () => void;
  onToggleOfflinePanel: () => void;
  onToggleAIPanel: () => void;
  onTogglePluginPanel: () => void;
  showNodePanel: boolean;
  showAssetPanel: boolean;
  showAssistant: boolean;
  showVersionPanel: boolean;
  showExportPanel: boolean;
  showCollaborationPanel: boolean;
  showPerformancePanel: boolean;
  showOfflinePanel: boolean;
  showAIPanel: boolean;
  showPluginPanel: boolean;
}

export function CanvasToolbar({
  onToggleNodePanel,
  onToggleAssetPanel,
  onToggleAssistant,
  onToggleVersionPanel,
  onToggleExportPanel,
  onToggleCollaborationPanel,
  onTogglePerformancePanel,
  onToggleOfflinePanel,
  onToggleAIPanel,
  onTogglePluginPanel,
  showNodePanel,
  showAssetPanel,
  showAssistant,
  showVersionPanel,
  showExportPanel,
  showCollaborationPanel,
  showPerformancePanel,
  showOfflinePanel,
  showAIPanel,
  showPluginPanel,
}: CanvasToolbarProps) {
  const { undo, redo, fitView, zoomIn, zoomOut } = useCanvasStore();

  return (
    <div className="canvas-toolbar">
      {/* 节点面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showNodePanel ? 'active' : ''}`}
        onClick={onToggleNodePanel}
        title="Toggle Node Panel"
      >
        📦
      </button>

      {/* 资产面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showAssetPanel ? 'active' : ''}`}
        onClick={onToggleAssetPanel}
        title="Toggle Asset Panel"
      >
        🗂️
      </button>

      {/* 助手面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showAssistant ? 'active' : ''}`}
        onClick={onToggleAssistant}
        title="Toggle Assistant"
      >
        🤖
      </button>

      {/* 分隔符 */}
      <div
        style={{
          width: '1px',
          height: '24px',
          background: 'var(--border)',
          margin: '0 4px',
        }}
      />

      {/* 版本管理面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showVersionPanel ? 'active' : ''}`}
        onClick={onToggleVersionPanel}
        title="Toggle Version Panel"
      >
        📸
      </button>

      {/* 导出面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showExportPanel ? 'active' : ''}`}
        onClick={onToggleExportPanel}
        title="Toggle Export Panel"
      >
        📤
      </button>

      {/* 协作面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showCollaborationPanel ? 'active' : ''}`}
        onClick={onToggleCollaborationPanel}
        title="Toggle Collaboration Panel"
      >
        👥
      </button>

      {/* 性能面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showPerformancePanel ? 'active' : ''}`}
        onClick={onTogglePerformancePanel}
        title="Toggle Performance Panel"
      >
        ⚡
      </button>

      {/* 离线面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showOfflinePanel ? 'active' : ''}`}
        onClick={onToggleOfflinePanel}
        title="Toggle Offline Panel"
      >
        📶
      </button>

      {/* AI面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showAIPanel ? 'active' : ''}`}
        onClick={onToggleAIPanel}
        title="Toggle AI Panel"
      >
        ✨
      </button>

      {/* 插件面板切换 */}
      <button
        className={`canvas-toolbar__btn ${showPluginPanel ? 'active' : ''}`}
        onClick={onTogglePluginPanel}
        title="Toggle Plugin Panel"
      >
        🔌
      </button>

      {/* 分隔符 */}
      <div
        style={{
          width: '1px',
          height: '24px',
          background: 'var(--border)',
          margin: '0 4px',
        }}
      />

      {/* 撤销 */}
      <button
        className="canvas-toolbar__btn"
        onClick={undo}
        title="Undo (Ctrl+Z)"
      >
        ↩️
      </button>

      {/* 重做 */}
      <button
        className="canvas-toolbar__btn"
        onClick={redo}
        title="Redo (Ctrl+Shift+Z)"
      >
        ↪️
      </button>

      {/* 分隔符 */}
      <div
        style={{
          width: '1px',
          height: '24px',
          background: 'var(--border)',
          margin: '0 4px',
        }}
      />

      {/* 放大 */}
      <button
        className="canvas-toolbar__btn"
        onClick={zoomIn}
        title="Zoom In"
      >
        🔍+
      </button>

      {/* 缩小 */}
      <button
        className="canvas-toolbar__btn"
        onClick={zoomOut}
        title="Zoom Out"
      >
        🔍-
      </button>

      {/* 适应视图 */}
      <button
        className="canvas-toolbar__btn"
        onClick={fitView}
        title="Fit View"
      >
        📐
      </button>
    </div>
  );
}
