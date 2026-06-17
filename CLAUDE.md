# konfi

go TUI for exploring and editing dotfiles (ghostty, starship, alacritty, hyprland, and more) with an ASCII box-drawing aesthetic.

## stack
- go 1.26
- charm v2 stack (charm.land namespace): bubbletea/v2, bubbles/v2, lipgloss/v2
- zerolog (file-only — TUI owns stdout), yaml.v3, fsnotify
- goldmark (markdown), sahilm/fuzzy (search); TOML/INI/flat parsing is hand-rolled in pkg/parser

## structure
follows gomono single-app conventions: app code under `src/`, maintenance tools under `tools/`, flat domain packages, setup units for DI, no `internal/`.

```
src/main.go → setup.InitApp(ctx, units) → ui.NewRoot(app) → tea.NewProgram
```

- `setup/` — unit pattern: sequential init with measurement, reverse shutdown
- `ui/` — bubble tea model tree. thin glue layer
  - `ui/editors/` — field editor implementations (color, enum, font, …); depends only on `pkg`, `theme`
  - `ui/widgets/` — stateless leaf renderers (worddiff, markdown, diffview); depends only on `theme`
- `theme/` — palette definitions, semantic lipgloss styles
- `konfables/` — feature-first domains. each app owns parser, schema, editor
- `pkg/` — shared foundation: config file, schema types, search, file utils
- `pkg/parser/` — format parsers: flat key-value, section/INI, JSON, TOML helpers
- `pkg/pixelart/` — pixel art rendering and logo animations
- `tools/` — maintainer commands; each tool owns a small module that depends on `src/`

## import flow (no cycles)
```
main → setup, ui
ui   → setup, konfables/*, theme, pkg, pkg/pixelart
ui   → ui/editors → pkg, theme
ui   → ui/widgets → theme
setup → konfables/*, theme, pkg
konfables/*    → pkg, pkg/parser
konfables/logos → pkg/pixelart
pkg  → pkg/parser
```
theme imports nothing internal (lipgloss only) — it's a leaf.

## commands
- `make run` — run the TUI
- `make build` — build binary
- `make test` — run tests
- `make lint` — golangci-lint
- `make schema-verify` — full schema verification (network + introspection)
- `make schema-check` — quick schema check (offline, no exec, strict)
- `make release-check` — check schema support against latest app releases
- `make release-field-check` — check whether newer app releases add config fields

## conventions
- comments in lower case, be cheap on comments
- no `internal/` directory
- tests focus on konfables: parser round-trip fidelity, schema loading
- user approval before any config writes (no silent dotfile modification)
- `setup/detection.go` imports concrete konfable packages — one-directional
- `theme/` never imports `konfables/`
- konfables import `pkg` (for ConfigFile, Schema), and `pkg/parser` (for format parsers)

## aim
 - main aim is to bring ease of use by having a single tui for various apps
 - QoL and ease of use is important
