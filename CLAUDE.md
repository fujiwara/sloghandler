# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A collection of handlers for Go's `log/slog` package. The root package provides a colored text log handler; subpackages provide metrics-collecting wrapper handlers for Prometheus and OpenTelemetry.

## Development Commands

```bash
# Run root package tests
go test -v ./...

# Run tests with race detector (as CI does)
go test -race ./...

# Run a single test
go test -v -run TestHandlerOutput ./...

# Subpackages have separate go.mod files — test them independently
cd otelmetrics && go test -v ./...
cd prommetrics && go test -v ./...
```

## Architecture

### Root package (`sloghandler`)

`logHandler` implements `slog.Handler`. It writes colored, human-readable text logs to an `io.Writer`. Key design points:

- **Color output**: Uses `github.com/fatih/color`. Colors are configured via package-level variables (`DebugColor`, `InfoColor`, etc.) and cached `FprintFunc` closures.
- **Source location**: Uses build-tag-selected implementations — `source_go125.go` uses `record.Source()` (Go 1.25+), `source_legacy.go` uses `runtime.CallersFrames` (Go 1.23–1.24). Shared path formatting logic is in `source.go` with a `sync.Map` cache.
- **Thread safety**: A shared `sync.Mutex` is used for writing; `WithAttrs()` returns a new handler sharing the same mutex and writer.

### Metrics subpackages (`otelmetrics/`, `prommetrics/`)

Both follow the same wrapper-handler pattern: they embed `slog.Handler`, intercept `Handle()` to increment a counter by log level, then delegate to the base handler. Both support `Options` with `MinLevel` and `LabelAttributes` for custom metric labels.

These are **independent Go modules** with their own `go.mod` — they cannot import or be imported by the root module directly.

## Build Tags

- `//go:build go1.25` — Go 1.25+ specific code
- `//go:build go1.23 && !go1.25` — Go 1.23–1.24 specific code

## CI

CI runs `go test -race ./...` against Go 1.23, 1.24, and 1.25. Subpackage tests are not in CI matrix (they have separate modules).

## TODO

- When minimum Go version is bumped to 1.24, replace `context.Background()` with `t.Context()` in tests
