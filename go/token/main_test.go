package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCapitalizeWord(t *testing.T) {
	if got := capitalizeWord(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := capitalizeWord("arcano"); got != "Arcano" {
		t.Fatalf("expected %q, got %q", "Arcano", got)
	}
}

func TestSelectedPNGName(t *testing.T) {
	pngs := []PNG{{Name: "A"}, {Name: "B"}}
	if got := selectedPNGName(pngs, -1); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := selectedPNGName(pngs, 1); got != "B" {
		t.Fatalf("expected %q, got %q", "B", got)
	}
	if got := selectedPNGName(pngs, 2); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestRandomPNGNameFormat(t *testing.T) {
	name := randomPNGName()
	parts := strings.Split(name, " ")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %q", len(parts), name)
	}
	for _, p := range parts {
		if p == "" {
			t.Fatalf("empty part in %q", name)
		}
		if strings.ToUpper(p[:1]) != p[:1] {
			t.Fatalf("part not capitalized: %q in %q", p, name)
		}
	}
}

func TestSaveLoadPNGList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pngs.json")

	input := []PNG{
		{Name: "Arcano Drago Dor", Counter: 2},
		{Name: "Mistico Vento Mir", Counter: 3},
	}
	if err := savePNGList(path, input, "Mistico Vento Mir"); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, selected, err := loadPNGList(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if selected != "Mistico Vento Mir" {
		t.Fatalf("expected selected %q, got %q", "Mistico Vento Mir", selected)
	}
	if len(got) != len(input) {
		t.Fatalf("expected %d items, got %d", len(input), len(got))
	}
	for i := range input {
		if got[i] != input[i] {
			t.Fatalf("mismatch at %d: got %+v, want %+v", i, got[i], input[i])
		}
	}
}

func TestLoadLegacyPNGList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pngs.json")

	legacy := `[
  {"Name":"Jack","Counter":1},
  {"Name":"John","Counter":3}
]`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, selected, err := loadPNGList(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if selected != "" {
		t.Fatalf("expected empty selected, got %q", selected)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Name != "Jack" || got[1].Name != "John" {
		t.Fatalf("unexpected names: %+v", got)
	}
}

func TestSelectionHelpers(t *testing.T) {
	dir := t.TempDir()
	old := dataFile
	dataFile = filepath.Join(dir, "pngs.json")
	t.Cleanup(func() { dataFile = old })

	m := model{
		pngs: []PNG{
			{Name: "Arcano Drago Dor", Counter: 2},
			{Name: "Mistico Vento Mir", Counter: 1},
		},
		selectedPNGIndex: -1,
	}

	m.selectNextPNG()
	if m.selectedPNGIndex != 0 {
		t.Fatalf("expected selected index 0, got %d", m.selectedPNGIndex)
	}

	m.selectNextPNG()
	if m.selectedPNGIndex != 1 {
		t.Fatalf("expected selected index 1, got %d", m.selectedPNGIndex)
	}

	m.selectPrevPNG()
	if m.selectedPNGIndex != 0 {
		t.Fatalf("expected selected index 0 after prev, got %d", m.selectedPNGIndex)
	}

	if _, err := os.Stat(dataFile); err != nil {
		t.Fatalf("expected save file to exist: %v", err)
	}
}

func TestMenuHandleChoiceCreateRandom(t *testing.T) {
	dir := t.TempDir()
	old := dataFile
	dataFile = filepath.Join(dir, "pngs.json")
	t.Cleanup(func() { dataFile = old })

	m := model{pngs: []PNG{}}
	var cmd tea.Cmd
	m, cmd = m.handleMenuChoice("Crea PNG casuale")
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
	}
	if len(m.pngs) != 1 {
		t.Fatalf("expected 1 png, got %d", len(m.pngs))
	}
	if m.selectedPNGIndex != 0 {
		t.Fatalf("expected selected index 0, got %d", m.selectedPNGIndex)
	}
	if _, err := os.Stat(dataFile); err != nil {
		t.Fatalf("expected save file to exist: %v", err)
	}
}

func TestMenuHandleChoiceSave(t *testing.T) {
	dir := t.TempDir()
	old := dataFile
	dataFile = filepath.Join(dir, "pngs.json")
	t.Cleanup(func() { dataFile = old })

	m := model{
		pngs:             []PNG{{Name: "Arcano Drago Dor", Counter: 2}},
		selectedPNGIndex: 0,
	}
	_, _ = m.handleMenuChoice("Salva PNG su disco")
	if _, err := os.Stat(dataFile); err != nil {
		t.Fatalf("expected save file to exist: %v", err)
	}
}

func TestMenuHandleChoiceReloadKeepsSelected(t *testing.T) {
	dir := t.TempDir()
	old := dataFile
	dataFile = filepath.Join(dir, "pngs.json")
	t.Cleanup(func() { dataFile = old })

	initial := []PNG{
		{Name: "Arcano Drago Dor", Counter: 2},
		{Name: "Mistico Vento Mir", Counter: 1},
	}
	if err := savePNGList(dataFile, initial, "Mistico Vento Mir"); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	m := model{
		pngs:             []PNG{{Name: "X", Counter: 0}},
		selectedPNGIndex: 0,
	}
	m, _ = m.handleMenuChoice("Ricarica PNG da disco")

	if len(m.pngs) != 2 {
		t.Fatalf("expected 2 pngs, got %d", len(m.pngs))
	}
	if m.selectedPNGIndex != 1 {
		t.Fatalf("expected selected index 1, got %d", m.selectedPNGIndex)
	}
}

func TestMenuHandleChoiceReloadLegacyKeepsSelectedName(t *testing.T) {
	dir := t.TempDir()
	old := dataFile
	dataFile = filepath.Join(dir, "pngs.json")
	t.Cleanup(func() { dataFile = old })

	legacy := `[
  {"Name":"Jack","Counter":1},
  {"Name":"John","Counter":3}
]`
	if err := os.WriteFile(dataFile, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	m := model{
		pngs: []PNG{
			{Name: "Jack", Counter: 1},
			{Name: "John", Counter: 3},
		},
		selectedPNGIndex: 1,
	}
	m, _ = m.handleMenuChoice("Ricarica PNG da disco")

	if len(m.pngs) != 2 {
		t.Fatalf("expected 2 pngs, got %d", len(m.pngs))
	}
	if m.selectedPNGIndex != 1 {
		t.Fatalf("expected selected index 1, got %d", m.selectedPNGIndex)
	}
}
