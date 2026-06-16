# konfi

TUI for editing dotfiles.

![konfi usage demo](assets/konfi-usage.webp)

## Install

### Install Script

```sh
curl -fsSL https://raw.githubusercontent.com/getkonfi/konfi/main/install.sh | sh
```

### Homebrew

```sh
brew tap getkonfi/tap
brew install konfi
```

### Nix

```sh
nix shell github:getkonfi/nur-packages#konfi
```

Release package files, including `.deb`, `.rpm`, and archive builds, are
available on the [latest release](https://github.com/getkonfi/konfi/releases/latest).

### From Source

```sh
git clone https://github.com/getkonfi/konfi.git
cd konfi
make build
sudo install -m 0755 konfi /usr/local/bin/konfi
```

Run without installing:

```sh
make run
```

## Supported Konfables

Konfi currently registers 21 konfables covering 911 editable configuration
entries. Version support comes from schema bounds; `not pinned` means the schema
does not declare a minimum or maximum app version.

| Konfable | Repository | Version support | Configurations |
| --- | --- | --- | ---: |
| `alacritty` | [alacritty/alacritty](https://github.com/alacritty/alacritty) | `0.13.0-0.17.0` | 74 |
| `brew` | [Homebrew/homebrew-bundle](https://github.com/Homebrew/homebrew-bundle) | `not pinned` | 5 |
| `dconf` | [GNOME/dconf](https://gitlab.gnome.org/GNOME/dconf) | `not pinned` | 16 |
| `fuzzel` | [dnkl/fuzzel](https://codeberg.org/dnkl/fuzzel) | `not pinned` | 13 |
| `ghostty` | [ghostty-org/ghostty](https://github.com/ghostty-org/ghostty) | `1.0.0-1.3.1` | 200 |
| `git` | [git/git](https://github.com/git/git) | `not pinned` | 37 |
| `gnome` | [GNOME/gsettings-desktop-schemas](https://gitlab.gnome.org/GNOME/gsettings-desktop-schemas) | `not pinned` | 36 |
| `gtk` | [GNOME/gtk](https://gitlab.gnome.org/GNOME/gtk) | `not pinned` | 9 |
| `helix` | [helix-editor/helix](https://github.com/helix-editor/helix) | `24.7.0-25.1.0` | 26 |
| `hyprland` | [hyprwm/Hyprland](https://github.com/hyprwm/Hyprland) | `0.40.0-0.55.2` | 103 |
| `kitty` | [kovidgoyal/kitty](https://github.com/kovidgoyal/kitty) | `0.35.0-0.47.1` | 26 |
| `konfi` | [getkonfi/konfi](https://github.com/getkonfi/konfi) | `not pinned` | 5 |
| `pacman` | [pacman/pacman](https://gitlab.archlinux.org/pacman/pacman) | `not pinned` | 23 |
| `powerlevel10k` | [romkatv/powerlevel10k](https://github.com/romkatv/powerlevel10k) | `<= 1.20.0` | 78 |
| `rio` | [raphamorim/rio](https://github.com/raphamorim/rio) | `0.1.0-0.4.5` | 25 |
| `ssh` | [openssh/openssh-portable](https://github.com/openssh/openssh-portable) | `not pinned` | 36 |
| `sshd` | [openssh/openssh-portable](https://github.com/openssh/openssh-portable) | `not pinned` | 44 |
| `starship` | [starship/starship](https://github.com/starship/starship) | `<= 1.25.1` | 99 |
| `tmux` | [tmux/tmux](https://github.com/tmux/tmux) | `not pinned` | 38 |
| `waybar` | [Alexays/Waybar](https://github.com/Alexays/Waybar) | `not pinned` | 11 |
| `yazi` | [sxyazi/yazi](https://github.com/sxyazi/yazi) | `not pinned` | 7 |

## Development

```sh
make tools
make test
make lint
```

Local release build:

```sh
make release-snapshot
```
