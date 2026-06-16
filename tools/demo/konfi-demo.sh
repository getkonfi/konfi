#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
demo="/tmp/konfi-usage-demo"
home_dir="$demo/home"
config_dir="$demo/config"
state_dir="$demo/state"
bin_dir="$demo/bin"

rm -rf "$demo"
mkdir -p "$home_dir" "$config_dir/konfi" "$config_dir/kitty" "$state_dir" "$bin_dir"

cat >"$config_dir/konfi/config.yaml" <<'YAML'
theme: catppuccin
log_level: info
nerd_font: false
browse_loads_app: false
backup_limit: 5
YAML

cat >"$config_dir/kitty/kitty.conf" <<'CONF'
font_family monospace
font_size 11.0
cursor_shape block
scrollback_lines 2000
tab_bar_style fade
CONF

cat >"$bin_dir/fake-version" <<'SH'
#!/bin/sh
name="${0##*/}"

case "$name" in
	fc-list)
		printf '%s\n' \
			'Fira Code' \
			'Inter' \
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
