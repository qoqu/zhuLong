// QA Lab - 集成测试框架（参考OpenClaw qa/）
package qa

import (
	"fmt"
	"strings"
	"time"
)

// TestCase 测试用例
type TestCase struct {
	Name     string
	Input    string
	Expected string
	Setup    func() error
	Run      func(input string) (string, error)
	Teardown func() error
}

// Result 测试结果
type Result struct {
	Name      string
	Passed    bool
	Duration  time.Duration
	Actual    string
	Error     string
}

// Lab QA实验室
type Lab struct {
	cases []TestCase
}

func NewLab() *Lab { return &Lab{} }

func (l *Lab) Add(tc TestCase) { l.cases = append(l.cases, tc) }

// RunAll 运行所有测试
func (l *Lab) RunAll() []Result {
	var results []Result
	for _, tc := range l.cases {
		results = append(results, l.run(tc))
	}
	return results
}

func (l *Lab) run(tc TestCase) Result {
	start := time.Now()

	if tc.Setup != nil {
		if err := tc.Setup(); err != nil {
			return Result{Name: tc.Name, Passed: false, Error: fmt.Sprintf("setup: %v", err)}
		}
	}

	actual, err := tc.Run(tc.Input)
	elapsed := time.Since(start)

	if tc.Teardown != nil {
		tc.Teardown()
	}

	if err != nil {
		return Result{Name: tc.Name, Passed: false, Duration: elapsed, Error: err.Error()}
	}

	passed := strings.Contains(actual, tc.Expected)
	return Result{
		Name: tc.Name, Passed: passed,
		Duration: elapsed, Actual: actual,
	}
}

// Summary 汇总报告
func Summary(results []Result) string {
	passed, failed := 0, 0
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	return fmt.Sprintf("QA Lab: %d/%d passed, %d failed", passed, len(results), failed)
}
