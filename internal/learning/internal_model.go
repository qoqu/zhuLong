package learning

import (
	"time"
)

// InternalModel represents the agent's cognitive model
type InternalModel struct {
	WorldModel WorldModel
	TaskModel  TaskModel
	ToolModel  ToolModel
	Updates    []ModelUpdate
}

// WorldModel represents the agent's understanding of the world
type WorldModel struct {
	KnownFacts  map[string]string
	Assumptions map[string]float64
	LastUpdated time.Time
}

// TaskModel represents the agent's understanding of tasks
type TaskModel struct {
	TaskPatterns map[string]TaskPattern
	Strategies   map[string]Strategy
	LastUpdated  time.Time
}

// ToolModel represents the agent's understanding of tools
type ToolModel struct {
	ToolCapabilities map[string]ToolCapability
	ToolReliability  map[string]float64
	LastUpdated      time.Time
}

// TaskPattern represents a pattern in tasks
type TaskPattern struct {
	Name       string
	Steps      []string
	SuccessRate float64
}

// Strategy represents a strategy
type Strategy struct {
	Name       string
	Steps      []string
	SuccessRate float64
}

// ToolCapability represents a tool's capability
type ToolCapability struct {
	ToolName    string
	Description string
	SupportedOps []string
}

// ModelUpdate represents an update to the model
type ModelUpdate struct {
	Timestamp  time.Time
	UpdateType string
	Details    string
}

// NewInternalModel creates a new internal model
func NewInternalModel() *InternalModel {
	return &InternalModel{
		WorldModel: WorldModel{
			KnownFacts:  make(map[string]string),
			Assumptions: make(map[string]float64),
			LastUpdated: time.Now(),
		},
		TaskModel: TaskModel{
			TaskPatterns: make(map[string]TaskPattern),
			Strategies:   make(map[string]Strategy),
			LastUpdated:  time.Now(),
		},
		ToolModel: ToolModel{
			ToolCapabilities: make(map[string]ToolCapability),
			ToolReliability:  make(map[string]float64),
			LastUpdated:      time.Now(),
		},
		Updates: make([]ModelUpdate, 0),
	}
}

// AddFact adds a fact to the world model
func (im *InternalModel) AddFact(key string, value string) {
	im.WorldModel.KnownFacts[key] = value
	im.WorldModel.LastUpdated = time.Now()
	im.addUpdate("fact_added", "Added fact: "+key)
}

// AddAssumption adds an assumption to the world model
func (im *InternalModel) AddAssumption(key string, confidence float64) {
	im.WorldModel.Assumptions[key] = confidence
	im.WorldModel.LastUpdated = time.Now()
	im.addUpdate("assumption_added", "Added assumption: "+key)
}

// AddTaskPattern adds a task pattern
func (im *InternalModel) AddTaskPattern(pattern TaskPattern) {
	im.TaskModel.TaskPatterns[pattern.Name] = pattern
	im.TaskModel.LastUpdated = time.Now()
	im.addUpdate("pattern_added", "Added task pattern: "+pattern.Name)
}

// AddStrategy adds a strategy
func (im *InternalModel) AddStrategy(strategy Strategy) {
	im.TaskModel.Strategies[strategy.Name] = strategy
	im.TaskModel.LastUpdated = time.Now()
	im.addUpdate("strategy_added", "Added strategy: "+strategy.Name)
}

// AddToolCapability adds a tool capability
func (im *InternalModel) AddToolCapability(capability ToolCapability) {
	im.ToolModel.ToolCapabilities[capability.ToolName] = capability
	im.ToolModel.LastUpdated = time.Now()
	im.addUpdate("capability_added", "Added tool capability: "+capability.ToolName)
}

// UpdateToolReliability updates tool reliability
func (im *InternalModel) UpdateToolReliability(toolName string, reliability float64) {
	im.ToolModel.ToolReliability[toolName] = reliability
	im.ToolModel.LastUpdated = time.Now()
	im.addUpdate("reliability_updated", "Updated tool reliability: "+toolName)
}

// GetFact gets a fact from the world model
func (im *InternalModel) GetFact(key string) (string, bool) {
	value, ok := im.WorldModel.KnownFacts[key]
	return value, ok
}

// GetAssumption gets an assumption from the world model
func (im *InternalModel) GetAssumption(key string) (float64, bool) {
	value, ok := im.WorldModel.Assumptions[key]
	return value, ok
}

// GetTaskPattern gets a task pattern
func (im *InternalModel) GetTaskPattern(name string) (TaskPattern, bool) {
	pattern, ok := im.TaskModel.TaskPatterns[name]
	return pattern, ok
}

// GetStrategy gets a strategy
func (im *InternalModel) GetStrategy(name string) (Strategy, bool) {
	strategy, ok := im.TaskModel.Strategies[name]
	return strategy, ok
}

// GetToolCapability gets a tool capability
func (im *InternalModel) GetToolCapability(toolName string) (ToolCapability, bool) {
	capability, ok := im.ToolModel.ToolCapabilities[toolName]
	return capability, ok
}

// GetToolReliability gets tool reliability
func (im *InternalModel) GetToolReliability(toolName string) float64 {
	reliability, ok := im.ToolModel.ToolReliability[toolName]
	if !ok {
		return 0.5 // Default reliability
	}
	return reliability
}

// GetUpdates returns all updates
func (im *InternalModel) GetUpdates() []ModelUpdate {
	return im.Updates
}

// addUpdate adds an update to the log
func (im *InternalModel) addUpdate(updateType string, details string) {
	im.Updates = append(im.Updates, ModelUpdate{
		Timestamp:  time.Now(),
		UpdateType: updateType,
		Details:    details,
	})
}
