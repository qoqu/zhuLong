# Zhulong (烛龙) — Design Document

> Version: v0.3
> Date: 2026-06-22
> Language: Go
> Status: P0-P4 Complete

---

## 1. Project Overview

Zhulong (烛龙, "Torch Dragon") is a **DeepSeek-powered autonomous loop agent framework** written in Go.

### Core Features

- **Autonomous Multi-round Loop**: Plan → Execute → Reflect → Replan, without per-round manual trigger
- **Efficient Context Management**: DeepSeek prefix-cache optimization to maximize cache hit rate
- **Adaptive Learning**: Strategy reuse, cognitive model updates, exploration/exploitation balance
- **Production-grade Reliability**: Checkpoint recovery, cost control, human-in-the-loop breakpoints, full observability

### Design Goals

| Goal | Description | Priority |
|------|-------------|----------|
| **High Cache Hit Rate** | Optimize DeepSeek prefix-cache; append only, never rewrite | 🔴 Highest |
| CLI + Desktop Sync | CLI and Windows desktop (Wails) share `pkg/` core logic | 🔴 Highest |
| General Purpose | Not bound to specific domains; customizable via tools and prompts | 🟡 High |
| Crash Recovery | Resume from nearest checkpoint at any point | 🟡 High |
| Cost Control | Multi-layer budget to prevent infinite token-burning loops | 🟡 High |
| Observable | Complete trace; every step decision is auditable | 🟢 Medium |

### Acknowledgements

Design inspired by [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix). All code is independently implemented.

---

## 2. Theoretical Foundation

Zhulong integrates six systems science theories:

| Theory | Core Application | Status |
|--------|-----------------|--------|
| General Systems Theory (Bertalanffy) | Open systems, alternative paths | ✅ |
| Engineering Cybernetics (Qian Xuesen) | Oscillation/divergence detection | ✅ |
| Information Theory (Shannon) | Information gain, density | ✅ |
| Dissipative Structures (Prigogine) | Stagnation detection, exploration triggers | ✅ |
| Synergetics (Haken) | Order parameter, slaving principle | ✅ |
| Complex Adaptive Systems (Holland) | Building blocks, cognitive models, diversity | ✅ |

---

## 3. Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Controller (FSM)                     │
│  Idle → Planning → Executing → Reflecting → Replanning  │
└─────────────────────────────────────────────────────────┘
        │                │                │
        ▼                ▼                ▼
   ┌─────────┐    ┌──────────┐    ┌───────────┐
   │ Planner │    │ Executor │    │ Reflector │
   └────┬────┘    └────┬─────┘    └─────┬─────┘
        │              │                │
        └──────────────┼────────────────┘
                       │
              ┌────────┴────────┐
              │   Memory (3L)   │
              └────────┬────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
    ┌─────────┐  ┌──────────┐  ┌─────────┐
    │Compressor│  │Checkpoint│  │ Budget  │
    └─────────┘  └──────────┘  └─────────┘
```

### Module Wiring (P0-P3)

All **34 modules** are wired into the agent main loop:

| Priority | Modules Wired | Modules Reserved (P4) |
|----------|--------------|----------------------|
| **P0** Foundation | Controller, Planner, Executor, Reflector, Memory, Compressor, Checkpoint, Budget, Trace, Human, Tools, Provider, DeepSeek, Skills, Approval, Hook | — |
| **P1** Enhancement | Stability, Information (gain), Stagnation, Exploration, Synergetics (order parameter + slaving), Learning (building blocks + diversity) | internal_model, edge_of_chaos, density |
| **P2** Extension | Alternative planner, Environment monitor | — |
| **P3** Options | i18n, Dashboard, Gateway, Models, Plugins, Backup | — |
| **P4** Standalone | — | Health, State, Upgrade, Cache, Loop |

---

## 4. Core Module Design

### 4.1 Controller — FSM-driven Loop

9 explicit states: `Idle → Planning → Executing → Reflecting → {Done | Replanning → Executing} | Error | Cancelled`

### 4.2 Planner

Decomposes user goals into executable step sequences. Supports `Plan()` and `Replan()`.

### 4.3 Executor

Executes individual steps via tool calls (`tool_call`) or LLM generation (`llm_generate`). Tools are registered in `ToolRegistry` with `Call(ctx, params)` signature.

### 4.4 Reflector

Evaluates execution results and decides: `Complete | Continue | Replan | Fail`.

### 4.5 Memory — Three Layers

```
Working Memory (current loop) → Session Memory (current session) → Long-term Memory (persistent)
```

Actual implementation: `internal/memory/store.go` (FileStore) + `internal/memento/store.go` (MEMORY.md + USER.md frozen snapshots)

### 4.6 Context Optimization

- **Compressor** (`internal/compressor/`): Prune stale tool results, skeleton compression
- **Prefix-Cache Iron Rule**: system prompt + skeleton never rewritten; history only appended; pruning only in dynamic interval

---

## 5. Enhancement Modules (P1)

### 5.1 Stability Analyzer (Cybernetics)
Detects A→B→A→B oscillation and divergence patterns. Wired before each Plan phase.

### 5.2 Information Gain (Information Theory)
Estimates information gain of tool calls. Wired before each executor step.

### 5.3 Stagnation Detection (Dissipative Structures)
Detects agent stuck states and triggers exploration when entropy is low.

### 5.4 Exploration Trigger (Dissipative Structures)
Generates exploration actions (increase temperature, try new tools) when stagnation is detected.

### 5.5 Synergetics
- **Order Parameter**: Identifies the driving goal/strategy from session context
- **Slaving Principle**: Filters actions to ensure alignment with order parameter

### 5.6 Adaptive Learning (CAS)
- **Building Blocks**: Auto-saves successful strategies for reuse
- **Diversity Manager**: Monitors tool/strategy usage patterns to avoid over-reliance

---

## 6. Extension Modules (P2-P3)

### P2: General Systems Theory
- **Alternative Planner**: Generates fallback plans when primary plan fails
- **Environment Monitor**: Watches working directory for external changes

### P3: Platform & Ecosystem
- **i18n**: 17-language bundle with default Chinese/English translations
- **Dashboard**: HTTP dashboard on port 4515 for runtime inspection
- **Gateway**: Message gateway with 20+ platform adapters (CLI adapter wired)
- **Models Pool**: Multi-model provider pool for fallback chains
- **Plugins**: Plugin manager with metadata injection into system prompt
- **Backup**: Auto-backup config/docs on successful session completion

---

## 7. Infinite Canvas (Desktop UI)

React Flow-based infinite canvas with 6 node types:
- **PlanNode**: Agent goal and step count
- **StepNode**: Individual step with tool and status
- **TextNode**: Generated text content
- **ImageNode**: Generated images with prompts
- **VideoNode**: Generated videos with duration
- **ConfigNode**: Model and tool configuration

CanvasOps: 7 atomic operations (addNode, removeNode, updateNode, addEdge, removeEdge, setViewport, batch)

---

## 8. Directory Structure

```
zhulong/
├── cmd/zhulong/          # CLI entry
├── pkg/                  # Shared core logic (Agent + types)
│   ├── agent.go          # Main agent loop (all modules wired)
│   └── types.go          # Common types
├── internal/
│   ├── controller/       # P0: FSM + loop control
│   ├── planner/          # P0: Planner + alternative paths
│   ├── executor/         # P0: Tool executor
│   ├── reflector/        # P0: Reflector
│   ├── memory/           # P0: Three-layer memory
│   ├── compressor/       # P0: Context compression
│   ├── checkpoint/       # P0: Checkpoint persistence
│   ├── budget/           # P0: Cost control
│   ├── trace/            # P0: Observability
│   ├── human/            # P0: Human-in-the-loop
│   ├── tools/            # P0: Tool abstraction
│   ├── mcp/              # MCP client
│   ├── provider/         # DeepSeek provider
│   ├── cache/            # Prefix cache
│   ├── security/         # 7-layer security engine
│   ├── terminal/         # Local/Docker/SSH backends
│   ├── stability/        # P1: Stability analyzer
│   ├── information/      # P1: Information theory
│   ├── stagnation/       # P1: Stagnation detection
│   ├── exploration/      # P1: Exploration trigger
│   ├── synergetics/      # P1: Synergetics
│   ├── learning/         # P1: Adaptive learning
│   ├── skills/           # P2: Skill pipeline
│   ├── approval/         # P2: Approval engine
│   ├── environment/      # P2: Environment monitor
│   ├── memento/          # Memory snapshots
│   ├── evolution/        # Self-evolution (reviewer/curator/suggester)
│   ├── board/            # Kanban engine (8 states)
│   ├── blueprint/        # Automation blueprints (7 templates)
│   ├── scheduler/        # Event bus + tick scheduler
│   ├── hook/             # 3-layer hook engine
│   ├── breaker/          # Circuit breaker
│   ├── workflow/         # full/hotfix/tweak modes
│   ├── quality/          # 8-dim GC scanner
│   ├── skillset/         # Pre-set skills registry
│   ├── i18n/             # 17-language bundle
│   ├── dashboard/        # Web dashboard
│   ├── gateway/          # Message gateway
│   ├── models/           # Multi-model pool
│   ├── plugins/          # Plugin manager
│   ├── backup/           # Backup manager
│   ├── profile/          # Profile isolation
│   ├── observe/          # Span tracing
│   ├── review/           # Session review reports
│   ├── health/           # Health checker (P4)
│   ├── state/            # State persistence (P4)
│   ├── upgrade/          # Smart upgrade (P4)
│   ├── loop/             # Autonomous loop engine (P4)
│   ├── cronx/            # Cron scheduler
│   ├── hub/              # Skills Hub
│   ├── acp/              # ACP JSON-RPC server
│   ├── qa/               # QA lab
│   └── voice/            # Voice engine
├── desktop/              # Wails + React desktop frontend
├── config/               # Configuration
├── docs/                 # Documentation
└── examples/             # Usage examples
```

---

## 9. Implementation Status

### Backend Modules: 53/53 ✅

All 53 packages compile and pass tests. 52 packages have unit tests (cmd/zhulong excluded by design).

### Test Coverage

- **Total packages**: 54 (53 internal + 1 pkg)
- **Passing**: 54/54
- **Runtime**: ~5m 7s (full suite), ~15s (short mode)
- **Key modules**: memento, security, quality, scheduler, breaker, evolution, workflow, pkg — all verified

---

## 10. Key Design Decisions

### Cache Hit Rate is the Supreme Law

> All architectural additions, optimizations, and new features must NOT compromise the high cache hit rate.

1. **Prefix only append, never modify**: system prompt + skeleton are never rewritten
2. **History only compress, never reorder**: old loops compressed as summary appended
3. **Pruning only in dynamic interval**: tool result pruning only in current loop

Any module that would break prefix stability must be redesigned or abandoned.

### Security: 7-Layer Defense

1. Container isolation
2. Input sanitization
3. SSRF protection
4. Credential filtering
5. File mutation validation
6. Supply chain audit
7. Hard blacklist (even YOLO mode cannot bypass)

### Approval Engine: 3 Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| `ask` | Ask for every risky operation | Debugging, learning |
| `auto` | Auto-approve low risk, ask for high | Daily use |
| `yolo` | Auto-approve everything | Fully trusted automation |

---

## 11. Getting Started

### Build

```bash
go build -o zhulong ./cmd/zhulong
```

### Run

```bash
# With heuristic provider (no API key needed)
./zhulong run "analyze project code quality"

# With real DeepSeek (set DEEPSEEK_API_KEY)
export DEEPSEEK_API_KEY="sk-..."
./zhulong run "write a Go test for main.go"
```

### Test

```bash
go test -short ./...
```

---

## 12. Version History

| Version | Date | Changes |
|---------|------|---------|
| v0.1 | 2026-06-21 | P0 modules + CLI |
| v0.2 | 2026-06-21 | P1-P2 + canvas + desktop |
| v0.3 | 2026-06-22 | P0-P4 full integration, 34 modules wired, 54 packages tested |
