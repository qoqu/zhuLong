// 五层内容生产架构面板（参考Toonflow）
import { useState, useCallback } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import type { 
  ProductionPipeline,
} from '../../types/canvas';
import { ProductionLayer } from '../../types/canvas';

interface ProductionPanelProps {
  onClose: () => void;
}

// 生产层级显示信息
const LAYER_INFO: Record<ProductionLayer, { label: string; icon: string; color: string }> = {
  [ProductionLayer.Import]: { label: 'Import', icon: '📥', color: '#0a84ff' },
  [ProductionLayer.Parse]: { label: 'Parse', icon: '🔍', color: '#5e5ce6' },
  [ProductionLayer.Character]: { label: 'Character', icon: '👤', color: '#bf5af2' },
  [ProductionLayer.Script]: { label: 'Script', icon: '📝', color: '#ff6482' },
  [ProductionLayer.Storyboard]: { label: 'Storyboard', icon: '🎬', color: '#ff9f0a' },
  [ProductionLayer.Video]: { label: 'Video', icon: '🎥', color: '#34c759' },
};

const LAYER_ORDER: ProductionLayer[] = [
  ProductionLayer.Import,
  ProductionLayer.Parse,
  ProductionLayer.Character,
  ProductionLayer.Script,
  ProductionLayer.Storyboard,
  ProductionLayer.Video,
];

export function ProductionPanel({ onClose }: ProductionPanelProps) {
  const [activePipelineId, setActivePipelineId] = useState<string | null>(null);
  const [showCreatePipeline, setShowCreatePipeline] = useState(false);
  const [pipelineName, setPipelineName] = useState('');
  const [importSource, setImportSource] = useState('');
  const [importFormat, setImportFormat] = useState<'txt' | 'docx' | 'pdf' | 'epub'>('txt');

  const {
    productionPipelines,
    addProductionPipeline,
    updateProductionPipeline,
    deleteProductionPipeline,
    setProductionLayerStatus,
  } = useCanvasStore();

  // 当前管道
  const currentPipeline = productionPipelines.find(p => p.id === activePipelineId);

  // 创建新管道
  const handleCreatePipeline = useCallback(() => {
    if (!pipelineName.trim()) return;

    const layers: ProductionLayer[] = LAYER_ORDER;
    const progress: Record<string, number> = {};
    layers.forEach(l => { progress[l] = 0; });

    const newPipeline: ProductionPipeline = {
      id: `pipeline-${Date.now()}`,
      name: pipelineName.trim(),
      layers,
      currentLayer: ProductionLayer.Import,
      status: 'idle',
      progress,
      config: {
        import: {
          sourceType: 'file',
          source: importSource,
          format: importFormat,
        },
      },
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    addProductionPipeline(newPipeline);
    setActivePipelineId(newPipeline.id);
    setShowCreatePipeline(false);
    setPipelineName('');
    setImportSource('');
  }, [pipelineName, importSource, importFormat, addProductionPipeline]);

  // 模拟运行一层
  const handleRunLayer = useCallback((layer: ProductionLayer) => {
    if (!currentPipeline) return;

    setProductionLayerStatus(currentPipeline.id, layer, 'running', 0);

    // 模拟进度
    let progress = 0;
    const interval = setInterval(() => {
      progress += Math.random() * 20 + 5;
      if (progress >= 100) {
        progress = 100;
        clearInterval(interval);
        setProductionLayerStatus(currentPipeline.id, layer, 'completed', 100);

        // 自动进入下一层
        const layerIndex = LAYER_ORDER.indexOf(layer);
        if (layerIndex < LAYER_ORDER.length - 1) {
          const nextLayer = LAYER_ORDER[layerIndex + 1];
          updateProductionPipeline(currentPipeline.id, { currentLayer: nextLayer });
        } else {
          updateProductionPipeline(currentPipeline.id, { status: 'completed' as any });
        }
      }
      setProductionLayerStatus(currentPipeline.id, layer, 'running', Math.min(Math.round(progress), 100));
    }, 300);
  }, [currentPipeline, setProductionLayerStatus, updateProductionPipeline]);

  // 渲染管道列表
  const renderPipelines = () => (
    <div className="production-panel__list">
      {productionPipelines.map(pipeline => (
        <div
          key={pipeline.id}
          className={`production-panel__item ${pipeline.id === activePipelineId ? 'selected' : ''}`}
          onClick={() => setActivePipelineId(pipeline.id)}
        >
          <div className="production-panel__item-title">{pipeline.name}</div>
          <div className="production-panel__item-meta">
            Status: {pipeline.status} · Layer: {pipeline.currentLayer}
          </div>
          <div className="production-panel__progress-bar">
            <div
              className="production-panel__progress-fill"
              style={{
                width: `${Math.round(
                  LAYER_ORDER.reduce((sum, layer) => sum + (pipeline.progress[layer] || 0), 0) / LAYER_ORDER.length
                )}%`,
              }}
            />
          </div>
        </div>
      ))}
      {showCreatePipeline ? (
        <div className="production-panel__create-form">
          <input
            className="production-panel__input"
            placeholder="Pipeline name..."
            value={pipelineName}
            onChange={(e) => setPipelineName(e.target.value)}
          />
          <select
            className="production-panel__input"
            value={importFormat}
            onChange={(e) => setImportFormat(e.target.value as any)}
          >
            <option value="txt">Plain Text</option>
            <option value="docx">Word (DOCX)</option>
            <option value="pdf">PDF</option>
            <option value="epub">EPUB</option>
          </select>
          <input
            className="production-panel__input"
            placeholder="File path or URL..."
            value={importSource}
            onChange={(e) => setImportSource(e.target.value)}
          />
          <div className="production-panel__form-actions">
            <button className="production-panel__btn" onClick={handleCreatePipeline}>Create</button>
            <button className="production-panel__btn production-panel__btn--secondary" onClick={() => setShowCreatePipeline(false)}>Cancel</button>
          </div>
        </div>
      ) : (
        <button className="production-panel__btn production-panel__btn--add" onClick={() => setShowCreatePipeline(true)}>
          + New Pipeline
        </button>
      )}
    </div>
  );

  // 渲染管道详情
  const renderPipelineDetail = () => {
    if (!currentPipeline) {
      return <div className="production-panel__empty">Select a pipeline to view details or create a new one</div>;
    }

    return (
      <div className="production-panel__detail">
        <h3 className="production-panel__detail-title">{currentPipeline.name}</h3>
        <div className="production-panel__detail-status">
          Status: <strong>{currentPipeline.status}</strong> · 
          Current Layer: <strong>{currentPipeline.currentLayer}</strong>
        </div>

        <div className="production-panel__layers">
          {LAYER_ORDER.map(layer => {
            const info = LAYER_INFO[layer];
            const layerProgress = currentPipeline.progress[layer] || 0;
            const isActive = currentPipeline.currentLayer === layer;
            const isCompleted = layerProgress >= 100;
            const isRunning = isActive && currentPipeline.status === 'running';

            return (
              <div
                key={layer}
                className={`production-panel__layer ${isActive ? 'active' : ''} ${isCompleted ? 'completed' : ''}`}
              >
                <div className="production-panel__layer-header">
                  <span className="production-panel__layer-icon" style={{ color: info.color }}>
                    {info.icon}
                  </span>
                  <div className="production-panel__layer-info">
                    <div className="production-panel__layer-name">{info.label}</div>
                    <div className="production-panel__layer-status">
                      {isCompleted ? '✅ Completed' : isRunning ? '🔄 Running...' : isActive ? '⏳ Ready' : '⏸️ Pending'}
                    </div>
                  </div>
                  <button
                    className="production-panel__btn production-panel__btn--run"
                    onClick={() => handleRunLayer(layer)}
                    disabled={!isActive || isCompleted || currentPipeline.status === 'running'}
                  >
                    {isCompleted ? 'Redo' : isRunning ? '...' : 'Run'}
                  </button>
                </div>
                <div className="production-panel__progress-bar">
                  <div
                    className="production-panel__progress-fill"
                    style={{
                      width: `${layerProgress}%`,
                      background: isCompleted ? '#34c759' : info.color,
                      transition: 'width 0.3s ease',
                    }}
                  />
                </div>
                <div className="production-panel__layer-progress">{layerProgress}%</div>
              </div>
            );
          })}
        </div>

        <div className="production-panel__actions">
          <button
            className="production-panel__btn production-panel__btn--run"
            onClick={() => {
              const firstIncomplete = LAYER_ORDER.find(
                l => (currentPipeline.progress[l] || 0) < 100
              );
              if (firstIncomplete) handleRunLayer(firstIncomplete);
            }}
            disabled={currentPipeline.status === 'running'}
          >
            ▶ Run All
          </button>
          <button
            className="production-panel__btn production-panel__btn--danger"
            onClick={() => deleteProductionPipeline(currentPipeline.id)}
          >
            Delete
          </button>
        </div>
      </div>
    );
  };

  return (
    <div className="production-panel">
      <div className="production-panel__header">
        <div className="production-panel__title">Content Production</div>
        <button className="production-panel__close" onClick={onClose}>×</button>
      </div>

      <div className="production-panel__content">
        <div className="production-panel__sidebar">
          {renderPipelines()}
        </div>
        <div className="production-panel__main">
          {renderPipelineDetail()}
        </div>
      </div>

      <div className="production-panel__footer">
        {productionPipelines.length > 0 && (
          <span>{productionPipelines.length} pipelines</span>
        )}
      </div>
    </div>
  );
}
