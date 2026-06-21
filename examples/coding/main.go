// 代码分析示例
// 展示如何使用 Zhulong Agent 进行代码质量分析
package main

import (
	"fmt"
	"os"

	"github.com/qoqu/zhuLong/pkg"
)

func main() {
	fmt.Println("=== Zhulong 代码分析示例 ===")
	fmt.Println()

	// 创建 Agent
	agent, err := pkg.NewAgent(
		pkg.WithGoal("分析当前项目的代码质量，包括代码结构、潜在问题和改进建议"),
		pkg.WithMaxLoops(10),
		pkg.WithVerbose(true),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 运行 Agent
	fmt.Println("开始分析...")
	result, err := agent.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	fmt.Println()
	fmt.Println("=== 分析结果 ===")
	fmt.Printf("状态: %s\n", result.Status)
	fmt.Printf("分析完成，共执行 %d 个步骤\n", result.Loops)
	fmt.Printf("Token 使用: %d\n", result.TokensUsed)
	fmt.Printf("耗时: %s\n", result.Duration)
	fmt.Println()
	fmt.Println("详细结果:")
	fmt.Println(result.Answer)
}
