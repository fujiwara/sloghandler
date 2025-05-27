package sloghandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/fatih/color"
)

var (
	// TimeFormat defines the timestamp format used in log output.
	// Default is RFC3339 with milliseconds.
	TimeFormat = "2006-01-02T15:04:05.000Z07:00"
	
	// DebugColor defines the color attribute for DEBUG level messages.
	// Default is dark gray (color.FgHiBlack).
	DebugColor = color.FgHiBlack
	
	// InfoColor defines the color attribute for INFO level messages.
	// Default is 0 (no color). Set to a color.Attribute value to enable coloring.
	InfoColor color.Attribute
	
	// WarnColor defines the color attribute for WARN level messages.
	// Default is yellow (color.FgYellow).
	WarnColor = color.FgYellow
	
	// ErrorColor defines the color attribute for ERROR level messages.
	// Default is red (color.FgRed).
	ErrorColor = color.FgRed
)

var (
	debugColoredFprintFunc = color.New(DebugColor).FprintFunc()
	warnColoredFprintFunc  = color.New(WarnColor).FprintFunc()
	errorColoredFprintFunc = color.New(ErrorColor).FprintFunc()
	defaultFprintFunc      = func(w io.Writer, args ...interface{}) {
		fmt.Fprint(w, args...)
	}
)

// HandlerOptions extends slog.HandlerOptions with additional formatting options.
type HandlerOptions struct {
	slog.HandlerOptions
	// Color enables colored output based on log level when set to true.
	// Colors can be customized using the global color variables.
	Color bool
}

type logHandler struct {
	opts         *HandlerOptions
	preformatted []byte
	mu           *sync.Mutex
	w            io.Writer
}

// NewLogHandler creates a new log handler that writes formatted log messages to w.
// The handler supports colored output when opts.Color is true, with customizable
// colors for each log level via global color variables.
func NewLogHandler(w io.Writer, opts *HandlerOptions) slog.Handler {
	return &logHandler{
		opts: opts,
		mu:   new(sync.Mutex),
		w:    w,
	}
}

func (h *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *logHandler) FprintFunc(level slog.Level) func(io.Writer, ...interface{}) {
	if h.opts.Color {
		switch level {
		case slog.LevelDebug:
			return debugColoredFprintFunc
		case slog.LevelInfo:
			if InfoColor != 0 {
				return color.New(InfoColor).FprintFunc()
			}
		case slog.LevelWarn:
			return warnColoredFprintFunc
		case slog.LevelError:
			return errorColoredFprintFunc
		}
	}
	return defaultFprintFunc
}

func (h *logHandler) Handle(ctx context.Context, record slog.Record) error {
	buf := new(bytes.Buffer)

	// Build the log message without color formatting
	fmt.Fprintf(buf, "%s", record.Time.Format(TimeFormat))
	fmt.Fprintf(buf, " [%s]", record.Level.String())

	if len(h.preformatted) > 0 {
		buf.Write(h.preformatted)
	}

	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "" {
			fmt.Fprintf(buf, " [%v]", a.Value)
		} else {
			fmt.Fprintf(buf, " [%s:%v]", a.Key, a.Value)
		}
		return true
	})

	fmt.Fprintf(buf, " %s\n", record.Message)

	// Apply color only once at the end if needed
	h.mu.Lock()
	defer h.mu.Unlock()
	var err error
	if h.opts.Color {
		fprint := h.FprintFunc(record.Level)
		fprint(h.w, buf.String())
	} else {
		// Write the buffer directly without color formatting
		_, err = h.w.Write(buf.Bytes())
	}
	return err
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	preformatted := []byte{}
	for _, a := range attrs {
		if a.Key == "" {
			preformatted = append(preformatted, fmt.Sprintf(" [%v]", a.Value)...)
		} else {
			// Preformat the attribute key-value pair
			preformatted = append(preformatted, fmt.Sprintf(" [%s:%v]", a.Key, a.Value)...)
		}
	}
	return &logHandler{
		opts:         h.opts,
		preformatted: preformatted,
		mu:           h.mu,
		w:            h.w,
	}
}

func (h *logHandler) WithGroup(group string) slog.Handler {
	return h
}
