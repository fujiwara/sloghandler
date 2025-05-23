package prommetrics

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
)

// Helper to return metrics as a map[level]count
func gatherCounts(t *testing.T, reg *prometheus.Registry, metricName string) map[string]float64 {
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}
	counts := make(map[string]float64)
	for _, mf := range metrics {
		if *mf.Name != metricName {
			continue
		}
		for _, m := range mf.Metric {
			var level string
			for _, l := range m.Label {
				if *l.Name == "level" {
					level = *l.Value
					break
				}
			}
			if level != "" {
				counts[level] = *m.Counter.Value
			}
		}
	}
	return counts
}

func TestPromHandler(t *testing.T) {
	// Create a test registry
	reg := prometheus.NewRegistry()

	// Create a test counter
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_messages_total",
			Help: "Total number of log messages by level",
		},
		[]string{"level"},
	)
	reg.MustRegister(counter)

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a base handler that writes to our buffer with Debug level enabled
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	baseHandler := slog.NewTextHandler(&buf, opts)

	// Create our slogmetrics handler
	handler := NewHandler(baseHandler, counter)

	// Create a logger with our handler
	logger := slog.New(handler)

	// Log messages at different levels - this is the typical usage pattern
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	// Log a few more messages to check increment
	logger.Info("Another info message")
	logger.Error("Another error message")

	// Small delay to ensure metrics are updated
	time.Sleep(10 * time.Millisecond)

	// Verify metrics were collected
	got := gatherCounts(t, reg, "log_messages_total")
	want := map[string]float64{
		"INFO":  2,
		"WARN":  1,
		"ERROR": 2,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Metric counts mismatch (-want +got):\n%s", diff)
	}
}

// TestMinLevel tests the handler with custom min level
func TestMinLevel(t *testing.T) {
	// Create a test registry
	reg := prometheus.NewRegistry()

	// Create a test counter
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_messages_min_level_total",
			Help: "Total number of log messages by level with min level",
		},
		[]string{"level"},
	)
	reg.MustRegister(counter)

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a base handler that writes to our buffer with Debug level enabled
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	baseHandler := slog.NewTextHandler(&buf, opts)

	// Create custom options with minimum level set to INFO
	customOpts := &Options{
		MinLevel: slog.LevelInfo,
	}

	// Create our handler with custom options
	handler := NewHandlerWithOptions(baseHandler, counter, customOpts)

	// Create a logger with our handler
	logger := slog.New(handler)

	// Log messages at different levels
	logger.Debug("Debug message") // Should not be recorded in metrics (below min level)
	logger.Info("Info message")   // Should be recorded
	logger.Warn("Warn message")   // Should be recorded
	logger.Error("Error message") // Should be recorded

	// Small delay to ensure metrics are updated
	time.Sleep(10 * time.Millisecond)

	// Verify metrics were collected
	got := gatherCounts(t, reg, "log_messages_min_level_total")
	want := map[string]float64{
		"INFO":  1,
		"WARN":  1,
		"ERROR": 1,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Metric counts mismatch (-want +got):\n%s", diff)
	}
}
