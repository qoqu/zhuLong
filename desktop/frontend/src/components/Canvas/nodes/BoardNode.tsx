// 看板节点组件（参考Hermes Kanban）
import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { BoardNodeData, BoardStatus } from '../../../types/canvas';

const statusConfig: Record<BoardStatus, { label: string; color: string; bg: string }> = {
  triage:   { label: 'Triage',   color: '#8e8e93', bg: 'rgba(142,142,147,0.15)' },
  todo:     { label: 'Todo',     color: '#0a84ff', bg: 'rgba(10,132,255,0.15)' },
  ready:    { label: 'Ready',    color: '#30d158', bg: 'rgba(48,209,88,0.15)' },
  running:  { label: 'Running',  color: '#5e5ce6', bg: 'rgba(94,92,230,0.15)' },
  blocked:  { label: 'Blocked',  color: '#ff453a', bg: 'rgba(255,69,58,0.15)' },
  review:   { label: 'Review',   color: '#ff9f0a', bg: 'rgba(255,159,10,0.15)' },
  done:     { label: 'Done',     color: '#34c759', bg: 'rgba(52,199,89,0.15)' },
  archived: { label: 'Archived', color: '#48484a', bg: 'rgba(72,72,74,0.15)' },
};

export const BoardNode = memo(({ data, selected }: NodeProps<BoardNodeData>) => {
  const cfg = statusConfig[data.boardStatus] || statusConfig.triage;

  return (
    <div
      style={{
        width: 220,
        background: 'var(--bg-elev, #1c1c1e)',
        border: `1px solid ${selected ? 'var(--accent, #0a84ff)' : 'var(--border, #38383a)'}`,
        borderRadius: 12,
        padding: 0,
        overflow: 'hidden',
        boxShadow: selected ? '0 0 0 2px rgba(10,132,255,0.3)' : 'none',
      }}
    >
      {/* 状态条 */}
      <div style={{
        height: 3,
        background: cfg.color,
      }} />

      {/* 标题 */}
      <div style={{ padding: '8px 12px', borderBottom: '1px solid var(--border, #38383a)' }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          marginBottom: 4,
        }}>
          <span style={{
            fontSize: 9,
            padding: '1px 6px',
            borderRadius: 4,
            background: cfg.bg,
            color: cfg.color,
            fontWeight: 500,
          }}>
            {cfg.label}
          </span>
          {data.priority > 0 && (
            <span style={{ fontSize: 10, color: '#ff9f0a' }}>{'★'.repeat(data.priority)}</span>
          )}
        </div>
        <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg, #fff)', lineHeight: 1.3 }}>
          {data.title}
        </div>
      </div>

      {/* 内容 */}
      <div style={{ padding: '8px 12px', fontSize: 11, color: 'var(--fg-muted, #98989d)' }}>
        {data.assignee && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginBottom: 4 }}>
            <span>👤</span>
            <span>{data.assignee}</span>
          </div>
        )}
        {data.dependsOn.length > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginBottom: 4 }}>
            <span>🔗</span>
            <span>{data.dependsOn.length} dependencies</span>
          </div>
        )}
        {data.consecutiveFails > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <span>⚠️</span>
            <span style={{ color: '#ff453a' }}>{data.consecutiveFails} failures</span>
          </div>
        )}
      </div>

      <Handle type="target" position={Position.Top} style={{ background: cfg.color, width: 8, height: 8 }} />
      <Handle type="source" position={Position.Bottom} style={{ background: cfg.color, width: 8, height: 8 }} />
    </div>
  );
});
