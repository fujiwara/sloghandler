// Package otelmetrics provides a log/slog handler implementation that collects
// OpenTelemetry metrics from log statements.
package otelmetrics

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// predefinedLevels contains the standard log levels in ascending order of severity
var predefinedLevels = []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}

// Options contains configuration for the SlogHandler.
type Options struct {
	// MinLevel specifies the minimum log level to record in metrics.
	// Logs with levels below this will not be counted in metrics.
	// If not set (zero value), all log levels will be recorded.
	MinLevel slog.Level
}

// DefaultOptions returns the default configuration options.
func DefaultOptions() *Options {
	return &Options{
		MinLevel: slog.LevelDebug, // Record Debug and above by default
	}
}

// SlogHandler is a slog.Handler that counts log messages by level in OpenTelemetry metrics.
type SlogHandler struct {
	slog.Handler
	counter  metric.Int64Counter
	minLevel slog.Level
}

// For testing purposes only
var _ slog.Handler = (*SlogHandler)(nil)

// NewHandler creates a new SlogHandler that wraps the given base handler.
// It will increment the provided OpenTelemetry counter for each log message,
// adding a "level" attribute with the log level.
//
// The counter should be created from an OpenTelemetry meter, for example:
//
//	meter := provider.Meter("example/logs")
//	counter, _ := meter.Int64Counter(
//	  "log_messages",
//	  metric.WithDescription("Number of log messages by level"),
//	)
//	handler := otelmetrics.NewHandler(baseHandler, counter)
//
// The handler initializes all log levels with zero values to ensure all levels
// appear in metrics output even before the first log at that level.
func NewHandler(base slog.Handler, counter metric.Int64Counter) slog.Handler {
	return NewHandlerWithOptions(base, counter, DefaultOptions())
}

// NewHandlerWithOptions creates a new SlogHandler with the provided options.
func NewHandlerWithOptions(base slog.Handler, counter metric.Int64Counter, opts *Options) slog.Handler {
	ctx := context.Background()
	// Initialize counters with zero value for metrics visibility
	for _, l := range predefinedLevels {
		if l >= opts.MinLevel {
			// Add a zero value for each level to ensure it appears in metrics
			// even if no logs have been recorded at that level yet.
			counter.Add(ctx, 0, metric.WithAttributes(attribute.String("level", l.String())))
		}
	}

	return &SlogHandler{
		Handler:  base,
		counter:  counter,
		minLevel: opts.MinLevel,
	}
}

// Handle processes the log record, increments the appropriate counter with
// the log level as an attribute, and passes the record to the underlying handler.
func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Check if we should record this level based on minLevel
	if r.Level >= h.minLevel {
		// Increment counter for this level
		h.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("level", r.Level.String()),
		))
	}

	// Always pass the record to the underlying handler
	return h.Handler.Handle(ctx, r)
}
