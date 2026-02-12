package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
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
	dicePath := filepath.Join(t.TempDir(), "dice.yaml")
	ui := newUI(monsters, nil, nil, nil, nil, nil, path, dicePath)
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
	if !ui.turnMode || ui.turnRound != 1 || ui.turnIndex != 0 {
		t.Fatalf("unexpected turn mode state: mode=%v round=%d idx=%d", ui.turnMode, ui.turnRound, ui.turnIndex)
	}

	ui.nextEncounterTurn()
	if ui.turnIndex != 1 || ui.turnRound != 1 {
		t.Fatalf("unexpected next turn: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}
	ui.nextEncounterTurn()
	if ui.turnIndex != 2 || ui.turnRound != 1 {
		t.Fatalf("unexpected second next turn: idx=%d round=%d", ui.turnIndex, ui.turnRound)
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

func TestDeleteUndoRedoDice(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "A", 10, 5, "1d1")})
	ui.diceLog = []DiceResult{
		{Expression: "1d1", Output: "1d1(1) = 1"},
		{Expression: "2d1", Output: "2d1(1+1) = 2"},
	}
	ui.renderDiceList()
	ui.dice.SetCurrentItem(0)

	ui.deleteSelectedDiceResult()
	if len(ui.diceLog) != 1 || ui.diceLog[0].Expression != "2d1" {
		t.Fatalf("delete dice failed: %#v", ui.diceLog)
	}

	ui.undoDiceCommand()
	if len(ui.diceLog) != 2 || ui.diceLog[0].Expression != "1d1" {
		t.Fatalf("undo dice failed: %#v", ui.diceLog)
	}

	ui.redoDiceCommand()
	if len(ui.diceLog) != 1 || ui.diceLog[0].Expression != "2d1" {
		t.Fatalf("redo dice failed: %#v", ui.diceLog)
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
	ui := newUI(monsters, nil, nil, nil, nil, nil, path, filepath.Join(tmp, "dice.yaml"))
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 13, CurrentHP: 8, HPFormula: "3d8", UseRolledHP: true, RolledHP: 10, HasInitRoll: true, InitRoll: 15},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 20, CurrentHP: 20, HPFormula: "4d8", UseRolledHP: false, RolledHP: 0, HasInitRoll: false, InitRoll: 0},
		{MonsterIndex: -1, Ordinal: 1, Custom: true, CustomName: "Solum", CustomInit: 2, CustomAC: "13", BaseHP: 10, CurrentHP: 6, HPFormula: "", UseRolledHP: false, RolledHP: 0, HasInitRoll: true, InitRoll: 12},
	}
	ui.encounterSerial = map[int]int{0: 1, 1: 1}
	ui.turnMode = true
	ui.turnIndex = 2
	ui.turnRound = 3

	if err := ui.saveEncountersAs(path); err != nil {
		t.Fatalf("saveEncountersAs failed: %v", err)
	}

	ui2 := newUI(monsters, nil, nil, nil, nil, nil, path, filepath.Join(tmp, "dice.yaml"))
	if err := ui2.loadEncounters(); err != nil {
		t.Fatalf("loadEncounters failed: %v", err)
	}
	if len(ui2.encounterItems) != 3 {
		t.Fatalf("expected 3 encounter items, got %d", len(ui2.encounterItems))
	}
	if !ui2.encounterItems[0].UseRolledHP || ui2.encounterItems[0].RolledHP != 10 {
		t.Fatalf("rolled hp not restored: %#v", ui2.encounterItems[0])
	}
	if !ui2.encounterItems[0].HasInitRoll || ui2.encounterItems[0].InitRoll != 15 {
		t.Fatalf("init roll not restored: %#v", ui2.encounterItems[0])
	}
	if !ui2.encounterItems[2].Custom || ui2.encounterItems[2].CustomName != "Solum" || ui2.encounterItems[2].CustomInit != 2 || ui2.encounterItems[2].CustomAC != "13" {
		t.Fatalf("custom entry not restored: %#v", ui2.encounterItems[2])
	}
	if !ui2.turnMode || ui2.turnIndex != 2 || ui2.turnRound != 3 {
		t.Fatalf("turn progress not restored: mode=%v idx=%d round=%d", ui2.turnMode, ui2.turnIndex, ui2.turnRound)
	}

	if got := readLastEncountersPath(); got != path {
		t.Fatalf("expected last path %q, got %q", path, got)
	}
}

func TestParseCustomInputs(t *testing.T) {
	if hasRoll, roll, base, ok := parseInitInput("17/2"); !ok || !hasRoll || roll != 17 || base != 2 {
		t.Fatalf("unexpected parsed init with roll: ok=%v hasRoll=%v roll=%d base=%d", ok, hasRoll, roll, base)
	}
	if hasRoll, roll, base, ok := parseInitInput(" 3 "); !ok || hasRoll || roll != 0 || base != 3 {
		t.Fatalf("unexpected parsed init base only: ok=%v hasRoll=%v roll=%d base=%d", ok, hasRoll, roll, base)
	}
	if _, _, _, ok := parseInitInput("bad"); ok {
		t.Fatal("expected invalid init input to fail")
	}

	if cur, max, ok := parseHPInput("13"); !ok || cur != 13 || max != 13 {
		t.Fatalf("unexpected parsed hp single value: ok=%v cur=%d max=%d", ok, cur, max)
	}
	if cur, max, ok := parseHPInput("17/20"); !ok || cur != 17 || max != 20 {
		t.Fatalf("unexpected parsed hp pair: ok=%v cur=%d max=%d", ok, cur, max)
	}
	if cur, max, ok := parseHPInput("21/20"); !ok || cur != 20 || max != 20 {
		t.Fatalf("unexpected parsed hp clamped: ok=%v cur=%d max=%d", ok, cur, max)
	}
	if _, _, ok := parseHPInput("8/-1"); ok {
		t.Fatal("expected invalid hp input to fail")
	}
}

func TestEncounterEntryDisplayCustomDoesNotShowOrdinal(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})
	custom := EncounterEntry{Custom: true, CustomName: "Solum", Ordinal: 5}
	if got := ui.encounterEntryDisplay(custom); got != "Solum" {
		t.Fatalf("expected custom display without ordinal, got %q", got)
	}

	regular := EncounterEntry{MonsterIndex: 0, Ordinal: 2}
	if got := ui.encounterEntryDisplay(regular); got != "Aarakocra #2" {
		t.Fatalf("expected regular display with ordinal, got %q", got)
	}
}

func TestRollDiceExpression(t *testing.T) {
	total, breakdown, err := rollDiceExpression("2d1+d1+5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 8 {
		t.Fatalf("expected total 8, got %d", total)
	}
	if !strings.Contains(breakdown, "2d1(") || !strings.Contains(breakdown, "1d1(") || !strings.Contains(breakdown, " = 8") {
		t.Fatalf("unexpected breakdown: %q", breakdown)
	}

	total, _, err = rollDiceExpression("d1+1")
	if err != nil {
		t.Fatalf("unexpected shorthand error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for d1+1, got %d", total)
	}
	total, _, err = rollDiceExpression("1d1-1")
	if err != nil {
		t.Fatalf("unexpected subtraction error: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0 for 1d1-1, got %d", total)
	}
	total, _, err = rollDiceExpression("2d1-5")
	if err != nil {
		t.Fatalf("unexpected subtraction clamp error: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected clamped total 0 for 2d1-5, got %d", total)
	}
	total, breakdown, err = rollDiceExpression("1d1+2 > 2")
	if err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if total != 3 || !strings.HasSuffix(breakdown, " ok") {
		t.Fatalf("expected checked ok result, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("1d1 > 2")
	if err != nil {
		t.Fatalf("unexpected check ko error: %v", err)
	}
	if total != 1 || !strings.HasSuffix(breakdown, " ko") {
		t.Fatalf("expected checked ko result, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("1d1>0 d1+3")
	if err != nil {
		t.Fatalf("unexpected conditional success expr error: %v", err)
	}
	if total != 1 || !strings.Contains(breakdown, "->") || !strings.Contains(breakdown, "= 4") {
		t.Fatalf("expected success expr result, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("1d1>2 d1+3")
	if err != nil {
		t.Fatalf("unexpected conditional ko error: %v", err)
	}
	if total != 1 || !strings.HasSuffix(breakdown, " ko") {
		t.Fatalf("expected conditional ko, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("1d1>=1")
	if err != nil {
		t.Fatalf("unexpected >= ok error: %v", err)
	}
	if total != 1 || !strings.HasSuffix(breakdown, " ok") {
		t.Fatalf("expected >= ok, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("1d1>=2 d1+3")
	if err != nil {
		t.Fatalf("unexpected >= conditional ko error: %v", err)
	}
	if total != 1 || !strings.HasSuffix(breakdown, " ko") {
		t.Fatalf("expected >= conditional ko, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("d1c>0 d1+3")
	if err != nil {
		t.Fatalf("unexpected crit c > success error: %v", err)
	}
	if total != 1 || !strings.Contains(breakdown, "-> 2d1(") || !strings.Contains(breakdown, "= 5") {
		t.Fatalf("expected doubled success dice on crit, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("d1c>=1 d1+3")
	if err != nil {
		t.Fatalf("unexpected crit c >= success error: %v", err)
	}
	if total != 1 || !strings.Contains(breakdown, "-> 2d1(") || !strings.Contains(breakdown, "= 5") {
		t.Fatalf("expected doubled success dice on crit with >=, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("d1v+5")
	if err != nil {
		t.Fatalf("unexpected v mode error: %v", err)
	}
	if total != 6 || !strings.Contains(breakdown, "1d1v(") {
		t.Fatalf("expected v mode parsed, got total=%d breakdown=%q", total, breakdown)
	}
	total, breakdown, err = rollDiceExpression("d1s+1")
	if err != nil {
		t.Fatalf("unexpected s mode error: %v", err)
	}
	if total != 2 || !strings.Contains(breakdown, "1d1s(") {
		t.Fatalf("expected s mode parsed, got total=%d breakdown=%q", total, breakdown)
	}

	if _, _, err := rollDiceExpression("2d+1"); err == nil {
		t.Fatal("expected invalid expression error")
	}
	if _, _, err := rollDiceExpression("2d0"); err == nil {
		t.Fatal("expected invalid dice faces error")
	}
	if _, _, err := rollDiceExpression("2d6++1"); err == nil {
		t.Fatal("expected invalid empty token error")
	}
	if _, _, err := rollDiceExpression("1d6 > x"); err == nil {
		t.Fatal("expected invalid threshold error")
	}
}

func TestChooseDiceMode(t *testing.T) {
	if got := chooseDiceMode('v', 7, 13); got != 13 {
		t.Fatalf("v mode should choose max, got %d", got)
	}
	if got := chooseDiceMode('s', 7, 13); got != 7 {
		t.Fatalf("s mode should choose min, got %d", got)
	}
}

func TestParseDiceRollBatch(t *testing.T) {
	expr, times, err := parseDiceRollBatch("1d6 x2")
	if err != nil || expr != "1d6" || times != 2 {
		t.Fatalf("unexpected batch parse: expr=%q times=%d err=%v", expr, times, err)
	}
	expr, times, err = parseDiceRollBatch("1d6x2")
	if err != nil || expr != "1d6" || times != 2 {
		t.Fatalf("unexpected no-space batch parse: expr=%q times=%d err=%v", expr, times, err)
	}
	expr, times, err = parseDiceRollBatch("1d20+5 > 10 x3")
	if err != nil || expr != "1d20+5 > 10" || times != 3 {
		t.Fatalf("unexpected batch+check parse: expr=%q times=%d err=%v", expr, times, err)
	}
	expr, times, err = parseDiceRollBatch("1d20+5>10x3")
	if err != nil || expr != "1d20+5>10" || times != 3 {
		t.Fatalf("unexpected no-space batch+check parse: expr=%q times=%d err=%v", expr, times, err)
	}
	expr, times, err = parseDiceRollBatch("2d6+1")
	if err != nil || expr != "2d6+1" || times != 1 {
		t.Fatalf("unexpected single parse: expr=%q times=%d err=%v", expr, times, err)
	}
	if _, _, err := parseDiceRollBatch("1d6 x0"); err == nil {
		t.Fatal("expected invalid x0 batch")
	}
}

func TestExpandDiceRollInput(t *testing.T) {
	values, err := expandDiceRollInput("d2,d3,d4")
	if err != nil {
		t.Fatalf("unexpected comma parse error: %v", err)
	}
	if !reflect.DeepEqual(values, []string{"d2", "d3", "d4"}) {
		t.Fatalf("unexpected comma expansion: %#v", values)
	}

	values, err = expandDiceRollInput("1d1x2,d2")
	if err != nil {
		t.Fatalf("unexpected mixed expansion error: %v", err)
	}
	if !reflect.DeepEqual(values, []string{"1d1", "1d1", "d2"}) {
		t.Fatalf("unexpected mixed expansion: %#v", values)
	}
}

func TestHelpForFocusIncludesPanelShortcuts(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})

	encHelp := ui.helpForFocus(ui.encounter)
	for _, expected := range []string{
		"i : tira iniziativa entry selezionata",
		"I : tira iniziativa per tutte le entry",
		"S : ordina entry per tiro iniziativa",
		"* : attiva/disattiva turn mode",
		"n / p : turno successivo / precedente",
	} {
		if !strings.Contains(encHelp, expected) {
			t.Fatalf("encounters help missing %q:\n%s", expected, encHelp)
		}
	}

	monHelp := ui.helpForFocus(ui.list)
	if ui.browseMode != BrowseMonsters {
		ui.browseMode = BrowseMonsters
	}
	monHelp = ui.helpForFocus(ui.list)
	if !strings.Contains(monHelp, "a : aggiungi mostro a Encounters") {
		t.Fatalf("monsters help missing add shortcut:\n%s", monHelp)
	}
	if !strings.Contains(monHelp, "/ : cerca nella Description del mostro selezionato") {
		t.Fatalf("monsters help missing raw search shortcut:\n%s", monHelp)
	}

	rawHelp := ui.helpForFocus(ui.detailRaw)
	if !strings.Contains(rawHelp, "/ : cerca testo nella Description corrente") {
		t.Fatalf("raw help missing raw find shortcut:\n%s", rawHelp)
	}
}

func TestTurnModeFullCycleAndRenderMarkers(t *testing.T) {
	ui := makeTestUI(t, []Monster{
		mkMonster(1, "A", 12, 5, "1d1"), // init base 1
		mkMonster(2, "B", 16, 5, "1d1"), // init base 3 (highest)
		mkMonster(3, "C", 8, 5, "1d1"),  // init base -1
	})
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
		{MonsterIndex: 2, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(2)

	ui.toggleEncounterTurnMode()
	if !ui.turnMode || ui.turnRound != 1 || ui.turnIndex != 1 {
		t.Fatalf("turn mode should start on top initiative: mode=%v round=%d idx=%d", ui.turnMode, ui.turnRound, ui.turnIndex)
	}
	line, _ := ui.encounter.GetItemText(1)
	if !strings.Contains(line, "*[1]") {
		t.Fatalf("active turn marker missing after start: %q", line)
	}

	ui.nextEncounterTurn()
	if ui.turnIndex != 2 || ui.turnRound != 1 {
		t.Fatalf("unexpected state after next: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}
	ui.nextEncounterTurn()
	if ui.turnIndex != 0 || ui.turnRound != 2 {
		t.Fatalf("unexpected wrap after next: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}
	line, _ = ui.encounter.GetItemText(0)
	if !strings.Contains(line, "*[2]") {
		t.Fatalf("expected round increment marker on first row, got: %q", line)
	}

	ui.prevEncounterTurn()
	if ui.turnIndex != 2 || ui.turnRound != 1 {
		t.Fatalf("unexpected state after prev wrap: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}
	ui.prevEncounterTurn()
	if ui.turnIndex != 1 || ui.turnRound != 1 {
		t.Fatalf("unexpected state after prev: idx=%d round=%d", ui.turnIndex, ui.turnRound)
	}

	ui.toggleEncounterTurnMode()
	if ui.turnMode || ui.turnRound != 0 {
		t.Fatalf("turn mode should be disabled after second toggle: mode=%v round=%d", ui.turnMode, ui.turnRound)
	}
	line, _ = ui.encounter.GetItemText(0)
	if strings.Contains(line, "*[") || strings.HasPrefix(strings.TrimSpace(line), "1 ") {
		t.Fatalf("turn prefixes should be removed when turn mode is off, got: %q", line)
	}
}

func TestGlobalInputCaptureTurnsTabIntoEnterWhileAddCustomVisible(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})
	capture := ui.app.GetInputCapture()
	if capture == nil {
		t.Fatal("expected global input capture to be configured")
	}

	ui.addCustomVisible = true
	ev := capture(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	if ev == nil || ev.Key() != tcell.KeyEnter {
		t.Fatalf("expected tab to be translated to enter in add custom mode, got %#v", ev)
	}

	ui.addCustomVisible = false
	ev = capture(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	if ev != nil {
		t.Fatalf("expected tab to be consumed by focus navigation when add custom mode is off, got %#v", ev)
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
