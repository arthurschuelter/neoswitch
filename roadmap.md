# Neoswitcher Roadmap

A Neovim profile switcher written in Go with a Bubble Tea TUI.

## Goals

1. **Separation of concerns**: no filesystem logic in the UI, no UI strings in the logic.
2. **Modular packages**: each part can be tested and replaced on its own.
3. **Multi-language support (i18n)**: every user-facing string comes from a locale file.
4. **Tested by design**: every module has tests, and the logic never touches the real `~/.config` during tests.

## Target architecture

```
neoswitcher/
├── cmd/
│   └── neoswitcher/
│       └── main.go          # wiring only: build dependencies, start the TUI
├── internal/
│   ├── paths/               # resolve ~/.config, nvim path, profiles path (XDG aware)
│   ├── profile/             # domain: Profile type, Manager (list/create/switch/import/delete)
│   ├── fsx/                 # thin filesystem abstraction (symlink, rename, stat) for testing
│   ├── config/              # app settings (language, default profile), stored on disk
│   ├── i18n/                # translator, locale loading, fallback logic
│   │   └── locales/         # en.json, pt-BR.json, es.json ... (embedded with go:embed)
│   ├── cli/                 # non-interactive subcommands (list, current, switch, create, import, run)
│   ├── testutil/            # test helpers: temp config home, fake FS, fixtures
│   └── tui/
│       ├── app.go           # root Bubble Tea model, screen routing
│       ├── screens/         # list, create, import, settings, confirm
│       ├── components/      # reusable widgets (header, footer/help, status line)
│       └── styles/          # lipgloss styles and theme
├── go.mod
└── roadmap.md
```

### Dependency rules

- `tui` and `cli` depend on `profile`, `i18n` and `config`. Nothing depends on them, and they never depend on each other.
- `profile` depends on `paths` and `fsx` only. It never imports Bubble Tea or lipgloss.
- `profile` returns typed errors (`ErrExists`, `ErrInvalidName`, `ErrNotFound`, `ErrNvimIsRealDir`). The UI translates them into messages through `i18n`.
- `i18n` has no dependency on any other internal package.
- `main.go` is the only place where concrete implementations are wired together.

## Phases

### Phase 0: Current state (done)
- [x] Single `main.go` with list, create and switch
- [x] First-run detection of `~/.config/nvim` and import as first profile
- [x] Symlink-based switching

### Phase 1: Split into modules
- [ ] Create `internal/paths` and move path resolution out of `main.go`
- [ ] Create `internal/profile` with a `Manager` struct exposing `List`, `Create`, `Switch`, `Import`, `Active`
- [ ] Define typed errors and replace string errors
- [ ] Create `internal/tui` with the root model and one file per screen
- [ ] Reduce `cmd/neoswitcher/main.go` to wiring only
- [ ] Create `internal/testutil` with a helper that builds a fake config home in `t.TempDir()`
- [ ] Unit tests for `paths` (XDG set, XDG unset, missing `$HOME`)
- [ ] Unit tests for `profile.Manager` (see the testing strategy below)
- [ ] First `tui` tests for the list and create screens

### Phase 2: i18n
- [ ] Create `internal/i18n` with a `Translator` (`T(key string, args ...any) string`)
- [ ] Locale files as JSON, embedded with `go:embed` (start with `en` and `pt-BR`)
- [ ] Detect language from `NEOSWITCHER_LANG`, then `LC_ALL` / `LANG`, then fall back to `en`
- [ ] Fallback chain: `pt-BR` → `pt` → `en`; a missing key falls back to English, never to an empty string
- [ ] Replace every hardcoded string in `tui` with translation keys (for example `list.help`, `create.prompt`, `error.exists`)
- [ ] Test that every locale contains the same keys as `en.json`
- [ ] Test that every placeholder (`{{.Name}}`) in a translation matches the one in `en.json`
- [ ] Test the fallback chain (`pt-BR` → `pt` → `en`) and the missing-key behavior
- [ ] Test language detection from `NEOSWITCHER_LANG`, `LC_ALL` and `LANG`
- [ ] Contribution guide: how to add a new language with one file and no code changes

### Phase 3: Settings, persistence and CLI commands
- [ ] Create `internal/config` storing `config.toml` (or JSON) in `~/.config/neoswitcher/`
- [ ] Settings screen with language selector (persisted)
- [ ] Remember the last active profile
- [ ] Create `internal/cli` with non-interactive subcommands that reuse `profile.Manager` (no logic duplicated from the TUI):
  - `neoswitcher` with no arguments opens the TUI
  - `neoswitcher list` prints profiles and marks the active one
  - `neoswitcher current` prints the active profile name
  - `neoswitcher switch <name>` switches and exits
  - `neoswitcher create <name>` creates a profile
  - `neoswitcher import <name>` imports `~/.config/nvim`
  - `--lang <code>` flag to override the language
  - `--help` and per-command usage text, translated through `i18n`
- [ ] Decide between the standard library `flag` package and `cobra` (cobra adds help, completion and nested commands)
- [ ] CLI conventions:
  - Exit code `0` on success, `1` on failure, `2` on usage errors; typed errors map to codes
  - Results on stdout, errors on stderr
  - Plain, pipe-friendly output for `list` and `current` (works with `fzf`, `grep`, scripts)
  - If stdin is not a TTY and no subcommand is given, print usage instead of starting the TUI
- [ ] Shell completion for bash, zsh and fish, including profile names for `switch`

### Phase 4: Profile management features
- [ ] Delete profile, with a confirmation screen
- [ ] Rename profile (keep the symlink valid if it is the active one)
- [ ] Clone profile from an existing one
- [ ] Create profile from a git repository URL
- [ ] Detect and handle a broken symlink
- [ ] Separate data, state and cache per profile using `NVIM_APPNAME` as an alternative to symlinks

### Phase 5: Polish
- [ ] Theming (light/dark, user-defined colors)
- [ ] Search and filter in the profile list
- [ ] Show per-profile info (path, last used, size, plugin manager detected)
- [ ] Launch `nvim` directly with the selected profile
- [ ] `neoswitcher run <name> [-- nvim args]`: set `NVIM_APPNAME=<name>` and start `nvim` without changing the active profile, so two profiles can run in two terminals at once (depends on the symlink vs `NVIM_APPNAME` decision)
- [ ] Windows support (junctions or `NVIM_APPNAME`, since symlinks need privileges)

### Phase 6: Distribution
- [ ] CI with `go vet`, `staticcheck` and `go test -race -cover ./...`
- [ ] Run CI on Linux, macOS and Windows, since symlink behavior differs
- [ ] Coverage report with a minimum threshold for `profile` and `i18n`
- [ ] Make it installable as a command: `go build -o neoswitcher ./cmd/neoswitcher`, and `go install .../cmd/neoswitcher@latest` once the module path is set
- [ ] Document that `$(go env GOPATH)/bin` must be in `PATH` for `go install`
- [ ] Release binaries for Linux, macOS and Windows (GoReleaser)
- [ ] Homebrew tap and AUR package
- [ ] README with screenshots and a section on adding translations

## Testing strategy

| Layer | Kind of test | What it covers |
|---|---|---|
| `paths` | Unit | Config home resolution with and without `XDG_CONFIG_HOME` |
| `profile` | Unit, table-driven, on `t.TempDir()` | Every `Manager` operation and every typed error |
| `fsx` | Unit | Behavior of the real implementation on a temp dir |
| `config` | Unit | Load and save round trip, missing file, corrupt file, defaults |
| `i18n` | Unit | Key parity across locales, placeholders, fallback, language detection |
| `tui` | Model tests | Send `tea.KeyMsg` to `Update` and assert on the model state and `View()` output |
| `cli` | Unit and integration | Each subcommand against a temp `XDG_CONFIG_HOME`: assert stdout, stderr and exit code; usage errors; non-TTY behavior |
| End to end | Integration | Run the whole flow (first run, import, create, switch) against a temp config home |

### `profile.Manager` cases to cover

- **First run:** no `nvim` and no profiles; real `nvim` dir present; `nvim` already a symlink to something outside our profiles.
- **Create:** valid name, invalid characters, empty name, duplicate name.
- **Switch:** to an existing profile, to a missing one, when `nvim` is a real directory (must refuse and leave it untouched), when `nvim` is a broken symlink.
- **Import:** moves the directory, creates the symlink, refuses when the profile name already exists, and leaves the original in place if the rename fails.
- **Active:** symlink inside the profiles dir, symlink outside it, relative symlink, no symlink.
- **Delete and rename (Phase 4):** deleting the active profile, renaming the active profile, name collisions.

### Testing principles

- Tests never read or write the real `~/.config`; they always set `XDG_CONFIG_HOME` to a temp dir.
- Prefer table-driven tests with clear case names.
- Assert on typed errors with `errors.Is` / `errors.As`, never on error text, so translations don't break tests.
- TUI tests drive `Update` directly and check state, so they don't need a real terminal.
- Use golden files for `View()` output only where the layout is stable; otherwise assert on key substrings.
- Fix every bug with a regression test first.

## Design notes

- **Testability:** `profile.Manager` takes an `fsx.FS` interface, so tests can use a fake or a temp dir without touching the real `~/.config`.
- **UI as a thin layer:** a Bubble Tea `Update` calls `Manager` methods and turns the result into a message. It holds no business rules.
- **Translation keys over sentences:** keys describe intent (`profile.created`) and carry placeholders (`"Profile {{.Name}} created."`), so word order can differ per language.
- **Errors carry data, not text:** the logic layer returns `ErrExists{Name}`; only the UI decides how to say it in the user's language.

## Open questions

- Symlinks (simple, one active profile) versus `NVIM_APPNAME` (several profiles usable at once). Support both?
- Which i18n library to use: a minimal in-house JSON translator, or `go-i18n`?
- Config format for the app itself: TOML, JSON or YAML?