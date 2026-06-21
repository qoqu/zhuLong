// 画布助手组件 - 支持@引用机制和画布快照集成
import { useState, useCallback, useRef, useEffect, useMemo } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import type { AssistantMessage, ResourceReference, CanvasNode, Asset } from '../../types/canvas';

interface CanvasAssistantProps {
  onClose: () => void;
}

// @引用解析结果
interface ReferenceMatch {
  type: 'node' | 'asset';
  id: string;
  fullMatch: string; // 完整匹配文本，如 @[node:xxx]
  startIndex: number;
  endIndex: number;
}

// 解析@引用
function parseReferences(text: string, nodes: CanvasNode[], assets: Asset[]): ReferenceMatch[] {
  const matches: ReferenceMatch[] = [];
  
  // 匹配 @[node:xxx] 或 @[asset:xxx]
  const regex = /@\[(node|asset):([^\]]+)\]/g;
  let match;
  
  while ((match = regex.exec(text)) !== null) {
    const type = match[1] as 'node' | 'asset';
    const id = match[2];
    
    // 验证引用是否存在
    if (type === 'node' && nodes.some(n => n.id === id)) {
      matches.push({
        type,
        id,
        fullMatch: match[0],
        startIndex: match.index,
        endIndex: match.index + match[0].length,
      });
    } else if (type === 'asset' && assets.some(a => a.id === id)) {
      matches.push({
        type,
        id,
        fullMatch: match[0],
        startIndex: match.index,
        endIndex: match.index + match[0].length,
      });
    }
  }
  
  return matches;
}

// 构建引用上下文
function buildReferenceContext(
  references: ReferenceMatch[],
  nodes: CanvasNode[],
  assets: Asset[]
): ResourceReference[] {
  return references.map(ref => {
    if (ref.type === 'node') {
      const node = nodes.find(n => n.id === ref.id);
      if (!node) return null;
      return {
        type: 'node' as const,
        id: node.id,
        title: node.data.title,
        content: 'content' in node.data ? (node.data as any).content : 
                 'prompt' in node.data ? (node.data as any).prompt : undefined,
        imageUrl: 'imageUrl' in node.data ? (node.data as any).imageUrl : undefined,
      };
    } else {
      const asset = assets.find(a => a.id === ref.id);
      if (!asset) return null;
      return {
        type: 'asset' as const,
        id: asset.id,
        title: asset.name,
        content: asset.content,
        imageUrl: asset.thumbnailUrl || asset.url,
      };
    }
  }).filter(Boolean) as ResourceReference[];
}

export function CanvasAssistant({ onClose }: CanvasAssistantProps) {
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [showMentions, setShowMentions] = useState(false);
  const [mentionFilter, setMentionFilter] = useState('');
  const [cursorPosition, setCursorPosition] = useState(0);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const {
    nodes,
    edges,
    assets,
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

  // 获取可引用的项目列表
  const mentionableItems = useMemo(() => {
    const items: Array<{ type: 'node' | 'asset'; id: string; title: string }> = [];
    
    // 添加节点
    nodes.forEach(node => {
      items.push({
        type: 'node',
        id: node.id,
        title: node.data.title,
      });
    });
    
    // 添加资产
    assets.forEach(asset => {
      items.push({
        type: 'asset',
        id: asset.id,
        title: asset.name,
      });
    });
    
    return items;
  }, [nodes, assets]);

  // 过滤可引用项目
  const filteredMentionableItems = useMemo(() => {
    if (!mentionFilter) return mentionableItems;
    const filter = mentionFilter.toLowerCase();
    return mentionableItems.filter(item => 
      item.title.toLowerCase().includes(filter) ||
      item.id.toLowerCase().includes(filter)
    );
  }, [mentionableItems, mentionFilter]);

  // 检测@触发
  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const cursor = e.target.selectionStart || 0;
    
    setInput(value);
    setCursorPosition(cursor);
    
    // 检查是否输入了@
    const beforeCursor = value.substring(0, cursor);
    const lastAtIndex = beforeCursor.lastIndexOf('@');
    
    if (lastAtIndex !== -1) {
      const afterAt = beforeCursor.substring(lastAtIndex + 1);
      // 如果@后面没有空格，显示提及菜单
      if (!afterAt.includes(' ') && afterAt.length <= 20) {
        setShowMentions(true);
        setMentionFilter(afterAt);
        return;
      }
    }
    
    setShowMentions(false);
    setMentionFilter('');
  }, []);

  // 插入@引用
  const insertMention = useCallback((item: { type: 'node' | 'asset'; id: string; title: string }) => {
    const beforeCursor = input.substring(0, cursorPosition);
    const lastAtIndex = beforeCursor.lastIndexOf('@');
    
    if (lastAtIndex !== -1) {
      const newInput = 
        input.substring(0, lastAtIndex) + 
        `@[${item.type}:${item.id}]` + 
        input.substring(cursorPosition);
      
      setInput(newInput);
      setShowMentions(false);
      setMentionFilter('');
      
      // 重新聚焦输入框
      setTimeout(() => {
        if (inputRef.current) {
          const newCursorPos = lastAtIndex + `@[${item.type}:${item.id}]`.length;
          inputRef.current.setSelectionRange(newCursorPos, newCursorPos);
          inputRef.current.focus();
        }
      }, 0);
    }
  }, [input, cursorPosition]);

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

  // 构建画布快照（完整版）
  const buildFullSnapshot = useCallback(() => {
    return {
      nodes: nodes.map((n) => ({
        id: n.id,
        type: n.type,
        title: n.data.title,
        status: n.data.status,
        content: 'content' in n.data ? (n.data as any).content : undefined,
        prompt: 'prompt' in n.data ? (n.data as any).prompt : undefined,
        imageUrl: 'imageUrl' in n.data ? (n.data as any).imageUrl : undefined,
      })),
      edges: edges.map((e) => ({
        source: e.source,
        target: e.target,
      })),
      selectedNodeIds,
      assetCount: assets.length,
    };
  }, [nodes, edges, selectedNodeIds, assets]);

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

    // 解析@引用
    const referenceMatches = parseReferences(input, nodes, assets);
    const referenceContext = buildReferenceContext(referenceMatches, nodes, assets);
    
    // 合并选中节点引用和@引用
    const selectedRefs = getSelectedReferences();
    const allReferences = [...selectedRefs, ...referenceContext];

    // 添加用户消息
    const userMessage: AssistantMessage = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: input,
      timestamp: new Date().toISOString(),
      references: allReferences,
    };
    addAssistantMessage(sessionId, userMessage);
    setInput('');
    setShowMentions(false);

    setIsLoading(true);

    try {
      // 构建完整上下文
      const snapshot = buildFullSnapshot();

      // 调用后端API（模拟）
      // 在实际实现中，这里会调用Wails后端的画布助手API
      const response = await simulateAssistantResponse(
        input,
        snapshot,
        [],
        allReferences
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
    nodes,
    assets,
    addAssistantSession,
    addAssistantMessage,
    getSelectedReferences,
    buildFullSnapshot,
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
      // Escape关闭提及菜单
      if (e.key === 'Escape') {
        setShowMentions(false);
      }
    },
    [handleSend]
  );

  // 渲染消息内容（支持@引用高亮）
  const renderMessageContent = useCallback((content: string) => {
    // 替换@引用为高亮标签
    const parts = content.split(/(@\[(?:node|asset):[^\]]+\])/g);
    
    return parts.map((part, index) => {
      const match = part.match(/@\[(node|asset):([^\]]+)\]/);
      if (match) {
        const type = match[1];
        const id = match[2];
        const item = type === 'node' 
          ? nodes.find(n => n.id === id)
          : assets.find(a => a.id === id);
        
        const title = item 
          ? (type === 'node' ? (item as CanvasNode).data.title : (item as Asset).name)
          : id;
        
        return (
          <span 
            key={index} 
            className="assistant-panel__mention"
            style={{
              background: type === 'node' ? 'rgba(10, 132, 255, 0.2)' : 'rgba(52, 199, 89, 0.2)',
              color: type === 'node' ? 'var(--accent)' : 'var(--ok)',
              padding: '2px 6px',
              borderRadius: '4px',
              fontSize: '0.9em',
            }}
          >
            @{title}
          </span>
        );
      }
      return part;
    });
  }, [nodes, assets]);

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
              Use @[node:id] or @[asset:id] to reference specific items
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
                {renderMessageContent(msg.content)}
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

      <div className="assistant-panel__input" style={{ position: 'relative' }}>
        {/* @提及菜单 */}
        {showMentions && filteredMentionableItems.length > 0 && (
          <div 
            className="assistant-panel__mentions-menu"
            style={{
              position: 'absolute',
              bottom: '100%',
              left: 0,
              right: 0,
              maxHeight: '200px',
              overflowY: 'auto',
              background: 'var(--bg-elev)',
              border: '1px solid var(--border)',
              borderRadius: '8px',
              marginBottom: '4px',
              boxShadow: '0 -4px 12px rgba(0,0,0,0.2)',
            }}
          >
            {filteredMentionableItems.slice(0, 10).map((item) => (
              <div
                key={`${item.type}-${item.id}`}
                className="assistant-panel__mention-item"
                onClick={() => insertMention(item)}
                style={{
                  padding: '8px 12px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  borderBottom: '1px solid var(--border)',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-hover)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent';
                }}
              >
                <span style={{ 
                  fontSize: '10px', 
                  padding: '2px 4px', 
                  borderRadius: '3px',
                  background: item.type === 'node' ? 'rgba(10, 132, 255, 0.2)' : 'rgba(52, 199, 89, 0.2)',
                  color: item.type === 'node' ? 'var(--accent)' : 'var(--ok)',
                }}>
                  {item.type}
                </span>
                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {item.title}
                </span>
                <span style={{ fontSize: '10px', color: 'var(--fg-faint)' }}>
                  {item.id.substring(0, 8)}...
                </span>
              </div>
            ))}
          </div>
        )}
        
        <input
          ref={inputRef}
          className="assistant-panel__input-field"
          placeholder="Ask about your canvas... (use @ to reference)"
          value={input}
          onChange={handleInputChange}
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
  _history: AssistantMessage[],
  references: ResourceReference[]
): Promise<{ content: string; ops?: any[] }> {
  // 模拟延迟
  await new Promise((resolve) => setTimeout(resolve, 1000));

  // 分析用户消息
  const lowerMessage = message.toLowerCase();

  // 如果有@引用，优先处理引用上下文
  if (references.length > 0) {
    const refNames = references.map(r => r.title).join(', ');
    
    // 如果提到了创建节点
    if (
      lowerMessage.includes('create') ||
      lowerMessage.includes('add') ||
      lowerMessage.includes('new')
    ) {
      if (lowerMessage.includes('text')) {
        return {
          content: `I'll create a text node based on the referenced items: ${refNames}`,
          ops: [
            {
              type: 'add_node',
              nodeType: 'text',
              position: { x: 100, y: 100 },
              metadata: { 
                title: 'New Text', 
                content: `Based on: ${refNames}`,
                referenceIds: references.map(r => r.id),
              },
            },
          ],
        };
      }
    }

    // 分析引用内容
    return {
      content: `I can see you've referenced: ${refNames}. What would you like to do with these items?`,
    };
  }

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
        snapshot.selectedNodeIds.length > 0
          ? `You have ${snapshot.selectedNodeIds.length} nodes selected.`
          : ''
      }`,
    };
  }

  // 默认响应
  return {
    content: `I understand you want to: "${message}". Let me help you with that. Could you provide more details about what you'd like to do?`,
  };
}
