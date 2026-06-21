package compressor

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IncrementalConfig contains configuration for incremental compression
type IncrementalConfig struct {
	// BaseCommit is the base commit for incremental comparison
	BaseCommit string

	// IncludePatterns are patterns to include
	IncludePatterns []string

	// ExcludePatterns are patterns to exclude
	ExcludePatterns []string

	// MaxFileSize is the maximum file size to process
	MaxFileSize int64
}

// DefaultIncrementalConfig returns default incremental configuration
func DefaultIncrementalConfig() *IncrementalConfig {
	return &IncrementalConfig{
		BaseCommit: "HEAD~1",
		IncludePatterns: []string{
			"*.go",
			"*.ts",
			"*.tsx",
			"*.js",
			"*.jsx",
			"*.py",
			"*.md",
		},
		ExcludePatterns: []string{
			".git",
			"node_modules",
			"dist",
			"build",
			"vendor",
			"__pycache__",
		},
		MaxFileSize: 1024 * 1024, // 1MB
	}
}

// FileChange represents a changed file
type FileChange struct {
	// Path is the file path
	Path string

	// Status is the change status (added, modified, deleted)
	Status string

	// OldHash is the old file hash (empty for added files)
	OldHash string

	// NewHash is the new file hash (empty for deleted files)
	NewHash string

	// Size is the file size
	Size int64
}

// IncrementalResult contains the result of incremental compression
type IncrementalResult struct {
	// Changes are the file changes
	Changes []FileChange

	// AddedFiles are the added files
	AddedFiles []string

	// ModifiedFiles are the modified files
	ModifiedFiles []string

	// DeletedFiles are the deleted files
	DeletedFiles []string

	// TotalFiles is the total number of files
	TotalFiles int

	// Skeleton is the incremental skeleton
	Skeleton string
}

// IncrementalCompressor handles incremental compression
type IncrementalCompressor struct {
	config *IncrementalConfig
}

// NewIncrementalCompressor creates a new incremental compressor
func NewIncrementalCompressor(config *IncrementalConfig) *IncrementalCompressor {
	if config == nil {
		config = DefaultIncrementalConfig()
	}

	return &IncrementalCompressor{
		config: config,
	}
}

// CompressIncremental compresses only changed files
func (ic *IncrementalCompressor) CompressIncremental(rootDir string) (*IncrementalResult, error) {
	result := &IncrementalResult{
		Changes:       make([]FileChange, 0),
		AddedFiles:    make([]string, 0),
		ModifiedFiles: make([]string, 0),
		DeletedFiles:  make([]string, 0),
	}

	// Walk the directory and find changed files
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file should be excluded
		if ic.shouldExclude(path) {
			return nil
		}

		// Check if file should be included
		if !ic.shouldInclude(path) {
			return nil
		}

		// Check file size
		if info.Size() > ic.config.MaxFileSize {
			return nil
		}

		// Compute file hash
		hash, err := ic.computeFileHash(path)
		if err != nil {
			return err
		}

		// For now, treat all files as modified
		// In a real implementation, this would compare with the base commit
		relPath, _ := filepath.Rel(rootDir, path)
		change := FileChange{
			Path:    relPath,
			Status:  "modified",
			NewHash: hash,
			Size:    info.Size(),
		}

		result.Changes = append(result.Changes, change)
		result.ModifiedFiles = append(result.ModifiedFiles, relPath)
		result.TotalFiles++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Generate incremental skeleton
	result.Skeleton = ic.generateIncrementalSkeleton(result)

	return result, nil
}

// shouldExclude checks if a file should be excluded
func (ic *IncrementalCompressor) shouldExclude(path string) bool {
	for _, pattern := range ic.config.ExcludePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// shouldInclude checks if a file should be included
func (ic *IncrementalCompressor) shouldInclude(path string) bool {
	if len(ic.config.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range ic.config.IncludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}

	return false
}

// computeFileHash computes the hash of a file
func (ic *IncrementalCompressor) computeFileHash(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash[:8]), nil
}

// generateIncrementalSkeleton generates a skeleton from incremental changes
func (ic *IncrementalCompressor) generateIncrementalSkeleton(result *IncrementalResult) string {
	var sb strings.Builder

	sb.WriteString("# Incremental Changes\n\n")

	sb.WriteString(fmt.Sprintf("Total files changed: %d\n", result.TotalFiles))
	sb.WriteString(fmt.Sprintf("- Added: %d\n", len(result.AddedFiles)))
	sb.WriteString(fmt.Sprintf("- Modified: %d\n", len(result.ModifiedFiles)))
	sb.WriteString(fmt.Sprintf("- Deleted: %d\n", len(result.DeletedFiles)))

	sb.WriteString("\n## Changed Files\n\n")
	for _, change := range result.Changes {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", change.Status, change.Path))
	}

	return sb.String()
}

// GetBaseCommit returns the base commit
func (ic *IncrementalCompressor) GetBaseCommit() string {
	return ic.config.BaseCommit
}

// SetBaseCommit sets the base commit
func (ic *IncrementalCompressor) SetBaseCommit(commit string) {
	ic.config.BaseCommit = commit
}
