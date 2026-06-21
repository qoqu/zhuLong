// Plan节点组件
import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { PlanNodeData } from '../../../types/canvas';

export const PlanNode = memo(({ data, selected }: NodeProps<PlanNodeData>) => {
  return (
    <div
      className={`canvas-node plan-node ${selected ? 'selected' : ''}`}
      style={{
        background: 'var(--bg-soft)',
        border: `2px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 'var(--radius-lg)',
        padding: 'var(--space-4)',
        minWidth: '200px',
      }}
    >
      <div className="plan-node__header" style={{ marginBottom: 'var(--space-2)' }}>
        <div
          style={{
            fontSize: 'var(--text-xs)',
            color: 'var(--fg-faint)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
          }}
        >
          Plan
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

      <div className="plan-node__content">
        <div
          style={{
            fontSize: 'var(--text-sm)',
            color: 'var(--fg-soft)',
            marginBottom: 'var(--space-2)',
          }}
        >
          {data.goal}
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-2)',
            fontSize: 'var(--text-xs)',
            color: 'var(--fg-faint)',
          }}
        >
          <span>{data.steps} steps</span>
          <span>•</span>
          <span
            style={{
              padding: '1px 6px',
              borderRadius: 'var(--radius-full)',
              background:
                data.status === 'completed'
                  ? 'rgba(52, 199, 89, 0.15)'
                  : data.status === 'running'
                  ? 'rgba(10, 132, 255, 0.15)'
                  : 'var(--bg-elev)',
              color:
                data.status === 'completed'
                  ? 'var(--ok)'
                  : data.status === 'running'
                  ? 'var(--accent)'
                  : 'var(--fg-faint)',
            }}
          >
            {data.status}
          </span>
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        style={{
          background: 'var(--accent)',
          width: '8px',
          height: '8px',
        }}
      />
    </div>
  );
});

PlanNode.displayName = 'PlanNode';
