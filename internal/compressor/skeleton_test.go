package compressor

import (
	"testing"
)

func TestNewSkeletonGenerator(t *testing.T) {
	gen := NewSkeletonGenerator(nil)
	if gen == nil {
		t.Fatal("NewSkeletonGenerator returned nil")
	}
	if gen.config == nil {
		t.Fatal("config is nil")
	}
}

func TestDefaultSkeletonConfig(t *testing.T) {
	config := DefaultSkeletonConfig()
	if config == nil {
		t.Fatal("DefaultSkeletonConfig returned nil")
	}
	if config.Preset != PresetCodebase {
		t.Errorf("expected preset codebase, got %s", config.Preset)
	}
	if config.FocusMode != FocusModeFull {
		t.Errorf("expected focus mode full, got %s", config.FocusMode)
	}
	if config.Density != SkeletonDensityAdaptive {
		t.Errorf("expected density adaptive, got %s", config.Density)
	}
	if config.MaxTokens != 2000 {
		t.Errorf("expected max tokens 2000, got %d", config.MaxTokens)
	}
}

func TestSkeletonGeneratorGenerateSkeleton(t *testing.T) {
	gen := NewSkeletonGenerator(nil)

	content := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
}

type Config struct {
	Name string
}

func (c *Config) GetName() string {
	return c.Name
}
`

	skeleton := gen.GenerateSkeleton(content)
	if skeleton == "" {
		t.Error("expected non-empty skeleton")
	}

	// Check that skeleton contains expected elements
	if !containsString(skeleton, "Codebase Skeleton") {
		t.Error("expected skeleton to contain 'Codebase Skeleton'")
	}
}

func TestSkeletonGeneratorFocusModes(t *testing.T) {
	content := `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello")
}

type MyStruct struct {
	Name string
}
`

	tests := []struct {
		name      string
		focusMode FocusMode
		contains  string
	}{
		{"full", FocusModeFull, "Codebase Skeleton"},
		{"tree", FocusModeTree, "Directory Tree"},
		{"imports", FocusModeImports, "Import Statements"},
		{"symbols", FocusModeSymbols, "Symbols"},
		{"outline", FocusModeOutline, "Outline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SkeletonConfig{
				Preset:    PresetCodebase,
				FocusMode: tt.focusMode,
				Density:   SkeletonDensityAdaptive,
				MaxTokens: 2000,
			}
			gen := NewSkeletonGenerator(config)

			skeleton := gen.GenerateSkeleton(content)
			if !containsString(skeleton, tt.contains) {
				t.Errorf("expected skeleton to contain '%s'", tt.contains)
			}
		})
	}
}

func TestSkeletonGeneratorPresets(t *testing.T) {
	content := "Test content"

	tests := []struct {
		name     string
		preset   Preset
		contains string
	}{
		{"codebase", PresetCodebase, "Codebase Skeleton"},
		{"writing", PresetWriting, "Writing Skeleton"},
		{"website", PresetWebsite, "Website Skeleton"},
		{"ecommerce", PresetEcommerce, "E-commerce Skeleton"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SkeletonConfig{
				Preset:    tt.preset,
				FocusMode: FocusModeFull,
				Density:   SkeletonDensityAdaptive,
				MaxTokens: 2000,
			}
			gen := NewSkeletonGenerator(config)

			skeleton := gen.GenerateSkeleton(content)
			if !containsString(skeleton, tt.contains) {
				t.Errorf("expected skeleton to contain '%s'", tt.contains)
			}
		})
	}
}

func TestSkeletonDensity(t *testing.T) {
	tests := []struct {
		name     string
		density  SkeletonDensity
		expected string
	}{
		{"adaptive", SkeletonDensityAdaptive, "adaptive"},
		{"standard", SkeletonDensityStandard, "standard"},
		{"compact", SkeletonDensityCompact, "compact"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.density) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.density))
			}
		})
	}
}

func TestFocusMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     FocusMode
		expected string
	}{
		{"full", FocusModeFull, "full"},
		{"tree", FocusModeTree, "tree"},
		{"imports", FocusModeImports, "imports"},
		{"symbols", FocusModeSymbols, "symbols"},
		{"outline", FocusModeOutline, "outline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.mode))
			}
		})
	}
}

func TestPreset(t *testing.T) {
	tests := []struct {
		name     string
		preset   Preset
		expected string
	}{
		{"codebase", PresetCodebase, "codebase"},
		{"writing", PresetWriting, "writing"},
		{"website", PresetWebsite, "website"},
		{"ecommerce", PresetEcommerce, "ecommerce"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.preset) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.preset))
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || containsString(s[1:], substr)))
}
