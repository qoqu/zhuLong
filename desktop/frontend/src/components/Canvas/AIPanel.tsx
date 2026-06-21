// AI增强面板组件
import { useState } from 'react';
import { useCanvasStore } from '../../stores/canvasStore';
import { CanvasNodeType } from '../../types/canvas';

interface AIPanelProps {
  onClose: () => void;
}

export function AIPanel({ onClose }: AIPanelProps) {
  const [isGenerating, setIsGenerating] = useState(false);
  const [selectedFeature, setSelectedFeature] = useState<string | null>(null);

  const { nodes, edges, addNode, applyOps } = useCanvasStore();

  // AI功能列表
  const aiFeatures = [
    {
      id: 'auto-layout',
      icon: '📐',
      title: 'Auto Layout',
      description: 'Automatically arrange nodes using AI',
      action: handleAutoLayout,
    },
    {
      id: 'suggest-nodes',
      icon: '💡',
      title: 'Suggest Nodes',
      description: 'Get AI suggestions for new nodes',
      action: handleSuggestNodes,
    },
    {
      id: 'optimize-workflow',
      icon: '⚡',
      title: 'Optimize Workflow',
      description: 'Get AI suggestions to optimize your workflow',
      action: handleOptimizeWorkflow,
    },
    {
      id: 'generate-content',
      icon: '✨',
      title: 'Generate Content',
      description: 'Generate content for selected nodes',
      action: handleGenerateContent,
    },
  ];

  // 自动布局（模拟）
  async function handleAutoLayout() {
    setIsGenerating(true);
    setSelectedFeature('auto-layout');

    // 模拟AI处理
    await new Promise((resolve) => setTimeout(resolve, 2000));

    // 在实际实现中，这里会调用AI API获取布局建议
    const suggestions = [
      { nodeId: nodes[0]?.id, position: { x: 100, y: 100 } },
      { nodeId: nodes[1]?.id, position: { x: 300, y: 100 } },
      { nodeId: nodes[2]?.id, position: { x: 500, y: 100 } },
    ];

    // 应用布局建议
    const ops = suggestions
      .filter((s) => s.nodeId)
      .map((s) => ({
        type: 'move_node' as const,
        id: s.nodeId!,
        position: s.position,
      }));

    applyOps(ops);

    setIsGenerating(false);
    setSelectedFeature(null);
    alert('Auto layout applied!');
  }

  // 建议节点（模拟）
  async function handleSuggestNodes() {
    setIsGenerating(true);
    setSelectedFeature('suggest-nodes');

    // 模拟AI处理
    await new Promise((resolve) => setTimeout(resolve, 1500));

    // 在实际实现中，这里会调用AI API获取节点建议
    const suggestions = [
      {
        type: CanvasNodeType.Text,
        title: 'AI Suggested Text',
        content: 'This is an AI-generated suggestion',
        position: { x: 200, y: 200 },
      },
    ];

    // 添加建议的节点
    suggestions.forEach((suggestion) => {
      const nodeId = `node-${Date.now()}-${Math.random()
        .toString(36)
        .substr(2, 9)}`;
      addNode({
        id: nodeId,
        type: suggestion.type,
        position: suggestion.position,
        size: { width: 200, height: 100 },
        data: {
          id: nodeId,
          type: suggestion.type,
          title: suggestion.title,
          content: suggestion.content,
          status: 'idle',
        } as any,
      });
    });

    setIsGenerating(false);
    setSelectedFeature(null);
    alert('AI suggestions added!');
  }

  // 优化工作流（模拟）
  async function handleOptimizeWorkflow() {
    setIsGenerating(true);
    setSelectedFeature('optimize-workflow');

    // 模拟AI处理
    await new Promise((resolve) => setTimeout(resolve, 2000));

    // 在实际实现中，这里会调用AI API获取优化建议
    const suggestions = [
      'Consider merging nodes 1 and 2 for better flow',
      'Add a decision node between steps 3 and 4',
      'Remove redundant connections',
    ];

    setIsGenerating(false);
    setSelectedFeature(null);
    alert(`Optimization suggestions:\n\n${suggestions.join('\n')}`);
  }

  // 生成内容（模拟）
  async function handleGenerateContent() {
    setIsGenerating(true);
    setSelectedFeature('generate-content');

    // 模拟AI处理
    await new Promise((resolve) => setTimeout(resolve, 2500));

    // 在实际实现中，这里会调用AI API生成内容
    const generatedContent = {
      text: 'AI-generated content based on your canvas context',
      suggestions: ['Add more details', 'Consider alternative approaches'],
    };

    setIsGenerating(false);
    setSelectedFeature(null);
    alert(`Generated content:\n\n${generatedContent.text}`);
  }

  return (
    <div className="ai-panel">
      <div className="ai-panel__header">
        <div className="ai-panel__title">AI Enhancement</div>
        <button className="ai-panel__close" onClick={onClose}>
          ×
        </button>
      </div>

      {/* AI功能列表 */}
      <div className="ai-panel__section">
        <div className="ai-panel__section-title">AI Features</div>
        <div className="ai-panel__features">
          {aiFeatures.map((feature) => (
            <button
              key={feature.id}
              className={`ai-panel__feature ${
                selectedFeature === feature.id ? 'active' : ''
              }`}
              onClick={feature.action}
              disabled={isGenerating}
            >
              <div className="ai-panel__feature-icon">{feature.icon}</div>
              <div className="ai-panel__feature-info">
                <div className="ai-panel__feature-title">{feature.title}</div>
                <div className="ai-panel__feature-description">
                  {feature.description}
                </div>
              </div>
              {isGenerating && selectedFeature === feature.id && (
                <div className="ai-panel__feature-loading">⏳</div>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* AI设置 */}
      <div className="ai-panel__section">
        <div className="ai-panel__section-title">AI Settings</div>
        <div className="ai-panel__settings">
          <div className="ai-panel__setting">
            <label className="ai-panel__setting-label">Model:</label>
            <select className="ai-panel__setting-select">
              <option value="gpt-4">GPT-4</option>
              <option value="gpt-3.5">GPT-3.5</option>
              <option value="claude">Claude</option>
            </select>
          </div>
          <div className="ai-panel__setting">
            <label className="ai-panel__setting-label">Temperature:</label>
            <input
              type="range"
              className="ai-panel__setting-range"
              min="0"
              max="1"
              step="0.1"
              defaultValue="0.7"
            />
          </div>
        </div>
      </div>

      {/* 使用说明 */}
      <div className="ai-panel__section">
        <div className="ai-panel__section-title">How AI Works</div>
        <div className="ai-panel__info">
          <p>
            <strong>Auto Layout:</strong> AI analyzes your canvas structure and
            suggests optimal node positions.
          </p>
          <p>
            <strong>Suggest Nodes:</strong> AI recommends new nodes based on your
            existing workflow.
          </p>
          <p>
            <strong>Optimize Workflow:</strong> AI identifies inefficiencies and
            suggests improvements.
          </p>
          <p>
            <strong>Generate Content:</strong> AI generates content for selected
            nodes based on context.
          </p>
        </div>
      </div>

      {/* 统计信息 */}
      <div className="ai-panel__section">
        <div className="ai-panel__section-title">Canvas Context</div>
        <div className="ai-panel__stats">
          <div className="ai-panel__stat">
            <span className="ai-panel__stat-label">Nodes:</span>
            <span className="ai-panel__stat-value">{nodes.length}</span>
          </div>
          <div className="ai-panel__stat">
            <span className="ai-panel__stat-label">Edges:</span>
            <span className="ai-panel__stat-value">{edges.length}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
