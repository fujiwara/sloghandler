# sloghandler

A collection of handlers for Go's `log/slog` package.

## Project Structure

- `main.go` - Main colored text log handler implementation
- `source.go` - Source file path utility (shared)
- `source_go125.go` - Go 1.25+ source location using `record.Source()`
- `source_legacy.go` - Go 1.23-1.24 source location using `runtime.CallersFrames`
- `otelmetrics/` - OpenTelemetry metrics handler (separate go.mod)
- `prommetrics/` - Prometheus metrics handler (separate go.mod)

## Development

```bash
# Run tests
go test -v ./...

# Run tests for subpackages
cd otelmetrics && go test -v ./...
cd prommetrics && go test -v ./...
```

## Build Tags

- `//go:build go1.25` - For Go 1.25+ specific code
- `//go:build go1.23 && !go1.25` - For Go 1.23-1.24 specific code

## Notes

- Subpackages (`otelmetrics/`, `prommetrics/`) have their own `go.mod` files
- CI tests against Go 1.23, 1.24, and 1.25

## TODO

- When minimum Go version is bumped to 1.24, replace `context.Background()` with `t.Context()` in tests
