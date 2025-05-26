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

// Helper to return metrics with labels as a map[labelKey]count
func gatherCountsWithLabels(t *testing.T, reg *prometheus.Registry, metricName string) map[string]float64 {
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
			var labelKey string
			labels := make([]string, 0, len(m.Label))
			for _, l := range m.Label {
				labels = append(labels, *l.Name+"="+*l.Value)
			}
			if len(labels) > 0 {
				labelKey = labels[0]
				if len(labels) > 1 {
					for i := 1; i < len(labels); i++ {
						labelKey += "," + labels[i]
					}
				}
			}
			if labelKey != "" {
				counts[labelKey] = *m.Counter.Value
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

// TestLabelAttributes tests the handler with custom label attributes
func TestLabelAttributes(t *testing.T) {
	// Create a test registry
	reg := prometheus.NewRegistry()

	// Create a test counter with additional labels
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_messages_with_attrs_total",
			Help: "Total number of log messages by level with custom attributes",
		},
		[]string{"level", "service", "component"},
	)
	reg.MustRegister(counter)

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a base handler that writes to our buffer with Debug level enabled
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	baseHandler := slog.NewTextHandler(&buf, opts)

	// Create custom options with label attributes
	customOpts := &Options{
		MinLevel:        slog.LevelInfo,
		LabelAttributes: []string{"service", "component"},
	}

	// Create our handler with custom options
	handler := NewHandlerWithOptions(baseHandler, counter, customOpts)

	// Create a logger with our handler
	logger := slog.New(handler)

	// Log messages with different attributes
	logger.Info("Service started",
		"service", "auth-service",
		"component", "http-server")
	logger.Error("Database connection failed",
		"service", "auth-service",
		"component", "database")
	logger.Info("Request processed",
		"service", "api-gateway",
		"component", "router")
	logger.Warn("High memory usage",
		"service", "api-gateway",
		"component", "monitor")

	// Small delay to ensure metrics are updated
	time.Sleep(10 * time.Millisecond)

	// Verify metrics were collected with correct labels
	got := gatherCountsWithLabels(t, reg, "log_messages_with_attrs_total")
	want := map[string]float64{
		"component=http-server,level=INFO,service=auth-service":  1,
		"component=database,level=ERROR,service=auth-service":    1,
		"component=router,level=INFO,service=api-gateway":        1,
		"component=monitor,level=WARN,service=api-gateway":       1,
		// Initial empty values created during handler initialization
		"component=,level=INFO,service=":                         0,
		"component=,level=WARN,service=":                         0,
		"component=,level=ERROR,service=":                        0,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Metric counts with labels mismatch (-want +got):\n%s", diff)
	}
}

// TestLabelAttributesPartialMatch tests behavior when some attributes are missing
func TestLabelAttributesPartialMatch(t *testing.T) {
	// Create a test registry
	reg := prometheus.NewRegistry()

	// Create a test counter with additional labels
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_messages_partial_attrs_total",
			Help: "Total number of log messages with partial attribute matching",
		},
		[]string{"level", "service"},
	)
	reg.MustRegister(counter)

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a base handler
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	baseHandler := slog.NewTextHandler(&buf, opts)

	// Create custom options with label attributes
	customOpts := &Options{
		MinLevel:        slog.LevelInfo,
		LabelAttributes: []string{"service"},
	}

	// Create our handler with custom options
	handler := NewHandlerWithOptions(baseHandler, counter, customOpts)

	// Create a logger with our handler
	logger := slog.New(handler)

	// Log messages - some with service attribute, some without
	logger.Info("Message with service", "service", "test-service")
	logger.Info("Message without service attribute")
	logger.Error("Error with service", "service", "error-service")

	// Small delay to ensure metrics are updated
	time.Sleep(10 * time.Millisecond)

	// Verify metrics were collected
	got := gatherCountsWithLabels(t, reg, "log_messages_partial_attrs_total")
	want := map[string]float64{
		"level=INFO,service=test-service":   1,
		"level=INFO,service=":               1, // Empty value for missing attribute
		"level=ERROR,service=error-service": 1,
		// Initial empty values created during handler initialization
		"level=WARN,service=":               0,
		"level=ERROR,service=":              0, // This will remain 0 since we log with "service" set
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Metric counts with partial labels mismatch (-want +got):\n%s", diff)
	}
}

// TestLabelAttributesIgnoreUnspecified tests that attributes not in LabelAttributes are ignored
func TestLabelAttributesIgnoreUnspecified(t *testing.T) {
	// Create a test registry
	reg := prometheus.NewRegistry()

	// Create a test counter with only specific labels
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_messages_ignore_attrs_total",
			Help: "Total number of log messages that ignore unspecified attributes",
		},
		[]string{"level", "service"},
	)
	reg.MustRegister(counter)

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a base handler
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	baseHandler := slog.NewTextHandler(&buf, opts)

	// Create custom options with only "service" as label attribute
	customOpts := &Options{
		MinLevel:        slog.LevelInfo,
		LabelAttributes: []string{"service"},
	}

	// Create our handler with custom options
	handler := NewHandlerWithOptions(baseHandler, counter, customOpts)

	// Create a logger with our handler
	logger := slog.New(handler)

	// Log messages with various attributes - only "service" should be used as label
	logger.Info("Message with service and extra attrs",
		"service", "test-service",
		"component", "http-server", // This should be ignored
		"user_id", "12345",         // This should be ignored
		"request_id", "req-001")    // This should be ignored

	logger.Error("Error with different attrs",
		"service", "error-service",
		"error_code", "E001",     // This should be ignored
		"retry_count", 3,         // This should be ignored
		"timeout", "30s")         // This should be ignored

	logger.Info("Message with only ignored attrs",
		"component", "database",  // This should be ignored
		"query_time", "150ms")    // This should be ignored

	// Small delay to ensure metrics are updated
	time.Sleep(10 * time.Millisecond)

	// Verify metrics were collected correctly
	got := gatherCountsWithLabels(t, reg, "log_messages_ignore_attrs_total")
	want := map[string]float64{
		"level=INFO,service=test-service":   1,
		"level=ERROR,service=error-service": 1,
		"level=INFO,service=":               1, // Message with only ignored attributes
		// Initial empty values created during handler initialization
		"level=WARN,service=":               0,
		"level=ERROR,service=":              0,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Metric counts with ignored attributes mismatch (-want +got):\n%s", diff)
	}
}
