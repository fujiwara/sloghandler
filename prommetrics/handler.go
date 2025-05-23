// Package prommetrics provides a log/slog handler implementation that collects
// Prometheus metrics from log statements
package prommetrics

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
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
		MinLevel: slog.LevelInfo, // Record Info and above by default
	}
}

// SlogHandler is a slog.Handler that counts log messages by level in Prometheus metrics.
type SlogHandler struct {
	slog.Handler
	counter  *prometheus.CounterVec
	minLevel slog.Level
}

// NewHandler creates a new SlogHandler that wraps the given base handler.
// It will increment the provided Prometheus counter for each log message,
// using the log level as a label.
//
// The counter should have a "level" label defined, for example:
//
//	counter := prometheus.NewCounterVec(
//	  prometheus.CounterOpts{
//	    Name: "log_messages_total",
//	    Help: "Total number of log messages by level",
//	  },
//	  []string{"level"},
//	)
//	prometheus.MustRegister(counter)
//	handler := prommetrics.NewHandler(baseHandler, counter)
//
// The handler initializes all log levels with zero values to ensure all levels
// appear in metrics output even before the first log at that level.
func NewHandler(base slog.Handler, counter *prometheus.CounterVec) slog.Handler {
	return NewHandlerWithOptions(base, counter, DefaultOptions())
}

// NewHandlerWithOptions creates a new SlogHandler with the provided options.
func NewHandlerWithOptions(base slog.Handler, counter *prometheus.CounterVec, opts *Options) slog.Handler {
	// Use InitLevels if provided, otherwise auto-generate from MinLevel
	for _, l := range predefinedLevels {
		if l >= opts.MinLevel {
			counter.WithLabelValues(l.String()).Add(0)
		}
	}

	return &SlogHandler{
		Handler:  base,
		counter:  counter,
		minLevel: opts.MinLevel,
	}
}

// Handle processes the log record, increments the appropriate counter,
// and passes the record to the underlying handler.
func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Check if we should record this level based on minLevel
	if r.Level >= h.minLevel {
		// Increment counter for this level
		h.counter.WithLabelValues(r.Level.String()).Inc()
	}

	// Always pass the record to the underlying handler
	return h.Handler.Handle(ctx, r)
}
