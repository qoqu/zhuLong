// 节点扩展面板组件
import { useState, useCallback } from 'react';
import { CanvasNodeType } from '../../types/canvas';

interface PluginPanelProps {
  onClose: () => void;
}

interface Plugin {
  id: string;
  name: string;
  description: string;
  version: string;
  author: string;
  icon: string;
  nodeTypes: CanvasNodeType[];
  isEnabled: boolean;
}

export function PluginPanel({ onClose }: PluginPanelProps) {
  const [plugins, setPlugins] = useState<Plugin[]>([
    {
      id: 'builtin-text',
      name: 'Text Nodes',
      description: 'Basic text content nodes',
      version: '1.0.0',
      author: 'Zhulong',
      icon: '📝',
      nodeTypes: [CanvasNodeType.Text],
      isEnabled: true,
    },
    {
      id: 'builtin-image',
      name: 'Image Nodes',
      description: 'Image generation and editing nodes',
      version: '1.0.0',
      author: 'Zhulong',
      icon: '🖼️',
      nodeTypes: [CanvasNodeType.Image],
      isEnabled: true,
    },
    {
      id: 'builtin-video',
      name: 'Video Nodes',
      description: 'Video generation nodes',
      version: '1.0.0',
      author: 'Zhulong',
      icon: '🎬',
      nodeTypes: [CanvasNodeType.Video],
      isEnabled: true,
    },
    {
      id: 'builtin-audio',
      name: 'Audio Nodes',
      description: 'Audio generation nodes',
      version: '1.0.0',
      author: 'Zhulong',
      icon: '🔊',
      nodeTypes: [CanvasNodeType.Audio],
      isEnabled: true,
    },
    {
      id: 'builtin-storyboard',
      name: 'Storyboard Nodes',
      description: 'Storyboard editor nodes',
      version: '1.0.0',
      author: 'Zhulong',
      icon: '🎬',
      nodeTypes: [CanvasNodeType.Storyboard],
      isEnabled: true,
    },
    {
      id: 'builtin-config',
      name: 'Config Nodes',
      description: 'Generation configuration nodes',
      version: '1.0.0',
      author: 'Zhulong',
      icon: '⚙️',
      nodeTypes: [CanvasNodeType.Config],
      isEnabled: true,
    },
  ]);

  const [searchQuery, setSearchQuery] = useState('');

  // 过滤插件
  const filteredPlugins = plugins.filter(
    (plugin) =>
      plugin.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      plugin.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // 切换插件状态
  const handleTogglePlugin = useCallback((pluginId: string) => {
    setPlugins((prev) =>
      prev.map((plugin) =>
        plugin.id === pluginId
          ? { ...plugin, isEnabled: !plugin.isEnabled }
          : plugin
      )
    );
  }, []);

  // 安装插件（模拟）
  const handleInstallPlugin = useCallback(() => {
    alert('Plugin installation will be implemented with a plugin marketplace');
  }, []);

  // 卸载插件（模拟）
  const handleUninstallPlugin = useCallback((pluginId: string) => {
    if (confirm('Are you sure you want to uninstall this plugin?')) {
      setPlugins((prev) => prev.filter((plugin) => plugin.id !== pluginId));
    }
  }, []);

  return (
    <div className="plugin-panel">
      <div className="plugin-panel__header">
        <div className="plugin-panel__title">Plugins</div>
        <button className="plugin-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* 搜索 */}
      <div className="plugin-panel__search">
        <input
          className="plugin-panel__search-input"
          placeholder="Search plugins..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      {/* 安装按钮 */}
      <div className="plugin-panel__install">
        <button
          className="plugin-panel__install-btn"
          onClick={handleInstallPlugin}
        >
          📦 Install from Marketplace
        </button>
      </div>

      {/* 插件列表 */}
      <div className="plugin-panel__list">
        {filteredPlugins.length === 0 ? (
          <div className="plugin-panel__empty">
            <div className="plugin-panel__empty-icon">🔌</div>
            <div className="plugin-panel__empty-text">
              {searchQuery ? 'No plugins match your search' : 'No plugins'}
            </div>
          </div>
        ) : (
          filteredPlugins.map((plugin) => (
            <div
              key={plugin.id}
              className={`plugin-panel__item ${
                !plugin.isEnabled ? 'disabled' : ''
              }`}
            >
              <div className="plugin-panel__item-icon">{plugin.icon}</div>
              <div className="plugin-panel__item-info">
                <div className="plugin-panel__item-name">{plugin.name}</div>
                <div className="plugin-panel__item-description">
                  {plugin.description}
                </div>
                <div className="plugin-panel__item-meta">
                  <span>v{plugin.version}</span>
                  <span>•</span>
                  <span>{plugin.author}</span>
                  <span>•</span>
                  <span>{plugin.nodeTypes.length} node types</span>
                </div>
              </div>
              <div className="plugin-panel__item-actions">
                <button
                  className={`plugin-panel__toggle-btn ${
                    plugin.isEnabled ? 'active' : ''
                  }`}
                  onClick={() => handleTogglePlugin(plugin.id)}
                  title={plugin.isEnabled ? 'Disable' : 'Enable'}
                >
                  {plugin.isEnabled ? '✅' : '⬜'}
                </button>
                {!plugin.id.startsWith('builtin-') && (
                  <button
                    className="plugin-panel__uninstall-btn"
                    onClick={() => handleUninstallPlugin(plugin.id)}
                    title="Uninstall"
                  >
                    🗑️
                  </button>
                )}
              </div>
            </div>
          ))
        )}
      </div>

      {/* 插件说明 */}
      <div className="plugin-panel__section">
        <div className="plugin-panel__section-title">About Plugins</div>
        <div className="plugin-panel__info">
          <p>
            <strong>Built-in Plugins:</strong> Core node types that come with
            Zhulong.
          </p>
          <p>
            <strong>Custom Plugins:</strong> Install additional node types from
            the marketplace.
          </p>
          <p>
            <strong>Enable/Disable:</strong> Toggle plugins without uninstalling
            them.
          </p>
        </div>
      </div>

      {/* 统计信息 */}
      <div className="plugin-panel__section">
        <div className="plugin-panel__section-title">Statistics</div>
        <div className="plugin-panel__stats">
          <div className="plugin-panel__stat">
            <span className="plugin-panel__stat-label">Total Plugins:</span>
            <span className="plugin-panel__stat-value">{plugins.length}</span>
          </div>
          <div className="plugin-panel__stat">
            <span className="plugin-panel__stat-label">Enabled:</span>
            <span className="plugin-panel__stat-value">
              {plugins.filter((p) => p.isEnabled).length}
            </span>
          </div>
          <div className="plugin-panel__stat">
            <span className="plugin-panel__stat-label">Node Types:</span>
            <span className="plugin-panel__stat-value">
              {plugins.filter((p) => p.isEnabled).reduce((acc, p) => acc + p.nodeTypes.length, 0)}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
