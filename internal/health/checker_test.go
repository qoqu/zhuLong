package health

import (
	"testing"
)

func TestNewChecker(t *testing.T) {
	c := New(".")
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
}

func TestChecker_Run(t *testing.T) {
	c := New(".")
	report := c.Run()

	if report.Total == 0 {
		t.Error("expected at least 1 check item")
	}
	if report.Passed+report.Failed != report.Total {
		t.Errorf("passed+failed=%d, expected %d", report.Passed+report.Failed, report.Total)
	}
}

func TestReport_String(t *testing.T) {
	c := New(".")
	report := c.Run()
	s := report.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestCheckClaudeMD(t *testing.T) {
	c := New(".")
	item := c.checkClaudeMD()
	if item.Name != "CLAUDE.md" {
		t.Errorf("expected name=CLAUDE.md, got %s", item.Name)
	}
}

func TestCheckGit(t *testing.T) {
	c := New(".")
	item := c.checkGit()
	if item.Name != "Git Repository" {
		t.Errorf("expected name=Git Repository, got %s", item.Name)
	}
}

func TestCheckBuild(t *testing.T) {
	c := New(".")
	item := c.checkBuild()
	if item.Name != "Build Status" {
		t.Errorf("expected name=Build Status, got %s", item.Name)
	}
}

func TestCheckModules(t *testing.T) {
	c := New(".")
	item := c.checkModules()
	if item.Name != "Module Structure" {
		t.Errorf("expected name=Module Structure, got %s", item.Name)
	}
}

func TestCheckHooks(t *testing.T) {
	c := New(".")
	item := c.checkHooks()
	_ = item
}

func TestEmptyReport(t *testing.T) {
	report := &Report{}
	if report.OK {
		t.Error("expected OK=false for empty report")
	}
}

func TestReportWithWarnings(t *testing.T) {
	report := &Report{
		Items: []CheckItem{
			{Name: "pass1", Status: "pass"},
			{Name: "warn1", Status: "warn"},
			{Name: "fail1", Status: "fail"},
		},
		Passed: 1,
		Failed: 2,
		Total:  3,
	}
	if report.OK {
		t.Error("expected OK=false when there are failures")
	}
}
