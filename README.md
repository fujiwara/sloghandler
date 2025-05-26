# sloghandler

sloghandler is a collection of handlers for the [log/slog](https://pkg.go.dev/log/slog) package.
It provides colored text output with customizable formatting and metrics integration for monitoring log activity.

## Features

- Simple, readable text output format
- Colorized output for warning and error levels
- Preservation of log attributes
- Metrics integration: Export log volume and level metrics to monitoring systems
  - `prommetrics`: Prometheus metrics handler for tracking log statistics
  - `otelmetrics`: OpenTelemetry metrics handler for observability integration

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

The following packages provide slog.Handler implementations for metrics integration.

These handlers are useful when you want to:

- Monitor the volume and level of log messages in your application as metrics.
- Visualize trends or spikes in log output (e.g., sudden increase in ERROR logs) using monitoring systems like Prometheus or OpenTelemetry.
- Set up alerts based on log activity, such as triggering an alert if ERROR logs exceed a threshold.
- Analyze log level distribution over time without parsing raw log files.
- Integrate log statistics into dashboards for observability and SRE/DevOps workflows.

By using these handlers, you can export log activity as metrics, making it easy to observe, alert, and analyze your application's behavior in production environments.

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

### Custom Label Attributes (Optional)

Both `otelmetrics` and `prommetrics` support custom label attributes, allowing you to add specific log attributes as metric labels for more detailed monitoring:

#### otelmetrics with Custom Labels

```go
// Create handler with custom label attributes
opts := &otelmetrics.Options{
    MinLevel:        slog.LevelInfo,
    LabelAttributes: []string{"service", "component"},
}
handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, opts)
logger := slog.New(handler)

// These attributes will become metric labels
logger.Info("Request processed", 
    "service", "api-gateway", 
    "component", "router",
    "user_id", "12345")  // user_id will be ignored (not in LabelAttributes)
```

#### prommetrics with Custom Labels

```go
// Create a Prometheus counter with custom labels
counter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "log_messages_total",
        Help: "Total number of log messages by level",
    },
    []string{"level", "service", "component"}, // Include custom labels
)

opts := &prommetrics.Options{
    MinLevel:        slog.LevelInfo,
    LabelAttributes: []string{"service", "component"},
}
handler := prommetrics.NewHandlerWithOptions(baseHandler, counter, opts)
```

This creates metrics with labels like `level=INFO,service=api-gateway,component=router`, enabling fine-grained monitoring and alerting based on specific service components.

## LICENSE

MIT

## Author

Fujiwara
