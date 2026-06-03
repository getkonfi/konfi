package setup

import (
	"context"

	"github.com/eminert/konfi/theme"
)

// InitTheme creates the Theme from the configured palette preference.
func InitTheme(_ context.Context, app *App) error {
	name := "catppuccin"
	if app.Config != nil && app.Config.Theme != "" {
		name = app.Config.Theme
	}

	p := theme.PaletteByName(name)
	app.Theme = theme.NewTheme(p)
	return nil
}
