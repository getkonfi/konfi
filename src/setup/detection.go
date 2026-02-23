package setup

import (
	"context"
	"os/exec"
	"time"

	"github.com/emin/konfigurator/konfables/alacritty"
	"github.com/emin/konfigurator/konfables/ghostty"
	"github.com/emin/konfigurator/konfables/hyprland"
	"github.com/emin/konfigurator/konfables/starship"
)

// versioned matches konfables.Versioned — defined locally to avoid import cycle.
type versioned interface {
	Version(ctx context.Context) (string, error)
}

type konfableEntry struct {
	binary string
	create func() Konfable
}

var allKonfables = []konfableEntry{
	{"ghostty", func() Konfable { return ghostty.New() }},
	{"starship", func() Konfable { return starship.New() }},
	{"alacritty", func() Konfable { return alacritty.New() }},
	{"Hyprland", func() Konfable { return hyprland.New() }},
}

// AllKonfables returns every registered konfable without probing PATH.
func AllKonfables() []Konfable {
	all := make([]Konfable, len(allKonfables))
	for i, k := range allKonfables {
		all[i] = k.create()
	}
	return all
}

// InitDetection probes PATH for known konfable binaries and populates app.Detected.
func InitDetection(_ context.Context, app *App) error {
	for _, k := range allKonfables {
		if _, err := exec.LookPath(k.binary); err == nil {
			inst := k.create()
			app.Detected = append(app.Detected, inst)

			if v, ok := inst.(versioned); ok {
				probeVersion(app, inst.Name(), v)
			}
		}
	}

	if app.Logger != nil {
		names := make([]string, len(app.Detected))
		for i, d := range app.Detected {
			names[i] = d.Name()
		}
		app.Logger.Info().
			Strs("apps", names).
			Fields(app.Versions).
			Msg("detection complete")
	}

	return nil
}

func probeVersion(app *App, name string, v versioned) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ver, err := v.Version(ctx)
	if err != nil {
		if app.Logger != nil {
			app.Logger.Warn().Err(err).Str("app", name).Msg("version probe failed")
		}
		return
	}
	app.Versions[name] = ver
}
