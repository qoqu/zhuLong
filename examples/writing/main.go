// 写作示例
// 展示如何使用 Zhulong Agent 进行内容创作
package main

import (
	"fmt"
	"os"

	"github.com/qoqu/zhuLong/pkg"
)

func main() {
	fmt.Println("=== Zhulong 写作示例 ===")
	fmt.Println()

	// 创建 Agent
	agent, err := pkg.NewAgent(
		pkg.WithGoal("撰写一篇关于人工智能在软件开发中应用的技术博客文章，包括引言、主要应用场景、挑战和未来展望"),
		pkg.WithMaxLoops(15),
		pkg.WithVerbose(true),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 运行 Agent
	fmt.Println("开始写作...")
	result, err := agent.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	fmt.Println()
	fmt.Println("=== 写作结果 ===")
	fmt.Printf("状态: %s\n", result.Status)
	fmt.Printf("写作完成，共执行 %d 个步骤\n", result.Loops)
	fmt.Printf("Token 使用: %d\n", result.TokensUsed)
	fmt.Printf("耗时: %s\n", result.Duration)
	fmt.Println()
	fmt.Println("生成的文章:")
	fmt.Println(result.Answer)
}
