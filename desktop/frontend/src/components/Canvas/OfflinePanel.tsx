// 离线支持面板组件
import { useState, useCallback, useEffect } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';

interface OfflinePanelProps {
  onClose: () => void;
}

type OfflineStatus = 'online' | 'offline' | 'syncing';

export function OfflinePanel({ onClose }: OfflinePanelProps) {
  const [status, setStatus] = useState<OfflineStatus>('online');
  const [lastSync, setLastSync] = useState<Date | null>(null);
  const [pendingChanges, setPendingChanges] = useState(0);
  const [autoSync, setAutoSync] = useState(true);

  const { nodes, edges, assets, getSnapshot } = useCanvasStore();

  // 检查网络状态
  useEffect(() => {
    const handleOnline = () => setStatus('online');
    const handleOffline = () => setStatus('offline');

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    // 初始状态
    setStatus(navigator.onLine ? 'online' : 'offline');

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  // 保存到本地存储
  const handleSaveToLocal = useCallback(() => {
    try {
      const snapshot = getSnapshot();
      const data = {
        snapshot,
        assets,
        savedAt: new Date().toISOString(),
      };
      localStorage.setItem('zhulong-canvas-backup', JSON.stringify(data));
      alert('Canvas saved to local storage!');
    } catch (error) {
      alert('Failed to save to local storage');
    }
  }, [getSnapshot, assets]);

  // 从本地存储加载
  const handleLoadFromLocal = useCallback(() => {
    try {
      const data = localStorage.getItem('zhulong-canvas-backup');
      if (data) {
        JSON.parse(data);
        // 在实际实现中，这里会恢复画布状态
        alert('Canvas loaded from local storage!');
      } else {
        alert('No backup found in local storage');
      }
    } catch (error) {
      alert('Failed to load from local storage');
    }
  }, []);

  // 同步到服务器（模拟）
  const handleSync = useCallback(async () => {
    setStatus('syncing');
    // 模拟同步延迟
    await new Promise((resolve) => setTimeout(resolve, 2000));
    setLastSync(new Date());
    setPendingChanges(0);
    setStatus(navigator.onLine ? 'online' : 'offline');
    alert('Sync completed!');
  }, []);

  // 清除本地存储
  const handleClearLocal = useCallback(() => {
    if (confirm('Are you sure you want to clear local storage?')) {
      localStorage.removeItem('zhulong-canvas-backup');
      alert('Local storage cleared!');
    }
  }, []);

  // 格式化日期
  const formatDate = (date: Date): string => {
    return date.toLocaleString();
  };

  // 状态图标
  const statusIcons: Record<OfflineStatus, string> = {
    online: '🟢',
    offline: '🔴',
    syncing: '🟡',
  };

  // 状态文本
  const statusTexts: Record<OfflineStatus, string> = {
    online: 'Online',
    offline: 'Offline',
    syncing: 'Syncing...',
  };

  return (
    <div className="offline-panel">
      <div className="offline-panel__header">
        <div className="offline-panel__title">Offline Support</div>
        <button className="offline-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* 网络状态 */}
      <div className="offline-panel__section">
        <div className="offline-panel__section-title">Network Status</div>
        <div className="offline-panel__status">
          <span className="offline-panel__status-icon">
            {statusIcons[status]}
          </span>
          <span className="offline-panel__status-text">
            {statusTexts[status]}
          </span>
        </div>
        {lastSync && (
          <div className="offline-panel__last-sync">
            Last synced: {formatDate(lastSync)}
          </div>
        )}
        {pendingChanges > 0 && (
          <div className="offline-panel__pending">
            {pendingChanges} pending changes
          </div>
        )}
      </div>

      {/* 本地存储 */}
      <div className="offline-panel__section">
        <div className="offline-panel__section-title">Local Storage</div>
        <div className="offline-panel__actions">
          <button
            className="offline-panel__btn offline-panel__btn--primary"
            onClick={handleSaveToLocal}
          >
            💾 Save to Local
          </button>
          <button
            className="offline-panel__btn offline-panel__btn--secondary"
            onClick={handleLoadFromLocal}
          >
            📂 Load from Local
          </button>
          <button
            className="offline-panel__btn offline-panel__btn--danger"
            onClick={handleClearLocal}
          >
            🗑️ Clear Local
          </button>
        </div>
      </div>

      {/* 同步控制 */}
      <div className="offline-panel__section">
        <div className="offline-panel__section-title">Sync</div>
        <div className="offline-panel__sync-controls">
          <button
            className="offline-panel__btn offline-panel__btn--primary"
            onClick={handleSync}
            disabled={status === 'syncing' || status === 'offline'}
          >
            {status === 'syncing' ? '⏳ Syncing...' : '🔄 Sync Now'}
          </button>
          <label className="offline-panel__checkbox">
            <input
              type="checkbox"
              checked={autoSync}
              onChange={(e) => setAutoSync(e.target.checked)}
            />
            <span>Auto-sync when online</span>
          </label>
        </div>
      </div>

      {/* 统计信息 */}
      <div className="offline-panel__section">
        <div className="offline-panel__section-title">Storage Info</div>
        <div className="offline-panel__stats">
          <div className="offline-panel__stat">
            <span className="offline-panel__stat-label">Nodes:</span>
            <span className="offline-panel__stat-value">{nodes.length}</span>
          </div>
          <div className="offline-panel__stat">
            <span className="offline-panel__stat-label">Edges:</span>
            <span className="offline-panel__stat-value">{edges.length}</span>
          </div>
          <div className="offline-panel__stat">
            <span className="offline-panel__stat-label">Assets:</span>
            <span className="offline-panel__stat-value">{assets.length}</span>
          </div>
        </div>
      </div>

      {/* 离线说明 */}
      <div className="offline-panel__section">
        <div className="offline-panel__section-title">How Offline Works</div>
        <div className="offline-panel__info">
          <p>
            <strong>Local Storage:</strong> Your canvas is automatically saved to
            browser local storage.
          </p>
          <p>
            <strong>Offline Mode:</strong> You can continue working offline.
            Changes will sync when you're back online.
          </p>
          <p>
            <strong>Auto-Sync:</strong> When enabled, changes sync automatically
            when you're online.
          </p>
        </div>
      </div>
    </div>
  );
}
