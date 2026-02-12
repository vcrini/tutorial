package main

import (
	_ "embed"
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

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

const helpText = " [black:gold] q [-:-] esci  [black:gold] / [-:-] cerca (Name/Description)  [black:gold] tab [-:-] focus  [black:gold] 0/1/2/3 [-:-] pannelli  [black:gold] a[-:-] roll Dice  [black:gold] f[-:-] fullscreen panel  [black:gold] j/k [-:-] naviga  [black:gold] d [-:-] del encounter  [black:gold] s/l [-:-] save/load  [black:gold] i/I [-:-] roll init one/all  [black:gold] S [-:-] sort init  [black:gold] * [-:-] turn mode  [black:gold] n/p [-:-] next/prev turn  [black:gold] u/r [-:-] undo/redo  [black:gold] spazio [-:-] avg/formula HP  [black:gold] ←/→ [-:-] danno/cura encounter  [black:gold] PgUp/PgDn [-:-] scroll Description "
const defaultEncountersPath = "encounters.yaml"
const lastEncountersPathFile = ".encounters_last_path"
const defaultDicePath = "dice.yaml"
const lastDicePathFile = ".dice_last_path"

//go:embed data/5e.yaml
var embeddedMonstersYAML []byte

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

type UI struct {
	app         *tview.Application
	monsters    []Monster
	filtered    []int
	envOptions  []string
	crOptions   []string
	typeOptions []string

	nameFilter string
	envFilter  string
	crFilter   string
	typeFilter string

	nameInput     *tview.InputField
	envDrop       *tview.DropDown
	crDrop        *tview.DropDown
	typeDrop      *tview.DropDown
	dice          *tview.List
	encounter     *tview.List
	list          *tview.List
	detailMeta    *tview.TextView
	detailRaw     *tview.TextView
	status        *tview.TextView
	pages         *tview.Pages
	leftPanel     *tview.Flex
	monstersPanel *tview.Flex
	mainRow       *tview.Flex
	detailPanel   *tview.Flex
	filterHost    *tview.Pages

	focusOrder []tview.Primitive
	rawText    string
	rawQuery   string
	diceLog    []DiceResult
	diceRender bool
	wideFilter bool

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

	helpVisible      bool
	helpReturnFocus  tview.Primitive
	addCustomVisible bool
	fullscreenActive bool
	fullscreenTarget string
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

	ui := newUI(monsters, envs, crs, types, encountersPath, dicePath)
	if err := ui.run(); err != nil {
		log.Fatal(err)
	}
	if err := ui.saveEncounters(); err != nil {
		log.Printf("errore salvataggio encounters (%s): %v", encountersPath, err)
	}
	if err := ui.saveDiceResults(); err != nil {
		log.Printf("errore salvataggio dice (%s): %v", ui.dicePath, err)
	}
}

func newUI(monsters []Monster, envs, crs, types []string, encountersPath string, dicePath string) *UI {
	setTheme()

	ui := &UI{
		app:             tview.NewApplication(),
		monsters:        monsters,
		envOptions:      append([]string{"All"}, envs...),
		crOptions:       append([]string{"All"}, crs...),
		typeOptions:     append([]string{"All"}, types...),
		filtered:        make([]int, 0, len(monsters)),
		encounterSerial: map[int]int{},
		encounterItems:  make([]EncounterEntry, 0, 16),
		encountersPath:  encountersPath,
		dicePath:        dicePath,
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
		SetLabel(" Env ").
		SetOptions(ui.envOptions, func(option string, _ int) {
			if option == "All" {
				ui.envFilter = ""
			} else {
				ui.envFilter = option
			}
			ui.applyFilters()
		})
	ui.envDrop.SetLabelColor(tcell.ColorGold)
	ui.envDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.envDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.envDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)

	ui.crDrop = tview.NewDropDown().
		SetLabel(" CR ").
		SetOptions(ui.crOptions, func(option string, _ int) {
			if option == "All" {
				ui.crFilter = ""
			} else {
				ui.crFilter = option
			}
			ui.applyFilters()
		})
	ui.crDrop.SetLabelColor(tcell.ColorGold)
	ui.crDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.crDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.crDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)

	ui.typeDrop = tview.NewDropDown().
		SetLabel(" Type ").
		SetOptions(ui.typeOptions, func(option string, _ int) {
			if option == "All" {
				ui.typeFilter = ""
			} else {
				ui.typeFilter = option
			}
			ui.applyFilters()
		})
	ui.typeDrop.SetLabelColor(tcell.ColorGold)
	ui.typeDrop.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	ui.typeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.typeDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(tcell.ColorGold).Foreground(tcell.ColorBlack),
	)

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

	ui.detailRaw = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(false)
	ui.detailRaw.SetBorder(true)
	ui.detailRaw.SetTitle(" [3]-Description ")
	ui.detailRaw.SetTitleColor(tcell.ColorGold)
	ui.detailRaw.SetBorderColor(tcell.ColorGold)
	ui.detailRaw.SetTextColor(tcell.ColorWhite)

	ui.detailPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.detailMeta, 10, 0, false).
		AddItem(ui.detailRaw, 0, 1, false)

	ui.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText)
	ui.status.SetBackgroundColor(tcell.ColorBlack)

	filterRowSingle := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.nameInput, 0, 3, true).
		AddItem(ui.envDrop, 0, 2, false).
		AddItem(ui.crDrop, 0, 1, false).
		AddItem(ui.typeDrop, 0, 2, false)

	filterRowTop := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.nameInput, 0, 2, true).
		AddItem(ui.envDrop, 0, 1, false)

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
	ui.focusOrder = []tview.Primitive{ui.dice, ui.encounter, ui.nameInput, ui.envDrop, ui.crDrop, ui.typeDrop, ui.list, ui.detailRaw}
	ui.app.SetFocus(ui.list)
	ui.envDrop.SetCurrentOption(0)
	ui.crDrop.SetCurrentOption(0)
	ui.typeDrop.SetCurrentOption(0)
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
			ui.app.SetFocus(ui.envDrop)
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 'c':
			ui.app.SetFocus(ui.crDrop)
			return nil
		case focus == ui.list && event.Key() == tcell.KeyRune && event.Rune() == 't':
			ui.app.SetFocus(ui.typeDrop)
			return nil
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
		return "Monsters"
	case ui.detailRaw:
		return "Description"
	case ui.nameInput:
		return "Name Filter"
	case ui.envDrop:
		return "Env Filter"
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
		"  0 / 1 / 2 / 3 : vai a Dice / Encounters / Monsters / Description\n\n"

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
		return header +
			"[black:gold]Monsters[-:-]\n" +
			"  j / k (o frecce) : naviga mostri\n" +
			"  / : cerca nella Description del mostro selezionato\n" +
			"  a : aggiungi mostro a Encounters\n" +
			"  n / e / c / t : focus su Name / Env / CR / Type\n" +
			"  PgUp / PgDn : scroll del pannello Description\n"
	case ui.detailRaw:
		return header +
			"[black:gold]Description[-:-]\n" +
			"  / : cerca testo nella Description corrente\n" +
			"  j / k (o frecce) : scroll contenuto\n"
	case ui.nameInput:
		return header +
			"[black:gold]Name Filter[-:-]\n" +
			"  scrivi testo : filtro per nome\n" +
			"  Enter / Esc : torna a Monsters\n"
	case ui.envDrop, ui.crDrop, ui.typeDrop:
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
	if ui.mainRow == nil || ui.leftPanel == nil || ui.detailPanel == nil || ui.filterHost == nil || ui.monstersPanel == nil {
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
	ui.detailPanel.ResizeItem(ui.detailMeta, 10, 0)
	ui.detailPanel.ResizeItem(ui.detailRaw, 0, 1)
}

func (ui *UI) fullscreenTargetForFocus(focus tview.Primitive) string {
	switch focus {
	case ui.dice:
		return "dice"
	case ui.encounter:
		return "encounter"
	case ui.list:
		return "monsters"
	case ui.detailRaw:
		return "description"
	case ui.nameInput, ui.envDrop, ui.crDrop, ui.typeDrop:
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
		ui.detailPanel.ResizeItem(ui.detailRaw, 0, 1)
	}
	ui.status.SetText(fmt.Sprintf(" [black:gold]fullscreen[-:-] %s  %s", target, helpText))
}

func (ui *UI) run() error {
	return ui.app.Run()
}

func (ui *UI) applyFilters() {
	ui.filtered = ui.filtered[:0]

	for i, m := range ui.monsters {
		if !matchName(m.Name, ui.nameFilter) {
			continue
		}
		if !matchCR(m.CR, ui.crFilter) {
			continue
		}
		if !matchEnv(m.Environment, ui.envFilter) {
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

	for _, idx := range ui.filtered {
		m := ui.monsters[idx]
		ui.list.AddItem(m.Name, "", 0, nil)
	}

	ui.status.SetText(fmt.Sprintf(" [black:gold] %d risultati [-:-] %s", len(ui.filtered), helpText))

	if len(ui.filtered) == 0 {
		ui.detailMeta.SetText("Nessun mostro con i filtri correnti.")
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
		ui.detailMeta.SetText("Seleziona un mostro dalla lista.")
		ui.detailRaw.SetText("")
		ui.rawText = ""
		return
	}
	ui.renderDetailByMonsterIndex(ui.filtered[listIndex])
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

func matchEnv(monsterEnvs []string, query string) bool {
	if query == "" {
		return true
	}
	for _, env := range monsterEnvs {
		if strings.EqualFold(env, query) {
			return true
		}
	}
	return false
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

var diceBatchRe = regexp.MustCompile(`(?i)^(.*?)(?:\s*x\s*(\d+))$`)
var diceTermForDoubleRe = regexp.MustCompile(`(?i)(\d*)d(\d+[a-zA-Z]*)`)

func expandDiceRollInput(input string) ([]string, error) {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		expr, times, err := parseDiceRollBatch(p)
		if err != nil {
			return nil, err
		}
		for i := 0; i < times; i++ {
			out = append(out, expr)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("vuota")
	}
	return out, nil
}

func parseDiceRollBatch(input string) (expr string, times int, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", 0, errors.New("vuota")
	}
	if m := diceBatchRe.FindStringSubmatch(s); len(m) == 3 {
		expr = strings.TrimSpace(m[1])
		if expr == "" {
			return "", 0, errors.New("espressione vuota")
		}
		n, convErr := strconv.Atoi(strings.TrimSpace(m[2]))
		if convErr != nil || n <= 0 {
			return "", 0, errors.New("moltiplicatore non valido")
		}
		if n > 200 {
			return "", 0, errors.New("moltiplicatore troppo alto")
		}
		return expr, n, nil
	}
	return s, 1, nil
}

func rollDiceExpression(expr string) (total int, breakdown string, err error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, "", errors.New("vuota")
	}
	fullExpr := expr
	checkTarget := 0
	hasCheck := false
	checkInclusive := false
	checkOnSuccessExpr := ""
	if opPos := strings.Index(expr, ">="); opPos >= 0 || strings.Contains(expr, ">") {
		op := ">"
		pos := strings.Index(expr, ">")
		if gePos := strings.Index(expr, ">="); gePos >= 0 && (pos < 0 || gePos <= pos) {
			op = ">="
			pos = gePos
			checkInclusive = true
		}
		if pos < 0 {
			return 0, "", errors.New("condizione > non valida")
		}
		expr = strings.TrimSpace(expr[:pos])
		condRaw := strings.TrimSpace(fullExpr[pos+len(op):])
		if expr == "" || condRaw == "" {
			return 0, "", errors.New("condizione > non valida")
		}
		m := regexp.MustCompile(`^([+-]?\d+)(?:\s+(.+))?$`).FindStringSubmatch(condRaw)
		if len(m) != 3 {
			return 0, "", fmt.Errorf("soglia non valida: %q", condRaw)
		}
		v, convErr := strconv.Atoi(strings.TrimSpace(m[1]))
		if convErr != nil {
			return 0, "", fmt.Errorf("soglia non valida: %q", condRaw)
		}
		checkTarget = v
		hasCheck = true
		checkOnSuccessExpr = strings.TrimSpace(m[2])
	}

	// Tokenize on + and - while preserving each term sign.
	terms := make([]string, 0, 8)
	start := 0
	for i := 1; i < len(expr); i++ {
		if expr[i] == '+' || expr[i] == '-' {
			terms = append(terms, expr[start:i])
			start = i
		}
	}
	terms = append(terms, expr[start:])

	pieces := make([]string, 0, len(terms))
	sum := 0
	critEnabledOnFirstDice := false
	critTriggered := false

	for termIndex, raw := range terms {
		token := strings.TrimSpace(raw)
		if token == "" {
			return 0, "", errors.New("token vuoto")
		}
		sign := 1
		if token[0] == '+' {
			token = strings.TrimSpace(token[1:])
		} else if token[0] == '-' {
			sign = -1
			token = strings.TrimSpace(token[1:])
		}
		if token == "" {
			return 0, "", errors.New("token vuoto")
		}
		if i := strings.IndexAny(token, "dD"); i >= 0 {
			countStr := strings.TrimSpace(token[:i])
			sidesStr := strings.TrimSpace(token[i+1:])
			count := 1
			if countStr != "" {
				v, convErr := strconv.Atoi(countStr)
				if convErr != nil || v <= 0 {
					return 0, "", fmt.Errorf("numero dadi non valido: %q", countStr)
				}
				count = v
			}
			sides, mode, critSuffix, convErr := parseDiceSidesSpec(sidesStr)
			if convErr != nil {
				return 0, "", fmt.Errorf("facce non valide: %q", sidesStr)
			}
			if termIndex == 0 && critSuffix {
				critEnabledOnFirstDice = true
			}
			if count > 1000 || sides > 100000 {
				return 0, "", errors.New("limite dadi superato")
			}
			rolls := make([]string, 0, count)
			termTotal := 0
			for i := 0; i < count; i++ {
				if mode == 0 {
					r := rand.Intn(sides) + 1
					termTotal += r
					rolls = append(rolls, strconv.Itoa(r))
					if termIndex == 0 && critSuffix && r == sides {
						critTriggered = true
					}
					continue
				}
				r1 := rand.Intn(sides) + 1
				r2 := rand.Intn(sides) + 1
				chosen := chooseDiceMode(mode, r1, r2)
				termTotal += chosen
				rolls = append(rolls, fmt.Sprintf("%d|%d->%d", r1, r2, chosen))
				if termIndex == 0 && critSuffix && chosen == sides {
					critTriggered = true
				}
			}
			sum += sign * termTotal
			modeSuffix := ""
			if mode != 0 {
				modeSuffix = string(mode)
			}
			critSuffixStr := ""
			if critSuffix {
				critSuffixStr = "c"
			}
			termPiece := fmt.Sprintf("%dd%d%s%s(%s)", count, sides, modeSuffix, critSuffixStr, strings.Join(rolls, "+"))
			if sign < 0 {
				termPiece = "-" + termPiece
			}
			pieces = append(pieces, termPiece)
			continue
		}

		v, convErr := strconv.Atoi(token)
		if convErr != nil {
			return 0, "", fmt.Errorf("costante non valida: %q", token)
		}
		sum += sign * v
		if sign < 0 {
			pieces = append(pieces, "-"+strconv.Itoa(v))
		} else {
			pieces = append(pieces, strconv.Itoa(v))
		}
	}
	final := sum
	breakdown = fmt.Sprintf("%s = %d", strings.Join(pieces, " + "), sum)
	if sum < 0 {
		final = 0
		breakdown = fmt.Sprintf("%s -> 0", breakdown)
	}
	if hasCheck {
		success := final > checkTarget
		if checkInclusive {
			success = final >= checkTarget
		}
		if checkOnSuccessExpr == "" {
			if success {
				breakdown += " ok"
			} else {
				breakdown += " ko"
			}
		} else {
			if success {
				successExpr := checkOnSuccessExpr
				if critEnabledOnFirstDice && critTriggered {
					successExpr = doubleDiceCounts(successExpr)
				}
				_, successBreakdown, successErr := rollDiceExpression(successExpr)
				if successErr != nil {
					return 0, "", fmt.Errorf("espressione success non valida: %w", successErr)
				}
				breakdown += " -> " + successBreakdown
			} else {
				breakdown += " ko"
			}
		}
	}
	return final, breakdown, nil
}

func chooseDiceMode(mode byte, a int, b int) int {
	switch mode {
	case 'v':
		if a >= b {
			return a
		}
		return b
	case 's':
		if a <= b {
			return a
		}
		return b
	default:
		return a
	}
}

func parseDiceSidesSpec(spec string) (sides int, mode byte, crit bool, err error) {
	m := regexp.MustCompile(`^(\d+)([a-zA-Z]*)$`).FindStringSubmatch(spec)
	if len(m) != 3 {
		return 0, 0, false, errors.New("invalid sides spec")
	}
	sides, err = strconv.Atoi(m[1])
	if err != nil || sides <= 0 {
		return 0, 0, false, errors.New("invalid sides")
	}
	suffix := strings.ToLower(strings.TrimSpace(m[2]))
	for _, ch := range suffix {
		switch ch {
		case 'v':
			if mode != 0 {
				return 0, 0, false, errors.New("mode duplicate")
			}
			mode = 'v'
		case 's':
			if mode != 0 {
				return 0, 0, false, errors.New("mode duplicate")
			}
			mode = 's'
		case 'c':
			crit = true
		default:
			return 0, 0, false, errors.New("unknown suffix")
		}
	}
	return sides, mode, crit, nil
}

func doubleDiceCounts(expr string) string {
	return diceTermForDoubleRe.ReplaceAllStringFunc(expr, func(term string) string {
		m := diceTermForDoubleRe.FindStringSubmatch(term)
		if len(m) != 3 {
			return term
		}
		count := 1
		if strings.TrimSpace(m[1]) != "" {
			v, err := strconv.Atoi(strings.TrimSpace(m[1]))
			if err != nil || v <= 0 {
				return term
			}
			count = v
		}
		return fmt.Sprintf("%dd%s", count*2, m[2])
	})
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

var hpFormulaRe = regexp.MustCompile(`^\s*(\d+)\s*[dD]\s*(\d+)(?:\s*([+-])\s*(\d+))?\s*$`)
var finalResultRe = regexp.MustCompile(`[-+]?\d+`)

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
