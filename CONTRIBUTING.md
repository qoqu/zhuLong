# 贡献指南

感谢你对 Zhulong（烛龙）项目的关注！我们欢迎任何形式的贡献。

## 目录

- [行为准则](#行为准则)
- [如何贡献](#如何贡献)
- [开发环境](#开发环境)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [Pull Request 流程](#pull-request-流程)
- [问题报告](#问题报告)
- [功能请求](#功能请求)

## 行为准则

本项目采用开源社区的行为准则。请尊重所有参与者，保持友善和专业的态度。

## 如何贡献

### 贡献方式

1. **代码贡献**：修复 bug、添加新功能、优化性能
2. **文档改进**：完善文档、修复错误、添加示例
3. **问题报告**：报告 bug、提出改进建议
4. **代码审查**：审查 Pull Request，提供反馈
5. **测试**：编写测试、报告测试结果

### 贡献流程

1. Fork 项目仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 开发环境

### 前置要求

- Go 1.26.3 或更高版本
- Git
- DeepSeek API Key（可选，用于真实 LLM 调用）

### 环境设置

```bash
# 克隆仓库
git clone https://github.com/qoqu/zhuLong.git
cd zhuLong

# 安装依赖
go mod tidy

# 运行测试
make test

# 构建项目
make build
```

### 项目结构

```
zhuLong/
├── cmd/zhulong/          # CLI 入口
├── config/               # 配置文件
├── desktop/              # Windows 桌面端
├── docs/                 # 文档
├── examples/             # 示例
├── internal/             # 内部模块
├── pkg/                  # 公共 API
├── Makefile              # 构建系统
└── README.md             # 项目文档
```

## 代码规范

### Go 代码规范

1. **命名规范**
   - 包名：小写单词，不使用下划线
   - 函数名：驼峰命名法，导出函数首字母大写
   - 变量名：驼峰命名法
   - 常量名：全大写，下划线分隔

2. **注释规范**
   - 所有导出的函数、类型、常量必须有注释
   - 注释以被注释的对象名称开头
   - 使用英文或中文注释，保持一致性

3. **错误处理**
   - 使用 `fmt.Errorf` 和 `%w` 包装错误
   - 不要忽略错误返回值
   - 提供有意义的错误消息

4. **代码组织**
   - 每个包只做一件事
   - 避免循环依赖
   - 使用接口解耦

### 示例代码

```go
// Agent is the main entry point for the Zhulong agent
type Agent struct {
	goal    string
	options *Options
}

// NewAgent creates a new agent with the given options
func NewAgent(opts ...Option) (*Agent, error) {
	// 实现代码
}

// Run runs the agent and returns the result
func (a *Agent) Run() (*AgentResult, error) {
	// 实现代码
}
```

## 提交规范

### 提交消息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### 类型（Type）

- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整（不影响功能）
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建工具、辅助工具等

### 范围（Scope）

可选，表示影响范围：
- `planner`: 规划器模块
- `executor`: 执行器模块
- `reflector`: 反省器模块
- `memory`: 记忆系统
- `desktop`: 桌面端
- 等等

### 示例

```
feat(planner): add alternative path planning

- Add AlternativePlanner struct
- Implement fallback strategy when primary plan fails
- Add unit tests for alternative planning

Closes #123
```

```
fix(executor): handle nil tool result gracefully

- Check for nil result before accessing fields
- Return meaningful error message
- Add test case for nil result scenario
```

## Pull Request 流程

### PR 标题

使用与提交消息相同的格式：
```
<type>(<scope>): <subject>
```

### PR 描述

包含以下内容：
1. **变更说明**：描述这个 PR 做了什么
2. **相关问题**：关联的 issue 编号
3. **测试情况**：说明如何测试这些变更
4. **截图**：如果有 UI 变更，提供截图
5. **检查清单**：完成以下检查

### 检查清单

- [ ] 代码遵循项目规范
- [ ] 添加了必要的测试
- [ ] 所有测试通过 (`make test`)
- [ ] 更新了相关文档
- [ ] 提交消息符合规范
- [ ] 没有引入新的警告

### 审查流程

1. 提交 PR 后，等待 CI 检查通过
2. 至少需要一位维护者审查
3. 根据审查意见修改代码
4. 审查通过后合并

## 问题报告

### 报告 Bug

使用 GitHub Issues 报告 bug，包含以下信息：

1. **环境信息**
   - 操作系统
   - Go 版本
   - 项目版本

2. **复现步骤**
   - 详细的操作步骤
   - 输入数据
   - 预期行为
   - 实际行为

3. **错误信息**
   - 完整的错误日志
   - 堆栈跟踪（如果有）

4. **截图**
   - 如果有 UI 问题，提供截图

### 示例

```markdown
## Bug 报告

### 环境信息
- OS: Windows 11
- Go: 1.26.3
- Zhulong: v0.1.0

### 复现步骤
1. 运行 `zhulong run "分析代码"`
2. 等待执行完成
3. 观察输出

### 预期行为
应该输出分析结果

### 实际行为
程序崩溃，错误信息：...

### 错误日志
```
panic: runtime error: ...
```
```

## 功能请求

### 请求新功能

使用 GitHub Issues 提交功能请求，包含以下信息：

1. **功能描述**：清晰描述想要的功能
2. **使用场景**：说明为什么需要这个功能
3. **解决方案**：如果有想法，描述可能的实现方式
4. **替代方案**：考虑过的其他方案

### 示例

```markdown
## 功能请求

### 功能描述
支持自定义工具注册

### 使用场景
用户需要调用特定的 API 或执行自定义命令

### 解决方案
提供 ToolRegistry 接口，允许用户注册自定义工具

### 替代方案
使用配置文件定义工具
```

## 开发指南

### 添加新模块

1. 在 `internal/` 下创建新目录
2. 实现模块功能
3. 编写单元测试
4. 更新文档
5. 提交 PR

### 添加新工具

1. 在 `internal/tools/` 下创建新文件
2. 实现 `Tool` 接口
3. 在 `executor` 中注册工具
4. 编写测试
5. 更新文档

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定模块测试
go test ./internal/planner/ -v

# 运行基准测试
make bench

# 生成覆盖率报告
make test-coverage
```

### 代码审查要点

1. **功能正确性**：代码是否实现了预期功能
2. **错误处理**：是否正确处理了错误情况
3. **性能**：是否有性能问题
4. **安全性**：是否有安全漏洞
5. **可读性**：代码是否易于理解
6. **测试覆盖**：是否有足够的测试

## 社区

### 沟通渠道

- **GitHub Issues**：问题报告和功能请求
- **GitHub Discussions**：技术讨论和问答
- **Pull Requests**：代码贡献和审查

### 行为准则

- 尊重所有参与者
- 保持友善和专业的态度
- 接受建设性的批评
- 关注对社区最有利的事情

## 许可证

本项目采用 MIT 许可证。贡献代码即表示你同意你的贡献将在 MIT 许可证下发布。

## 致谢

感谢所有为 Zhulong 项目做出贡献的人！

---

**感谢你的贡献！** 🎉
