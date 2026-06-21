# Zhulong（烛龙）Makefile
# 基于 DeepSeek 的通用自主循环 Agent 框架

# 变量定义
BINARY_NAME=zhulong
VERSION=0.1.0
BUILD_DIR=build
GO=go
GOFLAGS=-v

# 默认目标
.PHONY: all
all: clean build test

# 构建
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/zhulong
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test ./... -v

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# 运行基准测试
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GO) test ./... -bench=. -benchmem

# 静态分析
.PHONY: lint
lint:
	@echo "Running linter..."
	$(GO) vet ./...
	@echo "Lint complete"

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Format complete"

# 清理构建产物
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# 安装到本地
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# 运行
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) run "分析当前项目的代码质量"

# 构建桌面端
.PHONY: desktop
desktop:
	@echo "Building desktop app..."
	@cd desktop && $(GO) build $(GOFLAGS) -o ../$(BUILD_DIR)/zhulong-desktop.exe
	@echo "Desktop build complete: $(BUILD_DIR)/zhulong-desktop.exe"

# 运行桌面端
.PHONY: run-desktop
run-desktop: desktop
	@echo "Running desktop app..."
	@./$(BUILD_DIR)/zhulong-desktop.exe

# 生成文档
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@mkdir -p docs/api
	$(GO) doc ./... > docs/api/api.txt
	@echo "Documentation generated: docs/api/api.txt"

# 检查依赖
.PHONY: check-deps
check-deps:
	@echo "Checking dependencies..."
	$(GO) mod tidy
	$(GO) mod verify
	@echo "Dependencies check complete"

# 更新依赖
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "Dependencies updated"

# 显示帮助
.PHONY: help
help:
	@echo "Zhulong（烛龙）- 基于 DeepSeek 的通用自主循环 Agent 框架"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all            - Clean, build, and test"
	@echo "  build          - Build the CLI binary"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  bench          - Run benchmarks"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install binary to GOPATH/bin"
	@echo "  run            - Build and run with example goal"
	@echo "  desktop        - Build desktop app"
	@echo "  run-desktop    - Build and run desktop app"
	@echo "  docs           - Generate documentation"
	@echo "  check-deps     - Check and verify dependencies"
	@echo "  update-deps    - Update dependencies"
	@echo "  help           - Show this help"
