# Zhulong（烛龙） · [中文](README.md)

**DeepSeek-powered Autonomous Loop Agent Framework**

[![Go Version](https://img.shields.io/badge/Go-1.26.3-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## Overview

Zhulong (烛龙, "Torch Dragon") is a DeepSeek-powered autonomous loop agent framework written in Go. It drives a Plan → Execute → Reflect → Replan loop through a finite-state machine, without per-round manual triggers.

### Core Features

- **Autonomous Multi-round Loop**: Plan → Execute → Reflect → Replan
- **High Cache Hit Rate**: DeepSeek prefix-cache optimized; append-only context layout
- **Adaptive Learning**: Strategy reuse, cognitive model updates, explore/exploit balance
- **Production-grade Reliability**: Checkpoint recovery, cost control, human-in-the-loop, full observability
- **Complete Desktop UI**: 6 right tabs + 27 modules live status + Web Dashboard + plugin system

### Design Goals

| Goal | Description | Priority |
|------|-------------|----------|
| **High Cache Hit Rate** | DeepSeek prefix-cache optimization | 🔴 Highest |
| CLI + Desktop Sync | CLI and Windows desktop share `pkg/` core | 🔴 Highest |
| General Purpose | Not domain-bound; customizable via tools/prompts | 🟡 High |
| Adaptive Learning | Strategy reuse, exploration/exploitation balance | 🟡 High |
| Crash Recovery | Resume from nearest checkpoint | 🟡 High |
| Cost Control | Multi-layer budget; no infinite token burning | 🟡 High |
| Observable | Complete trace; every decision auditable | 🟢 Medium |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Controller (FSM)                        │
│  Idle → Planning → Executing → Reflecting → {Done|Replan}  │
└─────────────────────────────────────────────────────────────┘
        │              │               │
        ▼              ▼               ▼
   ┌─────────┐   ┌──────────┐   ┌───────────┐
   │ Planner │   │ Executor │   │ Reflector │
   └────┬────┘   └────┬─────┘   └─────┬─────┘
        │              │               │
        └──────────────┼───────────────┘
                       │
              ┌────────┴────────┐
              │   Memory (3L)   │
              └────────┬────────┘
                       │
         ┌─────────────┼─────────────┐
    ┌─────────┐  ┌──────────┐  ┌─────────┐
    │Compressor│  │Checkpoint│  │ Budget  │
    └─────────┘  └──────────┘  └─────────┘
```

### Module Wiring

**34 modules wired into the agent main loop**:

| Priority | Modules |
|----------|---------|
| **P0** Core | Controller, Planner, Executor, Reflector, Memory, Compressor, Checkpoint, Budget, Trace, Human, Tools, Provider, DeepSeek, Skills, Approval, Hook |
| **P1** Enhancement | Stability, Information (gain), Stagnation, Exploration, Synergetics (order parameter + slaving), Learning (building blocks + diversity) |
| **P2** Extension | Alternative planner, Environment monitor |
| **P3** Platform | i18n, Dashboard, Gateway, Models, Plugins, Backup |

See [design.en.md](docs/design.en.md) for full design documentation.

## Quick Start

### Build

```bash
go build -o zhulong ./cmd/zhulong
```

### Run (heuristic mode, no API key needed)

```bash
./zhulong run "analyze project code quality"
```

### Run (with DeepSeek API)

```bash
export DEEPSEEK_API_KEY="sk-..."
./zhulong run "write a Go test for the main module"
```

### Test

```bash
go test -count=1 -short ./...
```

## Desktop (Wails)

```bash
cd desktop
wails build
./build/bin/zhulong.exe
```

#### Desktop Features

- **6 right tabs**: Overview / Files / Changes / Memory / Learning / Modules
- **3-tier memory visualization**: Episodic / Semantic / Procedural
- **27 modules live status**: P0×12 + P1×6 + P2×4 + P3×5 all in real-time
- **Execution modes**: ask / auto / yolo
- **Input modes**: normal / plan / goal
- **Temperature control**: auto / 0.0 / 0.3 / 0.7 / 1.0
- **File tree**: Real-time workspace watching (depth 2, skipping node_modules/.git)
- **Approval modal**: For risky operations (ask/auto mode)
- **Web Dashboard**: Browser-based real-time monitor (port 7788)
- **Auto backup**: off / immediate / on-completion
- **i18n**: Chinese / English real-time switching

#### Desktop Settings

Sidebar → Settings configures:
- **Monitor path**: envMonitor watch directory
- **Backup mode**: off / immediate / on-completion + backup-on-failure toggle
- **Web Dashboard port**: Custom port (start/stop)
- **Manual backup**: Create backup snapshot now
- **Language**: Chinese / English

## Web Dashboard

After enabling in desktop settings, browse to `http://localhost:7788` (default port) to see:

- **System status card**: service / port / uptime / version
- **Active Sessions table**: id / goal / status / tokens
- **Modules table**: controller / planner / executor / 10 core modules

Native JS frontend, 3-second polling. HTML embedded via `//go:embed`, no external file dependencies.

## Plugin System

Zhulong supports Go plugin dynamic loading:

```go
// 1. Write a plugin that exports NewPlugin
package main

import "github.com/qoqu/zhuLong/internal/plugins"

type MyPlugin struct{}

func (p *MyPlugin) Name() string    { return "my-plugin" }
func (p *MyPlugin) Version() string { return "0.1.0" }
func (p *MyPlugin) Init() error     { return nil }
func (p *MyPlugin) Shutdown() error { return nil }

func NewPlugin() plugins.Plugin { return &MyPlugin{} }
```

```bash
# 2. Build
go build -buildmode=plugin -o my-plugin.so my-plugin.go

# 3. Place in plugins directory
cp my-plugin.so $TMP/zhulong-plugins/

# 4. Desktop settings → Change Plugins dir → Re-scan
```

Supports `.so` (Linux) / `.dll` (Windows) / `.dylib` (macOS).go test -short ./...
# 54/54 packages passing
```

## Theoretical Foundation

Zhulong integrates six systems science theories:

| Theory | Application |
|--------|------------|
| General Systems Theory (Bertalanffy) | Open systems, alternative paths |
| Engineering Cybernetics (Qian Xuesen) | Oscillation/divergence detection |
| Information Theory (Shannon) | Information gain, density |
| Dissipative Structures (Prigogine) | Stagnation detection, exploration |
| Synergetics (Haken) | Order parameter, slaving principle |
| Complex Adaptive Systems (Holland) | Building blocks, diversity, edge of chaos |

## Key Design Principles

### Cache Hit Rate is Supreme Law

1. **Prefix only append, never modify**: system prompt + skeleton never rewritten
2. **History only compress, never reorder**: old loops compressed as summary
3. **Pruning only in dynamic interval**: only current loop tool results pruned

### Security: 7-Layer Defense

Container → Input Sanitization → SSRF → Credential Filter → File Mutation → Supply Chain → Hard Blacklist

### Approval Engine: 3 Modes

| Mode | Behavior |
|------|----------|
| `ask` | Ask for every risky operation |
| `auto` | Auto-approve low risk, ask for high |
| `yolo` | Auto-approve everything |

## Project Structure

```
zhulong/
├── cmd/zhulong/          # CLI entry
├── pkg/                  # Shared core logic (Agent + types)
├── internal/             # 53 packages (see docs/design.en.md)
├── desktop/              # Wails + React desktop frontend
├── config/               # Configuration
├── docs/                 # Documentation (Chinese + English)
└── examples/             # Usage examples
```

## Infinite Canvas (Desktop)

React Flow-based canvas with 6 node types (Plan, Step, Text, Image, Video, Config) plus asset management, versioning, export, collaboration, offline support, and AI enhancement.

## Implementation Status

- **Backend modules**: 53/53 ✅
- **Packages with tests**: 52/53
- **Tests passing**: 54/54
- **Build**: `go build ./...` ✅
- **Vet**: `go vet ./...` ✅
- **Desktop UI ↔ backend connectivity**: 100% ✅
- **Web Dashboard**: running, real-time monitor
- **Plugin system**: real `plugin.Open` loader

## Documentation

- [Full Design Document (中文)](docs/design.md)
- [Design Overview (English)](docs/design.en.md)
- [Contributing Guide (中文)](CONTRIBUTING.md)

## Acknowledgements

Design inspired by [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix). All code independently implemented.

## License

MIT
