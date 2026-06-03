# konfigurator

go TUI for editing dotfiles (ghostty, starship, alacritty, hyprland) with ASCII box-drawing aesthetic and cross-app theme sync.

## stack
- go, bubbletea v1, lipgloss v1, bubbles v1, huh v0
- zerolog (file-only — TUI owns stdout), yaml.v3, go-toml/v2, fsnotify

## structure
follows gomono single-app conventions: app code under `src/`, maintenance tools under `tools/`, flat domain packages, setup units for DI, no `internal/`.

```
src/main.go → setup.InitApp(ctx, units) → ui.NewRoot(app) → tea.NewProgram
```

- `setup/` — unit pattern: sequential init with measurement, reverse shutdown
- `ui/` — bubble tea model tree. thin glue layer
- `theme/` — palette definitions, semantic lipgloss styles
- `konfables/` — feature-first domains. each app owns parser, schema, editor
- `pkg/` — shared foundation: config file, schema types, search, file utils
- `pkg/parser/` — format parsers: flat key-value, section/INI, JSON, TOML helpers
- `pkg/pixelart/` — pixel art rendering and logo animations
- `tools/` — maintainer commands; each tool owns a small module that depends on `src/`

## import flow (no cycles)
```
main → setup → ui → theme → pkg
              │ └→ konfables/* → theme
              │         └────→ pkg, pkg/parser
              └──────────────→ pkg
ui → pkg/pixelart, pkg (search, schema, config)
konfables/logos → pkg/pixelart
```

## commands
- `make run` — run the TUI
- `make build` — build binary
- `make test` — run tests
- `make lint` — golangci-lint
- `make schema-verify` — full schema verification (network + introspection)
- `make schema-check` — quick schema check (offline, no exec, strict)

## conventions
- comments in lower case, be cheap on comments
- no `internal/` directory
- tests focus on konfables: parser round-trip fidelity, schema loading
- user approval before any config writes (no silent dotfile modification)
- `setup/detection.go` imports concrete konfable packages — one-directional
- `theme/` never imports `konfables/`
- konfables import `theme` (for Theme in editors), `pkg` (for ConfigFile, Schema), and `pkg/parser` (for format parsers)

## aim
 - main aim is to bring ease of use by having a single tui for various apps
 - QoL and ease of use is important
