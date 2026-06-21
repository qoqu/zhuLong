// Config节点组件
import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { ConfigNodeData } from '../../../types/canvas';

export const ConfigNode = memo(({ data, selected }: NodeProps<ConfigNodeData>) => {
  const modeIcons = {
    text: '📝',
    image: '🖼️',
    video: '🎬',
    audio: '🔊',
  };

  const modeColors = {
    text: 'var(--accent)',
    image: 'var(--ok)',
    video: 'var(--warn)',
    audio: 'var(--info)',
  };

  return (
    <div
      className={`canvas-node config-node ${selected ? 'selected' : ''}`}
      style={{
        background: 'var(--bg-soft)',
        border: `2px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 'var(--radius-lg)',
        padding: 'var(--space-3)',
        minWidth: '200px',
      }}
    >
      <Handle
        type="target"
        position={Position.Left}
        style={{
          background: 'var(--fg-faint)',
          width: '6px',
          height: '6px',
        }}
      />

      <div className="config-node__header" style={{ marginBottom: 'var(--space-2)' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--space-2)',
            }}
          >
            <span>{modeIcons[data.generationMode]}</span>
            <div
              style={{
                fontSize: 'var(--text-xs)',
                color: 'var(--fg-faint)',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}
            >
              Config
            </div>
          </div>
          <span
            style={{
              padding: '1px 6px',
              borderRadius: 'var(--radius-full)',
              background: 'var(--bg-elev)',
              color: 'var(--fg-faint)',
              fontSize: 'var(--text-xs)',
            }}
          >
            {data.generationMode}
          </span>
        </div>
        <div
          style={{
            fontSize: 'var(--text-sm)',
            fontWeight: 600,
            color: 'var(--fg)',
            marginTop: 'var(--space-1)',
          }}
        >
          {data.title}
        </div>
      </div>

      <div className="config-node__content">
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 'var(--space-1)',
            fontSize: 'var(--text-xs)',
            color: 'var(--fg-faint)',
          }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
            }}
          >
            <span>Model:</span>
            <span style={{ color: 'var(--fg-soft)' }}>{data.model}</span>
          </div>

          {data.params && Object.keys(data.params).length > 0 && (
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
              }}
            >
              <span>Params:</span>
              <span style={{ color: 'var(--fg-soft)' }}>
                {Object.keys(data.params).length} settings
              </span>
            </div>
          )}

          {data.referenceIds && data.referenceIds.length > 0 && (
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
              }}
            >
              <span>References:</span>
              <span style={{ color: 'var(--fg-soft)' }}>
                {data.referenceIds.length} nodes
              </span>
            </div>
          )}
        </div>

        {data.prompt && (
          <div
            style={{
              marginTop: 'var(--space-2)',
              padding: 'var(--space-2)',
              background: 'var(--bg)',
              borderRadius: 'var(--radius-sm)',
              fontSize: 'var(--text-xs)',
              color: 'var(--fg-soft)',
              maxHeight: '40px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {data.prompt}
          </div>
        )}

        <div
          style={{
            marginTop: 'var(--space-2)',
            padding: 'var(--space-2)',
            background: modeColors[data.generationMode],
            borderRadius: 'var(--radius-sm)',
            textAlign: 'center',
            fontSize: 'var(--text-xs)',
            fontWeight: 600,
            color: 'white',
            cursor: 'pointer',
          }}
        >
          Generate {data.generationMode}
        </div>
      </div>
    </div>
  );
});

ConfigNode.displayName = 'ConfigNode';
