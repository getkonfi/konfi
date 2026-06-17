package setup

import (
	"context"
	"fmt"
	"os"

	"github.com/getkonfi/konfi/setup/cst"

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

	// console writer to file for readable dev logs. only warn and above are
	// recorded — the log file is a problem record, not an activity trace.
	w := zerolog.ConsoleWriter{Out: f, TimeFormat: "15:04:05"}
	logger := zerolog.New(w).With().Timestamp().Logger().Level(zerolog.WarnLevel)

	app.Logger = &logger
	app.closeFns = append(app.closeFns, func(_ context.Context, _ *App) error {
		return f.Close()
	})

	return nil
}
