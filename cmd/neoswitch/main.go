// Neoswitch - gerenciador de perfis do Neovim com TUI (Bubble Tea).
//
// Como funciona:
//   - Os perfis ficam em  $XDG_CONFIG_HOME/neoswitch/profiles/<nome>
//   - ~/.config/nvim passa a ser um symlink para o perfil ativo
//   - Na primeira execução, verifica se já existe ~/.config/nvim e oferece
//     importá-lo como primeiro perfil (sem perder nada).
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// File system paths and functions
// ---------------------------------------------------------------------------

type paths struct {
	nvim     string // ~/.config/nvim
	root     string // ~/.config/neoswitch
	profiles string // ~/.config/neoswitch/profiles
}

func resolvePaths() (paths, error) {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return paths{}, err
		}
		cfg = filepath.Join(home, ".config")
	}
	root := filepath.Join(cfg, "neoswitch")
	return paths{
		nvim:     filepath.Join(cfg, "nvim"),
		root:     root,
		profiles: filepath.Join(root, "profiles"),
	}, nil
}

type nvimKind int

const (
	nvimMissing nvimKind = iota // Do not exist
	nvimSymlink                 // Already a symlink
	nvimDir                     // Real directory
)

func (p paths) nvimKind() nvimKind {
	fi, err := os.Lstat(p.nvim)
	if err != nil {
		return nvimMissing
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nvimSymlink
	}
	return nvimDir
}

var validName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (p paths) listProfiles() []string {
	entries, err := os.ReadDir(p.profiles)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// activeProfile returns the profile name ~/.config/nvim points to.
func (p paths) activeProfile() string {
	target, err := os.Readlink(p.nvim)
	if err != nil {
		return ""
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(p.nvim), target)
	}
	if filepath.Dir(target) == p.profiles {
		return filepath.Base(target)
	}
	return ""
}

func (p paths) createProfile(name string) error {
	if !validName.MatchString(name) {
		return errors.New("invalid name: only letters, numbers, '-' and '_'")
	}
	dir := filepath.Join(p.profiles, name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("the profile %q already exists", name)
	}
	return os.MkdirAll(dir, 0o755)
}

func (p paths) switchTo(name string) error {
	dir := filepath.Join(p.profiles, name)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("profile %q not found", name)
	}
	switch p.nvimKind() {
	case nvimDir:
		return errors.New("~/.config/nvim is a real directory; import it as a profile before switching")
	case nvimSymlink:
		if err := os.Remove(p.nvim); err != nil {
			return err
		}
	}
	return os.Symlink(dir, p.nvim)
}

// importExisting moves ~/.config/nvim to a profile and creates a symlink.
func (p paths) importExisting(name string) error {
	if !validName.MatchString(name) {
		return errors.New("invalid name: only letters, numbers, '-' and '_'")
	}
	if err := os.MkdirAll(p.profiles, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(p.profiles, name)
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("the profile %q already exists", name)
	}
	if err := os.Rename(p.nvim, dest); err != nil {
		return err
	}
	return os.Symlink(dest, p.nvim)
}

// ---------------------------------------------------------------------------
// Bubble Tea Model
// ---------------------------------------------------------------------------

type screen int

const (
	screenImport screen = iota // First time: import existing config
	screenList                 // lists profiles
	screenCreate               // creates new profile
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7EBAE4"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
)

type model struct {
	p        paths
	screen   screen
	profiles []string
	active   string
	cursor   int
	input    textinput.Model
	msg      string
	isErr    bool
}

func newModel(p paths) model {
	ti := textinput.New()
	ti.CharLimit = 40
	ti.Width = 30

	m := model{p: p, input: ti}

	// First run = profile folder does not exist
	_, err := os.Stat(p.profiles)
	firstRun := os.IsNotExist(err)

	switch {
	case firstRun && p.nvimKind() == nvimDir:
		m.screen = screenImport
		m.input.Placeholder = "default"
		m.input.SetValue("default")
		m.input.Focus()
		m.msg = "Found existing config in ~/.config/nvim."
	default:
		m.screen = screenList
		if firstRun {
			_ = os.MkdirAll(p.profiles, 0o755)
			m.msg = "No ~/.config/nvim found. Create your first profile with 'n'."
		}
		m.refresh()
	}
	return m
}

func (m *model) refresh() {
	m.profiles = m.p.listProfiles()
	m.active = m.p.activeProfile()
	if m.cursor >= len(m.profiles) {
		m.cursor = max(0, len(m.profiles)-1)
	}
}

func (m *model) setErr(err error) { m.msg, m.isErr = err.Error(), true }
func (m *model) setOK(s string)   { m.msg, m.isErr = s, false }

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)
	if isKey && key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.screen {
	case screenImport:
		return m.updateImport(msg, key, isKey)
	case screenCreate:
		return m.updateCreate(msg, key, isKey)
	default:
		return m.updateList(key, isKey)
	}
}

func (m model) updateImport(msg tea.Msg, key tea.KeyMsg, isKey bool) (tea.Model, tea.Cmd) {
	if isKey {
		switch key.String() {
		case "esc":
			// Does not import: switching is blocked.
			_ = os.MkdirAll(m.p.profiles, 0o755)
			m.screen = screenList
			m.setOK("Importação ignorada. Perfis só poderão ser ativados após importar ~/.config/nvim.")
			m.refresh()
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if err := m.p.importExisting(name); err != nil {
				m.setErr(err)
				return m, nil
			}
			m.screen = screenList
			m.setOK(fmt.Sprintf("Configuração importada como perfil %q.", name))
			m.refresh()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateCreate(msg tea.Msg, key tea.KeyMsg, isKey bool) (tea.Model, tea.Cmd) {
	if isKey {
		switch key.String() {
		case "esc":
			m.screen = screenList
			m.msg = ""
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if err := m.p.createProfile(name); err != nil {
				m.setErr(err)
				return m, nil
			}
			m.screen = screenList
			m.setOK(fmt.Sprintf("Perfil %q criado.", name))
			m.refresh()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateList(key tea.KeyMsg, isKey bool) (tea.Model, tea.Cmd) {
	if !isKey {
		return m, nil
	}
	switch key.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.profiles)-1 {
			m.cursor++
		}
	case "n":
		m.screen = screenCreate
		m.input.Placeholder = "nome-do-perfil"
		m.input.SetValue("")
		m.input.Focus()
		m.msg = ""
		return m, textinput.Blink
	case "enter", " ":
		if len(m.profiles) == 0 {
			return m, nil
		}
		name := m.profiles[m.cursor]
		if err := m.p.switchTo(name); err != nil {
			m.setErr(err)
		} else {
			m.setOK(fmt.Sprintf("Active profile: %s", name))
		}
		m.refresh()
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Neoswitch") + "\n\n")

	switch m.screen {
	case screenImport:
		b.WriteString("Dê um nome para importar sua configuração atual como perfil:\n\n")
		b.WriteString(m.input.View() + "\n")
		b.WriteString(m.status())
		b.WriteString(helpStyle.Render("\nenter: importar • esc: pular • ctrl+c: sair"))

	case screenCreate:
		b.WriteString("Nome do novo perfil:\n\n")
		b.WriteString(m.input.View() + "\n")
		b.WriteString(m.status())
		b.WriteString(helpStyle.Render("\nenter: criar • esc: voltar"))

	default:
		if len(m.profiles) == 0 {
			b.WriteString(helpStyle.Render("Nenhum perfil ainda.") + "\n")
		}
		for i, name := range m.profiles {
			cursor, line := "  ", name
			if i == m.cursor {
				cursor = "> "
				line = selectedStyle.Render(name)
			}
			if name == m.active {
				line += activeStyle.Render("  ● active")
			}
			b.WriteString(cursor + line + "\n")
		}
		b.WriteString(m.status())
		b.WriteString(helpStyle.Render("\n↑/↓: move • enter: activate • n: new profile • q: exit"))
	}
	return b.String() + "\n"
}

func (m model) status() string {
	if m.msg == "" {
		return ""
	}
	if m.isErr {
		return "\n" + errStyle.Render(m.msg) + "\n"
	}
	return "\n" + okStyle.Render(m.msg) + "\n"
}

// ---------------------------------------------------------------------------

func main() {
	p, err := resolvePaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(newModel(p)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
