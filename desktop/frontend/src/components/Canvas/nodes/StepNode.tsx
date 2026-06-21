// Step节点组件
import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { StepNodeData } from '../../../types/canvas';

export const StepNode = memo(({ data, selected }: NodeProps<StepNodeData>) => {
  const statusColors: Record<string, { bg: string; color: string }> = {
    pending: { bg: 'var(--bg-elev)', color: 'var(--fg-faint)' },
    running: { bg: 'rgba(10, 132, 255, 0.15)', color: 'var(--accent)' },
    completed: { bg: 'rgba(52, 199, 89, 0.15)', color: 'var(--ok)' },
    failed: { bg: 'rgba(255, 59, 48, 0.15)', color: 'var(--err)' },
    idle: { bg: 'var(--bg-elev)', color: 'var(--fg-faint)' },
    success: { bg: 'rgba(52, 199, 89, 0.15)', color: 'var(--ok)' },
    error: { bg: 'rgba(255, 59, 48, 0.15)', color: 'var(--err)' },
  };

  const statusStyle = statusColors[data.status] || statusColors.pending;

  return (
    <div
      className={`canvas-node step-node ${selected ? 'selected' : ''}`}
      style={{
        background: 'var(--bg-soft)',
        border: `2px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 'var(--radius-lg)',
        padding: 'var(--space-3)',
        minWidth: '180px',
      }}
    >
      <Handle
        type="target"
        position={Position.Top}
        style={{
          background: 'var(--fg-faint)',
          width: '6px',
          height: '6px',
        }}
      />

      <div className="step-node__header" style={{ marginBottom: 'var(--space-2)' }}>
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
            Step {data.stepId}
          </div>
          <span
            style={{
              padding: '1px 6px',
              borderRadius: 'var(--radius-full)',
              background: statusStyle.bg,
              color: statusStyle.color,
              fontSize: 'var(--text-xs)',
            }}
          >
            {data.status}
          </span>
        </div>
      </div>

      <div className="step-node__content">
        <div
          style={{
            fontSize: 'var(--text-sm)',
            color: 'var(--fg)',
            marginBottom: 'var(--space-2)',
          }}
        >
          {data.description}
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
          <span>🔧 {data.tool}</span>
          {data.tokensUsed && (
            <>
              <span>•</span>
              <span>{data.tokensUsed} tokens</span>
            </>
          )}
          {data.duration && (
            <>
              <span>•</span>
              <span>{data.duration}</span>
            </>
          )}
        </div>

        {data.result && (
          <div
            style={{
              marginTop: 'var(--space-2)',
              padding: 'var(--space-2)',
              background: 'var(--bg)',
              borderRadius: 'var(--radius-sm)',
              fontSize: 'var(--text-xs)',
              color: 'var(--fg-soft)',
              maxHeight: '60px',
              overflow: 'hidden',
            }}
          >
            {data.result}
          </div>
        )}
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        style={{
          background: 'var(--accent)',
          width: '6px',
          height: '6px',
        }}
      />
    </div>
  );
});

StepNode.displayName = 'StepNode';
