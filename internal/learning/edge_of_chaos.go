package learning

import (
	"math/rand"
)

// EdgeOfChaos manages the balance between exploration and exploitation
type EdgeOfChaos struct {
	balance         float64
	baseTemperature float64
	maxTemperature  float64
}

// NewEdgeOfChaos creates a new edge of chaos manager
func NewEdgeOfChaos(baseTemperature, maxTemperature float64) *EdgeOfChaos {
	if baseTemperature <= 0 {
		baseTemperature = 0.7
	}
	if maxTemperature <= 0 {
		maxTemperature = 1.5
	}

	return &EdgeOfChaos{
		balance:         0.5,
		baseTemperature: baseTemperature,
		maxTemperature:  maxTemperature,
	}
}

// CalculateBalance calculates the optimal balance based on recent success
func (eoc *EdgeOfChaos) CalculateBalance(recentSuccessRate float64) float64 {
	if recentSuccessRate > 0.8 {
		// High success: increase exploration
		eoc.balance = 0.6
	} else if recentSuccessRate < 0.4 {
		// Low success: increase exploitation
		eoc.balance = 0.3
	} else {
		// Medium success: maintain balance
		eoc.balance = 0.5
	}

	return eoc.balance
}

// ShouldExplore returns true if exploration should be used
func (eoc *EdgeOfChaos) ShouldExplore() bool {
	return rand.Float64() < eoc.balance
}

// GetTemperature returns the current temperature
func (eoc *EdgeOfChaos) GetTemperature() float64 {
	// Higher balance = higher temperature
	temp := eoc.baseTemperature + (eoc.maxTemperature-eoc.baseTemperature)*eoc.balance
	if temp > eoc.maxTemperature {
		temp = eoc.maxTemperature
	}
	return temp
}

// GetBalance returns the current balance
func (eoc *EdgeOfChaos) GetBalance() float64 {
	return eoc.balance
}

// SetBalance sets the balance
func (eoc *EdgeOfChaos) SetBalance(balance float64) {
	if balance < 0 {
		balance = 0
	}
	if balance > 1 {
		balance = 1
	}
	eoc.balance = balance
}
