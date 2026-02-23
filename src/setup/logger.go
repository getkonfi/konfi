package setup

import (
	"context"
	"fmt"
	"os"

	"github.com/emin/konfigurator/setup/cst"

	"github.com/rs/zerolog"
)

// InitZerolog sets up zerolog writing to a file (TUI owns stdout).
func InitZerolog(_ context.Context, app *App) error {
	dir := cst.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	path := cst.LogFilePath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	level := zerolog.InfoLevel
	if app.Config != nil {
		switch app.Config.LogLevel {
		case "debug":
			level = zerolog.DebugLevel
		case "warn":
			level = zerolog.WarnLevel
		case "error":
			level = zerolog.ErrorLevel
		case "trace":
			level = zerolog.TraceLevel
		}
	}

	// console writer to file for readable dev logs
	w := zerolog.ConsoleWriter{Out: f, TimeFormat: "15:04:05"}
	logger := zerolog.New(w).With().Timestamp().Logger().Level(level)

	app.Logger = &logger
	app.closeFns = append(app.closeFns, func(_ context.Context, _ *App) error {
		return f.Close()
	})

	return nil
}
