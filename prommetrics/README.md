# prommetrics

A Prometheus metrics handler for Go's `log/slog` package that automatically counts log messages by level and custom attributes.

## Features

- **Automatic metrics collection**: Counts log messages by log level (DEBUG, INFO, WARN, ERROR)
- **Configurable minimum level**: Only count logs above a specified level
- **Custom label attributes**: Use specific log attributes as Prometheus labels
- **Zero initialization**: All metric levels start at 0 for consistent metric output
- **Wraps existing handlers**: Works with any `slog.Handler` implementation

## Installation

```bash
go get github.com/fujiwara/sloghandler/prommetrics
```

## Quick Start

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/fujiwara/sloghandler/prommetrics"
    "github.com/prometheus/client_golang/prometheus"
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

    // Create base handler (JSON handler writing to stdout)
    baseHandler := slog.NewJSONHandler(os.Stdout, nil)
    
    // Wrap with metrics handler
    handler := prommetrics.NewHandler(baseHandler, counter)
    
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
opts := &prommetrics.Options{
    MinLevel: slog.LevelInfo,
}
handler := prommetrics.NewHandlerWithOptions(baseHandler, counter, opts)
```

### Custom Label Attributes

Use specific log attributes as Prometheus labels:

```go
// Counter with additional labels
counter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "log_messages_total",
        Help: "Total number of log messages by level and service",
    },
    []string{"level", "service", "component"},
)

opts := &prommetrics.Options{
    MinLevel:        slog.LevelInfo,
    LabelAttributes: []string{"service", "component"},
}
handler := prommetrics.NewHandlerWithOptions(baseHandler, counter, opts)

logger := slog.New(handler)

// These attributes will become Prometheus labels
logger.Info("Request processed", 
    "service", "api-gateway",
    "component", "http-handler")
```

This creates metrics like:
```
log_messages_total{level="INFO",service="api-gateway",component="http-handler"} 1
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

#### `NewHandler(base slog.Handler, counter *prometheus.CounterVec) slog.Handler`
Creates a new handler with default options (minimum level: INFO).

#### `NewHandlerWithOptions(base slog.Handler, counter *prometheus.CounterVec, opts *Options) slog.Handler`
Creates a new handler with custom options.

#### `DefaultOptions() *Options`
Returns default configuration options.

## Prometheus Counter Requirements

The Prometheus counter must have at least a "level" label. When using `LabelAttributes`, include those labels as well:

```go
// Basic counter (level only)
counter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "log_messages_total",
        Help: "Total number of log messages",
    },
    []string{"level"},
)

// Counter with custom attributes
counter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "log_messages_total", 
        Help: "Total number of log messages",
    },
    []string{"level", "service", "component"}, // Order matters
)
```

## Examples

### With Different Base Handlers

```go
// With text handler
textHandler := slog.NewTextHandler(os.Stdout, nil)
handler := prommetrics.NewHandler(textHandler, counter)

// With JSON handler  
jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
handler := prommetrics.NewHandler(jsonHandler, counter)

// With custom handler
customHandler := &MyCustomHandler{}
handler := prommetrics.NewHandler(customHandler, counter)
```

### Service-Specific Metrics

```go
counter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "service_log_messages_total",
        Help: "Total log messages by service and level",
    },
    []string{"level", "service"},
)

opts := &prommetrics.Options{
    LabelAttributes: []string{"service"},
}
handler := prommetrics.NewHandlerWithOptions(baseHandler, counter, opts)

logger := slog.New(handler)
logger.Info("User logged in", "service", "auth")
logger.Error("Database error", "service", "user-db")
```

## License

This package is part of the [sloghandler](https://github.com/fujiwara/sloghandler) project.