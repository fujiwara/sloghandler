//go:build go1.23 && !go1.25

package sloghandler

import (
	"bytes"
	"fmt"
	"log/slog"
	"runtime"
)

func (h *logHandler) printSource(buf *bytes.Buffer, record slog.Record) {
	if record.PC == 0 {
		return
	}
	fs := runtime.CallersFrames([]uintptr{record.PC})
	f, _ := fs.Next()
	file := h.getFilePath(f.File)
	fmt.Fprintf(buf, " [%s:%d]", file, f.Line)
}
