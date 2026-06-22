package evolution

import (
	"testing"
	"time"
)

func TestNewCurator(t *testing.T) {
	c := NewCurator(nil)
	if c == nil {
		t.Fatal("expected non-nil curator")
	}
}

func TestCurator_Run_Active(t *testing.T) {
	c := NewCurator(&CuratorConfig{
		StaleAfterDays:   30,
		ArchiveAfterDays: 90,
		Interval:         24 * time.Hour,
	})

	skills := []SkillRecord{
		{Name: "active-skill", LastUsed: time.Now(), UseCount: 50, State: StateActive},
	}

	actions := c.Run(skills)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for active skill, got %d", len(actions))
	}
}

func TestCurator_Run_Stale(t *testing.T) {
	c := NewCurator(&CuratorConfig{
		StaleAfterDays:   1,
		ArchiveAfterDays: 90,
	})

	skills := []SkillRecord{
		{Name: "stale-skill", LastUsed: time.Now().Add(-48 * time.Hour), State: StateActive},
	}

	actions := c.Run(skills)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action for stale skill, got %d", len(actions))
	}
	if actions[0].Action != "mark_stale" {
		t.Errorf("expected mark_stale, got %s", actions[0].Action)
	}
}

func TestCurator_Run_Archived(t *testing.T) {
	c := NewCurator(&CuratorConfig{
		StaleAfterDays:   1,
		ArchiveAfterDays: 3,
	})

	skills := []SkillRecord{
		{Name: "old-skill", LastUsed: time.Now().Add(-96 * time.Hour), State: StateActive},
	}

	actions := c.Run(skills)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Action != "mark_archived" {
		t.Errorf("expected mark_archived, got %s", actions[0].Action)
	}
}

func TestCurator_Run_Pinned(t *testing.T) {
	c := NewCurator(&CuratorConfig{
		StaleAfterDays: 1,
	})

	skills := []SkillRecord{
		{Name: "pinned", LastUsed: time.Now().Add(-96 * time.Hour), Pinned: true, State: StateActive},
	}

	actions := c.Run(skills)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for pinned skill, got %d", len(actions))
	}
}

func TestCurator_Consolidate(t *testing.T) {
	c := NewCurator(&CuratorConfig{
		Consolidate: true,
	})

	skills := []SkillRecord{
		{Name: "skill-a", Tags: []string{"coding"}, LastUsed: time.Now()},
		{Name: "skill-b", Tags: []string{"coding"}, LastUsed: time.Now()},
		{Name: "skill-c", Tags: []string{"coding"}, LastUsed: time.Now()},
	}

	actions := c.Run(skills)
	hasConsolidation := false
	for _, a := range actions {
		if a.Action == "suggest_merge" {
			hasConsolidation = true
			break
		}
	}
	if !hasConsolidation {
		t.Error("expected consolidation suggestion")
	}
}

func TestCurator_ShouldRunNow(t *testing.T) {
	c := NewCurator(&CuratorConfig{Interval: 1 * time.Hour})

	if c.ShouldRunNow(time.Now().Add(-2 * time.Hour)) != true {
		t.Error("expected true for 2h ago")
	}
	if c.ShouldRunNow(time.Now()) != false {
		t.Error("expected false for now")
	}
}

func TestCurator_SkipArchiveOnStaleState(t *testing.T) {
	c := NewCurator(&CuratorConfig{
		StaleAfterDays:   1,
		ArchiveAfterDays: 3,
	})

	skills := []SkillRecord{
		{Name: "s", LastUsed: time.Now().Add(-96 * time.Hour), State: StateStale},
	}

	actions := c.Run(skills)
	if len(actions) != 1 {
		t.Errorf("expected 1 action (archive over stale), got %d", len(actions))
	}
}
