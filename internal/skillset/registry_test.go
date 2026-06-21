package skillset

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestRegisterAndList(t *testing.T) {
	r := New()
	r.Register(InitSkill())

	skills := r.List()
	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}
}

func TestMatch_Init(t *testing.T) {
	r := New()
	r.Register(InitSkill())
	r.Register(ModeSkill())
	r.Register(HelpSkill())

	skill, _, ok := r.Match("帮我初始化项目")
	if !ok {
		t.Fatal("expected match for init")
	}
	if skill.Name != "init" {
		t.Errorf("expected skill name=init, got %s", skill.Name)
	}
}

func TestMatch_Mode(t *testing.T) {
	r := New()
	r.Register(ModeSkill())
	r.Register(HelpSkill())

	skill, args, ok := r.Match("切换模式 hotfix")
	if !ok {
		t.Fatal("expected match for mode")
	}
	if skill.Name != "mode" {
		t.Errorf("expected skill name=mode, got %s", skill.Name)
	}
	if len(args) < 2 || args[1] != "hotfix" {
		t.Errorf("expected args[1]=hotfix, got %v", args)
	}
}

func TestMatch_NoMatch(t *testing.T) {
	r := New()
	r.Register(HelpSkill())

	_, _, ok := r.Match("something completely different")
	if ok {
		t.Error("expected no match")
	}
}

func TestInitSkill(t *testing.T) {
	s := InitSkill()
	result, err := s.Fn(nil)
	if err != nil {
		t.Fatalf("InitSkill failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestModeSkill_Full(t *testing.T) {
	s := ModeSkill()
	result, err := s.Fn([]string{"mode", "full"})
	if err != nil {
		t.Fatalf("ModeSkill failed: %v", err)
	}
	if !contains(result, "完整模式") {
		t.Error("expected 完整模式 in result")
	}
}

func TestModeSkill_Unknown(t *testing.T) {
	s := ModeSkill()
	result, err := s.Fn([]string{"mode", "unknown"})
	if err != nil {
		t.Fatalf("ModeSkill failed: %v", err)
	}
	if !contains(result, "未知模式") {
		t.Error("expected 未知模式 in result")
	}
}

func TestGCSkill(t *testing.T) {
	s := GCSkill()
	result, err := s.Fn(nil)
	if err != nil {
		t.Fatalf("GCSkill failed: %v", err)
	}
	if !contains(result, "GC扫描") {
		t.Error("expected GC扫描 in result")
	}
}

func TestVerifySkill(t *testing.T) {
	s := VerifySkill()
	result, err := s.Fn(nil)
	if err != nil {
		t.Fatalf("VerifySkill failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestHelpSkill(t *testing.T) {
	s := HelpSkill()
	result, _ := s.Fn(nil)
	if !contains(result, "init") {
		t.Error("expected init in help result")
	}
}

func TestMultipleSkills(t *testing.T) {
	r := New()
	r.Register(InitSkill())
	r.Register(ModeSkill())
	r.Register(GCSkill())
	r.Register(VerifySkill())
	r.Register(HelpSkill())

	skills := r.List()
	if len(skills) != 5 {
		t.Errorf("expected 5 skills, got %d", len(skills))
	}

	// 测试多个触发词
	triggers := []string{"初始化", "模式", "扫描", "验证", "帮助"}
	for i, trigger := range triggers {
		_, _, ok := r.Match(trigger)
		if !ok {
			t.Errorf("expected match for trigger %q", trigger)
		}
		_ = i
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
