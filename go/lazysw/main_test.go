package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if strings.TrimSpace(name) == "" {
		t.Fatalf("expected non-empty name")
	}
	parts := strings.Split(name, " ")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d: %q", len(parts), name)
	}
	if parts[0] == "" || parts[1] == "" {
		t.Fatalf("expected non-empty parts: %q", name)
	}
}

func TestSaveLoadPNGList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pngs.json")

	input := []PNG{
		{Name: "Arcano Drago Dor"},
		{Name: "Mistico Vento Mir"},
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
			{Name: "Arcano Drago Dor"},
			{Name: "Mistico Vento Mir"},
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

func TestEncounterConditionsBadgeAndLong(t *testing.T) {
	e := EncounterEntry{
		Conditions: map[string]int{
			"S": 1,
			"V": 2,
		},
	}
	if got := encounterConditionsBadge(e); got != "S1V2" {
		t.Fatalf("unexpected badge: %q", got)
	}
	long := encounterConditionsLong(e)
	if !strings.Contains(long, "S1 Scosso") || !strings.Contains(long, "V2 Vulnerabile") {
		t.Fatalf("unexpected long conditions: %q", long)
	}
}

func TestOrderedEncounterConditionsIgnoresNonPositive(t *testing.T) {
	got := orderedEncounterConditions(map[string]int{
		"S": 2,
		"T": 0,
		"Z": -1,
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(got))
	}
	if got[0].Code != "S" || got[0].Rounds != 2 {
		t.Fatalf("unexpected condition state: %+v", got[0])
	}
}

func TestEncounterConditionEffectsLong(t *testing.T) {
	e := EncounterEntry{
		Conditions: map[string]int{
			"T": 1,
		},
	}
	text := encounterConditionEffectsLong(e)
	if !strings.Contains(text, "Stordito") || !strings.Contains(text, "Vulnerabile") || !strings.Contains(text, "Prono") {
		t.Fatalf("unexpected effects text: %q", text)
	}
}

func TestApplyShakenOnWoundReduction(t *testing.T) {
	e := EncounterEntry{Wounds: 2}
	if !applyShakenOnWoundReduction(1, &e) {
		t.Fatalf("expected shaken to be applied")
	}
	if e.Conditions["S"] != 1 {
		t.Fatalf("expected S1, got %+v", e.Conditions)
	}
	if applyShakenOnWoundReduction(1, &e) {
		t.Fatalf("expected no-op when already shaken")
	}
}

func TestSaveLoadDiceHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dice_history.yml")
	input := []DiceResult{
		{Expression: "1d6", Output: "1d6(4) = 4"},
		{Expression: "2d8+1", Output: "2d8(3+7) + 1 = 11"},
	}
	if err := saveDiceHistory(path, input); err != nil {
		t.Fatalf("save dice history failed: %v", err)
	}
	got, err := loadDiceHistory(path)
	if err != nil {
		t.Fatalf("load dice history failed: %v", err)
	}
	if len(got) != len(input) {
		t.Fatalf("unexpected history size: got %d want %d", len(got), len(input))
	}
	for i := range input {
		if got[i] != input[i] {
			t.Fatalf("mismatch at %d: got %+v want %+v", i, got[i], input[i])
		}
	}
}

func TestRollDiceExpressionUpperDUsesDestiny(t *testing.T) {
	_, breakdown, err := rollDiceExpression("D2+1")
	if err != nil {
		t.Fatalf("unexpected D2+1 error: %v", err)
	}
	if !strings.Contains(breakdown, "1d2e(") || !strings.Contains(breakdown, "destino: 1d6e(") {
		t.Fatalf("unexpected breakdown for D2+1: %q", breakdown)
	}
}

func TestParseDiceJumpIndex(t *testing.T) {
	tests := []struct {
		query   string
		total   int
		wantIdx int
		wantOK  bool
	}{
		{query: "1", total: 10, wantIdx: 0, wantOK: true},
		{query: "10", total: 10, wantIdx: 9, wantOK: true},
		{query: "#3", total: 10, wantIdx: 2, wantOK: true},
		{query: "  #4  ", total: 10, wantIdx: 3, wantOK: true},
		{query: "0", total: 10, wantIdx: 0, wantOK: false},
		{query: "11", total: 10, wantIdx: 0, wantOK: false},
		{query: "abc", total: 10, wantIdx: 0, wantOK: false},
		{query: "", total: 10, wantIdx: 0, wantOK: false},
		{query: "5", total: 0, wantIdx: 0, wantOK: false},
	}

	for _, tt := range tests {
		gotIdx, gotOK := parseDiceJumpIndex(tt.query, tt.total)
		if gotOK != tt.wantOK {
			t.Fatalf("query=%q total=%d: got ok=%v want %v", tt.query, tt.total, gotOK, tt.wantOK)
		}
		if gotOK && gotIdx != tt.wantIdx {
			t.Fatalf("query=%q total=%d: got idx=%d want %d", tt.query, tt.total, gotIdx, tt.wantIdx)
		}
	}
}

func TestDiceGotoIndexFromRune(t *testing.T) {
	tests := []struct {
		r       rune
		total   int
		wantIdx int
		wantOK  bool
	}{
		{r: '^', total: 7, wantIdx: 0, wantOK: true},
		{r: '$', total: 7, wantIdx: 6, wantOK: true},
		{r: '2', total: 7, wantIdx: 1, wantOK: true},
		{r: '0', total: 7, wantIdx: 0, wantOK: false},
		{r: 'x', total: 7, wantIdx: 0, wantOK: false},
		{r: '$', total: 0, wantIdx: 0, wantOK: false},
	}

	for _, tt := range tests {
		gotIdx, gotOK := diceGotoIndexFromRune(tt.r, tt.total)
		if gotOK != tt.wantOK {
			t.Fatalf("r=%q total=%d: got ok=%v want %v", string(tt.r), tt.total, gotOK, tt.wantOK)
		}
		if gotOK && gotIdx != tt.wantIdx {
			t.Fatalf("r=%q total=%d: got idx=%d want %d", string(tt.r), tt.total, gotIdx, tt.wantIdx)
		}
	}
}
