package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/getkonfi/konfi/selfupdate"
	"github.com/getkonfi/konfi/setup"
	"github.com/getkonfi/konfi/setup/cst"
	"github.com/getkonfi/konfi/ui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if handled, code := runCommand(context.Background(), os.Args[1:]); handled {
		os.Exit(code)
	}

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
	p := tea.NewProgram(root)

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

func runCommand(ctx context.Context, args []string) (bool, int) {
	if len(args) == 0 || args[0] != "update" {
		return false, 0
	}

	fs := flag.NewFlagSet("konfi update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		checkOnly bool
		version   string
		repo      string
	)
	fs.BoolVar(&checkOnly, "check", false, "check whether an update is available")
	fs.StringVar(&version, "version", "", "install a specific release version or tag")
	fs.StringVar(&repo, "repo", envOrDefault("KONFI_REPO", selfupdate.DefaultRepo), "GitHub repo")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: konfi update [--check] [--version VERSION] [--repo OWNER/REPO]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return true, 0
		}
		return true, 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "konfi update: unexpected argument: %s\n", fs.Arg(0))
		fs.Usage()
		return true, 2
	}

	err := selfupdate.Run(ctx, selfupdate.Options{
		Repo:           repo,
		Version:        version,
		CurrentVersion: cst.AppVersion,
		CheckOnly:      checkOnly,
		Out:            os.Stdout,
	})
	if err != nil {
		selfupdate.FprintError(os.Stderr, err)
		return true, 1
	}
	return true, 0
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
