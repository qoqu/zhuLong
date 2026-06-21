// 性能优化面板组件
import { useState, useCallback, useEffect } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';

interface PerformancePanelProps {
  onClose: () => void;
}

interface PerformanceStats {
  totalNodes: number;
  visibleNodes: number;
  totalEdges: number;
  memoryUsage: number;
  renderTime: number;
  fps: number;
}

export function PerformancePanel({ onClose }: PerformancePanelProps) {
  const [stats, setStats] = useState<PerformanceStats>({
    totalNodes: 0,
    visibleNodes: 0,
    totalEdges: 0,
    memoryUsage: 0,
    renderTime: 0,
    fps: 60,
  });
  const [isMonitoring, setIsMonitoring] = useState(false);

  const { nodes, edges } = useCanvasStore();

  // 更新统计信息
  const updateStats = useCallback(() => {
    const newStats: PerformanceStats = {
      totalNodes: nodes.length,
      visibleNodes: Math.min(nodes.length, 50), // 模拟可见节点数
      totalEdges: edges.length,
      memoryUsage: estimateMemoryUsage(nodes, edges),
      renderTime: estimateRenderTime(nodes.length),
      fps: estimateFPS(nodes.length),
    };
    setStats(newStats);
  }, [nodes, edges]);

  // 估算内存使用
  const estimateMemoryUsage = (nodes: any[], edges: any[]): number => {
    const nodeSize = 500; // 每个节点约500字节
    const edgeSize = 100; // 每条边约100字节
    return (nodes.length * nodeSize + edges.length * edgeSize) / 1024; // KB
  };

  // 估算渲染时间
  const estimateRenderTime = (nodeCount: number): number => {
    if (nodeCount < 10) return 1;
    if (nodeCount < 50) return 5;
    if (nodeCount < 100) return 10;
    if (nodeCount < 500) return 20;
    return 50;
  };

  // 估算FPS
  const estimateFPS = (nodeCount: number): number => {
    if (nodeCount < 50) return 60;
    if (nodeCount < 100) return 45;
    if (nodeCount < 200) return 30;
    if (nodeCount < 500) return 20;
    return 10;
  };

  // 监控性能
  useEffect(() => {
    if (isMonitoring) {
      const interval = setInterval(updateStats, 1000);
      return () => clearInterval(interval);
    }
  }, [isMonitoring, updateStats]);

  // 初始更新
  useEffect(() => {
    updateStats();
  }, [updateStats]);

  // 性能建议
  const getPerformanceSuggestions = (): string[] => {
    const suggestions: string[] = [];

    if (nodes.length > 100) {
      suggestions.push('Consider using virtualization for large node counts');
    }
    if (edges.length > 200) {
      suggestions.push('Consider simplifying edge connections');
    }
    if (stats.fps < 30) {
      suggestions.push('Performance is low, consider reducing node count');
    }
    if (stats.memoryUsage > 1000) {
      suggestions.push('Memory usage is high, consider cleanup');
    }

    if (suggestions.length === 0) {
      suggestions.push('Performance is good!');
    }

    return suggestions;
  };

  // 格式化内存大小
  const formatMemory = (kb: number): string => {
    if (kb < 1024) return `${kb.toFixed(1)} KB`;
    return `${(kb / 1024).toFixed(1)} MB`;
  };

  // 性能等级
  const getPerformanceGrade = (): { grade: string; color: string } => {
    if (stats.fps >= 50) return { grade: 'A', color: 'var(--ok)' };
    if (stats.fps >= 30) return { grade: 'B', color: 'var(--warn)' };
    return { grade: 'C', color: 'var(--err)' };
  };

  const performanceGrade = getPerformanceGrade();

  return (
    <div className="performance-panel">
      <div className="performance-panel__header">
        <div className="performance-panel__title">Performance</div>
        <button className="performance-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* 性能等级 */}
      <div className="performance-panel__grade">
        <div
          className="performance-panel__grade-letter"
          style={{ color: performanceGrade.color }}
        >
          {performanceGrade.grade}
        </div>
        <div className="performance-panel__grade-label">Performance Grade</div>
      </div>

      {/* 统计信息 */}
      <div className="performance-panel__section">
        <div className="performance-panel__section-title">Statistics</div>
        <div className="performance-panel__stats">
          <div className="performance-panel__stat">
            <span className="performance-panel__stat-label">Total Nodes:</span>
            <span className="performance-panel__stat-value">
              {stats.totalNodes}
            </span>
          </div>
          <div className="performance-panel__stat">
            <span className="performance-panel__stat-label">
              Visible Nodes:
            </span>
            <span className="performance-panel__stat-value">
              {stats.visibleNodes}
            </span>
          </div>
          <div className="performance-panel__stat">
            <span className="performance-panel__stat-label">Total Edges:</span>
            <span className="performance-panel__stat-value">
              {stats.totalEdges}
            </span>
          </div>
          <div className="performance-panel__stat">
            <span className="performance-panel__stat-label">Memory Usage:</span>
            <span className="performance-panel__stat-value">
              {formatMemory(stats.memoryUsage)}
            </span>
          </div>
          <div className="performance-panel__stat">
            <span className="performance-panel__stat-label">Render Time:</span>
            <span className="performance-panel__stat-value">
              {stats.renderTime}ms
            </span>
          </div>
          <div className="performance-panel__stat">
            <span className="performance-panel__stat-label">FPS:</span>
            <span
              className="performance-panel__stat-value"
              style={{ color: performanceGrade.color }}
            >
              {stats.fps}
            </span>
          </div>
        </div>
      </div>

      {/* 性能建议 */}
      <div className="performance-panel__section">
        <div className="performance-panel__section-title">Suggestions</div>
        <div className="performance-panel__suggestions">
          {getPerformanceSuggestions().map((suggestion, index) => (
            <div key={index} className="performance-panel__suggestion">
              💡 {suggestion}
            </div>
          ))}
        </div>
      </div>

      {/* 监控控制 */}
      <div className="performance-panel__section">
        <div className="performance-panel__section-title">Monitoring</div>
        <button
          className={`performance-panel__monitor-btn ${
            isMonitoring ? 'active' : ''
          }`}
          onClick={() => setIsMonitoring(!isMonitoring)}
        >
          {isMonitoring ? '⏸️ Stop Monitoring' : '▶️ Start Monitoring'}
        </button>
      </div>

      {/* 性能优化提示 */}
      <div className="performance-panel__section">
        <div className="performance-panel__section-title">
          Optimization Tips
        </div>
        <div className="performance-panel__tips">
          <div className="performance-panel__tip">
            <strong>Virtualization:</strong> Only render visible nodes for large
            canvases
          </div>
          <div className="performance-panel__tip">
            <strong>Batch Operations:</strong> Group multiple operations together
          </div>
          <div className="performance-panel__tip">
            <strong>Lazy Loading:</strong> Load node content on demand
          </div>
          <div className="performance-panel__tip">
            <strong>Debounce:</strong> Debounce rapid updates
          </div>
        </div>
      </div>
    </div>
  );
}
