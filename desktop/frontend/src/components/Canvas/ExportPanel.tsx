// 导出与分享面板组件
import { useState, useCallback } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';

interface ExportPanelProps {
  onClose: () => void;
}

export function ExportPanel({ onClose }: ExportPanelProps) {
  const [exportFormat, setExportFormat] = useState<'json' | 'png' | 'svg'>('json');
  const [includeAssets, setIncludeAssets] = useState(false);
  const [shareLink, setShareLink] = useState<string | null>(null);

  const { nodes, edges, assets, getSnapshot } = useCanvasStore();

  // 导出为JSON
  const handleExportJSON = useCallback(() => {
    const snapshot = getSnapshot();
    const exportData = {
      version: '1.0',
      exportedAt: new Date().toISOString(),
      canvas: snapshot,
      assets: includeAssets ? assets : [],
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `zhulong-canvas-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [getSnapshot, assets, includeAssets]);

  // 导出为PNG（模拟）
  const handleExportPNG = useCallback(() => {
    // 在实际实现中，这里会使用html2canvas或类似库
    alert('PNG export will be implemented with html2canvas library');
  }, []);

  // 导出为SVG（模拟）
  const handleExportSVG = useCallback(() => {
    // 在实际实现中，这里会将画布转换为SVG
    alert('SVG export will be implemented');
  }, []);

  // 生成分享链接（模拟）
  const handleGenerateShareLink = useCallback(() => {
    // 在实际实现中，这里会上传到服务器并生成链接
    const mockLink = `https://zhulong.app/shared/${Date.now()}`;
    setShareLink(mockLink);

    // 复制到剪贴板
    navigator.clipboard.writeText(mockLink).then(() => {
      alert('Share link copied to clipboard!');
    });
  }, []);

  // 导入JSON
  const handleImportJSON = useCallback(() => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = (event) => {
        try {
          const data = JSON.parse(event.target?.result as string);
          if (data.canvas) {
            // 在实际实现中，这里会恢复画布状态
            alert('Import functionality will be implemented');
          }
        } catch (error) {
          alert('Invalid JSON file');
        }
      };
      reader.readAsText(file);
    };
    input.click();
  }, []);

  // 导出选项
  const exportOptions = [
    {
      format: 'json' as const,
      icon: '📄',
      label: 'JSON',
      description: 'Full canvas data with nodes, edges, and optional assets',
    },
    {
      format: 'png' as const,
      icon: '🖼️',
      label: 'PNG',
      description: 'Image snapshot of the canvas',
    },
    {
      format: 'svg' as const,
      icon: '🎨',
      label: 'SVG',
      description: 'Vector graphics of the canvas',
    },
  ];

  return (
    <div className="export-panel">
      <div className="export-panel__header">
        <div className="export-panel__title">Export & Share</div>
        <button className="export-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* 导出格式选择 */}
      <div className="export-panel__section">
        <div className="export-panel__section-title">Export Format</div>
        <div className="export-panel__format-options">
          {exportOptions.map((option) => (
            <div
              key={option.format}
              className={`export-panel__format-option ${
                exportFormat === option.format ? 'active' : ''
              }`}
              onClick={() => setExportFormat(option.format)}
            >
              <div className="export-panel__format-icon">{option.icon}</div>
              <div className="export-panel__format-info">
                <div className="export-panel__format-label">{option.label}</div>
                <div className="export-panel__format-description">
                  {option.description}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* 导出选项 */}
      <div className="export-panel__section">
        <div className="export-panel__section-title">Options</div>
        <label className="export-panel__checkbox">
          <input
            type="checkbox"
            checked={includeAssets}
            onChange={(e) => setIncludeAssets(e.target.checked)}
          />
          <span>Include assets</span>
        </label>
      </div>

      {/* 导出按钮 */}
      <div className="export-panel__section">
        <div className="export-panel__section-title">Export</div>
        <div className="export-panel__actions">
          {exportFormat === 'json' && (
            <button
              className="export-panel__btn export-panel__btn--primary"
              onClick={handleExportJSON}
            >
              Export as JSON
            </button>
          )}
          {exportFormat === 'png' && (
            <button
              className="export-panel__btn export-panel__btn--primary"
              onClick={handleExportPNG}
            >
              Export as PNG
            </button>
          )}
          {exportFormat === 'svg' && (
            <button
              className="export-panel__btn export-panel__btn--primary"
              onClick={handleExportSVG}
            >
              Export as SVG
            </button>
          )}
          <button
            className="export-panel__btn export-panel__btn--secondary"
            onClick={handleImportJSON}
          >
            Import from JSON
          </button>
        </div>
      </div>

      {/* 分享 */}
      <div className="export-panel__section">
        <div className="export-panel__section-title">Share</div>
        <button
          className="export-panel__btn export-panel__btn--primary"
          onClick={handleGenerateShareLink}
        >
          Generate Share Link
        </button>
        {shareLink && (
          <div className="export-panel__share-link">
            <input
              className="export-panel__share-input"
              value={shareLink}
              readOnly
            />
            <button
              className="export-panel__copy-btn"
              onClick={() => {
                navigator.clipboard.writeText(shareLink);
                alert('Copied!');
              }}
            >
              📋
            </button>
          </div>
        )}
      </div>

      {/* 统计信息 */}
      <div className="export-panel__section">
        <div className="export-panel__section-title">Statistics</div>
        <div className="export-panel__stats">
          <div className="export-panel__stat">
            <span className="export-panel__stat-label">Nodes:</span>
            <span className="export-panel__stat-value">{nodes.length}</span>
          </div>
          <div className="export-panel__stat">
            <span className="export-panel__stat-label">Edges:</span>
            <span className="export-panel__stat-value">{edges.length}</span>
          </div>
          <div className="export-panel__stat">
            <span className="export-panel__stat-label">Assets:</span>
            <span className="export-panel__stat-value">{assets.length}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
