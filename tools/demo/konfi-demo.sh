#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
demo="/tmp/konfi-usage-demo"
home_dir="$demo/home"
config_dir="$demo/config"
state_dir="$demo/state"
bin_dir="$demo/bin"

rm -rf "$demo"
mkdir -p "$home_dir/.ssh" "$config_dir/konfi" "$config_dir/ghostty" "$config_dir/hypr" "$config_dir/fuzzel" "$state_dir" "$bin_dir"

cat >"$config_dir/konfi/config.yaml" <<'YAML'
theme: catppuccin
log_level: info
nerd_font: true
browse_loads_app: false
backup_limit: 5
YAML

cat >"$config_dir/ghostty/config.ghostty" <<'CONF'
font-family = Fira Code
font-size = 12
theme = catppuccin-mocha
window-padding-x = 8
window-padding-y = 8
CONF

cat >"$config_dir/hypr/hyprland.conf" <<'CONF'
# demo hyprland config
general {
    border_size = 2
    gaps_in = 5
    gaps_out = 18
    layout = dwindle
}

decoration {
    rounding = 8
    active_opacity = 1.0
    inactive_opacity = 0.92
}

input {
    kb_layout = us
}
CONF

cat >"$config_dir/fuzzel/fuzzel.ini" <<'CONF'
font=Fira Code:size=12
dpi-aware=auto
terminal=foot -e
prompt=>
width=42
tabs=8

[colors]
background=1e1e2eff
text=cdd6f4ff
selection=313244ff
border=89b4faff
CONF

cat >"$home_dir/.ssh/config" <<'CONF'
Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/github_ed25519

Host *
    ServerAliveInterval 30
    AddKeysToAgent no
CONF

: >"$home_dir/.ssh/id_ed25519-cert.pub"
: >"$home_dir/.ssh/id_ed25519-sk-cert.pub"

cat >"$bin_dir/fake-version" <<'SH'
#!/bin/sh
name="${0##*/}"

case "$name" in
	fc-list)
		printf '%s\n' \
			'Fira Code' \
			'Inter' \
			'JetBrainsMono Nerd Font' \
			'JetBrains Mono' \
			'Monaspace Neon' \
			'Source Code Pro' \
			'Terminus'
		;;
	ghostty) echo "Ghostty 1.3.1" ;;
	starship) echo "starship 1.25.1" ;;
	alacritty) echo "alacritty 0.15.1" ;;
	Hyprland) echo "Hyprland" ;;
	hyprctl) echo "Tag: v0.55.4, commits: demo" ;;
	fuzzel) echo "fuzzel 1.14.1" ;;
	waybar) echo "Waybar v0.14.0" ;;
	yazi) echo "yazi 26.5.6" ;;
	git) echo "git version 2.54.0" ;;
	kitty) echo "kitty 0.42.2 created by Kovid Goyal" ;;
	tmux) echo "tmux 3.5a" ;;
	ssh) echo "OpenSSH_10.3p1, OpenSSL 3.5.0" >&2 ;;
	pacman) echo " .--. Pacman v7.1.0 - libalpm v15.0.0" ;;
	brew) echo "Homebrew 4.6.0" ;;
	*) echo "$name 1.0.0" ;;
esac
SH
chmod +x "$bin_dir/fake-version"

for name in fc-list ghostty starship alacritty Hyprland hyprctl fuzzel waybar yazi git kitty tmux ssh pacman brew; do
	ln -sf fake-version "$bin_dir/$name"
done

exec env -i \
	HOME="$home_dir" \
	XDG_CONFIG_HOME="$config_dir" \
	XDG_STATE_HOME="$state_dir" \
	PATH="$bin_dir" \
	TERM="${TERM:-xterm-256color}" \
	COLORTERM=truecolor \
	"$root/konfi"
