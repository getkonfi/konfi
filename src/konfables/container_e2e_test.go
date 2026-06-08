//go:build container_e2e

package konfables_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/eminert/konfi/konfables"
	appalacritty "github.com/eminert/konfi/konfables/alacritty"
	appclaude "github.com/eminert/konfi/konfables/claude"
	appdconf "github.com/eminert/konfi/konfables/dconf"
	appfuzzel "github.com/eminert/konfi/konfables/fuzzel"
	appghostty "github.com/eminert/konfi/konfables/ghostty"
	appgit "github.com/eminert/konfi/konfables/git"
	appgnome "github.com/eminert/konfi/konfables/gnome"
	appgtk "github.com/eminert/konfi/konfables/gtk"
	apphelix "github.com/eminert/konfi/konfables/helix"
	apphyprland "github.com/eminert/konfi/konfables/hyprland"
	appkitty "github.com/eminert/konfi/konfables/kitty"
	appkonfi "github.com/eminert/konfi/konfables/konfi"
	apppacman "github.com/eminert/konfi/konfables/pacman"
	apprio "github.com/eminert/konfi/konfables/rio"
	appssh "github.com/eminert/konfi/konfables/ssh"
	appsshd "github.com/eminert/konfi/konfables/sshd"
	appstarship "github.com/eminert/konfi/konfables/starship"
	apptmux "github.com/eminert/konfi/konfables/tmux"
	appwaybar "github.com/eminert/konfi/konfables/waybar"
	appyazi "github.com/eminert/konfi/konfables/yazi"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/setup"
	"gopkg.in/yaml.v3"
)

type containerCase struct {
	name         string
	newApp       func(t *testing.T, root string) konfables.Konfable
	sample       []byte
	existingKey  string
	existingWant string
	replaceWrite string
	replaceWant  string
	addKey       string
	addWrite     string
	addWant      string
	deleteKey    string
	survivorKey  string
	survivorWant string
	fileBacked   bool
}

func TestContainerE2E(t *testing.T) {
	requireArchContainer(t)

	root := t.TempDir()
	cases := containerCases(t)
	assertRegisteredCoverage(t, cases)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := tc.newApp(t, filepath.Join(root, tc.name))
			assertSchema(t, app)

			p := app.Parser()
			exerciseParser(t, p, tc)

			if tc.fileBacked {
				exerciseFileBackedEdit(t, app, tc)
			}
		})
	}
}

func requireArchContainer(t *testing.T) {
	t.Helper()
	if os.Getenv("KONFI_CONTAINER_E2E_ALLOW_HOST") == "1" {
		return
	}

	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		t.Fatalf("read /etc/os-release: %v", err)
	}
	if !bytes.Contains(data, []byte("\nID=arch\n")) &&
		!bytes.Contains(data, []byte("\nID=\"arch\"\n")) {
		t.Fatalf("container e2e must run on Arch; set KONFI_CONTAINER_E2E_ALLOW_HOST=1 only for local compile/debug runs")
	}
}

func assertRegisteredCoverage(t *testing.T, cases []containerCase) {
	t.Helper()
	covered := make(map[string]int, len(cases))
	for _, tc := range cases {
		covered[tc.name]++
	}

	for _, entry := range setup.AllKonfablesWithInfo() {
		name := entry.Konfable.Name()
		if covered[name] == 0 {
			t.Errorf("registered konfable %q has no container e2e case", name)
		}
		delete(covered, name)
	}

	var extras []string
	for name := range covered {
		extras = append(extras, name)
	}
	sort.Strings(extras)
	for _, name := range extras {
		t.Errorf("container e2e case %q is not registered in setup detection", name)
	}
}

func assertSchema(t *testing.T, app konfables.Konfable) {
	t.Helper()

	info := app.Info()
	if info.Name != app.Name() {
		t.Fatalf("Info().Name = %q, Name() = %q", info.Name, app.Name())
	}
	if info.Format == "" {
		t.Fatalf("%s has empty format", app.Name())
	}

	raw, err := app.Schema()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	var schema pkg.Schema
	if err := yaml.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if schema.App != info.Name {
		t.Fatalf("schema app = %q, Info().Name = %q", schema.App, info.Name)
	}
	if schema.Format != info.Format {
		t.Fatalf("schema format = %q, Info().Format = %q", schema.Format, info.Format)
	}
	if len(schema.Sections) == 0 {
		t.Fatalf("%s schema has no sections", app.Name())
	}
}

func exerciseParser(t *testing.T, p konfables.Parser, tc containerCase) {
	t.Helper()

	data := bytes.Clone(tc.sample)
	got, ok := p.FindValue(data, tc.existingKey)
	if !ok || got != tc.existingWant {
		t.Fatalf("FindValue(%q) = %q, %v; want %q, true", tc.existingKey, got, ok, tc.existingWant)
	}
	if _, ok := p.FindLine(data, tc.existingKey); !ok {
		t.Fatalf("FindLine(%q) did not find existing key", tc.existingKey)
	}
	if !hasKey(p.ListKeys(data), tc.existingKey) {
		t.Fatalf("ListKeys missing existing key %q", tc.existingKey)
	}

	edited, err := p.SetValue(data, tc.existingKey, tc.replaceWrite)
	if err != nil {
		t.Fatalf("SetValue(%q): %v", tc.existingKey, err)
	}
	got, ok = p.FindValue(edited, tc.existingKey)
	if !ok || got != tc.replaceWant {
		t.Fatalf("after replace FindValue(%q) = %q, %v; want %q, true", tc.existingKey, got, ok, tc.replaceWant)
	}

	edited, err = p.SetValue(edited, tc.addKey, tc.addWrite)
	if err != nil {
		t.Fatalf("SetValue(%q): %v", tc.addKey, err)
	}
	got, ok = p.FindValue(edited, tc.addKey)
	if !ok || got != tc.addWant {
		t.Fatalf("after add FindValue(%q) = %q, %v; want %q, true", tc.addKey, got, ok, tc.addWant)
	}
	if !hasKey(p.ListKeys(edited), tc.addKey) {
		t.Fatalf("ListKeys missing added key %q", tc.addKey)
	}

	edited, err = p.DeleteKey(edited, tc.deleteKey)
	if err != nil {
		t.Fatalf("DeleteKey(%q): %v", tc.deleteKey, err)
	}
	if got, ok := p.FindValue(edited, tc.deleteKey); ok {
		t.Fatalf("deleted key %q is still present with value %q", tc.deleteKey, got)
	}
	got, ok = p.FindValue(edited, tc.survivorKey)
	if !ok || got != tc.survivorWant {
		t.Fatalf("survivor FindValue(%q) = %q, %v; want %q, true", tc.survivorKey, got, ok, tc.survivorWant)
	}
}

func exerciseFileBackedEdit(t *testing.T, app konfables.Konfable, tc containerCase) {
	t.Helper()

	path := app.ConfigPath()
	if path == "" {
		t.Fatalf("%s marked file-backed with empty config path", app.Name())
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, tc.sample, 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	ctx := context.Background()
	cf, err := pkg.NewConfigFile(ctx, app)
	if err != nil {
		t.Fatalf("load config file: %v", err)
	}
	updated, err := app.Parser().SetValue(cf.Content(), tc.existingKey, tc.replaceWrite)
	if err != nil {
		t.Fatalf("edit config file content: %v", err)
	}
	cf.SetContent(updated)
	if !cf.Dirty() {
		t.Fatal("ConfigFile should be dirty after edit")
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config before save: %v", err)
	}
	if !bytes.Equal(onDisk, tc.sample) {
		t.Fatalf("config changed before save approval\ngot:\n%s\nwant:\n%s", onDisk, tc.sample)
	}

	if err := cf.Save(ctx); err != nil {
		t.Fatalf("save config file: %v", err)
	}
	onDisk, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after save: %v", err)
	}
	got, ok := app.Parser().FindValue(onDisk, tc.existingKey)
	if !ok || got != tc.replaceWant {
		t.Fatalf("saved FindValue(%q) = %q, %v; want %q, true", tc.existingKey, got, ok, tc.replaceWant)
	}

	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if !bytes.Equal(backup, tc.sample) {
		t.Fatalf("backup mismatch\ngot:\n%s\nwant:\n%s", backup, tc.sample)
	}
}

func hasKey(keys []string, want string) bool {
	for _, key := range keys {
		if key == want {
			return true
		}
	}
	return false
}

func fixture(t *testing.T, elems ...string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(elems...))
	if err != nil {
		t.Fatalf("read fixture %s: %v", strings.Join(elems, "/"), err)
	}
	return data
}

func filePersister(path string) *pkg.FilePersister {
	return pkg.NewFilePersister(path)
}

func containerCases(t *testing.T) []containerCase {
	t.Helper()

	return []containerCase{
		{
			name: "alacritty",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appalacritty.New(filePersister(filepath.Join(root, "alacritty.toml")))
			},
			sample:       fixture(t, "alacritty", "testdata", "config.txt"),
			existingKey:  "font.size",
			existingWant: "12.0",
			replaceWrite: "14.0",
			replaceWant:  "14.0",
			addKey:       "window.title",
			addWrite:     `"Terminal"`,
			addWant:      "Terminal",
			deleteKey:    "colors.primary.background",
			survivorKey:  "font.normal.family",
			survivorWant: "JetBrains Mono",
			fileBacked:   true,
		},
		{
			name: "claude",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appclaude.New(appclaude.NewTieredPersister(
					filepath.Join(root, "settings.json"),
					filepath.Join(root, "settings.local.json"),
					filepath.Join(root, "project", ".claude", "settings.json"),
				))
			},
			sample:       sampleClaude,
			existingKey:  "model",
			existingWant: "sonnet",
			replaceWrite: "opus",
			replaceWant:  "opus",
			addKey:       "permissions.askMode",
			addWrite:     "ask",
			addWant:      "ask",
			deleteKey:    "includeCoAuthoredBy",
			survivorKey:  "permissions.defaultMode",
			survivorWant: "default",
		},
		{
			name: "dconf",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appdconf.New(filePersister(filepath.Join(root, "dconf.txt")))
			},
			sample:       sampleDconf,
			existingKey:  "/org/gnome/desktop/wm/preferences/focus-mode",
			existingWant: "click",
			replaceWrite: "sloppy",
			replaceWant:  "sloppy",
			addKey:       "/org/gnome/desktop/wm/preferences/auto-raise",
			addWrite:     "true",
			addWant:      "true",
			deleteKey:    "/org/gnome/desktop/peripherals/touchpad/tap-to-click",
			survivorKey:  "/org/gnome/desktop/wm/preferences/button-layout",
			survivorWant: "appmenu:minimize,maximize,close",
		},
		{
			name: "fuzzel",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appfuzzel.New(filePersister(filepath.Join(root, "fuzzel.ini")))
			},
			sample:       sampleFuzzel,
			existingKey:  "width",
			existingWant: "30",
			replaceWrite: "50",
			replaceWant:  "50",
			addKey:       "colors.selection-match",
			addWrite:     "f38ba8ff",
			addWant:      "f38ba8ff",
			deleteKey:    "colors.match",
			survivorKey:  "font",
			survivorWant: "monospace",
			fileBacked:   true,
		},
		{
			name: "ghostty",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appghostty.New(filePersister(filepath.Join(root, "config")))
			},
			sample:       fixture(t, "ghostty", "testdata", "config.txt"),
			existingKey:  "font-size",
			existingWant: "14",
			replaceWrite: "16",
			replaceWant:  "16",
			addKey:       "cursor-style",
			addWrite:     "block",
			addWant:      "block",
			deleteKey:    "window-decoration",
			survivorKey:  "font-family",
			survivorWant: "JetBrains Mono",
			fileBacked:   true,
		},
		{
			name: "git",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appgit.New(filePersister(filepath.Join(root, ".gitconfig")))
			},
			sample:       sampleGit,
			existingKey:  "user.name",
			existingWant: "John Doe",
			replaceWrite: "Jane Doe",
			replaceWant:  "Jane Doe",
			addKey:       "core.pager",
			addWrite:     "less",
			addWant:      "less",
			deleteKey:    "core.editor",
			survivorKey:  "user.email",
			survivorWant: "john@example.com",
			fileBacked:   true,
		},
		{
			name: "gtk",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appgtk.New(appgtk.NewMirrorPersister(filepath.Join(root, "settings.ini")))
			},
			sample:       sampleGTK,
			existingKey:  "Settings.gtk-theme-name",
			existingWant: "Adwaita-dark",
			replaceWrite: "Adwaita",
			replaceWant:  "Adwaita",
			addKey:       "Settings.gtk-enable-animations",
			addWrite:     "false",
			addWant:      "false",
			deleteKey:    "Settings.gtk-icon-theme-name",
			survivorKey:  "Settings.gtk-font-name",
			survivorWant: "JetBrainsMono Nerd Font 11",
			fileBacked:   true,
		},
		{
			name: "gnome",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appgnome.New(filePersister(filepath.Join(root, "gnome.txt")))
			},
			sample:       sampleGnome,
			existingKey:  "org.gnome.desktop.interface/color-scheme",
			existingWant: "prefer-dark",
			replaceWrite: "default",
			replaceWant:  "default",
			addKey:       "org.gnome.desktop.interface/font-name",
			addWrite:     "Inter 11",
			addWant:      "Inter 11",
			deleteKey:    "org.gnome.desktop.interface/cursor-size",
			survivorKey:  "org.gnome.desktop.interface/gtk-theme",
			survivorWant: "Adwaita",
		},
		{
			name: "helix",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return apphelix.New(filePersister(filepath.Join(root, "config.toml")))
			},
			sample:       fixture(t, "helix", "testdata", "config.toml"),
			existingKey:  "theme",
			existingWant: "gruvbox",
			replaceWrite: `"catppuccin"`,
			replaceWant:  "catppuccin",
			addKey:       "editor.cursor-shape.select",
			addWrite:     `"underline"`,
			addWant:      "underline",
			deleteKey:    "editor.cursor-shape.normal",
			survivorKey:  "editor.mouse",
			survivorWant: "false",
			fileBacked:   true,
		},
		{
			name: "hyprland",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return apphyprland.New(filePersister(filepath.Join(root, "hyprland.conf")))
			},
			sample:       fixture(t, "hyprland", "testdata", "config.txt"),
			existingKey:  "general.border_size",
			existingWant: "2",
			replaceWrite: "4",
			replaceWant:  "4",
			addKey:       "misc.disable_hyprland_logo",
			addWrite:     "true",
			addWant:      "true",
			deleteKey:    "decoration.rounding",
			survivorKey:  "$mainMod",
			survivorWant: "SUPER",
			fileBacked:   true,
		},
		{
			name: "kitty",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appkitty.New(filePersister(filepath.Join(root, "kitty.conf")))
			},
			sample:       fixture(t, "kitty", "testdata", "config.conf"),
			existingKey:  "font_size",
			existingWant: "14.0",
			replaceWrite: "16.0",
			replaceWant:  "16.0",
			addKey:       "cursor_shape",
			addWrite:     "beam",
			addWant:      "beam",
			deleteKey:    "hide_window_decorations",
			survivorKey:  "font_family",
			survivorWant: "JetBrains Mono",
			fileBacked:   true,
		},
		{
			name: "konfi",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appkonfi.New(filePersister(filepath.Join(root, "config.yaml")))
			},
			sample:       fixture(t, "konfi", "testdata", "config.txt"),
			existingKey:  "theme",
			existingWant: "catppuccin",
			replaceWrite: "tokyonight",
			replaceWant:  "tokyonight",
			addKey:       "some_key",
			addWrite:     "value",
			addWant:      "value",
			deleteKey:    "log_level",
			survivorKey:  "theme",
			survivorWant: "tokyonight",
			fileBacked:   true,
		},
		{
			name: "pacman",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return apppacman.New(filePersister(filepath.Join(root, "pacman.conf")))
			},
			sample:       samplePacman,
			existingKey:  "options.ParallelDownloads",
			existingWant: "5",
			replaceWrite: "10",
			replaceWant:  "10",
			addKey:       "options.XferCommand",
			addWrite:     "/usr/bin/curl -L -C - -f -o %%o %%u",
			addWant:      "/usr/bin/curl -L -C - -f -o %%o %%u",
			deleteKey:    "options.Color",
			survivorKey:  "options.HoldPkg",
			survivorWant: "pacman glibc",
			fileBacked:   true,
		},
		{
			name: "rio",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return apprio.New(filePersister(filepath.Join(root, "config.toml")))
			},
			sample:       fixture(t, "rio", "testdata", "config.toml"),
			existingKey:  "renderer.performance",
			existingWant: "High",
			replaceWrite: `"Low"`,
			replaceWant:  "Low",
			addKey:       "fonts.regular.weight",
			addWrite:     "400",
			addWant:      "400",
			deleteKey:    "padding-x",
			survivorKey:  "renderer.backend",
			survivorWant: "Automatic",
			fileBacked:   true,
		},
		{
			name: "ssh",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appssh.New(filePersister(filepath.Join(root, "config")))
			},
			sample:       sampleSSH,
			existingKey:  "ServerAliveInterval",
			existingWant: "60",
			replaceWrite: "120",
			replaceWant:  "120",
			addKey:       "ForwardAgent",
			addWrite:     "yes",
			addWant:      "yes",
			deleteKey:    "Compression",
			survivorKey:  "AddKeysToAgent",
			survivorWant: "yes",
			fileBacked:   true,
		},
		{
			name: "sshd",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appsshd.New(filePersister(filepath.Join(root, "sshd_config")))
			},
			sample:       sampleSSHD,
			existingKey:  "PasswordAuthentication",
			existingWant: "no",
			replaceWrite: "yes",
			replaceWant:  "yes",
			addKey:       "ClientAliveInterval",
			addWrite:     "60",
			addWant:      "60",
			deleteKey:    "PermitRootLogin",
			survivorKey:  "Port",
			survivorWant: "22",
			fileBacked:   true,
		},
		{
			name: "starship",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appstarship.New(filePersister(filepath.Join(root, "starship.toml")))
			},
			sample:       fixture(t, "starship", "testdata", "config.txt"),
			existingKey:  "scan_timeout",
			existingWant: "30",
			replaceWrite: "60",
			replaceWant:  "60",
			addKey:       "add_newline",
			addWrite:     "false",
			addWant:      "false",
			deleteKey:    "package.disabled",
			survivorKey:  "format",
			survivorWant: "$all",
			fileBacked:   true,
		},
		{
			name: "tmux",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return apptmux.New(filePersister(filepath.Join(root, ".tmux.conf")))
			},
			sample:       sampleTmux,
			existingKey:  "escape-time",
			existingWant: "0",
			replaceWrite: "10",
			replaceWant:  "10",
			addKey:       "base-index",
			addWrite:     "1",
			addWant:      "1",
			deleteKey:    "mouse",
			survivorKey:  "history-limit",
			survivorWant: "10000",
			fileBacked:   true,
		},
		{
			name: "waybar",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appwaybar.New(filePersister(filepath.Join(root, "config")))
			},
			sample:       sampleWaybar,
			existingKey:  "height",
			existingWant: "30",
			replaceWrite: "32",
			replaceWant:  "32",
			addKey:       "tray.icon-size",
			addWrite:     "16",
			addWant:      "16",
			deleteKey:    "battery.format",
			survivorKey:  "clock.format",
			survivorWant: "{:%H:%M}",
			fileBacked:   true,
		},
		{
			name: "yazi",
			newApp: func(_ *testing.T, root string) konfables.Konfable {
				return appyazi.New(filePersister(filepath.Join(root, "yazi.toml")))
			},
			sample:       sampleYazi,
			existingKey:  "manager.show_hidden",
			existingWant: "false",
			replaceWrite: "true",
			replaceWant:  "true",
			addKey:       "manager.sort_reverse",
			addWrite:     "true",
			addWant:      "true",
			deleteKey:    "preview.wrap",
			survivorKey:  "manager.sort_by",
			survivorWant: "alphabetical",
			fileBacked:   true,
		},
	}
}

var sampleClaude = []byte(`{
  "model": "sonnet",
  "includeCoAuthoredBy": true,
  "permissions": {
    "defaultMode": "default",
    "allow": [
      "Bash(ls:*)"
    ]
  }
}
`)

var sampleFuzzel = []byte(`# fuzzel
font=monospace
dpi-aware=auto
terminal=foot -e
prompt="> "
icon-theme=Papirus
width=30
tabs=8

[colors]
background=fdf6e3ff
text=657b83ff
match=cb4b16ff
selection=eee8d5ff
selection-text=586e75ff
border=002b36ff
`)

var sampleDconf = []byte(`/org/gnome/desktop/wm/preferences/button-layout = appmenu:minimize,maximize,close
/org/gnome/desktop/wm/preferences/focus-mode = click
/org/gnome/desktop/wm/preferences/num-workspaces = 4
/org/gnome/desktop/peripherals/touchpad/tap-to-click = true
/org/gnome/desktop/peripherals/touchpad/speed = 0.5
/org/gnome/desktop/peripherals/mouse/accel-profile = adaptive
`)

var sampleGit = []byte(`[user]
	name = John Doe
	email = john@example.com
[core]
	editor = vim
	autocrlf = input
[init]
	defaultBranch = main
`)

var sampleGTK = []byte(`[Settings]
gtk-theme-name=Adwaita-dark
gtk-icon-theme-name=Papirus-Dark
gtk-cursor-theme-name=Bibata-Modern-Classic
gtk-cursor-theme-size=24
gtk-font-name=JetBrainsMono Nerd Font 11
gtk-application-prefer-dark-theme=true
`)

var sampleGnome = []byte(`org.gnome.desktop.interface/color-scheme = prefer-dark
org.gnome.desktop.interface/gtk-theme = Adwaita
org.gnome.desktop.interface/cursor-size = 24
org.gnome.desktop.interface/enable-animations = true
org.gnome.desktop.background/primary-color = #023c88
`)

var samplePacman = []byte(`#
# /etc/pacman.conf
#

[options]
HoldPkg = pacman glibc
Architecture = auto
Color
CheckSpace
ParallelDownloads = 5
SigLevel = Required DatabaseOptional
LocalFileSigLevel = Optional

#VerbosePkgLists
ILoveCandy

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist
`)

var sampleSSH = []byte(`# global settings
ServerAliveInterval 60
ServerAliveCountMax 3

Host *
    AddKeysToAgent yes
    Compression no
    IdentityFile ~/.ssh/id_ed25519

Host myserver
    HostName example.com
    User admin
    Port 2222
`)

var sampleSSHD = []byte(`# server defaults
Port 22
PermitRootLogin no
PasswordAuthentication no

Match User deploy
    PasswordAuthentication yes
    ForceCommand internal-sftp
`)

var sampleTmux = []byte(`# tmux config
set -g default-terminal "tmux-256color"
set -g escape-time 0
set-option -g mouse on
set -g history-limit 10000
set -g prefix C-a
`)

var sampleWaybar = []byte(`{
  "position": "top",
  "layer": "top",
  "height": 30,
  "modules-left": ["hyprland/workspaces", "hyprland/window"],
  "modules-center": ["clock"],
  "modules-right": ["network", "pulseaudio", "battery", "tray"],
  "clock": {
    "format": "{:%H:%M}"
  },
  "battery": {
    "format": "{capacity}% {icon}"
  },
  "network": {
    "format-wifi": "{essid} ({signalStrength}%)"
  },
  "pulseaudio": {
    "format": "{volume}% {icon}"
  },
  "tray": {
    "spacing": 10
  }
}
`)

var sampleYazi = []byte(`# yazi config

[mgr]
show_hidden = false
sort_by = "alphabetical"
sort_sensitive = false

[preview]
wrap = "no"
max_width = 600

[opener]
edit = [
  { run = "${EDITOR:-vi} %s", desc = "$EDITOR", for = "unix", block = true },
]

[open]
rules = [
  { mime = "text/*", use = "edit" },
  { url = "*", use = "open" },
]
`)
