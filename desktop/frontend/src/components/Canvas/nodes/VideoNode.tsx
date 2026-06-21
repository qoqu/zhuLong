// Video节点组件
import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { VideoNodeData } from '../../../types/canvas';

export const VideoNode = memo(({ data, selected }: NodeProps<VideoNodeData>) => {
  return (
    <div
      className={`canvas-node video-node ${selected ? 'selected' : ''}`}
      style={{
        background: 'var(--bg-soft)',
        border: `2px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 'var(--radius-lg)',
        padding: 'var(--space-3)',
        minWidth: '220px',
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

      <div className="video-node__header" style={{ marginBottom: 'var(--space-2)' }}>
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
            Video
          </div>
          <span
            style={{
              padding: '1px 6px',
              borderRadius: 'var(--radius-full)',
              background:
                data.status === 'success'
                  ? 'rgba(52, 199, 89, 0.15)'
                  : data.status === 'loading'
                  ? 'rgba(10, 132, 255, 0.15)'
                  : 'var(--bg-elev)',
              color:
                data.status === 'success'
                  ? 'var(--ok)'
                  : data.status === 'loading'
                  ? 'var(--accent)'
                  : 'var(--fg-faint)',
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

      <div className="video-node__content">
        {data.videoUrl ? (
          <div
            style={{
              width: '100%',
              height: '120px',
              borderRadius: 'var(--radius-sm)',
              overflow: 'hidden',
              marginBottom: 'var(--space-2)',
              background: 'var(--bg)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <video
              src={data.videoUrl}
              style={{
                width: '100%',
                height: '100%',
                objectFit: 'cover',
              }}
              muted
              loop
              playsInline
            />
          </div>
        ) : (
          <div
            style={{
              width: '100%',
              height: '120px',
              borderRadius: 'var(--radius-sm)',
              background: 'var(--bg)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginBottom: 'var(--space-2)',
              color: 'var(--fg-faint)',
              fontSize: 'var(--text-sm)',
            }}
          >
            {data.status === 'loading' ? 'Generating...' : 'No video'}
          </div>
        )}

        {data.prompt && (
          <div
            style={{
              fontSize: 'var(--text-xs)',
              color: 'var(--fg-faint)',
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
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-2)',
            marginTop: 'var(--space-2)',
            fontSize: 'var(--text-xs)',
            color: 'var(--fg-faint)',
          }}
        >
          {data.duration && <span>{data.duration}s</span>}
          {data.orientation && (
            <>
              <span>•</span>
              <span>{data.orientation}</span>
            </>
          )}
        </div>
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

VideoNode.displayName = 'VideoNode';
