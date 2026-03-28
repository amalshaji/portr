package stats

import (
	"testing"
	"time"
)

func TestStatsCollector(t *testing.T) {
	// Create the stats collector
	collector := NewStatsCollector()

	// Test that collector is not running initially
	if collector.IsRunning() {
		t.Error("Expected collector to not be running initially")
	}

	// Start the collector
	collector.Start()

	// Test that collector is running
	if !collector.IsRunning() {
		t.Error("Expected collector to be running after Start()")
	}

	// Wait a bit for some stats to be collected
	time.Sleep(6 * time.Second)

	// Get stats
	stats := collector.GetStats()

	// Should have collected some data points
	if len(stats) == 0 {
		t.Error("Expected to have collected some stats data points")
	}

	// Test latest stats
	if _, ok := collector.GetLatestStats(); !ok {
		t.Error("Expected to have latest stats available")
	}

	// Stop the collector
	collector.Stop()

	// Test that collector is not running after stop
	time.Sleep(100 * time.Millisecond) // Small delay to allow goroutine to stop
	if collector.IsRunning() {
		t.Error("Expected collector to not be running after Stop()")
	}
}

func TestRollingStats(t *testing.T) {
	rs := NewRollingStats(3)

	// Test initial state
	if rs.Size() != 0 {
		t.Errorf("Expected initial size to be 0, got %d", rs.Size())
	}

	// Add first data point
	data1 := StatsData{Timestamp: time.Now()}
	rs.Add(data1)

	if rs.Size() != 1 {
		t.Errorf("Expected size to be 1 after adding first item, got %d", rs.Size())
	}

	// Add second data point
	data2 := StatsData{Timestamp: time.Now()}
	rs.Add(data2)

	if rs.Size() != 2 {
		t.Errorf("Expected size to be 2 after adding second item, got %d", rs.Size())
	}

	// Add third data point
	data3 := StatsData{Timestamp: time.Now()}
	rs.Add(data3)

	if rs.Size() != 3 {
		t.Errorf("Expected size to be 3 after adding third item, got %d", rs.Size())
	}

	// Add fourth data point (should roll out the first one)
	data4 := StatsData{Timestamp: time.Now()}
	rs.Add(data4)

	if rs.Size() != 3 {
		t.Errorf("Expected size to remain 3 after rolling, got %d", rs.Size())
	}

	// Get all and check latest
	all := rs.GetAll()
	if len(all) != 3 {
		t.Errorf("Expected GetAll() to return 3 items, got %d", len(all))
	}

	// The last item should be data4
	if all[2].Timestamp != data4.Timestamp {
		t.Error("Expected last item in GetAll() to be the most recently added data point")
	}
}
