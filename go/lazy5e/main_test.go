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
	ui := newUI(monsters, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, path, dicePath)
	return ui
}

func makeCharacterUI(t *testing.T) *UI {
	t.Helper()
	path := filepath.Join(t.TempDir(), "encounters.yaml")
	dicePath := filepath.Join(t.TempDir(), "dice.yaml")
	classes := []Monster{
		{
			Name:        "Wizard",
			Source:      "PHB",
			CR:          "d6",
			Environment: []string{"INT", "WIS"},
			Raw:         map[string]any{"spellcastingAbility": "int"},
		},
		{
			Name:        "Fighter",
			Source:      "PHB",
			CR:          "d10",
			Environment: []string{"STR", "CON"},
			Raw:         map[string]any{},
		},
	}
	races := []Monster{
		{
			Name:   "Elf",
			Source: "PHB",
			Raw: map[string]any{
				"ability": []any{map[string]any{"dex": 2, "int": 1}},
				"speed":   map[string]any{"walk": 30},
			},
		},
	}
	return newUI(nil, nil, nil, classes, races, nil, nil, nil, nil, nil, nil, path, dicePath)
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

func TestLoadRacesFromBytes(t *testing.T) {
	yml := `races:
  - name: Elf
    source: PHB
    size: [M]
    lineage: none
    ability:
      - dex: 2
    entries:
      - Keen Senses.
`
	races, envs, crs, types, err := loadRacesFromBytes([]byte(yml))
	if err != nil {
		t.Fatalf("loadRacesFromBytes error: %v", err)
	}
	if len(races) != 1 || races[0].Name != "Elf" {
		t.Fatalf("unexpected races: %#v", races)
	}
	if !reflect.DeepEqual(envs, []string{"DEX"}) {
		t.Fatalf("unexpected race env options: %#v", envs)
	}
	if !reflect.DeepEqual(crs, []string{"M"}) {
		t.Fatalf("unexpected race size options: %#v", crs)
	}
	if !reflect.DeepEqual(types, []string{"none"}) {
		t.Fatalf("unexpected race lineage options: %#v", types)
	}
}

func TestLoadFeatsFromBytes(t *testing.T) {
	yml := `feats:
  - name: Alert
    source: PHB
    category: G
    prerequisite:
      - level: 4
    ability:
      - dex: 1
    entries:
      - Always on guard.
`
	feats, envs, crs, types, err := loadFeatsFromBytes([]byte(yml))
	if err != nil {
		t.Fatalf("loadFeatsFromBytes error: %v", err)
	}
	if len(feats) != 1 || feats[0].Name != "Alert" {
		t.Fatalf("unexpected feats: %#v", feats)
	}
	if len(envs) != 1 || !strings.Contains(strings.ToLower(envs[0]), "level") {
		t.Fatalf("unexpected feat prereq options: %#v", envs)
	}
	if !reflect.DeepEqual(crs, []string{"G"}) {
		t.Fatalf("unexpected feat category options: %#v", crs)
	}
	if len(types) != 1 || !strings.Contains(strings.ToLower(types[0]), "dex") {
		t.Fatalf("unexpected feat ability options: %#v", types)
	}
}

func TestGenerateCharacterSheetFromScores(t *testing.T) {
	cl := Monster{
		Name:        "Wizard",
		Source:      "PHB",
		CR:          "d6",
		Environment: []string{"INT", "WIS"},
		Raw: map[string]any{
			"spellcastingAbility": "int",
		},
	}
	rc := Monster{
		Name:   "Elf",
		Source: "PHB",
		Raw: map[string]any{
			"ability": []any{map[string]any{"dex": 2, "int": 1}},
			"speed":   map[string]any{"walk": 30},
		},
	}
	base := [6]int{10, 12, 14, 8, 13, 15}
	meta, body := generateCharacterSheetFromScores(cl, rc, 5, base)

	if !strings.Contains(meta, "Wizard Elf Lv5") {
		t.Fatalf("unexpected meta title: %q", meta)
	}
	if !strings.Contains(meta, "AC:[-] 12") {
		t.Fatalf("expected AC 12 in meta: %q", meta)
	}
	if !strings.Contains(meta, "HP:[-] 32") {
		t.Fatalf("expected HP 32 in meta: %q", meta)
	}
	if !strings.Contains(body, "Proficiency Bonus: +3") {
		t.Fatalf("expected proficiency +3 in body: %q", body)
	}
	if !strings.Contains(body, "Spell Save DC: 10") || !strings.Contains(body, "Spell Attack Bonus: +2") {
		t.Fatalf("expected spell stats in body: %q", body)
	}
	if !strings.Contains(body, "INT +2 (proficient)") {
		t.Fatalf("expected INT save with proficiency in body: %q", body)
	}
	if !strings.Contains(body, "Background:") {
		t.Fatalf("expected background section in body: %q", body)
	}
	if !strings.Contains(body, "Starting Equipment") {
		t.Fatalf("expected starting equipment section in body: %q", body)
	}
}

func TestGenerateCharacterSheetFromBuildMultiClass(t *testing.T) {
	ui := &UI{
		classes: []Monster{
			{
				Name:        "Wizard",
				Source:      "PHB",
				CR:          "d6",
				Environment: []string{"INT", "WIS"},
				Raw:         map[string]any{"spellcastingAbility": "int"},
			},
			{
				Name:        "Fighter",
				Source:      "PHB",
				CR:          "d10",
				Environment: []string{"STR", "CON"},
				Raw:         map[string]any{},
			},
		},
		races: []Monster{
			{
				Name:   "Elf",
				Source: "PHB",
				Raw: map[string]any{
					"ability": []any{map[string]any{"dex": 2, "int": 1}},
					"speed":   map[string]any{"walk": 30},
				},
			},
		},
		spells: []Monster{
			{Name: "Light", CR: "0", Source: "PHB", Type: "Evocation"},
			{Name: "Shield", CR: "1", Source: "PHB", Type: "Abjuration"},
		},
	}
	build := CharacterBuild{
		Name:       "Aramil",
		Race:       "Elf",
		Classes:    []CharacterClassLevel{{Name: "Wizard", Levels: 3}, {Name: "Fighter", Levels: 2}},
		BaseScores: []int{10, 14, 12, 15, 8, 10},
		Feats:      []string{"Alert"},
		Spells:     []string{"Misty Step"},
	}
	sheet, normalized, err := ui.generateCharacterSheetFromBuild(build)
	if err != nil {
		t.Fatalf("unexpected build generation error: %v", err)
	}
	if got := classLevelsTotal(normalized.Classes); got != 5 {
		t.Fatalf("expected total level 5, got %d", got)
	}
	if !strings.Contains(sheet.Meta, "Build:") || !strings.Contains(sheet.Meta, "Wizard 3") || !strings.Contains(sheet.Meta, "Fighter 2") {
		t.Fatalf("expected multiclass summary in meta: %q", sheet.Meta)
	}
	if !strings.Contains(sheet.Body, "Feats") || !strings.Contains(sheet.Body, "Alert") {
		t.Fatalf("expected feats in body: %q", sheet.Body)
	}
	if !strings.Contains(sheet.Body, "Custom Spells") || !strings.Contains(sheet.Body, "Misty Step") {
		t.Fatalf("expected custom spells in body: %q", sheet.Body)
	}
}

func TestExtractClassSkillChoices(t *testing.T) {
	raw := map[string]any{
		"startingProficiencies": map[string]any{
			"skills": []any{
				map[string]any{
					"choose": map[string]any{
						"count": 2,
						"from":  []any{"arcana", "history", "insight"},
					},
				},
			},
		},
	}
	n, opts := extractClassSkillChoices(raw)
	if n != 2 {
		t.Fatalf("expected count 2, got %d", n)
	}
	if len(opts) != 3 || opts[0] != "Arcana" {
		t.Fatalf("unexpected options: %#v", opts)
	}
}

func TestInferCharacterBuildFromEntry(t *testing.T) {
	ui := &UI{
		classes: []Monster{{Name: "Artificer"}, {Name: "Fighter"}},
		races:   []Monster{{Name: "Aarakocra"}, {Name: "Elf"}},
	}
	entry := EncounterEntry{
		Custom:     true,
		CustomName: "Artificer Aarakocra Lv20",
		CustomBody: "Artificer Aarakocra (Level 20)\nArmor Class: 12",
	}
	build, ok := ui.inferCharacterBuildFromEntry(entry)
	if !ok || build == nil {
		t.Fatalf("expected inferred character build, got %#v", build)
	}
	if build.Race != "Aarakocra" {
		t.Fatalf("unexpected inferred race: %#v", build)
	}
	if len(build.Classes) != 1 || build.Classes[0].Name != "Artificer" || build.Classes[0].Levels != 20 {
		t.Fatalf("unexpected inferred classes: %#v", build.Classes)
	}
}

func TestSpellMaxLevelForProgression(t *testing.T) {
	if got := spellMaxLevelForProgression("full", 5); got != 3 {
		t.Fatalf("full caster level 5 expected 3, got %d", got)
	}
	if got := spellMaxLevelForProgression("half", 1); got != 0 {
		t.Fatalf("half caster level 1 expected 0, got %d", got)
	}
	if got := spellMaxLevelForProgression("half", 9); got != 2 {
		t.Fatalf("half caster level 9 expected 2, got %d", got)
	}
	if got := spellMaxLevelForProgression("artificer", 1); got != 1 {
		t.Fatalf("artificer level 1 expected 1, got %d", got)
	}
}

func TestGenerateCharacterSpellSelection(t *testing.T) {
	class := Monster{
		Name: "Wizard",
		Raw: map[string]any{
			"casterProgression": "full",
			"cantripProgression": []any{
				3, 3, 3, 4, 4,
			},
			"spellsKnownProgression": []any{
				2, 3, 4, 5, 6,
			},
		},
	}
	spells := []Monster{
		{Name: "Acid Splash", CR: "0"},
		{Name: "Light", CR: "0"},
		{Name: "Mage Hand", CR: "0"},
		{Name: "Shield", CR: "1"},
		{Name: "Magic Missile", CR: "1"},
		{Name: "Misty Step", CR: "2"},
		{Name: "Fireball", CR: "3"},
	}
	out := generateCharacterSpellSelection(class, 5, spells)
	if !strings.Contains(out, "Cantrips") {
		t.Fatalf("expected cantrips in output: %q", out)
	}
	if !strings.Contains(out, "Leveled") {
		t.Fatalf("expected leveled spells in output: %q", out)
	}
}

func TestMonsterCRScaling(t *testing.T) {
	m := mkMonster(1, "Aarakocra", 14, 13, "3d8")
	m.CR = "1/4"
	m.Raw["ac"] = []any{map[string]any{"ac": 12}}

	p, ok := scaleMonsterByCR(m, 2)
	if !ok {
		t.Fatal("expected scaling preview")
	}
	if p.BaseCR != "1/4" || p.TargetCR != "1" {
		t.Fatalf("unexpected CR scaling: %+v", p)
	}
	if p.TargetHP <= p.BaseHP {
		t.Fatalf("expected HP to increase when scaling up: %+v", p)
	}
	if p.TargetAC < 1 {
		t.Fatalf("unexpected target AC: %+v", p)
	}
}

func TestScaleDamageInText(t *testing.T) {
	in := "Bite. Hit: 8 (1d8 + 4) piercing damage. The target takes 3 fire damage."
	out := scaleDamageInText(in, 1.5)
	if !strings.Contains(out, "12 (1d8 + 4) piercing damage") {
		t.Fatalf("expected scaled parenthesized damage, got: %q", out)
	}
	if !strings.Contains(out, "5 fire damage") {
		t.Fatalf("expected scaled simple damage, got: %q", out)
	}
}

func TestFormat5eStructuredText(t *testing.T) {
	in := []any{
		map[string]any{
			"type": "section",
			"name": "Chapter One",
			"entries": []any{
				"Intro text.",
				map[string]any{
					"type":  "list",
					"items": []any{"A", "B"},
				},
			},
		},
	}
	out := format5eStructuredText(in, 0)
	if !strings.Contains(out, "## Chapter One") {
		t.Fatalf("expected section header, got: %q", out)
	}
	if !strings.Contains(out, "- A") || !strings.Contains(out, "- B") {
		t.Fatalf("expected list formatting, got: %q", out)
	}
}

func TestAddGeneratedCharacterToEncounter(t *testing.T) {
	ui := makeTestUI(t, nil)
	build := &CharacterBuild{
		Name:       "Wizard Elf Lv5",
		Race:       "Elf",
		Classes:    []CharacterClassLevel{{Name: "Wizard", Levels: 5}},
		BaseScores: []int{10, 12, 14, 16, 8, 10},
	}
	ui.addGeneratedCharacterToEncounter("Wizard Elf Lv5", 2, 12, 32, "META", "BODY", build)

	if len(ui.encounterItems) != 1 {
		t.Fatalf("expected 1 encounter entry, got %d", len(ui.encounterItems))
	}
	got := ui.encounterItems[0]
	if !got.Custom {
		t.Fatal("expected custom encounter entry")
	}
	if got.CustomName != "Wizard Elf Lv5" {
		t.Fatalf("unexpected custom name: %q", got.CustomName)
	}
	if got.CustomInit != 2 || got.BaseHP != 32 || got.CurrentHP != 32 || got.CustomAC != "12" {
		t.Fatalf("unexpected custom entry payload: %#v", got)
	}
	if got.CustomMeta != "META" || got.CustomBody != "BODY" {
		t.Fatalf("expected custom meta/body to be stored: %#v", got)
	}
	if got.Character == nil || got.Character.Race != "Elf" {
		t.Fatalf("expected character build persisted on encounter: %#v", got.Character)
	}
	if label := ui.encounterEntryDisplay(got); strings.Contains(label, "#") {
		t.Fatalf("custom encounter label should not contain ordinal: %q", label)
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

func TestDeleteAllMonsterEncounterEntriesKeepsCustomAndCharacter(t *testing.T) {
	ui := makeTestUI(t, []Monster{
		mkMonster(1, "A", 10, 5, "1d1"),
		mkMonster(2, "B", 12, 6, "1d1"),
	})
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
		{
			Custom:     true,
			CustomName: "Custom Entry",
			CustomInit: 2,
			BaseHP:     10,
			CurrentHP:  10,
		},
		{
			Custom:     true,
			CustomName: "Wizard Lv5",
			CustomInit: 3,
			BaseHP:     28,
			CurrentHP:  28,
			Character: &CharacterBuild{
				Name: "Wizard Lv5",
			},
		},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 6, CurrentHP: 6},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(3)
	ui.turnMode = true
	ui.turnRound = 4
	ui.turnIndex = 3

	ui.deleteAllMonsterEncounterEntries()

	if len(ui.encounterItems) != 2 {
		t.Fatalf("expected 2 entries left, got %d", len(ui.encounterItems))
	}
	if !ui.encounterItems[0].Custom || ui.encounterItems[0].Character != nil {
		t.Fatalf("expected first remaining entry to be plain custom, got %#v", ui.encounterItems[0])
	}
	if !ui.encounterItems[1].Custom || ui.encounterItems[1].Character == nil {
		t.Fatalf("expected second remaining entry to be character custom, got %#v", ui.encounterItems[1])
	}
	if !ui.turnMode {
		t.Fatal("expected turn mode to stay enabled when entries remain")
	}
	if ui.turnIndex != 0 {
		t.Fatalf("expected turn index remapped to 0, got %d", ui.turnIndex)
	}
}

func TestDeleteAllMonsterEncounterEntriesNoopWhenOnlyCustom(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "A", 10, 5, "1d1")})
	ui.encounterItems = []EncounterEntry{
		{
			Custom:     true,
			CustomName: "Custom One",
			CustomInit: 1,
			BaseHP:     7,
			CurrentHP:  7,
		},
	}
	ui.renderEncounterList()
	beforeUndo := len(ui.encounterUndo)

	ui.deleteAllMonsterEncounterEntries()

	if len(ui.encounterItems) != 1 || !ui.encounterItems[0].Custom {
		t.Fatalf("expected custom entry unchanged, got %#v", ui.encounterItems)
	}
	if len(ui.encounterUndo) != beforeUndo {
		t.Fatalf("expected undo stack unchanged, before=%d after=%d", beforeUndo, len(ui.encounterUndo))
	}
}

func TestEncounterNPCLevels(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "A", 10, 5, "1d1")})
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 5, CurrentHP: 5},
		{
			Custom:     true,
			CustomName: "Wizard",
			Character: &CharacterBuild{
				Classes: []CharacterClassLevel{
					{Name: "Wizard", Levels: 4},
				},
			},
		},
		{
			Custom:     true,
			CustomName: "Fighter",
			Character: &CharacterBuild{
				Classes: []CharacterClassLevel{
					{Name: "Fighter", Levels: 2},
					{Name: "Wizard", Levels: 1},
				},
			},
		},
		{
			Custom:     true,
			CustomName: "No Build",
		},
	}
	levels := ui.encounterNPCLevels()
	if !reflect.DeepEqual(levels, []int{4, 3}) {
		t.Fatalf("unexpected npc levels: %#v", levels)
	}
}

func TestBuildEncounterGenerationPreviewAndApply(t *testing.T) {
	makeMonster := func(id int, name, cr string, envs []string, hp int) Monster {
		return Monster{
			ID:          id,
			Name:        name,
			CR:          cr,
			Environment: envs,
			Raw: map[string]any{
				"name": name,
				"dex":  12,
				"hp": map[string]any{
					"average": hp,
					"formula": "2d8",
				},
			},
		}
	}
	monsters := []Monster{
		makeMonster(1, "Wolf", "1/4", []string{"forest"}, 11),
		makeMonster(2, "Bandit", "1/8", []string{"urban"}, 11),
		makeMonster(3, "Goblin", "1/4", []string{"forest"}, 7),
	}
	ui := makeTestUI(t, monsters)
	ui.encounterItems = []EncounterEntry{
		{
			Custom:     true,
			CustomName: "NPC Wizard",
			BaseHP:     20,
			CurrentHP:  20,
			Character: &CharacterBuild{
				Classes: []CharacterClassLevel{
					{Name: "Wizard", Levels: 3},
				},
			},
		},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 11, CurrentHP: 11},
	}

	preview, err := ui.buildEncounterGenerationPreview(3, 1, 3, "forest")
	if err != nil {
		t.Fatalf("buildEncounterGenerationPreview error: %v", err)
	}
	if len(preview.MonsterIDs) != 3 {
		t.Fatalf("expected 3 generated monsters, got %d", len(preview.MonsterIDs))
	}
	for _, idx := range preview.MonsterIDs {
		if idx < 0 || idx >= len(ui.monsters) {
			t.Fatalf("invalid generated index: %d", idx)
		}
		if !monsterMatchesEnvironment(ui.monsters[idx], "forest") {
			t.Fatalf("expected forest monster, got %#v", ui.monsters[idx])
		}
	}

	added := ui.applyEncounterGenerationPreview(preview)
	if added != 3 {
		t.Fatalf("expected added=3, got %d", added)
	}
	if len(ui.encounterItems) != 4 {
		t.Fatalf("expected 4 encounter items after apply, got %d", len(ui.encounterItems))
	}
	if !ui.encounterItems[0].Custom || ui.encounterItems[0].Character == nil {
		t.Fatalf("expected first item to keep existing custom NPC, got %#v", ui.encounterItems[0])
	}
	for i := 1; i < len(ui.encounterItems); i++ {
		if ui.encounterItems[i].Custom {
			t.Fatalf("expected generated monster at %d, got custom %#v", i, ui.encounterItems[i])
		}
		if ui.monsterScale[ui.encounterItems[i].MonsterIndex] != 1 {
			t.Fatalf("expected monster scale +1 for generated monster, got %d", ui.monsterScale[ui.encounterItems[i].MonsterIndex])
		}
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

func TestSetFilterOptionsForItemsAndSpellsPopulateEnv(t *testing.T) {
	monsters := []Monster{mkMonster(1, "A", 10, 5, "1d1")}
	items := []Monster{
		{ID: 1, Name: "Item A", Source: "DMG", CR: "common", Type: "wondrous", Environment: []string{"DMG"}},
		{ID: 2, Name: "Item B", Source: "XGE", CR: "rare", Type: "weapon", Environment: []string{"XGE"}},
	}
	spells := []Monster{
		{ID: 1, Name: "Spell A", Source: "PHB", CR: "1", Type: "evocation", Environment: []string{"PHB"}},
		{ID: 2, Name: "Spell B", Source: "XGE", CR: "3", Type: "illusion", Environment: []string{"XGE"}},
	}

	path := filepath.Join(t.TempDir(), "encounters.yaml")
	dicePath := filepath.Join(t.TempDir(), "dice.yaml")
	ui := newUI(monsters, items, spells, nil, nil, nil, nil, nil, nil, nil, nil, path, dicePath)

	ui.browseMode = BrowseItems
	ui.setFilterOptionsForMode()
	if len(ui.envOptions) <= 1 {
		t.Fatalf("items env options not populated: %#v", ui.envOptions)
	}
	if !containsString(ui.envOptions, "DMG") || !containsString(ui.envOptions, "XGE") {
		t.Fatalf("items env options missing expected sources: %#v", ui.envOptions)
	}

	ui.browseMode = BrowseSpells
	ui.setFilterOptionsForMode()
	if len(ui.envOptions) <= 1 {
		t.Fatalf("spells env options not populated: %#v", ui.envOptions)
	}
	if !containsString(ui.envOptions, "PHB") || !containsString(ui.envOptions, "XGE") {
		t.Fatalf("spells env options missing expected sources: %#v", ui.envOptions)
	}
}

func TestEmbeddedItemsAndSpellsEnvOptionsNotOnlyAll(t *testing.T) {
	items, _, _, _, err := loadItemsFromBytes(embeddedItemsYAML)
	if err != nil {
		t.Fatalf("load embedded items failed: %v", err)
	}
	spells, _, _, _, err := loadSpellsFromBytes(embeddedSpellsYAML)
	if err != nil {
		t.Fatalf("load embedded spells failed: %v", err)
	}

	monsters := []Monster{mkMonster(1, "A", 10, 5, "1d1")}
	path := filepath.Join(t.TempDir(), "encounters.yaml")
	dicePath := filepath.Join(t.TempDir(), "dice.yaml")
	ui := newUI(monsters, items, spells, nil, nil, nil, nil, nil, nil, nil, nil, path, dicePath)

	ui.browseMode = BrowseItems
	ui.setFilterOptionsForMode()
	if len(ui.envOptions) <= 1 {
		t.Fatalf("embedded items env options not populated: %#v", ui.envOptions)
	}

	ui.browseMode = BrowseSpells
	ui.setFilterOptionsForMode()
	if len(ui.envOptions) <= 1 {
		t.Fatalf("embedded spells env options not populated: %#v", ui.envOptions)
	}
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
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
	t.Setenv("LAZY5E_HOME", tmp)
	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(oldWd) }()

	monsters := []Monster{mkMonster(100, "A", 12, 13, "3d8"), mkMonster(200, "B", 14, 20, "4d8")}
	path := filepath.Join(tmp, "my-enc.yaml")
	ui := newUI(monsters, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, path, filepath.Join(tmp, "dice.yaml"))
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 13, CurrentHP: 8, HPFormula: "3d8", UseRolledHP: true, RolledHP: 10, HasInitRoll: true, InitRoll: 15},
		{MonsterIndex: 1, Ordinal: 1, BaseHP: 20, CurrentHP: 20, HPFormula: "4d8", UseRolledHP: false, RolledHP: 0, HasInitRoll: false, InitRoll: 0},
		{MonsterIndex: -1, Ordinal: 1, Custom: true, CustomName: "Solum", CustomInit: 2, CustomAC: "13", CustomPassive: 14, HasCustomPassive: true, BaseHP: 10, CurrentHP: 6, HPFormula: "", UseRolledHP: false, RolledHP: 0, HasInitRoll: true, InitRoll: 12},
	}
	ui.encounterSerial = map[int]int{0: 1, 1: 1}
	ui.turnMode = true
	ui.turnIndex = 2
	ui.turnRound = 3

	if err := ui.saveEncountersAs(path); err != nil {
		t.Fatalf("saveEncountersAs failed: %v", err)
	}

	ui2 := newUI(monsters, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, path, filepath.Join(tmp, "dice.yaml"))
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
	if !ui2.encounterItems[2].HasCustomPassive || ui2.encounterItems[2].CustomPassive != 14 {
		t.Fatalf("custom passive not restored: %#v", ui2.encounterItems[2])
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

func TestEncounterConditionsBadgeAndRoundProgress(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})
	ui.encounterItems = []EncounterEntry{
		{
			MonsterIndex: 0,
			Ordinal:      1,
			BaseHP:       13,
			CurrentHP:    13,
			Conditions: map[string]int{
				"O": 1, // Poisoned
				"U": 2, // Unconscious
			},
		},
	}
	ui.turnMode = true
	ui.turnIndex = 0
	ui.turnRound = 1
	ui.renderEncounterList()

	line, _ := ui.encounter.GetItemText(0)
	if !strings.Contains(line, "*[1]") || !strings.Contains(line, "O1U2") {
		t.Fatalf("unexpected encounter condition badge: %q", line)
	}

	// Advance one round and ensure condition rounds increase.
	ui.nextEncounterTurn()
	if ui.encounterItems[0].Conditions["O"] != 2 || ui.encounterItems[0].Conditions["U"] != 3 {
		t.Fatalf("expected condition rounds to increase: %#v", ui.encounterItems[0].Conditions)
	}
}

func TestRemoveEncounterConditionByCode(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})
	ui.encounterItems = []EncounterEntry{
		{
			MonsterIndex: 0,
			Ordinal:      1,
			BaseHP:       13,
			CurrentHP:    13,
			Conditions:   map[string]int{"O": 2, "U": 1},
		},
	}
	if ok := ui.removeEncounterConditionByCode(0, "O"); !ok {
		t.Fatal("expected condition O to be removed")
	}
	if _, ok := ui.encounterItems[0].Conditions["O"]; ok {
		t.Fatalf("condition O should be gone: %#v", ui.encounterItems[0].Conditions)
	}
	if ui.encounterItems[0].Conditions["U"] != 1 {
		t.Fatalf("other conditions must remain untouched: %#v", ui.encounterItems[0].Conditions)
	}
}

func TestEncounterDetailsIncludeConditionsForCustomEntry(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})
	ui.encounterItems = []EncounterEntry{
		{
			Custom:     true,
			CustomName: "Artificer Aarakocra Lv20",
			CustomMeta: "Artificer Aarakocra Lv20\nAC: 12\nHP: 123\nInit: +2",
			CustomInit: 2,
			BaseHP:     123,
			CurrentHP:  123,
			Conditions: map[string]int{
				"B": 11,
				"C": 13,
			},
		},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(0)
	ui.renderDetailByEncounterIndex(0)

	meta := ui.detailMeta.GetText(false)
	if !strings.Contains(meta, "Conditions:") {
		t.Fatalf("expected Conditions line in details, got: %q", meta)
	}
	if !strings.Contains(strings.ToLower(meta), "b11 blinded") || !strings.Contains(strings.ToLower(meta), "c13 charmed") {
		t.Fatalf("expected condition names and rounds in details, got: %q", meta)
	}
}

func TestEncounterDetailsIncludePassivePerceptionForMonster(t *testing.T) {
	mon := mkMonster(1, "Scout", 14, 16, "3d8")
	mon.Raw["wis"] = 13
	ui := makeTestUI(t, []Monster{mon})
	ui.encounterItems = []EncounterEntry{
		{MonsterIndex: 0, Ordinal: 1, BaseHP: 16, CurrentHP: 16},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(0)
	ui.renderDetailByEncounterIndex(0)

	meta := ui.detailMeta.GetText(false)
	if !strings.Contains(meta, "Passive Perception:") {
		t.Fatalf("expected Passive Perception line, got: %q", meta)
	}
	if !strings.Contains(meta, "11") { // 10 + WIS mod(13) = 11
		t.Fatalf("expected passive perception 11, got: %q", meta)
	}
}

func TestMonsterDetailsIncludeXPFromCR(t *testing.T) {
	mon := mkMonster(1, "Goblin", 14, 7, "2d6")
	mon.CR = "1/4"
	ui := makeTestUI(t, []Monster{mon})
	ui.renderDetailByMonsterIndex(0)
	meta := ui.detailMeta.GetText(false)
	if !strings.Contains(meta, "XP:") || !strings.Contains(meta, "50") {
		t.Fatalf("expected XP 50 in monster details, got: %q", meta)
	}
}

func TestEncounterDetailsIncludePassivePerceptionForCharacterBuild(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "A", 10, 5, "1d1")})
	ui.encounterItems = []EncounterEntry{
		{
			Custom:     true,
			CustomName: "Wizard PNG",
			CustomMeta: "Wizard PNG\nAC: 15\nHP: 20\nInit: +2",
			BaseHP:     20,
			CurrentHP:  20,
			Character: &CharacterBuild{
				BaseScores: []int{8, 14, 12, 16, 15, 10},
			},
		},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(0)
	ui.renderDetailByEncounterIndex(0)

	meta := ui.detailMeta.GetText(false)
	if !strings.Contains(meta, "Passive Perception:") {
		t.Fatalf("expected Passive Perception line, got: %q", meta)
	}
	if !strings.Contains(meta, "12") { // 10 + WIS mod(15) = 12
		t.Fatalf("expected passive perception 12, got: %q", meta)
	}
}

func TestEncounterDetailsIncludeUnknownPassivePerceptionForCustom(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "A", 10, 5, "1d1")})
	ui.encounterItems = []EncounterEntry{
		{
			Custom:     true,
			CustomName: "Custom NPC",
			BaseHP:     12,
			CurrentHP:  12,
		},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(0)
	ui.renderDetailByEncounterIndex(0)

	meta := ui.detailMeta.GetText(false)
	if !strings.Contains(meta, "Passive Perception:") || !strings.Contains(meta, "?") {
		t.Fatalf("expected unknown Passive Perception line, got: %q", meta)
	}
}

func TestEncounterDetailsIncludeCustomPassivePerceptionInput(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "A", 10, 5, "1d1")})
	ui.encounterItems = []EncounterEntry{
		{
			Custom:           true,
			CustomName:       "Custom NPC",
			CustomPassive:    17,
			HasCustomPassive: true,
			BaseHP:           12,
			CurrentHP:        12,
		},
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(0)
	ui.renderDetailByEncounterIndex(0)

	meta := ui.detailMeta.GetText(false)
	if !strings.Contains(meta, "Passive Perception:") || !strings.Contains(meta, "17") {
		t.Fatalf("expected custom passive perception 17, got: %q", meta)
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

func TestGenerateIndividualTreasure(t *testing.T) {
	seq := []int{0, 0, 0, 0, 0, 0} // d100=1, then minimal dice rolls
	i := 0
	randFn := func(max int) int {
		if i >= len(seq) {
			return 0
		}
		v := seq[i]
		i++
		if v < 0 {
			v = 0
		}
		if v >= max {
			v = max - 1
		}
		return v
	}

	out, err := generateIndividualTreasure("1/2", randFn)
	if err != nil {
		t.Fatalf("unexpected treasure error: %v", err)
	}
	if out.Band != "CR 0-4" {
		t.Fatalf("unexpected band: %s", out.Band)
	}
	if out.D100 != 1 {
		t.Fatalf("unexpected d100: %d", out.D100)
	}
	if out.Coins["cp"] != 5 {
		t.Fatalf("expected 5 cp, got %+v", out.Coins)
	}
}

func TestGenerateIndividualTreasureHighCR(t *testing.T) {
	seq := []int{55, 0, 0, 0} // d100=56 -> CR 17+ last row
	i := 0
	randFn := func(max int) int {
		if i >= len(seq) {
			return 0
		}
		v := seq[i]
		i++
		if v < 0 {
			v = 0
		}
		if v >= max {
			v = max - 1
		}
		return v
	}

	out, err := generateIndividualTreasure("20", randFn)
	if err != nil {
		t.Fatalf("unexpected treasure error: %v", err)
	}
	if out.Band != "CR 17+" {
		t.Fatalf("unexpected band: %s", out.Band)
	}
	if out.Coins["gp"] != 1000 || out.Coins["pp"] != 200 {
		t.Fatalf("unexpected coins: %+v", out.Coins)
	}
}

func TestGenerateIndividualTreasureInvalidCR(t *testing.T) {
	if _, err := generateIndividualTreasure("bad-cr", nil); err == nil {
		t.Fatal("expected invalid CR error")
	}
}

func TestGenerateLairTreasure(t *testing.T) {
	seq := []int{0, 0, 0, 0, 0, 0, 0, 0} // d100=1, then minimum dice
	i := 0
	randFn := func(max int) int {
		if i >= len(seq) {
			return 0
		}
		v := seq[i]
		i++
		if v < 0 {
			v = 0
		}
		if v >= max {
			v = max - 1
		}
		return v
	}
	out, err := generateLairTreasure("2", randFn)
	if err != nil {
		t.Fatalf("unexpected lair treasure error: %v", err)
	}
	if out.Kind != "Lair (Hoard) Treasure" {
		t.Fatalf("unexpected kind: %s", out.Kind)
	}
	if out.Band != "CR 0-4" {
		t.Fatalf("unexpected band: %s", out.Band)
	}
	if out.Coins["cp"] != 600 || out.Coins["sp"] != 300 || out.Coins["gp"] != 20 {
		t.Fatalf("unexpected base coins: %+v", out.Coins)
	}
}

func TestGenerateLairTreasureIncludesTypesInExtras(t *testing.T) {
	seq := []int{
		9,                // d100=10 -> CR 0-4, gems 10gp
		0, 0, 0, 0, 0, 0, // base coins dice
		0, 0, // gems count 2d6 -> 2
		0, 1, // gem type picks
	}
	i := 0
	randFn := func(max int) int {
		if i >= len(seq) {
			return 0
		}
		v := seq[i]
		i++
		if v < 0 {
			v = 0
		}
		if v >= max {
			v = max - 1
		}
		return v
	}
	out, err := generateLairTreasure("1", randFn)
	if err != nil {
		t.Fatalf("unexpected lair treasure error: %v", err)
	}
	if len(out.Extras) == 0 {
		t.Fatalf("expected extras with detailed types, got none")
	}
	if !strings.Contains(out.Extras[0], ": ") {
		t.Fatalf("expected typed extras, got %q", out.Extras[0])
	}
}

func TestFormatItemBasePriceAndMagicEconomy(t *testing.T) {
	raw := map[string]any{"value": 2500, "wondrous": true}
	price := formatItemBasePrice(raw)
	if price == "" {
		t.Fatal("expected formatted price")
	}
	if !strings.Contains(price, "gp") {
		t.Fatalf("expected gp in formatted price, got %q", price)
	}

	econ, ok := magicItemEconomy(raw, "rare")
	if !ok {
		t.Fatal("expected magical economy for rare item")
	}
	if !strings.Contains(econ.BuyCost, "5,000") {
		t.Fatalf("unexpected buy cost: %q", econ.BuyCost)
	}
	if len(econ.Procedure) == 0 {
		t.Fatal("expected non-empty crafting procedure")
	}
}

func TestMagicItemEconomyNonMagical(t *testing.T) {
	raw := map[string]any{"value": 50}
	if _, ok := magicItemEconomy(raw, ""); ok {
		t.Fatal("expected non-magical item to not have magic economy")
	}
}

func TestFilterItemsByTreasureType(t *testing.T) {
	items := []Monster{
		{Name: "Potion of Healing", Type: "potion", Raw: map[string]any{"potion": true}},
		{Name: "Ring of Protection", Type: "ring", Raw: map[string]any{"ring": true}},
		{Name: "Staff of Power", Type: "staff", Raw: map[string]any{"staff": true}},
	}
	got := filterItemsByTreasureType(items, "potion")
	if len(got) != 1 || got[0].Name != "Potion of Healing" {
		t.Fatalf("unexpected potion filter result: %#v", got)
	}
	got = filterItemsByTreasureType(items, "random")
	if len(got) != 3 {
		t.Fatalf("expected random to return all items, got %d", len(got))
	}
}

func TestFilterItemsByTreasureKinds(t *testing.T) {
	items := []Monster{
		{ID: 1, Name: "Potion of Healing", Type: "potion", Raw: map[string]any{"potion": true}},
		{ID: 2, Name: "Ring of Protection", Type: "ring", Raw: map[string]any{"ring": true}},
		{ID: 3, Name: "Staff of Power", Type: "staff", Raw: map[string]any{"staff": true}},
	}
	got := filterItemsByTreasureKinds(items, []string{"potion", "ring"})
	if len(got) != 2 {
		t.Fatalf("expected 2 filtered items, got %d", len(got))
	}
	got = filterItemsByTreasureKinds(items, []string{"random"})
	if len(got) != 3 {
		t.Fatalf("expected random to include all, got %d", len(got))
	}
}

func TestFilterSpellsByLevel(t *testing.T) {
	spells := []Monster{
		{Name: "Magic Missile", CR: "1", Type: "Evocation"},
		{Name: "Fireball", CR: "3", Type: "Evocation"},
		{Name: "Wish", CR: "9", Type: "Conjuration"},
	}
	got := filterSpellsByLevel(spells, "3")
	if len(got) != 1 || got[0].Name != "Fireball" {
		t.Fatalf("unexpected level filter result: %#v", got)
	}
	got = filterSpellsByLevel(spells, "random")
	if len(got) != 3 {
		t.Fatalf("random should return all spells, got %d", len(got))
	}
	got = filterSpellsByFilter(spells, SpellTreasureFilter{Level: "3", School: "Evocation"})
	if len(got) != 1 || got[0].Name != "Fireball" {
		t.Fatalf("unexpected advanced spell filter result: %#v", got)
	}
}

func TestSaveTreasureToPath(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "A", 10, 5, "1d1")})
	ui.treasureText = "sample treasure"
	path := filepath.Join(t.TempDir(), "tesoro-test.yaml")
	if err := ui.saveTreasureToPath(path, false); err != nil {
		t.Fatalf("saveTreasureToPath failed: %v", err)
	}
	if !fileExists(path) {
		t.Fatalf("expected file to exist: %s", path)
	}
	if err := ui.saveTreasureToPath(path, false); err == nil {
		t.Fatal("expected overwrite protection error")
	}
	if err := ui.saveTreasureToPath(path, true); err != nil {
		t.Fatalf("expected overwrite save to succeed: %v", err)
	}
}

func TestHelpForFocusIncludesPanelShortcuts(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})

	encHelp := ui.helpForFocus(ui.encounter)
	for _, expected := range []string{
		"i : tira iniziativa entry selezionata",
		"I : tira iniziativa per tutte le entry",
		"S : ordina entry per tiro iniziativa",
		"D : elimina tutte le entry mostro (mantiene custom/personaggi)",
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

func TestRerollAllDiceResults(t *testing.T) {
	ui := makeTestUI(t, []Monster{mkMonster(1, "Aarakocra", 14, 13, "3d8")})
	ui.diceLog = []DiceResult{
		{Expression: "1d1+2", Output: "old"},
		{Expression: "d1", Output: "old"},
		{Expression: "", Output: "old"},
	}
	ui.renderDiceList()
	ui.rerollAllDiceResults()
	if len(ui.diceLog) != 3 {
		t.Fatalf("expected same number of rows, got %d", len(ui.diceLog))
	}
	if !strings.Contains(ui.diceLog[0].Output, "= 3") {
		t.Fatalf("expected first row rerolled, got %q", ui.diceLog[0].Output)
	}
	if !strings.Contains(ui.diceLog[1].Output, "= 1") {
		t.Fatalf("expected second row rerolled, got %q", ui.diceLog[1].Output)
	}
	if ui.diceLog[2].Output != "old" {
		t.Fatalf("expected empty expression row unchanged, got %q", ui.diceLog[2].Output)
	}
}

func TestResolveCreateCharacterSubmit(t *testing.T) {
	if got := resolveCreateCharacterSubmit(0, -1, false); got != submitFocusRace {
		t.Fatalf("expected focus race action, got %v", got)
	}
	if got := resolveCreateCharacterSubmit(1, -1, true); got != submitNone {
		t.Fatalf("expected no action while race is open, got %v", got)
	}
	if got := resolveCreateCharacterSubmit(1, -1, false); got != submitGenerate {
		t.Fatalf("expected generate action, got %v", got)
	}
	if got := resolveCreateCharacterSubmit(-1, 1, false); got != submitCancel {
		t.Fatalf("expected cancel action, got %v", got)
	}
}

func TestResolveEncounterEditSubmit(t *testing.T) {
	if got := resolveEncounterEditSubmit(0, -1, false); got != submitFocusRace {
		t.Fatalf("expected focus class action, got %v", got)
	}
	if got := resolveEncounterEditSubmit(1, -1, false); got != submitFocusLevels {
		t.Fatalf("expected focus levels action, got %v", got)
	}
	if got := resolveEncounterEditSubmit(2, -1, false); got != submitApply {
		t.Fatalf("expected apply action, got %v", got)
	}
	if got := resolveEncounterEditSubmit(-1, 1, false); got != submitCancel {
		t.Fatalf("expected cancel action, got %v", got)
	}
}

func TestEncounterCharacterBuildRoundTrip(t *testing.T) {
	ui := makeCharacterUI(t)
	build := CharacterBuild{
		Name:       "Aramil",
		Race:       "Elf",
		Classes:    []CharacterClassLevel{{Name: "Wizard", Levels: 5}},
		BaseScores: []int{10, 14, 12, 15, 8, 10},
	}
	ui.addGeneratedCharacterToEncounter("Aramil", 2, 13, 30, "META", "BODY", &build)
	if err := ui.saveEncounters(); err != nil {
		t.Fatalf("save encounters failed: %v", err)
	}
	loaded := makeCharacterUI(t)
	loaded.encountersPath = ui.encountersPath
	if err := loaded.loadEncounters(); err != nil {
		t.Fatalf("load encounters failed: %v", err)
	}
	if len(loaded.encounterItems) != 1 || loaded.encounterItems[0].Character == nil {
		t.Fatalf("expected loaded character build, got %#v", loaded.encounterItems)
	}
	if got := loaded.encounterItems[0].Character.Classes[0].Name; got != "Wizard" {
		t.Fatalf("unexpected class after round trip: %s", got)
	}
}

func TestApplyCharacterBuildEditAndUndoRedo(t *testing.T) {
	ui := makeCharacterUI(t)
	build := CharacterBuild{
		Name:       "Aramil",
		Race:       "Elf",
		Classes:    []CharacterClassLevel{{Name: "Wizard", Levels: 3}},
		BaseScores: []int{10, 14, 12, 15, 8, 10},
	}
	ui.addGeneratedCharacterToEncounter("Aramil", 2, 13, 24, "META", "BODY", &build)
	ui.encounter.SetCurrentItem(0)
	next := build
	next.Classes = []CharacterClassLevel{{Name: "Wizard", Levels: 4}, {Name: "Fighter", Levels: 1}}
	ui.pushEncounterUndo()
	if err := ui.applyCharacterBuildToEncounter(0, next); err != nil {
		t.Fatalf("apply edit failed: %v", err)
	}
	if got := classLevelsTotal(ui.encounterItems[0].Character.Classes); got != 5 {
		t.Fatalf("expected level 5 after edit, got %d", got)
	}
	ui.undoEncounterCommand()
	if got := classLevelsTotal(ui.encounterItems[0].Character.Classes); got != 3 {
		t.Fatalf("expected level 3 after undo, got %d", got)
	}
	ui.redoEncounterCommand()
	if got := classLevelsTotal(ui.encounterItems[0].Character.Classes); got != 5 {
		t.Fatalf("expected level 5 after redo, got %d", got)
	}
}

func TestSaveLoadCharacterBuildFile(t *testing.T) {
	ui := makeCharacterUI(t)
	build := CharacterBuild{
		Name:       "Aramil",
		Race:       "Elf",
		Classes:    []CharacterClassLevel{{Name: "Wizard", Levels: 2}},
		BaseScores: []int{10, 14, 12, 15, 8, 10},
	}
	ui.addGeneratedCharacterToEncounter("Aramil", 2, 13, 18, "META", "BODY", &build)
	ui.encounter.SetCurrentItem(0)
	path := filepath.Join(t.TempDir(), "char-build.yaml")
	if err := ui.saveCharacterBuildAs(path); err != nil {
		t.Fatalf("save build failed: %v", err)
	}
	ui.encounterItems[0].Character.Classes[0].Levels = 1
	if err := ui.loadCharacterBuildFrom(path); err != nil {
		t.Fatalf("load build failed: %v", err)
	}
	if got := ui.encounterItems[0].Character.Classes[0].Levels; got != 2 {
		t.Fatalf("expected loaded level 2, got %d", got)
	}
}
