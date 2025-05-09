package sloghandler

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestNewLogHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &HandlerOptions{
		HandlerOptions: slog.HandlerOptions{Level: slog.LevelInfo},
		Color:          false,
	}
	handler := NewLogHandler(buf, opts)

	if handler == nil {
		t.Fatal("NewLogHandler returned nil")
	}

	h, ok := handler.(*logHandler)
	if !ok {
		t.Fatal("NewLogHandler did not return a *logHandler")
	}

	if h.opts != opts {
		t.Errorf("handler options not set correctly, got %v, want %v", h.opts, opts)
	}

	if h.w != buf {
		t.Errorf("handler writer not set correctly")
	}

	if h.mu == nil {
		t.Error("handler mutex not initialized")
	}
}

func TestEnabled(t *testing.T) {
	buf := &bytes.Buffer{}
	tests := []struct {
		level        slog.Level
		handlerLevel slog.Level
		want         bool
	}{
		{slog.LevelDebug, slog.LevelInfo, false},
		{slog.LevelInfo, slog.LevelInfo, true},
		{slog.LevelWarn, slog.LevelInfo, true},
		{slog.LevelError, slog.LevelInfo, true},
		{slog.LevelDebug, slog.LevelDebug, true},
		{slog.LevelInfo, slog.LevelWarn, false},
		{slog.LevelWarn, slog.LevelWarn, true},
		{slog.LevelError, slog.LevelWarn, true},
	}

	for _, tt := range tests {
		opts := &HandlerOptions{
			HandlerOptions: slog.HandlerOptions{Level: tt.handlerLevel},
			Color:          false,
		}
		handler := NewLogHandler(buf, opts).(*logHandler)
		got := handler.Enabled(context.Background(), tt.level)
		if got != tt.want {
			t.Errorf("Enabled(%v) with handler level %v = %v, want %v", tt.level, tt.handlerLevel, got, tt.want)
		}
	}
}

func TestFprintfFunc(t *testing.T) {
	buf := &bytes.Buffer{}
	tests := []struct {
		level slog.Level
		color bool
	}{
		{slog.LevelDebug, false},
		{slog.LevelInfo, false},
		{slog.LevelWarn, false},
		{slog.LevelError, false},
		{slog.LevelDebug, true},
		{slog.LevelInfo, true},
		{slog.LevelWarn, true},
		{slog.LevelError, true},
	}

	for _, tt := range tests {
		opts := &HandlerOptions{
			HandlerOptions: slog.HandlerOptions{},
			Color:          tt.color,
		}
		handler := NewLogHandler(buf, opts).(*logHandler)
		f := handler.FprintfFunc(tt.level)
		if f == nil {
			t.Errorf("FprintfFunc(%v, color=%v) returned nil", tt.level, tt.color)
		}

		// Basic test that the function works without error
		buf.Reset()
		f(buf, "test")
		if buf.Len() == 0 {
			t.Errorf("FprintfFunc(%v, color=%v) did not write to buffer", tt.level, tt.color)
		}
	}
}

func TestHandle(t *testing.T) {
	testTime := time.Date(2023, 1, 2, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name         string
		record       slog.Record
		color        bool
		attrs        []slog.Attr
		wantContains []string
	}{
		{
			name: "info level no attrs no color",
			record: slog.Record{
				Time:    testTime,
				Level:   slog.LevelInfo,
				Message: "test message",
			},
			color: false,
			attrs: nil,
			wantContains: []string{
				"2023-01-02T15:04:05.000Z",
				"[INFO]",
				"test message",
			},
		},
		{
			name: "error level with attrs no color",
			record: slog.Record{
				Time:    testTime,
				Level:   slog.LevelError,
				Message: "error message",
			},
			color: false,
			attrs: []slog.Attr{
				slog.String("key1", "value1"),
				slog.Int("key2", 42),
			},
			wantContains: []string{
				"2023-01-02T15:04:05.000Z",
				"[ERROR]",
				"[key1:value1]",
				"[key2:42]",
				"error message",
			},
		},
		{
			name: "info level with empty key attr",
			record: slog.Record{
				Time:    testTime,
				Level:   slog.LevelInfo,
				Message: "message with empty key",
			},
			color: false,
			attrs: []slog.Attr{
				{Key: "", Value: slog.StringValue("no key")},
				slog.String("key1", "value1"),
			},
			wantContains: []string{
				"2023-01-02T15:04:05.000Z",
				"[INFO]",
				"[no key]",
				"[key1:value1]",
				"message with empty key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			opts := &HandlerOptions{
				HandlerOptions: slog.HandlerOptions{},
				Color:          tt.color,
			}
			handler := NewLogHandler(buf, opts).(*logHandler)

			// Add record attributes
			recordWithAttrs := tt.record.Clone()
			for _, attr := range tt.attrs {
				recordWithAttrs.AddAttrs(attr)
			}

			err := handler.Handle(context.Background(), recordWithAttrs)
			if err != nil {
				t.Errorf("Handle() error = %v", err)
				return
			}

			output := buf.String()
			for _, want := range tt.wantContains {
				if !bytes.Contains(buf.Bytes(), []byte(want)) {
					t.Errorf("Handle() output = %q, should contain %q", output, want)
				}
			}
		})
	}
}

func TestWithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &HandlerOptions{
		HandlerOptions: slog.HandlerOptions{},
		Color:          false,
	}
	handler := NewLogHandler(buf, opts).(*logHandler)

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	newHandler := handler.WithAttrs(attrs).(*logHandler)

	// Check that the handler has the right options
	if newHandler.opts != handler.opts {
		t.Errorf("WithAttrs() handler options not preserved")
	}

	// Check that preformatted contains the attributes
	if len(newHandler.preformatted) == 0 {
		t.Errorf("WithAttrs() preformatted is empty")
	}

	// Check that writer and mutex are preserved
	if newHandler.w != buf {
		t.Errorf("WithAttrs() writer not preserved")
	}
	if newHandler.mu != handler.mu {
		t.Errorf("WithAttrs() mutex not preserved")
	}

	// Test that the preformatted attributes are included in the output
	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test message",
	}

	buf.Reset()
	err := newHandler.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("[key1:value1]")) {
		t.Errorf("Output should contain preformatted attribute key1, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("[key2:42]")) {
		t.Errorf("Output should contain preformatted attribute key2, got: %s", output)
	}
}

func TestWithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &HandlerOptions{
		HandlerOptions: slog.HandlerOptions{},
		Color:          false,
	}
	handler := NewLogHandler(buf, opts)

	// WithGroup should return the same handler since groups are not supported
	newHandler := handler.WithGroup("test")
	if newHandler != handler {
		t.Errorf("WithGroup() should return the same handler")
	}
}

func TestIntegration(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &HandlerOptions{
		HandlerOptions: slog.HandlerOptions{Level: slog.LevelDebug},
		Color:          false,
	}
	handler := NewLogHandler(buf, opts)
	logger := slog.New(handler)

	// Test basic logging
	logger.Info("info message")
	if !bytes.Contains(buf.Bytes(), []byte("info message")) {
		t.Errorf("Logger.Info() output doesn't contain message, got: %s", buf.String())
	}

	// Reset buffer
	buf.Reset()

	// Test logging with attributes
	logger.Info("info with attrs", "key1", "value1", "key2", 42)
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("info with attrs")) {
		t.Errorf("Logger.Info() output doesn't contain message, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("[key1:value1]")) {
		t.Errorf("Logger.Info() output doesn't contain attribute key1, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("[key2:42]")) {
		t.Errorf("Logger.Info() output doesn't contain attribute key2, got: %s", output)
	}

	// Reset buffer
	buf.Reset()

	// Test with handler with attributes
	loggerWithAttrs := logger.With("service", "test")
	loggerWithAttrs.Warn("warning message")
	output = buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("warning message")) {
		t.Errorf("Logger.With().Warn() output doesn't contain message, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("[service:test]")) {
		t.Errorf("Logger.With().Warn() output doesn't contain handler attribute, got: %s", output)
	}
}
