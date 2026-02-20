package main

import (
	crand "crypto/rand"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vcrini/diceroll"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

const (
	helpText               = " [black:gold] q [-:-] esci  [black:gold] / [-:-] cerca (Name/Description)  [black:gold] tab [-:-] focus  [black:gold] 0/1/2/3 [-:-] pannelli  [black:gold] [/] [-:-] cycle Monsters/Items/Spells  [black:gold] a[-:-] roll Dice  [black:gold] f[-:-] fullscreen panel  [black:gold] j/k [-:-] naviga  [black:gold] d [-:-] del encounter | details<->treasure  [black:gold] s/l [-:-] save/load  [black:gold] i/I [-:-] roll init one/all  [black:gold] S [-:-] sort init  [black:gold] * [-:-] turn mode  [black:gold] n/p [-:-] next/prev turn  [black:gold] u/r [-:-] undo/redo  [black:gold] spazio [-:-] avg/formula HP  [black:gold] ←/→ [-:-] danno/cura encounter  [black:gold] PgUp/PgDn [-:-] scroll Description "
	defaultEncountersPath  = "encounters.yaml"
	lastEncountersPathFile = ".encounters_last_path"
	defaultDicePath        = "dice.yaml"
	lastDicePathFile       = ".dice_last_path"
	filtersStatePath       = ".filters_state.yaml"
)

//go:embed data/monster.yaml
var embeddedMonstersYAML []byte

//go:embed data/item.yaml
var embeddedItemsYAML []byte

//go:embed data/spell.yaml
var embeddedSpellsYAML []byte

type BrowseMode int

const (
	BrowseMonsters BrowseMode = iota
	BrowseItems
	BrowseSpells
)

type Monster struct {
	ID          int
	Name        string
	CR          string
	Environment []string
	Source      string
	Type        string
	Raw         map[string]any
}

type dataset struct {
	Monsters []map[string]any `yaml:"monsters"`
}

type itemsDataset struct {
	Items []map[string]any `yaml:"items"`
}

type spellsDataset struct {
	Spells []map[string]any `yaml:"spells"`
}

type EncounterEntry struct {
	MonsterIndex int
	Ordinal      int
	Custom       bool
	CustomName   string
	CustomInit   int
	CustomAC     string
	BaseHP       int
	CurrentHP    int
	HPFormula    string
	UseRolledHP  bool
	RolledHP     int
	HasInitRoll  bool
	InitRoll     int
}

type PersistedEncounters struct {
	Version   int                      `yaml:"version"`
	Items     []PersistedEncounterItem `yaml:"items"`
	TurnMode  bool                     `yaml:"turn_mode,omitempty"`
	TurnIndex int                      `yaml:"turn_index,omitempty"`
	TurnRound int                      `yaml:"turn_round,omitempty"`
}

type PersistedEncounterItem struct {
	MonsterID  int    `yaml:"monster_id"`
	Ordinal    int    `yaml:"ordinal"`
	Custom     bool   `yaml:"custom,omitempty"`
	CustomName string `yaml:"custom_name,omitempty"`
	CustomInit int    `yaml:"custom_init,omitempty"`
	CustomAC   string `yaml:"custom_ac,omitempty"`
	BaseHP     int    `yaml:"base_hp"`
	CurrentHP  int    `yaml:"current_hp"`
	HPFormula  string `yaml:"hp_formula,omitempty"`
	UseRolled  bool   `yaml:"use_rolled,omitempty"`
	RolledHP   int    `yaml:"rolled_hp,omitempty"`
	InitRolled bool   `yaml:"init_rolled,omitempty"`
	InitRoll   int    `yaml:"init_roll,omitempty"`
}

type PersistedDice struct {
	Version int          `yaml:"version"`
	Items   []DiceResult `yaml:"items"`
}

type PersistedFilterMode struct {
	Name    string   `yaml:"name,omitempty"`
	Env     string   `yaml:"env,omitempty"`
	Sources []string `yaml:"sources,omitempty"`
	CR      string   `yaml:"cr,omitempty"`
	Type    string   `yaml:"type,omitempty"`
}

type PersistedFilters struct {
	Version  int                 `yaml:"version"`
	Active   string              `yaml:"active,omitempty"`
	Monsters PersistedFilterMode `yaml:"monsters,omitempty"`
	Items    PersistedFilterMode `yaml:"items,omitempty"`
	Spells   PersistedFilterMode `yaml:"spells,omitempty"`
}

type EncounterUndoState struct {
	Items    []EncounterEntry
	Serial   map[int]int
	Selected int
}

type DiceResult struct {
	Expression string `yaml:"expression"`
	Output     string `yaml:"output"`
}

type DiceUndoState struct {
	Items    []DiceResult
	Selected int
}

type treasureCoinRoll struct {
	Currency  string
	DiceN     int
	DiceSides int
	Mult      int
}

type treasureOutcome struct {
	Kind      string
	Band      string
	D100      int
	Coins     map[string]int
	Breakdown []string
	Extras    []string
}

type UI struct {
	app           *tview.Application
	monsters      []Monster
	items         []Monster
	spells        []Monster
	browseMode    BrowseMode
	filtered      []int
	envOptions    []string
	sourceOptions []string
	crOptions     []string
	typeOptions   []string

	nameFilter    string
	envFilter     string
	sourceFilters map[string]struct{}
	crFilter      string
	typeFilter    string

	nameInput      *tview.InputField
	envDrop        *tview.DropDown
	sourceDrop     *tview.DropDown
	crDrop         *tview.DropDown
	typeDrop       *tview.DropDown
	dice           *tview.List
	encounter      *tview.List
	list           *tview.List
	detailMeta     *tview.TextView
	detailTreasure *tview.TextView
	detailRaw      *tview.TextView
	detailBottom   *tview.Pages
	status         *tview.TextView
	pages          *tview.Pages
	leftPanel      *tview.Flex
	monstersPanel  *tview.Flex
	mainRow        *tview.Flex
	detailPanel    *tview.Flex
	filterHost     *tview.Pages

	focusOrder   []tview.Primitive
	rawText      string
	rawQuery     string
	treasureText string
	diceLog      []DiceResult
	diceRender   bool
	wideFilter   bool
	modeFilters  map[BrowseMode]PersistedFilterMode

	encounterSerial map[int]int
	encounterItems  []EncounterEntry
	encountersPath  string
	dicePath        string
	encounterUndo   []EncounterUndoState
	encounterRedo   []EncounterUndoState
	diceUndo        []DiceUndoState
	diceRedo        []DiceUndoState
	turnMode        bool
	turnIndex       int
	turnRound       int

	helpVisible          bool
	helpReturnFocus      tview.Primitive
	addCustomVisible     bool
	fullscreenActive     bool
	fullscreenTarget     string
	spellShortcutAlt     bool
	updatingSourceDrop   bool
	activeBottomPanel    string
	itemTreasureVisible  bool
	spellTreasureVisible bool
}

func main() {
	rand.Seed(time.Now().UnixNano())

	yamlPath := strings.TrimSpace(os.Getenv("MONSTERS_YAML"))
	encountersPath := strings.TrimSpace(os.Getenv("ENCOUNTERS_YAML"))
	dicePath := strings.TrimSpace(os.Getenv("DICE_YAML"))
	if encountersPath == "" {
		encountersPath = readLastEncountersPath()
	}
	if dicePath == "" {
		dicePath = readLastDicePath()
	}
	var (
		monsters []Monster
		items    []Monster
		spells   []Monster
		envs     []string
		crs      []string
		types    []string
		err      error
	)

	if yamlPath != "" {
		monsters, envs, crs, types, err = loadMonstersFromPath(yamlPath)
		if err != nil {
			log.Fatalf("errore caricamento YAML esterno (%s): %v", yamlPath, err)
		}
	} else {
		monsters, envs, crs, types, err = loadMonstersFromBytes(embeddedMonstersYAML)
		if err != nil {
			log.Fatalf("errore caricamento YAML embedded: %v", err)
		}
	}

	items, _, _, _, err = loadItemsFromBytes(embeddedItemsYAML)
	if err != nil {
		log.Fatalf("errore caricamento item YAML embedded: %v", err)
	}
	spells, _, _, _, err = loadSpellsFromBytes(embeddedSpellsYAML)
	if err != nil {
		log.Fatalf("errore caricamento spell YAML embedded: %v", err)
	}

	ui := newUI(monsters, items, spells, envs, crs, types, encountersPath, dicePath)
	if err := ui.run(); err != nil {
		log.Fatal(err)
	}
	if err := ui.saveEncounters(); err != nil {
		log.Printf("errore salvataggio encounters (%s): %v", encountersPath, err)
	}
	if err := ui.saveDiceResults(); err != nil {
		log.Printf("errore salvataggio dice (%s): %v", ui.dicePath, err)
	}
	if err := ui.saveFilterStates(); err != nil {
		log.Printf("errore salvataggio filtri: %v", err)
	}
}

func newUI(monsters, items, spells []Monster, envs, crs, types []string, encountersPath string, dicePath string) *UI {
	setTheme()

	ui := &UI{
		app:               tview.NewApplication(),
		monsters:          monsters,
		items:             items,
		spells:            spells,
		browseMode:        BrowseMonsters,
		sourceFilters:     map[string]struct{}{},
		envOptions:        append([]string{"All"}, envs...),
		sourceOptions:     []string{"All"},
		crOptions:         append([]string{"All"}, crs...),
		typeOptions:       append([]string{"All"}, types...),
		filtered:          make([]int, 0, len(monsters)),
		encounterSerial:   map[int]int{},
		encounterItems:    make([]EncounterEntry, 0, 16),
		encountersPath:    encountersPath,
		dicePath:          dicePath,
		modeFilters:       map[BrowseMode]PersistedFilterMode{},
		activeBottomPanel: "description",
	}

	ui.nameInput = tview.NewInputField().
		SetLabel(" Name ").
		SetFieldWidth(26)
	ui.nameInput.SetLabelColor(tcell.ColorGold)
	ui.nameInput.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.nameInput.SetFieldTextColor(tcell.ColorWhite)
	ui.nameInput.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite))
	ui.nameInput.SetChangedFunc(func(text string) {
		ui.nameFilter = strings.TrimSpace(text)
		ui.applyFilters()
	})
	ui.nameInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter, tcell.KeyEscape:
			ui.app.SetFocus(ui.list)
		}
	})

	ui.envDrop = tview.NewDropDown().
		SetLabel(" Env ")
	ui.envDrop.SetOptions(ui.envOptions, func(option string, _ int) {
		if option == "All" {
			ui.envFilter = ""
		} else {
			ui.envFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	ui.envDrop.SetLabelColor(tcell.ColorGold)
	ui.envDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.envDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.envDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.envDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.sourceDrop = tview.NewDropDown().
		SetLabel(" Source ")
	ui.sourceDrop.SetLabelColor(tcell.ColorGold)
	ui.sourceDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.sourceDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.sourceDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.sourceDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.crDrop = tview.NewDropDown().
		SetLabel(" CR ").
		SetOptions(ui.crOptions, func(option string, _ int) {
			if option == "All" {
				ui.crFilter = ""
			} else {
				ui.crFilter = option
			}
			ui.applyFilters()
			ui.maybeReturnFocusToListFromFilter()
		})
	ui.crDrop.SetLabelColor(tcell.ColorGold)
	ui.crDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.crDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.crDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.crDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.typeDrop = tview.NewDropDown().
		SetLabel(" Type ").
		SetOptions(ui.typeOptions, func(option string, _ int) {
			if option == "All" {
				ui.typeFilter = ""
			} else {
				ui.typeFilter = option
			}
			ui.applyFilters()
			ui.maybeReturnFocusToListFromFilter()
		})
	ui.typeDrop.SetLabelColor(tcell.ColorGold)
	ui.typeDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.typeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.typeDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)
	ui.typeDrop.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.app.SetFocus(ui.list)
		}
	})

	ui.list = tview.NewList()
	ui.list.SetBorder(false)
	ui.list.SetMainTextColor(tcell.ColorWhite)
	ui.list.SetSecondaryTextColor(tcell.ColorLightGray)
	ui.list.SetSelectedTextColor(tcell.ColorBlack)
	ui.list.SetSelectedBackgroundColor(tcell.ColorGold)
	ui.list.ShowSecondaryText(false)
	ui.list.SetChangedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByListIndex(index)
	})
	ui.list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByListIndex(index)
	})

	ui.encounter = tview.NewList()
	ui.encounter.SetBorder(true)
	ui.encounter.SetTitle(" [1]-Encounters ")
	ui.encounter.SetTitleColor(tcell.ColorGold)
	ui.encounter.SetBorderColor(tcell.ColorGold)
	ui.encounter.SetMainTextColor(tcell.ColorWhite)
	ui.encounter.SetSelectedTextColor(tcell.ColorBlack)
	ui.encounter.SetSelectedBackgroundColor(tcell.ColorGold)
	ui.encounter.ShowSecondaryText(false)
	ui.encounter.SetChangedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByEncounterIndex(index)
	})
	ui.encounter.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByEncounterIndex(index)
	})
	ui.encounter.AddItem("Nessun mostro nell'encounter", "", 0, nil)

	ui.dice = tview.NewList()
	ui.dice.SetBorder(true)
	ui.dice.SetTitle(" [0]-Dice ")
	ui.dice.SetTitleColor(tcell.ColorGold)
	ui.dice.SetBorderColor(tcell.ColorGold)
	ui.dice.SetUseStyleTags(true, false)
	ui.dice.SetMainTextColor(tcell.ColorWhite)
	ui.dice.SetSelectedTextColor(tcell.ColorWhite)
	ui.dice.SetSelectedBackgroundColor(tcell.ColorDefault)
	ui.dice.ShowSecondaryText(false)
	ui.dice.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if ui.diceRender {
			return
		}
		ui.renderDiceList()
	})

	ui.detailMeta = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	ui.detailMeta.SetBorder(true)
	ui.detailMeta.SetTitle(" Details ")
	ui.detailMeta.SetTitleColor(tcell.ColorGold)
	ui.detailMeta.SetBorderColor(tcell.ColorGold)
	ui.detailMeta.SetTextColor(tcell.ColorWhite)
	ui.detailMeta.SetWrap(true)

	ui.detailTreasure = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	ui.detailTreasure.SetBorder(true)
	ui.detailTreasure.SetTitle(" Treasure ")
	ui.detailTreasure.SetTitleColor(tcell.ColorGold)
	ui.detailTreasure.SetBorderColor(tcell.ColorGold)
	ui.detailTreasure.SetTextColor(tcell.ColorWhite)
	ui.detailTreasure.SetWrap(true)
	ui.treasureText = "Nessun tesoro generato."
	ui.detailTreasure.SetText(ui.treasureText)

	ui.detailRaw = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(false)
	ui.detailRaw.SetBorder(true)
	ui.detailRaw.SetTitle(" [3]-Description ")
	ui.detailRaw.SetTitleColor(tcell.ColorGold)
	ui.detailRaw.SetBorderColor(tcell.ColorGold)
	ui.detailRaw.SetTextColor(tcell.ColorWhite)

	ui.detailBottom = tview.NewPages().
		AddPage("description", ui.detailRaw, true, true).
		AddPage("treasure", ui.detailTreasure, true, false)

	ui.detailPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.detailMeta, 8, 0, false).
		AddItem(ui.detailBottom, 0, 1, false)

	ui.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText)
	ui.status.SetBackgroundColor(tcell.ColorBlack)

	filterRowSingle := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.nameInput, 0, 4, true).
		AddItem(ui.envDrop, 0, 2, false).
		AddItem(ui.sourceDrop, 0, 2, false).
		AddItem(ui.crDrop, 0, 1, false).
		AddItem(ui.typeDrop, 0, 2, false)

	filterRowTop := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.nameInput, 0, 3, true).
		AddItem(ui.envDrop, 0, 1, false).
		AddItem(ui.sourceDrop, 0, 1, false)

	filterRowBottom := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.crDrop, 0, 1, false).
		AddItem(ui.typeDrop, 0, 2, false)

	filterRow := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(filterRowTop, 1, 0, false).
		AddItem(filterRowBottom, 1, 0, false)

	ui.filterHost = tview.NewPages().
		AddPage("single", filterRowSingle, true, false).
		AddPage("double", filterRow, true, true)

	ui.monstersPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.filterHost, 2, 0, true).
		AddItem(ui.list, 0, 1, false)
	ui.monstersPanel.SetBorder(true)
	ui.monstersPanel.SetTitle(" [2]-Monsters ")
	ui.monstersPanel.SetTitleColor(tcell.ColorGold)
	ui.monstersPanel.SetBorderColor(tcell.ColorGold)

	ui.leftPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.dice, 7, 0, false).
		AddItem(ui.encounter, 8, 0, false).
		AddItem(ui.monstersPanel, 0, 1, true)

	ui.mainRow = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.leftPanel, 0, 1, false).
		AddItem(ui.detailPanel, 0, 1, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.mainRow, 0, 1, false).
		AddItem(ui.status, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", root, true, true)
	ui.app.SetRoot(ui.pages, true)
	ui.focusOrder = []tview.Primitive{ui.dice, ui.encounter, ui.nameInput, ui.envDrop, ui.sourceDrop, ui.crDrop, ui.typeDrop, ui.list, ui.detailMeta, ui.detailTreasure, ui.detailRaw}
	ui.app.SetFocus(ui.list)
	ui.modeFilters[BrowseMonsters] = PersistedFilterMode{}
	ui.modeFilters[BrowseItems] = PersistedFilterMode{}
	ui.modeFilters[BrowseSpells] = PersistedFilterMode{}
	if err := ui.loadFilterStates(); err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] errore load filtri[-:-] %v  %s", err, helpText))
	}
	ui.applyModeFilters(ui.browseMode)
	ui.updateBrowsePanelTitle()
	ui.updateFilterLayout(0)

	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		focus := ui.app.GetFocus()
		_, focusIsInputField := focus.(*tview.InputField)

		if ui.addCustomVisible && event.Key() == tcell.KeyTab {
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		}

		if ui.helpVisible {
			if event.Key() == tcell.KeyEscape ||
				(event.Key() == tcell.KeyRune && (event.Rune() == '?' || event.Rune() == 'q')) {
				ui.closeHelpOverlay()
				return nil
			}
			// Let the help TextView handle scrolling keys (j/k, arrows, PgUp/PgDn).
			return event
		}

		if ui.itemTreasureVisible || ui.spellTreasureVisible {
			if event.Key() == tcell.KeyEscape {
				if ui.itemTreasureVisible {
					ui.closeItemTreasureModal()
				}
				if ui.spellTreasureVisible {
					ui.closeSpellTreasureModal()
				}
				return nil
			}
			// While modal is open, do not process global shortcuts (1/2/3/q/...).
			return event
		}

		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == '?':
			ui.openHelpOverlay(focus)
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'a':
			ui.openDiceRollInput()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyEnter:
			ui.rerollSelectedDiceResult()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'e':
			ui.openDiceReRollInput()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			ui.deleteSelectedDiceResult()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'c':
			ui.clearDiceResults()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 's':
			ui.openDiceSaveAsInput()
			return nil
		case focus == ui.dice && event.Key() == tcell.KeyRune && event.Rune() == 'l':
			ui.openDiceLoadInput()
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'f':
			ui.toggleFullscreenForFocus(focus)
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'q':
			ui.app.Stop()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == '/':
			if focusIsInputField {
				return event
			}
			if focus == ui.list {
				ui.openRawSearch(ui.list)
				return nil
			}
			if focus == ui.encounter {
				ui.openRawSearch(ui.encounter)
				return nil
			}
			if focus == ui.detailRaw {
				ui.openRawSearch(ui.detailRaw)
				return nil
			}
			ui.app.SetFocus(ui.nameInput)
			return nil
		case event.Key() == tcell.KeyTab:
			ui.focusNext()
			return nil
		case event.Key() == tcell.KeyBacktab:
			ui.focusPrev()
			return nil
		case focus == ui.sourceDrop && (event.Key() == tcell.KeyEnter || (event.Key() == tcell.KeyRune && event.Rune() == ' ')):
			ui.openSourceMultiSelectModal()
			return nil
		case (focus == ui.envDrop || focus == ui.crDrop || focus == ui.typeDrop) && event.Key() == tcell.KeyEnter:
			ui.app.SetFocus(ui.list)
			return nil
		case event.Key() == tcell.KeyEscape &&
			(focus == ui.list || focus == ui.nameInput || focus == ui.envDrop || focus == ui.sourceDrop || focus == ui.crDrop || focus == ui.typeDrop):
			ui.app.SetFocus(ui.list)
			return nil
		case focus == ui.nameInput && event.Key() == tcell.KeyEscape:
			ui.app.SetFocus(ui.list)
			return nil
		case focus == ui.list && event.Key() == tcell.KeyPgUp:
			if len(ui.filtered) > 0 {
				ui.scrollDetailByPage(-1)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyPgDn:
			if len(ui.filtered) > 0 {
				ui.scrollDetailByPage(1)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'a':
			ui.addSelectedMonsterToEncounter()
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'n':
			ui.app.SetFocus(ui.nameInput)
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'e':
			if ui.browseMode == BrowseMonsters || ui.browseMode == BrowseItems || ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.envDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'c':
			if ui.browseMode == BrowseMonsters {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.typeDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 't':
			if ui.browseMode == BrowseMonsters || ui.browseMode == BrowseItems {
				ui.app.SetFocus(ui.typeDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 's':
			if ui.browseMode == BrowseMonsters {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			if ui.browseMode == BrowseItems {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.sourceDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'r':
			if ui.browseMode == BrowseItems {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case (focus == ui.envDrop || focus == ui.sourceDrop || focus == ui.crDrop || focus == ui.typeDrop) &&
			event.Key() == tcell.KeyRune && event.Rune() == 'r':
			if ui.browseMode == BrowseItems {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case (focus == ui.envDrop || focus == ui.sourceDrop || focus == ui.crDrop || focus == ui.typeDrop) &&
			event.Key() == tcell.KeyRune && event.Rune() == 'c':
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.typeDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'm':
			if ui.browseMode == BrowseMonsters {
				ui.openTreasureByCRInput()
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'l':
			if ui.browseMode == BrowseMonsters {
				ui.openLairTreasureByCRInput()
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.app.SetFocus(ui.crDrop)
				return nil
			}
			return event
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'g':
			if ui.browseMode == BrowseItems {
				ui.openItemTreasureInput()
				return nil
			}
			if ui.browseMode == BrowseSpells {
				ui.openSpellTreasureInput()
				return nil
			}
			return event
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'a':
			ui.openAddCustomEncounterForm()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyLeft:
			ui.openEncounterHPInput(-1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRight:
			ui.openEncounterHPInput(1)
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			ui.deleteSelectedEncounterEntry()
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'd':
			if focus == ui.list || focus == ui.detailMeta || focus == ui.detailTreasure || focus == ui.detailRaw {
				ui.toggleDetailsTreasureFocus()
				return nil
			}
			return event
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 's':
			ui.openEncounterSaveAsInput()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'l':
			ui.openEncounterLoadInput()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'i':
			ui.rollEncounterInitiative()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'I':
			ui.rollAllEncounterInitiative()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'S':
			ui.sortEncounterByInitiative()
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'S':
			if ui.browseMode == BrowseItems {
				ui.openTreasureSaveAsInput()
				return nil
			}
			return event
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == '*':
			ui.toggleEncounterTurnMode()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'n':
			ui.nextEncounterTurn()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == 'p':
			ui.prevEncounterTurn()
			return nil
		case focus == ui.encounter && event.Key() == tcell.KeyRune && event.Rune() == ' ':
			ui.toggleEncounterHPMode()
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'u':
			if focus == ui.dice {
				ui.undoDiceCommand()
			} else {
				ui.undoEncounterCommand()
			}
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == 'r':
			if focus == ui.dice {
				ui.redoDiceCommand()
			} else {
				ui.redoEncounterCommand()
			}
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '1':
			ui.app.SetFocus(ui.encounter)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '2':
			ui.app.SetFocus(ui.list)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '3':
			ui.app.SetFocus(ui.detailRaw)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '0':
			ui.app.SetFocus(ui.dice)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == '[':
			ui.cycleBrowseMode(-1)
			return nil
		case !focusIsInputField && event.Key() == tcell.KeyRune && event.Rune() == ']':
			ui.cycleBrowseMode(1)
			return nil
		case focus != ui.nameInput && event.Key() == tcell.KeyRune && event.Rune() == 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case focus != ui.nameInput && event.Key() == tcell.KeyRune && event.Rune() == 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		default:
			return event
		}
	})
	ui.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		w, _ := screen.Size()
		ui.updateFilterLayout(w)
		return false
	})

	ui.applyFilters()
	if err := ui.loadEncounters(); err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] errore load encounters[-:-] %v  %s", err, helpText))
	}
	if err := ui.loadDiceResults(); err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] errore load dice[-:-] %v  %s", err, helpText))
	}
	ui.renderEncounterList()
	return ui
}

func (ui *UI) openHelpOverlay(focus tview.Primitive) {
	ui.helpReturnFocus = focus
	ui.helpVisible = true

	text := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	text.SetBorder(true)
	text.SetBorderColor(tcell.ColorGold)
	text.SetTitleColor(tcell.ColorGold)
	text.SetTitle(fmt.Sprintf(" Help - %s ", ui.panelNameForFocus(focus)))
	text.SetText(ui.helpForFocus(focus))

	helpBody := ui.helpForFocus(focus) + "\n[gray]Scroll: j/k, frecce, PgUp/PgDn[-]"
	text.SetText(helpBody)

	// Bigger modal so panel-specific shortcuts are not clipped on common terminal sizes.
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(text, 22, 0, true).
			AddItem(nil, 0, 1, false), 92, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("help-overlay", modal, true, true)
	ui.app.SetFocus(text)
}

func (ui *UI) closeHelpOverlay() {
	ui.pages.RemovePage("help-overlay")
	ui.helpVisible = false
	if ui.helpReturnFocus != nil {
		ui.app.SetFocus(ui.helpReturnFocus)
	} else {
		ui.app.SetFocus(ui.list)
	}
}

func (ui *UI) panelNameForFocus(focus tview.Primitive) string {
	switch focus {
	case ui.dice:
		return "Dice"
	case ui.encounter:
		return "Encounters"
	case ui.list:
		switch ui.browseMode {
		case BrowseItems:
			return "Items"
		case BrowseSpells:
			return "Spells"
		default:
			return "Monsters"
		}
	case ui.detailRaw:
		return "Description"
	case ui.detailMeta:
		return "Details"
	case ui.detailTreasure:
		return "Treasure"
	case ui.nameInput:
		return "Name Filter"
	case ui.envDrop:
		return "Env Filter"
	case ui.sourceDrop:
		return "Source Filter"
	case ui.crDrop:
		return "CR Filter"
	case ui.typeDrop:
		return "Type Filter"
	default:
		return "Panel"
	}
}

func (ui *UI) helpForFocus(focus tview.Primitive) string {
	header := "[black:gold]Global[-:-]\n" +
		"  ? : apri/chiudi questo help\n" +
		"  Esc : chiudi help\n" +
		"  q : esci programma\n" +
		"  f : fullscreen on/off pannello corrente\n" +
		"  Tab / Shift+Tab : cambia focus\n" +
		"  0 / 1 / 2 / 3 : vai a Dice / Encounters / Catalogo / Description\n" +
		"  [ / ] : ciclo Monsters / Items / Spells\n\n"

	switch focus {
	case ui.dice:
		return header +
			"[black:gold]Dice[-:-]\n" +
			"  a : tira espressione dadi (es. 2d6+d20+1)\n" +
			"  Enter : rilancia riga selezionata\n" +
			"  e : modifica + rilancia riga selezionata\n" +
			"  d : elimina riga selezionata\n" +
			"  c : cancella tutte le righe\n" +
			"  s : salva risultati dadi (save as)\n" +
			"  l : carica risultati dadi (load)\n" +
			"  f : fullscreen on/off del pannello Dice\n" +
			"\n" +
			"[black:gold]Esempi[-:-]\n" +
			"  2d6+d20+1\n" +
			"  d20v+5   (v = scegli il tiro piu alto su 2)\n" +
			"  d20s+1   (s = scegli il tiro piu basso su 2)\n" +
			"  d2,d3,d4\n" +
			"  4d10+6d6+5\n" +
			"  1d6 x2\n" +
			"  1d6-1\n" +
			"  1d20+5 > 2\n" +
			"  2d6+d20-1 > 15\n" +
			"  1d20+5 > 10 x3\n"
	case ui.encounter:
		return header +
			"[black:gold]Encounters[-:-]\n" +
			"  j / k (o frecce) : seleziona entry\n" +
			"  / : cerca nella Description del mostro selezionato\n" +
			"  a : aggiungi entry custom\n" +
			"  d : elimina entry selezionata\n" +
			"  s : salva encounter su file (save as)\n" +
			"  l : carica encounter da file (load)\n" +
			"  i : tira iniziativa entry selezionata\n" +
			"  I : tira iniziativa per tutte le entry\n" +
			"  S : ordina entry per tiro iniziativa\n" +
			"  * : attiva/disattiva turn mode\n" +
			"  n / p : turno successivo / precedente\n" +
			"  u : undo ultima operazione encounter\n" +
			"  r : redo operazione encounter annullata\n" +
			"  spazio : switch HP average/formula (roll)\n" +
			"  freccia sinistra : sottrai HP\n" +
			"  freccia destra : aggiungi HP\n"
	case ui.list:
		if ui.browseMode == BrowseMonsters {
			return header +
				"[black:gold]Monsters[-:-]\n" +
				"  j / k (o frecce) : naviga mostri\n" +
				"  / : cerca nella Description del mostro selezionato\n" +
				"  a : aggiungi mostro a Encounters\n" +
				"  m : genera tesoro da CR (regole 5e)\n" +
				"  l : genera lair treasure da CR (regole 5e)\n" +
				"  n / e / s / c / t : focus su Name / Env / Source(multi) / CR / Type\n" +
				"  [ / ] : cambia panel Monsters/Items/Spells\n" +
				"  PgUp / PgDn : scroll del pannello Description\n"
		}
		if ui.browseMode == BrowseItems {
			return header +
				"[black:gold]Items[-:-]\n" +
				"  j / k (o frecce) : naviga lista\n" +
				"  / : cerca nella Description della voce selezionata\n" +
				"  g : genera treasure items (tipo + quantita)\n" +
				"  S : salva Treasure su file\n" +
				"  n / s / r / t : focus su Name / Source(multi) / Rarity / Type\n" +
				"  [ / ] : cambia panel Monsters/Items/Spells\n" +
				"  PgUp / PgDn : scroll del pannello Description\n"
		}
		return header +
			"[black:gold]Spells[-:-]\n" +
			"  j / k (o frecce) : naviga lista\n" +
			"  / : cerca nella Description della voce selezionata\n" +
			"  g : genera spells (level + quantita)\n" +
			"  n / s / l / c : focus su Name / Source(multi) / Level / School\n" +
			"  [ / ] : cambia panel Monsters/Items/Spells\n" +
			"  PgUp / PgDn : scroll del pannello Description\n"
	case ui.detailRaw:
		return header +
			"[black:gold]Description[-:-]\n" +
			"  / : cerca testo nella Description corrente\n" +
			"  j / k (o frecce) : scroll contenuto\n"
	case ui.detailMeta, ui.detailTreasure:
		return header +
			"[black:gold]Details/Treasure[-:-]\n" +
			"  d : switch focus tra Details e Treasure\n" +
			"  j / k (o frecce) : scroll contenuto\n"
	case ui.nameInput:
		return header +
			"[black:gold]Name Filter[-:-]\n" +
			"  scrivi testo : filtro per nome\n" +
			"  Enter / Esc : torna a Monsters\n"
	case ui.envDrop, ui.sourceDrop, ui.crDrop, ui.typeDrop:
		return header +
			"[black:gold]Filter Dropdown[-:-]\n" +
			"  frecce / Invio : cambia valore filtro\n"
	default:
		return header + "[black:gold]Panel[-:-]\n  Nessuna scorciatoia specifica.\n"
	}
}

func (ui *UI) focusNext() {
	current := ui.app.GetFocus()
	for i, p := range ui.focusOrder {
		if p == current {
			ui.app.SetFocus(ui.focusOrder[(i+1)%len(ui.focusOrder)])
			return
		}
	}
	ui.app.SetFocus(ui.list)
}

func (ui *UI) focusPrev() {
	current := ui.app.GetFocus()
	for i, p := range ui.focusOrder {
		if p == current {
			prev := i - 1
			if prev < 0 {
				prev = len(ui.focusOrder) - 1
			}
			ui.app.SetFocus(ui.focusOrder[prev])
			return
		}
	}
	ui.app.SetFocus(ui.list)
}

func (ui *UI) toggleDetailsTreasureFocus() {
	if ui.detailBottom == nil {
		return
	}
	if ui.activeBottomPanel == "treasure" {
		ui.activeBottomPanel = "description"
		ui.detailBottom.SwitchToPage("description")
		ui.app.SetFocus(ui.detailRaw)
		return
	}
	ui.activeBottomPanel = "treasure"
	ui.detailBottom.SwitchToPage("treasure")
	ui.app.SetFocus(ui.detailTreasure)
}

func (ui *UI) scrollDetailByPage(direction int) {
	if direction == 0 {
		return
	}

	_, _, _, height := ui.detailRaw.GetInnerRect()
	if height <= 0 {
		height = 10
	}

	row, _ := ui.detailRaw.GetScrollOffset()
	step := height - 1
	if step < 1 {
		step = 1
	}

	nextRow := row + (step * direction)
	if nextRow < 0 {
		nextRow = 0
	}
	ui.detailRaw.ScrollTo(nextRow, 0)
}

func (ui *UI) openTreasureByCRInput() {
	input := tview.NewInputField().
		SetLabel("CR: ").
		SetFieldWidth(16)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Generate Treasure (5e Individual) ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	if cr := ui.currentMonsterCR(); cr != "" {
		input.SetText(cr)
	}

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 44, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("treasure-input")
		ui.app.SetFocus(ui.list)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		crText := strings.TrimSpace(input.GetText())
		outcome, err := generateIndividualTreasure(crText, rand.Intn)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] CR non valido[-:-] \"%s\"  %s", crText, helpText))
			return
		}
		ui.renderTreasureOutcome(crText, outcome)
		ui.status.SetText(fmt.Sprintf(" [black:gold]tesoro[-:-] generato per CR %s  %s", crText, helpText))
	})

	ui.pages.AddPage("treasure-input", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openLairTreasureByCRInput() {
	input := tview.NewInputField().
		SetLabel("CR: ").
		SetFieldWidth(16)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Generate Lair Treasure (5e Hoard) ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	if cr := ui.currentMonsterCR(); cr != "" {
		input.SetText(cr)
	}

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 46, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("lair-treasure-input")
		ui.app.SetFocus(ui.list)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		crText := strings.TrimSpace(input.GetText())
		outcome, err := generateLairTreasure(crText, rand.Intn)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] CR non valido[-:-] \"%s\"  %s", crText, helpText))
			return
		}
		ui.renderTreasureOutcome(crText, outcome)
		ui.status.SetText(fmt.Sprintf(" [black:gold]lair treasure[-:-] generato per CR %s  %s", crText, helpText))
	})

	ui.pages.AddPage("lair-treasure-input", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) currentMonsterCR() string {
	if ui.browseMode != BrowseMonsters || len(ui.filtered) == 0 {
		return ""
	}
	cur := ui.list.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filtered) {
		return ""
	}
	idx := ui.filtered[cur]
	if idx < 0 || idx >= len(ui.monsters) {
		return ""
	}
	return strings.TrimSpace(ui.monsters[idx].CR)
}

func (ui *UI) openItemTreasureInput() {
	typeOptions := []string{"random", "potion", "scroll", "staff", "wand", "rod", "ring", "weapon", "armor", "wondrous"}
	selectedTypes := map[string]struct{}{"random": {}}

	typeList := tview.NewList()
	typeList.SetBorder(true)
	typeList.SetTitle(" Type (Space=toggle, Enter=Qty) ")
	typeList.SetBorderColor(tcell.ColorGold)
	typeList.SetTitleColor(tcell.ColorGold)
	typeList.SetMainTextColor(tcell.ColorWhite)
	typeList.SetSelectedTextColor(tcell.ColorBlack)
	typeList.SetSelectedBackgroundColor(tcell.ColorGold)
	typeList.ShowSecondaryText(false)

	renderTypes := func() {
		current := typeList.GetCurrentItem()
		typeList.Clear()
		for _, opt := range typeOptions {
			mark := "[ ]"
			if _, ok := selectedTypes[opt]; ok {
				mark = "[x]"
			}
			typeList.AddItem(fmt.Sprintf("%s %s", mark, opt), "", 0, nil)
		}
		if current < 0 {
			current = 0
		}
		if current >= len(typeOptions) {
			current = len(typeOptions) - 1
		}
		typeList.SetCurrentItem(current)
	}

	toggleAt := func(idx int) {
		if idx < 0 || idx >= len(typeOptions) {
			return
		}
		opt := typeOptions[idx]
		if opt == "random" {
			selectedTypes = map[string]struct{}{"random": {}}
			return
		}
		delete(selectedTypes, "random")
		if _, ok := selectedTypes[opt]; ok {
			delete(selectedTypes, opt)
		} else {
			selectedTypes[opt] = struct{}{}
		}
		if len(selectedTypes) == 0 {
			selectedTypes["random"] = struct{}{}
		}
	}

	closeModal := func() {
		ui.closeItemTreasureModal()
	}
	qtyInput := tview.NewInputField().SetLabel(" Qty ").SetFieldWidth(8).SetText("1")
	qtyInput.SetLabelColor(tcell.ColorGold)
	qtyInput.SetFieldBackgroundColor(tcell.ColorWhite)
	qtyInput.SetFieldTextColor(tcell.ColorBlack)
	qtyInput.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	qtyInput.SetBorder(true)
	qtyInput.SetBorderColor(tcell.ColorGold)

	runGenerate := func() {
		count, err := strconv.Atoi(strings.TrimSpace(qtyInput.GetText()))
		if err != nil || count <= 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] quantita non valida[-:-] \"%s\"  %s", qtyInput.GetText(), helpText))
			return
		}
		kinds := keysSorted(selectedTypes)
		items, err := ui.generateItemTreasureByKinds(kinds, count)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] %v[-:-]  %s", err, helpText))
			return
		}
		ui.renderGeneratedItemTreasure(strings.Join(kinds, ","), items)
		ui.status.SetText(fmt.Sprintf(" [black:gold]item treasure[-:-] generati %d item (%s)  %s", len(items), strings.Join(kinds, ","), helpText))
		closeModal()
	}

	typeList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == ' ':
			idx := typeList.GetCurrentItem()
			toggleAt(idx)
			renderTypes()
			return nil
		case event.Key() == tcell.KeyEnter:
			ui.app.SetFocus(qtyInput)
			return nil
		case event.Key() == tcell.KeyEscape:
			closeModal()
			return nil
		default:
			return event
		}
	})
	qtyInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			runGenerate()
		case tcell.KeyEscape:
			closeModal()
		}
	})

	renderTypes()

	modalBody := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(typeList, 11, 0, true).
		AddItem(qtyInput, 3, 0, false)
	modalBody.SetBorder(true)
	modalBody.SetTitle(" Generate Item Treasure ")
	modalBody.SetTitleColor(tcell.ColorGold)
	modalBody.SetBorderColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modalBody, 16, 0, true).
			AddItem(nil, 0, 1, false), 62, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("items-treasure-input", modal, true, true)
	ui.itemTreasureVisible = true
	ui.app.SetFocus(typeList)
}

func (ui *UI) openSpellTreasureInput() {
	levelOptions := []string{"random", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	schoolOptions := []string{"random"}
	schoolSet := map[string]struct{}{}
	for _, sp := range ui.spells {
		if s := strings.TrimSpace(sp.Type); s != "" {
			schoolSet[s] = struct{}{}
		}
	}
	schoolOptions = append(schoolOptions, keysSorted(schoolSet)...)

	level := "random"
	school := "random"
	qty := "1"

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Generate Spell Treasure ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	form.SetFieldBackgroundColor(tcell.ColorWhite)
	form.SetFieldTextColor(tcell.ColorBlack)
	form.SetLabelColor(tcell.ColorGold)
	form.AddDropDown("Level", levelOptions, 0, func(option string, _ int) { level = option })
	form.AddDropDown("School", schoolOptions, 0, func(option string, _ int) { school = option })
	form.AddInputField("Qty", qty, 8, nil, func(text string) { qty = text })

	closeModal := func() {
		ui.closeSpellTreasureModal()
	}
	runGenerate := func() {
		count, err := strconv.Atoi(strings.TrimSpace(qty))
		if err != nil || count <= 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] quantita non valida[-:-] \"%s\"  %s", qty, helpText))
			return
		}
		filter := SpellTreasureFilter{
			Level:  level,
			School: school,
		}
		spells, err := ui.generateSpellTreasure(filter, count)
		if err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] %v[-:-]  %s", err, helpText))
			return
		}
		ui.renderGeneratedSpellTreasure(filter, spells)
		ui.status.SetText(fmt.Sprintf(" [black:gold]spell treasure[-:-] generate %d spells (level=%s school=%s)  %s", len(spells), filter.Level, filter.School, helpText))
		closeModal()
	}
	form.AddButton("Generate", runGenerate)
	form.AddButton("Cancel", closeModal)
	form.SetCancelFunc(closeModal)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			formIdx, btnIdx := form.GetFocusedItemIndex()
			if formIdx == 2 && btnIdx < 0 {
				runGenerate()
				return nil
			}
		}
		return event
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 13, 0, true).
			AddItem(nil, 0, 1, false), 64, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("spells-treasure-input", modal, true, true)
	ui.spellTreasureVisible = true
	ui.app.SetFocus(form)
}

func (ui *UI) openTreasureSaveAsInput() {
	content := strings.TrimSpace(ui.treasureText)
	if content == "" || strings.EqualFold(content, "Nessun tesoro generato.") {
		ui.status.SetText(fmt.Sprintf(" [white:red] nessun Treasure da salvare[-:-]  %s", helpText))
		return
	}
	defaultName := fmt.Sprintf("tesoro-%s.yaml", newShortUUID())
	input := tview.NewInputField().
		SetLabel("File: ").
		SetFieldWidth(60).
		SetText(defaultName)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBorder(true)
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetTitle(" Save Treasure As ")

	closeModal := func() {
		ui.pages.RemovePage("treasure-saveas")
		ui.app.SetFocus(ui.list)
	}
	trySave := func(path string, overwrite bool) {
		if err := ui.saveTreasureToPath(path, overwrite); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] errore save treasure[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold]treasure salvato[-:-] %s  %s", path, helpText))
		closeModal()
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			closeModal()
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] nome file non valido[-:-]  %s", helpText))
			return
		}
		if fileExists(path) {
			ui.openTreasureOverwriteConfirm(path, func(confirmed bool) {
				if !confirmed {
					ui.status.SetText(fmt.Sprintf(" [black:gold]save treasure[-:-] annullato (file esistente)  %s", helpText))
					return
				}
				trySave(path, true)
			})
			return
		}
		trySave(path, false)
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 76, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("treasure-saveas", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openTreasureOverwriteConfirm(path string, done func(bool)) {
	msg := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)
	msg.SetBorder(true)
	msg.SetBorderColor(tcell.ColorGold)
	msg.SetTitleColor(tcell.ColorGold)
	msg.SetTitle(" Overwrite Warning ")
	msg.SetText(fmt.Sprintf("Il file esiste gia:\n[white]%s[-]\n\nSovrascrivere? [black:gold]y[-:-]/[black:gold]n[-:-]", path))

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(msg, 8, 0, true).
			AddItem(nil, 0, 1, false), 76, 0, true).
		AddItem(nil, 0, 1, false)

	msg.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && (event.Rune() == 'n' || event.Rune() == 'N')) {
			ui.pages.RemovePage("treasure-overwrite")
			done(false)
			return nil
		}
		if event.Key() == tcell.KeyRune && (event.Rune() == 'y' || event.Rune() == 'Y') {
			ui.pages.RemovePage("treasure-overwrite")
			done(true)
			return nil
		}
		return event
	})

	ui.pages.AddPage("treasure-overwrite", modal, true, true)
	ui.app.SetFocus(msg)
}

func (ui *UI) saveTreasureToPath(path string, overwrite bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	if !overwrite && fileExists(path) {
		return errors.New("file already exists")
	}
	content := strings.TrimSpace(ui.treasureText)
	if content == "" {
		return errors.New("empty treasure content")
	}
	return os.WriteFile(path, []byte(content+"\n"), 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func newShortUUID() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	hexv := hex.EncodeToString(b[:])
	if len(hexv) < 16 {
		return hexv
	}
	return hexv[:16]
}

type SpellTreasureFilter struct {
	Level  string
	School string
}

func (ui *UI) generateSpellTreasure(filter SpellTreasureFilter, count int) ([]Monster, error) {
	if count < 1 {
		return nil, errors.New("quantita deve essere >= 1")
	}
	candidates := filterSpellsByFilter(ui.spells, filter)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("nessuna spell trovata per level=%q school=%q", filter.Level, filter.School)
	}
	out := make([]Monster, 0, count)
	for i := 0; i < count; i++ {
		idx := rand.Intn(len(candidates))
		out = append(out, candidates[idx])
	}
	return out, nil
}

func filterSpellsByFilter(spells []Monster, filter SpellTreasureFilter) []Monster {
	level := strings.TrimSpace(strings.ToLower(filter.Level))
	school := strings.TrimSpace(strings.ToLower(filter.School))
	out := make([]Monster, 0, len(spells))
	for _, sp := range spells {
		if level != "" && level != "random" && !strings.EqualFold(strings.TrimSpace(sp.CR), level) {
			continue
		}
		if school != "" && school != "random" && !strings.EqualFold(strings.TrimSpace(sp.Type), school) {
			continue
		}
		out = append(out, sp)
	}
	return out
}

func filterSpellsByLevel(spells []Monster, level string) []Monster {
	return filterSpellsByFilter(spells, SpellTreasureFilter{Level: level})
}

func (ui *UI) renderGeneratedSpellTreasure(filter SpellTreasureFilter, spells []Monster) {
	meta := &strings.Builder{}
	fmt.Fprintf(meta, "[yellow]Spell Treasure[-]\n")
	fmt.Fprintf(meta, "[white]Level:[-] %s\n", blankIfEmpty(filter.Level, "random"))
	fmt.Fprintf(meta, "[white]School:[-] %s\n", blankIfEmpty(filter.School, "random"))
	fmt.Fprintf(meta, "[white]Count:[-] %d\n", len(spells))
	ui.detailMeta.SetText(meta.String())
	ui.detailMeta.ScrollToBeginning()

	lines := make([]string, 0, len(spells))
	for i, sp := range spells {
		lines = append(lines, fmt.Sprintf("%d. %s [%s] (Level %s, %s)", i+1, sp.Name, sp.Source, sp.CR, sp.Type))
	}
	ui.treasureText = fmt.Sprintf("[yellow]Generated Spells[-]\n[white]Level:[-] %s  [white]School:[-] %s  [white]Qty:[-] %d\n\n%s", blankIfEmpty(filter.Level, "random"), blankIfEmpty(filter.School, "random"), len(spells), strings.Join(lines, "\n"))
	ui.detailTreasure.SetText(ui.treasureText)
	ui.detailTreasure.ScrollToBeginning()
	ui.activeBottomPanel = "treasure"
	if ui.detailBottom != nil {
		ui.detailBottom.SwitchToPage("treasure")
	}
}

func (ui *UI) generateItemTreasureByKinds(kinds []string, count int) ([]Monster, error) {
	if count < 1 {
		return nil, errors.New("quantita deve essere >= 1")
	}
	candidates := filterItemsByTreasureKinds(ui.items, kinds)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("nessun item trovato per tipo \"%s\"", strings.Join(kinds, ","))
	}
	out := make([]Monster, 0, count)
	for i := 0; i < count; i++ {
		idx := rand.Intn(len(candidates))
		out = append(out, candidates[idx])
	}
	return out, nil
}

func filterItemsByTreasureKinds(items []Monster, kinds []string) []Monster {
	if len(kinds) == 0 {
		return append([]Monster(nil), items...)
	}
	set := map[string]struct{}{}
	for _, k := range kinds {
		k = strings.TrimSpace(k)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	if len(set) == 0 {
		return append([]Monster(nil), items...)
	}
	if _, ok := set["random"]; ok {
		return append([]Monster(nil), items...)
	}
	if _, ok := set["any"]; ok {
		return append([]Monster(nil), items...)
	}
	if _, ok := set["*"]; ok {
		return append([]Monster(nil), items...)
	}
	merged := make([]Monster, 0, len(items))
	seen := map[int]struct{}{}
	for k := range set {
		for _, it := range filterItemsByTreasureType(items, k) {
			if _, ok := seen[it.ID]; ok {
				continue
			}
			seen[it.ID] = struct{}{}
			merged = append(merged, it)
		}
	}
	return merged
}

func filterItemsByTreasureType(items []Monster, kind string) []Monster {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" || kind == "random" || kind == "*" || kind == "any" {
		return append([]Monster(nil), items...)
	}
	matches := make([]Monster, 0, len(items))
	for _, it := range items {
		raw := it.Raw
		typ := strings.ToLower(strings.TrimSpace(it.Type))
		name := strings.ToLower(strings.TrimSpace(it.Name))
		hasFlag := func(key string) bool {
			b, ok := raw[key].(bool)
			return ok && b
		}
		ok := false
		switch kind {
		case "potion":
			ok = hasFlag("potion") || strings.Contains(name, "potion")
		case "scroll", "spell":
			ok = hasFlag("scroll") || strings.Contains(name, "scroll")
		case "staff":
			ok = hasFlag("staff") || strings.Contains(typ, "staff")
		case "wand":
			ok = hasFlag("wand") || strings.Contains(typ, "wand")
		case "rod":
			ok = hasFlag("rod") || strings.Contains(typ, "rod")
		case "ring":
			ok = hasFlag("ring") || strings.Contains(typ, "ring")
		case "weapon":
			ok = hasFlag("weapon") || strings.Contains(typ, "weapon")
		case "armor", "armour":
			ok = hasFlag("armor") || strings.Contains(typ, "armor")
		case "wondrous":
			ok = hasFlag("wondrous") || strings.Contains(typ, "wondrous")
		default:
			ok = strings.Contains(typ, kind) || strings.Contains(name, kind)
		}
		if ok {
			matches = append(matches, it)
		}
	}
	return matches
}

func (ui *UI) renderGeneratedItemTreasure(kind string, items []Monster) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "random"
	}
	meta := &strings.Builder{}
	fmt.Fprintf(meta, "[yellow]Item Treasure[-]\n")
	fmt.Fprintf(meta, "[white]Type:[-] %s\n", kind)
	fmt.Fprintf(meta, "[white]Count:[-] %d\n", len(items))
	ui.detailMeta.SetText(meta.String())
	ui.detailMeta.ScrollToBeginning()

	lines := make([]string, 0, len(items))
	for i, it := range items {
		rarity := strings.TrimSpace(it.CR)
		if rarity == "" {
			rarity = "n/a"
		}
		price := formatItemBasePrice(it.Raw)
		if price == "" {
			price = "n/a"
		}
		lines = append(lines, fmt.Sprintf("%d. %s [%s] (%s) - %s", i+1, it.Name, it.Source, rarity, price))
	}

	ui.treasureText = fmt.Sprintf("[yellow]Generated Items[-]\n[white]Type:[-] %s  [white]Qty:[-] %d\n\n%s", kind, len(items), strings.Join(lines, "\n"))
	ui.detailTreasure.SetText(ui.treasureText)
	ui.detailTreasure.ScrollToBeginning()
	ui.activeBottomPanel = "treasure"
	if ui.detailBottom != nil {
		ui.detailBottom.SwitchToPage("treasure")
	}
}

func (ui *UI) renderTreasureOutcome(crText string, out treasureOutcome) {
	order := []string{"cp", "sp", "ep", "gp", "pp"}
	coins := make([]string, 0, len(order))
	totalGP := 0.0
	values := map[string]float64{
		"cp": 0.01,
		"sp": 0.1,
		"ep": 0.5,
		"gp": 1.0,
		"pp": 10.0,
	}
	for _, c := range order {
		n := out.Coins[c]
		if n <= 0 {
			continue
		}
		coins = append(coins, fmt.Sprintf("%d %s", n, c))
		totalGP += float64(n) * values[c]
	}
	if len(coins) == 0 {
		coins = append(coins, "0 gp")
	}
	kind := strings.TrimSpace(out.Kind)
	if kind == "" {
		kind = "Individual Treasure"
	}
	meta := &strings.Builder{}
	fmt.Fprintf(meta, "[yellow]Treasure Generator[-]\n")
	fmt.Fprintf(meta, "[white]CR:[-] %s\n", crText)
	fmt.Fprintf(meta, "[white]Table:[-] %s (%s)\n", kind, out.Band)
	fmt.Fprintf(meta, "[white]d100:[-] %d\n", out.D100)
	fmt.Fprintf(meta, "[white]Coins:[-] %s\n", strings.Join(coins, ", "))
	if len(out.Extras) > 0 {
		fmt.Fprintf(meta, "[white]Extras:[-] %s\n", strings.Join(out.Extras, "; "))
	}
	fmt.Fprintf(meta, "[white]GP eq:[-] %.2f", totalGP)
	ui.detailMeta.SetText(meta.String())
	ui.detailMeta.ScrollToBeginning()

	tre := &strings.Builder{}
	fmt.Fprintf(tre, "[yellow]%s[-]\n", kind)
	fmt.Fprintf(tre, "[white]CR:[-] %s   [white]Band:[-] %s   [white]d100:[-] %d\n", crText, out.Band, out.D100)
	fmt.Fprintf(tre, "[white]Coins:[-] %s\n", strings.Join(coins, ", "))
	if len(out.Extras) > 0 {
		fmt.Fprintf(tre, "[white]Extras:[-]\n")
		for _, ex := range out.Extras {
			fmt.Fprintf(tre, "- %s\n", ex)
		}
	}
	fmt.Fprintf(tre, "[white]GP eq:[-] %.2f", totalGP)
	ui.treasureText = tre.String()
	ui.detailTreasure.SetText(ui.treasureText)
	ui.detailTreasure.ScrollToBeginning()
	ui.activeBottomPanel = "treasure"
	if ui.detailBottom != nil {
		ui.detailBottom.SwitchToPage("treasure")
	}

	raw := &strings.Builder{}
	fmt.Fprintf(raw, "Treasure Generation (D&D 5e - %s)\n", kind)
	fmt.Fprintf(raw, "CR input: %s\n", crText)
	fmt.Fprintf(raw, "Band: %s\n", out.Band)
	fmt.Fprintf(raw, "d100 roll: %d\n", out.D100)
	fmt.Fprintf(raw, "\nRoll Breakdown\n")
	for _, line := range out.Breakdown {
		fmt.Fprintf(raw, "- %s\n", line)
	}
	if len(out.Extras) > 0 {
		fmt.Fprintf(raw, "\nExtra Loot\n")
		for _, line := range out.Extras {
			fmt.Fprintf(raw, "- %s\n", line)
		}
	}
	fmt.Fprintf(raw, "\nResult\n%s\n", strings.Join(coins, ", "))
	fmt.Fprintf(raw, "GP equivalent: %.2f\n", totalGP)

	ui.rawText = strings.TrimSpace(raw.String())
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) openDiceRollInput() {
	input := tview.NewInputField().
		SetLabel("Roll ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Dice Roll (e.g. 2d6+d20+1 or 1d6-1) ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)

	closeModal := func() {
		ui.pages.RemovePage("dice-roll")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			exprInput := strings.TrimSpace(input.GetText())
			if exprInput == "" {
				closeModal()
				return
			}
			batchExprs, err := expandDiceRollInput(exprInput)
			if err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] espressione dadi non valida[-:-] %v  %s", err, helpText))
				return
			}
			ui.pushDiceUndo()
			lastTotal := 0
			for _, expr := range batchExprs {
				total, breakdown, rollErr := rollDiceExpression(expr)
				if rollErr != nil {
					ui.status.SetText(fmt.Sprintf(" [white:red] espressione dadi non valida[-:-] %v  %s", rollErr, helpText))
					return
				}
				lastTotal = total
				ui.appendDiceLog(DiceResult{
					Expression: expr,
					Output:     breakdown,
				})
			}
			if len(batchExprs) > 1 {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] creati %d lanci (ultimo=%d)  %s", len(batchExprs), lastTotal, helpText))
			} else {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] %s = %d  %s", batchExprs[0], lastTotal, helpText))
			}
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-roll", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openDiceReRollInput() {
	if len(ui.diceLog) == 0 {
		ui.openDiceRollInput()
		return
	}
	index := ui.dice.GetCurrentItem()
	if index < 0 || index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}

	input := tview.NewInputField().
		SetLabel("Roll ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Edit + Re-roll Dice ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.diceLog[index].Expression)

	closeModal := func() {
		ui.pages.RemovePage("dice-reroll")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			exprInput := strings.TrimSpace(input.GetText())
			if exprInput == "" {
				closeModal()
				return
			}
			batchExprs, err := expandDiceRollInput(exprInput)
			if err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] espressione dadi non valida[-:-] %v  %s", err, helpText))
				return
			}
			ui.pushDiceUndo()
			total, breakdown, rollErr := rollDiceExpression(batchExprs[0])
			if rollErr != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] espressione dadi non valida[-:-] %v  %s", rollErr, helpText))
				return
			}
			ui.diceLog[index] = DiceResult{Expression: batchExprs[0], Output: breakdown}
			insertAt := index + 1
			lastTotal := total
			for i := 1; i < len(batchExprs); i++ {
				t, b, e := rollDiceExpression(batchExprs[i])
				if e != nil {
					ui.status.SetText(fmt.Sprintf(" [white:red] espressione dadi non valida[-:-] %v  %s", e, helpText))
					return
				}
				lastTotal = t
				entry := DiceResult{Expression: batchExprs[i], Output: b}
				ui.diceLog = append(ui.diceLog, DiceResult{})
				copy(ui.diceLog[insertAt+1:], ui.diceLog[insertAt:])
				ui.diceLog[insertAt] = entry
				insertAt++
			}
			ui.renderDiceList()
			ui.dice.SetCurrentItem(index)
			if len(batchExprs) > 1 {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] aggiornato in %d lanci (ultimo=%d)  %s", len(batchExprs), lastTotal, helpText))
			} else {
				ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] aggiornato %s = %d  %s", batchExprs[0], total, helpText))
			}
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-reroll", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) rerollSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		return
	}
	index := ui.dice.GetCurrentItem()
	if index < 0 || index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}
	expr := strings.TrimSpace(ui.diceLog[index].Expression)
	if expr == "" {
		return
	}
	total, breakdown, err := rollDiceExpression(expr)
	if err != nil {
		ui.status.SetText(fmt.Sprintf(" [white:red] espressione dadi non valida[-:-] %v  %s", err, helpText))
		return
	}
	ui.pushDiceUndo()
	ui.diceLog[index] = DiceResult{
		Expression: expr,
		Output:     breakdown,
	}
	ui.renderDiceList()
	ui.dice.SetCurrentItem(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] rilanciato %s = %d  %s", expr, total, helpText))
}

func (ui *UI) appendDiceLog(entry DiceResult) {
	ui.diceLog = append(ui.diceLog, entry)
	if len(ui.diceLog) > 100 {
		ui.diceLog = ui.diceLog[len(ui.diceLog)-100:]
	}
	ui.renderDiceList()
	if len(ui.diceLog) > 0 {
		ui.dice.SetCurrentItem(len(ui.diceLog) - 1)
	}
}

func (ui *UI) renderDiceList() {
	ui.diceRender = true
	defer func() { ui.diceRender = false }()
	current := ui.dice.GetCurrentItem()
	if current < 0 {
		current = 0
	}
	ui.dice.Clear()
	for i, row := range ui.diceLog {
		expr := row.Expression
		out := row.Output
		if i == current {
			expr = "[black:gold]" + expr + "[-:-]"
			out = highlightDiceFinalResult(out)
		}
		ui.dice.AddItem(fmt.Sprintf("%d %s => %s", i+1, expr, out), "", 0, nil)
	}
	if len(ui.diceLog) == 0 {
		return
	}
	if current >= len(ui.diceLog) {
		current = len(ui.diceLog) - 1
	}
	ui.dice.SetCurrentItem(current)
}

func highlightDiceFinalResult(output string) string {
	locs := finalResultRe.FindAllStringIndex(output, -1)
	if len(locs) == 0 {
		return output
	}
	last := locs[len(locs)-1]
	return output[:last[0]] + "[black:gold]" + output[last[0]:last[1]] + "[-:-]" + output[last[1]:]
}

func (ui *UI) deleteSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		return
	}
	ui.pushDiceUndo()
	index := ui.dice.GetCurrentItem()
	if index < 0 || index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}
	ui.diceLog = append(ui.diceLog[:index], ui.diceLog[index+1:]...)
	ui.renderDiceList()
	if len(ui.diceLog) == 0 {
		ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] lista vuota  %s", helpText))
		return
	}
	if index >= len(ui.diceLog) {
		index = len(ui.diceLog) - 1
	}
	ui.dice.SetCurrentItem(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] riga eliminata  %s", helpText))
}

func (ui *UI) clearDiceResults() {
	if len(ui.diceLog) == 0 {
		return
	}
	ui.pushDiceUndo()
	ui.diceLog = nil
	ui.renderDiceList()
	ui.status.SetText(fmt.Sprintf(" [black:gold]dice[-:-] tutte le righe cancellate  %s", helpText))
}

func (ui *UI) pushDiceUndo() {
	snap := DiceUndoState{
		Items:    append([]DiceResult(nil), ui.diceLog...),
		Selected: ui.dice.GetCurrentItem(),
	}
	ui.diceUndo = append(ui.diceUndo, snap)
	ui.diceRedo = ui.diceRedo[:0]
}

func (ui *UI) captureDiceState() DiceUndoState {
	return DiceUndoState{
		Items:    append([]DiceResult(nil), ui.diceLog...),
		Selected: ui.dice.GetCurrentItem(),
	}
}

func (ui *UI) restoreDiceState(state DiceUndoState) {
	ui.diceLog = append([]DiceResult(nil), state.Items...)
	ui.renderDiceList()
	if len(ui.diceLog) == 0 {
		return
	}
	idx := state.Selected
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ui.diceLog) {
		idx = len(ui.diceLog) - 1
	}
	ui.dice.SetCurrentItem(idx)
}

func (ui *UI) undoDiceCommand() {
	if len(ui.diceUndo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] nessuna operazione dice da annullare[-:-]  %s", helpText))
		return
	}
	current := ui.captureDiceState()
	last := ui.diceUndo[len(ui.diceUndo)-1]
	ui.diceUndo = ui.diceUndo[:len(ui.diceUndo)-1]
	ui.diceRedo = append(ui.diceRedo, current)
	ui.restoreDiceState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] undo[-:-] operazione dice annullata  %s", helpText))
}

func (ui *UI) redoDiceCommand() {
	if len(ui.diceRedo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] nessuna operazione dice da ripristinare[-:-]  %s", helpText))
		return
	}
	current := ui.captureDiceState()
	last := ui.diceRedo[len(ui.diceRedo)-1]
	ui.diceRedo = ui.diceRedo[:len(ui.diceRedo)-1]
	ui.diceUndo = append(ui.diceUndo, current)
	ui.restoreDiceState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] redo[-:-] operazione dice ripristinata  %s", helpText))
}

func (ui *UI) openDiceSaveAsInput() {
	input := tview.NewInputField().
		SetLabel("Dice file ").
		SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Save Dice Results As ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.dicePath)

	closeModal := func() {
		ui.pages.RemovePage("dice-save")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			path := strings.TrimSpace(input.GetText())
			if path == "" {
				ui.status.SetText(fmt.Sprintf(" [white:red] nome file non valido[-:-]  %s", helpText))
				return
			}
			if err := ui.saveDiceResultsAs(path); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] errore save dice[-:-] %v  %s", err, helpText))
				return
			}
			ui.status.SetText(fmt.Sprintf(" [black:gold] salvato dice[-:-] %s  %s", ui.dicePath, helpText))
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-save", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openDiceLoadInput() {
	input := tview.NewInputField().
		SetLabel("Dice file ").
		SetFieldWidth(56)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetTitle(" Load Dice Results ")
	input.SetBorder(true)
	input.SetTitleColor(tcell.ColorGold)
	input.SetBorderColor(tcell.ColorGold)
	input.SetText(ui.dicePath)

	closeModal := func() {
		ui.pages.RemovePage("dice-load")
		ui.app.SetFocus(ui.dice)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			closeModal()
		case tcell.KeyEnter:
			path := strings.TrimSpace(input.GetText())
			if path == "" {
				ui.status.SetText(fmt.Sprintf(" [white:red] nome file non valido[-:-]  %s", helpText))
				return
			}
			prevState := ui.captureDiceState()
			prev := ui.dicePath
			ui.dicePath = path
			if err := ui.loadDiceResults(); err != nil {
				ui.dicePath = prev
				ui.status.SetText(fmt.Sprintf(" [white:red] errore load dice[-:-] %v  %s", err, helpText))
				return
			}
			ui.diceUndo = append(ui.diceUndo, prevState)
			ui.diceRedo = ui.diceRedo[:0]
			ui.status.SetText(fmt.Sprintf(" [black:gold] caricato dice[-:-] %s  %s", ui.dicePath, helpText))
			closeModal()
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("dice-load", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) updateFilterLayout(screenWidth int) {
	if ui.filterHost == nil || ui.leftPanel == nil || ui.monstersPanel == nil {
		return
	}
	wide := screenWidth >= 140
	if ui.wideFilter == wide {
		return
	}
	ui.wideFilter = wide
	if ui.fullscreenActive {
		return
	}
	if wide {
		ui.filterHost.SwitchToPage("single")
		ui.monstersPanel.ResizeItem(ui.filterHost, 1, 0)
	} else {
		ui.filterHost.SwitchToPage("double")
		ui.monstersPanel.ResizeItem(ui.filterHost, 2, 0)
	}
}

func (ui *UI) applyBaseLayout() {
	if ui.mainRow == nil || ui.leftPanel == nil || ui.detailPanel == nil || ui.filterHost == nil || ui.monstersPanel == nil || ui.detailBottom == nil {
		return
	}
	ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
	ui.mainRow.ResizeItem(ui.detailPanel, 0, 1)
	ui.leftPanel.ResizeItem(ui.dice, 7, 0)
	ui.leftPanel.ResizeItem(ui.encounter, 8, 0)
	ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 1)
	if ui.wideFilter {
		ui.filterHost.SwitchToPage("single")
		ui.monstersPanel.ResizeItem(ui.filterHost, 1, 0)
	} else {
		ui.filterHost.SwitchToPage("double")
		ui.monstersPanel.ResizeItem(ui.filterHost, 2, 0)
	}
	ui.monstersPanel.ResizeItem(ui.list, 0, 1)
	ui.detailPanel.ResizeItem(ui.detailMeta, 8, 0)
	ui.detailPanel.ResizeItem(ui.detailBottom, 0, 1)
}

func (ui *UI) fullscreenTargetForFocus(focus tview.Primitive) string {
	switch focus {
	case ui.dice:
		return "dice"
	case ui.encounter:
		return "encounter"
	case ui.list:
		return "monsters"
	case ui.detailRaw, ui.detailTreasure, ui.detailMeta:
		return "description"
	case ui.nameInput, ui.envDrop, ui.sourceDrop, ui.crDrop, ui.typeDrop:
		return "filters"
	default:
		return ""
	}
}

func (ui *UI) toggleFullscreenForFocus(focus tview.Primitive) {
	if ui.fullscreenActive {
		ui.fullscreenActive = false
		ui.fullscreenTarget = ""
		ui.applyBaseLayout()
		ui.status.SetText(fmt.Sprintf(" [black:gold]fullscreen[-:-] disattivato  %s", helpText))
		return
	}
	target := ui.fullscreenTargetForFocus(focus)
	if target == "" || ui.mainRow == nil || ui.leftPanel == nil || ui.detailPanel == nil || ui.filterHost == nil || ui.monstersPanel == nil {
		return
	}
	ui.applyBaseLayout()
	ui.fullscreenActive = true
	ui.fullscreenTarget = target

	switch target {
	case "dice":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 1)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 0)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 0)
	case "encounter":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 0)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 1)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 0)
	case "filters":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 0)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 0)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 1)
		ui.monstersPanel.ResizeItem(ui.filterHost, 0, 1)
		ui.monstersPanel.ResizeItem(ui.list, 0, 0)
	case "monsters":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 1)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 0)
		ui.leftPanel.ResizeItem(ui.dice, 0, 0)
		ui.leftPanel.ResizeItem(ui.encounter, 0, 0)
		ui.leftPanel.ResizeItem(ui.monstersPanel, 0, 1)
		ui.monstersPanel.ResizeItem(ui.filterHost, 0, 0)
		ui.monstersPanel.ResizeItem(ui.list, 0, 1)
	case "description":
		ui.mainRow.ResizeItem(ui.leftPanel, 0, 0)
		ui.mainRow.ResizeItem(ui.detailPanel, 0, 1)
		ui.detailPanel.ResizeItem(ui.detailMeta, 0, 0)
		ui.detailPanel.ResizeItem(ui.detailBottom, 0, 1)
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold]fullscreen[-:-] %s  %s", target, helpText))
}

func (ui *UI) run() error {
	return ui.app.Run()
}

func (ui *UI) closeItemTreasureModal() {
	ui.pages.RemovePage("items-treasure-input")
	ui.itemTreasureVisible = false
	ui.app.SetFocus(ui.list)
}

func (ui *UI) closeSpellTreasureModal() {
	ui.pages.RemovePage("spells-treasure-input")
	ui.spellTreasureVisible = false
	ui.app.SetFocus(ui.list)
}

func (ui *UI) browseModeName() string {
	switch ui.browseMode {
	case BrowseItems:
		return "Items"
	case BrowseSpells:
		return "Spells"
	default:
		return "Monsters"
	}
}

func (ui *UI) activeEntries() []Monster {
	switch ui.browseMode {
	case BrowseItems:
		return ui.items
	case BrowseSpells:
		return ui.spells
	default:
		return ui.monsters
	}
}

func (ui *UI) setFilterOptionsForMode() {
	switch ui.browseMode {
	case BrowseItems:
		ui.nameInput.SetLabel(" Name ")
		ui.envDrop.SetLabel(" Env ")
		ui.sourceDrop.SetLabel(" Source ")
		ui.crDrop.SetLabel(" Rarity ")
		ui.typeDrop.SetLabel(" Type ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenSource := map[string]struct{}{}
		seenCR := map[string]struct{}{}
		seenType := map[string]struct{}{}
		for _, it := range ui.items {
			if s := strings.TrimSpace(it.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(it.CR); s != "" {
				seenCR[s] = struct{}{}
			}
			if s := strings.TrimSpace(it.Type); s != "" {
				seenType[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenSource)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, keysSorted(seenCR)...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenType)...)
	case BrowseSpells:
		ui.nameInput.SetLabel(" Name ")
		ui.envDrop.SetLabel(" Env ")
		ui.sourceDrop.SetLabel(" Source ")
		ui.crDrop.SetLabel(" Level ")
		ui.typeDrop.SetLabel(" School ")
		ui.envOptions = []string{"All"}
		ui.sourceOptions = []string{"All"}
		ui.crOptions = []string{"All"}
		ui.typeOptions = []string{"All"}
		seenSource := map[string]struct{}{}
		seenCR := map[string]struct{}{}
		seenType := map[string]struct{}{}
		for _, sp := range ui.spells {
			if s := strings.TrimSpace(sp.Source); s != "" {
				seenSource[s] = struct{}{}
			}
			if s := strings.TrimSpace(sp.CR); s != "" {
				seenCR[s] = struct{}{}
			}
			if s := strings.TrimSpace(sp.Type); s != "" {
				seenType[s] = struct{}{}
			}
		}
		ui.envOptions = append(ui.envOptions, keysSorted(seenSource)...)
		ui.sourceOptions = append(ui.sourceOptions, keysSorted(seenSource)...)
		ui.crOptions = append(ui.crOptions, sortCR(keysSorted(seenCR))...)
		ui.typeOptions = append(ui.typeOptions, keysSorted(seenType)...)
	default:
		ui.nameInput.SetLabel(" Name ")
		ui.envDrop.SetLabel(" Env ")
		ui.sourceDrop.SetLabel(" Source ")
		ui.crDrop.SetLabel(" CR ")
		ui.typeDrop.SetLabel(" Type ")
		ui.envOptions = append([]string{"All"}, ui.collectMonsterEnvOptions()...)
		ui.sourceOptions = append([]string{"All"}, ui.collectMonsterSourceOptions()...)
		ui.crOptions = append([]string{"All"}, ui.collectMonsterCROptions()...)
		ui.typeOptions = append([]string{"All"}, ui.collectMonsterTypeOptions()...)
	}
	ui.envDrop.SetOptions(ui.envOptions, func(option string, _ int) {
		if option == "All" {
			ui.envFilter = ""
		} else {
			ui.envFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	ui.setDropDownByValue(ui.envDrop, ui.envOptions, ui.envFilter)
	ui.refreshSourceDropOptions(-1)
	ui.crDrop.SetOptions(ui.crOptions, func(option string, _ int) {
		if option == "All" {
			ui.crFilter = ""
		} else {
			ui.crFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	ui.typeDrop.SetOptions(ui.typeOptions, func(option string, _ int) {
		if option == "All" {
			ui.typeFilter = ""
		} else {
			ui.typeFilter = option
		}
		ui.applyFilters()
		ui.maybeReturnFocusToListFromFilter()
	})
	if ui.browseMode != BrowseMonsters {
		ui.envFilter = ""
		ui.envDrop.SetCurrentOption(0)
	}
	ui.setDropDownByValue(ui.crDrop, ui.crOptions, ui.crFilter)
	ui.setDropDownByValue(ui.typeDrop, ui.typeOptions, ui.typeFilter)
}

func (ui *UI) refreshSourceDropOptions(preferredIdx int) {
	_ = preferredIdx
	label := "All"
	if n := len(ui.sourceFilters); n > 0 {
		label = fmt.Sprintf("%d selected", n)
	}
	ui.updatingSourceDrop = true
	ui.sourceDrop.SetOptions([]string{label}, nil)
	ui.sourceDrop.SetCurrentOption(0)
	ui.updatingSourceDrop = false
}

func (ui *UI) toggleCurrentSourceOption() {
	// legacy no-op; source multi-select now uses dedicated modal.
}

func (ui *UI) selectedSourcesSorted() []string {
	return keysSorted(ui.sourceFilters)
}

func (ui *UI) openSourceMultiSelectModal() {
	if len(ui.sourceOptions) <= 1 {
		return
	}
	temp := map[string]struct{}{}
	for k := range ui.sourceFilters {
		temp[k] = struct{}{}
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Source Filter (Space=toggle, Enter=apply, Esc=cancel) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	render := func() {
		cur := list.GetCurrentItem()
		list.Clear()
		if len(temp) == 0 {
			list.AddItem("[x] All", "", 0, nil)
		} else {
			list.AddItem("[ ] All", "", 0, nil)
		}
		for _, opt := range ui.sourceOptions[1:] {
			mark := "[ ]"
			if _, ok := temp[opt]; ok {
				mark = "[x]"
			}
			list.AddItem(fmt.Sprintf("%s %s", mark, opt), "", 0, nil)
		}
		if cur < 0 {
			cur = 0
		}
		if cur >= list.GetItemCount() {
			cur = list.GetItemCount() - 1
		}
		if cur < 0 {
			cur = 0
		}
		list.SetCurrentItem(cur)
	}

	closeModal := func(apply bool) {
		ui.pages.RemovePage("source-multi")
		if apply {
			ui.sourceFilters = temp
			ui.refreshSourceDropOptions(-1)
			ui.applyFilters()
			ui.app.SetFocus(ui.list)
			return
		}
		ui.app.SetFocus(ui.sourceDrop)
	}

	toggle := func() {
		idx := list.GetCurrentItem()
		if idx <= 0 {
			temp = map[string]struct{}{}
			render()
			return
		}
		if idx >= len(ui.sourceOptions) {
			return
		}
		opt := ui.sourceOptions[idx]
		if _, ok := temp[opt]; ok {
			delete(temp, opt)
		} else {
			temp[opt] = struct{}{}
		}
		render()
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == ' ':
			toggle()
			return nil
		case event.Key() == tcell.KeyEnter:
			closeModal(true)
			return nil
		case event.Key() == tcell.KeyEscape:
			closeModal(false)
			return nil
		default:
			return event
		}
	})

	render()
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 16, 0, true).
			AddItem(nil, 0, 1, false), 70, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("source-multi", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *UI) setSelectedSources(values []string) {
	ui.sourceFilters = map[string]struct{}{}
	allowed := map[string]struct{}{}
	for _, v := range ui.sourceOptions[1:] {
		allowed[v] = struct{}{}
	}
	for _, v := range values {
		if _, ok := allowed[v]; ok {
			ui.sourceFilters[v] = struct{}{}
		}
	}
}

func (ui *UI) setDropDownByValue(drop *tview.DropDown, options []string, value string) {
	if strings.TrimSpace(value) == "" {
		drop.SetCurrentOption(0)
		return
	}
	for i, opt := range options {
		if strings.EqualFold(opt, value) {
			drop.SetCurrentOption(i)
			return
		}
	}
	drop.SetCurrentOption(0)
}

func (ui *UI) maybeReturnFocusToListFromFilter() {
	focus := ui.app.GetFocus()
	if focus == ui.envDrop || focus == ui.crDrop || focus == ui.typeDrop {
		ui.app.SetFocus(ui.list)
	}
}

func (ui *UI) saveCurrentModeFilters() {
	ui.modeFilters[ui.browseMode] = PersistedFilterMode{
		Name:    strings.TrimSpace(ui.nameFilter),
		Env:     strings.TrimSpace(ui.envFilter),
		Sources: ui.selectedSourcesSorted(),
		CR:      strings.TrimSpace(ui.crFilter),
		Type:    strings.TrimSpace(ui.typeFilter),
	}
}

func (ui *UI) applyModeFilters(mode BrowseMode) {
	state, ok := ui.modeFilters[mode]
	if !ok {
		state = PersistedFilterMode{}
	}
	ui.nameFilter = strings.TrimSpace(state.Name)
	ui.envFilter = strings.TrimSpace(state.Env)
	ui.crFilter = strings.TrimSpace(state.CR)
	ui.typeFilter = strings.TrimSpace(state.Type)
	ui.setFilterOptionsForMode()
	ui.setSelectedSources(state.Sources)
	ui.refreshSourceDropOptions(-1)
	ui.nameInput.SetText(ui.nameFilter)
	ui.setDropDownByValue(ui.crDrop, ui.crOptions, ui.crFilter)
	ui.setDropDownByValue(ui.typeDrop, ui.typeOptions, ui.typeFilter)
}

func (ui *UI) collectMonsterEnvOptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		for _, env := range m.Environment {
			if strings.TrimSpace(env) != "" {
				set[env] = struct{}{}
			}
		}
	}
	return keysSorted(set)
}

func (ui *UI) collectMonsterSourceOptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		if s := strings.TrimSpace(m.Source); s != "" {
			set[s] = struct{}{}
		}
	}
	return keysSorted(set)
}

func (ui *UI) collectMonsterCROptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		if s := strings.TrimSpace(m.CR); s != "" {
			set[s] = struct{}{}
		}
	}
	return sortCR(keysSorted(set))
}

func (ui *UI) collectMonsterTypeOptions() []string {
	set := map[string]struct{}{}
	for _, m := range ui.monsters {
		if s := strings.TrimSpace(m.Type); s != "" {
			set[s] = struct{}{}
		}
	}
	return keysSorted(set)
}

func (ui *UI) updateBrowsePanelTitle() {
	ui.monstersPanel.SetTitle(fmt.Sprintf(" [2]-%s [Monsters > Items > Spells] ", ui.browseModeName()))
}

func (ui *UI) cycleBrowseMode(delta int) {
	if delta == 0 {
		return
	}
	count := 3
	next := (int(ui.browseMode) + delta) % count
	if next < 0 {
		next += count
	}
	ui.saveCurrentModeFilters()
	ui.browseMode = BrowseMode(next)
	ui.spellShortcutAlt = false
	ui.applyModeFilters(ui.browseMode)
	ui.updateBrowsePanelTitle()
	ui.applyFilters()
	ui.status.SetText(fmt.Sprintf(" [black:gold]browse[-:-] %s  %s", ui.browseModeName(), helpText))
}

func (ui *UI) applyFilters() {
	ui.filtered = ui.filtered[:0]

	for i, m := range ui.activeEntries() {
		if !matchName(m.Name, ui.nameFilter) {
			continue
		}
		if !matchCR(m.CR, ui.crFilter) {
			continue
		}
		if !matchEnv(m.Environment, ui.envFilter) {
			continue
		}
		if !matchEnvMulti([]string{m.Source}, ui.sourceFilters) {
			continue
		}
		if !matchType(m.Type, ui.typeFilter) {
			continue
		}
		ui.filtered = append(ui.filtered, i)
	}

	ui.renderList()
}

func (ui *UI) renderList() {
	ui.list.Clear()

	entries := ui.activeEntries()
	for _, idx := range ui.filtered {
		if idx < 0 || idx >= len(entries) {
			continue
		}
		m := entries[idx]
		ui.list.AddItem(m.Name, "", 0, nil)
	}

	ui.status.SetText(fmt.Sprintf(" [black:gold] %d risultati [-:-] %s", len(ui.filtered), helpText))

	if len(ui.filtered) == 0 {
		ui.detailMeta.SetText(fmt.Sprintf("Nessun risultato in %s con i filtri correnti.", ui.browseModeName()))
		ui.detailRaw.SetText("")
		ui.rawText = ""
		return
	}

	current := ui.list.GetCurrentItem()
	if current < 0 || current >= len(ui.filtered) {
		current = 0
		ui.list.SetCurrentItem(0)
	}
	ui.renderDetailByListIndex(current)
}

func (ui *UI) renderDetailByListIndex(listIndex int) {
	if listIndex < 0 || listIndex >= len(ui.filtered) {
		ui.detailMeta.SetText(fmt.Sprintf("Seleziona un elemento da %s.", ui.browseModeName()))
		ui.detailRaw.SetText("")
		ui.rawText = ""
		return
	}
	activeIndex := ui.filtered[listIndex]
	switch ui.browseMode {
	case BrowseItems:
		ui.renderDetailByItemIndex(activeIndex)
	case BrowseSpells:
		ui.renderDetailBySpellIndex(activeIndex)
	default:
		ui.renderDetailByMonsterIndex(activeIndex)
	}
}

func (ui *UI) renderDetailByEncounterIndex(encounterIndex int) {
	if encounterIndex < 0 || encounterIndex >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[encounterIndex]
	if entry.Custom {
		ui.renderDetailByCustomEntry(entry)
		return
	}
	ui.renderDetailByMonsterIndex(entry.MonsterIndex)
}

func (ui *UI) renderDetailByMonsterIndex(monsterIndex int) {
	if monsterIndex < 0 || monsterIndex >= len(ui.monsters) {
		return
	}

	m := ui.monsters[monsterIndex]

	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", m.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(m.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Type:[-] %s\n", blankIfEmpty(m.Type, "n/a"))
	fmt.Fprintf(builder, "[white]CR:[-] %s\n", blankIfEmpty(m.CR, "n/a"))
	if ac := extractAC(m.Raw); ac != "" {
		fmt.Fprintf(builder, "[white]AC:[-] %s\n", ac)
	}
	if speed := extractSpeed(m.Raw); speed != "" {
		fmt.Fprintf(builder, "[white]Speed:[-] %s\n", speed)
	}
	hpAverage, hpFormula := extractHP(m.Raw)
	if hpAverage != "" || hpFormula != "" {
		switch {
		case hpAverage != "" && hpFormula != "":
			fmt.Fprintf(builder, "[white]HP:[-] %s (%s)\n", hpAverage, hpFormula)
		case hpAverage != "":
			fmt.Fprintf(builder, "[white]HP:[-] %s\n", hpAverage)
		default:
			fmt.Fprintf(builder, "[white]HP:[-] %s\n", hpFormula)
		}
	}
	if len(m.Environment) > 0 {
		fmt.Fprintf(builder, "[white]Environment:[-] %s\n", strings.Join(m.Environment, ", "))
	} else {
		fmt.Fprintf(builder, "[white]Environment:[-] n/a\n")
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildMonsterDescriptionText(m)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByItemIndex(itemIndex int) {
	if itemIndex < 0 || itemIndex >= len(ui.items) {
		return
	}
	it := ui.items[itemIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", it.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(it.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Type:[-] %s\n", blankIfEmpty(it.Type, "n/a"))
	fmt.Fprintf(builder, "[white]Rarity:[-] %s\n", blankIfEmpty(it.CR, "n/a"))
	if price := formatItemBasePrice(it.Raw); price != "" {
		fmt.Fprintf(builder, "[white]Price:[-] %s\n", price)
	}
	if attune := strings.TrimSpace(asString(it.Raw["reqAttune"])); attune != "" {
		fmt.Fprintf(builder, "[white]Attunement:[-] %s\n", attune)
	}
	if ac := strings.TrimSpace(asString(it.Raw["ac"])); ac != "" {
		fmt.Fprintf(builder, "[white]AC:[-] %s\n", ac)
	}
	if econ, ok := magicItemEconomy(it.Raw, it.CR); ok {
		fmt.Fprintf(builder, "[white]Buy Cost:[-] %s\n", econ.BuyCost)
		fmt.Fprintf(builder, "[white]Find Time:[-] %s\n", econ.FindTime)
		fmt.Fprintf(builder, "[white]Craft Cost:[-] %s\n", econ.CraftCost)
		fmt.Fprintf(builder, "[white]Craft Time:[-] %s\n", econ.CraftTime)
		fmt.Fprintf(builder, "[white]Craft Procedure:[-] %s\n", strings.Join(econ.Procedure, " -> "))
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildItemDescriptionText(it)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailBySpellIndex(spellIndex int) {
	if spellIndex < 0 || spellIndex >= len(ui.spells) {
		return
	}
	sp := ui.spells[spellIndex]
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", sp.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(sp.Source, "n/a"))
	fmt.Fprintf(builder, "[white]School:[-] %s\n", blankIfEmpty(sp.Type, "n/a"))
	fmt.Fprintf(builder, "[white]Level:[-] %s\n", blankIfEmpty(sp.CR, "n/a"))
	if cast := extractSpellTime(sp.Raw); cast != "" {
		fmt.Fprintf(builder, "[white]Casting Time:[-] %s\n", cast)
	}
	if rng := extractSpellRange(sp.Raw); rng != "" {
		fmt.Fprintf(builder, "[white]Range:[-] %s\n", rng)
	}
	if dur := extractSpellDuration(sp.Raw); dur != "" {
		fmt.Fprintf(builder, "[white]Duration:[-] %s\n", dur)
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildSpellDescriptionText(sp)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func (ui *UI) renderDetailByCustomEntry(entry EncounterEntry) {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", ui.encounterEntryDisplay(entry))
	if init, ok := ui.encounterInitBase(entry); ok {
		if entry.HasInitRoll {
			fmt.Fprintf(builder, "[white]Init:[-] %d/%d\n", entry.InitRoll, init)
		} else {
			fmt.Fprintf(builder, "[white]Init:[-] %d\n", init)
		}
	}
	if strings.TrimSpace(entry.CustomAC) != "" {
		fmt.Fprintf(builder, "[white]AC:[-] %s\n", entry.CustomAC)
	}
	maxHP := ui.encounterMaxHP(entry)
	if maxHP > 0 {
		fmt.Fprintf(builder, "[white]HP:[-] %d/%d\n", entry.CurrentHP, maxHP)
	} else {
		fmt.Fprintf(builder, "[white]HP:[-] ?\n")
	}
	ui.detailMeta.SetText(builder.String())
	ui.detailMeta.ScrollToBeginning()
	ui.rawText = buildCustomDescriptionText(entry, maxHP)
	ui.rawQuery = ""
	ui.renderRawWithHighlight("", -1)
	ui.detailRaw.ScrollToBeginning()
}

func buildMonsterDescriptionText(m Monster) string {
	raw := m.Raw
	b := &strings.Builder{}

	fmt.Fprintf(b, "Name: %s\n", m.Name)
	if src := strings.TrimSpace(m.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if t := strings.TrimSpace(m.Type); t != "" {
		fmt.Fprintf(b, "Type: %s\n", t)
	}
	if cr := strings.TrimSpace(m.CR); cr != "" {
		fmt.Fprintf(b, "Challenge: %s\n", cr)
	}
	if align := plainAny(raw["alignment"]); align != "" {
		fmt.Fprintf(b, "Alignment: %s\n", align)
	}
	if ac := extractAC(raw); ac != "" {
		fmt.Fprintf(b, "Armor Class: %s\n", ac)
	}
	hpAverage, hpFormula := extractHP(raw)
	if hpAverage != "" || hpFormula != "" {
		if hpAverage != "" && hpFormula != "" {
			fmt.Fprintf(b, "Hit Points: %s (%s)\n", hpAverage, hpFormula)
		} else if hpAverage != "" {
			fmt.Fprintf(b, "Hit Points: %s\n", hpAverage)
		} else {
			fmt.Fprintf(b, "Hit Points: %s\n", hpFormula)
		}
	}
	if speed := extractSpeed(raw); speed != "" {
		fmt.Fprintf(b, "Speed: %s\n", speed)
	}
	if s := abilityBlock(raw); s != "" {
		fmt.Fprintf(b, "\n%s\n", s)
	}

	orderedFields := []struct {
		key   string
		label string
	}{
		{"save", "Saving Throws"},
		{"skill", "Skills"},
		{"vulnerable", "Damage Vulnerabilities"},
		{"resist", "Damage Resistances"},
		{"immune", "Damage Immunities"},
		{"conditionImmune", "Condition Immunities"},
		{"senses", "Senses"},
		{"languages", "Languages"},
	}
	for _, f := range orderedFields {
		if txt := plainAny(raw[f.key]); txt != "" {
			fmt.Fprintf(b, "%s: %s\n", f.label, txt)
		}
	}

	sectionOrder := []struct {
		key   string
		label string
	}{
		{"trait", "Traits"},
		{"action", "Actions"},
		{"bonus", "Bonus Actions"},
		{"reaction", "Reactions"},
		{"legendary", "Legendary Actions"},
		{"mythic", "Mythic Actions"},
	}
	for _, sec := range sectionOrder {
		if body := plainSection(raw[sec.key]); body != "" {
			fmt.Fprintf(b, "\n%s\n%s\n", sec.label, body)
		}
	}
	return strings.TrimSpace(b.String())
}

func buildItemDescriptionText(it Monster) string {
	raw := it.Raw
	b := &strings.Builder{}

	fmt.Fprintf(b, "Name: %s\n", it.Name)
	if src := strings.TrimSpace(it.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if t := strings.TrimSpace(it.Type); t != "" {
		fmt.Fprintf(b, "Type: %s\n", t)
	}
	if rarity := strings.TrimSpace(it.CR); rarity != "" {
		fmt.Fprintf(b, "Rarity: %s\n", rarity)
	}
	if price := formatItemBasePrice(raw); price != "" {
		fmt.Fprintf(b, "Price: %s\n", price)
	}
	if req := strings.TrimSpace(asString(raw["reqAttune"])); req != "" {
		fmt.Fprintf(b, "Attunement: %s\n", req)
	}
	if weight := strings.TrimSpace(asString(raw["weight"])); weight != "" {
		fmt.Fprintf(b, "Weight: %s\n", weight)
	}
	if value := strings.TrimSpace(asString(raw["value"])); value != "" {
		fmt.Fprintf(b, "Value: %s\n", value)
	}
	if econ, ok := magicItemEconomy(raw, it.CR); ok {
		fmt.Fprintf(b, "Buy Cost: %s\n", econ.BuyCost)
		fmt.Fprintf(b, "Find Time in Shop: %s\n", econ.FindTime)
		fmt.Fprintf(b, "Craft Cost: %s\n", econ.CraftCost)
		fmt.Fprintf(b, "Craft Time: %s\n", econ.CraftTime)
		fmt.Fprintf(b, "Craft Procedure: %s\n", strings.Join(econ.Procedure, " -> "))
	}
	if entries := plainAny(raw["entries"]); entries != "" {
		fmt.Fprintf(b, "\nDescription\n%s\n", entries)
	}
	return strings.TrimSpace(b.String())
}

func buildSpellDescriptionText(sp Monster) string {
	raw := sp.Raw
	b := &strings.Builder{}

	fmt.Fprintf(b, "Name: %s\n", sp.Name)
	if src := strings.TrimSpace(sp.Source); src != "" {
		fmt.Fprintf(b, "Source: %s\n", src)
	}
	if level := strings.TrimSpace(sp.CR); level != "" {
		fmt.Fprintf(b, "Level: %s\n", level)
	}
	if school := strings.TrimSpace(sp.Type); school != "" {
		fmt.Fprintf(b, "School: %s\n", school)
	}
	if cast := extractSpellTime(raw); cast != "" {
		fmt.Fprintf(b, "Casting Time: %s\n", cast)
	}
	if rng := extractSpellRange(raw); rng != "" {
		fmt.Fprintf(b, "Range: %s\n", rng)
	}
	if dur := extractSpellDuration(raw); dur != "" {
		fmt.Fprintf(b, "Duration: %s\n", dur)
	}
	if comps := plainAny(raw["components"]); comps != "" {
		fmt.Fprintf(b, "Components: %s\n", comps)
	}
	if entries := plainAny(raw["entries"]); entries != "" {
		fmt.Fprintf(b, "\nDescription\n%s\n", entries)
	}
	if higher := plainAny(raw["entriesHigherLevel"]); higher != "" {
		fmt.Fprintf(b, "\nAt Higher Levels\n%s\n", higher)
	}
	return strings.TrimSpace(b.String())
}

func buildCustomDescriptionText(entry EncounterEntry, maxHP int) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "Name: %s\n", entry.CustomName)
	fmt.Fprintf(b, "Initiative: %d\n", entry.CustomInit)
	if entry.HasInitRoll {
		fmt.Fprintf(b, "Initiative Roll: %d\n", entry.InitRoll)
	}
	if strings.TrimSpace(entry.CustomAC) != "" {
		fmt.Fprintf(b, "Armor Class: %s\n", entry.CustomAC)
	}
	if maxHP > 0 {
		fmt.Fprintf(b, "Hit Points: %d/%d\n", entry.CurrentHP, maxHP)
	} else {
		fmt.Fprintf(b, "Hit Points: ?\n")
	}
	return strings.TrimSpace(b.String())
}

func abilityBlock(raw map[string]any) string {
	keys := []string{"str", "dex", "con", "int", "wis", "cha"}
	labels := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	values := make([]int, len(keys))
	for i, k := range keys {
		v, ok := anyToInt(raw[k])
		if !ok {
			return ""
		}
		values[i] = v
	}
	b := &strings.Builder{}
	fmt.Fprintf(b, "%s  %s  %s  %s  %s  %s\n", labels[0], labels[1], labels[2], labels[3], labels[4], labels[5])
	for i, v := range values {
		if i > 0 {
			b.WriteString("  ")
		}
		mod := (v / 2) - 5
		fmt.Fprintf(b, "%2d (%+d)", v, mod)
	}
	return b.String()
}

func plainSection(v any) string {
	items, ok := v.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	lines := make([]string, 0, len(items))
	for _, it := range items {
		switch x := it.(type) {
		case map[string]any:
			name := strings.TrimSpace(asString(x["name"]))
			body := strings.TrimSpace(plainAny(x["entries"]))
			if name != "" && body != "" {
				lines = append(lines, fmt.Sprintf("%s. %s", name, body))
			} else if name != "" {
				lines = append(lines, name)
			} else if body != "" {
				lines = append(lines, body)
			}
		default:
			txt := strings.TrimSpace(plainAny(it))
			if txt != "" {
				lines = append(lines, txt)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func plainAny(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case int, int8, int16, int32, int64, float32, float64, bool:
		return strings.TrimSpace(fmt.Sprintf("%v", x))
	case []string:
		return strings.Join(x, ", ")
	case []any:
		out := make([]string, 0, len(x))
		for _, it := range x {
			txt := strings.TrimSpace(plainAny(it))
			if txt != "" {
				out = append(out, txt)
			}
		}
		return strings.Join(out, ", ")
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			txt := strings.TrimSpace(plainAny(x[k]))
			if txt != "" {
				pairs = append(pairs, fmt.Sprintf("%s %s", k, txt))
			}
		}
		return strings.Join(pairs, ", ")
	case map[any]any:
		tmp := make(map[string]any, len(x))
		for k, vv := range x {
			tmp[asString(k)] = vv
		}
		return plainAny(tmp)
	default:
		return strings.TrimSpace(asString(v))
	}
}

func (ui *UI) openRawSearch(returnFocus tview.Primitive) {
	input := tview.NewInputField().
		SetLabel("/ ").
		SetFieldWidth(40)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Find In Description ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(ui.rawQuery)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 52, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("raw-search")
		ui.app.SetFocus(returnFocus)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		query := strings.TrimSpace(input.GetText())
		if query == "" {
			ui.rawQuery = ""
			ui.renderRawWithHighlight("", -1)
			ui.status.SetText(helpText)
			return
		}
		line, ok := ui.findRawMatch(query)
		if !ok {
			ui.rawQuery = query
			ui.renderRawWithHighlight(query, -1)
			ui.status.SetText(fmt.Sprintf(" [white:red] nessun match nella Description [-:-] \"%s\"  %s", query, helpText))
			return
		}
		ui.rawQuery = query
		ui.renderRawWithHighlight(query, line)
		ui.detailRaw.ScrollTo(line, 0)
		ui.status.SetText(fmt.Sprintf(" [black:gold] trovato nella Description[-:-] \"%s\" (riga %d)  %s", query, line+1, helpText))
	})

	ui.pages.AddPage("raw-search", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openEncounterSaveAsInput() {
	input := tview.NewInputField().
		SetLabel("File: ").
		SetFieldWidth(52)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Save Encounters As ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(ui.encountersPath)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("encounter-saveas")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] nome file non valido[-:-]  %s", helpText))
			return
		}
		if err := ui.saveEncountersAs(path); err != nil {
			ui.status.SetText(fmt.Sprintf(" [white:red] errore save encounters[-:-] %v  %s", err, helpText))
			return
		}
		ui.status.SetText(fmt.Sprintf(" [black:gold] salvato[-:-] %s  %s", ui.encountersPath, helpText))
	})

	ui.pages.AddPage("encounter-saveas", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) openEncounterLoadInput() {
	input := tview.NewInputField().
		SetLabel("File: ").
		SetFieldWidth(52)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	input.SetTitle(" Load Encounters ")
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)
	input.SetText(ui.encountersPath)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 72, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("encounter-load")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}
		path := strings.TrimSpace(input.GetText())
		if path == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] nome file non valido[-:-]  %s", helpText))
			return
		}

		prev := ui.encountersPath
		ui.encountersPath = path
		if err := ui.loadEncounters(); err != nil {
			ui.encountersPath = prev
			ui.status.SetText(fmt.Sprintf(" [white:red] errore load encounters[-:-] %v  %s", err, helpText))
			return
		}
		ui.encounterUndo = ui.encounterUndo[:0]
		ui.encounterRedo = ui.encounterRedo[:0]
		ui.renderEncounterList()
		if len(ui.encounterItems) > 0 {
			idx := 0
			if ui.turnMode {
				idx = ui.turnIndex
			}
			if idx < 0 || idx >= len(ui.encounterItems) {
				idx = 0
			}
			ui.encounter.SetCurrentItem(idx)
			ui.renderDetailByEncounterIndex(idx)
		} else {
			ui.detailMeta.SetText("Nessun mostro nell'encounter.")
			ui.detailRaw.SetText("")
			ui.rawText = ""
		}
		_ = writeLastEncountersPath(ui.encountersPath)
		ui.status.SetText(fmt.Sprintf(" [black:gold] caricato[-:-] %s  %s", ui.encountersPath, helpText))
	})

	ui.pages.AddPage("encounter-load", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) addSelectedMonsterToEncounter() {
	if ui.browseMode != BrowseMonsters {
		ui.status.SetText(fmt.Sprintf(" [white:red] aggiunta encounter disponibile solo da Monsters[-:-]  %s", helpText))
		return
	}
	if len(ui.filtered) == 0 {
		return
	}

	listIndex := ui.list.GetCurrentItem()
	if listIndex < 0 || listIndex >= len(ui.filtered) {
		return
	}

	monsterIndex := ui.filtered[listIndex]
	ui.pushEncounterUndo()
	ui.encounterSerial[monsterIndex]++
	ordinal := ui.encounterSerial[monsterIndex]
	baseHP, ok := extractHPAverageInt(ui.monsters[monsterIndex].Raw)
	if !ok {
		baseHP = 0
	}
	_, hpFormula := extractHP(ui.monsters[monsterIndex].Raw)
	ui.encounterItems = append(ui.encounterItems, EncounterEntry{
		MonsterIndex: monsterIndex,
		Ordinal:      ordinal,
		BaseHP:       baseHP,
		CurrentHP:    baseHP,
		HPFormula:    hpFormula,
		UseRolledHP:  false,
		RolledHP:     0,
		HasInitRoll:  false,
		InitRoll:     0,
	})
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(len(ui.encounterItems) - 1)

	m := ui.monsters[monsterIndex]
	ui.status.SetText(fmt.Sprintf(" [black:gold] aggiunto[-:-] %s #%d  %s", m.Name, ordinal, helpText))
}

func (ui *UI) openAddCustomEncounterForm() {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Add Custom Encounter ")
	form.SetBorderColor(tcell.ColorGold)
	form.SetTitleColor(tcell.ColorGold)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		}
		return event
	})
	ui.addCustomVisible = true

	nameField := tview.NewInputField().SetLabel("Name: ").SetFieldWidth(28)
	initField := tview.NewInputField().SetLabel("Init (x or x/x): ").SetFieldWidth(16)
	hpField := tview.NewInputField().SetLabel("HP (z or x/y): ").SetFieldWidth(16)
	acField := tview.NewInputField().SetLabel("AC (optional): ").SetFieldWidth(8)

	setFieldStyle := func(f *tview.InputField) {
		f.SetLabelColor(tcell.ColorGold)
		f.SetFieldBackgroundColor(tcell.ColorWhite)
		f.SetFieldTextColor(tcell.ColorBlack)
		f.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	}
	setFieldStyle(nameField)
	setFieldStyle(initField)
	setFieldStyle(hpField)
	setFieldStyle(acField)

	form.AddFormItem(nameField)
	form.AddFormItem(initField)
	form.AddFormItem(hpField)
	form.AddFormItem(acField)

	form.AddButton("Save", func() {
		name := strings.TrimSpace(nameField.GetText())
		if name == "" {
			ui.status.SetText(fmt.Sprintf(" [white:red] nome non valido[-:-]  %s", helpText))
			return
		}

		hasRoll, initRoll, initBase, ok := parseInitInput(initField.GetText())
		if !ok {
			ui.status.SetText(fmt.Sprintf(" [white:red] init non valida[-:-]  %s", helpText))
			return
		}

		currentHP, maxHP, ok := parseHPInput(hpField.GetText())
		if !ok {
			ui.status.SetText(fmt.Sprintf(" [white:red] HP non validi[-:-]  %s", helpText))
			return
		}

		ac := strings.TrimSpace(acField.GetText())
		if ac != "" {
			if _, err := strconv.Atoi(ac); err != nil {
				ui.status.SetText(fmt.Sprintf(" [white:red] AC non valida[-:-]  %s", helpText))
				return
			}
		}

		ui.pushEncounterUndo()
		ordinal := ui.nextCustomOrdinal(name)
		ui.encounterItems = append(ui.encounterItems, EncounterEntry{
			MonsterIndex: -1,
			Ordinal:      ordinal,
			Custom:       true,
			CustomName:   name,
			CustomInit:   initBase,
			CustomAC:     ac,
			BaseHP:       maxHP,
			CurrentHP:    currentHP,
			HasInitRoll:  hasRoll,
			InitRoll:     initRoll,
		})

		ui.pages.RemovePage("encounter-add-custom")
		ui.addCustomVisible = false
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(len(ui.encounterItems) - 1)
		ui.renderDetailByEncounterIndex(len(ui.encounterItems) - 1)
		ui.app.SetFocus(ui.encounter)
		ui.status.SetText(fmt.Sprintf(" [black:gold] aggiunta[-:-] entry custom %s  %s", name, helpText))
	})
	form.AddButton("Cancel", func() {
		ui.pages.RemovePage("encounter-add-custom")
		ui.addCustomVisible = false
		ui.app.SetFocus(ui.encounter)
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 12, 0, true).
			AddItem(nil, 0, 1, false), 74, 0, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("encounter-add-custom", modal, true, true)
	ui.app.SetFocus(form)
}

func (ui *UI) renderEncounterList() {
	ui.encounter.Clear()
	if len(ui.encounterItems) == 0 {
		ui.turnMode = false
		ui.encounter.AddItem("Nessun mostro nell'encounter", "", 0, nil)
		return
	}
	if ui.turnMode {
		if ui.turnRound <= 0 {
			ui.turnRound = 1
		}
		if ui.turnIndex < 0 {
			ui.turnIndex = 0
		}
		if ui.turnIndex >= len(ui.encounterItems) {
			ui.turnIndex = 0
		}
	}

	for i, item := range ui.encounterItems {
		label := ui.encounterEntryDisplay(item)
		if init, ok := ui.encounterInitBase(item); ok {
			if item.HasInitRoll {
				label = fmt.Sprintf("%s [Init %d/%d]", label, item.InitRoll, init)
			} else {
				label = fmt.Sprintf("%s [Init %d]", label, init)
			}
		}
		maxHP := ui.encounterMaxHP(item)
		if maxHP > 0 {
			if item.CurrentHP <= 0 {
				label = "X " + label
			}
			label = fmt.Sprintf("%s [HP %d/%d]", label, item.CurrentHP, maxHP)
		} else {
			label = fmt.Sprintf("%s [HP ?]", label)
		}
		if ui.turnMode {
			prefix := fmt.Sprintf("%d", i+1)
			if i == ui.turnIndex {
				prefix += fmt.Sprintf("*[%d]", ui.turnRound)
			}
			label = prefix + " " + label
		}
		ui.encounter.AddItem(label, "", 0, nil)
	}
}

func (ui *UI) openEncounterHPInput(direction int) {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]
	if entry.BaseHP <= 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] HP non disponibile per %s[-:-]  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}
	if direction == 0 {
		return
	}

	input := tview.NewInputField().
		SetLabel("HP ").
		SetFieldWidth(12)
	input.SetLabelColor(tcell.ColorGold)
	input.SetFieldBackgroundColor(tcell.ColorWhite)
	input.SetFieldTextColor(tcell.ColorBlack)
	input.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	input.SetBackgroundColor(tcell.ColorBlack)
	input.SetBorder(true)
	if direction < 0 {
		input.SetTitle(" Damage Encounter ")
	} else {
		input.SetTitle(" Heal Encounter ")
	}
	input.SetBorderColor(tcell.ColorGold)
	input.SetTitleColor(tcell.ColorGold)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(input, 3, 0, true).
			AddItem(nil, 0, 1, false), 40, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		ui.pages.RemovePage("encounter-damage")
		ui.app.SetFocus(ui.encounter)
		if key == tcell.KeyEscape || key != tcell.KeyEnter {
			return
		}

		text := strings.TrimSpace(input.GetText())
		damage, err := strconv.Atoi(text)
		if err != nil || damage <= 0 {
			ui.status.SetText(fmt.Sprintf(" [white:red] valore danno non valido[-:-] \"%s\"  %s", text, helpText))
			return
		}

		ui.pushEncounterUndo()
		if direction < 0 {
			ui.encounterItems[index].CurrentHP -= damage
			if ui.encounterItems[index].CurrentHP < 0 {
				ui.encounterItems[index].CurrentHP = 0
			}
		} else {
			ui.encounterItems[index].CurrentHP += damage
			maxHP := ui.encounterMaxHP(ui.encounterItems[index])
			if maxHP > 0 && ui.encounterItems[index].CurrentHP > maxHP {
				ui.encounterItems[index].CurrentHP = maxHP
			}
		}
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)

		if direction < 0 {
			ui.status.SetText(fmt.Sprintf(" [black:gold] danno[-:-] %s -%d HP (%d/%d)  %s",
				ui.encounterEntryDisplay(ui.encounterItems[index]),
				damage,
				ui.encounterItems[index].CurrentHP,
				ui.encounterMaxHP(ui.encounterItems[index]),
				helpText,
			))
		} else {
			ui.status.SetText(fmt.Sprintf(" [black:gold] cura[-:-] %s +%d HP (%d/%d)  %s",
				ui.encounterEntryDisplay(ui.encounterItems[index]),
				damage,
				ui.encounterItems[index].CurrentHP,
				ui.encounterMaxHP(ui.encounterItems[index]),
				helpText,
			))
		}
	})

	ui.pages.AddPage("encounter-damage", modal, true, true)
	ui.app.SetFocus(input)
}

func (ui *UI) deleteSelectedEncounterEntry() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}
	entry := ui.encounterItems[index]

	ui.pushEncounterUndo()
	ui.encounterItems = append(ui.encounterItems[:index], ui.encounterItems[index+1:]...)
	if ui.turnMode {
		if len(ui.encounterItems) == 0 {
			ui.turnMode = false
			ui.turnIndex = 0
			ui.turnRound = 0
		} else {
			if index < ui.turnIndex {
				ui.turnIndex--
			}
			if ui.turnIndex >= len(ui.encounterItems) {
				ui.turnIndex = 0
			}
			if ui.turnRound <= 0 {
				ui.turnRound = 1
			}
		}
	}
	ui.renderEncounterList()
	if len(ui.encounterItems) > 0 {
		if index >= len(ui.encounterItems) {
			index = len(ui.encounterItems) - 1
		}
		ui.encounter.SetCurrentItem(index)
		ui.renderDetailByEncounterIndex(index)
	} else {
		ui.detailMeta.SetText("Nessun mostro nell'encounter.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold] eliminato[-:-] %s  %s", ui.encounterEntryDisplay(entry), helpText))
}

func (ui *UI) toggleEncounterHPMode() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}

	entry := ui.encounterItems[index]
	if strings.TrimSpace(entry.HPFormula) == "" {
		ui.status.SetText(fmt.Sprintf(" [white:red] formula HP non disponibile per %s[-:-]  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}

	if entry.UseRolledHP {
		ui.pushEncounterUndo()
		ui.encounterItems[index].UseRolledHP = false
		maxHP := ui.encounterMaxHP(ui.encounterItems[index])
		if maxHP > 0 && ui.encounterItems[index].CurrentHP > maxHP {
			ui.encounterItems[index].CurrentHP = maxHP
		}
		ui.renderEncounterList()
		ui.encounter.SetCurrentItem(index)
		ui.status.SetText(fmt.Sprintf(" [black:gold] hp mode[-:-] %s -> average  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}

	rolled, ok := rollHPFormula(entry.HPFormula)
	if !ok {
		ui.status.SetText(fmt.Sprintf(" [white:red] formula HP non supportata[-:-] \"%s\"  %s", entry.HPFormula, helpText))
		return
	}
	ui.pushEncounterUndo()
	ui.encounterItems[index].UseRolledHP = true
	ui.encounterItems[index].RolledHP = rolled
	maxHP := ui.encounterMaxHP(ui.encounterItems[index])
	if maxHP > 0 && ui.encounterItems[index].CurrentHP > maxHP {
		ui.encounterItems[index].CurrentHP = maxHP
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold] hp mode[-:-] %s -> formula (%s = %d)  %s", ui.encounterEntryDisplay(entry), entry.HPFormula, rolled, helpText))
}

func (ui *UI) rollEncounterInitiative() {
	if len(ui.encounterItems) == 0 {
		return
	}
	index := ui.encounter.GetCurrentItem()
	if index < 0 || index >= len(ui.encounterItems) {
		return
	}

	entry := ui.encounterItems[index]
	initValue, ok := ui.encounterInitBase(entry)
	if !ok {
		ui.status.SetText(fmt.Sprintf(" [white:red] init non disponibile per %s[-:-]  %s", ui.encounterEntryDisplay(entry), helpText))
		return
	}

	ui.pushEncounterUndo()
	roll := (rand.Intn(20) + 1) + initValue
	ui.encounterItems[index].HasInitRoll = true
	ui.encounterItems[index].InitRoll = roll
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(index)
	ui.renderDetailByEncounterIndex(index)
	ui.status.SetText(fmt.Sprintf(" [black:gold] initiative[-:-] %s = %d/%d  %s", ui.encounterEntryDisplay(entry), roll, initValue, helpText))
}

func (ui *UI) rollAllEncounterInitiative() {
	if len(ui.encounterItems) == 0 {
		return
	}

	ui.pushEncounterUndo()
	rolledCount := 0
	for i := range ui.encounterItems {
		entry := ui.encounterItems[i]
		initValue, ok := ui.encounterInitBase(entry)
		if !ok {
			continue
		}
		ui.encounterItems[i].HasInitRoll = true
		ui.encounterItems[i].InitRoll = (rand.Intn(20) + 1) + initValue
		rolledCount++
	}

	ui.renderEncounterList()
	if len(ui.encounterItems) > 0 {
		idx := ui.encounter.GetCurrentItem()
		if idx < 0 {
			idx = 0
		}
		if idx >= len(ui.encounterItems) {
			idx = len(ui.encounterItems) - 1
		}
		ui.encounter.SetCurrentItem(idx)
		ui.renderDetailByEncounterIndex(idx)
	}

	if rolledCount == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] nessuna entry con dex disponibile[-:-]  %s", helpText))
		return
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold] initiative[-:-] tirata per %d entry  %s", rolledCount, helpText))
}

func (ui *UI) sortEncounterByInitiative() {
	if len(ui.encounterItems) < 2 {
		return
	}

	current := ui.encounter.GetCurrentItem()
	if current < 0 || current >= len(ui.encounterItems) {
		current = 0
	}
	selected := ui.encounterItems[current]
	active := EncounterEntry{}
	hasActive := false
	if ui.turnMode && ui.turnIndex >= 0 && ui.turnIndex < len(ui.encounterItems) {
		active = ui.encounterItems[ui.turnIndex]
		hasActive = true
	}

	ui.pushEncounterUndo()

	sort.SliceStable(ui.encounterItems, func(i, j int) bool {
		a := ui.encounterItems[i]
		b := ui.encounterItems[j]

		if a.HasInitRoll != b.HasInitRoll {
			return a.HasInitRoll
		}
		if a.HasInitRoll && b.HasInitRoll && a.InitRoll != b.InitRoll {
			return a.InitRoll > b.InitRoll
		}

		aInit, aok := ui.encounterInitBase(a)
		bInit, bok := ui.encounterInitBase(b)
		if aok != bok {
			return aok
		}
		if aok && bok && aInit != bInit {
			return aInit > bInit
		}

		an := ui.encounterEntryName(a)
		bn := ui.encounterEntryName(b)
		if strings.ToLower(an) != strings.ToLower(bn) {
			return strings.ToLower(an) < strings.ToLower(bn)
		}
		return a.Ordinal < b.Ordinal
	})

	ui.renderEncounterList()

	newIndex := 0
	newTurnIndex := -1
	for i, it := range ui.encounterItems {
		if it.MonsterIndex == selected.MonsterIndex && it.Ordinal == selected.Ordinal {
			newIndex = i
		}
		if hasActive && it.MonsterIndex == active.MonsterIndex && it.Ordinal == active.Ordinal {
			newTurnIndex = i
		}
	}
	if ui.turnMode && hasActive && newTurnIndex >= 0 {
		ui.turnIndex = newTurnIndex
	}
	ui.encounter.SetCurrentItem(newIndex)
	ui.renderDetailByEncounterIndex(newIndex)
	ui.status.SetText(fmt.Sprintf(" [black:gold] sort[-:-] encounters ordinati per iniziativa  %s", helpText))
}

func (ui *UI) pushEncounterUndo() {
	snap := EncounterUndoState{
		Items:    append([]EncounterEntry(nil), ui.encounterItems...),
		Serial:   cloneIntMap(ui.encounterSerial),
		Selected: ui.encounter.GetCurrentItem(),
	}
	ui.encounterUndo = append(ui.encounterUndo, snap)
	ui.encounterRedo = ui.encounterRedo[:0]
}

func (ui *UI) undoEncounterCommand() {
	if len(ui.encounterUndo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] nessuna operazione da annullare[-:-]  %s", helpText))
		return
	}
	current := ui.captureEncounterState()
	last := ui.encounterUndo[len(ui.encounterUndo)-1]
	ui.encounterUndo = ui.encounterUndo[:len(ui.encounterUndo)-1]
	ui.encounterRedo = append(ui.encounterRedo, current)
	ui.restoreEncounterState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] undo[-:-] operazione encounter annullata  %s", helpText))
}

func (ui *UI) redoEncounterCommand() {
	if len(ui.encounterRedo) == 0 {
		ui.status.SetText(fmt.Sprintf(" [white:red] nessuna operazione da ripristinare[-:-]  %s", helpText))
		return
	}
	current := ui.captureEncounterState()
	last := ui.encounterRedo[len(ui.encounterRedo)-1]
	ui.encounterRedo = ui.encounterRedo[:len(ui.encounterRedo)-1]
	ui.encounterUndo = append(ui.encounterUndo, current)
	ui.restoreEncounterState(last)
	ui.status.SetText(fmt.Sprintf(" [black:gold] redo[-:-] operazione encounter ripristinata  %s", helpText))
}

func (ui *UI) captureEncounterState() EncounterUndoState {
	return EncounterUndoState{
		Items:    append([]EncounterEntry(nil), ui.encounterItems...),
		Serial:   cloneIntMap(ui.encounterSerial),
		Selected: ui.encounter.GetCurrentItem(),
	}
}

func (ui *UI) restoreEncounterState(state EncounterUndoState) {
	ui.encounterItems = append([]EncounterEntry(nil), state.Items...)
	ui.encounterSerial = cloneIntMap(state.Serial)
	ui.renderEncounterList()

	if len(ui.encounterItems) > 0 {
		idx := state.Selected
		if idx < 0 {
			idx = 0
		}
		if idx >= len(ui.encounterItems) {
			idx = len(ui.encounterItems) - 1
		}
		ui.encounter.SetCurrentItem(idx)
		ui.renderDetailByEncounterIndex(idx)
		return
	}

	ui.detailMeta.SetText("Nessun mostro nell'encounter.")
	ui.detailRaw.SetText("")
	ui.rawText = ""
}

func (ui *UI) toggleEncounterTurnMode() {
	if len(ui.encounterItems) == 0 {
		return
	}
	if ui.turnMode {
		ui.turnMode = false
		ui.turnRound = 0
		ui.renderEncounterList()
		idx := ui.encounter.GetCurrentItem()
		if idx < 0 {
			idx = 0
		}
		ui.encounter.SetCurrentItem(idx)
		ui.renderDetailByEncounterIndex(idx)
		ui.status.SetText(fmt.Sprintf(" [black:gold] turn mode[-:-] disattivato  %s", helpText))
		return
	}
	idx := ui.findTopInitiativeEncounterIndex()
	ui.turnMode = true
	ui.turnIndex = idx
	ui.turnRound = 1
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(idx)
	ui.renderDetailByEncounterIndex(idx)
	ui.status.SetText(fmt.Sprintf(" [black:gold] turn mode[-:-] attivo (round 1)  %s", helpText))
}

func (ui *UI) findTopInitiativeEncounterIndex() int {
	if len(ui.encounterItems) == 0 {
		return 0
	}
	best := 0
	bestVal := -1 << 30
	bestHas := false
	for i, e := range ui.encounterItems {
		v, ok := ui.encounterInitBase(e)
		if e.HasInitRoll {
			v = e.InitRoll
			ok = true
		}
		if !ok {
			continue
		}
		if !bestHas || v > bestVal {
			bestHas = true
			bestVal = v
			best = i
		}
	}
	if bestHas {
		return best
	}
	return 0
}

func (ui *UI) nextEncounterTurn() {
	if !ui.turnMode || len(ui.encounterItems) == 0 {
		return
	}
	if ui.turnIndex >= len(ui.encounterItems)-1 {
		ui.turnIndex = 0
		ui.turnRound++
		if ui.turnRound <= 0 {
			ui.turnRound = 1
		}
	} else {
		ui.turnIndex++
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(ui.turnIndex)
	ui.renderDetailByEncounterIndex(ui.turnIndex)
	ui.status.SetText(fmt.Sprintf(" [black:gold] turn[-:-] round %d, entry %d  %s", ui.turnRound, ui.turnIndex+1, helpText))
}

func (ui *UI) prevEncounterTurn() {
	if !ui.turnMode || len(ui.encounterItems) == 0 {
		return
	}
	if ui.turnIndex <= 0 {
		ui.turnIndex = len(ui.encounterItems) - 1
		if ui.turnRound > 1 {
			ui.turnRound--
		}
	} else {
		ui.turnIndex--
	}
	ui.renderEncounterList()
	ui.encounter.SetCurrentItem(ui.turnIndex)
	ui.renderDetailByEncounterIndex(ui.turnIndex)
	ui.status.SetText(fmt.Sprintf(" [black:gold] turn[-:-] round %d, entry %d  %s", ui.turnRound, ui.turnIndex+1, helpText))
}

func (ui *UI) loadEncounters() error {
	b, err := os.ReadFile(ui.encountersPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var data PersistedEncounters
	if err := yaml.Unmarshal(b, &data); err != nil {
		return err
	}

	idToIndex := make(map[int]int, len(ui.monsters))
	for i, m := range ui.monsters {
		idToIndex[m.ID] = i
	}

	ui.encounterItems = ui.encounterItems[:0]
	ui.encounterSerial = map[int]int{}
	ui.turnMode = false
	ui.turnIndex = 0
	ui.turnRound = 0

	for _, it := range data.Items {
		monsterIndex := -1
		if !it.Custom {
			var ok bool
			monsterIndex, ok = idToIndex[it.MonsterID]
			if !ok {
				continue
			}
		}

		ordinal := it.Ordinal
		if ordinal <= 0 {
			if it.Custom {
				maxOrd := 0
				for _, e := range ui.encounterItems {
					if e.Custom && strings.EqualFold(strings.TrimSpace(e.CustomName), strings.TrimSpace(it.CustomName)) && e.Ordinal > maxOrd {
						maxOrd = e.Ordinal
					}
				}
				ordinal = maxOrd + 1
			} else {
				ordinal = ui.encounterSerial[monsterIndex] + 1
			}
		}

		baseHP := it.BaseHP
		hpFormula := strings.TrimSpace(it.HPFormula)
		if !it.Custom {
			if baseHP <= 0 {
				if avg, ok := extractHPAverageInt(ui.monsters[monsterIndex].Raw); ok {
					baseHP = avg
				}
			}
			if hpFormula == "" {
				_, hpFormula = extractHP(ui.monsters[monsterIndex].Raw)
			}
		}

		currentHP := it.CurrentHP
		maxHP := baseHP
		if it.UseRolled && it.RolledHP > 0 {
			maxHP = it.RolledHP
		}
		if currentHP < 0 {
			currentHP = 0
		}
		if maxHP > 0 && currentHP > maxHP {
			currentHP = maxHP
		}

		entry := EncounterEntry{
			MonsterIndex: monsterIndex,
			Ordinal:      ordinal,
			Custom:       it.Custom,
			CustomName:   it.CustomName,
			CustomInit:   it.CustomInit,
			CustomAC:     it.CustomAC,
			BaseHP:       baseHP,
			CurrentHP:    currentHP,
			HPFormula:    hpFormula,
			UseRolledHP:  it.UseRolled,
			RolledHP:     it.RolledHP,
			HasInitRoll:  it.InitRolled,
			InitRoll:     it.InitRoll,
		}
		ui.encounterItems = append(ui.encounterItems, entry)
		if !it.Custom && ordinal > ui.encounterSerial[monsterIndex] {
			ui.encounterSerial[monsterIndex] = ordinal
		}
	}
	if len(ui.encounterItems) > 0 {
		ui.turnMode = data.TurnMode
		ui.turnIndex = data.TurnIndex
		ui.turnRound = data.TurnRound
		if ui.turnRound <= 0 {
			ui.turnRound = 1
		}
		if ui.turnIndex < 0 || ui.turnIndex >= len(ui.encounterItems) {
			ui.turnIndex = 0
		}
	}
	return nil
}

func (ui *UI) saveEncounters() error {
	data := PersistedEncounters{
		Version:   1,
		Items:     make([]PersistedEncounterItem, 0, len(ui.encounterItems)),
		TurnMode:  ui.turnMode,
		TurnIndex: ui.turnIndex,
		TurnRound: ui.turnRound,
	}

	for _, it := range ui.encounterItems {
		item := PersistedEncounterItem{
			Ordinal:    it.Ordinal,
			Custom:     it.Custom,
			CustomName: it.CustomName,
			CustomInit: it.CustomInit,
			CustomAC:   it.CustomAC,
			BaseHP:     it.BaseHP,
			CurrentHP:  it.CurrentHP,
			HPFormula:  it.HPFormula,
			UseRolled:  it.UseRolledHP,
			RolledHP:   it.RolledHP,
			InitRolled: it.HasInitRoll,
			InitRoll:   it.InitRoll,
		}
		if !it.Custom {
			if it.MonsterIndex < 0 || it.MonsterIndex >= len(ui.monsters) {
				continue
			}
			item.MonsterID = ui.monsters[it.MonsterIndex].ID
		}
		data.Items = append(data.Items, item)
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(ui.encountersPath, out, 0o644); err != nil {
		return err
	}
	_ = writeLastEncountersPath(ui.encountersPath)
	return nil
}

func (ui *UI) saveEncountersAs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	prev := ui.encountersPath
	ui.encountersPath = path
	if err := ui.saveEncounters(); err != nil {
		ui.encountersPath = prev
		return err
	}
	return nil
}

func (ui *UI) loadDiceResults() error {
	b, err := os.ReadFile(ui.dicePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ui.diceLog = nil
			ui.renderDiceList()
			return nil
		}
		return err
	}
	var data PersistedDice
	if err := yaml.Unmarshal(b, &data); err != nil {
		// Backward compatibility: old format had items as []string.
		var legacy struct {
			Version int      `yaml:"version"`
			Items   []string `yaml:"items"`
		}
		if legacyErr := yaml.Unmarshal(b, &legacy); legacyErr != nil {
			return err
		}
		data.Version = legacy.Version
		data.Items = make([]DiceResult, 0, len(legacy.Items))
		for _, it := range legacy.Items {
			text := strings.TrimSpace(it)
			if text == "" {
				continue
			}
			expr := text
			out := ""
			if parts := strings.SplitN(text, "=>", 2); len(parts) == 2 {
				expr = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(parts[0], "[black:gold]", ""), "[-:-]", ""))
				out = strings.TrimSpace(parts[1])
			}
			data.Items = append(data.Items, DiceResult{Expression: expr, Output: out})
		}
	}
	ui.diceLog = append([]DiceResult(nil), data.Items...)
	ui.renderDiceList()
	_ = writeLastDicePath(ui.dicePath)
	return nil
}

func (ui *UI) saveDiceResults() error {
	data := PersistedDice{
		Version: 1,
		Items:   append([]DiceResult(nil), ui.diceLog...),
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(ui.dicePath, out, 0o644); err != nil {
		return err
	}
	_ = writeLastDicePath(ui.dicePath)
	return nil
}

func (ui *UI) saveDiceResultsAs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("empty path")
	}
	prev := ui.dicePath
	ui.dicePath = path
	if err := ui.saveDiceResults(); err != nil {
		ui.dicePath = prev
		return err
	}
	return nil
}

func modeToKey(mode BrowseMode) string {
	switch mode {
	case BrowseItems:
		return "items"
	case BrowseSpells:
		return "spells"
	default:
		return "monsters"
	}
}

func modeFromKey(s string) BrowseMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "items":
		return BrowseItems
	case "spells":
		return BrowseSpells
	default:
		return BrowseMonsters
	}
}

func (ui *UI) loadFilterStates() error {
	b, err := os.ReadFile(filtersStatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var data PersistedFilters
	if err := yaml.Unmarshal(b, &data); err != nil {
		return err
	}
	ui.modeFilters[BrowseMonsters] = data.Monsters
	ui.modeFilters[BrowseItems] = data.Items
	ui.modeFilters[BrowseSpells] = data.Spells
	ui.browseMode = modeFromKey(data.Active)
	return nil
}

func (ui *UI) saveFilterStates() error {
	ui.saveCurrentModeFilters()
	data := PersistedFilters{
		Version:  1,
		Active:   modeToKey(ui.browseMode),
		Monsters: ui.modeFilters[BrowseMonsters],
		Items:    ui.modeFilters[BrowseItems],
		Spells:   ui.modeFilters[BrowseSpells],
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(filtersStatePath, out, 0o644)
}

func readLastEncountersPath() string {
	b, err := os.ReadFile(lastEncountersPathFile)
	if err != nil {
		return defaultEncountersPath
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return defaultEncountersPath
	}
	return p
}

func writeLastEncountersPath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	dir := filepath.Dir(lastEncountersPathFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(lastEncountersPathFile, []byte(p+"\n"), 0o644)
}

func readLastDicePath() string {
	b, err := os.ReadFile(lastDicePathFile)
	if err != nil {
		return defaultDicePath
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return defaultDicePath
	}
	return p
}

func writeLastDicePath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	dir := filepath.Dir(lastDicePathFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(lastDicePathFile, []byte(p+"\n"), 0o644)
}

func (ui *UI) renderRawWithHighlight(query string, lineToHighlight int) {
	if ui.rawText == "" {
		ui.detailRaw.SetText("")
		return
	}

	lines := strings.Split(ui.rawText, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if query != "" && i == lineToHighlight {
			b.WriteString(highlightEscaped(line, query))
		} else {
			b.WriteString(tview.Escape(line))
		}
	}
	ui.detailRaw.SetText(b.String())
}

func highlightEscaped(line, query string) string {
	if query == "" {
		return tview.Escape(line)
	}
	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)

	var b strings.Builder
	start := 0
	for {
		idx := strings.Index(lowerLine[start:], lowerQuery)
		if idx < 0 {
			b.WriteString(tview.Escape(line[start:]))
			break
		}
		abs := start + idx
		end := abs + len(query)
		b.WriteString(tview.Escape(line[start:abs]))
		b.WriteString("[black:gold]")
		b.WriteString(tview.Escape(line[abs:end]))
		b.WriteString("[-:-]")
		start = end
		if start >= len(line) {
			break
		}
	}
	return b.String()
}

func (ui *UI) findRawMatch(query string) (int, bool) {
	if strings.TrimSpace(query) == "" || ui.rawText == "" {
		return 0, false
	}
	lines := strings.Split(ui.rawText, "\n")
	if len(lines) == 0 {
		return 0, false
	}

	q := strings.ToLower(query)
	start, _ := ui.detailRaw.GetScrollOffset()
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		start = len(lines) - 1
	}

	for i := start + 1; i < len(lines); i++ {
		if strings.Contains(strings.ToLower(lines[i]), q) {
			return i, true
		}
	}
	for i := 0; i <= start && i < len(lines); i++ {
		if strings.Contains(strings.ToLower(lines[i]), q) {
			return i, true
		}
	}
	return 0, false
}

func loadMonstersFromPath(path string) ([]Monster, []string, []string, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return loadMonstersFromBytes(b)
}

func loadMonstersFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds dataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Monsters) == 0 {
		return nil, nil, nil, nil, errors.New("nessun mostro trovato nel yaml")
	}

	monsters := make([]Monster, 0, len(ds.Monsters))
	envSet := map[string]struct{}{}
	crSet := map[string]struct{}{}
	typeSet := map[string]struct{}{}

	for i, raw := range ds.Monsters {
		name := asString(raw["name"])
		if name == "" {
			continue
		}

		envs := asStringSlice(raw["environment"])
		for _, env := range envs {
			envSet[env] = struct{}{}
		}

		cr := extractCR(raw["cr"])
		if cr == "" {
			cr = "Unknown"
		}
		crSet[cr] = struct{}{}

		monsters = append(monsters, Monster{
			ID:          i,
			Name:        name,
			CR:          cr,
			Environment: envs,
			Source:      asString(raw["source"]),
			Type:        extractType(raw["type"]),
			Raw:         raw,
		})
		mType := extractType(raw["type"])
		if mType == "" {
			mType = "Unknown"
		}
		typeSet[mType] = struct{}{}
	}

	sort.Slice(monsters, func(i, j int) bool {
		return strings.ToLower(monsters[i].Name) < strings.ToLower(monsters[j].Name)
	})

	return monsters, keysSorted(envSet), sortCR(keysSorted(crSet)), keysSorted(typeSet), nil
}

func loadItemsFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds itemsDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Items) == 0 {
		return nil, nil, nil, nil, errors.New("nessun item trovato nel yaml")
	}

	items := make([]Monster, 0, len(ds.Items))
	envSet := map[string]struct{}{}
	crSet := map[string]struct{}{}
	typeSet := map[string]struct{}{}

	for i, raw := range ds.Items {
		name := asString(raw["name"])
		if name == "" {
			continue
		}
		source := asString(raw["source"])
		envs := []string{}
		if source != "" {
			envs = []string{source}
			envSet[source] = struct{}{}
		}
		rarity := strings.TrimSpace(asString(raw["rarity"]))
		if rarity == "" {
			rarity = "Unknown"
		}
		crSet[rarity] = struct{}{}

		itemType := extractItemType(raw)
		if itemType == "" {
			itemType = "Unknown"
		}
		typeSet[itemType] = struct{}{}

		items = append(items, Monster{
			ID:          i,
			Name:        name,
			CR:          rarity,
			Environment: envs,
			Source:      source,
			Type:        itemType,
			Raw:         raw,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, keysSorted(envSet), keysSorted(crSet), keysSorted(typeSet), nil
}

func loadSpellsFromBytes(b []byte) ([]Monster, []string, []string, []string, error) {
	var ds spellsDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, nil, err
	}
	if len(ds.Spells) == 0 {
		return nil, nil, nil, nil, errors.New("nessuna spell trovata nel yaml")
	}

	spells := make([]Monster, 0, len(ds.Spells))
	envSet := map[string]struct{}{}
	crSet := map[string]struct{}{}
	typeSet := map[string]struct{}{}

	for i, raw := range ds.Spells {
		name := asString(raw["name"])
		if name == "" {
			continue
		}
		source := asString(raw["source"])
		envs := []string{}
		if source != "" {
			envs = []string{source}
			envSet[source] = struct{}{}
		}
		level := extractSpellLevel(raw["level"])
		if level == "" {
			level = "Unknown"
		}
		crSet[level] = struct{}{}
		school := extractSpellSchool(raw["school"])
		if school == "" {
			school = "Unknown"
		}
		typeSet[school] = struct{}{}

		spells = append(spells, Monster{
			ID:          i,
			Name:        name,
			CR:          level,
			Environment: envs,
			Source:      source,
			Type:        school,
			Raw:         raw,
		})
	}

	sort.Slice(spells, func(i, j int) bool {
		return strings.ToLower(spells[i].Name) < strings.ToLower(spells[j].Name)
	})
	return spells, keysSorted(envSet), sortCR(keysSorted(crSet)), keysSorted(typeSet), nil
}

func matchName(monsterName, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(monsterName), strings.ToLower(query))
}

func matchCR(monsterCR, query string) bool {
	if query == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(monsterCR), strings.TrimSpace(query))
}

func matchEnvMulti(values []string, selected map[string]struct{}) bool {
	if len(selected) == 0 {
		return true
	}
	for _, v := range values {
		if _, ok := selected[v]; ok {
			return true
		}
	}
	return false
}

func matchEnv(values []string, query string) bool {
	if strings.TrimSpace(query) == "" {
		return true
	}
	return matchEnvMulti(values, map[string]struct{}{strings.TrimSpace(query): {}})
}

func matchType(monsterType, query string) bool {
	if query == "" {
		return true
	}
	if strings.TrimSpace(monsterType) == "" {
		monsterType = "Unknown"
	}
	return strings.EqualFold(strings.TrimSpace(monsterType), strings.TrimSpace(query))
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return ""
	}
}

func asStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		if one := asString(v); one != "" {
			return []string{one}
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s := asString(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func extractCR(v any) string {
	if s := asString(v); s != "" {
		return s
	}
	switch x := v.(type) {
	case map[string]any:
		return asString(x["cr"])
	case map[any]any:
		return asString(x["cr"])
	}
	return ""
}

func extractType(v any) string {
	if s := asString(v); s != "" {
		return s
	}
	switch x := v.(type) {
	case map[string]any:
		return asString(x["type"])
	case map[any]any:
		return asString(x["type"])
	}
	return ""
}

func extractItemType(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	base := strings.TrimSpace(extractType(raw["type"]))
	flags := make([]string, 0, 3)
	boolFlag := func(key, label string) {
		v, ok := raw[key]
		if !ok {
			return
		}
		b, ok := v.(bool)
		if ok && b {
			flags = append(flags, label)
		}
	}
	boolFlag("wondrous", "wondrous")
	boolFlag("weapon", "weapon")
	boolFlag("armor", "armor")
	boolFlag("staff", "staff")
	boolFlag("ring", "ring")
	boolFlag("potion", "potion")
	boolFlag("wand", "wand")
	boolFlag("rod", "rod")
	boolFlag("scroll", "scroll")

	if base == "" && len(flags) == 0 {
		return ""
	}
	if base == "" {
		return strings.Join(flags, ", ")
	}
	if len(flags) == 0 {
		return base
	}
	return base + " (" + strings.Join(flags, ", ") + ")"
}

type itemEconomyInfo struct {
	BuyCost   string
	FindTime  string
	CraftCost string
	CraftTime string
	Procedure []string
}

func formatItemBasePrice(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	v, ok := raw["value"]
	if !ok || v == nil {
		return ""
	}
	cp, ok := anyToInt64(v)
	if !ok || cp <= 0 {
		return ""
	}
	return formatCopperValue(cp)
}

func magicItemEconomy(raw map[string]any, rarity string) (itemEconomyInfo, bool) {
	if !isMagicalItem(raw, rarity) {
		return itemEconomyInfo{}, false
	}
	key := normalizeRarity(rarity)
	switch key {
	case "common":
		return itemEconomyInfo{
			BuyCost:   "50-100 gp",
			FindTime:  "1d4 giorni",
			CraftCost: "50 gp + componenti",
			CraftTime: "1 workweek",
			Procedure: []string{"trova formula/schemi", "materiali speciali adatti alla rarita", "strumenti/proficienze richieste", "tempo di downtime e spesa del costo"},
		}, true
	case "uncommon":
		return itemEconomyInfo{
			BuyCost:   "101-500 gp",
			FindTime:  "1d6 giorni",
			CraftCost: "200 gp + componenti",
			CraftTime: "2 workweeks",
			Procedure: []string{"formula dell'oggetto", "raccolta ingredienti rari", "proficienza strumenti o Arcana", "craft in downtime"},
		}, true
	case "rare":
		return itemEconomyInfo{
			BuyCost:   "501-5,000 gp",
			FindTime:  "1d4 settimane",
			CraftCost: "2,000 gp + componenti rari",
			CraftTime: "10 workweeks",
			Procedure: []string{"schema/formula completa", "componenti da creature o luoghi speciali", "supporto artigiano o incantatore esperto", "downtime continuativo"},
		}, true
	case "very rare":
		return itemEconomyInfo{
			BuyCost:   "5,001-50,000 gp",
			FindTime:  "1d6 settimane",
			CraftCost: "20,000 gp + componenti molto rari",
			CraftTime: "25 workweeks",
			Procedure: []string{"ricerca avanzata della formula", "quest per materiale chiave", "laboratorio/forgia adeguata", "downtime esteso con verifica DM"},
		}, true
	case "legendary":
		return itemEconomyInfo{
			BuyCost:   "50,001+ gp",
			FindTime:  "2d6 settimane (o piu)",
			CraftCost: "100,000 gp + componenti leggendari",
			CraftTime: "50 workweeks",
			Procedure: []string{"formula unica o perduta", "componenti leggendari ottenuti tramite avventura", "maestria elevata e laboratorio speciale", "craft lungo supervisionato dal DM"},
		}, true
	case "artifact":
		return itemEconomyInfo{
			BuyCost:   "non acquistabile",
			FindTime:  "non disponibile in negozio",
			CraftCost: "non craftabile con regole standard",
			CraftTime: "n/a",
			Procedure: []string{"solo rituali/quest eccezionali", "intervento narrativo del DM", "fonti di potere uniche"},
		}, true
	default:
		return itemEconomyInfo{
			BuyCost:   "variabile (a discrezione DM)",
			FindTime:  "da alcuni giorni a settimane",
			CraftCost: "in base a rarita/effetto",
			CraftTime: "in base a rarita/effetto",
			Procedure: []string{"definisci rarita effettiva", "determina formula e componenti", "applica downtime coerente"},
		}, true
	}
}

func isMagicalItem(raw map[string]any, rarity string) bool {
	key := normalizeRarity(rarity)
	switch key {
	case "common", "uncommon", "rare", "very rare", "legendary", "artifact", "varies":
		return true
	}
	if raw == nil {
		return false
	}
	for _, k := range []string{"wondrous", "staff", "wand", "rod", "ring", "potion", "scroll"} {
		if b, ok := raw[k].(bool); ok && b {
			return true
		}
	}
	return false
}

func normalizeRarity(r string) string {
	s := strings.ToLower(strings.TrimSpace(r))
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func anyToInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int64:
		return x, true
	case float64:
		return int64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return 0, false
		}
		return int64(f), true
	default:
		return 0, false
	}
}

func formatCopperValue(cp int64) string {
	if cp <= 0 {
		return "0 cp"
	}
	pp := cp / 1000
	cp = cp % 1000
	gp := cp / 100
	cp = cp % 100
	sp := cp / 10
	cp = cp % 10
	parts := make([]string, 0, 4)
	if pp > 0 {
		parts = append(parts, fmt.Sprintf("%d pp", pp))
	}
	if gp > 0 {
		parts = append(parts, fmt.Sprintf("%d gp", gp))
	}
	if sp > 0 {
		parts = append(parts, fmt.Sprintf("%d sp", sp))
	}
	if cp > 0 {
		parts = append(parts, fmt.Sprintf("%d cp", cp))
	}
	return strings.Join(parts, " ")
}

func extractSpellLevel(v any) string {
	switch x := v.(type) {
	case int:
		if x == 0 {
			return "0"
		}
		return strconv.Itoa(x)
	case int64:
		if x == 0 {
			return "0"
		}
		return strconv.FormatInt(x, 10)
	case float64:
		i := int(x)
		if i == 0 {
			return "0"
		}
		return strconv.Itoa(i)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return ""
		}
		return s
	default:
		return ""
	}
}

func extractSpellSchool(v any) string {
	code := strings.TrimSpace(asString(v))
	switch strings.ToUpper(code) {
	case "A":
		return "Abjuration"
	case "C":
		return "Conjuration"
	case "D":
		return "Divination"
	case "E":
		return "Enchantment"
	case "V":
		return "Evocation"
	case "I":
		return "Illusion"
	case "N":
		return "Necromancy"
	case "T":
		return "Transmutation"
	default:
		return code
	}
}

func extractSpellRange(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	return plainAny(raw["range"])
}

func extractSpellTime(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	return plainAny(raw["time"])
}

func extractSpellDuration(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	return plainAny(raw["duration"])
}

func extractInitFromDex(raw map[string]any) (int, bool) {
	if raw == nil {
		return 0, false
	}
	dexRaw, ok := raw["dex"]
	if !ok {
		return 0, false
	}
	dex, ok := anyToInt(dexRaw)
	if !ok {
		return 0, false
	}
	return (dex / 2) - 5, true
}

func anyToInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(x))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func extractHP(raw map[string]any) (average string, formula string) {
	if raw == nil {
		return "", ""
	}
	hp := raw["hp"]
	if hp == nil {
		return "", ""
	}

	switch x := hp.(type) {
	case map[string]any:
		return asString(x["average"]), asString(x["formula"])
	case map[any]any:
		return asString(x["average"]), asString(x["formula"])
	default:
		// Fallback for odd records where hp can be a scalar.
		one := asString(hp)
		return one, ""
	}
}

func extractHPAverageInt(raw map[string]any) (int, bool) {
	if raw == nil {
		return 0, false
	}
	hp := raw["hp"]
	if hp == nil {
		return 0, false
	}

	getAvg := func(v any) (int, bool) {
		s := strings.TrimSpace(asString(v))
		if s == "" {
			return 0, false
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return i, true
	}

	switch x := hp.(type) {
	case map[string]any:
		return getAvg(x["average"])
	case map[any]any:
		return getAvg(x["average"])
	default:
		return 0, false
	}
}

func extractAC(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	ac := raw["ac"]
	if ac == nil {
		return ""
	}

	one := func(v any) string {
		switch x := v.(type) {
		case map[string]any:
			if s := asString(x["ac"]); s != "" {
				return s
			}
		case map[any]any:
			if s := asString(x["ac"]); s != "" {
				return s
			}
		default:
			return asString(v)
		}
		return ""
	}

	switch x := ac.(type) {
	case []any:
		values := make([]string, 0, len(x))
		for _, item := range x {
			if s := one(item); s != "" {
				values = append(values, s)
			}
		}
		return strings.Join(values, ", ")
	default:
		return one(ac)
	}
}

func extractSpeed(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	v := raw["speed"]
	if v == nil {
		return ""
	}

	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case map[string]any:
		return formatSpeedMapStringAny(x)
	case map[any]any:
		tmp := map[string]any{}
		for k, val := range x {
			key := asString(k)
			if key != "" {
				tmp[key] = val
			}
		}
		return formatSpeedMapStringAny(tmp)
	default:
		return asString(v)
	}
}

func formatSpeedMapStringAny(m map[string]any) string {
	order := []string{"walk", "burrow", "climb", "fly", "swim"}
	used := map[string]struct{}{}
	parts := make([]string, 0, len(m))

	formatVal := func(key string, val any) string {
		switch x := val.(type) {
		case int:
			return fmt.Sprintf("%s %d ft.", key, x)
		case int64:
			return fmt.Sprintf("%s %d ft.", key, x)
		case float64:
			if x == float64(int64(x)) {
				return fmt.Sprintf("%s %d ft.", key, int64(x))
			}
			return fmt.Sprintf("%s %s ft.", key, strconv.FormatFloat(x, 'f', -1, 64))
		case string:
			s := strings.TrimSpace(x)
			if s == "" {
				return ""
			}
			return fmt.Sprintf("%s %s", key, s)
		case map[string]any:
			n := asString(x["number"])
			c := asString(x["condition"])
			if n != "" && c != "" {
				return fmt.Sprintf("%s %s ft. %s", key, n, c)
			}
			if n != "" {
				return fmt.Sprintf("%s %s ft.", key, n)
			}
			if c != "" {
				return fmt.Sprintf("%s %s", key, c)
			}
			return ""
		case map[any]any:
			n := asString(x["number"])
			c := asString(x["condition"])
			if n != "" && c != "" {
				return fmt.Sprintf("%s %s ft. %s", key, n, c)
			}
			if n != "" {
				return fmt.Sprintf("%s %s ft.", key, n)
			}
			if c != "" {
				return fmt.Sprintf("%s %s", key, c)
			}
			return ""
		default:
			s := asString(val)
			if s == "" {
				return ""
			}
			return fmt.Sprintf("%s %s", key, s)
		}
	}

	for _, key := range order {
		if val, ok := m[key]; ok {
			if s := formatVal(key, val); s != "" {
				parts = append(parts, s)
			}
			used[key] = struct{}{}
		}
	}

	for key, val := range m {
		if _, ok := used[key]; ok || key == "canHover" {
			continue
		}
		if s := formatVal(key, val); s != "" {
			parts = append(parts, s)
		}
	}

	return strings.Join(parts, ", ")
}

func keysSorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func sortCR(values []string) []string {
	sort.Slice(values, func(i, j int) bool {
		a, aok := crToFloat(values[i])
		b, bok := crToFloat(values[j])
		if aok && bok {
			if a == b {
				return values[i] < values[j]
			}
			return a < b
		}
		if aok != bok {
			return aok
		}
		return strings.ToLower(values[i]) < strings.ToLower(values[j])
	})
	return values
}

func crToFloat(cr string) (float64, bool) {
	cr = strings.TrimSpace(strings.ToLower(cr))
	if cr == "" || cr == "unknown" {
		return 0, false
	}
	if strings.Contains(cr, "/") {
		parts := strings.SplitN(cr, "/", 2)
		n, err1 := strconv.ParseFloat(parts[0], 64)
		d, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 != nil || err2 != nil || d == 0 {
			return 0, false
		}
		return n / d, true
	}
	v, err := strconv.ParseFloat(cr, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func generateIndividualTreasure(crText string, randIntn func(int) int) (treasureOutcome, error) {
	cr, ok := crToFloat(crText)
	if !ok {
		return treasureOutcome{}, errors.New("invalid cr")
	}
	if randIntn == nil {
		randIntn = rand.Intn
	}
	d100 := randIntn(100) + 1

	roll := func(n, sides, mult int, cur string) (int, string) {
		sum := 0
		for i := 0; i < n; i++ {
			sum += randIntn(sides) + 1
		}
		total := sum * mult
		if mult == 1 {
			return total, fmt.Sprintf("%s: %dd%d = %d", cur, n, sides, total)
		}
		return total, fmt.Sprintf("%s: %dd%d x %d = %d", cur, n, sides, mult, total)
	}

	out := treasureOutcome{
		Kind:      "Individual Treasure",
		D100:      d100,
		Coins:     map[string]int{},
		Breakdown: []string{},
	}
	add := func(cur string, n, sides, mult int) {
		total, detail := roll(n, sides, mult, cur)
		out.Coins[cur] += total
		out.Breakdown = append(out.Breakdown, detail)
	}

	switch {
	case cr <= 4:
		out.Band = "CR 0-4"
		switch {
		case d100 <= 30:
			add("cp", 5, 6, 1)
		case d100 <= 60:
			add("sp", 4, 6, 1)
		case d100 <= 70:
			add("ep", 3, 6, 1)
		case d100 <= 95:
			add("gp", 3, 6, 1)
		default:
			add("pp", 1, 6, 1)
		}
	case cr <= 10:
		out.Band = "CR 5-10"
		switch {
		case d100 <= 30:
			add("cp", 4, 6, 100)
			add("ep", 1, 6, 10)
		case d100 <= 60:
			add("sp", 6, 6, 10)
			add("gp", 2, 6, 10)
		case d100 <= 70:
			add("ep", 3, 6, 10)
			add("gp", 2, 6, 10)
		case d100 <= 95:
			add("gp", 4, 6, 10)
		default:
			add("gp", 2, 6, 10)
			add("pp", 3, 6, 1)
		}
	case cr <= 16:
		out.Band = "CR 11-16"
		switch {
		case d100 <= 20:
			add("sp", 4, 6, 100)
			add("gp", 1, 6, 100)
		case d100 <= 35:
			add("ep", 1, 6, 100)
			add("gp", 1, 6, 100)
		case d100 <= 75:
			add("gp", 2, 6, 100)
			add("pp", 1, 6, 10)
		default:
			add("gp", 2, 6, 100)
			add("pp", 2, 6, 10)
		}
	default:
		out.Band = "CR 17+"
		switch {
		case d100 <= 15:
			add("ep", 2, 6, 1000)
			add("gp", 8, 6, 100)
		case d100 <= 55:
			add("gp", 1, 6, 1000)
			add("pp", 1, 6, 100)
		default:
			add("gp", 1, 6, 1000)
			add("pp", 2, 6, 100)
		}
	}
	return out, nil
}

func generateLairTreasure(crText string, randIntn func(int) int) (treasureOutcome, error) {
	cr, ok := crToFloat(crText)
	if !ok {
		return treasureOutcome{}, errors.New("invalid cr")
	}
	if randIntn == nil {
		randIntn = rand.Intn
	}
	d100 := randIntn(100) + 1

	roll := func(n, sides, mult int, label string) (int, string) {
		sum := 0
		for i := 0; i < n; i++ {
			sum += randIntn(sides) + 1
		}
		total := sum * mult
		if mult == 1 {
			return total, fmt.Sprintf("%s: %dd%d = %d", label, n, sides, total)
		}
		return total, fmt.Sprintf("%s: %dd%d x %d = %d", label, n, sides, mult, total)
	}

	out := treasureOutcome{
		Kind:      "Lair (Hoard) Treasure",
		D100:      d100,
		Coins:     map[string]int{},
		Breakdown: []string{},
		Extras:    []string{},
	}
	addCoin := func(cur string, n, sides, mult int) {
		total, detail := roll(n, sides, mult, cur)
		out.Coins[cur] += total
		out.Breakdown = append(out.Breakdown, detail)
	}
	addGemArt := func(kind string, n, sides, mult int, value int) {
		total, detail := roll(n, sides, mult, kind)
		out.Breakdown = append(out.Breakdown, detail)
		if total <= 0 {
			return
		}
		if kind == "gems" {
			types := rollNamedLootTypes(total, gemTypeTableByValue(value), randIntn)
			out.Extras = append(out.Extras, fmt.Sprintf("%d gems (%d gp ciascuna): %s", total, value, strings.Join(types, "; ")))
			return
		}
		types := rollNamedLootTypes(total, artObjectTableByValue(value), randIntn)
		out.Extras = append(out.Extras, fmt.Sprintf("%d art objects (%d gp ciascuno): %s", total, value, strings.Join(types, "; ")))
	}
	addMagic := func(n, sides int, table string) {
		total, detail := roll(n, sides, 1, "Magic Items")
		out.Breakdown = append(out.Breakdown, detail)
		if total <= 0 {
			return
		}
		types := rollNamedLootTypes(total, magicItemTypeByTable(table), randIntn)
		out.Extras = append(out.Extras, fmt.Sprintf("%d item/i da Magic Item Table %s: %s", total, table, strings.Join(types, "; ")))
	}

	switch {
	case cr <= 4:
		out.Band = "CR 0-4"
		addCoin("cp", 6, 6, 100)
		addCoin("sp", 3, 6, 100)
		addCoin("gp", 2, 6, 10)
		switch {
		case d100 <= 6:
		case d100 <= 16:
			addGemArt("gems", 2, 6, 1, 10)
		case d100 <= 26:
			addGemArt("art objects", 2, 4, 1, 25)
		case d100 <= 36:
			addGemArt("gems", 2, 6, 1, 50)
		case d100 <= 44:
			addGemArt("gems", 2, 6, 1, 10)
			addMagic(1, 6, "A")
		case d100 <= 52:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 6, "A")
		case d100 <= 60:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 6, "A")
		case d100 <= 65:
			addGemArt("gems", 2, 6, 1, 10)
			addMagic(1, 4, "B")
		case d100 <= 70:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "B")
		case d100 <= 75:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "B")
		case d100 <= 78:
			addGemArt("gems", 2, 6, 1, 10)
			addMagic(1, 4, "C")
		case d100 <= 80:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "C")
		case d100 <= 85:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "C")
		case d100 <= 92:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "F")
		case d100 <= 97:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "F")
		case d100 <= 99:
			addGemArt("art objects", 2, 4, 1, 25)
			addMagic(1, 4, "G")
		default:
			addGemArt("gems", 2, 6, 1, 50)
			addMagic(1, 4, "G")
		}
	case cr <= 10:
		out.Band = "CR 5-10"
		addCoin("cp", 2, 6, 100)
		addCoin("sp", 2, 6, 1000)
		addCoin("gp", 6, 6, 100)
		addCoin("pp", 3, 6, 10)
		switch {
		case d100 <= 4:
		case d100 <= 10:
			addGemArt("art objects", 2, 4, 1, 25)
		case d100 <= 16:
			addGemArt("gems", 3, 6, 1, 50)
		case d100 <= 22:
			addGemArt("gems", 3, 6, 1, 100)
		case d100 <= 28:
			addGemArt("art objects", 2, 4, 1, 250)
		case d100 <= 44:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 6, "A")
		case d100 <= 63:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 4, "B")
		case d100 <= 74:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 4, "C")
		case d100 <= 80:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 4, "D")
		case d100 <= 94:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 4, "F")
		case d100 <= 98:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 4, "G")
		default:
			addGemArt("gems", 3, 6, 1, 100)
			addMagic(1, 4, "H")
		}
	case cr <= 16:
		out.Band = "CR 11-16"
		addCoin("gp", 4, 6, 1000)
		addCoin("pp", 5, 6, 100)
		switch {
		case d100 <= 3:
		case d100 <= 15:
			addGemArt("gems", 3, 6, 1, 500)
			addMagic(1, 4, "A")
			addMagic(1, 6, "B")
		case d100 <= 29:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "A")
			addMagic(1, 6, "B")
		case d100 <= 50:
			addGemArt("art objects", 2, 4, 1, 250)
			addMagic(1, 6, "C")
		case d100 <= 66:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "D")
		case d100 <= 74:
			addGemArt("art objects", 2, 4, 1, 750)
			addMagic(1, 6, "E")
		case d100 <= 82:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "F")
			addMagic(1, 4, "G")
		case d100 <= 94:
			addGemArt("art objects", 2, 4, 1, 750)
			addMagic(1, 4, "H")
		default:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 4, "I")
		}
	default:
		out.Band = "CR 17+"
		addCoin("gp", 12, 6, 1000)
		addCoin("pp", 8, 6, 1000)
		switch {
		case d100 <= 2:
		case d100 <= 14:
			addGemArt("gems", 3, 6, 1, 1000)
			addMagic(1, 8, "C")
		case d100 <= 46:
			addGemArt("art objects", 1, 10, 1, 2500)
			addMagic(1, 6, "D")
		case d100 <= 68:
			addGemArt("gems", 1, 8, 1, 5000)
			addMagic(1, 6, "E")
		case d100 <= 76:
			addGemArt("art objects", 1, 4, 1, 7500)
			addMagic(1, 4, "F")
			addMagic(1, 4, "G")
		case d100 <= 93:
			addGemArt("gems", 1, 8, 1, 5000)
			addMagic(1, 6, "H")
		default:
			addGemArt("art objects", 1, 4, 1, 7500)
			addMagic(1, 4, "I")
		}
	}

	return out, nil
}

func rollNamedLootTypes(count int, pool []string, randIntn func(int) int) []string {
	if count <= 0 || len(pool) == 0 {
		return nil
	}
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		idx := randIntn(len(pool))
		out = append(out, pool[idx])
	}
	return out
}

func gemTypeTableByValue(value int) []string {
	switch value {
	case 10:
		return []string{"azzurrite", "agata a bande", "occhio di tigre", "ematite", "lapislazzuli", "malachite"}
	case 50:
		return []string{"sardonica", "corniola", "diaspro sanguigno", "calcedonio", "quarzo stellato", "ambra"}
	case 100:
		return []string{"ametista", "granato", "perla", "spinello", "tormalina", "topazio"}
	case 500:
		return []string{"acquamarina", "perla nera", "peridoto", "zaffiro blu pallido", "topazio imperiale", "opale nero"}
	case 1000:
		return []string{"smeraldo", "rubino", "zaffiro", "diamante giallo", "opale di fuoco", "giada imperiale"}
	case 5000:
		return []string{"diamante", "rubino stellato", "smeraldo perfetto", "zaffiro stellato", "opale di fuoco puro", "diamante blu"}
	default:
		return []string{"gemma comune"}
	}
}

func artObjectTableByValue(value int) []string {
	switch value {
	case 25:
		return []string{"anello d'argento cesellato", "coppa di rame sbalzata", "maschera cerimoniale lignea", "bracciale d'avorio", "spilla in bronzo", "statuetta in osso"}
	case 250:
		return []string{"brocca d'argento filigranata", "collana con perle piccole", "arazzo fine", "specchio in argento", "cofanetto laccato con intarsi", "icona religiosa in argento"}
	case 750:
		return []string{"corona d'oro sottile", "calice d'oro e smalto", "pendente con zaffiro", "bracciale d'oro massiccio", "arazzo di corte", "strumento musicale intarsiato"}
	case 2500:
		return []string{"diadema con gemme", "scettro d'oro e avorio", "pettorale cerimoniale", "statuetta in oro pieno", "maschera rituale in oro", "coppa regale con rubini"}
	case 7500:
		return []string{"corona regale con diamanti", "scultura in giada e oro", "calice imperiale con zaffiri", "armilla in platino", "cofanetto reale tempestato di gemme", "statuetta divina in oro e gemme"}
	default:
		return []string{"oggetto d'arte comune"}
	}
}

func magicItemTypeByTable(table string) []string {
	switch strings.ToUpper(strings.TrimSpace(table)) {
	case "A":
		return []string{"pozione", "pergamena", "munizioni +1", "sacca utility", "piccolo oggetto wondrous", "trinket magico"}
	case "B":
		return []string{"pozione maggiore", "armatura +1", "arma +1", "bastone minore", "anello minore", "oggetto wondrous non comune"}
	case "C":
		return []string{"pergamena superiore", "pozione superiore", "scudo +1", "arma +2", "verga minore", "oggetto wondrous raro"}
	case "D":
		return []string{"armatura +2", "anello raro", "bastone raro", "bacchetta rara", "oggetto wondrous raro", "arma con proprietà speciale"}
	case "E":
		return []string{"pergamena alta magia", "pozione suprema", "verga rara", "anello potente", "bastone potente", "oggetto wondrous molto raro"}
	case "F":
		return []string{"arma +1/+2", "scudo +2", "armatura +1 con proprietà", "arma con danno extra", "oggetto wondrous marziale", "anello difensivo"}
	case "G":
		return []string{"arma +2", "armatura +2", "scudo +2", "verga offensiva", "bastone di potere", "oggetto wondrous molto raro"}
	case "H":
		return []string{"arma +3", "armatura +3", "anello leggendario", "bastone leggendario", "verga leggendaria", "oggetto wondrous leggendario"}
	case "I":
		return []string{"artefatto minore", "arma reliquia", "oggetto unico", "focus leggendario", "armatura mitica", "reliquia antica"}
	default:
		return []string{"oggetto magico"}
	}
}

func blankIfEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (ui *UI) nextCustomOrdinal(name string) int {
	name = strings.TrimSpace(strings.ToLower(name))
	maxOrd := 0
	for _, it := range ui.encounterItems {
		if it.Custom && strings.TrimSpace(strings.ToLower(it.CustomName)) == name && it.Ordinal > maxOrd {
			maxOrd = it.Ordinal
		}
	}
	return maxOrd + 1
}

func parseInitInput(s string) (hasRoll bool, roll int, base int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, 0, 0, false
	}
	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		a, errA := strconv.Atoi(strings.TrimSpace(parts[0]))
		b, errB := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errA != nil || errB != nil {
			return false, 0, 0, false
		}
		return true, a, b, true
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return false, 0, 0, false
	}
	return false, 0, v, true
}

func parseHPInput(s string) (current int, max int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		c, errC := strconv.Atoi(strings.TrimSpace(parts[0]))
		m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errC != nil || errM != nil || m < 0 {
			return 0, 0, false
		}
		if c < 0 {
			c = 0
		}
		if c > m && m > 0 {
			c = m
		}
		return c, m, true
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return 0, 0, false
	}
	return v, v, true
}

func expandDiceRollInput(input string) ([]string, error) {
	return diceroll.ExpandRollInput(input)
}

func parseDiceRollBatch(input string) (expr string, times int, err error) {
	return diceroll.ParseRollBatch(input)
}

func rollDiceExpression(expr string) (total int, breakdown string, err error) {
	return diceroll.RollExpression(expr)
}

func chooseDiceMode(mode byte, a int, b int) int {
	return diceroll.ChooseMode(mode, a, b)
}

func (ui *UI) encounterEntryName(entry EncounterEntry) string {
	if entry.Custom {
		if strings.TrimSpace(entry.CustomName) == "" {
			return "Custom"
		}
		return entry.CustomName
	}
	if entry.MonsterIndex < 0 || entry.MonsterIndex >= len(ui.monsters) {
		return "Unknown"
	}
	return ui.monsters[entry.MonsterIndex].Name
}

func (ui *UI) encounterEntryDisplay(entry EncounterEntry) string {
	name := ui.encounterEntryName(entry)
	if entry.Custom {
		return name
	}
	return fmt.Sprintf("%s #%d", name, entry.Ordinal)
}

func (ui *UI) encounterInitBase(entry EncounterEntry) (int, bool) {
	if entry.Custom {
		return entry.CustomInit, true
	}
	if entry.MonsterIndex < 0 || entry.MonsterIndex >= len(ui.monsters) {
		return 0, false
	}
	return extractInitFromDex(ui.monsters[entry.MonsterIndex].Raw)
}

func (ui *UI) encounterMaxHP(entry EncounterEntry) int {
	if entry.UseRolledHP && entry.RolledHP > 0 {
		return entry.RolledHP
	}
	return entry.BaseHP
}

var (
	hpFormulaRe   = regexp.MustCompile(`^\s*(\d+)\s*[dD]\s*(\d+)(?:\s*([+-])\s*(\d+))?\s*$`)
	finalResultRe = regexp.MustCompile(`[-+]?\d+`)
)

func rollHPFormula(formula string) (int, bool) {
	m := hpFormulaRe.FindStringSubmatch(strings.TrimSpace(formula))
	if len(m) == 0 {
		return 0, false
	}

	nDice, err1 := strconv.Atoi(m[1])
	dieFaces, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || nDice <= 0 || dieFaces <= 0 {
		return 0, false
	}
	if nDice > 200 || dieFaces > 10000 {
		return 0, false
	}

	total := 0
	for i := 0; i < nDice; i++ {
		total += rand.Intn(dieFaces) + 1
	}
	if m[3] != "" && m[4] != "" {
		mod, err := strconv.Atoi(m[4])
		if err != nil {
			return 0, false
		}
		if m[3] == "-" {
			total -= mod
		} else {
			total += mod
		}
	}
	if total < 0 {
		total = 0
	}
	return total, true
}

func cloneIntMap(src map[int]int) map[int]int {
	dst := make(map[int]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func setTheme() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.BorderColor = tcell.ColorGold
	tview.Styles.TitleColor = tcell.ColorGold
	tview.Styles.GraphicsColor = tcell.ColorGold
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorLightGray
	tview.Styles.TertiaryTextColor = tcell.ColorAqua
	tview.Styles.InverseTextColor = tcell.ColorBlack
	tview.Styles.ContrastSecondaryTextColor = tcell.ColorBlack
}
