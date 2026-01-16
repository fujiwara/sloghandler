//go:build go1.25

package sloghandler

import (
	"bytes"
	"fmt"
	"log/slog"
)

func (h *logHandler) printSource(buf *bytes.Buffer, record slog.Record) {
	if s := record.Source(); s != nil {
		file := h.getFilePath(s.File)
		fmt.Fprintf(buf, " [%s:%d]", file, s.Line)
	}
}
