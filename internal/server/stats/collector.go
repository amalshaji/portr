package stats

import (
	"runtime"
	"sync"
	"time"

	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// CPU tracking variables
var (
	lastCPUTime     types.CPUTimes
	lastMeasureTime time.Time
	cpuMutex        sync.RWMutex
)

// calculateCPUUsage calculates CPU usage percentage based on time differences
func calculateCPUUsage(currentCPU types.CPUTimes) float64 {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	now := time.Now()

	// If this is the first measurement, store it and return 0
	if lastMeasureTime.IsZero() {
		lastCPUTime = currentCPU
		lastMeasureTime = now
		return 0.0
	}

	// Calculate time difference
	timeDiff := now.Sub(lastMeasureTime).Seconds()
	if timeDiff <= 0 {
		return 0.0
	}

	// Calculate CPU time differences (in nanoseconds, convert to seconds)
	userDiff := float64(currentCPU.User-lastCPUTime.User) / 1e9
	systemDiff := float64(currentCPU.System-lastCPUTime.System) / 1e9

	// Total CPU time used
	totalCPUDiff := userDiff + systemDiff

	// CPU usage percentage (considering number of CPU cores)
	numCPU := float64(runtime.NumCPU())
	cpuUsage := (totalCPUDiff / (timeDiff * numCPU)) * 100

	// Update last measurements
	lastCPUTime = currentCPU
	lastMeasureTime = now

	// Ensure reasonable bounds
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	return cpuUsage
}

// StatsCollector manages the collection and storage of statistics
type StatsCollector struct {
	rollingStats *RollingStats
	stopChan     chan struct{}
	isRunning    bool
	mutex        sync.RWMutex
}

// NewStatsCollector creates a new stats collector
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		rollingStats: NewRollingStats(30), // Keep last 30 datapoints
		stopChan:     make(chan struct{}),
		isRunning:    false,
	}
}

// collectStats collects memory and CPU usage statistics
func (sc *StatsCollector) collectStats() StatsData {
	data := StatsData{
		Timestamp: time.Now(),
	}

	// Get system memory usage
	if host, err := sysinfo.Host(); err == nil {
		if memory, err := host.Memory(); err == nil {
			data.MemoryUsage = memory.Used
		}

		// Get CPU usage
		if cpuTime, err := host.CPUTime(); err == nil {
			data.CPUUsage = calculateCPUUsage(cpuTime)
		}
	}

	return data
}

// collectLoop runs the collection loop every 5 seconds
func (sc *StatsCollector) collectLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := sc.collectStats()
			sc.rollingStats.Add(data)
		case <-sc.stopChan:
			return
		}
	}
}

// Start begins the stats collection goroutine
func (sc *StatsCollector) Start() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if sc.isRunning {
		return
	}

	sc.isRunning = true
	go sc.collectLoop()
}

// Stop stops the stats collection goroutine
func (sc *StatsCollector) Stop() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if !sc.isRunning {
		return
	}

	sc.isRunning = false
	close(sc.stopChan)
}

// IsRunning returns whether the collector is currently running
func (sc *StatsCollector) IsRunning() bool {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	return sc.isRunning
}

// GetStats returns the rolling statistics
func (sc *StatsCollector) GetStats() []StatsData {
	return sc.rollingStats.GetAll()
}

// GetLatestStats returns the most recent statistics
func (sc *StatsCollector) GetLatestStats() (StatsData, bool) {
	allStats := sc.GetStats()
	if len(allStats) == 0 {
		return StatsData{}, false
	}
	return allStats[len(allStats)-1], true
}
