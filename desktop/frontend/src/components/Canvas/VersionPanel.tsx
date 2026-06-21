// 版本管理面板组件
import { useState, useCallback } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import type { CanvasSnapshot } from '../../types/canvas';

interface VersionPanelProps {
  onClose: () => void;
}

interface Version {
  id: string;
  name: string;
  description?: string;
  snapshot: CanvasSnapshot;
  createdAt: string;
}

export function VersionPanel({ onClose }: VersionPanelProps) {
  const [versions, setVersions] = useState<Version[]>([]);
  const [newVersionName, setNewVersionName] = useState('');
  const [newVersionDescription, setNewVersionDescription] = useState('');

  const { getSnapshot, restoreSnapshot } = useCanvasStore();

  // 创建新版本
  const handleCreateVersion = useCallback(() => {
    if (!newVersionName.trim()) return;

    const snapshot = getSnapshot();
    const newVersion: Version = {
      id: `version-${Date.now()}`,
      name: newVersionName,
      description: newVersionDescription,
      snapshot,
      createdAt: new Date().toISOString(),
    };

    setVersions((prev) => [newVersion, ...prev]);
    setNewVersionName('');
    setNewVersionDescription('');
  }, [newVersionName, newVersionDescription, getSnapshot]);

  // 恢复版本
  const handleRestoreVersion = useCallback(
    (version: Version) => {
      if (
        confirm(
          `Are you sure you want to restore version "${version.name}"? Current changes will be saved as a new version.`
        )
      ) {
        // 保存当前状态为新版本
        const currentSnapshot = getSnapshot();
        const autoSaveVersion: Version = {
          id: `version-${Date.now()}`,
          name: `Auto-save before restoring "${version.name}"`,
          snapshot: currentSnapshot,
          createdAt: new Date().toISOString(),
        };
        setVersions((prev) => [autoSaveVersion, ...prev]);

        // 恢复选中的版本
        restoreSnapshot(version.snapshot);
      }
    },
    [getSnapshot, restoreSnapshot]
  );

  // 删除版本
  const handleDeleteVersion = useCallback(
    (versionId: string) => {
      if (confirm('Are you sure you want to delete this version?')) {
        setVersions((prev) => prev.filter((v) => v.id !== versionId));
      }
    },
    []
  );

  // 格式化日期
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  // 计算快照差异
  const getSnapshotStats = (snapshot: CanvasSnapshot) => {
    return {
      nodes: snapshot.nodes.length,
      edges: snapshot.edges.length,
    };
  };

  return (
    <div className="version-panel">
      <div className="version-panel__header">
        <div className="version-panel__title">Versions</div>
        <button className="version-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* 创建新版本 */}
      <div className="version-panel__create">
        <div className="version-panel__create-title">Create New Version</div>
        <input
          className="version-panel__input"
          placeholder="Version name"
          value={newVersionName}
          onChange={(e) => setNewVersionName(e.target.value)}
        />
        <textarea
          className="version-panel__textarea"
          placeholder="Description (optional)"
          value={newVersionDescription}
          onChange={(e) => setNewVersionDescription(e.target.value)}
          rows={2}
        />
        <button
          className="version-panel__create-btn"
          onClick={handleCreateVersion}
          disabled={!newVersionName.trim()}
        >
          Create Version
        </button>
      </div>

      {/* 版本列表 */}
      <div className="version-panel__list">
        {versions.length === 0 ? (
          <div className="version-panel__empty">
            <div className="version-panel__empty-icon">📸</div>
            <div className="version-panel__empty-text">No versions yet</div>
            <div className="version-panel__empty-hint">
              Create versions to save snapshots of your canvas
            </div>
          </div>
        ) : (
          versions.map((version) => {
            const stats = getSnapshotStats(version.snapshot);
            return (
              <div key={version.id} className="version-panel__item">
                <div className="version-panel__item-header">
                  <div className="version-panel__item-name">{version.name}</div>
                  <div className="version-panel__item-actions">
                    <button
                      className="version-panel__item-btn"
                      onClick={() => handleRestoreVersion(version)}
                      title="Restore this version"
                    >
                      ↩️
                    </button>
                    <button
                      className="version-panel__item-btn version-panel__item-btn--delete"
                      onClick={() => handleDeleteVersion(version.id)}
                      title="Delete this version"
                    >
                      🗑️
                    </button>
                  </div>
                </div>
                {version.description && (
                  <div className="version-panel__item-description">
                    {version.description}
                  </div>
                )}
                <div className="version-panel__item-meta">
                  <span>{stats.nodes} nodes</span>
                  <span>•</span>
                  <span>{stats.edges} edges</span>
                  <span>•</span>
                  <span>{formatDate(version.createdAt)}</span>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
