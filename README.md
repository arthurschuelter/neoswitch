# Neoswitch

A fast Neovim profile switcher with a terminal UI, written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Keep multiple Neovim configurations switch between them in a keystroke.

## How it works

- Profiles live in `~/.config/neoswitch/profiles/<name>` (respects `XDG_CONFIG_HOME`).
- `~/.config/nvim` becomes a symlink to the active profile.
- On first run, neoswitch checks for an existing `~/.config/nvim` and offers to import it as your first profile. Nothing is lost.

## Install

```bash
git clone <repo-url> neoswitch
cd neoswitch
go mod tidy
go build -o neoswitch .
sudo mv neoswitch /usr/local/bin/
```

Or run it directly:

```bash
go run .
```

## Usage

Run `neoswitch` to open the TUI.

| Key | Action |
|---|---|
| `↑` / `↓` or `j` / `k` | Move selection |
| `enter` | Activate profile |
| `n` | New profile |
| `q` | Quit |

## Roadmap

Planned: modular packages, CLI subcommands (`list`, `switch`, `create`...), multi-language support, and tests. See [roadmap.md](roadmap.md).
