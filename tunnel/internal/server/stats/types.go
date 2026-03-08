package stats

import (
	"sync"
	"time"
)

// StatsData represents a single datapoint of collected statistics
type StatsData struct {
	Timestamp   time.Time `json:"timestamp"`
	MemoryUsage uint64    `json:"memory_usage"` // bytes (system memory used)
	CPUUsage    float64   `json:"cpu_usage"`    // percentage
}

// RollingStats maintains a rolling window of statistics
type RollingStats struct {
	data    []StatsData
	maxSize int
	mutex   sync.RWMutex
}

// NewRollingStats creates a new rolling stats container
func NewRollingStats(maxSize int) *RollingStats {
	return &RollingStats{
		data:    make([]StatsData, 0, maxSize),
		maxSize: maxSize,
		mutex:   sync.RWMutex{},
	}
}

// Add adds a new datapoint to the rolling window
func (rs *RollingStats) Add(data StatsData) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.data = append(rs.data, data)
	if len(rs.data) > rs.maxSize {
		rs.data = rs.data[1:]
	}
}

// GetAll returns all current datapoints
func (rs *RollingStats) GetAll() []StatsData {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	result := make([]StatsData, len(rs.data))
	copy(result, rs.data)
	return result
}

// Size returns the current number of datapoints
func (rs *RollingStats) Size() int {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()
	return len(rs.data)
}
