// 画布助手组件
import { useState, useCallback, useRef, useEffect } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import type { AssistantMessage, ResourceReference } from '../../types/canvas';

interface CanvasAssistantProps {
  onClose: () => void;
}

export function CanvasAssistant({ onClose }: CanvasAssistantProps) {
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const {
    nodes,
    edges,
    selectedNodeIds,
    assistantSessions,
    activeAssistantSessionId,
    addAssistantSession,
    addAssistantMessage,
    applyOps,
  } = useCanvasStore();

  // 获取当前会话
  const currentSession = assistantSessions.find(
    (s) => s.id === activeAssistantSessionId
  );

  // 自动滚动到底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [currentSession?.messages]);

  // 获取选中节点的引用
  const getSelectedReferences = useCallback((): ResourceReference[] => {
    return selectedNodeIds
      .map((id) => nodes.find((n) => n.id === id))
      .filter(Boolean)
      .map((node) => ({
        type: 'node' as const,
        id: node!.id,
        title: node!.data.title,
        content:
          'content' in node!.data
            ? (node!.data as any).content
            : 'prompt' in node!.data
            ? (node!.data as any).prompt
            : undefined,
        imageUrl:
          'imageUrl' in node!.data
            ? (node!.data as any).imageUrl
            : undefined,
      }));
  }, [selectedNodeIds, nodes]);

  // 构建画布快照
  const buildSnapshot = useCallback(() => {
    return {
      nodes: nodes.map((n) => ({
        id: n.id,
        type: n.type,
        title: n.data.title,
        status: n.data.status,
      })),
      edges: edges.map((e) => ({
        source: e.source,
        target: e.target,
      })),
    };
  }, [nodes, edges]);

  // 发送消息
  const handleSend = useCallback(async () => {
    if (!input.trim() || isLoading) return;

    // 如果没有活跃会话，创建一个
    let sessionId = activeAssistantSessionId;
    if (!sessionId) {
      sessionId = `session-${Date.now()}`;
      addAssistantSession({
        id: sessionId,
        title: input.substring(0, 30) + (input.length > 30 ? '...' : ''),
        messages: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      });
    }

    // 添加用户消息
    const userMessage: AssistantMessage = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: input,
      timestamp: new Date().toISOString(),
      references: getSelectedReferences(),
    };
    addAssistantMessage(sessionId, userMessage);
    setInput('');
    setIsLoading(true);

    try {
      // 构建上下文
      const snapshot = buildSnapshot();

      // 调用后端API（模拟）
      // 在实际实现中，这里会调用Wails后端的画布助手API
      const response = await simulateAssistantResponse(
        input,
        snapshot,
        getSelectedReferences()
      );

      // 添加助手消息
      const assistantMessage: AssistantMessage = {
        id: `msg-${Date.now()}`,
        role: 'assistant',
        content: response.content,
        timestamp: new Date().toISOString(),
      };
      addAssistantMessage(sessionId, assistantMessage);

      // 如果有操作建议，应用到画布
      if (response.ops && response.ops.length > 0) {
        applyOps(response.ops);
      }
    } catch (error) {
      console.error('Assistant error:', error);
      const errorMessage: AssistantMessage = {
        id: `msg-${Date.now()}`,
        role: 'assistant',
        content: 'Sorry, I encountered an error. Please try again.',
        timestamp: new Date().toISOString(),
      };
      addAssistantMessage(sessionId, errorMessage);
    } finally {
      setIsLoading(false);
    }
  }, [
    input,
    isLoading,
    activeAssistantSessionId,
    addAssistantSession,
    addAssistantMessage,
    getSelectedReferences,
    buildSnapshot,
    currentSession?.messages,
    applyOps,
  ]);

  // 键盘快捷键
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend]
  );

  return (
    <div className="assistant-panel">
      <div className="assistant-panel__header">
        <div className="assistant-panel__title">Canvas Assistant</div>
        <button className="assistant-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      <div className="assistant-panel__messages">
        {currentSession?.messages.length === 0 ? (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: 'var(--fg-faint)',
              fontSize: 'var(--text-sm)',
              textAlign: 'center',
              padding: 'var(--space-4)',
            }}
          >
            <div style={{ fontSize: '32px', marginBottom: 'var(--space-3)' }}>
              🎨
            </div>
            <div>Ask me anything about your canvas</div>
            <div style={{ marginTop: 'var(--space-2)', fontSize: 'var(--text-xs)' }}>
              I can help you create nodes, analyze workflows, and more.
            </div>
          </div>
        ) : (
          currentSession?.messages.map((msg) => (
            <div
              key={msg.id}
              className={`assistant-panel__message assistant-panel__message--${msg.role}`}
            >
              {msg.role === 'assistant' && (
                <div className="assistant-panel__message-avatar assistant-panel__message-avatar--assistant">
                  AI
                </div>
              )}
              <div
                className={`assistant-panel__message-content assistant-panel__message-content--${msg.role}`}
              >
                {msg.content}
              </div>
              {msg.role === 'user' && (
                <div className="assistant-panel__message-avatar assistant-panel__message-avatar--user">
                  U
                </div>
              )}
            </div>
          ))
        )}
        {isLoading && (
          <div className="assistant-panel__message assistant-panel__message--assistant">
            <div className="assistant-panel__message-avatar assistant-panel__message-avatar--assistant">
              AI
            </div>
            <div className="assistant-panel__message-content assistant-panel__message-content--assistant">
              <div className="animate-pulse">Thinking...</div>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="assistant-panel__input">
        <input
          className="assistant-panel__input-field"
          placeholder="Ask about your canvas..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isLoading}
        />
        <button
          className="assistant-panel__send-btn"
          onClick={handleSend}
          disabled={!input.trim() || isLoading}
        >
          ↑
        </button>
      </div>
    </div>
  );
}

// 模拟助手响应（实际实现中会调用后端API）
async function simulateAssistantResponse(
  message: string,
  snapshot: any,
  references: ResourceReference[]
): Promise<{ content: string; ops?: any[] }> {
  // 模拟延迟
  await new Promise((resolve) => setTimeout(resolve, 1000));

  // 分析用户消息
  const lowerMessage = message.toLowerCase();

  // 如果提到了创建节点
  if (
    lowerMessage.includes('create') ||
    lowerMessage.includes('add') ||
    lowerMessage.includes('new')
  ) {
    if (lowerMessage.includes('text')) {
      return {
        content:
          "I'll create a text node for you. You can edit it to add your content.",
        ops: [
          {
            type: 'add_node',
            nodeType: 'text',
            position: { x: 100, y: 100 },
            metadata: { title: 'New Text', content: 'Edit this text...' },
          },
        ],
      };
    }
    if (lowerMessage.includes('image')) {
      return {
        content:
          "I'll create an image node for you. You can configure it to generate an image.",
        ops: [
          {
            type: 'add_node',
            nodeType: 'image',
            position: { x: 100, y: 100 },
            metadata: { title: 'New Image', status: 'idle' },
          },
        ],
      };
    }
  }

  // 如果提到了分析
  if (
    lowerMessage.includes('analyze') ||
    lowerMessage.includes('what') ||
    lowerMessage.includes('describe')
  ) {
    const nodeCount = snapshot.nodes.length;
    const edgeCount = snapshot.edges.length;

    if (nodeCount === 0) {
      return {
        content:
          "Your canvas is empty. Would you like me to help you create some nodes to get started?",
      };
    }

    return {
      content: `Your canvas has ${nodeCount} nodes and ${edgeCount} connections. ${
        references.length > 0
          ? `You have ${references.length} nodes selected.`
          : ''
      }`,
    };
  }

  // 默认响应
  return {
    content: `I understand you want to: "${message}". Let me help you with that. Could you provide more details about what you'd like to do?`,
  };
}
