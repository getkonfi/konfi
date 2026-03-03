package setup

import (
	"context"
	"os/exec"
	"time"

	"github.com/emin/konfigurator/konfables/alacritty"
	"github.com/emin/konfigurator/konfables/ghostty"
	"github.com/emin/konfigurator/konfables/gnome"
	"github.com/emin/konfigurator/konfables/hyprland"
	"github.com/emin/konfigurator/konfables/konfigurator"
	"github.com/emin/konfigurator/konfables/starship"
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/setup/cst"
)

// versioned matches konfables.Versioned — defined locally to avoid import cycle.
type versioned interface {
	Version(ctx context.Context) (string, error)
}

type konfableEntry struct {
	binary string
	create func() Konfable
	system bool // virtual konfable, skip PATH detection
	probe  func() bool // optional detection beyond PATH check
}

var allKonfables = []konfableEntry{
	{"ghostty", func() Konfable {
		return ghostty.New(pkg.NewFilePersister(pkg.XDGConfigPath("ghostty", "config")))
	}, false, nil},
	{"starship", func() Konfable {
		return starship.New(pkg.NewFilePersister(starship.DefaultConfigPath()))
	}, false, nil},
	{"alacritty", func() Konfable {
		return alacritty.New(pkg.NewFilePersister(pkg.XDGConfigPath("alacritty", "alacritty.toml")))
	}, false, nil},
	{"Hyprland", func() Konfable {
		return hyprland.New(pkg.NewFilePersister(pkg.XDGConfigPath("hypr", "hyprland.conf")))
	}, false, nil},
	{"", func() Konfable {
		return konfigurator.New(pkg.NewFilePersister(
			cst.ConfigFilePath(),
			pkg.WithDefaultContent([]byte("theme: catppuccin\nlog_level: info\n")),
		))
	}, true, nil},
	{"gsettings", func() Konfable {
		return gnome.New(&gnome.GsettingsPersister{})
	}, false, probeGsettings},
}

// probeGsettings checks whether the gnome desktop interface schema is available.
func probeGsettings() bool {
	out, err := exec.Command("gsettings", "list-keys", "org.gnome.desktop.interface").Output()
	return err == nil && len(out) > 0
}

// KonfableInfo pairs a konfable with its registration metadata.
type KonfableInfo struct {
	Konfable Konfable
	System   bool
}

// AllKonfablesWithInfo returns every registered konfable with metadata.
// all entries are included regardless of probe result — probes only gate
// detection (installed status), not sidebar visibility.
func AllKonfablesWithInfo() []KonfableInfo {
	out := make([]KonfableInfo, 0, len(allKonfables))
	for _, k := range allKonfables {
		out = append(out, KonfableInfo{Konfable: k.create(), System: k.system})
	}
	return out
}

// AllKonfables returns every registered konfable without probing PATH.
func AllKonfables() []Konfable {
	all := make([]Konfable, 0, len(allKonfables))
	for _, k := range allKonfables {
		all = append(all, k.create())
	}
	return all
}

// InitDetection probes PATH for known konfable binaries and populates app.Detected.
// system entries bypass PATH detection and are always included.
func InitDetection(_ context.Context, app *App) error {
	for _, k := range allKonfables {
		// skip entries that require a probe and fail it
		if k.probe != nil && !k.probe() {
			continue
		}

		if k.system {
			app.Detected = append(app.Detected, k.create())
			continue
		}
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
