package learning

import (
	"time"
)

// BuildingBlock represents a reusable strategy unit
type BuildingBlock struct {
	ID          string
	Name        string
	Description string
	Pattern     StrategyPattern
	Context     TaskContext
	SuccessRate float64
	UsageCount  int
	CreatedAt   time.Time
	LastUsedAt  time.Time
}

// StrategyPattern represents the pattern of a strategy
type StrategyPattern struct {
	Steps      []StepTemplate
	ToolsUsed  []string
	KeyInsight string
	Conditions []string
}

// StepTemplate represents a template for a step
type StepTemplate struct {
	Description string
	ActionType  string
	ToolName    string
}

// TaskContext represents the context of a task
type TaskContext struct {
	GoalType   string
	Complexity string
	Domain     string
}

// Instantiate creates a step from the template
func (st *StepTemplate) Instantiate() Step {
	return Step{
		Description: st.Description,
		ActionType:  st.ActionType,
		ToolName:    st.ToolName,
	}
}

// Step represents a step
type Step struct {
	Description string
	ActionType  string
	ToolName    string
}

// BuildingBlockStore stores building blocks
type BuildingBlockStore struct {
	blocks map[string]*BuildingBlock
}

// NewBuildingBlockStore creates a new building block store
func NewBuildingBlockStore() *BuildingBlockStore {
	return &BuildingBlockStore{
		blocks: make(map[string]*BuildingBlock),
	}
}

// SaveBlock saves a building block
func (store *BuildingBlockStore) SaveBlock(block *BuildingBlock) {
	store.blocks[block.ID] = block
}

// GetBlock gets a building block by ID
func (store *BuildingBlockStore) GetBlock(id string) (*BuildingBlock, bool) {
	block, ok := store.blocks[id]
	return block, ok
}

// FindMatchingBlocks finds blocks matching a task context
func (store *BuildingBlockStore) FindMatchingBlocks(taskCtx TaskContext) []*BuildingBlock {
	matches := make([]*BuildingBlock, 0)

	for _, block := range store.blocks {
		if store.isMatching(block.Context, taskCtx) {
			matches = append(matches, block)
		}
	}

	return matches
}

// isMatching checks if two contexts match
func (store *BuildingBlockStore) isMatching(ctx1, ctx2 TaskContext) bool {
	// Simple matching: same goal type or same domain
	return ctx1.GoalType == ctx2.GoalType || ctx1.Domain == ctx2.Domain
}

// GetAllBlocks returns all blocks
func (store *BuildingBlockStore) GetAllBlocks() []*BuildingBlock {
	blocks := make([]*BuildingBlock, 0, len(store.blocks))
	for _, block := range store.blocks {
		blocks = append(blocks, block)
	}
	return blocks
}

// DeleteBlock deletes a block
func (store *BuildingBlockStore) DeleteBlock(id string) {
	delete(store.blocks, id)
}

// UpdateBlockUsage updates the usage count and last used time
func (store *BuildingBlockStore) UpdateBlockUsage(id string) {
	if block, ok := store.blocks[id]; ok {
		block.UsageCount++
		block.LastUsedAt = time.Now()
	}
}
