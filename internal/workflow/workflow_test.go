package workflow

import (
	"testing"
)

func TestDefaultConfig_Full(t *testing.T) {
	cfg := DefaultConfig(ModeFull)
	if cfg.ApprovalLevel != "strict" {
		t.Errorf("expected strict approval for full mode, got %s", cfg.ApprovalLevel)
	}
	if cfg.CheckIntensity != 1.0 {
		t.Errorf("expected intensity 1.0 for full mode, got %f", cfg.CheckIntensity)
	}
	if cfg.AutoFix {
		t.Error("expected AutoFix=false for full mode")
	}
}

func TestDefaultConfig_Hotfix(t *testing.T) {
	cfg := DefaultConfig(ModeHotfix)
	if cfg.ApprovalLevel != "relaxed" {
		t.Errorf("expected relaxed approval for hotfix, got %s", cfg.ApprovalLevel)
	}
	if cfg.CheckIntensity != 0.3 {
		t.Errorf("expected intensity 0.3 for hotfix, got %f", cfg.CheckIntensity)
	}
	if !cfg.AutoFix {
		t.Error("expected AutoFix=true for hotfix")
	}
}

func TestDefaultConfig_Tweak(t *testing.T) {
	cfg := DefaultConfig(ModeTweak)
	if cfg.ApprovalLevel != "normal" {
		t.Errorf("expected normal approval for tweak, got %s", cfg.ApprovalLevel)
	}
	if cfg.CheckIntensity != 0.5 {
		t.Errorf("expected intensity 0.5 for tweak, got %f", cfg.CheckIntensity)
	}
}

func TestWorkflow_SetMode(t *testing.T) {
	w := New(nil)

	w.SetMode(ModeHotfix)
	if w.config.Mode != ModeHotfix {
		t.Errorf("expected hotfix mode, got %s", w.config.Mode)
	}
}

func TestWorkflow_SetStage(t *testing.T) {
	w := New(DefaultConfig(ModeFull))

	w.SetStage(StageFix)
	if w.config.Stage != StageFix {
		t.Errorf("expected fix stage, got %s", w.config.Stage)
	}
}

func TestWorkflow_String(t *testing.T) {
	w := New(DefaultConfig(ModeFull))
	s := w.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestWorkflow_DefaultIsFull(t *testing.T) {
	w := New(nil)
	if w.config.Mode != ModeFull {
		t.Errorf("expected default mode full, got %s", w.config.Mode)
	}
}

func TestWorkflow_ConfigRetention(t *testing.T) {
	newCfg := DefaultConfig(ModeFull)
	newCfg.ApprovalLevel = "relaxed"
	w2 := New(newCfg)

	if w2.config.ApprovalLevel != "relaxed" {
		t.Errorf("expected relaxed approval, got %s", w2.config.ApprovalLevel)
	}
}
