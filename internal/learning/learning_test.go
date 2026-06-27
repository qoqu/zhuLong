package learning

import (
	"testing"
	"time"
)

// Building Block Tests

func TestNewBuildingBlockStore(t *testing.T) {
	store := NewBuildingBlockStore()

	if store == nil {
		t.Error("NewBuildingBlockStore() should not return nil")
	}
}

func TestBuildingBlockStore_SaveAndGet(t *testing.T) {
	store := NewBuildingBlockStore()

	block := &BuildingBlock{
		ID:          "block-1",
		Name:        "Test Block",
		Description: "A test block",
		SuccessRate: 0.9,
		UsageCount:  0,
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
	}

	store.SaveBlock(block)

	got, ok := store.GetBlock("block-1")
	if !ok {
		t.Error("GetBlock() should find the block")
	}
	if got.ID != block.ID {
		t.Errorf("GetBlock().ID = %v, want %v", got.ID, block.ID)
	}
}

func TestBuildingBlockStore_FindMatchingBlocks(t *testing.T) {
	store := NewBuildingBlockStore()

	store.SaveBlock(&BuildingBlock{
		ID:   "block-1",
		Name: "Block 1",
		Context: TaskContext{
			GoalType: "file_operation",
			Domain:   "coding",
		},
	})

	store.SaveBlock(&BuildingBlock{
		ID:   "block-2",
		Name: "Block 2",
		Context: TaskContext{
			GoalType: "web_search",
			Domain:   "research",
		},
	})

	// Search for file operations
	matches := store.FindMatchingBlocks(TaskContext{
		GoalType: "file_operation",
		Domain:   "coding",
	})

	if len(matches) != 1 {
		t.Errorf("FindMatchingBlocks() returned %v matches, want 1", len(matches))
	}
}

func TestBuildingBlockStore_UpdateBlockUsage(t *testing.T) {
	store := NewBuildingBlockStore()

	block := &BuildingBlock{
		ID:         "block-1",
		UsageCount: 0,
	}

	store.SaveBlock(block)
	store.UpdateBlockUsage("block-1")

	got, _ := store.GetBlock("block-1")
	if got.UsageCount != 1 {
		t.Errorf("UsageCount = %v, want 1", got.UsageCount)
	}
}

// Internal Model Tests

func TestNewInternalModel(t *testing.T) {
	model := NewInternalModel()

	if model == nil {
		t.Error("NewInternalModel() should not return nil")
	}
	if model.WorldModel.KnownFacts == nil {
		t.Error("WorldModel.KnownFacts should not be nil")
	}
}

func TestInternalModel_AddFact(t *testing.T) {
	model := NewInternalModel()

	model.AddFact("key1", "value1")

	value, ok := model.GetFact("key1")
	if !ok {
		t.Error("GetFact() should find the fact")
	}
	if value != "value1" {
		t.Errorf("GetFact() = %v, want 'value1'", value)
	}
}

func TestInternalModel_AddAssumption(t *testing.T) {
	model := NewInternalModel()

	model.AddAssumption("assumption1", 0.8)

	confidence, ok := model.GetAssumption("assumption1")
	if !ok {
		t.Error("GetAssumption() should find the assumption")
	}
	if confidence != 0.8 {
		t.Errorf("GetAssumption() = %v, want 0.8", confidence)
	}
}

func TestInternalModel_AddTaskPattern(t *testing.T) {
	model := NewInternalModel()

	pattern := TaskPattern{
		Name:        "pattern1",
		Steps:       []string{"step1", "step2"},
		SuccessRate: 0.9,
	}

	model.AddTaskPattern(pattern)

	got, ok := model.GetTaskPattern("pattern1")
	if !ok {
		t.Error("GetTaskPattern() should find the pattern")
	}
	if got.Name != pattern.Name {
		t.Errorf("GetTaskPattern().Name = %v, want %v", got.Name, pattern.Name)
	}
}

func TestInternalModel_GetToolReliability(t *testing.T) {
	model := NewInternalModel()

	// Default reliability
	reliability := model.GetToolReliability("unknown_tool")
	if reliability != 0.5 {
		t.Errorf("Default reliability = %v, want 0.5", reliability)
	}

	// Updated reliability
	model.UpdateToolReliability("read_file", 0.9)
	reliability = model.GetToolReliability("read_file")
	if reliability != 0.9 {
		t.Errorf("Updated reliability = %v, want 0.9", reliability)
	}
}

// Diversity Tests

func TestNewDiversityManager(t *testing.T) {
	dm := NewDiversityManager(1.0)

	if dm == nil {
		t.Error("NewDiversityManager() should not return nil")
	}
	if dm.threshold != 1.0 {
		t.Errorf("DiversityManager.threshold = %v, want 1.0", dm.threshold)
	}
}

func TestDiversityManager_CheckDiversity(t *testing.T) {
	dm := NewDiversityManager(1.0)

	// No usage
	report := dm.CheckDiversity()
	if report.IsHealthy {
		t.Error("Should not be healthy with no usage")
	}

	// Add diverse usage
	dm.RecordToolUsage("read_file")
	dm.RecordToolUsage("write_file")
	dm.RecordToolUsage("search_file")
	dm.RecordStrategyUsage("strategy1")
	dm.RecordStrategyUsage("strategy2")

	report = dm.CheckDiversity()
	// With diverse usage, diversity should be higher
	if report.ToolDiversity <= 0 {
		t.Errorf("ToolDiversity = %v, should be > 0", report.ToolDiversity)
	}
}

func TestDiversityManager_GetUnderusedTools(t *testing.T) {
	dm := NewDiversityManager(1.0)

	dm.RecordToolUsage("read_file")
	dm.RecordToolUsage("read_file")
	dm.RecordToolUsage("write_file")

	report := dm.CheckDiversity()
	if len(report.Recommendations) == 0 {
		t.Error("Should have recommendations for underused tools")
	}
}

// Edge of Chaos Tests

func TestNewEdgeOfChaos(t *testing.T) {
	eoc := NewEdgeOfChaos(0.7, 1.5)

	if eoc == nil {
		t.Error("NewEdgeOfChaos() should not return nil")
	}
	if eoc.baseTemperature != 0.7 {
		t.Errorf("EdgeOfChaos.baseTemperature = %v, want 0.7", eoc.baseTemperature)
	}
}

func TestEdgeOfChaos_CalculateBalance(t *testing.T) {
	eoc := NewEdgeOfChaos(0.7, 1.5)

	// High success rate → more exploitation (lower balance)
	balance := eoc.CalculateBalance(0.9)
	if balance >= 0.5 {
		t.Errorf("Balance with high success = %v, should be < 0.5 (more exploitation)", balance)
	}

	// Low success rate → more exploration (higher balance)
	balance = eoc.CalculateBalance(0.3)
	if balance <= 0.5 {
		t.Errorf("Balance with low success = %v, should be > 0.5 (more exploration)", balance)
	}
}

func TestEdgeOfChaos_GetTemperature(t *testing.T) {
	eoc := NewEdgeOfChaos(0.7, 1.5)

	eoc.SetBalance(0.5)
	temp := eoc.GetTemperature()

	if temp < 0.7 || temp > 1.5 {
		t.Errorf("Temperature = %v, should be between 0.7 and 1.5", temp)
	}
}

func TestEdgeOfChaos_SetBalance(t *testing.T) {
	eoc := NewEdgeOfChaos(0.7, 1.5)

	eoc.SetBalance(0.8)
	if eoc.GetBalance() != 0.8 {
		t.Errorf("Balance = %v, want 0.8", eoc.GetBalance())
	}

	// Test clamping
	eoc.SetBalance(1.5)
	if eoc.GetBalance() != 1.0 {
		t.Errorf("Balance = %v, should be clamped to 1.0", eoc.GetBalance())
	}

	eoc.SetBalance(-0.5)
	if eoc.GetBalance() != 0.0 {
		t.Errorf("Balance = %v, should be clamped to 0.0", eoc.GetBalance())
	}
}
