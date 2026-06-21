// 多Agent工作空间协作面板（参考TapCanvas）
import { useState, useCallback } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import type { AgentInfo, AgentMessage, AgentWorkspace } from '../../types/canvas';

interface CollaborationPanelProps {
  onClose: () => void;
}

export function CollaborationPanel({ onClose }: CollaborationPanelProps) {
  const [activeTab, setActiveTab] = useState<'workspaces' | 'agents' | 'messages'>('workspaces');
  const [showCreateWorkspace, setShowCreateWorkspace] = useState(false);
  const [workspaceName, setWorkspaceName] = useState('');
  const [agentName, setAgentName] = useState('');
  const [agentRole, setAgentRole] = useState('');
  const [messageInput, setMessageInput] = useState('');

  const {
    agentWorkspaces,
    activeAgentWorkspaceId,
    addAgentWorkspace,
    setActiveAgentWorkspace,
    addAgent,
    removeAgent,
    sendAgentMessage,
  } = useCanvasStore();

  // 获取当前工作空间
  const currentWorkspace = agentWorkspaces.find(ws => ws.id === activeAgentWorkspaceId);

  // 创建工作空间
  const handleCreateWorkspace = useCallback(() => {
    if (!workspaceName.trim()) return;

    const newWorkspace: AgentWorkspace = {
      id: `ws-${Date.now()}`,
      name: workspaceName.trim(),
      agents: [],
      messages: [],
      handoffs: [],
      sharedAssets: [],
      ownerId: 'user',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    addAgentWorkspace(newWorkspace);
    setWorkspaceName('');
    setShowCreateWorkspace(false);
  }, [workspaceName, addAgentWorkspace]);

  // 添加Agent到工作空间
  const handleAddAgent = useCallback(() => {
    if (!currentWorkspace || !agentName.trim() || !agentRole.trim()) return;

    const newAgent: AgentInfo = {
      id: `agent-${Date.now()}`,
      name: agentName.trim(),
      role: agentRole.trim(),
      status: 'idle',
      capabilities: [],
      lastActive: new Date().toISOString(),
    };

    addAgent(currentWorkspace.id, newAgent);
    setAgentName('');
    setAgentRole('');
  }, [currentWorkspace, agentName, agentRole, addAgent]);

  // 发送消息到工作空间
  const handleSendMessage = useCallback(() => {
    if (!currentWorkspace || !messageInput.trim()) return;

    const newMsg: AgentMessage = {
      id: `msg-${Date.now()}`,
      fromAgentId: 'user',
      toAgentId: '*',
      type: 'broadcast',
      protocol: 'chat',
      payload: { text: messageInput.trim() },
      status: 'sent',
      timestamp: new Date().toISOString(),
    };

    sendAgentMessage(currentWorkspace.id, newMsg);
    setMessageInput('');
  }, [currentWorkspace, messageInput, sendAgentMessage]);

  // 渲染工作空间列表
  const renderWorkspaces = () => (
    <div className="collab-panel__list">
      {agentWorkspaces.map(ws => (
        <div
          key={ws.id}
          className={`collab-panel__item ${ws.id === activeAgentWorkspaceId ? 'selected' : ''}`}
          onClick={() => setActiveAgentWorkspace(ws.id)}
        >
          <div className="collab-panel__item-title">{ws.name}</div>
          <div className="collab-panel__item-meta">
            {ws.agents.length} agents · {ws.messages.length} messages
          </div>
        </div>
      ))}
      {showCreateWorkspace ? (
        <div className="collab-panel__create-form">
          <input
            className="collab-panel__input"
            placeholder="Workspace name..."
            value={workspaceName}
            onChange={(e) => setWorkspaceName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreateWorkspace()}
            autoFocus
          />
          <div className="collab-panel__form-actions">
            <button className="collab-panel__btn" onClick={handleCreateWorkspace}>
              Create
            </button>
            <button className="collab-panel__btn collab-panel__btn--secondary" onClick={() => setShowCreateWorkspace(false)}>
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button className="collab-panel__btn collab-panel__btn--add" onClick={() => setShowCreateWorkspace(true)}>
          + New Workspace
        </button>
      )}
    </div>
  );

  // 渲染Agent列表
  const renderAgents = () => {
    if (!currentWorkspace) {
      return <div className="collab-panel__empty">Select a workspace to manage agents</div>;
    }

    return (
      <div className="collab-panel__list">
        {currentWorkspace.agents.map(agent => (
          <div key={agent.id} className="collab-panel__item">
            <div className="collab-panel__item-header">
              <span className={`collab-panel__status collab-panel__status--${agent.status}`} />
              <div className="collab-panel__item-title">{agent.name}</div>
            </div>
            <div className="collab-panel__item-meta">{agent.role}</div>
            <button
              className="collab-panel__btn collab-panel__btn--danger"
              onClick={() => removeAgent(currentWorkspace.id, agent.id)}
              style={{ fontSize: '10px', padding: '2px 8px' }}
            >
              Remove
            </button>
          </div>
        ))}
        <div className="collab-panel__create-form">
          <input
            className="collab-panel__input"
            placeholder="Agent name..."
            value={agentName}
            onChange={(e) => setAgentName(e.target.value)}
            style={{ marginBottom: '4px' }}
          />
          <input
            className="collab-panel__input"
            placeholder="Agent role (e.g. Writer, Reviewer)..."
            value={agentRole}
            onChange={(e) => setAgentRole(e.target.value)}
            style={{ marginBottom: '4px' }}
          />
          <button className="collab-panel__btn" onClick={handleAddAgent}>
            + Add Agent
          </button>
        </div>
      </div>
    );
  };

  // 渲染消息列表
  const renderMessages = () => {
    if (!currentWorkspace) {
      return <div className="collab-panel__empty">Select a workspace to view messages</div>;
    }

    return (
      <div className="collab-panel__messages">
        <div className="collab-panel__messages-list">
          {currentWorkspace.messages.map(msg => (
            <div key={msg.id} className="collab-panel__message">
              <div className="collab-panel__message-header">
                <strong>{msg.fromAgentId === 'user' ? 'You' : msg.fromAgentId}</strong>
                <span className="collab-panel__message-type">[{msg.type}]</span>
                <span className="collab-panel__message-time">
                  {new Date(msg.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div className="collab-panel__message-body">
                {typeof msg.payload === 'object' ? msg.payload.text || JSON.stringify(msg.payload) : msg.payload}
              </div>
            </div>
          ))}
          {currentWorkspace.messages.length === 0 && (
            <div className="collab-panel__empty">No messages yet</div>
          )}
        </div>
        <div className="collab-panel__message-input">
          <input
            className="collab-panel__input"
            placeholder="Type a message..."
            value={messageInput}
            onChange={(e) => setMessageInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
          />
          <button className="collab-panel__btn" onClick={handleSendMessage}>
            Send
          </button>
        </div>
      </div>
    );
  };

  return (
    <div className="collab-panel">
      <div className="collab-panel__header">
        <div className="collab-panel__title">Multi-Agent Collaboration</div>
        <button className="collab-panel__close" onClick={onClose}>×</button>
      </div>

      <div className="collab-panel__tabs">
        <button
          className={`collab-panel__tab ${activeTab === 'workspaces' ? 'active' : ''}`}
          onClick={() => setActiveTab('workspaces')}
        >
          Workspaces
        </button>
        <button
          className={`collab-panel__tab ${activeTab === 'agents' ? 'active' : ''}`}
          onClick={() => setActiveTab('agents')}
        >
          Agents
        </button>
        <button
          className={`collab-panel__tab ${activeTab === 'messages' ? 'active' : ''}`}
          onClick={() => setActiveTab('messages')}
        >
          Messages
        </button>
      </div>

      <div className="collab-panel__content">
        {activeTab === 'workspaces' && renderWorkspaces()}
        {activeTab === 'agents' && renderAgents()}
        {activeTab === 'messages' && renderMessages()}
      </div>

      <div className="collab-panel__footer">
        {currentWorkspace && (
          <span>{currentWorkspace.agents.length} agents · {currentWorkspace.messages.length} messages</span>
        )}
      </div>
    </div>
  );
}
