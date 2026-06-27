package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a trace entry
type Entry struct {
	Timestamp time.Time   `json:"timestamp"`
	Loop      int         `json:"loop"`
	Phase     string      `json:"phase"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// Logger logs trace entries
type Logger struct {
	sessionID string
	entries   []Entry
	startTime time.Time
	config    *Config
}

// Config contains logger configuration
type Config struct {
	Enabled   bool
	OutputDir string
	Format    string
	Verbose   bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:   true,
		OutputDir: ".zhulong/traces",
		Format:    "json",
		Verbose:   false,
	}
}

// NewLogger creates a new logger
func NewLogger(sessionID string, config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	return &Logger{
		sessionID: sessionID,
		entries:   make([]Entry, 0),
		startTime: time.Now(),
		config:    config,
	}
}

// Start starts the logger
func (l *Logger) Start() {
	l.startTime = time.Now()
	l.Log("system", "started", nil)
}

// Log logs a trace entry
func (l *Logger) Log(phase string, event string, data interface{}) {
	if !l.config.Enabled {
		return
	}

	entry := Entry{
		Timestamp: time.Now(),
		Phase:     phase,
		Event:     event,
		Data:      data,
	}

	l.entries = append(l.entries, entry)

	if l.config.Verbose {
		fmt.Printf("[%s] %s: %s\n", phase, event, formatData(data))
	}
}

// LogWithLoop logs a trace entry with loop number
func (l *Logger) LogWithLoop(loop int, phase string, event string, data interface{}) {
	if !l.config.Enabled {
		return
	}

	entry := Entry{
		Timestamp: time.Now(),
		Loop:      loop,
		Phase:     phase,
		Event:     event,
		Data:      data,
	}

	l.entries = append(l.entries, entry)

	if l.config.Verbose {
		fmt.Printf("[Loop %d][%s] %s: %s\n", loop, phase, event, formatData(data))
	}
}

// LogWithDuration logs a trace entry with duration
func (l *Logger) LogWithDuration(phase string, event string, data interface{}, duration time.Duration) {
	if !l.config.Enabled {
		return
	}

	entry := Entry{
		Timestamp: time.Now(),
		Phase:     phase,
		Event:     event,
		Data:      data,
		Duration:  duration,
	}

	l.entries = append(l.entries, entry)

	if l.config.Verbose {
		fmt.Printf("[%s] %s (%s): %s\n", phase, event, duration.Round(time.Millisecond), formatData(data))
	}
}

// GetEntries returns all trace entries
func (l *Logger) GetEntries() []Entry {
	return l.entries
}

// Save saves the trace to a file
func (l *Logger) Save() error {
	if !l.config.Enabled {
		return nil
	}

	// Create output directory
	if err := os.MkdirAll(l.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build filename
	filename := fmt.Sprintf("%s_%s.%s", l.sessionID, time.Now().Format("20060102_150405"), l.config.Format)
	path := filepath.Join(l.config.OutputDir, filename)

	// Write entries in JSONL format (one JSON object per line)
	var buf []byte
	for _, entry := range l.entries {
		line, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	if err := os.WriteFile(path, buf, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// formatData formats data for display
func formatData(data interface{}) string {
	if data == nil {
		return ""
	}

	switch v := data.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", v)
	}
}
