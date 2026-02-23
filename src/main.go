package main

import (
	"context"
	"log"
	"os"

	"github.com/emin/konfigurator/setup"
	"github.com/emin/konfigurator/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	units := []setup.Unit{
		{Name: "Config", InitFn: setup.InitConfig},
		{Name: "Logger", InitFn: setup.InitZerolog},
		{Name: "Theme", InitFn: setup.InitTheme},
		{Name: "Detection", InitFn: setup.InitDetection},
	}

	app, err := setup.InitApp(context.Background(), units)
	if err != nil {
		log.Fatalf("init: %v", err)
	}

	root := ui.NewRoot(app)
	p := tea.NewProgram(root, tea.WithAltScreen())

	// allow watcher callbacks to inject messages into the event loop
	if pr, ok := root.(ui.ProgramSetter); ok {
		pr.SetProgram(p)
	}

	if _, err := p.Run(); err != nil {
		app.Logger.Error().Err(err).Msg("tui crashed")
		app.Shutdown()
		os.Exit(1)
	}

	app.Shutdown()
}
