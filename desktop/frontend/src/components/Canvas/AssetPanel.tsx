// 资产管理面板组件
import { useState, useCallback } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import { AssetType, type Asset } from '../../types/canvas';

interface AssetPanelProps {
  onClose: () => void;
}

export function AssetPanel({ onClose }: AssetPanelProps) {
  const [filterType, setFilterType] = useState<AssetType | 'all'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const { assets, addAsset, deleteAsset } = useCanvasStore();

  // 过滤资产
  const filteredAssets = assets.filter((asset) => {
    if (filterType !== 'all' && asset.type !== filterType) return false;
    if (
      searchQuery &&
      !asset.name.toLowerCase().includes(searchQuery.toLowerCase())
    )
      return false;
    return true;
  });

  // 创建新资产
  const handleCreateAsset = useCallback(
    (type: AssetType) => {
      const newAsset: Asset = {
        id: `asset-${Date.now()}`,
        type,
        name: `New ${type}`,
        description: '',
        content: type === AssetType.Text ? 'New text content' : undefined,
        projectId: 'default',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      addAsset(newAsset);
    },
    [addAsset]
  );

  // 删除资产
  const handleDeleteAsset = useCallback(
    (id: string) => {
      if (confirm('Are you sure you want to delete this asset?')) {
        deleteAsset(id);
      }
    },
    [deleteAsset]
  );

  // 资产类型图标
  const assetTypeIcons: Record<AssetType, string> = {
    [AssetType.Text]: '📝',
    [AssetType.Image]: '🖼️',
    [AssetType.Video]: '🎬',
    [AssetType.Audio]: '🔊',
    [AssetType.Script]: '📜',
    [AssetType.Storyboard]: '🎬',
    [AssetType.Outline]: '📋',
    [AssetType.Character]: '👤',
    [AssetType.Scene]: '🏞️',
    [AssetType.Prop]: '🎭',
    [AssetType.Reference]: '📎',
    [AssetType.Template]: '📄',
  };

  return (
    <div className="asset-panel">
      <div className="asset-panel__header">
        <div className="asset-panel__title">Assets</div>
        <button className="asset-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* 搜索和过滤 */}
      <div className="asset-panel__filters">
        <input
          className="asset-panel__search"
          placeholder="Search assets..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
        <select
          className="asset-panel__filter-select"
          value={filterType}
          onChange={(e) =>
            setFilterType(e.target.value as AssetType | 'all')
          }
        >
          <option value="all">All Types</option>
          {Object.values(AssetType).map((type) => (
            <option key={type} value={type}>
              {assetTypeIcons[type]} {type}
            </option>
          ))}
        </select>
      </div>

      {/* 创建资产按钮 */}
      <div className="asset-panel__create">
        <div className="asset-panel__create-label">Create New:</div>
        <div className="asset-panel__create-buttons">
          {[
            AssetType.Text,
            AssetType.Image,
            AssetType.Video,
            AssetType.Audio,
          ].map((type) => (
            <button
              key={type}
              className="asset-panel__create-btn"
              onClick={() => handleCreateAsset(type)}
              title={`Create ${type}`}
            >
              {assetTypeIcons[type]}
            </button>
          ))}
        </div>
      </div>

      {/* 资产列表 */}
      <div className="asset-panel__list">
        {filteredAssets.length === 0 ? (
          <div className="asset-panel__empty">
            <div className="asset-panel__empty-icon">📦</div>
            <div className="asset-panel__empty-text">
              {searchQuery || filterType !== 'all'
                ? 'No assets match your filters'
                : 'No assets yet'}
            </div>
            <div className="asset-panel__empty-hint">
              Create assets to use in your workflow
            </div>
          </div>
        ) : (
          filteredAssets.map((asset) => (
            <div key={asset.id} className="asset-panel__item">
              <div className="asset-panel__item-icon">
                {assetTypeIcons[asset.type]}
              </div>
              <div className="asset-panel__item-info">
                <div className="asset-panel__item-name">{asset.name}</div>
                <div className="asset-panel__item-type">{asset.type}</div>
              </div>
              <button
                className="asset-panel__item-delete"
                onClick={() => handleDeleteAsset(asset.id)}
                title="Delete asset"
              >
                ×
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
