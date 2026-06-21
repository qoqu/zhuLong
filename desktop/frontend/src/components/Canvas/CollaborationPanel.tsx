// 协作功能面板组件
import { useState, useCallback } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';

interface CollaborationPanelProps {
  onClose: () => void;
}

interface Collaborator {
  id: string;
  name: string;
  role: 'owner' | 'editor' | 'viewer';
  color: string;
  isOnline: boolean;
  cursor?: { x: number; y: number };
}

export function CollaborationPanel({ onClose }: CollaborationPanelProps) {
  const [collaborators, setCollaborators] = useState<Collaborator[]>([
    {
      id: 'user-1',
      name: 'You',
      role: 'owner',
      color: '#0a84ff',
      isOnline: true,
    },
  ]);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<'editor' | 'viewer'>('editor');

  const { nodes, edges } = useCanvasStore();

  // 邀请协作者（模拟）
  const handleInvite = useCallback(() => {
    if (!inviteEmail.trim()) return;

    const newCollaborator: Collaborator = {
      id: `user-${Date.now()}`,
      name: inviteEmail.split('@')[0],
      role: inviteRole,
      color: getRandomColor(),
      isOnline: false,
    };

    setCollaborators((prev) => [...prev, newCollaborator]);
    setInviteEmail('');
    alert(`Invitation sent to ${inviteEmail}`);
  }, [inviteEmail, inviteRole]);

  // 移除协作者
  const handleRemoveCollaborator = useCallback((id: string) => {
    if (confirm('Are you sure you want to remove this collaborator?')) {
      setCollaborators((prev) => prev.filter((c) => c.id !== id));
    }
  }, []);

  // 更新协作者角色
  const handleUpdateRole = useCallback(
    (id: string, role: 'editor' | 'viewer') => {
      setCollaborators((prev) =>
        prev.map((c) => (c.id === id ? { ...c, role } : c))
      );
    },
    []
  );

  // 生成随机颜色
  function getRandomColor(): string {
    const colors = [
      '#ff6b6b',
      '#51cf66',
      '#339af0',
      '#ffd43b',
      '#cc5de8',
      '#20c997',
      '#ff922b',
    ];
    return colors[Math.floor(Math.random() * colors.length)];
  }

  // 角色图标
  const roleIcons: Record<string, string> = {
    owner: '👑',
    editor: '✏️',
    viewer: '👁️',
  };

  return (
    <div className="collaboration-panel">
      <div className="collaboration-panel__header">
        <div className="collaboration-panel__title">Collaboration</div>
        <button className="collaboration-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* 邀请协作者 */}
      <div className="collaboration-panel__section">
        <div className="collaboration-panel__section-title">
          Invite Collaborator
        </div>
        <div className="collaboration-panel__invite-form">
          <input
            className="collaboration-panel__input"
            placeholder="Email address"
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.target.value)}
            type="email"
          />
          <select
            className="collaboration-panel__select"
            value={inviteRole}
            onChange={(e) =>
              setInviteRole(e.target.value as 'editor' | 'viewer')
            }
          >
            <option value="editor">Editor</option>
            <option value="viewer">Viewer</option>
          </select>
          <button
            className="collaboration-panel__invite-btn"
            onClick={handleInvite}
            disabled={!inviteEmail.trim()}
          >
            Invite
          </button>
        </div>
      </div>

      {/* 协作者列表 */}
      <div className="collaboration-panel__section">
        <div className="collaboration-panel__section-title">
          Collaborators ({collaborators.length})
        </div>
        <div className="collaboration-panel__list">
          {collaborators.map((collaborator) => (
            <div
              key={collaborator.id}
              className="collaboration-panel__item"
            >
              <div
                className="collaboration-panel__avatar"
                style={{ background: collaborator.color }}
              >
                {collaborator.name.charAt(0).toUpperCase()}
              </div>
              <div className="collaboration-panel__info">
                <div className="collaboration-panel__name">
                  {collaborator.name}
                  {collaborator.role === 'owner' && (
                    <span className="collaboration-panel__owner-badge">
                      (You)
                    </span>
                  )}
                </div>
                <div className="collaboration-panel__role">
                  {roleIcons[collaborator.role]} {collaborator.role}
                </div>
              </div>
              <div className="collaboration-panel__status">
                <div
                  className={`collaboration-panel__status-dot ${
                    collaborator.isOnline ? 'online' : 'offline'
                  }`}
                />
                <span className="collaboration-panel__status-text">
                  {collaborator.isOnline ? 'Online' : 'Offline'}
                </span>
              </div>
              {collaborator.role !== 'owner' && (
                <div className="collaboration-panel__actions">
                  <select
                    className="collaboration-panel__role-select"
                    value={collaborator.role}
                    onChange={(e) =>
                      handleUpdateRole(
                        collaborator.id,
                        e.target.value as 'editor' | 'viewer'
                      )
                    }
                  >
                    <option value="editor">Editor</option>
                    <option value="viewer">Viewer</option>
                  </select>
                  <button
                    className="collaboration-panel__remove-btn"
                    onClick={() =>
                      handleRemoveCollaborator(collaborator.id)
                    }
                    title="Remove collaborator"
                  >
                    ×
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* 协作统计 */}
      <div className="collaboration-panel__section">
        <div className="collaboration-panel__section-title">
          Collaboration Stats
        </div>
        <div className="collaboration-panel__stats">
          <div className="collaboration-panel__stat">
            <span className="collaboration-panel__stat-label">
              Total Collaborators:
            </span>
            <span className="collaboration-panel__stat-value">
              {collaborators.length}
            </span>
          </div>
          <div className="collaboration-panel__stat">
            <span className="collaboration-panel__stat-label">
              Online Now:
            </span>
            <span className="collaboration-panel__stat-value">
              {collaborators.filter((c) => c.isOnline).length}
            </span>
          </div>
          <div className="collaboration-panel__stat">
            <span className="collaboration-panel__stat-label">
              Canvas Nodes:
            </span>
            <span className="collaboration-panel__stat-value">
              {nodes.length}
            </span>
          </div>
          <div className="collaboration-panel__stat">
            <span className="collaboration-panel__stat-label">
              Canvas Edges:
            </span>
            <span className="collaboration-panel__stat-value">
              {edges.length}
            </span>
          </div>
        </div>
      </div>

      {/* 协作说明 */}
      <div className="collaboration-panel__section">
        <div className="collaboration-panel__section-title">
          How Collaboration Works
        </div>
        <div className="collaboration-panel__info-text">
          <p>
            <strong>Real-time sync:</strong> All changes are synchronized in
            real-time across all collaborators.
          </p>
          <p>
            <strong>Roles:</strong> Owners can manage collaborators, editors can
            edit the canvas, viewers can only view.
          </p>
          <p>
            <strong>Cursors:</strong> See where other collaborators are working
            on the canvas.
          </p>
        </div>
      </div>
    </div>
  );
}
