package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testPaths(t *testing.T) paths {
	t.Helper()
	config := t.TempDir()
	root := filepath.Join(config, "neoswitch")
	return paths{
		nvim:     filepath.Join(config, "nvim"),
		root:     root,
		profiles: filepath.Join(root, "profiles"),
	}
}

func TestPathsCreateAndListProfiles(t *testing.T) {
	p := testPaths(t)
	for _, name := range []string{"work", "personal_2", "default"} {
		if err := p.createProfile(name); err != nil {
			t.Fatalf("createProfile(%q) error = %v", name, err)
		}
	}
	want := []string{"default", "personal_2", "work"}
	if got := p.listProfiles(); !reflect.DeepEqual(got, want) {
		t.Errorf("listProfiles() = %v, want %v", got, want)
	}
}

func TestPathsCreateProfileRejectsInvalidAndDuplicateNames(t *testing.T) {
	p := testPaths(t)
	for _, name := range []string{"", "has space", "../escape"} {
		if err := p.createProfile(name); err == nil {
			t.Errorf("createProfile(%q) error = nil, want invalid-name error", name)
		}
	}
	if err := p.createProfile("work"); err != nil {
		t.Fatalf("createProfile(work) error = %v", err)
	}
	if err := p.createProfile("work"); err == nil {
		t.Error("createProfile(work) for existing profile error = nil, want error")
	}
}

func TestPathsSwitchToProfile(t *testing.T) {
	p := testPaths(t)
	if err := p.createProfile("work"); err != nil {
		t.Fatalf("createProfile(work) error = %v", err)
	}
	if err := p.switchTo("work"); err != nil {
		t.Fatalf("switchTo(work) error = %v", err)
	}
	if got := p.nvimKind(); got != nvimSymlink {
		t.Fatalf("nvimKind() = %v, want symlink", got)
	}
	if got := p.activeProfile(); got != "work" {
		t.Errorf("activeProfile() = %q, want work", got)
	}
	target, err := os.Readlink(p.nvim)
	if err != nil {
		t.Fatalf("Readlink(nvim) error = %v", err)
	}
	if want := filepath.Join(p.profiles, "work"); target != want {
		t.Errorf("symlink target = %q, want %q", target, want)
	}
}

func TestPathsSwitchToRejectsMissingProfileAndRealDirectory(t *testing.T) {
	p := testPaths(t)
	if err := p.switchTo("missing"); err == nil {
		t.Error("switchTo(missing) error = nil, want error")
	}
	if err := os.MkdirAll(p.nvim, 0o755); err != nil {
		t.Fatalf("MkdirAll(nvim) error = %v", err)
	}
	if err := p.createProfile("work"); err != nil {
		t.Fatalf("createProfile(work) error = %v", err)
	}
	if err := p.switchTo("work"); err == nil {
		t.Error("switchTo(work) with real nvim directory error = nil, want error")
	}
}

func TestModelCreateProfileFlow(t *testing.T) {
	p := testPaths(t)
	m := newModel(p)
	if m.screen != screenList {
		t.Fatalf("initial screen = %v, want list", m.screen)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(model)
	if m.screen != screenCreate || cmd == nil {
		t.Fatalf("after n: screen = %v, cmd nil = %t; want create screen and blink command", m.screen, cmd == nil)
	}
	m.input.SetValue("work")
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd != nil {
		t.Error("after creating profile: command != nil")
	}
	if m.screen != screenList || !reflect.DeepEqual(m.profiles, []string{"work"}) {
		t.Errorf("after creation: screen = %v, profiles = %v; want list with [work]", m.screen, m.profiles)
	}
	if m.msg != `Perfil "work" criado.` || m.isErr {
		t.Errorf("creation status = (%q, isErr=%t), want success message", m.msg, m.isErr)
	}
}

func TestModelListViewShowsEmptyState(t *testing.T) {
	m := newModel(testPaths(t))
	view := m.View()
	if !contains(view, "Nenhum perfil ainda.") {
		t.Errorf("View() = %q, want empty-profile message", view)
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && index(s, substr) >= 0)
}

func index(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
