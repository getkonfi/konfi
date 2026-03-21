package setup

import (
	"context"
	"os/exec"
	"runtime"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/emin/konfigurator/konfables/alacritty"
	"github.com/emin/konfigurator/konfables/dconf"
	"github.com/emin/konfigurator/konfables/ghostty"
	"github.com/emin/konfigurator/konfables/git"
	"github.com/emin/konfigurator/konfables/gnome"
	"github.com/emin/konfigurator/konfables/helix"
	"github.com/emin/konfigurator/konfables/hyprland"
	"github.com/emin/konfigurator/konfables/kitty"
	"github.com/emin/konfigurator/konfables/pacman"
	"github.com/emin/konfigurator/konfables/konfigurator"
	"github.com/emin/konfigurator/konfables/rio"
	"github.com/emin/konfigurator/konfables/ssh"
	"github.com/emin/konfigurator/konfables/starship"
	"github.com/emin/konfigurator/konfables/tmux"
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
	{"dconf", func() Konfable {
		return dconf.New(&dconf.DconfPersister{})
	}, false, probeDconf},
	{"kitty", func() Konfable {
		return kitty.New(pkg.NewFilePersister(pkg.XDGConfigPath("kitty", "kitty.conf")))
	}, false, nil},
	{"hx", func() Konfable {
		return helix.New(pkg.NewFilePersister(pkg.XDGConfigPath("helix", "config.toml")))
	}, false, nil},
	{"rio", func() Konfable {
		return rio.New(pkg.NewFilePersister(pkg.XDGConfigPath("rio", "config.toml")))
	}, false, nil},
	{"git", func() Konfable {
		return git.New(pkg.NewFilePersister(git.DefaultConfigPath()))
	}, false, nil},
	{"tmux", func() Konfable {
		return tmux.New(pkg.NewFilePersister(tmux.DefaultConfigPath()))
	}, false, nil},
	{"ssh", func() Konfable {
		return ssh.New(pkg.NewFilePersister(ssh.DefaultConfigPath()))
	}, false, nil},
	{"pacman", func() Konfable {
		return pacman.New(pkg.NewFilePersister("/etc/pacman.conf"))
	}, false, nil},
}

// probeGsettings checks whether the gnome desktop interface schema is available.
func probeGsettings() bool {
	out, err := exec.Command("gsettings", "list-keys", "org.gnome.desktop.interface").Output()
	return err == nil && len(out) > 0
}

// probeDconf checks whether the wm.preferences schema is available via dconf.
func probeDconf() bool {
	out, err := exec.Command("dconf", "read", "/org/gnome/desktop/wm/preferences/button-layout").Output()
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

// detectedEntry holds a detected konfable alongside its original registration
// index so results can be sorted back into registry order after parallel collection.
type detectedEntry struct {
	index int
	inst  Konfable
}

// InitDetection probes PATH for known konfable binaries and populates app.Detected.
// system entries bypass PATH detection and are always included.
// detection runs in parallel, capped at NumCPU goroutines.
func InitDetection(ctx context.Context, app *App) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	var mu sync.Mutex
	var results []detectedEntry

	for i, k := range allKonfables {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}

			if k.probe != nil && !k.probe() {
				return nil
			}

			if k.system {
				mu.Lock()
				results = append(results, detectedEntry{index: i, inst: k.create()})
				mu.Unlock()
				return nil
			}

			if _, err := exec.LookPath(k.binary); err == nil {
				inst := k.create()

				if v, ok := inst.(versioned); ok {
					vCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
					ver, err := v.Version(vCtx)
					cancel()
					if err == nil {
						mu.Lock()
						app.Versions[inst.Name()] = ver
						mu.Unlock()
					} else if app.Logger != nil {
						app.Logger.Warn().Err(err).Str("app", inst.Name()).Msg("version probe failed")
					}
				}

				mu.Lock()
				results = append(results, detectedEntry{index: i, inst: inst})
				mu.Unlock()
			}

			return nil
		})
	}

	_ = g.Wait()

	// restore registration order
	sort.Slice(results, func(a, b int) bool {
		return results[a].index < results[b].index
	})
	app.Detected = make([]Konfable, len(results))
	for i, r := range results {
		app.Detected[i] = r.inst
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

