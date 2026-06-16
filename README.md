# konfi

TUI for editing dotfiles.

![konfi usage demo](assets/konfi-usage.webp)

## Install

### Install Script

```bash
curl -fsSL https://raw.githubusercontent.com/getkonfi/konfi/main/install.sh | sh
```

### Homebrew

```bash
brew tap getkonfi/tap
brew install konfi
```

### Nix

```bash
nix shell github:getkonfi/nur-packages#konfi
```

Release package files, including `.deb`, `.rpm`, and archive builds, are
available on the [latest release](https://github.com/getkonfi/konfi/releases/latest).

### From Source

```bash
git clone https://github.com/getkonfi/konfi.git
cd konfi
make build
sudo install -m 0755 konfi /usr/local/bin/konfi
```

Run without installing:

```bash
make run
```

## Supported Konfables

Konfi currently registers 21 konfables covering 911 editable configuration
entries. Version support comes from schema bounds; `not pinned` means the schema
does not declare a minimum or maximum app version.

| Konfable | Version support | Configurations |
| --- | --- | ---: |
| [`alacritty`](https://github.com/alacritty/alacritty) | `0.13.0-0.17.0` | [74](src/konfables/alacritty/schema.yaml) |
| [`brew`](https://github.com/Homebrew/homebrew-bundle) | `not pinned` | [5](src/konfables/brew/schema.yaml) |
| [`dconf`](https://gitlab.gnome.org/GNOME/dconf) | `not pinned` | [16](src/konfables/dconf/schema.yaml) |
| [`fuzzel`](https://codeberg.org/dnkl/fuzzel) | `not pinned` | [13](src/konfables/fuzzel/schema.yaml) |
| [`ghostty`](https://github.com/ghostty-org/ghostty) | `1.0.0-1.3.1` | [200](src/konfables/ghostty/schema.yaml) |
| [`git`](https://github.com/git/git) | `not pinned` | [37](src/konfables/git/schema.yaml) |
| [`gnome`](https://gitlab.gnome.org/GNOME/gsettings-desktop-schemas) | `not pinned` | [36](src/konfables/gnome/schema.yaml) |
| [`gtk`](https://gitlab.gnome.org/GNOME/gtk) | `not pinned` | [9](src/konfables/gtk/schema.yaml) |
| [`helix`](https://github.com/helix-editor/helix) | `24.7.0-25.1.0` | [26](src/konfables/helix/schema.yaml) |
| [`hyprland`](https://github.com/hyprwm/Hyprland) | `0.40.0-0.55.2` | [103](src/konfables/hyprland/schema.yaml) |
| [`kitty`](https://github.com/kovidgoyal/kitty) | `0.35.0-0.47.1` | [26](src/konfables/kitty/schema.yaml) |
| [`konfi`](https://github.com/getkonfi/konfi) | `not pinned` | [5](src/konfables/konfi/schema.yaml) |
| [`pacman`](https://gitlab.archlinux.org/pacman/pacman) | `not pinned` | [23](src/konfables/pacman/schema.yaml) |
| [`powerlevel10k`](https://github.com/romkatv/powerlevel10k) | `<= 1.20.0` | [78](src/konfables/powerlevel10k/schema.yaml) |
| [`rio`](https://github.com/raphamorim/rio) | `0.1.0-0.4.5` | [25](src/konfables/rio/schema.yaml) |
| [`ssh`](https://github.com/openssh/openssh-portable) | `not pinned` | [36](src/konfables/ssh/schema.yaml) |
| [`sshd`](https://github.com/openssh/openssh-portable) | `not pinned` | [44](src/konfables/sshd/schema.yaml) |
| [`starship`](https://github.com/starship/starship) | `<= 1.25.1` | [99](src/konfables/starship/schema.yaml) |
| [`tmux`](https://github.com/tmux/tmux) | `not pinned` | [38](src/konfables/tmux/schema.yaml) |
| [`waybar`](https://github.com/Alexays/Waybar) | `not pinned` | [11](src/konfables/waybar/schema.yaml) |
| [`yazi`](https://github.com/sxyazi/yazi) | `not pinned` | [7](src/konfables/yazi/schema.yaml) |

## Development

```bash
make tools
make test
make lint
```

Local release build:

```bash
make release-snapshot
```
