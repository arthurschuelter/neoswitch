# Neoswitcher

A fast Neovim profile switcher with a terminal UI, written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Keep multiple Neovim configurations side by side and switch between them in a keystroke.

## How it works

- Profiles live in `~/.config/neoswitcher/profiles/<name>` (respects `XDG_CONFIG_HOME`).
- `~/.config/nvim` becomes a symlink to the active profile.
- On first run, Neoswitcher checks for an existing `~/.config/nvim` and offers to import it as your first profile. Nothing is lost.

## Install

```bash
git clone <repo-url> neoswitcher
cd neoswitcher
go mod tidy
go build -o neoswitcher .
sudo mv neoswitcher /usr/local/bin/
```

Or run it directly:

```bash
go run .
```

## Usage

Run `neoswitcher` to open the TUI.

| Key | Action |
|---|---|
| `↑` / `↓` or `j` / `k` | Move selection |
| `enter` | Activate profile |
| `n` | New profile |
| `q` | Quit |

## Roadmap

Planned: modular packages, CLI subcommands (`list`, `switch`, `create`...), multi-language support, and tests. See [roadmap.md](roadmap.md).
