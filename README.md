# Zhulong（烛龙）

**基于 DeepSeek 的通用自主循环 Agent 框架**

[![Go Version](https://img.shields.io/badge/Go-1.26.3-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 项目简介

Zhulong（烛龙）是一个基于 DeepSeek 的通用自主循环 Agent 框架，采用 Go 语言实现。它通过状态机驱动的循环控制，实现规划 → 执行 → 反省 → 重规划的自主多轮循环，无需每轮人工触发。

### 核心特性

- **自主多轮循环**：规划 → 执行 → 反省 → 重新规划，无需每轮人工触发
- **高效上下文管理**：深度优化 DeepSeek prefix-cache，最大化缓存命中率
- **自适应学习**：成功策略复用、认知模型更新、探索/利用平衡
- **生产级可靠性**：检查点恢复、成本控制、人机协作断点、完整可观测性

### 设计目标

| 目标 | 说明 | 优先级 |
|------|------|--------|
| **高缓存命中率** | **深度优化 DeepSeek prefix-cache，上下文布局只追加不重写** | 🔴 最高 |
| CLI + 桌面端同步 | CLI 和 Windows 桌面端同步开发，共享核心逻辑 | 🔴 最高 |
| 通用性 | 不绑定特定场景（编程/写作/数据分析），通过工具和 prompt 定制 | 🟡 高 |
| 自适应学习 | 成功策略复用、认知模型更新、探索/利用平衡 | 🟡 高 |
| 崩溃可恢复 | 任意时刻中断都能从最近检查点恢复 | 🟡 高 |
| 成本可控 | 多层预算机制，防止无限循环烧 token | 🟡 高 |
| 可观测 | 完整 trace，每步决策可追溯 | 🟢 中 |

## 架构概览

```
┌──────────────────────────────────────────────────────────────────┐
│                          Zhulong（烛龙）                          │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                     Controller                            │    │
│  │                                                           │    │
│  │   ┌────────┐    ┌────────┐    ┌──────────┐    ┌────────┐ │    │
│  │   │Planner │───→│Executor│───→│ Reflector│───→│RePlanner│ │    │
│  │   └───▲────┘    └────────┘    └─────┬────┘    └───┬────┘ │    │
│  │       │                             │             │       │    │
│  │       └─────────────┬───────────────┘             │       │    │
│  │                     ▼                             │       │    │
│  │              ┌─────────────┐                      │       │    │
│  │              │   Goal FSM  │◄─────────────────────┘       │    │
│  │              │   目标状态机  │                              │    │
│  │              └──────┬──────┘                              │    │
│  └─────────────────────┼──────────────────────────────────────┘    │
│                        │                                          │
│  ┌─────────────────────┼──────────────────────────────────────┐    │
│  │                     ▼           Infrastructure             │    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐│    │
│  │  │   Memory   │ │ Compressor │ │   Tools    │ │Checkpoint││    │
│  │  │  三层记忆   │ │  上下文压缩 │ │  工具抽象层 │ │  检查点  ││    │
│  │  └────────────┘ └────────────┘ └────────────┘ └──────────┘│    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐│    │
│  │  │   Budget   │ │   Trace    │ │   Human    │ │ Learning ││    │
│  │  │  成本控制   │ │  可观测性   │ │  人机协作   │ │ 自适应学习││    │
│  │  └────────────┘ └────────────┘ └────────────┘ └──────────┘│    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │                    Providers 接口层                         │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │  │ DeepSeek API │  │  MCP Client  │  │ Custom Tools │     │    │
│  │  │  深度优化     │  │  工具协议     │  │  自定义工具   │     │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

## 安装

### 前置要求

- Go 1.26.3 或更高版本
- DeepSeek API Key（可选，用于真实 LLM 调用）

### 安装步骤

```bash
# 克隆仓库
git clone https://github.com/qoqu/zhuLong.git
cd zhuLong

# 安装依赖
go mod tidy

# 构建 CLI
go build -o zhulong ./cmd/zhulong
```

## 使用方法

### CLI 使用

```bash
# 设置 DeepSeek API Key（可选）
export DEEPSEEK_API_KEY="your-api-key"

# 运行 Agent
./zhulong --goal "分析当前项目的代码质量"
```

### 桌面端使用

```bash
# 进入桌面端目录
cd desktop

# 安装前端依赖
cd frontend && npm install && cd ..

# 运行桌面端
go run .
```

### 编程接口使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/qoqu/zhuLong/pkg"
)

func main() {
    // 创建 Agent
    agent, err := pkg.NewAgent(
        pkg.WithGoal("分析当前项目的代码质量"),
        pkg.WithMaxLoops(10),
        pkg.WithVerbose(true),
    )
    if err != nil {
        panic(err)
    }

    // 设置 Provider（可选，默认使用 HeuristicProvider）
    // agent.SetProvider(pkg.NewDeepSeekProvider("your-api-key", "deepseek-chat"))

    // 运行 Agent
    result, err := agent.Run()
    if err != nil {
        panic(err)
    }

    fmt.Printf("状态: %s\n", result.Status)
    fmt.Printf("答案: %s\n", result.Answer)
    fmt.Printf("循环次数: %d\n", result.Loops)
    fmt.Printf("Token 使用: %d\n", result.TokensUsed)
    fmt.Printf("耗时: %s\n", result.Duration)
}
```

## 配置

配置文件位于 `config/default.yaml`，包含以下配置项：

- **DeepSeek 配置**：模型、API Key、Base URL、温度、最大 Token
- **循环控制**：最大循环数、最大运行时间、检查点间隔
- **成本控制**：最大 Token 数、最大成本、警告阈值
- **规划器**：最大步骤数、允许重规划、最大重规划次数
- **执行器**：工具超时时间
- **反省器**：置信度阈值、自动失败阈值
- **压缩**：修剪配置、骨架配置
- **停滞检测**：窗口大小、熵阈值
- **探索**：基础温度、最大温度
- **学习**：积木块、内部模型、多样性、混沌边缘

## 无限画布功能

Zhulong 提供无限画布功能，用于可视化 Agent 工作循环和内容创作流程。

### 核心功能

| 功能 | 说明 |
|------|------|
| **无限画布** | React Flow 引擎，支持拖拽、缩放、连线 |
| **多媒体节点** | 文本、图片、视频、音频、分镜、配置节点 |
| **Ops 操作抽象** | 统一操作接口，支持撤销/重做 |
| **画布助手** | 上下文对话、选中节点引用、多轮对话 |
| **资产管理** | 资产列表、类型过滤、搜索、创建、删除 |
| **版本管理** | 版本创建、恢复、删除、列表 |
| **导出与分享** | JSON/PNG/SVG 导出、JSON 导入、分享链接 |
| **协作功能** | 协作者邀请、角色管理、在线状态 |
| **性能优化** | 性能等级、统计、建议、监控 |
| **离线支持** | 网络状态、本地存储、同步控制 |
| **AI 增强** | 自动布局、节点建议、工作流优化 |
| **节点扩展** | 插件列表、启用/禁用、安装/卸载 |

### 使用方式

1. 启动桌面端应用
2. 点击顶部的 **"🎨 Canvas"** 按钮切换到画布视图
3. 使用工具栏的按钮打开各种面板
4. 拖拽节点到画布创建内容

### 设计原则

- **缓存命中率铁律**：画布功能不影响 Agent 循环的缓存命中率
- **只借鉴不抄袭**：学习 TapCanvas、Toonflow、infinite-canvas 的设计思路，从零实现

## 项目结构

```
zhuLong/
├── cmd/zhulong/          # CLI 入口
├── config/               # 配置文件
│   └── default.yaml      # 默认配置
├── desktop/              # Windows 桌面端（Wails + React）
│   ├── main.go
│   ├── app.go
│   ├── agent.go
│   └── frontend/         # React 前端
├── docs/                 # 文档
│   └── design.md         # 详细设计文档
├── internal/             # 内部模块
│   ├── approval/         # 审批引擎
│   ├── budget/           # 成本控制
│   ├── checkpoint/       # 检查点
│   ├── compressor/       # 上下文压缩
│   ├── controller/       # 状态机控制器
│   ├── environment/      # 环境感知
│   ├── executor/         # 执行器
│   ├── exploration/      # 探索触发
│   ├── human/            # 人机协作
│   ├── information/      # 信息论
│   ├── learning/         # 自适应学习
│   ├── memory/           # 记忆系统
│   ├── planner/          # 规划器
│   ├── provider/         # LLM Provider
│   ├── reflector/        # 反省器
│   ├── skills/           # 技能管理
│   ├── stability/        # 稳定性分析
│   ├── stagnation/       # 停滞检测
│   ├── synergetics/      # 协同学
│   ├── tools/            # 工具层
│   └── trace/            # 可观测性
├── pkg/                  # 公共 API（CLI 和桌面端共享）
│   ├── agent.go          # Agent 核心逻辑
│   └── types.go          # 公共类型
├── examples/             # 示例
├── testing/              # 测试
├── go.mod
└── README.md
```

## 理论基础

Zhulong 融合六大系统科学理论：

| 理论 | 核心应用 | 可行性 |
|------|---------|--------|
| 一般系统论（贝塔朗菲） | 开放系统、备选路径 | ✅ 已实现 |
| 工程控制论（钱学森） | 振荡/发散检测、性能指标 | ✅ 已实现 |
| 信息论（香农） | 信息增益、信息密度 | ✅ 已实现 |
| 耗散结构理论（普利高津） | 停滞检测、探索触发 | ✅ 已实现 |
| 协同学（哈肯） | 序参量识别、役使原理 | ✅ 已实现 |
| 复杂适应系统（霍兰德） | 积木块、认知模型、多样性 | ✅ 已实现 |

## 缓存命中率铁律

> **所有架构补充、优化、新功能，绝对不能破坏高缓存命中率这一核心优势。**

1. prefix 只追加不修改（system prompt + skeleton 永不改写）
2. 历史只压缩不重排（旧循环压缩为 summary 追加）
3. 裁剪只在动态区间（工具结果裁剪只在当前循环）

## 开发阶段

- **Phase 1**: MVP（状态机+三阶段循环+基础配置）✅ 已完成
- **Phase 2**: 上下文优化（Memory+裁剪+预算）✅ 已完成
- **Phase 3**: 生产级特性（检查点+Trace+人机断点+并行+骨架压缩）✅ 已完成
- **Phase 4**: 多模型+工具生态 🚧 进行中
- **Phase 5**: 打磨+文档 📋 计划中

## 贡献

欢迎贡献！请遵循以下步骤：

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 致谢

- [DeepSeek](https://www.deepseek.com/) - 提供强大的 LLM API
- [Wails](https://wails.io/) - Go 桌面应用框架
- [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) - 参考设计思路
- [Ailoom-Context](https://github.com/EvanLyu-oss/Ailoom-Context) - 骨架压缩结构设计

## 联系方式

- 项目链接: https://github.com/qoqu/zhuLong
- 问题反馈: [Issues](https://github.com/qoqu/zhuLong/issues)
