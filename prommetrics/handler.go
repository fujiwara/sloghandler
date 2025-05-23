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

	// LabelAttributes specifies the attributes to use as labels in the Prometheus counter.
	LabelAttributes []string
}

// DefaultOptions returns the default configuration options.
func DefaultOptions() *Options {
	return &Options{
		MinLevel:        slog.LevelInfo, // Record Info and above by default
		LabelAttributes: []string{},
	}
}

// SlogHandler is a slog.Handler that counts log messages by level in Prometheus metrics.
type SlogHandler struct {
	slog.Handler
	counter *prometheus.CounterVec
	options *Options
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
	// Initialize counters for each level with appropriate label values
	for _, l := range predefinedLevels {
		if l >= opts.MinLevel {
			if len(opts.LabelAttributes) == 0 {
				counter.WithLabelValues(l.String()).Add(0)
			} else {
				// When using label attributes, initialize with empty values for other labels
				labels := make([]string, len(opts.LabelAttributes)+1)
				labels[0] = l.String()
				for i := 1; i < len(labels); i++ {
					labels[i] = ""
				}
				counter.WithLabelValues(labels...).Add(0)
			}
		}
	}

	return &SlogHandler{
		Handler: base,
		counter: counter,
		options: opts,
	}
}

// Handle processes the log record, increments the appropriate counter,
// and passes the record to the underlying handler.
func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < h.options.MinLevel {
		return h.Handler.Handle(ctx, r)
	}
	if l := len(h.options.LabelAttributes); l == 0 {
		h.counter.WithLabelValues(r.Level.String()).Inc()
	} else {
		// Use the specified label attributes
		labels := make([]string, l+1)
		labels[0] = r.Level.String()
		// Initialize all attribute labels with empty strings
		for i := 1; i < len(labels); i++ {
			labels[i] = ""
		}
		for i, attr := range h.options.LabelAttributes {
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == attr {
					labels[i+1] = a.Value.String()
				}
				return true
			})
		}
		h.counter.WithLabelValues(labels...).Inc()
	}

	return h.Handler.Handle(ctx, r)
}
