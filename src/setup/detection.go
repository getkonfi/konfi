package setup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/getkonfi/konfi/konfables/alacritty"
	"github.com/getkonfi/konfi/konfables/brew"
	"github.com/getkonfi/konfi/konfables/dconf"
	"github.com/getkonfi/konfi/konfables/fuzzel"
	"github.com/getkonfi/konfi/konfables/ghostty"
	"github.com/getkonfi/konfi/konfables/git"
	"github.com/getkonfi/konfi/konfables/gnome"
	"github.com/getkonfi/konfi/konfables/gtk"
	"github.com/getkonfi/konfi/konfables/helix"
	"github.com/getkonfi/konfi/konfables/hyprland"
	"github.com/getkonfi/konfi/konfables/kitty"
	"github.com/getkonfi/konfi/konfables/konfi"
	"github.com/getkonfi/konfi/konfables/pacman"
	"github.com/getkonfi/konfi/konfables/powerlevel10k"
	"github.com/getkonfi/konfi/konfables/rio"
	"github.com/getkonfi/konfi/konfables/ssh"
	"github.com/getkonfi/konfi/konfables/sshd"
	"github.com/getkonfi/konfi/konfables/starship"
	"github.com/getkonfi/konfi/konfables/tmux"
	"github.com/getkonfi/konfi/konfables/waybar"
	"github.com/getkonfi/konfi/konfables/yazi"
	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/setup/cst"
)

// versioned matches konfables.Versioned — defined locally to avoid import cycle.
type versioned interface {
	Version(ctx context.Context) (string, error)
}

type konfableEntry struct {
	binary string
	create func(*KonfConfig) Konfable
	system bool        // virtual konfable, skip PATH detection
	probe  func() bool // optional detection beyond PATH check
}

var allKonfables = []konfableEntry{
	{"ghostty", func(cfg *KonfConfig) Konfable {
		return ghostty.New(newFilePersister(ghostty.DefaultConfigPath(), cfg))
	}, false, nil},
	{"starship", func(cfg *KonfConfig) Konfable {
		return starship.New(newFilePersister(starship.DefaultConfigPath(), cfg))
	}, false, nil},
	{"zsh", func(cfg *KonfConfig) Konfable {
		return powerlevel10k.New(newFilePersister(powerlevel10k.DefaultConfigPath(), cfg))
	}, false, probePowerlevel10k},
	{"alacritty", func(cfg *KonfConfig) Konfable {
		return alacritty.New(newFilePersister(alacritty.DefaultConfigPath(), cfg))
	}, false, nil},
	{"Hyprland", func(cfg *KonfConfig) Konfable {
		return hyprland.New(newFilePersister(pkg.XDGConfigPath("hypr", "hyprland.conf"), cfg))
	}, false, nil},
	// hypridle hidden for now — re-add to enable detection
	{"fuzzel", func(cfg *KonfConfig) Konfable {
		return fuzzel.New(newFilePersister(fuzzel.DefaultConfigPath(), cfg))
	}, false, nil},
	{"waybar", func(cfg *KonfConfig) Konfable {
		return waybar.New(newFilePersister(waybar.DefaultConfigPath(), cfg))
	}, false, nil},
	{"", func(cfg *KonfConfig) Konfable {
		return konfi.New(newFilePersister(
			cst.ConfigFilePath(),
			cfg,
			pkg.WithDefaultContent([]byte("theme: catppuccin\nlog_level: info\nbackup_limit: 5\n")),
		))
	}, true, nil},
	{"gsettings", func(_ *KonfConfig) Konfable {
		return gnome.New(gnome.NewPersister())
	}, false, probeGnome},
	{"dconf", func(_ *KonfConfig) Konfable {
		return dconf.New(dconf.NewPersister())
	}, false, probeDconf},
	{"kitty", func(cfg *KonfConfig) Konfable {
		return kitty.New(newFilePersister(kitty.DefaultConfigPath(), cfg))
	}, false, nil},
	{"hx", func(cfg *KonfConfig) Konfable {
		return helix.New(newFilePersister(pkg.XDGConfigPath("helix", "config.toml"), cfg))
	}, false, nil},
	{"yazi", func(cfg *KonfConfig) Konfable {
		return yazi.New(newFilePersister(yazi.DefaultConfigPath(), cfg))
	}, false, nil},
	{"rio", func(cfg *KonfConfig) Konfable {
		return rio.New(newFilePersister(rio.DefaultConfigPath(), cfg))
	}, false, nil},
	{"git", func(cfg *KonfConfig) Konfable {
		return git.New(newFilePersister(git.DefaultConfigPath(), cfg))
	}, false, nil},
	{"gtk-launch", func(cfg *KonfConfig) Konfable {
		primary, mirrors := gtk.ResolvePaths()
		p := gtk.NewMirrorPersister(primary, mirrors...)
		p.SetBackupLimit(cfg.EffectiveBackupLimit())
		return gtk.New(p)
	}, false, nil},
	{"tmux", func(cfg *KonfConfig) Konfable {
		return tmux.New(newFilePersister(tmux.DefaultConfigPath(), cfg))
	}, false, nil},
	{"ssh", func(cfg *KonfConfig) Konfable {
		return ssh.New(newFilePersister(ssh.DefaultConfigPath(), cfg, pkg.WithMissingContent([]byte(""))))
	}, false, nil},
	{sshd.BinaryPath(), func(cfg *KonfConfig) Konfable {
		return sshd.New(newFilePersister(sshd.DefaultConfigPath(), cfg, pkg.WithMissingContent([]byte(""))))
	}, false, nil},
	{"pacman", func(cfg *KonfConfig) Konfable {
		return pacman.New(newFilePersister("/etc/pacman.conf", cfg))
	}, false, nil},
	{"brew", func(cfg *KonfConfig) Konfable {
		return brew.New(newFilePersister(brew.DefaultConfigPath(), cfg))
	}, false, nil},
}

func newFilePersister(path string, cfg *KonfConfig, opts ...pkg.FilePersisterOption) *pkg.FilePersister {
	allOpts := make([]pkg.FilePersisterOption, 0, len(opts)+1)
	allOpts = append(allOpts, pkg.WithBackupLimit(cfg.EffectiveBackupLimit()))
	allOpts = append(allOpts, opts...)
	return pkg.NewFilePersister(path, allOpts...)
}

// probeGnome reports whether a GNOME desktop is actually installed. The
// org.gnome.desktop.interface schema alone is a false positive: it ships with
// gsettings-desktop-schemas, a common GTK dependency present on non-GNOME
// systems. gnome-shell in PATH is the reliable signal that GNOME is installed.
func probeGnome() bool {
	if _, err := exec.LookPath("gnome-shell"); err != nil {
		return false
	}
	out, err := exec.Command("gsettings", "list-keys", "org.gnome.desktop.interface").Output()
	return err == nil && len(out) > 0
}

// probeDconf checks whether the wm.preferences schema is available via dconf.
func probeDconf() bool {
	out, err := exec.Command("dconf", "read", "/org/gnome/desktop/wm/preferences/button-layout").Output()
	return err == nil && len(out) > 0
}

// probePowerlevel10k checks for either the generated config or common theme install paths.
func probePowerlevel10k() bool {
	if pkg.FileExists(powerlevel10k.DefaultConfigPath()) {
		return true
	}

	var candidates []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, "powerlevel10k", "powerlevel10k.zsh-theme"),
			filepath.Join(home, ".oh-my-zsh", "custom", "themes", "powerlevel10k", "powerlevel10k.zsh-theme"),
			filepath.Join(home, ".local", "share", "zsh", "plugins", "powerlevel10k", "powerlevel10k.zsh-theme"),
		)
	}
	candidates = append(candidates,
		"/opt/homebrew/share/powerlevel10k/powerlevel10k.zsh-theme",
		"/usr/local/share/powerlevel10k/powerlevel10k.zsh-theme",
		"/usr/share/zsh-theme-powerlevel10k/powerlevel10k.zsh-theme",
		"/usr/share/zsh/plugins/powerlevel10k/powerlevel10k.zsh-theme",
	)

	for _, path := range candidates {
		if pkg.FileExists(path) {
			return true
		}
	}
	return false
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
	return AllKonfablesWithInfoConfig(nil)
}

// AllKonfablesWithInfoConfig returns every registered konfable with metadata.
func AllKonfablesWithInfoConfig(cfg *KonfConfig) []KonfableInfo {
	out := make([]KonfableInfo, 0, len(allKonfables))
	for _, k := range allKonfables {
		out = append(out, KonfableInfo{Konfable: k.create(cfg), System: k.system})
	}
	return out
}

// AllKonfables returns every registered konfable without probing PATH.
func AllKonfables() []Konfable {
	return AllKonfablesConfig(nil)
}

// AllKonfablesConfig returns every registered konfable without probing PATH.
func AllKonfablesConfig(cfg *KonfConfig) []Konfable {
	all := make([]Konfable, 0, len(allKonfables))
	for _, k := range allKonfables {
		all = append(all, k.create(cfg))
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
				results = append(results, detectedEntry{index: i, inst: k.create(app.Config)})
				mu.Unlock()
				return nil
			}

			if _, err := exec.LookPath(k.binary); err == nil {
				inst := k.create(app.Config)

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
