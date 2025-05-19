# sloghandler

sloghandler is a simple text handler for the [log/slog](https://pkg.go.dev/log/slog) package.
It provides colored output for log levels and customizable formatting.

## Features

- Simple, readable text output format
- Colorized output for warning and error levels
- Support for log level filtering
- Custom time format
- Preservation of log attributes

### Installation

```
go get github.com/fujiwara/sloghandler
```

### Usage

Example usage with default settings:

```go
package main

import (
	"log/slog"
	"os"

	"github.com/fujiwara/sloghandler"
)

func main() {
	// Create handler with default options
	opts := &sloghandler.HandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
		Color: true, // Colorize the output based on log level
	}
	handler := sloghandler.NewLogHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	// Basic usage
	slog.Debug("This is a debug message")
	slog.Info("This is an info message")
	slog.Warn("This is a warning message") // Yellow output
	slog.Error("This is an error message") // Red output

	// With attributes
	slog.Info("Server started", "port", 8080, "environment", "production")

	// Create child logger with attached attributes
	serverLogger := logger.With("component", "server", "id", "main")
	serverLogger.Info("Listening for connections")
}
```

Output format:

```
2023-05-09T12:34:56.789+09:00 [DEBUG] This is a debug message
2023-05-09T12:34:56.790+09:00 [INFO] This is an info message
2023-05-09T12:34:56.791+09:00 [WARN] This is a warning message
2023-05-09T12:34:56.792+09:00 [ERROR] This is an error message
2023-05-09T12:34:56.793+09:00 [INFO] [port:8080] [environment:production] Server started
2023-05-09T12:34:56.794+09:00 [INFO] [component:server] [id:main] Listening for connections
```

## Customization

You can customize the time format and colors:

```go
// Customize time format
sloghandler.TimeFormat = "2006/01/02 15:04:05"

// Customize colors
sloghandler.WarnColor = color.FgMagenta
sloghandler.ErrorColor = color.FgHiRed
```

---

# Metrics Handlers

The following packages provide slog.Handler implementations for metrics integration. These are separate from the main sloghandler package.

## otelmetrics: OpenTelemetry Metrics Handler

```go
import (
	"log/slog"
	"go.opentelemetry.io/otel/metric"
	"github.com/fujiwara/sloghandler/otelmetrics"
)

func main() {
	// Create an OpenTelemetry meter and counter
	meter := provider.Meter("example/logs")
	counter, _ := meter.Int64Counter(
		"log_messages",
		metric.WithDescription("Number of log messages by level"),
	)
	// Create a base slog handler
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	// Wrap with the OpenTelemetry handler
	handler := otelmetrics.NewHandler(baseHandler, counter)
	logger := slog.New(handler)

	logger.Info("This is an info message")
	logger.Error("This is an error message")
}
```

## prommetrics: Prometheus Metrics Handler

```go
import (
	"log/slog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/fujiwara/sloghandler/prommetrics"
)

func main() {
	// Create a Prometheus counter
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_messages_total",
			Help: "Total number of log messages by level",
		},
		[]string{"level"},
	)
	prometheus.MustRegister(counter)
	// Create a base slog handler
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	// Wrap with the Prometheus handler
	handler := prommetrics.NewHandler(baseHandler, counter)
	logger := slog.New(handler)

	logger.Info("This is an info message")
	logger.Error("This is an error message")
}
```

## LICENSE

MIT

## Author

Fujiwara
