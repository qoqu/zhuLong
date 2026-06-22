package blueprint

import (
	"testing"
)

func TestNewCatalog(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("expected non-nil catalog")
	}
}

func TestCatalog_List(t *testing.T) {
	c := New()
	list := c.List()
	if len(list) == 0 {
		t.Error("expected at least 1 blueprint")
	}
}

func TestCatalog_Get(t *testing.T) {
	c := New()
	bp, ok := c.Get("morning-brief")
	if !ok {
		t.Fatal("expected morning-brief blueprint")
	}
	if bp.Title != "晨间简报" {
		t.Errorf("expected title=晨间简报, got %s", bp.Title)
	}
}

func TestCatalog_Get_NotFound(t *testing.T) {
	c := New()
	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestCatalog_Register(t *testing.T) {
	c := New()
	c.Register(&Blueprint{
		Key: "custom", Title: "Custom", Category: "general",
		Description: "custom blueprint",
		ScheduleTemplate: "0 9 * * *",
		PromptTemplate:   "run custom task",
	})

	bp, ok := c.Get("custom")
	if !ok {
		t.Fatal("expected custom blueprint")
	}
	if bp.Title != "Custom" {
		t.Errorf("expected title=Custom, got %s", bp.Title)
	}
}

func TestCatalog_Fill(t *testing.T) {
	c := New()
	filled, err := c.Fill("morning-brief", map[string]string{
		"minute": "30",
		"hour":   "8",
		"date":   "2026-06-22",
	})
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}
	if filled.Schedule != "30 8 * * *" {
		t.Errorf("expected schedule=30 8 * * *, got %s", filled.Schedule)
	}
	if !contains(filled.Prompt, "2026-06-22") {
		t.Error("expected date in prompt")
	}
}

func TestCatalog_Fill_MissingRequired(t *testing.T) {
	c := New()
	_, err := c.Fill("report-gen", map[string]string{})
	if err == nil {
		t.Error("expected error for missing required slot")
	}
}

func TestCatalog_Fill_InvalidEnum(t *testing.T) {
	c := New()
	_, err := c.Fill("weekly-review", map[string]string{
		"minute": "0",
		"hour":   "17",
		"dow":    "9",
	})
	if err == nil {
		t.Error("expected error for invalid enum value")
	}
}

func TestCatalog_Fill_NotFount(t *testing.T) {
	c := New()
	_, err := c.Fill("nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent blueprint")
	}
}

func TestCatalog_ListByCategory(t *testing.T) {
	c := New()
	daily := c.ListByCategory("daily")
	if len(daily) == 0 {
		t.Error("expected daily blueprints")
	}
}

func TestParseCron(t *testing.T) {
	fields, err := ParseCron("0 9 * * 1-5")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}
	if fields["minute"] != "0" {
		t.Errorf("expected minute=0, got %s", fields["minute"])
	}
	if fields["hour"] != "9" {
		t.Errorf("expected hour=9, got %s", fields["hour"])
	}
}

func TestParseCron_Invalid(t *testing.T) {
	_, err := ParseCron("invalid")
	if err == nil {
		t.Error("expected error for invalid cron")
	}
}

func TestDescribeSchedule(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0 9 * * *", "每天 09:00"},
		{"30 14 * * 5", "每周五 14:30"},
		{"* * * * *", "每分钟"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			desc := DescribeSchedule(tt.input)
			if desc != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, desc)
			}
		})
	}
}

func TestDefaultBlueprintCount(t *testing.T) {
	c := New()
	list := c.List()
	if len(list) != 7 {
		t.Errorf("expected 8 built-in blueprints, got %d", len(list))
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
