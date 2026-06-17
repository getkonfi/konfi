package setup

import (
	"context"

	"github.com/getkonfi/konfi/theme"
)

// InitTheme creates the Theme from the configured palette preference.
func InitTheme(_ context.Context, app *App) error {
	name := "rose pine"
	if app.Config != nil && app.Config.Theme != "" {
		name = app.Config.Theme
	}

	p := theme.PaletteByName(name)
	app.Theme = theme.NewTheme(p)
	return nil
}
