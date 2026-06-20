package human

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// BreakpointType represents the type of breakpoint
type BreakpointType int

const (
	BreakpointOnError     BreakpointType = iota
	BreakpointOnReplan
	BreakpointCostExceed
	BreakpointCustom
)

// Breakpoint represents a breakpoint
type Breakpoint struct {
	Type    BreakpointType
	Message string
}

// HumanInput represents input from a human
type HumanInput struct {
	Action  string
	Content string
}

// Manager manages human-in-the-loop interactions
type Manager struct {
	config     *Config
	breakpoints []Breakpoint
	reader     *bufio.Reader
}

// Config contains human manager configuration
type Config struct {
	Enabled       bool
	Interactive   bool
	Breakpoints   []BreakpointType
}

// DefaultConfig returns default human manager configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		Interactive: true,
		Breakpoints: []BreakpointType{
			BreakpointOnError,
			BreakpointOnReplan,
		},
	}
}

// NewManager creates a new human manager
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	return &Manager{
		config:      config,
		breakpoints: make([]Breakpoint, 0),
		reader:      bufio.NewReader(os.Stdin),
	}
}

// ShouldPause returns true if the agent should pause for human input
func (m *Manager) ShouldPause(event string, data interface{}) bool {
	if !m.config.Enabled {
		return false
	}

	// Check if the event matches any breakpoint
	for _, bp := range m.config.Breakpoints {
		switch bp {
		case BreakpointOnError:
			if event == "error" {
				return true
			}
		case BreakpointOnReplan:
			if event == "replan" {
				return true
			}
		case BreakpointCostExceed:
			if event == "cost_exceed" {
				return true
			}
		}
	}

	return false
}

// WaitForInput waits for human input
func (m *Manager) WaitForInput(prompt string) (*HumanInput, error) {
	if !m.config.Interactive {
		// Non-interactive mode: auto-continue
		return &HumanInput{
			Action:  "continue",
			Content: "",
		}, nil
	}

	fmt.Println("\n" + prompt)
	fmt.Println("Options:")
	fmt.Println("  [c] Continue - Continue execution")
	fmt.Println("  [a] Abort    - Abort execution")
	fmt.Println("  [m] Modify   - Modify the plan")
	fmt.Print("\nYour choice: ")

	input, err := m.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "c", "continue", "":
		return &HumanInput{
			Action:  "continue",
			Content: "",
		}, nil
	case "a", "abort":
		return &HumanInput{
			Action:  "abort",
			Content: "",
		}, nil
	case "m", "modify":
		fmt.Print("Enter your modification: ")
		modification, err := m.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read modification: %w", err)
		}
		return &HumanInput{
			Action:  "modify",
			Content: strings.TrimSpace(modification),
		}, nil
	default:
		return &HumanInput{
			Action:  "continue",
			Content: "",
		}, nil
	}
}

// Confirm asks for confirmation
func (m *Manager) Confirm(prompt string) (bool, error) {
	if !m.config.Interactive {
		return true, nil
	}

	fmt.Printf("%s [y/N]: ", prompt)

	input, err := m.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

// DisplayStatus displays the current status
func (m *Manager) DisplayStatus(status string) {
	if m.config.Interactive {
		fmt.Printf("\n[Status] %s\n", status)
	}
}

// DisplayProgress displays progress
func (m *Manager) DisplayProgress(current int, total int, message string) {
	if m.config.Interactive {
		fmt.Printf("\r[Progress] %d/%d - %s", current, total, message)
	}
}
