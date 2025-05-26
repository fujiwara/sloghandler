# otelmetrics

An OpenTelemetry metrics handler for Go's `log/slog` package that automatically counts log messages by level and custom attributes.

## Features

- **Automatic metrics collection**: Counts log messages by log level (DEBUG, INFO, WARN, ERROR)
- **Configurable minimum level**: Only count logs above a specified level
- **Custom label attributes**: Use specific log attributes as OpenTelemetry labels
- **Zero initialization**: All metric levels start at 0 for consistent metric output
- **Wraps existing handlers**: Works with any `slog.Handler` implementation

## Installation

```bash
go get github.com/fujiwara/sloghandler/otelmetrics
```

## Quick Start

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/fujiwara/sloghandler/otelmetrics"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
    "go.opentelemetry.io/otel/metric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
    // Create an OpenTelemetry meter provider
    exporter, _ := stdoutmetric.New()
    reader := sdkmetric.NewPeriodicReader(exporter)
    provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
    otel.SetMeterProvider(provider)

    // Create a meter and counter
    meter := provider.Meter("example/logs")
    counter, _ := meter.Int64Counter(
        "log_messages",
        metric.WithDescription("Number of log messages by level"),
    )

    // Create base handler (JSON handler writing to stdout)
    baseHandler := slog.NewJSONHandler(os.Stdout, nil)
    
    // Wrap with metrics handler
    handler := otelmetrics.NewHandler(baseHandler, counter)
    
    // Create logger
    logger := slog.New(handler)
    
    // Use logger normally - metrics are collected automatically
    logger.Info("Application started")
    logger.Warn("This is a warning")
    logger.Error("Something went wrong")
}
```

## Advanced Usage

### Custom Minimum Level

Only count logs at INFO level and above:

```go
opts := &otelmetrics.Options{
    MinLevel: slog.LevelInfo,
}
handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, opts)
```

### Custom Label Attributes

Use specific log attributes as OpenTelemetry labels:

```go
// Create meter and counter
meter := provider.Meter("example/logs")
counter, _ := meter.Int64Counter(
    "log_messages",
    metric.WithDescription("Number of log messages by level and service"),
)

opts := &otelmetrics.Options{
    MinLevel:        slog.LevelInfo,
    LabelAttributes: []string{"service", "component"},
}
handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, opts)

logger := slog.New(handler)

// These attributes will become OpenTelemetry labels
logger.Info("Request processed", 
    "service", "api-gateway",
    "component", "http-handler")
```

This creates metrics with attributes like:
```
log_messages{level="INFO",service="api-gateway",component="http-handler"} 1
```

## API Reference

### Types

#### `Options`
Configuration options for the handler.

```go
type Options struct {
    MinLevel        slog.Level // Minimum log level to record
    LabelAttributes []string   // Attributes to use as labels
}
```

#### `SlogHandler`
The main handler that wraps another `slog.Handler` and adds metrics collection.

### Functions

#### `NewHandler(base slog.Handler, counter metric.Int64Counter) slog.Handler`
Creates a new handler with default options (minimum level: INFO).

#### `NewHandlerWithOptions(base slog.Handler, counter metric.Int64Counter, opts *Options) slog.Handler`
Creates a new handler with custom options.

#### `DefaultOptions() *Options`
Returns default configuration options.

## OpenTelemetry Counter Requirements

The OpenTelemetry counter must be an `Int64Counter` created from a meter. The handler automatically adds a "level" attribute. When using `LabelAttributes`, those attributes are also added:

```go
// Basic counter (level attribute added automatically)
meter := provider.Meter("example/logs")
counter, _ := meter.Int64Counter(
    "log_messages",
    metric.WithDescription("Number of log messages"),
)

// When using custom attributes, the handler adds both "level" and custom attributes
opts := &otelmetrics.Options{
    LabelAttributes: []string{"service", "component"},
}
// Results in attributes: level, service, component
```

## Examples

### With Different Base Handlers

```go
// With text handler
textHandler := slog.NewTextHandler(os.Stdout, nil)
handler := otelmetrics.NewHandler(textHandler, counter)

// With JSON handler  
jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
handler := otelmetrics.NewHandler(jsonHandler, counter)

// With custom handler
customHandler := &MyCustomHandler{}
handler := otelmetrics.NewHandler(customHandler, counter)
```

### Service-Specific Metrics

```go
meter := provider.Meter("example/logs")
counter, _ := meter.Int64Counter(
    "service_log_messages",
    metric.WithDescription("Total log messages by service and level"),
)

opts := &otelmetrics.Options{
    LabelAttributes: []string{"service"},
}
handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, opts)

logger := slog.New(handler)
logger.Info("User logged in", "service", "auth")
logger.Error("Database error", "service", "user-db")
```

### Complete OpenTelemetry Setup

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"
    
    "github.com/fujiwara/sloghandler/otelmetrics"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
    "go.opentelemetry.io/otel/metric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
    // Setup OpenTelemetry
    exporter, _ := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
    reader := sdkmetric.NewPeriodicReader(
        exporter,
        sdkmetric.WithInterval(5*time.Second),
    )
    provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
    otel.SetMeterProvider(provider)

    // Create meter and counter
    meter := provider.Meter("myapp/logs")
    counter, _ := meter.Int64Counter(
        "log_messages_total",
        metric.WithDescription("Total number of log messages by level"),
    )

    // Create handler with custom options
    baseHandler := slog.NewJSONHandler(os.Stdout, nil)
    opts := &otelmetrics.Options{
        MinLevel:        slog.LevelInfo,
        LabelAttributes: []string{"service"},
    }
    handler := otelmetrics.NewHandlerWithOptions(baseHandler, counter, opts)

    // Use the logger
    logger := slog.New(handler)
    logger.Info("Application started", "service", "web-server")
    logger.Error("Database connection failed", "service", "database")

    // Cleanup
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        provider.Shutdown(ctx)
    }()
}
```

## License

This package is part of the [sloghandler](https://github.com/fujiwara/sloghandler) project.