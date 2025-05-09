# sloghandler

sloghandler is a simple text handler for the [log/slog](https://pkg.go.dev/log/slog) package.
It provides colored output for log levels and customizable formatting.

## Features

- Simple, readable text output format
- Colorized output for warning and error levels
- Support for log level filtering
- Custom time format
- Preservation of log attributes

## Installation

```
go get github.com/fujiwara/sloghandler
```

## Usage

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

## Testing

The package includes comprehensive tests. Run them with:

```
go test -v
```

## LICENSE

MIT

## Author

Fujiwara
