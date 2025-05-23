package prommetrics

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

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
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find our counter metrics
	var found bool
	for _, mf := range metrics {
		if *mf.Name == "log_messages_total" {
			found = true

			// Map to store counts by level
			counts := make(map[string]float64)

			// Extract all the metrics
			for _, m := range mf.Metric {
				// Find the level label
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

			// Verify expected counts
			expectedCounts := map[string]float64{
				"INFO":  2,
				"WARN":  1,
				"ERROR": 2,
			}

			for level, expected := range expectedCounts {
				actual, ok := counts[level]
				if !ok {
					t.Errorf("Missing metrics for level %s", level)
					continue
				}

				if actual != expected {
					t.Errorf("Expected count for level %s to be %f, got %f", level, expected, actual)
				}
			}

			break
		}
	}

	if !found {
		t.Error("Could not find metrics for log_messages_total")
	}

	// Verify logs were correctly written to the buffer
	logOutput := buf.String()
	if logOutput == "" {
		t.Error("No log output captured")
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
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find our counter metrics
	var found bool
	for _, mf := range metrics {
		if *mf.Name == "log_messages_min_level_total" {
			found = true

			// Map to store counts by level
			counts := make(map[string]float64)

			// Extract all the metrics
			for _, m := range mf.Metric {
				// Find the level label
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

			// Verify expected counts - only INFO, WARN, and ERROR should be counted
			expectedCounts := map[string]float64{
				"INFO":  1,
				"WARN":  1,
				"ERROR": 1,
			}

			for level, expected := range expectedCounts {
				actual, ok := counts[level]
				if !ok {
					t.Errorf("Missing metrics for level %s", level)
					continue
				}

				if actual != expected {
					t.Errorf("Expected count for level %s to be %f, got %f", level, expected, actual)
				}
			}

			// DEBUG should not have been recorded (below min level)
			if _, exists := counts["DEBUG"]; exists {
				t.Errorf("DEBUG level was recorded but shouldn't have been (below min level)")
			}

			break
		}
	}

	if !found {
		t.Error("Could not find metrics for log_messages_min_level_total")
	}

	// Verify all logs were correctly written to the buffer (all levels should be logged, even if not counted)
	logOutput := buf.String()
	if !strings.Contains(logOutput, "Debug message") {
		t.Error("Debug message not found in log output")
	}
	if !strings.Contains(logOutput, "Info message") {
		t.Error("Info message not found in log output")
	}
	if !strings.Contains(logOutput, "Warn message") {
		t.Error("Warn message not found in log output")
	}
	if !strings.Contains(logOutput, "Error message") {
		t.Error("Error message not found in log output")
	}
}
