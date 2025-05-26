package otelmetrics_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/fujiwara/sloghandler/otelmetrics"
	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// setup provider for testing
func setupProvider(t *testing.T) (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	exporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(100*time.Millisecond),
		)),
	)

	otel.SetMeterProvider(provider)
	return provider, reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) map[string]int64 {
	ctx := context.Background()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Logf("Failed to collect final metrics: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("No scope metrics found")
	}
	scopeMetrics := rm.ScopeMetrics[0]
	if len(scopeMetrics.Metrics) == 0 {
		t.Fatal("No metrics found")
	}

	metricData := scopeMetrics.Metrics[0]
	if metricData.Name != "log_messages" {
		t.Errorf("Expected metric name 'log_messages', got %s", metricData.Name)
	}

	countByLevel := map[string]int64{}
	if sum, ok := metricData.Data.(metricdata.Sum[int64]); ok {
		if len(sum.DataPoints) == 0 {
			t.Fatal("No data points found")
		}
		for _, dp := range sum.DataPoints {
			// Check if the level attribute is present
			for _, attr := range dp.Attributes.ToSlice() {
				if attr.Key == "level" {
					countByLevel[attr.Value.AsString()] = dp.Value
					break
				}
			}
		}
	}
	return countByLevel
}

// Helper to collect metrics with labels as a map[labelKey]count
func collectMetricsWithLabels(t *testing.T, reader *sdkmetric.ManualReader) map[string]int64 {
	ctx := context.Background()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Logf("Failed to collect final metrics: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("No scope metrics found")
	}
	scopeMetrics := rm.ScopeMetrics[0]
	if len(scopeMetrics.Metrics) == 0 {
		t.Fatal("No metrics found")
	}

	metricData := scopeMetrics.Metrics[0]
	if metricData.Name != "log_messages" {
		t.Errorf("Expected metric name 'log_messages', got %s", metricData.Name)
	}

	countByLabels := map[string]int64{}
	if sum, ok := metricData.Data.(metricdata.Sum[int64]); ok {
		if len(sum.DataPoints) == 0 {
			t.Fatal("No data points found")
		}
		for _, dp := range sum.DataPoints {
			var labelKey string
			labels := make([]string, 0, dp.Attributes.Len())
			iter := dp.Attributes.Iter()
			for iter.Next() {
				attr := iter.Attribute()
				labels = append(labels, string(attr.Key)+"="+attr.Value.AsString())
			}
			if len(labels) > 0 {
				labelKey = labels[0]
				for i := 1; i < len(labels); i++ {
					labelKey += "," + labels[i]
				}
			}
			if labelKey != "" {
				countByLabels[labelKey] = dp.Value
			}
		}
	}
	return countByLabels
}

func TestMetrics(t *testing.T) {
	provider, reader := setupProvider(t)

	meter := provider.Meter("example/logs")
	counter, _ := meter.Int64Counter(
		"log_messages",
		metric.WithDescription("Number of log messages by level"),
	)
	// Create a base slog handler
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	// Wrap with the OpenTelemetry handler
	handler := otelmetrics.NewHandler(baseHandler, counter)
	logger := slog.New(handler)

	logger.Debug("This is a debug message") // Should not be recorded
	logger.Info("This is an info message")
	logger.Info("This is another info message")
	logger.Error("This is an error message")
	logger.Error("This is another error message")
	expectedCounts := map[string]int64{
		"INFO":  2,
		"WARN":  0,
		"ERROR": 2,
	}

	countByLevel := collectMetrics(t, reader)

	if diff := cmp.Diff(expectedCounts, countByLevel); diff != "" {
		t.Errorf("Metric counts mismatch (-want +got):\n%s", diff)
	}
}

func TestMinLevel(t *testing.T) {
	provider, reader := setupProvider(t)

	meter := provider.Meter("example/logs")
	counter, _ := meter.Int64Counter(
		"log_messages",
		metric.WithDescription("Number of log messages by level"),
	)
	// Create a base slog handler
	level := slog.LevelWarn
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	// Wrap with the OpenTelemetry handler
	handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, &otelmetrics.Options{
		MinLevel: level,
	})
	logger := slog.New(handler)

	logger.Debug("This is a debug message")     // Should not be recorded
	logger.Info("This is an info message")      // Should not be recorded
	logger.Info("This is another info message") // Should not be recorded
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")
	logger.Error("This is another error message")
	expectedCounts := map[string]int64{
		"WARN":  1,
		"ERROR": 2,
	}
	countByLevel := collectMetrics(t, reader)

	if diff := cmp.Diff(expectedCounts, countByLevel); diff != "" {
		t.Errorf("Metric counts mismatch (-want +got):\n%s", diff)
	}
}

// TestLabelAttributes tests the handler with custom label attributes
func TestLabelAttributes(t *testing.T) {
	provider, reader := setupProvider(t)

	meter := provider.Meter("example/logs")
	counter, _ := meter.Int64Counter(
		"log_messages",
		metric.WithDescription("Number of log messages by level with custom attributes"),
	)

	// Create a base slog handler
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

	// Create custom options with label attributes
	customOpts := &otelmetrics.Options{
		MinLevel:        slog.LevelInfo,
		LabelAttributes: []string{"service", "component"},
	}

	// Wrap with the OpenTelemetry handler
	handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, customOpts)
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

	expectedCounts := map[string]int64{
		"component=http-server,level=INFO,service=auth-service":  1,
		"component=database,level=ERROR,service=auth-service":    1,
		"component=router,level=INFO,service=api-gateway":        1,
		"component=monitor,level=WARN,service=api-gateway":       1,
		// Initial empty values created during handler initialization
		"component=,level=INFO,service=":                         0,
		"component=,level=WARN,service=":                         0,
		"component=,level=ERROR,service=":                        0,
	}

	countByLabels := collectMetricsWithLabels(t, reader)

	if diff := cmp.Diff(expectedCounts, countByLabels); diff != "" {
		t.Errorf("Metric counts with labels mismatch (-want +got):\n%s", diff)
	}
}

// TestLabelAttributesPartialMatch tests behavior when some attributes are missing
func TestLabelAttributesPartialMatch(t *testing.T) {
	provider, reader := setupProvider(t)

	meter := provider.Meter("example/logs")
	counter, _ := meter.Int64Counter(
		"log_messages",
		metric.WithDescription("Number of log messages with partial attribute matching"),
	)

	// Create a base slog handler
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Create custom options with label attributes
	customOpts := &otelmetrics.Options{
		MinLevel:        slog.LevelInfo,
		LabelAttributes: []string{"service"},
	}

	// Wrap with the OpenTelemetry handler
	handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, customOpts)
	logger := slog.New(handler)

	// Log messages - some with service attribute, some without
	logger.Info("Message with service", "service", "test-service")
	logger.Info("Message without service attribute")
	logger.Error("Error with service", "service", "error-service")

	expectedCounts := map[string]int64{
		"level=INFO,service=test-service":   1,
		"level=INFO,service=":               1, // Empty value for missing attribute
		"level=ERROR,service=error-service": 1,
		// Initial empty values created during handler initialization
		"level=WARN,service=":               0,
		"level=ERROR,service=":              0,
	}

	countByLabels := collectMetricsWithLabels(t, reader)

	if diff := cmp.Diff(expectedCounts, countByLabels); diff != "" {
		t.Errorf("Metric counts with partial labels mismatch (-want +got):\n%s", diff)
	}
}

// TestLabelAttributesIgnoreUnspecified tests that attributes not in LabelAttributes are ignored
func TestLabelAttributesIgnoreUnspecified(t *testing.T) {
	provider, reader := setupProvider(t)

	meter := provider.Meter("example/logs")
	counter, _ := meter.Int64Counter(
		"log_messages",
		metric.WithDescription("Number of log messages that ignore unspecified attributes"),
	)

	// Create a base slog handler
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Create custom options with only "service" as label attribute
	customOpts := &otelmetrics.Options{
		MinLevel:        slog.LevelInfo,
		LabelAttributes: []string{"service"},
	}

	// Wrap with the OpenTelemetry handler
	handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, customOpts)
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

	expectedCounts := map[string]int64{
		"level=INFO,service=test-service":   1,
		"level=ERROR,service=error-service": 1,
		"level=INFO,service=":               1, // Message with only ignored attributes
		// Initial empty values created during handler initialization
		"level=WARN,service=":               0,
		"level=ERROR,service=":              0,
	}

	countByLabels := collectMetricsWithLabels(t, reader)

	if diff := cmp.Diff(expectedCounts, countByLabels); diff != "" {
		t.Errorf("Metric counts with ignored attributes mismatch (-want +got):\n%s", diff)
	}
}
