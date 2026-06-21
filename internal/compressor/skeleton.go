package compressor

import (
	"fmt"
	"strings"
)

// FocusMode represents the focus mode for skeleton generation
type FocusMode string

const (
	FocusModeFull    FocusMode = "full"    // Full skeleton
	FocusModeTree    FocusMode = "tree"    // Directory tree focus
	FocusModeImports FocusMode = "imports" // Import statements focus
	FocusModeSymbols FocusMode = "symbols" // Symbols (functions, classes) focus
	FocusModeOutline FocusMode = "outline" // Writing outline focus
)

// SkeletonDensity represents the density of the skeleton
type SkeletonDensity string

const (
	SkeletonDensityAdaptive SkeletonDensity = "adaptive" // Adaptive density
	SkeletonDensityStandard SkeletonDensity = "standard" // Standard density
	SkeletonDensityCompact  SkeletonDensity = "compact"  // Compact density
)

// Preset represents a compression preset
type Preset string

const (
	PresetCodebase  Preset = "codebase"  // Codebase preset
	PresetWriting   Preset = "writing"   // Writing preset
	PresetWebsite   Preset = "website"   // Website preset
	PresetEcommerce Preset = "ecommerce" // E-commerce preset
)

// SkeletonConfig contains configuration for skeleton generation
type SkeletonConfig struct {
	// Preset is the compression preset
	Preset Preset

	// FocusMode is the focus mode
	FocusMode FocusMode

	// Density is the skeleton density
	Density SkeletonDensity

	// MaxTokens is the maximum tokens for the skeleton
	MaxTokens int

	// ExcludePatterns are patterns to exclude
	ExcludePatterns []string

	// IncludeHidden includes hidden files
	IncludeHidden bool
}

// DefaultSkeletonConfig returns default skeleton configuration
func DefaultSkeletonConfig() *SkeletonConfig {
	return &SkeletonConfig{
		Preset:    PresetCodebase,
		FocusMode: FocusModeFull,
		Density:   SkeletonDensityAdaptive,
		MaxTokens: 2000,
		ExcludePatterns: []string{
			".git",
			"node_modules",
			"dist",
			"build",
			"__pycache__",
			".venv",
			"vendor",
		},
		IncludeHidden: false,
	}
}

// SkeletonGenerator generates skeletons from content
type SkeletonGenerator struct {
	config *SkeletonConfig
}

// NewSkeletonGenerator creates a new skeleton generator
func NewSkeletonGenerator(config *SkeletonConfig) *SkeletonGenerator {
	if config == nil {
		config = DefaultSkeletonConfig()
	}

	return &SkeletonGenerator{
		config: config,
	}
}

// GenerateSkeleton generates a skeleton from content
func (sg *SkeletonGenerator) GenerateSkeleton(content string) string {
	switch sg.config.Preset {
	case PresetCodebase:
		return sg.generateCodebaseSkeleton(content)
	case PresetWriting:
		return sg.generateWritingSkeleton(content)
	case PresetWebsite:
		return sg.generateWebsiteSkeleton(content)
	case PresetEcommerce:
		return sg.generateEcommerceSkeleton(content)
	default:
		return sg.generateCodebaseSkeleton(content)
	}
}

// generateCodebaseSkeleton generates a skeleton for codebase
func (sg *SkeletonGenerator) generateCodebaseSkeleton(content string) string {
	lines := strings.Split(content, "\n")

	switch sg.config.FocusMode {
	case FocusModeTree:
		return sg.generateTreeSkeleton(lines)
	case FocusModeImports:
		return sg.generateImportsSkeleton(lines)
	case FocusModeSymbols:
		return sg.generateSymbolsSkeleton(lines)
	case FocusModeOutline:
		return sg.generateOutlineSkeleton(lines)
	default:
		return sg.generateFullSkeleton(lines)
	}
}

// generateFullSkeleton generates a full skeleton
func (sg *SkeletonGenerator) generateFullSkeleton(lines []string) string {
	var sb strings.Builder

	sb.WriteString("# Codebase Skeleton\n\n")

	// Count lines
	totalLines := len(lines)
	sb.WriteString(fmt.Sprintf("Total lines: %d\n\n", totalLines))

	// Extract structure
	structure := sg.extractStructure(lines)
	sb.WriteString("## Structure\n\n")
	sb.WriteString(structure)

	// Extract key elements
	keyElements := sg.extractKeyElements(lines)
	sb.WriteString("\n## Key Elements\n\n")
	sb.WriteString(keyElements)

	return sb.String()
}

// generateTreeSkeleton generates a tree-focused skeleton
func (sg *SkeletonGenerator) generateTreeSkeleton(lines []string) string {
	var sb strings.Builder

	sb.WriteString("# Directory Tree\n\n")

	// Extract tree structure
	tree := sg.extractTreeStructure(lines)
	sb.WriteString(tree)

	return sb.String()
}

// generateImportsSkeleton generates an imports-focused skeleton
func (sg *SkeletonGenerator) generateImportsSkeleton(lines []string) string {
	var sb strings.Builder

	sb.WriteString("# Import Statements\n\n")

	// Extract imports
	imports := sg.extractImports(lines)
	sb.WriteString(imports)

	return sb.String()
}

// generateSymbolsSkeleton generates a symbols-focused skeleton
func (sg *SkeletonGenerator) generateSymbolsSkeleton(lines []string) string {
	var sb strings.Builder

	sb.WriteString("# Symbols\n\n")

	// Extract symbols
	symbols := sg.extractSymbols(lines)
	sb.WriteString(symbols)

	return sb.String()
}

// generateOutlineSkeleton generates an outline-focused skeleton
func (sg *SkeletonGenerator) generateOutlineSkeleton(lines []string) string {
	var sb strings.Builder

	sb.WriteString("# Outline\n\n")

	// Extract outline
	outline := sg.extractOutline(lines)
	sb.WriteString(outline)

	return sb.String()
}

// generateWritingSkeleton generates a skeleton for writing
func (sg *SkeletonGenerator) generateWritingSkeleton(content string) string {
	lines := strings.Split(content, "\n")

	var sb strings.Builder
	sb.WriteString("# Writing Skeleton\n\n")

	// Extract chapters
	chapters := sg.extractChapters(lines)
	sb.WriteString("## Chapters\n\n")
	sb.WriteString(chapters)

	// Extract key points
	keyPoints := sg.extractKeyPoints(lines)
	sb.WriteString("\n## Key Points\n\n")
	sb.WriteString(keyPoints)

	return sb.String()
}

// generateWebsiteSkeleton generates a skeleton for website
func (sg *SkeletonGenerator) generateWebsiteSkeleton(content string) string {
	var sb strings.Builder

	sb.WriteString("# Website Skeleton\n\n")
	sb.WriteString("Website structure analysis\n")

	return sb.String()
}

// generateEcommerceSkeleton generates a skeleton for e-commerce
func (sg *SkeletonGenerator) generateEcommerceSkeleton(content string) string {
	var sb strings.Builder

	sb.WriteString("# E-commerce Skeleton\n\n")
	sb.WriteString("E-commerce structure analysis\n")

	return sb.String()
}

// extractStructure extracts the structure from lines
func (sg *SkeletonGenerator) extractStructure(lines []string) string {
	var sb strings.Builder

	// Simple structure extraction
	packages := make(map[string]int)
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			pkg := strings.TrimPrefix(line, "package ")
			packages[pkg]++
		}
	}

	for pkg, count := range packages {
		sb.WriteString(fmt.Sprintf("- %s (%d files)\n", pkg, count))
	}

	return sb.String()
}

// extractKeyElements extracts key elements from lines
func (sg *SkeletonGenerator) extractKeyElements(lines []string) string {
	var sb strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "struct ") {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}

	return sb.String()
}

// extractTreeStructure extracts tree structure from lines
func (sg *SkeletonGenerator) extractTreeStructure(lines []string) string {
	var sb strings.Builder

	// Simple tree extraction
	for _, line := range lines {
		if strings.Contains(line, "/") || strings.Contains(line, "\\") {
			sb.WriteString(fmt.Sprintf("- %s\n", line))
		}
	}

	return sb.String()
}

// extractImports extracts imports from lines
func (sg *SkeletonGenerator) extractImports(lines []string) string {
	var sb strings.Builder

	inImport := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "import (" {
			inImport = true
			continue
		}
		if inImport && trimmed == ")" {
			inImport = false
			continue
		}
		if inImport {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}

	return sb.String()
}

// extractSymbols extracts symbols from lines
func (sg *SkeletonGenerator) extractSymbols(lines []string) string {
	var sb strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "var ") ||
			strings.HasPrefix(trimmed, "const ") {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}

	return sb.String()
}

// extractOutline extracts outline from lines
func (sg *SkeletonGenerator) extractOutline(lines []string) string {
	var sb strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "//") ||
			strings.HasPrefix(trimmed, "/*") {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}

	return sb.String()
}

// extractChapters extracts chapters from lines
func (sg *SkeletonGenerator) extractChapters(lines []string) string {
	var sb strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# Chapter") ||
			strings.HasPrefix(trimmed, "## Chapter") {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}

	return sb.String()
}

// extractKeyPoints extracts key points from lines
func (sg *SkeletonGenerator) extractKeyPoints(lines []string) string {
	var sb strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}

	return sb.String()
}
