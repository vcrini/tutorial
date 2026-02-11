package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mkMonster(id int, name string, dex int, hpAvg int, hpFormula string) Monster {
	raw := map[string]any{
		"name": name,
		"dex":  dex,
		"hp": map[string]any{
			"average": hpAvg,
			"formula": hpFormula,
		},
	}
	return Monster{
		ID:   id,
		Name: name,
		Raw:  raw,
	}
}

func makeTestUI(t *testing.T, monsters []Monster) *UI {
	t.Helper()
	path := filepath.Join(t.TempDir(), "encounters.yaml")
	ui := newUI(monsters, nil, nil, nil, path)
	return ui
}

func TestLoadMonstersFromBytes(t *testing.T) {
	yml := `monsters:
  - name: Aarakocra
    source: MM
    environment: [mountain, grassland]
    type: humanoid
    cr: '1/4'
    dex: 14
    hp:
      average: 13
      formula: 3d8
  - name: Cult Fanatic
    source: MM
    environment: [urban]
    type:
      type: humanoid
    cr:
      cr: '2'
    dex: 10
    hp:
      average: 33
      formula: 6d8+6
`
	monsters, envs, crs, types, err := loadMonstersFromBytes([]byte(yml))
	if err != nil {
		t.Fatalf("loadMonstersFromBytes error: %v", err)
	}
	if len(monsters) != 2 {
		t.Fatalf("expected 2 monsters, got %d", len(monsters))
	}
	if monsters[0].Name != "Aarakocra" {
		t.Fatalf("expected sorted name Aarakocra first, got %q", monsters[0].Name)
	}
	if !reflect.DeepEqual(envs, []string{"grassland", "mountain", "urban"}) {
		t.Fatalf("unexpected envs: %#v", envs)
	}
	if !reflect.DeepEqual(crs, []string{"1/4", "2"}) {
		t.Fatalf("unexpected crs: %#v", crs)
	}
	if !reflect.DeepEqual(types, []string{"humanoid"}) {
		t.Fatalf("unexpected types: %#v", types)
	}
}

func TestMatchFunctions(t *testing.T) {
	if !matchName("Adult Red Dragon", "red") {
		t.Fatal("expected matchName true")
	}
	if matchName("Goblin", "dragon") {
		t.Fatal("expected matchName false")
	}
	if !matchCR("1/2", "1/2") || !matchCR("1/2", " 1/2 ") {
		t.Fatal("expected matchCR true")
	}
	if matchCR("2", "3") {
		t.Fatal("expected matchCR false")
	}
	if !matchEnv([]string{"forest", "urban"}, "urban") {
		t.Fatal("expected matchEnv true")
	}
	if matchEnv([]string{"forest"}, "desert") {
		t.Fatal("expected matchEnv false")
	}
	if !matchType("humanoid", "humanoid") {
		t.Fatal("expected matchType true")
	}
	if !matchType("", "Unknown") {
		t.Fatal("expected empty type to match Unknown")
	}
}

func TestExtractors(t *testing.T) {
	raw := map[string]any{
		"dex": 14,
		"hp":  map[string]any{"average": 13, "formula": "3d8"},
		"ac":  []any{map[string]any{"ac": 15}},
		"speed": map[string]any{
			"walk": 30,
			"fly":  map[string]any{"number": 50, "condition": "(hover)"},
		},
	}
	if init, ok := extractInitFromDex(raw); !ok || init != 2 {
		t.Fatalf("unexpected init from dex: ok=%v init=%d", ok, init)
	}
	avg, formula := extractHP(raw)
	if avg != "13" || formula != "3d8" {
		t.Fatalf("unexpected hp extract: avg=%q formula=%q", avg, formula)
	}
	if hp, ok := extractHPAverageInt(raw); !ok || hp != 13 {
		t.Fatalf("unexpected hp avg int: ok=%v hp=%d", ok, hp)
	}
	if ac := extractAC(raw); ac != "15" {
		t.Fatalf("unexpected ac: %q", ac)
	}
	speed := extractSpeed(raw)
	if !strings.Contains(speed, "walk 30 ft.") || !strings.Contains(speed, "fly 50 ft. (hover)") {
		t.Fatalf("unexpected speed formatting: %q", speed)
	}
}

func TestRollHPFormula(t *testing.T) {
	if _, ok := rollHPFormula("3d8+2"); !ok {
		t.Fatal("expected valid formula")
	}
	for i := 0; i < 10; i++ {
		v, ok := rollHPFormula("1d1")
		if !ok || v != 1 {
			t.Fatalf("1d1 should always roll 1, got %d ok=%v", v, ok)
		}
	}
	if _, ok := rollHPFormula("bad"); ok {
		t.Fatal("expected invalid formula")
	}
}

func TestToggleEncounterHPModeClampsCurrentHP(t *testing.T) {
	m := mkMonster(1, "Aarakocra", 14, 13, "1d1")
	ui := makeTestUI(t, []Monster{m})
	ui.encounterItems = []EncounterEntry{{
		MonsterIndex: 0,
		Ordinal:      1,
		BaseHP:       13,
		CurrentHP:    13,
		HPFormula:    "1d1",
	}}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(0)

	ui.toggleEncounterHPMode()
	if !ui.encounterItems[0].UseRolledHP {
		t.Fatal("expected rolled mode enabled")
	}
	if ui.encounterItems[0].RolledHP != 1 {
		t.Fatalf("expected rolled hp 1, got %d", ui.encounterItems[0].RolledHP)
	}
	if ui.encounterItems[0].CurrentHP != 1 {
		t.Fatalf("expected current hp clamped to 1, got %d", ui.encounterItems[0].CurrentHP)
	}
}

func TestTurnModeAndAdvance(t *testing.T) {
	ui := makeTestUI(t, []Monster{
		mkMonster(1, "A", 10, 5, "1d1"),
		mkMonster(2, "B", 10, 5, "1d1"),
		mkMonster(3, "C", 10, 5, "1d1"),
	})
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
		{MonsterIndex: 2, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(1)

	ui.toggleEncounterTurnMode()
	if !ui.turnMode || ui.turnRound != 1 || ui.turnIndex != 1 {
		t.Fatalf("unexpected turn mode state: mode=%v round=%d idx=%d", ui.turnMode, ui.turnRound, ui.turnIndex)
	}

	ui.nextEncounterTurn()
	if ui.turnIndex != 2 || ui.turnRound != 1 {
		t.Fatalf("unexpected next turn: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}
	ui.nextEncounterTurn()
	if ui.turnIndex != 0 || ui.turnRound != 2 {
		t.Fatalf("unexpected wrapped next turn: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}
	ui.prevEncounterTurn()
	if ui.turnIndex != 2 || ui.turnRound != 1 {
		t.Fatalf("unexpected prev turn: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}
}

func TestDeleteUndoRedoEncounter(t *testing.T) {
	ui := makeTestUI(t, []Monster{
		mkMonster(1, "A", 10, 5, "1d1"),
		mkMonster(2, "B", 10, 5, "1d1"),
	})
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
	}
	ui.encounterSerial = map[int]int{0: 1, 1: 1}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(0)

	ui.deleteSelectedEncounterEntry()
	if len(ui.encounterItems) != 1 || ui.encounterItems[0].MonsterIndex != 1 {
		t.Fatalf("delete failed, items=%#v", ui.encounterItems)
	}

	ui.undoEncounterCommand()
	if len(ui.encounterItems) != 2 {
		t.Fatalf("undo failed, items=%#v", ui.encounterItems)
	}
	ui.redoEncounterCommand()
	if len(ui.encounterItems) != 1 || ui.encounterItems[0].MonsterIndex != 1 {
		t.Fatalf("redo failed, items=%#v", ui.encounterItems)
	}
}

func TestSortEncounterByInitiative(t *testing.T) {
	ui := makeTestUI(t, []Monster{
		mkMonster(1, "A", 12, 5, "1d1"),
		mkMonster(2, "B", 14, 5, "1d1"),
		mkMonster(3, "C", 8, 5, "1d1"),
	})
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, HasInitRoll: true, InitRoll: 5},
		{MonsterIndex: 1, Ordinal: 1, HasInitRoll: true, InitRoll: 20},
		{MonsterIndex: 2, Ordinal: 1, HasInitRoll: true, InitRoll: 11},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(2)
	selected := ui.encounterItems[2]

	ui.sortEncounterByInitiative()
	if ui.encounterItems[0].InitRoll != 20 || ui.encounterItems[1].InitRoll != 11 || ui.encounterItems[2].InitRoll != 5 {
		t.Fatalf("sort by init failed: %#v", ui.encounterItems)
	}
	idx := ui.encounter.GetCurrentItem()
	if ui.encounterItems[idx].MonsterIndex != selected.MonsterIndex || ui.encounterItems[idx].Ordinal != selected.Ordinal {
		t.Fatal("selected encounter should stay on same entry after sort")
	}
}

func TestSaveLoadEncountersRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(oldWd) }()

	monsters := []Monster{mkMonster(100, "A", 12, 13, "3d8"), mkMonster(200, "B", 14, 20, "4d8")}
	path := filepath.Join(tmp, "my-enc.yaml")
	ui := newUI(monsters, nil, nil, nil, path)
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 13, CurrentHP: 8, HPFormula: "3d8", UseRolledHP: true, RolledHP: 10, HasInitRoll: true, InitRoll: 15},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 20, CurrentHP: 20, HPFormula: "4d8", UseRolledHP: false, RolledHP: 0, HasInitRoll: false, InitRoll: 0},
	}
	ui.encounterSerial = map[int]int{0: 1, 1: 1}

	if err := ui.saveEncountersAs(path); err != nil {
		t.Fatalf("saveEncountersAs failed: %v", err)
	}

	ui2 := newUI(monsters, nil, nil, nil, path)
	if err := ui2.loadEncounters(); err != nil {
		t.Fatalf("loadEncounters failed: %v", err)
	}
	if len(ui2.encounterItems) != 2 {
		t.Fatalf("expected 2 encounter items, got %d", len(ui2.encounterItems))
	}
	if !ui2.encounterItems[0].UseRolledHP || ui2.encounterItems[0].RolledHP != 10 {
		t.Fatalf("rolled hp not restored: %#v", ui2.encounterItems[0])
	}
	if !ui2.encounterItems[0].HasInitRoll || ui2.encounterItems[0].InitRoll != 15 {
		t.Fatalf("init roll not restored: %#v", ui2.encounterItems[0])
	}

	if got := readLastEncountersPath(); got != path {
		t.Fatalf("expected last path %q, got %q", path, got)
	}
}

func TestHighlightEscapedAndFindRawMatch(t *testing.T) {
	line := `"name": "Aarakocra"`
	h := highlightEscaped(line, "Aarakocra")
	if !strings.Contains(h, "[black:gold]") {
		t.Fatalf("expected highlight tags, got %q", h)
	}

	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})
	ui.rawText = "line1\nline2 TARGET\nline3"
	ui.detailRaw.SetText(ui.rawText)
	lineIdx, ok := ui.findRawMatch("target")
	if !ok || lineIdx != 1 {
		t.Fatalf("expected line 1 match, got idx=%d ok=%v", lineIdx, ok)
	}
}
