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
	TimeFormat = "2006-01-02T15:04:05.000Z07:00"
	WarnColor  = color.FgYellow
	ErrorColor = color.FgRed
)

var (
	warnColoredFprintFunc  = color.New(WarnColor).FprintFunc()
	errorColoredFprintFunc = color.New(ErrorColor).FprintFunc()
	defaultFprintFunc      = func(w io.Writer, args ...interface{}) {
		fmt.Fprint(w, args...)
	}
)

type HandlerOptions struct {
	slog.HandlerOptions
	Color bool
}

type logHandler struct {
	opts         *HandlerOptions
	preformatted []byte
	mu           *sync.Mutex
	w            io.Writer
}

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
			// no color
		case slog.LevelInfo:
			// no color
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
		preformatted = append(preformatted, fmt.Sprintf(" [%s:%v]", a.Key, a.Value)...)
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
