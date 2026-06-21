// 节点面板组件
import { useCallback } from 'react';
import { CanvasNodeType } from '../../types/canvas';

interface NodePanelProps {
  onClose: () => void;
}

// 节点类型定义
const nodeTypes = [
  {
    type: CanvasNodeType.Text,
    icon: '📝',
    label: 'Text',
    description: 'Text content node',
  },
  {
    type: CanvasNodeType.Image,
    icon: '🖼️',
    label: 'Image',
    description: 'Image generation node',
  },
  {
    type: CanvasNodeType.Video,
    icon: '🎬',
    label: 'Video',
    description: 'Video generation node',
  },
  {
    type: CanvasNodeType.Audio,
    icon: '🔊',
    label: 'Audio',
    description: 'Audio generation node',
  },
  {
    type: CanvasNodeType.Config,
    icon: '⚙️',
    label: 'Config',
    description: 'Generation config node',
  },
  {
    type: CanvasNodeType.Storyboard,
    icon: '🎬',
    label: 'Storyboard',
    description: 'Storyboard editor node',
  },
];

export function NodePanel({ onClose }: NodePanelProps) {
  // 拖拽开始处理
  const onDragStart = useCallback(
    (event: React.DragEvent, nodeType: CanvasNodeType) => {
      event.dataTransfer.setData('application/reactflow', nodeType);
      event.dataTransfer.effectAllowed = 'move';
    },
    []
  );

  return (
    <div className="node-panel">
      <div className="node-panel__header">
        <div className="node-panel__title">Nodes</div>
        <button className="node-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      <div className="node-panel__items">
        {nodeTypes.map((nodeType) => (
          <div
            key={nodeType.type}
            className="node-panel__item"
            draggable
            onDragStart={(e) => onDragStart(e, nodeType.type)}
            title={nodeType.description}
          >
            <div className="node-panel__item-icon">{nodeType.icon}</div>
            <div className="node-panel__item-label">{nodeType.label}</div>
          </div>
        ))}
      </div>

      <div className="node-panel__hint">
        Drag nodes to the canvas to create them
      </div>
    </div>
  );
}
