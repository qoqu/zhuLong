// Text节点组件
import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { TextNodeData } from '../../../types/canvas';

export const TextNode = memo(({ data, selected }: NodeProps<TextNodeData>) => {
  return (
    <div
      className={`canvas-node text-node ${selected ? 'selected' : ''}`}
      style={{
        background: 'var(--bg-soft)',
        border: `2px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 'var(--radius-lg)',
        padding: 'var(--space-3)',
        minWidth: '200px',
        maxWidth: '300px',
      }}
    >
      <div className="text-node__header" style={{ marginBottom: 'var(--space-2)' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div
            style={{
              fontSize: 'var(--text-xs)',
              color: 'var(--fg-faint)',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            Text
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
            {data.status}
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

      <div className="text-node__content">
        <div
          style={{
            fontSize: 'var(--text-sm)',
            color: 'var(--fg-soft)',
            lineHeight: 1.5,
            maxHeight: '100px',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {data.content || 'Empty text node'}
        </div>

        {data.wordCount && (
          <div
            style={{
              marginTop: 'var(--space-2)',
              fontSize: 'var(--text-xs)',
              color: 'var(--fg-faint)',
            }}
          >
            {data.wordCount} words
          </div>
        )}
      </div>

      <Handle
        type="source"
        position={Position.Right}
        style={{
          background: 'var(--accent)',
          width: '6px',
          height: '6px',
        }}
      />
    </div>
  );
});

TextNode.displayName = 'TextNode';
