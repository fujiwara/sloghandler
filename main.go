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
	warnColoredFprintfFunc  = color.New(WarnColor).FprintfFunc()
	errorColoredFprintfFunc = color.New(ErrorColor).FprintfFunc()
	defaultFprintfFunc      = func(w io.Writer, format string, args ...interface{}) {
		fmt.Fprintf(w, format, args...)
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

func (h *logHandler) FprintfFunc(level slog.Level) func(io.Writer, string, ...interface{}) {
	if h.opts.Color {
		switch level {
		case slog.LevelDebug:
			// no color
		case slog.LevelInfo:
			// no color
		case slog.LevelWarn:
			return warnColoredFprintfFunc
		case slog.LevelError:
			return errorColoredFprintfFunc
		}
	}
	return defaultFprintfFunc
}

func (h *logHandler) Handle(ctx context.Context, record slog.Record) error {
	buf := new(bytes.Buffer)
	fprintf := h.FprintfFunc(record.Level)
	fprintf(buf, "%s", record.Time.Format(TimeFormat))
	fprintf(buf, " [%s]", record.Level.String())
	if len(h.preformatted) > 0 {
		buf.Write(h.preformatted)
	}
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "" {
			fprintf(buf, " [%v]", a.Value)
		} else {
			fprintf(buf, " [%s:%v]", a.Key, a.Value)
		}
		return true
	})
	fprintf(buf, " %s\n", record.Message)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
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
