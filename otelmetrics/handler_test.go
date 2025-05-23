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
