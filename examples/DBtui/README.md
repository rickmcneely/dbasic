# DBtui

VBDOS-inspired visual TUI IDE for **DBasic**, written entirely in DBasic on top of charm.land's bubbletea v2 / lipgloss v2.

DBtui is the successor to the old `vdbterm` example. Its design is based on [vgtui](https://github.com/zditech/vgtui), the Go-targeting sibling — same VBDOS palette, same panel layout, same project format philosophy — but the whole thing is implemented in DBasic so you can read, modify, and rebuild it with the DBasic toolchain.

## Status

This is the **minimal-boot scaffold**: the bubbletea model is wired up and paints a blank workspace in the VBDOS theme. Subsequent sessions add the project tree, widget palette, visual form designer, code editor, properties panel, package browser, and the codegen subcommand.

## Build & run

```bash
cd examples/DBtui
dbasic build DBtui.dbas -o DBtui
./DBtui
```

The compiled binary is `.gitignored` — `dbasic build` rebuilds it from source.

If `proxy.golang.org` is unreachable, use offline mode:

```bash
dbasic build DBtui.dbas --offline -o DBtui
```

Press `q` or `Ctrl+C` to quit.

## Layout

```
DBtui.dbas              # entrypoint (CLI + tea.NewProgram)
internal/
  app/app.dbas          # bubbletea Model + Init/Update/View
  ui/theme.dbas         # VBDOS palette + lipgloss styles
  ui/layout.dbas        # focus / mode enums
projects/
  default/              # placeholder for the default project (forthcoming)
```

The directory layout deliberately mirrors `~/vgtui/internal/...` so that future ports of vgtui's panels and widgets drop in with minimal renaming.

## Conventions

- Charm v2 throughout: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2/...`
- `View()` returns `tea.View` (use `tea.NewView(...)` and set `.AltScreen` / `.MouseMode`)
- Key events: `tea.KeyPressMsg` (`.String()` returns "q", "ctrl+c", "f10", etc.)
- `tea.Quit` is a `Cmd` — `RETURN m, tea.Quit`
- Hex colours live as `CONST` strings in `theme.dbas`; wrapper `FUNCTION`s build `lipgloss.Color` and `lipgloss.Style` values lazily.
