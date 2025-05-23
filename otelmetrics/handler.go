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

	// LabelAttributes specifies the attributes to use as labels in the OpenTelemetry counter.
	LabelAttributes []string
}

// DefaultOptions returns the default configuration options.
func DefaultOptions() *Options {
	return &Options{
		MinLevel:        slog.LevelInfo, // Record Info and above by default
		LabelAttributes: []string{},
	}
}

// SlogHandler is a slog.Handler that counts log messages by level in OpenTelemetry metrics.
type SlogHandler struct {
	slog.Handler
	counter metric.Int64Counter
	options *Options
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
			if len(opts.LabelAttributes) == 0 {
				// Add a zero value for each level to ensure it appears in metrics
				// even if no logs have been recorded at that level yet.
				counter.Add(ctx, 0, metric.WithAttributes(attribute.String("level", l.String())))
			} else {
				// When using label attributes, initialize with empty values for other attributes
				attrs := make([]attribute.KeyValue, len(opts.LabelAttributes)+1)
				attrs[0] = attribute.String("level", l.String())
				for i, attr := range opts.LabelAttributes {
					attrs[i+1] = attribute.String(attr, "")
				}
				counter.Add(ctx, 0, metric.WithAttributes(attrs...))
			}
		}
	}

	return &SlogHandler{
		Handler: base,
		counter: counter,
		options: opts,
	}
}

// Handle processes the log record, increments the appropriate counter with
// the log level as an attribute, and passes the record to the underlying handler.
func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Check if we should record this level based on MinLevel
	if r.Level < h.options.MinLevel {
		return h.Handler.Handle(ctx, r)
	}

	if len(h.options.LabelAttributes) == 0 {
		// Increment counter for this level only
		h.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("level", r.Level.String()),
		))
	} else {
		// Use the specified label attributes
		attrs := make([]attribute.KeyValue, len(h.options.LabelAttributes)+1)
		attrs[0] = attribute.String("level", r.Level.String())
		
		// Initialize all attribute values with empty strings
		for i, attrName := range h.options.LabelAttributes {
			attrs[i+1] = attribute.String(attrName, "")
		}
		
		// Find matching attributes in the log record
		for i, attrName := range h.options.LabelAttributes {
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == attrName {
					attrs[i+1] = attribute.String(attrName, a.Value.String())
				}
				return true
			})
		}
		
		h.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	// Always pass the record to the underlying handler
	return h.Handler.Handle(ctx, r)
}
