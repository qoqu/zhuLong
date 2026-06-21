package environment

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChangeType represents the type of change
type ChangeType int

const (
	ChangeModified ChangeType = iota
	ChangeAdded
	ChangeDeleted
)

// String returns the string representation
func (ct ChangeType) String() string {
	switch ct {
	case ChangeModified:
		return "modified"
	case ChangeAdded:
		return "added"
	case ChangeDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// Change represents a file change
type Change struct {
	Path      string
	Type      ChangeType
	Timestamp time.Time
}

// WatcherConfig represents the configuration for a watcher
type WatcherConfig struct {
	Path         string
	Recursive    bool
	IgnorePatterns []string
	PollInterval time.Duration
}

// Monitor monitors external environment changes
type Monitor struct {
	configs []WatcherConfig
	changes chan Change
	running bool
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewMonitor creates a new environment monitor
func NewMonitor() *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		configs: make([]WatcherConfig, 0),
		changes: make(chan Change, 100),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddWatcher adds a watcher configuration
func (m *Monitor) AddWatcher(config WatcherConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs = append(m.configs, config)
}

// Start starts the monitor
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.running = true

	// Start watchers
	for _, config := range m.configs {
		go m.watch(config)
	}

	return nil
}

// Stop stops the monitor
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	m.cancel()
	close(m.changes)
}

// GetChanges returns the changes channel
func (m *Monitor) GetChanges() <-chan Change {
	return m.changes
}

// watch watches a directory for changes
func (m *Monitor) watch(config WatcherConfig) {
	// Track file states
	fileStates := make(map[string]time.Time)

	// Initial scan
	files := m.scanFiles(config)
	for _, file := range files {
		info, err := os.Stat(file)
		if err == nil {
			fileStates[file] = info.ModTime()
		}
	}

	// Poll for changes
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkChanges(config, fileStates)
		}
	}
}

// scanFiles scans a directory for files
func (m *Monitor) scanFiles(config WatcherConfig) []string {
	files := make([]string, 0)

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip ignored patterns
		for _, pattern := range config.IgnorePatterns {
			matched, _ := filepath.Match(pattern, info.Name())
			if matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !info.IsDir() {
			files = append(files, path)
		}

		return nil
	}

	if config.Recursive {
		filepath.Walk(config.Path, walkFunc)
	} else {
		entries, err := os.ReadDir(config.Path)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					// Check ignore patterns
					ignored := false
					for _, pattern := range config.IgnorePatterns {
						matched, _ := filepath.Match(pattern, entry.Name())
						if matched {
							ignored = true
							break
						}
					}
					if !ignored {
						files = append(files, filepath.Join(config.Path, entry.Name()))
					}
				}
			}
		}
	}

	return files
}

// checkChanges checks for file changes
func (m *Monitor) checkChanges(config WatcherConfig, fileStates map[string]time.Time) {
	currentFiles := m.scanFiles(config)
	currentFileSet := make(map[string]bool)

	// Check for new and modified files
	for _, file := range currentFiles {
		currentFileSet[file] = true

		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		lastMod, exists := fileStates[file]
		if !exists {
			// New file
			fileStates[file] = info.ModTime()
			m.sendChange(file, ChangeAdded)
		} else if info.ModTime().After(lastMod) {
			// Modified file
			fileStates[file] = info.ModTime()
			m.sendChange(file, ChangeModified)
		}
	}

	// Check for deleted files
	for file := range fileStates {
		if !currentFileSet[file] {
			delete(fileStates, file)
			m.sendChange(file, ChangeDeleted)
		}
	}
}

// sendChange sends a change to the channel
func (m *Monitor) sendChange(path string, changeType ChangeType) {
	change := Change{
		Path:      path,
		Type:      changeType,
		Timestamp: time.Now(),
	}

	select {
	case m.changes <- change:
	default:
		// Channel full, drop oldest change
		select {
		case <-m.changes:
			m.changes <- change
		default:
		}
	}
}
