package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/vcrini/diceroll"
	"gopkg.in/yaml.v3"
)

const (
	helpText     = " [black:gold]?[-:-] help "
	historyLimit = 200
)

const (
	focusDice = iota
	focusPNG
	focusEncounter
	focusMonSearch
	focusMonRole
	focusMonRank
	focusMonList
	focusEnvSearch
	focusEnvType
	focusEnvRank
	focusEnvList
	focusEqSearch
	focusEqType
	focusEqItemType
	focusEqRank
	focusEqList
	focusCardSearch
	focusCardClass
	focusCardType
	focusCardList
	focusClassSearch
	focusClassName
	focusClassSubclass
	focusClassList
	focusNotesSearch
	focusNotesList
	focusTreasure
	focusDetail
)

type DiceResult struct {
	Expression string `yaml:"expression"`
	Output     string `yaml:"output"`
}

type classPreset struct {
	Traits    string
	Primary   string
	Secondary string
	Armor     string
	ExtraA    string
	ExtraB    string
	Abiti     []string
	Attitude  []string
}

type uiSnapshot struct {
	pngs      []PNG
	encounter []EncounterEntry
	selected  int
}

type tviewUI struct {
	app    *tview.Application
	pages  *tview.Pages
	status *tview.TextView

	dice            *tview.List
	diceLog         []DiceResult
	diceRenderLock  bool
	diceGotoPending bool

	pngList        *tview.List
	encList        *tview.List
	search         *tview.InputField
	roleDrop       *tview.DropDown
	rankDrop       *tview.DropDown
	monList        *tview.List
	envSearch      *tview.InputField
	envTypeDrop    *tview.DropDown
	envRankDrop    *tview.DropDown
	envList        *tview.List
	eqSearch       *tview.InputField
	eqTypeDrop     *tview.DropDown
	eqItemTypeDrop *tview.DropDown
	eqRankDrop     *tview.DropDown
	eqList         *tview.List
	cardSearch     *tview.InputField
	cardClassDrop  *tview.DropDown
	cardTypeDrop   *tview.DropDown
	cardList       *tview.List
	classSearch    *tview.InputField
	classNameDrop  *tview.DropDown
	classSubDrop   *tview.DropDown
	classList      *tview.List
	notesSearch    *tview.InputField
	notesList      *tview.List
	detailBottom   *tview.Pages
	detail         *tview.TextView
	detailTreasure *tview.TextView

	monstersPanel     *tview.Flex
	environmentsPanel *tview.Flex
	equipmentPanel    *tview.Flex
	cardsPanel        *tview.Flex
	classesPanel      *tview.Flex
	notesPanel        *tview.Flex
	catalogPanel      *tview.Pages
	leftPanel         *tview.Flex
	mainRow           *tview.Flex

	focus    []tview.Primitive
	focusIdx int
	message  string

	pngs             []PNG
	selected         int
	monsters         []Monster
	environments     []Environment
	equipment        []EquipmentItem
	cards            []CardItem
	classes          []ClassItem
	notes            []string
	encounter        []EncounterEntry
	filtered         []int
	filteredEnv      []int
	filteredEq       []int
	filteredCards    []int
	filteredClasses  []int
	filteredNotes    []int
	roleOpts         []string
	rankOpts         []string
	envTypeOpts      []string
	envRankOpts      []string
	eqTypeOpts       []string
	eqItemTypeOpts   []string
	eqRankOpts       []string
	cardClassOpts    []string
	cardTypeOpts     []string
	classNameOpts    []string
	classSubOpts     []string
	roleFilter       string
	rankFilter       string
	envTypeFilter    string
	envRankFilter    string
	eqTypeFilter     string
	eqItemTypeFilter string
	eqRankFilter     string
	cardClassFilter  string
	cardTypeFilter   string
	classNameFilter  string
	classSubFilter   string
	catalogMode      string

	detailRaw   string
	detailQuery string
	treasureRaw string

	helpVisible     bool
	helpReturnFocus tview.Primitive

	modalVisible bool
	modalName    string

	fullscreenActive bool
	fullscreenTarget string
	activeBottomPane string
	paure            int
	undoStack        []uiSnapshot
	redoStack        []uiSnapshot
}

func runTViewUI() error {
	if err := initStoragePaths(); err != nil {
		return err
	}
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

	ui, err := newTViewUI()
	if err != nil {
		return err
	}
	ui.refreshAll()
	ui.app.SetRoot(ui.pages, true).SetFocus(ui.monList).EnableMouse(true)
	ui.focusIdx = focusMonList
	return ui.app.Run()
}

func newTViewUI() (*tviewUI, error) {
	pngs, selectedName, err := loadPNGList(dataFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", dataFile, err)
	}
	monsters, err := loadMonsters(monstersFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", monstersFile, err)
	}
	environments, err := loadEnvironments(environmentsFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", environmentsFile, err)
	}
	equipment, err := loadEquipment(equipmentFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", equipmentFile, err)
	}
	cards, err := loadCards(cardsFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", cardsFile, err)
	}
	classes, err := loadClasses(classesFile)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", classesFile, err)
	}
	encounter, err := loadEncounter(encounterFile, monsters)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricare %s: %w", encounterFile, err)
	}
	paure := 0
	if p, err := loadFearState(fearStateFile); err == nil {
		paure = p
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("errore nel caricare %s: %w", fearStateFile, err)
	}
	notes := []string{}
	if ns, err := loadNotes(notesFile); err == nil {
		notes = ns
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("errore nel caricare %s: %w", notesFile, err)
	}

	selected := -1
	if selectedName != "" {
		for i, p := range pngs {
			if p.Name == selectedName {
				selected = i
				break
			}
		}
	}
	if selected < 0 && len(pngs) > 0 {
		selected = 0
	}

	ui := &tviewUI{
		app:              tview.NewApplication(),
		pngs:             pngs,
		selected:         selected,
		monsters:         monsters,
		environments:     environments,
		equipment:        equipment,
		cards:            cards,
		classes:          classes,
		notes:            notes,
		encounter:        encounter,
		message:          "Pronto.",
		catalogMode:      "mostri",
		activeBottomPane: "details",
		paure:            paure,
	}
	ui.build()
	if log, current, err := loadDiceLog(defaultDiceFilePath()); err == nil {
		ui.diceLog = log
		ui.renderDiceList()
		if len(ui.diceLog) > 0 {
			if current < 0 {
				current = 0
			}
			if current >= len(ui.diceLog) {
				current = len(ui.diceLog) - 1
			}
			ui.dice.SetCurrentItem(current)
		}
	} else if !os.IsNotExist(err) {
		ui.message = fmt.Sprintf("Errore caricamento dadi: %v", err)
	}
	return ui, nil
}

func (ui *tviewUI) build() {
	ui.dice = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.dice.SetBorder(true).SetTitle(" [0]-Dadi ")
	ui.dice.SetChangedFunc(func(int, string, string, rune) {
		if ui.diceRenderLock {
			return
		}
		ui.refreshDetail()
	})

	ui.pngList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.pngList.SetBorder(true).SetTitle(" [1]-PNG ")
	ui.pngList.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(ui.pngs) {
			ui.selected = index
			ui.persistPNGs()
		}
		ui.refreshDetail()
	})

	ui.encList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.encList.SetBorder(true).SetTitle(" [2]-Encounter ")
	ui.encList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	ui.notesSearch = tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(0).SetPlaceholder("testo nota...")
	ui.notesSearch.SetChangedFunc(func(_ string) {
		ui.refreshNotes()
		ui.refreshDetail()
	})
	ui.notesSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})

	ui.notesList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.notesList.SetBorder(false)
	ui.notesList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	notesFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.notesSearch, 0, 1, false)

	ui.notesPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(notesFilters, 2, 0, false).
		AddItem(ui.notesList, 0, 1, true)
	ui.notesPanel.SetBorder(true)

	ui.search = tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(0).SetPlaceholder("nome mostro...")
	ui.search.SetChangedFunc(func(_ string) {
		ui.refreshMonsters()
		ui.refreshDetail()
	})
	ui.search.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})

	ui.roleFilter = "Tutti"
	ui.rankFilter = "Tutti"
	ui.roleOpts, ui.rankOpts = ui.buildMonsterFilterOptions()

	ui.roleDrop = tview.NewDropDown().SetLabel(" Ruolo ")
	ui.roleDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.roleDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.roleDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.roleDrop.SetOptions(ui.roleOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.roleFilter = text
		ui.refreshMonsters()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.roleDrop.SetCurrentOption(0)

	ui.rankDrop = tview.NewDropDown().SetLabel(" Rango ")
	ui.rankDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.rankDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.rankDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.rankDrop.SetOptions(ui.rankOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.rankFilter = text
		ui.refreshMonsters()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.rankDrop.SetCurrentOption(0)

	ui.monList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.monList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})
	ui.monList.SetSelectedFunc(func(int, string, string, rune) {
		ui.addSelectedMonsterToEncounter()
	})

	filters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.search, 0, 2, false).
		AddItem(ui.roleDrop, 0, 1, false).
		AddItem(ui.rankDrop, 0, 1, false)

	ui.monstersPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filters, 2, 0, false).
		AddItem(ui.monList, 0, 1, true)
	ui.monstersPanel.SetBorder(true)

	ui.envSearch = tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(0).SetPlaceholder("nome ambiente...")
	ui.envSearch.SetChangedFunc(func(_ string) {
		ui.refreshEnvironments()
		ui.refreshDetail()
	})
	ui.envSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})

	ui.envRankFilter = "Tutti"
	ui.envTypeFilter = "Tutti"
	ui.envTypeOpts = ui.buildEnvironmentTypeOptions()
	ui.envRankOpts = ui.buildEnvironmentRankOptions()

	ui.envTypeDrop = tview.NewDropDown().SetLabel(" Tipo ")
	ui.envTypeDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.envTypeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.envTypeDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.envTypeDrop.SetOptions(ui.envTypeOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.envTypeFilter = text
		ui.refreshEnvironments()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.envTypeDrop.SetCurrentOption(0)

	ui.envRankDrop = tview.NewDropDown().SetLabel(" Rango ")
	ui.envRankDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.envRankDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.envRankDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.envRankDrop.SetOptions(ui.envRankOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.envRankFilter = text
		ui.refreshEnvironments()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.envRankDrop.SetCurrentOption(0)

	ui.envList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.envList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	envFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.envSearch, 0, 2, false).
		AddItem(ui.envTypeDrop, 0, 1, false).
		AddItem(ui.envRankDrop, 0, 1, false)

	ui.environmentsPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(envFilters, 2, 0, false).
		AddItem(ui.envList, 0, 1, true)
	ui.environmentsPanel.SetBorder(true)

	ui.eqSearch = tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(0).SetPlaceholder("nome equipaggiamento...")
	ui.eqSearch.SetChangedFunc(func(_ string) {
		ui.refreshEquipment()
		ui.refreshDetail()
	})
	ui.eqSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})

	ui.eqTypeFilter = "Tutti"
	ui.eqItemTypeFilter = "Tutti"
	ui.eqRankFilter = "Tutti"
	ui.eqTypeOpts = ui.buildEquipmentTypeOptions()
	ui.eqItemTypeOpts = ui.buildEquipmentItemTypeOptions()
	ui.eqRankOpts = ui.buildEquipmentRankOptions()

	ui.eqTypeDrop = tview.NewDropDown().SetLabel(" Categoria ")
	ui.eqTypeDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.eqTypeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.eqTypeDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.eqTypeDrop.SetOptions(ui.eqTypeOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.eqTypeFilter = text
		ui.refreshEquipment()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.eqTypeDrop.SetCurrentOption(0)

	ui.eqItemTypeDrop = tview.NewDropDown().SetLabel(" Tipo ")
	ui.eqItemTypeDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.eqItemTypeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.eqItemTypeDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.eqItemTypeDrop.SetOptions(ui.eqItemTypeOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.eqItemTypeFilter = text
		ui.refreshEquipment()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.eqItemTypeDrop.SetCurrentOption(0)

	ui.eqRankDrop = tview.NewDropDown().SetLabel(" Rango ")
	ui.eqRankDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.eqRankDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.eqRankDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.eqRankDrop.SetOptions(ui.eqRankOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.eqRankFilter = text
		ui.refreshEquipment()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.eqRankDrop.SetCurrentOption(0)

	ui.eqList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.eqList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	eqFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.eqSearch, 0, 2, false).
		AddItem(ui.eqTypeDrop, 0, 1, false).
		AddItem(ui.eqItemTypeDrop, 0, 1, false).
		AddItem(ui.eqRankDrop, 0, 1, false)

	ui.equipmentPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(eqFilters, 2, 0, false).
		AddItem(ui.eqList, 0, 1, true)
	ui.equipmentPanel.SetBorder(true)

	ui.cardSearch = tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(0).SetPlaceholder("nome carta...")
	ui.cardSearch.SetChangedFunc(func(_ string) {
		ui.refreshCards()
		ui.refreshDetail()
	})
	ui.cardSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})

	ui.cardClassFilter = "Tutti"
	ui.cardTypeFilter = "Tutti"
	ui.cardClassOpts = ui.buildCardClassOptions()
	ui.cardTypeOpts = ui.buildCardTypeOptions()

	ui.cardClassDrop = tview.NewDropDown().SetLabel(" Classe ")
	ui.cardClassDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.cardClassDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.cardClassDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.cardClassDrop.SetOptions(ui.cardClassOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.cardClassFilter = text
		ui.refreshCards()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.cardClassDrop.SetCurrentOption(0)

	ui.cardTypeDrop = tview.NewDropDown().SetLabel(" Tipo ")
	ui.cardTypeDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.cardTypeDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.cardTypeDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.cardTypeDrop.SetOptions(ui.cardTypeOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.cardTypeFilter = text
		ui.refreshCards()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.cardTypeDrop.SetCurrentOption(0)

	ui.cardList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.cardList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	cardFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.cardSearch, 0, 2, false).
		AddItem(ui.cardClassDrop, 0, 1, false).
		AddItem(ui.cardTypeDrop, 0, 1, false)

	ui.cardsPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(cardFilters, 2, 0, false).
		AddItem(ui.cardList, 0, 1, true)
	ui.cardsPanel.SetBorder(true)

	ui.classSearch = tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(0).SetPlaceholder("nome classe/sottoclasse...")
	ui.classSearch.SetChangedFunc(func(_ string) {
		ui.refreshClasses()
		ui.refreshDetail()
	})
	ui.classSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.focusActiveCatalogList()
		}
	})

	ui.classNameFilter = "Tutti"
	ui.classSubFilter = "Tutti"
	ui.classNameOpts = ui.buildClassNameOptions()
	ui.classSubOpts = ui.buildClassSubclassOptions()

	ui.classNameDrop = tview.NewDropDown().SetLabel(" Classe ")
	ui.classNameDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.classNameDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.classNameDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.classNameDrop.SetOptions(ui.classNameOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.classNameFilter = text
		ui.refreshClasses()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.classNameDrop.SetCurrentOption(0)

	ui.classSubDrop = tview.NewDropDown().SetLabel(" Sottoclasse ")
	ui.classSubDrop.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.classSubDrop.SetFieldTextColor(tcell.ColorWhite)
	ui.classSubDrop.SetListStyles(
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
	)
	ui.classSubDrop.SetOptions(ui.classSubOpts, func(text string, _ int) {
		if text == "" {
			text = "Tutti"
		}
		ui.classSubFilter = text
		ui.refreshClasses()
		ui.refreshDetail()
		ui.focusActiveCatalogList()
	})
	ui.classSubDrop.SetCurrentOption(0)

	ui.classList = tview.NewList().ShowSecondaryText(false).SetSelectedFocusOnly(true)
	ui.classList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

	classFilters := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.classSearch, 0, 2, false).
		AddItem(ui.classNameDrop, 0, 1, false).
		AddItem(ui.classSubDrop, 0, 1, false)

	ui.classesPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(classFilters, 2, 0, false).
		AddItem(ui.classList, 0, 1, true)
	ui.classesPanel.SetBorder(true)

	ui.catalogPanel = tview.NewPages().
		AddPage("mostri", ui.monstersPanel, true, true).
		AddPage("ambienti", ui.environmentsPanel, true, false).
		AddPage("equipaggiamento", ui.equipmentPanel, true, false).
		AddPage("carte", ui.cardsPanel, true, false).
		AddPage("classe", ui.classesPanel, true, false).
		AddPage("note", ui.notesPanel, true, false)
	ui.refreshCatalogTitles()

	ui.leftPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.dice, 7, 0, false).
		AddItem(ui.pngList, 0, 1, true).
		AddItem(ui.encList, 0, 1, false).
		AddItem(ui.catalogPanel, 0, 1, false)

	ui.detail = tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	ui.detail.SetBorder(true).SetTitle(" Dettagli ")

	ui.detailTreasure = tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	ui.detailTreasure.SetBorder(true).SetTitle(" Treasure ")
	ui.treasureRaw = "Nessun treasure generato."
	ui.renderTreasure()

	ui.detailBottom = tview.NewPages().
		AddPage("details", ui.detail, true, true).
		AddPage("treasure", ui.detailTreasure, true, false)

	ui.mainRow = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.leftPanel, 0, 1, false).
		AddItem(ui.detailBottom, 0, 1, false)

	ui.status = tview.NewTextView().SetDynamicColors(true).SetText(helpText)
	ui.status.SetBackgroundColor(tcell.ColorBlack)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.mainRow, 0, 1, true).
		AddItem(ui.status, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", root, true, true)
	ui.focus = []tview.Primitive{
		ui.dice,
		ui.pngList, ui.encList, ui.search, ui.roleDrop, ui.rankDrop, ui.monList,
		ui.envSearch, ui.envTypeDrop, ui.envRankDrop, ui.envList,
		ui.eqSearch, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqList,
		ui.cardSearch, ui.cardClassDrop, ui.cardTypeDrop, ui.cardList,
		ui.classSearch, ui.classNameDrop, ui.classSubDrop, ui.classList,
		ui.notesSearch, ui.notesList,
		ui.detailTreasure,
		ui.detail,
	}
	ui.focusIdx = focusMonList
	ui.app.SetFocus(ui.monList)
	ui.app.SetInputCapture(ui.handleGlobalKeys)
	ui.renderDiceList()
	ui.refreshNotes()
}

func (ui *tviewUI) handleGlobalKeys(ev *tcell.EventKey) *tcell.EventKey {
	focus := ui.app.GetFocus()
	_, focusIsInput := focus.(*tview.InputField)

	if ui.helpVisible {
		if ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && (ev.Rune() == '?' || ev.Rune() == 'q')) {
			ui.closeHelpOverlay()
			return nil
		}
		return ev
	}
	if ui.modalVisible {
		if ev.Key() == tcell.KeyEscape {
			ui.closeModal()
			return nil
		}
		return ev
	}

	if focusIsInput && ev.Key() == tcell.KeyEsc {
		ui.focusPanel(ui.activeCatalogListFocus())
		ui.refreshStatus()
		return nil
	}
	if ui.diceGotoPending {
		if ev.Key() == tcell.KeyEscape {
			ui.diceGotoPending = false
			ui.message = "Goto dadi annullato."
			ui.refreshStatus()
			return nil
		}
		if focus != ui.dice {
			ui.diceGotoPending = false
		} else if ev.Key() == tcell.KeyRune {
			r := ev.Rune()
			switch {
			case r == '^':
				ui.diceGotoPending = false
				ui.jumpToDiceRow(1)
				return nil
			case r == '$':
				ui.diceGotoPending = false
				ui.jumpToDiceRow(len(ui.diceLog))
				return nil
			case r >= '1' && r <= '9':
				ui.diceGotoPending = false
				ui.jumpToDiceRow(int(r - '0'))
				return nil
			}
			ui.diceGotoPending = false
		} else {
			ui.diceGotoPending = false
		}
	}

	switch ev.Key() {
	case tcell.KeyCtrlC:
		ui.app.Stop()
		return nil
	case tcell.KeyEnter:
		if focus == ui.dice {
			ui.rerollSelectedDiceResult()
			return nil
		}
	case tcell.KeyTAB:
		ui.focusNext()
		return nil
	case tcell.KeyBacktab:
		ui.focusPrev()
		return nil
	case tcell.KeyLeft:
		if focus == ui.pngList {
			if ev.Modifiers()&tcell.ModAlt != 0 {
				ui.adjustSelectedPNGArmor(-1)
				return nil
			}
			if ev.Modifiers()&tcell.ModShift != 0 {
				ui.adjustSelectedPNGVitals(-1, 0)
				return nil
			}
			ui.adjustSelectedToken(-1)
			return nil
		}
		if focus == ui.encList && ev.Modifiers()&tcell.ModShift != 0 {
			ui.adjustEncounterWounds(1)
			return nil
		}
	case tcell.KeyRight:
		if focus == ui.pngList {
			if ev.Modifiers()&tcell.ModAlt != 0 {
				ui.adjustSelectedPNGArmor(1)
				return nil
			}
			if ev.Modifiers()&tcell.ModShift != 0 {
				ui.adjustSelectedPNGVitals(1, 0)
				return nil
			}
			ui.adjustSelectedToken(1)
			return nil
		}
		if focus == ui.encList && ev.Modifiers()&tcell.ModShift != 0 {
			ui.adjustEncounterWounds(-1)
			return nil
		}
	case tcell.KeyDown:
		if focus == ui.pngList && ev.Modifiers()&tcell.ModAlt != 0 {
			ui.adjustSelectedPNGHope(-1)
			return nil
		}
		if focus == ui.pngList && ev.Modifiers()&tcell.ModShift != 0 {
			ui.adjustSelectedPNGVitals(0, -1)
			return nil
		}
		if focus == ui.encList && ev.Modifiers()&tcell.ModShift != 0 {
			ui.adjustEncounterStress(-1)
			return nil
		}
	case tcell.KeyUp:
		if focus == ui.pngList && ev.Modifiers()&tcell.ModAlt != 0 {
			ui.adjustSelectedPNGHope(1)
			return nil
		}
		if focus == ui.pngList && ev.Modifiers()&tcell.ModShift != 0 {
			ui.adjustSelectedPNGVitals(0, 1)
			return nil
		}
		if focus == ui.encList && ev.Modifiers()&tcell.ModShift != 0 {
			ui.adjustEncounterStress(1)
			return nil
		}
	case tcell.KeyPgUp:
		if focus == ui.detail || focus == ui.detailTreasure || focus == ui.dice || focus == ui.pngList || focus == ui.encList || focus == ui.notesList || focus == ui.notesSearch || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.envList || focus == ui.envSearch || focus == ui.envTypeDrop || focus == ui.envRankDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop || focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop {
			ui.scrollDetailByPage(-1)
			return nil
		}
	case tcell.KeyPgDn:
		if focus == ui.detail || focus == ui.detailTreasure || focus == ui.dice || focus == ui.pngList || focus == ui.encList || focus == ui.notesList || focus == ui.notesSearch || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.envList || focus == ui.envSearch || focus == ui.envTypeDrop || focus == ui.envRankDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop || focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop {
			ui.scrollDetailByPage(1)
			return nil
		}
	}

	switch ev.Rune() {
	case '?':
		ui.openHelpOverlay(focus)
		return nil
	case 'f':
		if !focusIsInput {
			ui.toggleFullscreenForFocus(focus)
			return nil
		}
	case 'u':
		if !focusIsInput {
			ui.undoLastChange()
			return nil
		}
	case 'r':
		if !focusIsInput {
			ui.redoLastChange()
			return nil
		}
	case 'G':
		ui.openGotoPanelModal()
		return nil
	case 'S':
		if !focusIsInput {
			ui.openStateFileModal("save", "fear")
			return nil
		}
	case 'L':
		if !focusIsInput {
			ui.openStateFileModal("load", "fear")
			return nil
		}
	case 'N':
		ui.catalogMode = "note"
		ui.catalogPanel.SwitchToPage("note")
		ui.refreshCatalogTitles()
		ui.focusPanel(focusNotesList)
		return nil
	case 'q':
		ui.app.Stop()
		return nil
	case 's':
		if !focusIsInput {
			if focus == ui.pngList {
				ui.openStateFileModal("save", "png")
				return nil
			}
			if focus == ui.encList {
				ui.openStateFileModal("save", "encounter")
				return nil
			}
			if focus == ui.dice {
				ui.openStateFileModal("save", "dice")
				return nil
			}
		}
	case 'l':
		if !focusIsInput {
			if focus == ui.pngList {
				ui.openStateFileModal("load", "png")
				return nil
			}
			if focus == ui.encList {
				ui.openStateFileModal("load", "encounter")
				return nil
			}
			if focus == ui.dice {
				ui.openStateFileModal("load", "dice")
				return nil
			}
		}
	case '1':
		ui.focusPanel(focusPNG)
		return nil
	case '2':
		ui.focusPanel(focusEncounter)
		return nil
	case '3':
		ui.focusPanel(ui.activeCatalogListFocus())
		return nil
	case '4':
		ui.catalogMode = "note"
		ui.catalogPanel.SwitchToPage("note")
		ui.refreshCatalogTitles()
		ui.focusPanel(focusNotesList)
		return nil
	case '0':
		ui.focusPanel(focusDice)
		return nil
	case '[':
		ui.switchCatalog(-1)
		return nil
	case ']':
		ui.switchCatalog(1)
		return nil
	case '/':
		if !focusIsInput {
			ui.openRawSearch(focus)
			return nil
		}
	case 'c':
		if focus == ui.dice {
			ui.clearDiceResults()
			return nil
		}
		return nil
	case 'x':
		if focus == ui.pngList {
			ui.deleteSelectedPNG()
			return nil
		}
		return nil
	case 'D':
		if focus == ui.pngList {
			ui.clearAllPNGs()
			return nil
		}
		if focus == ui.encList {
			ui.clearAllEncounter()
			return nil
		}
		return nil
	case 'R':
		if focus == ui.pngList {
			ui.openResetTokensConfirm()
			return nil
		}
	case 'a':
		if focus == ui.pngList {
			ui.openCreatePNGModal()
			return nil
		}
		if focus == ui.notesList || focus == ui.notesSearch {
			ui.openAddNoteModal()
			return nil
		}
		if focus == ui.dice {
			ui.openDiceRollInput()
			return nil
		}
		if ui.catalogMode == "mostri" && (focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop) {
			ui.addSelectedMonsterToEncounter()
			return nil
		}
		if ui.catalogMode == "classe" && (focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop) {
			ui.openClassPNGInput()
			return nil
		}
	case 'n':
		if ui.catalogMode == "mostri" && (focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop) {
			ui.openRandomEncounterFromMonstersInput()
			return nil
		}
	case 'e':
		if focus == ui.notesList || focus == ui.notesSearch {
			ui.openEditNoteModal()
			return nil
		}
		if focus == ui.pngList {
			ui.openEditPNGModal()
			return nil
		}
		if focus == ui.dice {
			ui.openDiceReRollInput()
			return nil
		}
	case 'b':
		if ui.catalogMode == "equipaggiamento" && (focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.detail || focus == ui.detailTreasure) {
			ui.openEquipmentTreasureInput()
			return nil
		}
	case 'U':
		switch ui.catalogMode {
		case "mostri":
			ui.focusPanel(focusMonSearch)
		case "ambienti":
			ui.focusPanel(focusEnvSearch)
		case "equipaggiamento":
			ui.focusPanel(focusEqSearch)
		case "carte":
			ui.focusPanel(focusCardSearch)
		case "note":
			ui.focusPanel(focusNotesSearch)
		default:
			ui.focusPanel(focusClassSearch)
		}
		return nil
	case 't':
		switch ui.catalogMode {
		case "mostri":
			ui.focusPanel(focusMonRole)
		case "ambienti":
			ui.focusPanel(focusEnvType)
		case "equipaggiamento":
			ui.focusPanel(focusEqType)
		case "carte":
			ui.focusPanel(focusCardClass)
		case "note":
			ui.focusPanel(focusNotesSearch)
		default:
			ui.focusPanel(focusClassName)
		}
		return nil
	case 'g':
		if focus == ui.dice {
			ui.diceGotoPending = true
			ui.message = "Goto dadi: g# (1-9), g^ (prima), g$ (ultima)."
			ui.refreshStatus()
			return nil
		}
		switch ui.catalogMode {
		case "mostri":
			ui.focusPanel(focusMonRank)
		case "ambienti":
			ui.focusPanel(focusEnvRank)
		case "equipaggiamento":
			ui.focusPanel(focusEqRank)
		case "carte":
			ui.focusPanel(focusCardType)
		case "note":
			ui.focusPanel(focusNotesSearch)
		default:
			ui.focusPanel(focusClassSubclass)
		}
		return nil
	case 'y':
		if ui.catalogMode == "equipaggiamento" {
			ui.focusPanel(focusEqItemType)
			return nil
		}
	case 'v':
		switch ui.catalogMode {
		case "mostri":
			ui.resetMonsterFilters()
		case "ambienti":
			ui.resetEnvironmentFilters()
		case "equipaggiamento":
			ui.resetEquipmentFilters()
		case "carte":
			ui.resetCardFilters()
		case "note":
			ui.resetNotesFilters()
		default:
			ui.resetClassFilters()
		}
		return nil
	case 'd':
		if focus == ui.dice {
			ui.deleteSelectedDiceResult()
			return nil
		}
		if focus == ui.notesList {
			ui.deleteSelectedNote()
			return nil
		}
		if focus == ui.pngList {
			ui.deleteSelectedPNG()
			return nil
		}
		if focus == ui.encList {
			ui.removeSelectedEncounter()
			return nil
		}
		if ui.catalogMode == "equipaggiamento" && (focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.detail || focus == ui.detailTreasure) {
			ui.toggleDetailsTreasureFocus()
			return nil
		}
	case '+':
		ui.adjustPaure(1)
		return nil
	case '-':
		ui.adjustPaure(-1)
		return nil
	}
	return ev
}

func (ui *tviewUI) focusNext() {
	for i := 0; i < len(ui.focus); i++ {
		ui.focusIdx = (ui.focusIdx + 1) % len(ui.focus)
		if ui.isFocusVisible(ui.focusIdx) {
			ui.app.SetFocus(ui.focus[ui.focusIdx])
			break
		}
	}
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) focusPrev() {
	for i := 0; i < len(ui.focus); i++ {
		ui.focusIdx--
		if ui.focusIdx < 0 {
			ui.focusIdx = len(ui.focus) - 1
		}
		if ui.isFocusVisible(ui.focusIdx) {
			ui.app.SetFocus(ui.focus[ui.focusIdx])
			break
		}
	}
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) focusPanel(panel int) {
	if panel == focusMonList && ui.catalogMode == "ambienti" {
		panel = focusEnvList
	}
	if panel == focusMonSearch && ui.catalogMode == "ambienti" {
		panel = focusEnvSearch
	}
	if panel == focusMonRank && ui.catalogMode == "ambienti" {
		panel = focusEnvRank
	}
	if panel == focusMonRole && ui.catalogMode == "ambienti" {
		panel = focusEnvType
	}
	if panel == focusMonRole && ui.catalogMode == "equipaggiamento" {
		panel = focusEqItemType
	}
	if panel == focusMonSearch && ui.catalogMode == "equipaggiamento" {
		panel = focusEqSearch
	}
	if panel == focusMonRank && ui.catalogMode == "equipaggiamento" {
		panel = focusEqRank
	}
	if panel == focusMonList && ui.catalogMode == "equipaggiamento" {
		panel = focusEqList
	}
	if panel == focusMonRole && ui.catalogMode == "carte" {
		panel = focusCardClass
	}
	if panel == focusMonSearch && ui.catalogMode == "carte" {
		panel = focusCardSearch
	}
	if panel == focusMonRank && ui.catalogMode == "carte" {
		panel = focusCardType
	}
	if panel == focusMonList && ui.catalogMode == "carte" {
		panel = focusCardList
	}
	if panel == focusMonRole && ui.catalogMode == "classe" {
		panel = focusClassName
	}
	if panel == focusMonSearch && ui.catalogMode == "classe" {
		panel = focusClassSearch
	}
	if panel == focusMonRank && ui.catalogMode == "classe" {
		panel = focusClassSubclass
	}
	if panel == focusMonList && ui.catalogMode == "classe" {
		panel = focusClassList
	}
	if panel == focusMonSearch && ui.catalogMode == "note" {
		panel = focusNotesSearch
	}
	if panel == focusMonList && ui.catalogMode == "note" {
		panel = focusNotesList
	}
	if panel == focusMonRole && ui.catalogMode == "note" {
		panel = focusNotesSearch
	}
	if panel == focusMonRank && ui.catalogMode == "note" {
		panel = focusNotesSearch
	}
	if panel < 0 || panel >= len(ui.focus) {
		return
	}
	if !ui.isFocusVisible(panel) {
		return
	}
	ui.focusIdx = panel
	ui.app.SetFocus(ui.focus[panel])
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) isFocusVisible(idx int) bool {
	switch idx {
	case focusMonSearch, focusMonRole, focusMonRank, focusMonList:
		return ui.catalogMode == "mostri"
	case focusEnvSearch, focusEnvType, focusEnvRank, focusEnvList:
		return ui.catalogMode == "ambienti"
	case focusEqSearch, focusEqType, focusEqItemType, focusEqRank, focusEqList:
		return ui.catalogMode == "equipaggiamento"
	case focusCardSearch, focusCardClass, focusCardType, focusCardList:
		return ui.catalogMode == "carte"
	case focusClassSearch, focusClassName, focusClassSubclass, focusClassList:
		return ui.catalogMode == "classe"
	case focusNotesSearch, focusNotesList:
		return ui.catalogMode == "note"
	default:
		return true
	}
}

func (ui *tviewUI) activeCatalogListFocus() int {
	if ui.catalogMode == "ambienti" {
		return focusEnvList
	}
	if ui.catalogMode == "equipaggiamento" {
		return focusEqList
	}
	if ui.catalogMode == "carte" {
		return focusCardList
	}
	if ui.catalogMode == "classe" {
		return focusClassList
	}
	if ui.catalogMode == "note" {
		return focusNotesList
	}
	return focusMonList
}

func (ui *tviewUI) focusActiveCatalogList() {
	if len(ui.focus) == 0 {
		return
	}
	ui.focusPanel(ui.activeCatalogListFocus())
}

func (ui *tviewUI) catalogLabel(mode string) string {
	switch mode {
	case "ambienti":
		return "Ambienti"
	case "equipaggiamento":
		return "Equipaggiamento"
	case "carte":
		return "Carte"
	case "classe":
		return "Classe"
	case "note":
		return "Note"
	default:
		return "Mostri"
	}
}

func (ui *tviewUI) refreshCatalogTitles() {
	order := []string{"mostri", "ambienti", "equipaggiamento", "carte", "classe", "note"}
	for i, mode := range order {
		prev := order[(i-1+len(order))%len(order)]
		next := order[(i+1)%len(order)]
		title := fmt.Sprintf(" [3] %s | '[' %s | ']' %s ", ui.catalogLabel(mode), ui.catalogLabel(prev), ui.catalogLabel(next))
		switch mode {
		case "mostri":
			ui.monstersPanel.SetTitle(title)
		case "ambienti":
			ui.environmentsPanel.SetTitle(title)
		case "equipaggiamento":
			ui.equipmentPanel.SetTitle(title)
		case "carte":
			ui.cardsPanel.SetTitle(title)
		case "classe":
			ui.classesPanel.SetTitle(title)
		case "note":
			ui.notesPanel.SetTitle(title)
		}
	}
}

func (ui *tviewUI) switchCatalog(delta int) {
	if delta == 0 {
		return
	}
	order := []string{"mostri", "ambienti", "equipaggiamento", "carte", "classe", "note"}
	cur := 0
	for i, name := range order {
		if name == ui.catalogMode {
			cur = i
			break
		}
	}
	nextIdx := (cur + delta) % len(order)
	if nextIdx < 0 {
		nextIdx += len(order)
	}
	next := order[nextIdx]
	ui.catalogMode = next
	ui.catalogPanel.SwitchToPage(next)
	ui.refreshCatalogTitles()
	switch next {
	case "ambienti":
		ui.message = "Catalogo: Ambienti"
	case "equipaggiamento":
		ui.message = "Catalogo: Equipaggiamento"
	case "carte":
		ui.message = "Catalogo: Carte"
	case "classe":
		ui.message = "Catalogo: Classe"
	case "note":
		ui.message = "Catalogo: Note"
	default:
		ui.message = "Catalogo: Mostri"
	}
	ui.focusPanel(ui.activeCatalogListFocus())
	ui.refreshStatus()
}

func (ui *tviewUI) refreshAll() {
	ui.refreshPNGs()
	ui.refreshMonsters()
	ui.refreshEnvironments()
	ui.refreshEquipment()
	ui.refreshCards()
	ui.refreshClasses()
	ui.refreshNotes()
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) refreshPNGs() {
	current := ui.selected
	if current < 0 && len(ui.pngs) > 0 {
		current = 0
	}
	ui.pngList.Clear()
	if len(ui.pngs) == 0 {
		ui.pngList.AddItem("(nessun PNG)", "", 0, nil)
		return
	}
	for _, p := range ui.pngs {
		label := fmt.Sprintf("%s [token %d | PF %d | ST %d | ARM %d | SPE %d]", p.Name, p.Token, p.PF, p.Stress, p.ArmorScore, p.Hope)
		ui.pngList.AddItem(label, "", 0, nil)
	}
	if current >= len(ui.pngs) {
		current = len(ui.pngs) - 1
	}
	if current >= 0 {
		ui.pngList.SetCurrentItem(current)
		ui.selected = current
	}
}

func (ui *tviewUI) refreshMonsters() {
	query := strings.ToLower(strings.TrimSpace(ui.search.GetText()))
	ui.filtered = ui.filtered[:0]
	for i, m := range ui.monsters {
		if query != "" && !strings.Contains(strings.ToLower(m.Name), query) {
			continue
		}
		if ui.roleFilter != "" && ui.roleFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(m.Role), ui.roleFilter) {
			continue
		}
		if ui.rankFilter != "" && ui.rankFilter != "Tutti" && strconv.Itoa(m.Rank) != ui.rankFilter {
			continue
		}
		ui.filtered = append(ui.filtered, i)
	}

	// During initial build dropdown callbacks can fire before the list is created.
	if ui.monList == nil {
		return
	}

	current := ui.monList.GetCurrentItem()
	ui.monList.Clear()
	if len(ui.filtered) == 0 {
		ui.monList.AddItem("(nessun mostro)", "", 0, nil)
		return
	}
	for _, idx := range ui.filtered {
		m := ui.monsters[idx]
		ui.monList.AddItem(fmt.Sprintf("%s [R%d] PF:%d", m.Name, m.Rank, m.PF), "", 0, nil)
	}
	if current >= len(ui.filtered) {
		current = len(ui.filtered) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.monList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshEnvironments() {
	query := strings.ToLower(strings.TrimSpace(ui.envSearch.GetText()))
	ui.filteredEnv = ui.filteredEnv[:0]
	for i, e := range ui.environments {
		if query != "" && !strings.Contains(strings.ToLower(e.Name), query) {
			continue
		}
		if ui.envTypeFilter != "" && ui.envTypeFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(e.Kind), ui.envTypeFilter) {
			continue
		}
		if ui.envRankFilter != "" && ui.envRankFilter != "Tutti" && strconv.Itoa(e.Rank) != ui.envRankFilter {
			continue
		}
		ui.filteredEnv = append(ui.filteredEnv, i)
	}

	if ui.envList == nil {
		return
	}

	current := ui.envList.GetCurrentItem()
	ui.envList.Clear()
	if len(ui.filteredEnv) == 0 {
		ui.envList.AddItem("(nessun ambiente)", "", 0, nil)
		return
	}
	for _, idx := range ui.filteredEnv {
		e := ui.environments[idx]
		ui.envList.AddItem(fmt.Sprintf("%s [%s R%d]", e.Name, e.Kind, e.Rank), "", 0, nil)
	}
	if current >= len(ui.filteredEnv) {
		current = len(ui.filteredEnv) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.envList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshEquipment() {
	query := strings.ToLower(strings.TrimSpace(ui.eqSearch.GetText()))
	ui.filteredEq = ui.filteredEq[:0]
	for i, it := range ui.equipment {
		if query != "" && !strings.Contains(strings.ToLower(it.Name), query) {
			continue
		}
		if ui.eqTypeFilter != "" && ui.eqTypeFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(it.Category), ui.eqTypeFilter) {
			continue
		}
		if ui.eqItemTypeFilter != "" && ui.eqItemTypeFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(it.Type), ui.eqItemTypeFilter) {
			continue
		}
		if ui.eqRankFilter != "" && ui.eqRankFilter != "Tutti" && strconv.Itoa(it.Rank) != ui.eqRankFilter {
			continue
		}
		ui.filteredEq = append(ui.filteredEq, i)
	}

	if ui.eqList == nil {
		return
	}
	current := ui.eqList.GetCurrentItem()
	ui.eqList.Clear()
	if len(ui.filteredEq) == 0 {
		ui.eqList.AddItem("(nessun equipaggiamento)", "", 0, nil)
		return
	}
	for _, idx := range ui.filteredEq {
		it := ui.equipment[idx]
		ui.eqList.AddItem(fmt.Sprintf("%s [%s R%d]", it.Name, it.Category, it.Rank), "", 0, nil)
	}
	if current >= len(ui.filteredEq) {
		current = len(ui.filteredEq) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.eqList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshCards() {
	query := strings.ToLower(strings.TrimSpace(ui.cardSearch.GetText()))
	ui.filteredCards = ui.filteredCards[:0]
	for i, c := range ui.cards {
		if query != "" && !strings.Contains(strings.ToLower(c.Name), query) {
			continue
		}
		if ui.cardClassFilter != "" && ui.cardClassFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Class), ui.cardClassFilter) {
			continue
		}
		if ui.cardTypeFilter != "" && ui.cardTypeFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Type), ui.cardTypeFilter) {
			continue
		}
		ui.filteredCards = append(ui.filteredCards, i)
	}

	if ui.cardList == nil {
		return
	}
	current := ui.cardList.GetCurrentItem()
	ui.cardList.Clear()
	if len(ui.filteredCards) == 0 {
		ui.cardList.AddItem("(nessuna carta)", "", 0, nil)
		return
	}
	for _, idx := range ui.filteredCards {
		c := ui.cards[idx]
		head := cardDescriptionHead(c.Description)
		label := fmt.Sprintf("%s [%s - %s]", c.Name, c.Class, c.Type)
		if head != "" {
			label = fmt.Sprintf("%s | %s", head, label)
		}
		ui.cardList.AddItem(label, "", 0, nil)
	}
	if current >= len(ui.filteredCards) {
		current = len(ui.filteredCards) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.cardList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshClasses() {
	query := strings.ToLower(strings.TrimSpace(ui.classSearch.GetText()))
	ui.filteredClasses = ui.filteredClasses[:0]
	for i, c := range ui.classes {
		if query != "" {
			text := strings.ToLower(strings.TrimSpace(c.Name) + " " + strings.TrimSpace(c.Subclass))
			if !strings.Contains(text, query) {
				continue
			}
		}
		if ui.classNameFilter != "" && ui.classNameFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Name), ui.classNameFilter) {
			continue
		}
		if ui.classSubFilter != "" && ui.classSubFilter != "Tutti" && !strings.EqualFold(strings.TrimSpace(c.Subclass), ui.classSubFilter) {
			continue
		}
		ui.filteredClasses = append(ui.filteredClasses, i)
	}

	if ui.classList == nil {
		return
	}
	current := ui.classList.GetCurrentItem()
	ui.classList.Clear()
	if len(ui.filteredClasses) == 0 {
		ui.classList.AddItem("(nessuna classe)", "", 0, nil)
		return
	}
	for _, idx := range ui.filteredClasses {
		c := ui.classes[idx]
		ui.classList.AddItem(fmt.Sprintf("%s | %s", c.Subclass, c.Name), "", 0, nil)
	}
	if current >= len(ui.filteredClasses) {
		current = len(ui.filteredClasses) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.classList.SetCurrentItem(current)
}

func cardDescriptionHead(desc string) string {
	s := strings.TrimSpace(desc)
	if s == "" || strings.EqualFold(s, "Da screenshot.") {
		return ""
	}
	if i := strings.Index(s, ":"); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func (ui *tviewUI) refreshEncounter() {
	current := ui.encList.GetCurrentItem()
	ui.encList.Clear()
	if len(ui.encounter) == 0 {
		ui.encList.AddItem("(vuoto)", "", 0, nil)
		return
	}
	for i, e := range ui.encounter {
		base := e.BasePF
		if base == 0 {
			base = e.Monster.PF
		}
		baseStress := e.BaseStress
		if baseStress == 0 {
			baseStress = e.Monster.Stress
		}
		currentStress := max(e.Stress, 0)
		if baseStress > 0 && currentStress > baseStress {
			currentStress = baseStress
		}
		remaining := max(base-e.Wounds, 0)
		label := ui.encounterLabelAt(i)
		ui.encList.AddItem(fmt.Sprintf("%s [PF %d/%d | ST %d/%d]", label, remaining, base, currentStress, baseStress), "", 0, nil)
	}
	if current >= len(ui.encounter) {
		current = len(ui.encounter) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.encList.SetCurrentItem(current)
}

func (ui *tviewUI) refreshNotes() {
	if ui.notesList == nil {
		return
	}
	query := ""
	if ui.notesSearch != nil {
		query = strings.ToLower(strings.TrimSpace(ui.notesSearch.GetText()))
	}
	current := ui.notesList.GetCurrentItem()
	ui.notesList.Clear()
	ui.filteredNotes = ui.filteredNotes[:0]
	for i, note := range ui.notes {
		if query == "" || strings.Contains(strings.ToLower(note), query) {
			ui.filteredNotes = append(ui.filteredNotes, i)
		}
	}
	if len(ui.filteredNotes) == 0 {
		ui.notesList.AddItem("(nessuna nota) premi 'a' per aggiungere", "", 0, nil)
		ui.notesList.SetCurrentItem(0)
		return
	}
	if len(ui.notes) == 0 {
		ui.notesList.AddItem("(vuoto) premi 'a' per aggiungere", "", 0, nil)
		ui.notesList.SetCurrentItem(0)
		return
	}
	for _, idx := range ui.filteredNotes {
		note := ui.notes[idx]
		title := strings.TrimSpace(note)
		if idx := strings.Index(title, "\n"); idx >= 0 {
			title = strings.TrimSpace(title[:idx])
		}
		if title == "" {
			title = "(nota senza titolo)"
		}
		ui.notesList.AddItem(fmt.Sprintf("%d) %s", idx+1, title), "", 0, nil)
	}
	if current >= len(ui.filteredNotes) {
		current = len(ui.filteredNotes) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.notesList.SetCurrentItem(current)
}

func (ui *tviewUI) currentNoteIndex() int {
	if len(ui.filteredNotes) == 0 {
		return -1
	}
	if ui.notesList == nil {
		return -1
	}
	cur := ui.notesList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredNotes) {
		return -1
	}
	return ui.filteredNotes[cur]
}

func (ui *tviewUI) refreshDetail() {
	if ui.detail == nil {
		return
	}
	focus := ui.app.GetFocus()
	if focus == ui.detailTreasure {
		ui.renderTreasure()
		return
	}
	if focus == ui.dice {
		ui.detailRaw = ui.buildDiceDetail()
		ui.renderDetail()
		return
	}
	if focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop {
		idx := ui.currentMonsterIndex()
		if idx < 0 {
			ui.detailRaw = "Nessun mostro selezionato."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildMonsterDetails(ui.monsters[idx], ui.monsters[idx].Name, "")
		ui.renderDetail()
		return
	}
	if focus == ui.envList || focus == ui.envSearch || focus == ui.envTypeDrop || focus == ui.envRankDrop {
		idx := ui.currentEnvironmentIndex()
		if idx < 0 {
			ui.detailRaw = "Nessun ambiente selezionato."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildEnvironmentDetails(ui.environments[idx])
		ui.renderDetail()
		return
	}
	if focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop {
		idx := ui.currentEquipmentIndex()
		if idx < 0 {
			ui.detailRaw = "Nessun equipaggiamento selezionato."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildEquipmentDetails(ui.equipment[idx])
		ui.renderDetail()
		return
	}
	if focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop {
		idx := ui.currentCardIndex()
		if idx < 0 {
			ui.detailRaw = "Nessuna carta selezionata."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildCardDetails(ui.cards[idx])
		ui.renderDetail()
		return
	}
	if focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop {
		idx := ui.currentClassIndex()
		if idx < 0 {
			ui.detailRaw = "Nessuna classe selezionata."
			ui.renderDetail()
			return
		}
		ui.detailRaw = ui.buildClassDetails(ui.classes[idx])
		ui.renderDetail()
		return
	}
	if focus == ui.encList {
		idx := ui.currentEncounterIndex()
		if idx < 0 {
			ui.detailRaw = "Encounter vuoto."
			ui.renderDetail()
			return
		}
		e := ui.encounter[idx]
		base := e.BasePF
		if base == 0 {
			base = e.Monster.PF
		}
		remaining := max(base-e.Wounds, 0)
		baseStress := e.BaseStress
		if baseStress == 0 {
			baseStress = e.Monster.Stress
		}
		currentStress := max(e.Stress, 0)
		if baseStress > 0 && currentStress > baseStress {
			currentStress = baseStress
		}
		extra := fmt.Sprintf("PF correnti: %d/%d | Ferite: %d | Stress: %d/%d", remaining, base, e.Wounds, currentStress, baseStress)
		ui.detailRaw = ui.buildMonsterDetails(e.Monster, ui.encounterLabelAt(idx), extra)
		ui.renderDetail()
		return
	}
	if focus == ui.notesList || focus == ui.notesSearch {
		idx := ui.currentNoteIndex()
		if idx < 0 || idx >= len(ui.notes) {
			ui.detailRaw = "Nessuna nota."
			ui.renderDetail()
			return
		}
		ui.detailRaw = fmt.Sprintf("Nota %d\n\n%s", idx+1, strings.TrimSpace(ui.notes[idx]))
		ui.renderDetail()
		return
	}
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.detailRaw = "Nessun PNG selezionato."
		ui.renderDetail()
		return
	}
	p := ui.pngs[ui.selected]
	var b strings.Builder
	fmt.Fprintf(&b, "%s\nToken: %d", p.Name, p.Token)
	fmt.Fprintf(&b, "\nPF: %d | Stress: %d | Armatura: %d | Speranza: %d", p.PF, p.Stress, p.ArmorScore, p.Hope)
	if strings.TrimSpace(p.Class) != "" || strings.TrimSpace(p.Subclass) != "" || p.Level > 0 {
		classLine := ""
		if strings.TrimSpace(p.Subclass) != "" {
			classLine += strings.TrimSpace(p.Subclass)
		}
		if strings.TrimSpace(p.Class) != "" {
			if classLine != "" {
				classLine += " | "
			}
			classLine += strings.TrimSpace(p.Class)
		}
		if p.Level > 0 {
			if classLine != "" {
				classLine += " "
			}
			classLine += fmt.Sprintf("L%d", p.Level)
		}
		if classLine != "" {
			b.WriteString("\nClasse: " + classLine)
		}
	}
	if p.Level > 0 {
		rank := p.Rank
		if rank <= 0 {
			rank = rankFromLevel(p.Level)
		}
		compBonus := p.CompBonus
		expBonus := p.ExpBonus
		if compBonus == 0 && p.Level > 1 {
			compBonus = progressionBonusAtLevel(p.Level)
		}
		if expBonus == 0 && p.Level > 1 {
			expBonus = progressionBonusAtLevel(p.Level)
		}
		fmt.Fprintf(&b, "\nLivello: %d | Rango: %d", p.Level, rank)
		fmt.Fprintf(&b, "\nBonus Competenza (livello): +%d", compBonus)
		fmt.Fprintf(&b, "\nEsperienze aggiuntive (livello): +%d", expBonus)
	}
	if def := ui.findClassDefinition(p.Class, p.Subclass); def != nil {
		if def.Evasion > 0 {
			fmt.Fprintf(&b, "\nEvasione iniziale: %d", def.Evasion)
		}
		if def.HP > 0 {
			fmt.Fprintf(&b, "\nPF iniziali: %d", def.HP)
		}
		if p.Level > 0 {
			b.WriteString("\nRegola soglie: aggiungi il livello attuale alle soglie base dell'armatura.")
		}
		if strings.TrimSpace(def.CasterTrait) != "" {
			b.WriteString("\nTratto da Incantatore: " + strings.TrimSpace(def.CasterTrait))
		}
		if strings.TrimSpace(def.HopePrivilege) != "" {
			b.WriteString("\n\nPrivilegio della Speranza:\n" + strings.TrimSpace(def.HopePrivilege))
		}
		if len(def.ClassPrivileges) > 0 {
			b.WriteString("\n\nPrivilegi di Classe:")
			for _, it := range def.ClassPrivileges {
				it = strings.TrimSpace(it)
				if it == "" {
					continue
				}
				b.WriteString("\n- " + it)
			}
		}
		if len(def.BasePrivileges) > 0 {
			b.WriteString("\n\nPrivilegi Base:")
			for _, it := range def.BasePrivileges {
				it = strings.TrimSpace(it)
				if it == "" {
					continue
				}
				b.WriteString("\n- " + it)
			}
		}
		if strings.TrimSpace(def.Specialization) != "" {
			b.WriteString("\n\nSpecializzazione:\n" + strings.TrimSpace(def.Specialization))
		}
		if strings.TrimSpace(def.Mastery) != "" {
			b.WriteString("\n\nMaestria:\n" + strings.TrimSpace(def.Mastery))
		}
	}
	if strings.TrimSpace(p.Traits) != "" {
		b.WriteString("\n\nTratti consigliati:\n" + strings.TrimSpace(p.Traits))
	}
	if strings.TrimSpace(p.Primary) != "" {
		b.WriteString("\nArma primaria consigliata:\n" + strings.TrimSpace(p.Primary))
	}
	if strings.TrimSpace(p.Secondary) != "" {
		b.WriteString("\nArma secondaria consigliata:\n" + strings.TrimSpace(p.Secondary))
	}
	if strings.TrimSpace(p.Armor) != "" {
		b.WriteString("\nArmatura consigliata:\n" + strings.TrimSpace(p.Armor))
	}
	if strings.TrimSpace(p.Inventory) != "" {
		b.WriteString("\nInventario suggerito:\n" + strings.TrimSpace(p.Inventory))
	}
	if strings.TrimSpace(p.Look) != "" {
		b.WriteString("\nDescrizione scelta:\n" + strings.TrimSpace(p.Look))
	}
	if strings.TrimSpace(p.Description) != "" {
		b.WriteString("\n\nDescrizione:\n" + strings.TrimSpace(p.Description))
	}
	ui.detailRaw = b.String()
	ui.renderDetail()
}

func (ui *tviewUI) renderDetail() {
	if ui.detail == nil {
		return
	}
	text := ui.detailRaw
	if strings.TrimSpace(text) == "" {
		text = "Nessun dettaglio."
	}
	out := tview.Escape(text)
	lines := strings.Split(out, "\n")
	if len(lines) > 0 {
		lines[0] = "[yellow]" + lines[0] + "[-]"
		out = strings.Join(lines, "\n")
	}
	if strings.TrimSpace(ui.detailQuery) != "" {
		out = highlightMatches(out, ui.detailQuery)
	}
	ui.detail.SetText(out)
}

func highlightMatches(text, query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return text
	}
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(q))
	if err != nil {
		return text
	}
	return re.ReplaceAllStringFunc(text, func(m string) string {
		return "[black:gold]" + m + "[-:-]"
	})
}

func (ui *tviewUI) refreshStatus() {
	ui.status.SetText(fmt.Sprintf("%s | Paure [black:gold]%d/12[-:-]", helpText, clampFear(ui.paure)))
}

func (ui *tviewUI) currentMonsterIndex() int {
	if len(ui.filtered) == 0 {
		return -1
	}
	if ui.monList == nil {
		return -1
	}
	cur := ui.monList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filtered) {
		return -1
	}
	return ui.filtered[cur]
}

func (ui *tviewUI) currentEnvironmentIndex() int {
	if len(ui.filteredEnv) == 0 {
		return -1
	}
	if ui.envList == nil {
		return -1
	}
	cur := ui.envList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredEnv) {
		return -1
	}
	return ui.filteredEnv[cur]
}

func (ui *tviewUI) currentEquipmentIndex() int {
	if len(ui.filteredEq) == 0 || ui.eqList == nil {
		return -1
	}
	cur := ui.eqList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredEq) {
		return -1
	}
	return ui.filteredEq[cur]
}

func (ui *tviewUI) currentCardIndex() int {
	if len(ui.filteredCards) == 0 || ui.cardList == nil {
		return -1
	}
	cur := ui.cardList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredCards) {
		return -1
	}
	return ui.filteredCards[cur]
}

func (ui *tviewUI) currentClassIndex() int {
	if len(ui.filteredClasses) == 0 || ui.classList == nil {
		return -1
	}
	cur := ui.classList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.filteredClasses) {
		return -1
	}
	return ui.filteredClasses[cur]
}

func (ui *tviewUI) buildMonsterFilterOptions() ([]string, []string) {
	roleSet := map[string]struct{}{}
	rankSet := map[int]struct{}{}

	for _, m := range ui.monsters {
		role := strings.TrimSpace(m.Role)
		if role != "" {
			roleSet[role] = struct{}{}
		}
		if m.Rank > 0 {
			rankSet[m.Rank] = struct{}{}
		}
	}

	roles := make([]string, 0, len(roleSet)+1)
	roles = append(roles, "Tutti")
	for role := range roleSet {
		roles = append(roles, role)
	}
	sort.Strings(roles[1:])

	ranksInt := make([]int, 0, len(rankSet))
	for rank := range rankSet {
		ranksInt = append(ranksInt, rank)
	}
	sort.Ints(ranksInt)

	ranks := make([]string, 0, len(ranksInt)+1)
	ranks = append(ranks, "Tutti")
	for _, rank := range ranksInt {
		ranks = append(ranks, strconv.Itoa(rank))
	}

	return roles, ranks
}

func (ui *tviewUI) buildEnvironmentRankOptions() []string {
	rankSet := map[int]struct{}{}
	for _, e := range ui.environments {
		if e.Rank > 0 {
			rankSet[e.Rank] = struct{}{}
		}
	}
	ranksInt := make([]int, 0, len(rankSet))
	for rank := range rankSet {
		ranksInt = append(ranksInt, rank)
	}
	sort.Ints(ranksInt)
	ranks := make([]string, 0, len(ranksInt)+1)
	ranks = append(ranks, "Tutti")
	for _, rank := range ranksInt {
		ranks = append(ranks, strconv.Itoa(rank))
	}
	return ranks
}

func (ui *tviewUI) buildEnvironmentTypeOptions() []string {
	typeSet := map[string]struct{}{}
	for _, e := range ui.environments {
		kind := strings.TrimSpace(e.Kind)
		if kind != "" {
			typeSet[kind] = struct{}{}
		}
	}
	types := make([]string, 0, len(typeSet)+1)
	types = append(types, "Tutti")
	for kind := range typeSet {
		types = append(types, kind)
	}
	sort.Strings(types[1:])
	return types
}

func (ui *tviewUI) buildEquipmentTypeOptions() []string {
	typeSet := map[string]struct{}{}
	for _, it := range ui.equipment {
		k := strings.TrimSpace(it.Category)
		if k != "" {
			typeSet[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(typeSet)+1)
	opts = append(opts, "Tutti")
	for k := range typeSet {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildEquipmentItemTypeOptions() []string {
	typeSet := map[string]struct{}{}
	for _, it := range ui.equipment {
		k := strings.TrimSpace(it.Type)
		if k != "" {
			typeSet[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(typeSet)+1)
	opts = append(opts, "Tutti")
	for k := range typeSet {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildEquipmentRankOptions() []string {
	set := map[int]struct{}{}
	for _, it := range ui.equipment {
		if it.Rank > 0 {
			set[it.Rank] = struct{}{}
		}
	}
	ints := make([]int, 0, len(set))
	for r := range set {
		ints = append(ints, r)
	}
	sort.Ints(ints)
	opts := make([]string, 0, len(ints)+1)
	opts = append(opts, "Tutti")
	for _, r := range ints {
		opts = append(opts, strconv.Itoa(r))
	}
	return opts
}

func (ui *tviewUI) buildCardClassOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.cards {
		k := strings.TrimSpace(c.Class)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildCardTypeOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.cards {
		k := strings.TrimSpace(c.Type)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildClassNameOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.classes {
		k := strings.TrimSpace(c.Name)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) buildClassSubclassOptions() []string {
	set := map[string]struct{}{}
	for _, c := range ui.classes {
		k := strings.TrimSpace(c.Subclass)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	opts := make([]string, 0, len(set)+1)
	opts = append(opts, "Tutti")
	for k := range set {
		opts = append(opts, k)
	}
	sort.Strings(opts[1:])
	return opts
}

func (ui *tviewUI) resetMonsterFilters() {
	ui.roleFilter = "Tutti"
	ui.rankFilter = "Tutti"
	ui.search.SetText("")
	if ui.roleDrop != nil {
		ui.roleDrop.SetCurrentOption(0)
	}
	if ui.rankDrop != nil {
		ui.rankDrop.SetCurrentOption(0)
	}
	ui.refreshMonsters()
	ui.refreshDetail()
	ui.message = "Filtri Mostri resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetEnvironmentFilters() {
	ui.envTypeFilter = "Tutti"
	ui.envRankFilter = "Tutti"
	ui.envSearch.SetText("")
	if ui.envTypeDrop != nil {
		ui.envTypeDrop.SetCurrentOption(0)
	}
	if ui.envRankDrop != nil {
		ui.envRankDrop.SetCurrentOption(0)
	}
	ui.refreshEnvironments()
	ui.refreshDetail()
	ui.message = "Filtri Ambienti resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetEquipmentFilters() {
	ui.eqTypeFilter = "Tutti"
	ui.eqItemTypeFilter = "Tutti"
	ui.eqRankFilter = "Tutti"
	ui.eqSearch.SetText("")
	if ui.eqTypeDrop != nil {
		ui.eqTypeDrop.SetCurrentOption(0)
	}
	if ui.eqItemTypeDrop != nil {
		ui.eqItemTypeDrop.SetCurrentOption(0)
	}
	if ui.eqRankDrop != nil {
		ui.eqRankDrop.SetCurrentOption(0)
	}
	ui.refreshEquipment()
	ui.refreshDetail()
	ui.message = "Filtri Equipaggiamento resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetCardFilters() {
	ui.cardClassFilter = "Tutti"
	ui.cardTypeFilter = "Tutti"
	ui.cardSearch.SetText("")
	if ui.cardClassDrop != nil {
		ui.cardClassDrop.SetCurrentOption(0)
	}
	if ui.cardTypeDrop != nil {
		ui.cardTypeDrop.SetCurrentOption(0)
	}
	ui.refreshCards()
	ui.refreshDetail()
	ui.message = "Filtri Carte resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetClassFilters() {
	ui.classNameFilter = "Tutti"
	ui.classSubFilter = "Tutti"
	ui.classSearch.SetText("")
	if ui.classNameDrop != nil {
		ui.classNameDrop.SetCurrentOption(0)
	}
	if ui.classSubDrop != nil {
		ui.classSubDrop.SetCurrentOption(0)
	}
	ui.refreshClasses()
	ui.refreshDetail()
	ui.message = "Filtri Classe resettati."
	ui.refreshStatus()
}

func (ui *tviewUI) resetNotesFilters() {
	if ui.notesSearch != nil {
		ui.notesSearch.SetText("")
	}
	ui.refreshNotes()
	ui.refreshDetail()
	ui.message = "Filtro Note resettato."
	ui.refreshStatus()
}

func (ui *tviewUI) buildEnvironmentDetails(e Environment) string {
	var b strings.Builder
	b.WriteString(e.Name + "\n")
	fmt.Fprintf(&b, "Tipo: %s | Rango: %d\n", e.Kind, e.Rank)
	if strings.TrimSpace(e.Difficulty) != "" {
		b.WriteString("Difficoltà: " + strings.TrimSpace(e.Difficulty) + "\n")
	}
	if strings.TrimSpace(e.Impeti) != "" {
		b.WriteString("Impeti: " + strings.TrimSpace(e.Impeti) + "\n")
	}
	if strings.TrimSpace(e.PotentialAdversaries) != "" {
		b.WriteString("Potenziali Avversari: " + strings.TrimSpace(e.PotentialAdversaries) + "\n")
	}
	if len(e.Characteristics) > 0 {
		b.WriteString("\nCaratteristiche:\n")
		for _, c := range e.Characteristics {
			line := "- " + c.Name
			if strings.TrimSpace(c.Kind) != "" {
				line += " (" + c.Kind + ")"
			}
			if strings.TrimSpace(c.Text) != "" {
				line += ": " + strings.TrimSpace(c.Text)
			}
			b.WriteString(line + "\n")
		}
	}
	if strings.TrimSpace(e.Description) != "" {
		b.WriteString("\n" + strings.TrimSpace(e.Description))
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) buildEquipmentDetails(it EquipmentItem) string {
	hasValue := func(v string) bool {
		s := strings.TrimSpace(v)
		return s != "" && s != "—" && s != "-"
	}

	var b strings.Builder
	b.WriteString(it.Name + "\n")
	fmt.Fprintf(&b, "Categoria: %s | Tipo: %s | Rango: %d", it.Category, it.Type, it.Rank)
	if hasValue(it.Levels) {
		fmt.Fprintf(&b, " | Livelli: %s", it.Levels)
	}
	b.WriteString("\n")

	if strings.EqualFold(strings.TrimSpace(it.Type), "armatura") {
		if hasValue(it.Trait) {
			b.WriteString("Soglie base: " + strings.TrimSpace(it.Trait) + "\n")
		}
		if hasValue(it.Range) {
			b.WriteString("Punteggio base: " + strings.TrimSpace(it.Range) + "\n")
		}
	} else if strings.EqualFold(strings.TrimSpace(it.Type), "bottino") || strings.EqualFold(strings.TrimSpace(it.Type), "scorte") {
		if hasValue(it.Trait) {
			b.WriteString("Tiro: " + strings.TrimSpace(it.Trait) + "\n")
		}
	} else {
		if hasValue(it.Trait) {
			b.WriteString("Tratto: " + strings.TrimSpace(it.Trait) + "\n")
		}
		if hasValue(it.Range) {
			b.WriteString("Portata: " + strings.TrimSpace(it.Range) + "\n")
		}
		if hasValue(it.Damage) {
			b.WriteString("Danno: " + strings.TrimSpace(it.Damage) + "\n")
		}
		if hasValue(it.Grip) {
			b.WriteString("Impugnatura: " + strings.TrimSpace(it.Grip) + "\n")
		}
	}
	if hasValue(it.Characteristic) {
		b.WriteString("\nCaratteristica:\n" + strings.TrimSpace(it.Characteristic))
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) buildCardDetails(c CardItem) string {
	var b strings.Builder
	b.WriteString(c.Name + "\n")
	fmt.Fprintf(&b, "Classe: %s | Tipo: %s\n", strings.TrimSpace(c.Class), strings.TrimSpace(c.Type))
	if strings.TrimSpace(c.CasterTrait) != "" {
		b.WriteString("Tratto da Incantatore: " + strings.TrimSpace(c.CasterTrait) + "\n")
	}
	if strings.TrimSpace(c.Description) != "" {
		b.WriteString("\n" + strings.TrimSpace(c.Description) + "\n")
	}
	if len(c.Effects) > 0 {
		b.WriteString("\nEffetti:\n")
		for _, e := range c.Effects {
			if strings.TrimSpace(e) == "" {
				continue
			}
			b.WriteString("- " + strings.TrimSpace(e) + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) buildClassDetails(c ClassItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s - %s\n", c.Name, c.Subclass)
	fmt.Fprintf(&b, "Rango: %d\n", c.Rank)
	if strings.TrimSpace(c.Domains) != "" {
		b.WriteString("Domini: " + strings.TrimSpace(c.Domains) + "\n")
	}
	if c.Evasion > 0 {
		fmt.Fprintf(&b, "Evasione iniziale: %d\n", c.Evasion)
	}
	if c.HP > 0 {
		fmt.Fprintf(&b, "Punti Ferita iniziali: %d\n", c.HP)
	}
	if strings.TrimSpace(c.ClassItem) != "" {
		b.WriteString("Oggetti di classe: " + strings.TrimSpace(c.ClassItem) + "\n")
	}
	if strings.TrimSpace(c.HopePrivilege) != "" {
		b.WriteString("\nPrivilegio della Speranza:\n" + strings.TrimSpace(c.HopePrivilege) + "\n")
	}
	if len(c.ClassPrivileges) > 0 {
		b.WriteString("\nPrivilegi di Classe:\n")
		for _, p := range c.ClassPrivileges {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			b.WriteString("- " + p + "\n")
		}
	}
	if strings.TrimSpace(c.Description) != "" {
		b.WriteString("\nDescrizione:\n" + strings.TrimSpace(c.Description) + "\n")
	}
	if strings.TrimSpace(c.CasterTrait) != "" {
		b.WriteString("\nTratto da Incantatore:\n" + strings.TrimSpace(c.CasterTrait) + "\n")
	}
	if len(c.BasePrivileges) > 0 {
		b.WriteString("\nPrivilegi Base:\n")
		for _, p := range c.BasePrivileges {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			b.WriteString("- " + p + "\n")
		}
	}
	if strings.TrimSpace(c.Specialization) != "" {
		b.WriteString("\nSpecializzazione:\n" + strings.TrimSpace(c.Specialization) + "\n")
	}
	if strings.TrimSpace(c.Mastery) != "" {
		b.WriteString("\nMaestria:\n" + strings.TrimSpace(c.Mastery) + "\n")
	}
	if len(c.BackgroundQs) > 0 {
		b.WriteString("\nDomande sul Background:\n")
		for _, q := range c.BackgroundQs {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			b.WriteString("- " + q + "\n")
		}
	}
	if len(c.Bonds) > 0 {
		b.WriteString("\nLegami:\n")
		for _, q := range c.Bonds {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			b.WriteString("- " + q + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) currentEncounterIndex() int {
	if len(ui.encounter) == 0 {
		return -1
	}
	cur := ui.encList.GetCurrentItem()
	if cur < 0 || cur >= len(ui.encounter) {
		return -1
	}
	return cur
}

func (ui *tviewUI) buildMonsterDetails(m Monster, title string, extraLine string) string {
	var b strings.Builder
	b.WriteString(title + "\n")
	fmt.Fprintf(&b, "Ruolo: %s | Rango: %d\n", m.Role, m.Rank)
	if extraLine != "" {
		b.WriteString(extraLine + "\n")
	}
	fmt.Fprintf(&b, "PF: %d | Stress: %d | Difficoltà: %d\n", m.PF, m.Stress, m.Difficulty)
	if th := formatThresholds(m.Thresholds); th != "" {
		b.WriteString("Soglie: " + th + "\n")
	}
	if m.Attack.Name != "" {
		bonus := strings.TrimSpace(m.Attack.Bonus)
		bonus = strings.ReplaceAll(bonus, "−", "-")
		bonus = strings.ReplaceAll(bonus, "–", "-")
		if bonus != "" && !strings.HasPrefix(bonus, "+") && !strings.HasPrefix(bonus, "-") {
			bonus = "+" + bonus
		}
		if bonus != "" {
			fmt.Fprintf(&b, "Attacco: %s (%s) %s %s (%s)\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType, bonus)
		} else {
			fmt.Fprintf(&b, "Attacco: %s (%s) %s %s\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType)
		}
	}
	if strings.TrimSpace(m.MotivationsTactics) != "" {
		b.WriteString("\nMotivazioni/Tattiche:\n" + strings.TrimSpace(m.MotivationsTactics) + "\n")
	}
	if len(m.Traits) > 0 {
		b.WriteString("\nTratti:\n")
		for _, t := range m.Traits {
			line := "- " + t.Name
			if strings.TrimSpace(t.Kind) != "" {
				line += " (" + t.Kind + ")"
			}
			if strings.TrimSpace(t.Text) != "" {
				line += ": " + strings.TrimSpace(t.Text)
			}
			b.WriteString(line + "\n")
		}
	}
	if strings.TrimSpace(m.Description) != "" {
		b.WriteString("\n" + strings.TrimSpace(m.Description))
	}
	return strings.TrimSpace(b.String())
}

func formatThresholds(t Thresholds) string {
	if len(t.Values) > 0 {
		var parts []string
		for _, v := range t.Values {
			parts = append(parts, fmt.Sprintf("%d", v))
		}
		return strings.Join(parts, "/")
	}
	return strings.TrimSpace(t.Text)
}

func (ui *tviewUI) encounterLabelAt(idx int) string {
	if idx < 0 || idx >= len(ui.encounter) {
		return ""
	}
	e := ui.encounter[idx]
	if e.Seq > 0 {
		return fmt.Sprintf("%s #%d", e.Monster.Name, e.Seq)
	}
	name := e.Monster.Name
	seen := 0
	for i := 0; i <= idx; i++ {
		if ui.encounter[i].Monster.Name == name {
			seen++
		}
	}
	return fmt.Sprintf("%s #%d", name, seen)
}

func (ui *tviewUI) adjustSelectedToken(delta int) {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	p := &ui.pngs[ui.selected]
	newToken := min(max(p.Token+delta, minToken), maxToken)
	if newToken == p.Token {
		return
	}
	ui.beginUndoableChange()
	p.Token = newToken
	ui.persistPNGs()
	ui.message = fmt.Sprintf("Token di %s: %d", p.Name, p.Token)
	ui.refreshPNGs()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) openCreatePNGModal() {
	returnFocus := ui.app.GetFocus()
	defaultName := uniqueRandomPNGName(ui.pngs)
	selectedPF := 0
	selectedStress := 0
	selectedArmor := 3
	selectedHope := 2

	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Crea PNG").SetTitleAlign(tview.AlignLeft)
	advanceToGenerate := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Crea"))
	}
	form.AddInputField("Nome PNG", defaultName, 24, nil, nil)
	form.AddInputField("PF", "0", 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedPF = v
		}
	})
	form.AddInputField("Stress", "0", 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedStress = v
		}
	})
	form.AddInputField("Armatura", "3", 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedArmor = v
		}
	})
	form.AddInputField("Speranza", "2", 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedHope = v
		}
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyEnter {
			return event
		}
		itemIdx, buttonIdx := form.GetFocusedItemIndex()
		switch {
		case itemIdx == 0:
			form.SetFocus(1)
			return nil
		case itemIdx == 1:
			form.SetFocus(2)
			return nil
		case itemIdx == 2:
			form.SetFocus(3)
			return nil
		case itemIdx == 3:
			form.SetFocus(4)
			return nil
		case itemIdx == 4:
			advanceToGenerate()
			return nil
		case buttonIdx >= 0:
			return event
		default:
			return event
		}
	})

	form.AddButton("Crea", func() {
		name := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		if name == "" {
			name = uniqueRandomPNGName(ui.pngs)
		}
		for _, p := range ui.pngs {
			if strings.EqualFold(p.Name, name) {
				ui.message = "Nome già esistente."
				ui.refreshStatus()
				return
			}
		}
		if v, err := strconv.Atoi(strings.TrimSpace(form.GetFormItem(1).(*tview.InputField).GetText())); err == nil && v >= 0 {
			selectedPF = v
		} else {
			selectedPF = 0
		}
		if v, err := strconv.Atoi(strings.TrimSpace(form.GetFormItem(2).(*tview.InputField).GetText())); err == nil && v >= 0 {
			selectedStress = v
		} else {
			selectedStress = 0
		}
		if v, err := strconv.Atoi(strings.TrimSpace(form.GetFormItem(3).(*tview.InputField).GetText())); err == nil && v >= 0 {
			selectedArmor = v
		} else {
			selectedArmor = 3
		}
		if v, err := strconv.Atoi(strings.TrimSpace(form.GetFormItem(4).(*tview.InputField).GetText())); err == nil && v >= 0 {
			selectedHope = v
		} else {
			selectedHope = 2
		}

		ui.beginUndoableChange()
		ui.pngs = append(ui.pngs, PNG{Name: name, Token: defaultToken, PF: selectedPF, Stress: selectedStress, ArmorScore: selectedArmor, Hope: selectedHope})
		ui.selected = len(ui.pngs) - 1
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("Creato PNG %s.", name)
		ui.refreshAll()
	})
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetButtonsAlign(tview.AlignLeft)

	modal := ui.fullscreenModal(form)

	ui.modalVisible = true
	ui.modalName = "create_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func (ui *tviewUI) openEditPNGModal() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}

	cur := ui.pngs[ui.selected]
	selectedName := cur.Name
	selectedToken := cur.Token
	selectedPF := cur.PF
	selectedStress := cur.Stress
	selectedArmor := cur.ArmorScore
	selectedHope := cur.Hope
	returnFocus := ui.app.GetFocus()

	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Modifica PNG").SetTitleAlign(tview.AlignLeft)
	advanceToSave := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Salva"))
	}
	form.AddInputField("Nome", cur.Name, 28, nil, func(text string) {
		selectedName = strings.TrimSpace(text)
	})
	form.AddInputField("Token", strconv.Itoa(cur.Token), 3, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil {
			selectedToken = v
		}
	})
	form.AddInputField("PF", strconv.Itoa(cur.PF), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedPF = v
		}
	})
	form.AddInputField("Stress", strconv.Itoa(cur.Stress), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedStress = v
		}
	})
	form.AddInputField("Armatura", strconv.Itoa(cur.ArmorScore), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedArmor = v
		}
	})
	form.AddInputField("Speranza", strconv.Itoa(cur.Hope), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedHope = v
		}
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyEnter {
			return event
		}
		itemIdx, buttonIdx := form.GetFocusedItemIndex()
		switch {
		case itemIdx == 0:
			form.SetFocus(1)
			return nil
		case itemIdx == 1:
			form.SetFocus(2)
			return nil
		case itemIdx == 2:
			form.SetFocus(3)
			return nil
		case itemIdx == 3:
			form.SetFocus(4)
			return nil
		case itemIdx == 4:
			form.SetFocus(5)
			return nil
		case itemIdx == 5:
			advanceToSave()
			return nil
		case buttonIdx >= 0:
			return event
		default:
			return event
		}
	})
	form.AddButton("Salva", func() {
		if strings.TrimSpace(selectedName) == "" {
			ui.message = "Nome PNG non valido."
			ui.refreshStatus()
			return
		}
		for i, p := range ui.pngs {
			if i == ui.selected {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(selectedName)) {
				ui.message = "Nome già esistente."
				ui.refreshStatus()
				return
			}
		}
		before := ui.pngs[ui.selected]
		afterName := strings.TrimSpace(selectedName)
		afterToken := selectedToken
		afterPF := selectedPF
		afterStress := selectedStress
		afterArmor := selectedArmor
		afterHope := selectedHope
		if afterToken < minToken {
			afterToken = minToken
		}
		if afterToken > maxToken {
			afterToken = maxToken
		}
		if afterPF < 0 {
			afterPF = 0
		}
		if afterStress < 0 {
			afterStress = 0
		}
		if afterArmor < 0 {
			afterArmor = 0
		}
		if afterHope < 0 {
			afterHope = 0
		}
		if before.Name == afterName && before.Token == afterToken && before.PF == afterPF && before.Stress == afterStress && before.ArmorScore == afterArmor && before.Hope == afterHope {
			ui.closeModal()
			ui.focusPanel(focusPNG)
			ui.message = "Nessuna modifica al PNG."
			ui.refreshStatus()
			return
		}
		ui.beginUndoableChange()
		selectedToken = max(clampStat(selectedToken, maxToken), minToken)
		if selectedPF < 0 {
			selectedPF = 0
		}
		if selectedStress < 0 {
			selectedStress = 0
		}
		if selectedArmor < 0 {
			selectedArmor = 0
		}
		if selectedHope < 0 {
			selectedHope = 0
		}
		ui.pngs[ui.selected].Name = strings.TrimSpace(selectedName)
		ui.pngs[ui.selected].Token = selectedToken
		ui.pngs[ui.selected].PF = selectedPF
		ui.pngs[ui.selected].Stress = selectedStress
		ui.pngs[ui.selected].ArmorScore = selectedArmor
		ui.pngs[ui.selected].Hope = selectedHope
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("PNG aggiornato: %s.", ui.pngs[ui.selected].Name)
		ui.refreshAll()
	})
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetButtonsAlign(tview.AlignLeft)

	modal := ui.fullscreenModal(form)

	ui.modalVisible = true
	ui.modalName = "edit_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func (ui *tviewUI) openClassPNGInput() {
	idx := ui.currentClassIndex()
	if idx < 0 || idx >= len(ui.classes) {
		ui.message = "Nessuna classe selezionata."
		ui.refreshStatus()
		return
	}
	c := ui.classes[idx]
	returnFocus := ui.app.GetFocus()

	levels := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		levels = append(levels, strconv.Itoa(i))
	}
	selectedLevel := 1
	selectedPF := max(c.HP, 0)
	selectedStress := 0
	selectedArmor := 3
	selectedHope := 2
	ready := false

	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Crea PNG da Classe").SetTitleAlign(tview.AlignLeft)
	advanceToGenerate := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
	}
	form.AddDropDown("Livello", levels, 0, func(option string, _ int) {
		if option == "" {
			return
		}
		if v, err := strconv.Atoi(option); err == nil && v > 0 {
			selectedLevel = v
		}
		if ready {
			form.SetFocus(1)
		}
	})
	form.AddInputField("PF", strconv.Itoa(selectedPF), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedPF = v
		}
	})
	form.AddInputField("Stress", strconv.Itoa(selectedStress), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedStress = v
		}
	})
	form.AddInputField("Armatura", strconv.Itoa(selectedArmor), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedArmor = v
		}
	})
	form.AddInputField("Speranza", strconv.Itoa(selectedHope), 4, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v >= 0 {
			selectedHope = v
		}
	})
	if item := form.GetFormItem(0); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
		}
	}
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyEnter {
			return event
		}
		itemIdx, buttonIdx := form.GetFocusedItemIndex()
		switch {
		case itemIdx == 0:
			form.SetFocus(1)
			return nil
		case itemIdx == 1:
			form.SetFocus(2)
			return nil
		case itemIdx == 2:
			form.SetFocus(3)
			return nil
		case itemIdx == 3:
			form.SetFocus(4)
			return nil
		case itemIdx == 4:
			advanceToGenerate()
			return nil
		case buttonIdx >= 0:
			return event
		default:
			return event
		}
	})

	form.AddButton("Genera", func() {
		baseName := uniqueRandomPNGName(ui.pngs)
		preset := classPresetFor(c.Name)
		inv := buildSuggestedInventory(preset)
		look := buildSuggestedLook(preset)
		png := PNG{
			Name:        fmt.Sprintf("%s (%s | %s L%d)", baseName, c.Subclass, c.Name, selectedLevel),
			Token:       defaultToken,
			PF:          selectedPF,
			Stress:      selectedStress,
			ArmorScore:  selectedArmor,
			Hope:        selectedHope,
			Class:       strings.TrimSpace(c.Name),
			Subclass:    strings.TrimSpace(c.Subclass),
			Level:       selectedLevel,
			Rank:        rankFromLevel(selectedLevel),
			CompBonus:   progressionBonusAtLevel(selectedLevel),
			ExpBonus:    progressionBonusAtLevel(selectedLevel),
			Description: strings.TrimSpace(c.Description),
			Traits:      strings.TrimSpace(preset.Traits),
			Primary:     strings.TrimSpace(preset.Primary),
			Secondary:   strings.TrimSpace(preset.Secondary),
			Armor:       strings.TrimSpace(preset.Armor),
			Look:        look,
			Inventory:   inv,
		}
		ui.beginUndoableChange()
		ui.pngs = append(ui.pngs, png)
		ui.selected = len(ui.pngs) - 1
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("Creato PNG da classe: %s | %s L%d", c.Subclass, c.Name, selectedLevel)
		ui.refreshAll()
	})
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetButtonsAlign(tview.AlignLeft)
	ready = true

	info := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	info.SetText(fmt.Sprintf("[yellow]%s | %s[-]\nScegli il livello e genera un PNG.", c.Subclass, c.Name))

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(info, 8, 0, false).
		AddItem(form, 0, 1, true)

	modal := ui.fullscreenModal(container)

	ui.modalVisible = true
	ui.modalName = "class_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func chooseOne(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.TrimSpace(items[rand.IntN(len(items))])
}

func buildSuggestedInventory(p classPreset) string {
	base := []string{
		"torcia",
		"16 metri di corda",
		"provviste di base",
		"una manciata d'oro",
	}
	potion := chooseOne([]string{"Pozione di Guarigione Minore", "Pozione di Recupero Minore"})
	extra := chooseOne([]string{strings.TrimSpace(p.ExtraA), strings.TrimSpace(p.ExtraB)})
	if potion != "" {
		base = append(base, potion)
	}
	if extra != "" {
		base = append(base, extra)
	}
	return strings.Join(base, ", ")
}

func buildSuggestedLook(p classPreset) string {
	eyes := []string{
		"vivaci", "del colore della terra", "dell'oceano", "di fuoco", "verde edera", "lilla", "la notte", "schiuma del mare", "gelidi",
	}
	body := []string{
		"spalle larghe", "scolpita", "formosa", "allampanata", "tondeggiante", "piccola statura", "robusta", "alta", "slanciata", "minuta", "allenata",
	}
	skin := []string{
		"cenere", "nivea", "sabbia", "ossidiana", "rosea", "trifoglio", "zaffiro", "glicine",
	}

	abiti := chooseOne(p.Abiti)
	atteggiamento := chooseOne(p.Attitude)
	occhi := chooseOne(eyes)
	corporatura := chooseOne(body)
	carnagione := chooseOne(skin)

	parts := []string{}
	if abiti != "" {
		parts = append(parts, "Abiti: "+abiti)
	}
	if occhi != "" {
		parts = append(parts, "Occhi: "+occhi)
	}
	if corporatura != "" {
		parts = append(parts, "Corporatura: "+corporatura)
	}
	if carnagione != "" {
		parts = append(parts, "Carnagione: "+carnagione)
	}
	if atteggiamento != "" {
		parts = append(parts, "Atteggiamento: "+atteggiamento)
	}
	return strings.Join(parts, " | ")
}

func classPresetFor(className string) classPreset {
	switch strings.ToLower(strings.TrimSpace(className)) {
	case "bardo":
		return classPreset{
			Traits:    "0 Agilita, -1 Forza, +1 Astuzia, 0 Istinto, +2 Presenza, +1 Conoscenza",
			Primary:   "Stocco - Presenza, Mischia - d8 fis - A una mano",
			Secondary: "Stiletto - Astuzia, Mischia - d8 fis - A una mano",
			Armor:     "Gambesone - Soglie 5/11 - Punteggio Base 3 (Flessibile: +1 all'Evasione)",
			ExtraA:    "un racconto romantico",
			ExtraB:    "una lettera mai aperta",
			Abiti:     []string{"stravaganti", "lussuosi", "vistosi", "di una taglia in piu", "cenciosi", "eleganti", "grezzi"},
			Attitude:  []string{"taverniere", "prestigiatore", "circense", "rockstar", "smargiasso"},
		}
	case "consacrato":
		return classPreset{
			Traits:    "0 Agilita, +2 Forza, 0 Astuzia, +1 Istinto, +1 Presenza, -1 Conoscenza",
			Primary:   "Ascia Consacrata - Forza, Mischia - d8+1 mag - A una mano",
			Secondary: "Scudo Rotella - Forza, Mischia - d4 fis - A una mano",
			Armor:     "Cotta di Maglia - Soglie 7/15 - Punteggio Base 4 (Pesante: -1 all'Evasione)",
			ExtraA:    "una raccolta di offerte",
			ExtraB:    "il simbolo sacro della vostra divinita",
			Abiti:     []string{"splendenti", "ondeggianti", "ornati", "aderenti", "modesti", "strani", "naturali"},
			Attitude:  []string{"angelico", "di un medico", "di un predicatore", "monastico", "sacerdotale"},
		}
	case "druido":
		return classPreset{
			Traits:    "+1 Agilita, 0 Forza, +1 Astuzia, +2 Istinto, -1 Presenza, 0 Conoscenza",
			Primary:   "Verga - Istinto, Ravvicinata - d8+1 mag - A una mano",
			Secondary: "Scudo Rotella - Forza, Mischia - d4 fis - A una mano",
			Armor:     "Corazza di Cuoio - Soglie 6/13 - Punteggio Base 3",
			ExtraA:    "una borsa piena di pietruzze e ossicini",
			ExtraB:    "uno strano pendente scovato nella sporcizia",
			Abiti:     []string{"mimetici", "di fibre vegetali", "confortevoli", "naturali", "di pezze cucite insieme", "regali", "stracci"},
			Attitude:  []string{"esplosivo", "astuto come una volpe", "da guida nella foresta", "come un figlio dei fiori", "stregonesco"},
		}
	case "fuorilegge":
		return classPreset{
			Traits:    "+1 Agilita, -1 Forza, +2 Astuzia, 0 Istinto, +1 Presenza, 0 Conoscenza",
			Primary:   "Pugnale - Astuzia, Mischia - d8+1 fis - A una mano",
			Secondary: "Stiletto - Astuzia, Mischia - d8 fis - A una mano",
			Armor:     "Gambesone - Soglie 5/11 - Punteggio Base 3 (Flessibile: +1 all'Evasione)",
			ExtraA:    "attrezzatura da falsario",
			ExtraB:    "un rampino",
			Abiti:     []string{"puliti", "scuri", "anonimi", "in pelle", "inquietanti", "mimetici", "tattici", "aderenti"},
			Attitude:  []string{"da bandito", "da truffatore", "da giocatore d'azzardo", "da capobanda", "da pirata"},
		}
	case "guardiano":
		return classPreset{
			Traits:    "+1 Agilita, +2 Forza, -1 Astuzia, 0 Istinto, +1 Presenza, 0 Conoscenza",
			Primary:   "Ascia da Battaglia - Forza, Mischia - d10+3 fis - A due mani",
			Secondary: "",
			Armor:     "Cotta di Maglia - Soglie 7/15 - Punteggio Base 4 (Pesante: -1 all'Evasione)",
			ExtraA:    "un ricordo del vostro mentore",
			ExtraB:    "una chiave misteriosa",
			Abiti:     []string{"casual", "ornati", "confortevoli", "imbottiti", "regali", "tattici", "consunti"},
			Attitude:  []string{"di un capitano", "di un guardiano", "di un elefante", "di un generale", "di un lottatore"},
		}
	case "guerriero":
		return classPreset{
			Traits:    "+2 Agilita, +1 Forza, 0 Astuzia, +1 Istinto, -1 Presenza, 0 Conoscenza",
			Primary:   "Spada Lunga - Agilita, Mischia - d8+3 fis - A due mani",
			Secondary: "",
			Armor:     "Cotta di Maglia - Soglie 7/15 - Punteggio Base 4 (Pesante: -1 all'Evasione)",
			ExtraA:    "il ritratto di chi amate",
			ExtraB:    "una cote per affilare",
			Abiti:     []string{"provocanti", "rattoppati", "rinforzati", "regali", "eleganti", "di ricambio", "consunti"},
			Attitude:  []string{"da toro", "da soldato fedele", "da gladiatore", "eroico", "da mercenario"},
		}
	case "mago":
		return classPreset{
			Traits:    "-1 Agilita, 0 Forza, 0 Astuzia, +1 Istinto, +1 Presenza, +2 Conoscenza",
			Primary:   "Bordone - Conoscenza, Remota - d6 mag - A due mani",
			Secondary: "",
			Armor:     "Corazza di Cuoio - Soglie 6/13 - Punteggio Base 3",
			ExtraA:    "un libro che state cercando di tradurre",
			ExtraB:    "un piccolo e innocuo cucciolo elementale",
			Abiti:     []string{"belli", "puliti", "ordinari", "fluenti", "a strati", "rattoppati", "aderenti"},
			Attitude:  []string{"eccentrico", "da bibliotecario", "di una miccia accesa", "da filosofo", "da professore"},
		}
	case "ranger":
		return classPreset{
			Traits:    "+2 Agilita, 0 Forza, +1 Astuzia, +1 Istinto, -1 Presenza, 0 Conoscenza",
			Primary:   "Arco Corto - Agilita, Lontana - d6+3 fis - A due mani",
			Secondary: "",
			Armor:     "Corazza di Cuoio - Soglie 6/13 - Punteggio Base 3",
			ExtraA:    "un trofeo della vostra prima preda",
			ExtraB:    "una bussola apparentemente rotta",
			Abiti:     []string{"fluenti", "dai colori spenti", "naturali", "macchiati", "tattici", "aderenti", "di lana o di lino"},
			Attitude:  []string{"di un bambino", "spettrale", "di un survivalista", "di un insegnante", "di un cane da guardia"},
		}
	case "stregone":
		return classPreset{
			Traits:    "0 Agilita, -1 Forza, +1 Astuzia, +2 Istinto, +1 Presenza, 0 Conoscenza",
			Primary:   "Bastone Doppio - Istinto, Lontana - 1d6+3 mag - A due mani",
			Secondary: "",
			Armor:     "Gambesone - Soglie 5/11 - Punteggio Base 3 (Flessibile: +1 all'Evasione)",
			ExtraA:    "un globo sussurrante",
			ExtraB:    "un cimelio di famiglia",
			Abiti:     []string{"a strati", "aderenti", "decorati", "poco appariscenti", "sempre in movimento", "sgargianti"},
			Attitude:  []string{"burlone", "da celebrita", "da condottiero", "da politico", "da lupo travestito da agnello"},
		}
	default:
		return classPreset{}
	}
}

func rankFromLevel(level int) int {
	switch {
	case level <= 1:
		return 1
	case level <= 4:
		return 2
	case level <= 7:
		return 3
	default:
		return 4
	}
}

func progressionBonusAtLevel(level int) int {
	bonus := 0
	if level >= 2 {
		bonus++
	}
	if level >= 5 {
		bonus++
	}
	if level >= 8 {
		bonus++
	}
	return bonus
}

func (ui *tviewUI) findClassDefinition(className, subclass string) *ClassItem {
	for i := range ui.classes {
		c := &ui.classes[i]
		if strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(className)) &&
			strings.EqualFold(strings.TrimSpace(c.Subclass), strings.TrimSpace(subclass)) {
			return c
		}
	}
	return nil
}

func (ui *tviewUI) openResetTokensConfirm() {
	if len(ui.pngs) == 0 {
		ui.message = "Nessun PNG disponibile."
		ui.refreshStatus()
		return
	}
	ui.openConfirmModal("Conferma", "Resettare tutti i token PNG?", func() {
		ui.resetTokens()
	})
}

func (ui *tviewUI) openConfirmModal(title, message string, onConfirm func()) {
	returnFocus := ui.app.GetFocus()
	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	text.SetBorder(true).SetTitle(title)
	text.SetText(message + "\n\n[yellow]Invio/y[-] conferma  [yellow]Esc/n[-] annulla")
	text.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEnter || (ev.Key() == tcell.KeyRune && (ev.Rune() == 'y' || ev.Rune() == 'Y')) {
			ui.closeModal()
			onConfirm()
			return nil
		}
		if ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && (ev.Rune() == 'n' || ev.Rune() == 'N')) {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			ui.refreshStatus()
			return nil
		}
		return ev
	})

	modal := ui.fullscreenModal(text)

	ui.modalVisible = true
	ui.modalName = "confirm"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(text)
}

func (ui *tviewUI) openRawSearch(focus tview.Primitive) {
	input := tview.NewInputField().SetLabel(" Cerca ").SetFieldWidth(28)
	input.SetBorder(true).SetTitle("Ricerca")
	if focus == ui.dice {
		if len(ui.diceLog) > 0 {
			cur := ui.dice.GetCurrentItem()
			if cur >= 0 && cur < len(ui.diceLog) {
				input.SetText(ui.diceLog[cur].Expression)
			}
		}
	}
	if focus == ui.search || focus == ui.monList || focus == ui.roleDrop || focus == ui.rankDrop {
		input.SetText(ui.search.GetText())
	}
	if focus == ui.envSearch || focus == ui.envList || focus == ui.envTypeDrop || focus == ui.envRankDrop {
		input.SetText(ui.envSearch.GetText())
	}
	if focus == ui.eqSearch || focus == ui.eqList || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop {
		input.SetText(ui.eqSearch.GetText())
	}
	if focus == ui.cardSearch || focus == ui.cardList || focus == ui.cardClassDrop || focus == ui.cardTypeDrop {
		input.SetText(ui.cardSearch.GetText())
	}
	if focus == ui.classSearch || focus == ui.classList || focus == ui.classNameDrop || focus == ui.classSubDrop {
		input.SetText(ui.classSearch.GetText())
	}
	if focus == ui.notesSearch || focus == ui.notesList {
		input.SetText(ui.notesSearch.GetText())
	}
	if focus == ui.detail {
		input.SetText(ui.detailQuery)
	}
	if focus == ui.detailTreasure {
		input.SetText(ui.detailQuery)
	}
	if focus == ui.notesList {
		input.SetText(ui.detailQuery)
	}

	returnFocus := focus
	modal := ui.fullscreenModal(input)

	ui.modalVisible = true
	ui.modalName = "raw_search"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		query := strings.TrimSpace(input.GetText())
		switch returnFocus {
		case ui.dice:
			ui.jumpToDiceResult(query)
			ui.focusPanel(focusDice)
		case ui.search, ui.monList, ui.roleDrop, ui.rankDrop:
			ui.search.SetText(query)
			ui.refreshMonsters()
			ui.focusPanel(focusMonList)
			ui.message = "Filtro mostri aggiornato."
		case ui.envSearch, ui.envList, ui.envTypeDrop, ui.envRankDrop:
			ui.envSearch.SetText(query)
			ui.refreshEnvironments()
			ui.focusPanel(focusEnvList)
			ui.message = "Filtro ambienti aggiornato."
		case ui.eqSearch, ui.eqList, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop:
			ui.eqSearch.SetText(query)
			ui.refreshEquipment()
			ui.focusPanel(focusEqList)
			ui.message = "Filtro equipaggiamento aggiornato."
		case ui.cardSearch, ui.cardList, ui.cardClassDrop, ui.cardTypeDrop:
			ui.cardSearch.SetText(query)
			ui.refreshCards()
			ui.focusPanel(focusCardList)
			ui.message = "Filtro carte aggiornato."
		case ui.classSearch, ui.classList, ui.classNameDrop, ui.classSubDrop:
			ui.classSearch.SetText(query)
			ui.refreshClasses()
			ui.focusPanel(focusClassList)
			ui.message = "Filtro classi aggiornato."
		case ui.notesSearch, ui.notesList:
			ui.notesSearch.SetText(query)
			ui.refreshNotes()
			ui.focusPanel(focusNotesList)
			ui.message = "Filtro note aggiornato."
		case ui.encList:
			ui.jumpToEncounter(query)
		case ui.detail:
			ui.detailQuery = query
			ui.renderDetail()
			if query == "" {
				ui.message = "Highlight dettagli rimosso."
			} else {
				ui.message = fmt.Sprintf("Highlight dettagli: %s", query)
			}
		case ui.detailTreasure:
			ui.detailQuery = query
			ui.renderTreasure()
			if query == "" {
				ui.message = "Highlight treasure rimosso."
			} else {
				ui.message = fmt.Sprintf("Highlight treasure: %s", query)
			}
		case ui.notesList:
			ui.detailQuery = query
			ui.refreshDetail()
			if query == "" {
				ui.message = "Highlight note rimosso."
			} else {
				ui.message = fmt.Sprintf("Highlight note: %s", query)
			}
		default:
			ui.search.SetText(query)
			ui.refreshMonsters()
			ui.focusPanel(focusMonList)
		}
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
}

func (ui *tviewUI) jumpToEncounter(query string) {
	if strings.TrimSpace(query) == "" {
		ui.message = "Ricerca encounter vuota."
		return
	}
	q := strings.ToLower(query)
	for i, e := range ui.encounter {
		if strings.Contains(strings.ToLower(e.Monster.Name), q) {
			ui.encList.SetCurrentItem(i)
			ui.message = fmt.Sprintf("Trovato in encounter: %s", e.Monster.Name)
			ui.refreshDetail()
			return
		}
	}
	ui.message = fmt.Sprintf("Nessun match encounter per '%s'.", query)
}

func (ui *tviewUI) jumpToDiceResult(query string) {
	if strings.TrimSpace(query) == "" {
		ui.message = "Ricerca dadi vuota."
		return
	}
	q := strings.ToLower(query)
	for i, e := range ui.diceLog {
		line := strings.ToLower(e.Expression + " " + e.Output)
		if strings.Contains(line, q) {
			ui.dice.SetCurrentItem(i)
			ui.message = fmt.Sprintf("Trovato in dadi: #%d", i+1)
			ui.refreshDetail()
			return
		}
	}
	ui.message = fmt.Sprintf("Nessun match dadi per '%s'.", query)
}

func (ui *tviewUI) jumpToDiceRow(oneBased int) {
	total := len(ui.diceLog)
	if total == 0 {
		ui.message = "Nessun tiro in lista."
		ui.refreshStatus()
		return
	}
	if oneBased < 1 {
		ui.message = "Riga dadi non valida."
		ui.refreshStatus()
		return
	}
	if oneBased > total {
		oneBased = total
	}
	ui.dice.SetCurrentItem(oneBased - 1)
	ui.message = fmt.Sprintf("Riga dadi: %d/%d", oneBased, total)
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) scrollDetailByPage(direction int) {
	target := ui.detail
	if ui.app.GetFocus() == ui.detailTreasure {
		target = ui.detailTreasure
	}
	row, col := target.GetScrollOffset()
	_, _, _, h := target.GetInnerRect()
	if h <= 0 {
		h = 24
	}
	step := max(h/2, 1)
	row += direction * step
	if row < 0 {
		row = 0
	}
	target.ScrollTo(row, col)
}

func (ui *tviewUI) deleteSelectedPNG() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	name := ui.pngs[ui.selected].Name
	ui.beginUndoableChange()
	ui.pngs = append(ui.pngs[:ui.selected], ui.pngs[ui.selected+1:]...)
	if len(ui.pngs) == 0 {
		ui.selected = -1
	} else if ui.selected >= len(ui.pngs) {
		ui.selected = len(ui.pngs) - 1
	}
	ui.persistPNGs()
	ui.message = fmt.Sprintf("PNG %s eliminato.", name)
	ui.refreshAll()
}

func (ui *tviewUI) resetTokens() {
	changed := false
	for i := range ui.pngs {
		if ui.pngs[i].Token != defaultToken {
			changed = true
			break
		}
	}
	if !changed {
		return
	}
	ui.beginUndoableChange()
	for i := range ui.pngs {
		ui.pngs[i].Token = defaultToken
	}
	ui.persistPNGs()
	ui.message = fmt.Sprintf("Token PNG resettati a %d.", defaultToken)
	ui.refreshAll()
}

func (ui *tviewUI) addSelectedMonsterToEncounter() {
	idx := ui.currentMonsterIndex()
	if idx < 0 {
		ui.message = "Nessun mostro selezionato."
		ui.refreshStatus()
		return
	}
	mon := ui.monsters[idx]
	ui.beginUndoableChange()
	ui.encounter = append(ui.encounter, EncounterEntry{
		Monster:    mon,
		Seq:        nextEncounterSeq(ui.encounter, mon.Name),
		BasePF:     mon.PF,
		Stress:     mon.Stress,
		BaseStress: mon.Stress,
	})
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Aggiunto %s a encounter.", mon.Name)
	ui.refreshEncounter()
	ui.refreshStatus()
}

func battleCostForRole(role string) int {
	r := strings.ToLower(strings.TrimSpace(role))
	switch {
	case strings.Contains(r, "seguace"):
		return 1
	case strings.Contains(r, "controparte"), strings.Contains(r, "supporto"):
		return 1
	case strings.Contains(r, "orda"), strings.Contains(r, "tiratore"), strings.Contains(r, "sicario"), strings.Contains(r, "base"):
		return 2
	case strings.Contains(r, "condottiero"):
		return 3
	case strings.Contains(r, "bruto"):
		return 4
	case strings.Contains(r, "solitario"):
		return 5
	default:
		return 0
	}
}

func isFollowerRole(role string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(role)), "seguace")
}

func battleBudgetModifierByDifficulty(label string) int {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "facile":
		return -2
	case "difficile":
		return 2
	default:
		return 0
	}
}

type generatedEncounterSummary struct {
	Rank          int
	PGCount       int
	BaseBudget    int
	BudgetMod     int
	FinalBudget   int
	Spent         int
	Remaining     int
	AddedEntries  int
	AddedGroups   int
	ByMonsterName map[string]int
}

func (ui *tviewUI) openRandomEncounterFromMonstersInput() {
	rankOptions := make([]string, 0, len(ui.rankOpts))
	defaultRankIdx := 0
	currentMonsterIdx := ui.currentMonsterIndex()
	currentRank := 0
	if currentMonsterIdx >= 0 && currentMonsterIdx < len(ui.monsters) {
		currentRank = ui.monsters[currentMonsterIdx].Rank
	}
	for _, opt := range ui.rankOpts {
		if strings.EqualFold(strings.TrimSpace(opt), "Tutti") {
			continue
		}
		rankOptions = append(rankOptions, opt)
		if currentRank > 0 && opt == strconv.Itoa(currentRank) {
			defaultRankIdx = len(rankOptions) - 1
		}
	}
	if len(rankOptions) == 0 {
		ui.message = "Nessun rango disponibile nei Mostri."
		ui.refreshStatus()
		return
	}

	defaultPG := max(len(ui.pngs), 1)
	selectedRank, _ := strconv.Atoi(rankOptions[defaultRankIdx])
	if selectedRank <= 0 {
		selectedRank = 1
	}
	selectedPG := defaultPG
	difficultyOptions := []string{"Normale", "Facile", "Difficile"}
	selectedDifficulty := difficultyOptions[0]
	ready := false
	returnFocus := ui.app.GetFocus()

	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Genera Encounter Random da Mostri").SetTitleAlign(tview.AlignLeft)
	advanceToGenerate := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
	}
	form.AddDropDown("Rango gruppo", rankOptions, defaultRankIdx, func(option string, _ int) {
		if option == "" {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(option)); err == nil && v > 0 {
			selectedRank = v
		}
		if ready {
			form.SetFocus(1)
		}
	})
	form.AddInputField("PG in combatt.", strconv.Itoa(defaultPG), 5, func(textToCheck string, lastChar rune) bool {
		if textToCheck == "" {
			return true
		}
		_, err := strconv.Atoi(textToCheck)
		return err == nil
	}, func(text string) {
		if v, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && v > 0 {
			selectedPG = v
		}
	})
	form.AddDropDown("Difficoltà", difficultyOptions, 0, func(option string, _ int) {
		if option == "" {
			return
		}
		selectedDifficulty = option
		if ready {
			advanceToGenerate()
		}
	})

	if item := form.GetFormItem(0); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
			dd.SetFinishedFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter, tcell.KeyTab:
					form.SetFocus(1)
				case tcell.KeyBacktab:
					form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Annulla"))
				}
			})
		}
	}
	if item := form.GetFormItem(1); item != nil {
		if input, ok := item.(*tview.InputField); ok {
			input.SetFieldBackgroundColor(tcell.ColorBlack)
			input.SetFieldTextColor(tcell.ColorWhite)
			input.SetDoneFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter, tcell.KeyTab:
					form.SetFocus(2)
				case tcell.KeyBacktab:
					form.SetFocus(0)
				}
			})
		}
	}
	if item := form.GetFormItem(2); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			dd.SetFieldBackgroundColor(tcell.ColorBlack)
			dd.SetFieldTextColor(tcell.ColorWhite)
			dd.SetListStyles(
				tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
				tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
			)
			dd.SetFinishedFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter, tcell.KeyTab:
					advanceToGenerate()
				case tcell.KeyBacktab:
					form.SetFocus(1)
				}
			})
		}
	}

	form.AddButton("Genera", func() {
		v := strings.TrimSpace(form.GetFormItem(1).(*tview.InputField).GetText())
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			ui.message = "Numero PG non valido."
			ui.refreshStatus()
			return
		}
		selectedPG = n
		mod := battleBudgetModifierByDifficulty(selectedDifficulty)
		summary := ui.generateRandomEncounterFromMonsters(selectedRank, selectedPG, mod)
		if summary.AddedEntries == 0 {
			ui.message = fmt.Sprintf("Nessun mostro generato (R%d, %d PG).", selectedRank, selectedPG)
			ui.refreshStatus()
			return
		}
		ui.closeModal()
		ui.focusPanel(focusEncounter)
		ui.message = fmt.Sprintf("Encounter random R%d: +%d nemici (%d PB spesi, %d residui).", selectedRank, summary.AddedEntries, summary.Spent, summary.Remaining)
		ui.refreshEncounter()
		ui.detailRaw = buildGeneratedEncounterDetails(summary)
		ui.renderDetail()
		ui.detail.ScrollTo(0, 0)
		ui.refreshStatus()
	})
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetButtonsAlign(tview.AlignLeft)

	info := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	info.SetText("Punti Battaglia: (3 x PG in combattimento) + 2.\nDifficoltà: Facile -2, Normale 0, Difficile +2.\nCosti ruolo: Seguace/Controparte/Supporto=1, Base/Tiratore/Sicario/Orda=2, Condottiero=3, Bruto=4, Solitario=5.")

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(info, 4, 0, false).
		AddItem(form, 0, 1, true)

	modal := ui.fullscreenModal(container)

	ui.modalVisible = true
	ui.modalName = "monster_random_encounter"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
	ready = true
}

func (ui *tviewUI) generateRandomEncounterFromMonsters(rank int, pgCount int, budgetMod int) generatedEncounterSummary {
	summary := generatedEncounterSummary{
		Rank:          rank,
		PGCount:       pgCount,
		BaseBudget:    3*pgCount + 2,
		BudgetMod:     budgetMod,
		ByMonsterName: map[string]int{},
	}
	if pgCount < 1 {
		return summary
	}
	finalBudget := max(summary.BaseBudget+budgetMod, 1)
	summary.FinalBudget = finalBudget

	type candidate struct {
		mon  Monster
		cost int
	}
	candidates := make([]candidate, 0, len(ui.monsters))
	for _, m := range ui.monsters {
		if m.Rank != rank {
			continue
		}
		cost := battleCostForRole(m.Role)
		if cost <= 0 {
			continue
		}
		candidates = append(candidates, candidate{mon: m, cost: cost})
	}
	if len(candidates) == 0 {
		summary.Remaining = finalBudget
		return summary
	}

	remaining := finalBudget
	added := 0
	spent := 0
	historyCaptured := false
	for remaining > 0 {
		affordable := make([]candidate, 0, len(candidates))
		for _, c := range candidates {
			if c.cost <= remaining {
				affordable = append(affordable, c)
			}
		}
		if len(affordable) == 0 {
			break
		}
		pick := affordable[rand.IntN(len(affordable))]
		qty := 1
		if isFollowerRole(pick.mon.Role) {
			qty = pgCount
		}
		if !historyCaptured {
			ui.beginUndoableChange()
			historyCaptured = true
		}
		for i := 0; i < qty; i++ {
			ui.encounter = append(ui.encounter, EncounterEntry{
				Monster:    pick.mon,
				Seq:        nextEncounterSeq(ui.encounter, pick.mon.Name),
				BasePF:     pick.mon.PF,
				Stress:     pick.mon.Stress,
				BaseStress: pick.mon.Stress,
			})
			added++
		}
		summary.AddedGroups++
		summary.ByMonsterName[pick.mon.Name] += qty
		remaining -= pick.cost
		spent += pick.cost
	}
	if added > 0 {
		ui.persistEncounter()
	}
	summary.AddedEntries = added
	summary.Spent = spent
	summary.Remaining = remaining
	return summary
}

func buildGeneratedEncounterDetails(s generatedEncounterSummary) string {
	var b strings.Builder
	b.WriteString("Encounter random generato\n")
	fmt.Fprintf(&b, "Rango gruppo: %d | PG: %d\n", s.Rank, s.PGCount)
	fmt.Fprintf(&b, "Punti Battaglia: %d %+d = %d\n", s.BaseBudget, s.BudgetMod, s.FinalBudget)
	fmt.Fprintf(&b, "Spesi: %d | Residui: %d\n", s.Spent, s.Remaining)
	fmt.Fprintf(&b, "Nemici aggiunti: %d (gruppi estratti: %d)\n", s.AddedEntries, s.AddedGroups)
	b.WriteString("\nDettaglio:\n")
	if len(s.ByMonsterName) == 0 {
		b.WriteString("- Nessun mostro aggiunto.\n")
		return strings.TrimSpace(b.String())
	}
	names := make([]string, 0, len(s.ByMonsterName))
	for name := range s.ByMonsterName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&b, "- %s x%d\n", name, s.ByMonsterName[name])
	}
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) removeSelectedEncounter() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	label := ui.encounterLabelAt(idx)
	ui.beginUndoableChange()
	ui.encounter = append(ui.encounter[:idx], ui.encounter[idx+1:]...)
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Rimosso %s da encounter.", label)
	ui.refreshAll()
}

func (ui *tviewUI) clearAllPNGs() {
	if len(ui.pngs) == 0 {
		ui.message = "Nessun PNG da eliminare."
		ui.refreshStatus()
		return
	}
	count := len(ui.pngs)
	ui.beginUndoableChange()
	ui.pngs = []PNG{}
	ui.selected = -1
	ui.persistPNGs()
	ui.message = fmt.Sprintf("Eliminati %d PNG.", count)
	ui.refreshAll()
	ui.focusPanel(focusPNG)
}

func (ui *tviewUI) clearAllEncounter() {
	if len(ui.encounter) == 0 {
		ui.message = "Encounter già vuoto."
		ui.refreshStatus()
		return
	}
	count := len(ui.encounter)
	ui.beginUndoableChange()
	ui.encounter = []EncounterEntry{}
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Eliminati %d elementi da Encounter.", count)
	ui.refreshAll()
	ui.focusPanel(focusEncounter)
}

func clampStat(value int, max int) int {
	if value < 0 {
		return 0
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

// adjustVitalsLikeEncounter applies PF and Stress deltas with the same semantics used in Encounter:
// - stress decrease below zero converts into PF loss (1-to-1)
// - values are clamped to [0..max] when max > 0, otherwise only lower bound is enforced.
func adjustVitalsLikeEncounter(currentPF, currentStress, maxPF, maxStress, pfDelta, stressDelta int) (int, int) {
	currentPF = clampStat(currentPF, maxPF)
	currentStress = clampStat(currentStress, maxStress)

	if pfDelta != 0 {
		currentPF = clampStat(currentPF+pfDelta, maxPF)
	}

	if stressDelta > 0 {
		currentStress = clampStat(currentStress+stressDelta, maxStress)
	} else if stressDelta < 0 {
		steps := -stressDelta
		for range steps {
			if currentStress > 0 {
				currentStress--
			} else if currentPF > 0 {
				currentPF--
			}
		}
	}
	return clampStat(currentPF, maxPF), clampStat(currentStress, maxStress)
}

func (ui *tviewUI) adjustSelectedPNGVitals(pfDelta, stressDelta int) {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	p := &ui.pngs[ui.selected]
	newPF, newStress := adjustVitalsLikeEncounter(p.PF, p.Stress, 0, 0, pfDelta, stressDelta)
	if newPF == p.PF && newStress == p.Stress {
		return
	}
	ui.beginUndoableChange()
	p = &ui.pngs[ui.selected]
	newPF, newStress = adjustVitalsLikeEncounter(p.PF, p.Stress, 0, 0, pfDelta, stressDelta)
	p.PF = newPF
	p.Stress = newStress
	ui.persistPNGs()
	ui.message = fmt.Sprintf("PNG %s: PF %d | ST %d", p.Name, p.PF, p.Stress)
	ui.refreshPNGs()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) adjustSelectedPNGArmor(delta int) {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	p := &ui.pngs[ui.selected]
	next := max(p.ArmorScore+delta, 0)
	if next == p.ArmorScore {
		return
	}
	ui.beginUndoableChange()
	p = &ui.pngs[ui.selected]
	p.ArmorScore = next
	ui.persistPNGs()
	ui.message = fmt.Sprintf("PNG %s: Armatura %d", p.Name, p.ArmorScore)
	ui.refreshPNGs()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) adjustSelectedPNGHope(delta int) {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	p := &ui.pngs[ui.selected]
	next := max(p.Hope+delta, 0)
	if next == p.Hope {
		return
	}
	ui.beginUndoableChange()
	p = &ui.pngs[ui.selected]
	p.Hope = next
	ui.persistPNGs()
	ui.message = fmt.Sprintf("PNG %s: Speranza %d", p.Name, p.Hope)
	ui.refreshPNGs()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) adjustEncounterWounds(delta int) {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	e := &ui.encounter[idx]
	base := e.BasePF
	if base == 0 {
		base = e.Monster.PF
	}
	baseStress := e.BaseStress
	if baseStress == 0 {
		baseStress = e.Monster.Stress
	}
	currentPF := max(base-e.Wounds, 0)
	currentPF, currentStress := adjustVitalsLikeEncounter(currentPF, e.Stress, base, baseStress, -delta, 0)
	nextWounds := max(base-currentPF, 0)
	if nextWounds == e.Wounds && currentStress == e.Stress {
		return
	}
	ui.beginUndoableChange()
	e = &ui.encounter[idx]
	base = e.BasePF
	if base == 0 {
		base = e.Monster.PF
	}
	baseStress = e.BaseStress
	if baseStress == 0 {
		baseStress = e.Monster.Stress
	}
	currentPF = max(base-e.Wounds, 0)
	currentPF, currentStress = adjustVitalsLikeEncounter(currentPF, e.Stress, base, baseStress, -delta, 0)
	e.Wounds = max(base-currentPF, 0)
	e.Stress = currentStress
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Ferite %s: %d", e.Monster.Name, e.Wounds)
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) adjustEncounterStress(delta int) {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	e := &ui.encounter[idx]
	baseStress := e.BaseStress
	if baseStress == 0 {
		baseStress = e.Monster.Stress
	}
	if e.Stress < 0 {
		e.Stress = 0
	}
	if baseStress > 0 && e.Stress > baseStress {
		e.Stress = baseStress
	}

	basePF := e.BasePF
	if basePF == 0 {
		basePF = e.Monster.PF
	}
	currentPF := max(basePF-e.Wounds, 0)
	currentPF, currentStress := adjustVitalsLikeEncounter(currentPF, e.Stress, basePF, baseStress, 0, delta)
	nextWounds := max(basePF-currentPF, 0)
	if nextWounds == e.Wounds && currentStress == e.Stress {
		return
	}
	ui.beginUndoableChange()
	e = &ui.encounter[idx]
	baseStress = e.BaseStress
	if baseStress == 0 {
		baseStress = e.Monster.Stress
	}
	basePF = e.BasePF
	if basePF == 0 {
		basePF = e.Monster.PF
	}
	currentPF = max(basePF-e.Wounds, 0)
	currentPF, currentStress = adjustVitalsLikeEncounter(currentPF, e.Stress, basePF, baseStress, 0, delta)
	e.Wounds = max(basePF-currentPF, 0)
	e.Stress = currentStress

	ui.persistEncounter()
	ui.message = fmt.Sprintf("Stress %s: %d/%d", e.Monster.Name, e.Stress, baseStress)
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) openHelpOverlay(focus tview.Primitive) {
	if ui.helpVisible {
		return
	}
	ui.helpVisible = true
	ui.helpReturnFocus = focus

	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	text.SetBorder(true).SetTitle("Help")
	text.SetText(ui.buildHelpContent(focus))

	modal := ui.fullscreenModal(text)

	ui.pages.AddAndSwitchToPage("help", modal, true)
	ui.app.SetFocus(text)
}

func (ui *tviewUI) buildHelpContent(focus tview.Primitive) string {
	var b strings.Builder
	b.WriteString("LazyDaggerheart - scorciatoie\n\n")

	panel := "Dettagli"
	var panelLines []string
	switch focus {
	case ui.dice:
		panel = "Dadi"
		panelLines = []string{
			"- a: nuovo tiro (anche multiplo, es. 1d20+3 x2)",
			"- Invio: rilancia il tiro selezionato",
			"- g# / g^ / g$: vai a riga # / prima / ultima",
			"- e: modifica + rilancia il tiro selezionato",
			"- d: elimina il tiro selezionato",
			"- c: svuota storico tiri",
			"- s / l: salva / carica dadi da file",
			"- Sintassi tiro: NdM, NdM+K, NdM-K, dM",
			"- Batch: expr xN (es. 1d20+5 x3)",
			"- Multi-espr.: expr,expr (es. d6,d8,1d20+4)",
			"- Confronto: expr>DC, expr>=DC, expr<DC, expr<=DC",
		}
	case ui.pngList:
		panel = "PNG"
		panelLines = []string{
			"- a: crea PNG",
			"- e: modifica PNG selezionato",
			"- s / l: salva / carica PNG da file",
			"- d: elimina PNG selezionato (senza conferma)",
			"- D: elimina tutti i PNG (senza conferma)",
			"- R: reset token di tutti i PNG",
			"- ← / →: diminuisci/aumenta token selezionato",
			"- Shift+← / Shift+→: PF -1 / +1 sul selezionato",
			"- Shift+↓ / Shift+↑: stress -1 / +1 sul selezionato",
			"- Alt+← / Alt+→: armatura -1 / +1",
			"- Alt+↓ / Alt+↑: speranza -1 / +1",
			"- stress a 0: ulteriore riduzione stress riduce PF",
		}
	case ui.encList:
		panel = "Encounter"
		panelLines = []string{
			"- s / l: salva / carica Encounter da file",
			"- d: rimuovi mostro selezionato (senza conferma)",
			"- D: svuota Encounter (senza conferma)",
			"- Shift+← / Shift+→: PF -1 / +1 sul selezionato",
			"- Shift+↓ / Shift+↑: stress -1 / +1 sul selezionato",
			"- stress a 0: ulteriore riduzione stress riduce PF",
		}
	case ui.notesSearch, ui.notesList:
		panel = "Note"
		panelLines = []string{
			"- a: nuova nota (editor, Ctrl+S salva / Esc annulla)",
			"- e: modifica nota selezionata",
			"- d: elimina nota selezionata",
			"- U / t / g: focus filtro Nome",
			"- v: reset filtro Note",
		}
	case ui.search, ui.roleDrop, ui.rankDrop, ui.monList:
		panel = "Mostri"
		panelLines = []string{
			"- a: aggiungi mostro selezionato a Encounter",
			"- n: genera Encounter random (Punti Battaglia)",
			"- U / t / g: focus filtro Nome / Ruolo / Rango",
			"- v: reset filtri Mostri",
		}
	case ui.envSearch, ui.envTypeDrop, ui.envRankDrop, ui.envList:
		panel = "Ambienti"
		panelLines = []string{
			"- U / t / g: focus filtro Nome / Tipo / Rango",
			"- v: reset filtri Ambienti (Nome/Tipo/Rango)",
		}
	case ui.eqSearch, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqList:
		panel = "Equipaggiamento"
		panelLines = []string{
			"- U / t / g / y: focus filtro Nome / Categoria / Rango / Tipo",
			"- v: reset filtri Equipaggiamento (Nome/Categoria/Tipo/Rango)",
			"- b: genera bottino (Treasure) da categoria + dadi",
			"- d: switch Dettagli <-> Treasure",
		}
	case ui.detailTreasure:
		panel = "Treasure"
		panelLines = []string{
			"- d: switch Treasure <-> Dettagli",
			"- /: evidenzia testo nel treasure corrente",
		}
	case ui.cardSearch, ui.cardClassDrop, ui.cardTypeDrop, ui.cardList:
		panel = "Carte"
		panelLines = []string{
			"- U / t / g: focus filtro Nome / Classe / Tipo",
			"- v: reset filtri Carte (Nome/Classe/Tipo)",
		}
	case ui.classSearch, ui.classNameDrop, ui.classSubDrop, ui.classList:
		panel = "Classe"
		panelLines = []string{
			"- U / t / g: focus filtro Cerca / Classe / Sottoclasse",
			"- v: reset filtri Classe (Cerca/Classe/Sottoclasse)",
			"- a: genera PNG dalla classe selezionata (con livello)",
		}
	default:
		panelLines = []string{"- /: evidenzia testo nei dettagli"}
	}

	b.WriteString("[yellow]" + panel + "[-]\n")
	for _, line := range panelLines {
		b.WriteString(line + "\n")
	}

	b.WriteString("\n[yellow]Globali[-]\n")
	b.WriteString("- q: esci\n")
	b.WriteString("- ?: apri/chiudi help\n")
	b.WriteString("- tab / shift+tab: cambia focus\n")
	b.WriteString("- 0 / 1 / 2 / 3 / 4: focus Dadi / PNG / Encounter / Catalogo / Note\n")
	b.WriteString("- + / -: aumenta / diminuisce Paure (0..12)\n")
	b.WriteString("- Shift+S: salva Paure su file\n")
	b.WriteString("- Shift+L: carica Paure da file\n")
	b.WriteString("- [ / ]: alterna Mostri / Ambienti / Equipaggiamento / Carte / Classe / Note\n")
	b.WriteString("- G: apri modal 'Vai a pannello' (include Note)\n")
	b.WriteString("- N: focus diretto su Note\n")
	b.WriteString("- u / r: undo / redo\n")
	b.WriteString("- /: ricerca rapida sul pannello corrente\n")
	b.WriteString("- f: fullscreen pannello corrente\n")
	b.WriteString("- PgUp / PgDn: scroll Dettagli\n")
	b.WriteString("\nEsc/?/q per chiudere")
	return b.String()
}

func defaultDiceFilePath() string {
	if strings.TrimSpace(appStateDir) != "" {
		return filepath.Join(appStateDir, "dice.yml")
	}
	return "dice.yml"
}

type dicePersist struct {
	Dice    []DiceResult `yaml:"dice"`
	Current int          `yaml:"current,omitempty"`
}

func saveDiceLog(path string, log []DiceResult, current int) error {
	payload := dicePersist{Dice: log, Current: current}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadDiceLog(path string) ([]DiceResult, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	var payload dicePersist
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, 0, err
	}
	if payload.Dice == nil {
		payload.Dice = []DiceResult{}
	}
	return payload.Dice, payload.Current, nil
}

func (ui *tviewUI) buildEncounterPersistEntries() []encounterPersistEntry {
	entries := make([]encounterPersistEntry, 0, len(ui.encounter))
	for _, e := range ui.encounter {
		base := e.BasePF
		if base == 0 {
			base = e.Monster.PF
		}
		baseStress := e.BaseStress
		if baseStress == 0 {
			baseStress = e.Monster.Stress
		}
		currentStress := max(e.Stress, 0)
		if baseStress > 0 && currentStress > baseStress {
			currentStress = baseStress
		}
		entries = append(entries, encounterPersistEntry{
			Name:       e.Monster.Name,
			Seq:        e.Seq,
			Wounds:     e.Wounds,
			PF:         base,
			Stress:     currentStress,
			BaseStress: baseStress,
		})
	}
	return entries
}

func (ui *tviewUI) openStateFileModal(action, target string) {
	if ui.modalVisible {
		return
	}
	returnFocus := ui.app.GetFocus()

	targetLabel := map[string]string{
		"png":       "PNG",
		"encounter": "Encounter",
		"dice":      "Dadi",
		"fear":      "Paure",
	}[target]
	if targetLabel == "" {
		targetLabel = target
	}

	defaultPath := ""
	switch target {
	case "png":
		defaultPath = dataFile
	case "encounter":
		defaultPath = encounterFile
	case "dice":
		defaultPath = defaultDiceFilePath()
	case "fear":
		defaultPath = fearStateFile
	}

	form := tview.NewForm()
	titleVerb := "Salva"
	btnVerb := "Salva"
	if action == "load" {
		titleVerb = "Carica"
		btnVerb = "Carica"
	}
	form.SetBorder(true).SetTitle(fmt.Sprintf("%s %s", titleVerb, targetLabel)).SetTitleAlign(tview.AlignLeft)
	form.AddInputField("File", defaultPath, 56, nil, nil)
	form.AddButton(btnVerb, func() {
		path := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		if path == "" {
			ui.message = "Percorso file non valido."
			ui.refreshStatus()
			return
		}

		switch {
		case action == "save" && target == "png":
			if err := savePNGList(path, ui.pngs, selectedPNGName(ui.pngs, ui.selected)); err != nil {
				ui.message = fmt.Sprintf("Errore salvataggio PNG: %v", err)
				ui.refreshStatus()
				return
			}
		case action == "load" && target == "png":
			pngs, selectedName, err := loadPNGList(path)
			if err != nil {
				ui.message = fmt.Sprintf("Errore caricamento PNG: %v", err)
				ui.refreshStatus()
				return
			}
			ui.beginUndoableChange()
			ui.pngs = pngs
			ui.selected = -1
			if selectedName != "" {
				for i, p := range ui.pngs {
					if p.Name == selectedName {
						ui.selected = i
						break
					}
				}
			}
			if ui.selected < 0 && len(ui.pngs) > 0 {
				ui.selected = 0
			}
			ui.persistPNGs()
			ui.refreshPNGs()
		case action == "save" && target == "encounter":
			if err := saveEncounter(path, ui.buildEncounterPersistEntries()); err != nil {
				ui.message = fmt.Sprintf("Errore salvataggio Encounter: %v", err)
				ui.refreshStatus()
				return
			}
		case action == "load" && target == "encounter":
			entries, err := loadEncounter(path, ui.monsters)
			if err != nil {
				ui.message = fmt.Sprintf("Errore caricamento Encounter: %v", err)
				ui.refreshStatus()
				return
			}
			ui.beginUndoableChange()
			ui.encounter = entries
			ui.persistEncounter()
			ui.refreshEncounter()
		case action == "save" && target == "dice":
			if err := saveDiceLog(path, ui.diceLog, ui.dice.GetCurrentItem()); err != nil {
				ui.message = fmt.Sprintf("Errore salvataggio Dadi: %v", err)
				ui.refreshStatus()
				return
			}
		case action == "load" && target == "dice":
			log, current, err := loadDiceLog(path)
			if err != nil {
				ui.message = fmt.Sprintf("Errore caricamento Dadi: %v", err)
				ui.refreshStatus()
				return
			}
			ui.diceLog = log
			ui.renderDiceList()
			if len(ui.diceLog) > 0 {
				if current < 0 {
					current = 0
				}
				if current >= len(ui.diceLog) {
					current = len(ui.diceLog) - 1
				}
				ui.dice.SetCurrentItem(current)
			}
		case action == "save" && target == "fear":
			if err := saveFearState(path, ui.paure); err != nil {
				ui.message = fmt.Sprintf("Errore salvataggio Paure: %v", err)
				ui.refreshStatus()
				return
			}
		case action == "load" && target == "fear":
			paure, err := loadFearState(path)
			if err != nil {
				ui.message = fmt.Sprintf("Errore caricamento Paure: %v", err)
				ui.refreshStatus()
				return
			}
			ui.paure = clampFear(paure)
			ui.persistPaure()
		default:
			ui.message = "Operazione non supportata."
			ui.refreshStatus()
			return
		}

		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshDetail()
		ui.message = fmt.Sprintf("%s %s completato: %s", titleVerb, targetLabel, path)
		ui.refreshStatus()
	})
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetButtonsAlign(tview.AlignLeft)
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyEnter {
			return event
		}
		itemIdx, _ := form.GetFocusedItemIndex()
		if itemIdx == 0 {
			form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex(btnVerb))
			return nil
		}
		return event
	})

	modal := ui.fullscreenModal(form)

	ui.modalVisible = true
	ui.modalName = "state_file_modal"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
}

func (ui *tviewUI) openGotoPanelModal() {
	if ui.modalVisible {
		return
	}

	returnFocus := ui.app.GetFocus()
	selectPanel := func(label string, focusID int, catalogPage string) {
		ui.closeModal()
		if catalogPage != "" {
			ui.catalogMode = catalogPage
			ui.catalogPanel.SwitchToPage(catalogPage)
			ui.refreshCatalogTitles()
		}
		ui.focusPanel(focusID)
		if focusID >= 0 && focusID < len(ui.focus) && ui.focus[focusID] != nil {
			ui.app.SetFocus(ui.focus[focusID])
		}
		ui.message = fmt.Sprintf("Focus: %s", label)
		ui.refreshDetail()
		ui.refreshStatus()
	}

	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	text.SetBorder(true).SetTitle("Vai a pannello (premi un tasto)")
	text.SetText(strings.Join([]string{
		"[yellow]0[-] Dadi",
		"[yellow]1[-] PNG",
		"[yellow]2[-] Encounter",
		"[yellow]3[-] Mostri",
		"[yellow]4[-] Ambienti",
		"[yellow]5[-] Equipaggiamento",
		"[yellow]6[-] Carte",
		"[yellow]7[-] Classe",
		"[yellow]N[-] Note",
		"[yellow]8[-] Dettagli",
		"[yellow]9[-] Treasure",
		"",
		"Esc / q per annullare",
	}, "\n"))
	text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && (event.Rune() == 'q' || event.Rune() == 'Q')) {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			ui.refreshStatus()
			return nil
		}
		if event.Key() != tcell.KeyRune {
			return event
		}
		switch event.Rune() {
		case '0':
			selectPanel("Dadi", focusDice, "")
		case '1':
			selectPanel("PNG", focusPNG, "")
		case '2':
			selectPanel("Encounter", focusEncounter, "")
		case '3':
			selectPanel("Mostri", focusMonList, "mostri")
		case '4':
			selectPanel("Ambienti", focusEnvList, "ambienti")
		case '5':
			selectPanel("Equipaggiamento", focusEqList, "equipaggiamento")
		case '6':
			selectPanel("Carte", focusCardList, "carte")
		case '7':
			selectPanel("Classe", focusClassList, "classe")
		case 'n', 'N':
			selectPanel("Note", focusNotesList, "note")
		case '8':
			ui.activeBottomPane = "details"
			ui.detailBottom.SwitchToPage("details")
			selectPanel("Dettagli", focusDetail, "")
		case '9':
			ui.activeBottomPane = "treasure"
			ui.detailBottom.SwitchToPage("treasure")
			selectPanel("Treasure", focusTreasure, "")
		default:
			return event
		}
		return nil
	})

	modal := ui.fullscreenModal(text)

	ui.modalVisible = true
	ui.modalName = "goto_panel"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(text)
}

func (ui *tviewUI) closeHelpOverlay() {
	if !ui.helpVisible {
		return
	}
	ui.helpVisible = false
	ui.pages.RemovePage("help")
	if ui.helpReturnFocus != nil {
		ui.app.SetFocus(ui.helpReturnFocus)
	}
}

func (ui *tviewUI) fullscreenModal(content tview.Primitive) tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true)
}

func (ui *tviewUI) closeModal() {
	if !ui.modalVisible {
		return
	}
	if ui.modalName != "" {
		ui.pages.RemovePage(ui.modalName)
	}
	ui.modalVisible = false
	ui.modalName = ""
}

func (ui *tviewUI) fullscreenTargetForFocus(focus tview.Primitive) string {
	switch focus {
	case ui.dice:
		return "dadi"
	case ui.pngList:
		return "png"
	case ui.encList:
		return "encounter"
	case ui.notesSearch, ui.notesList:
		return "note"
	case ui.search, ui.monList, ui.roleDrop, ui.rankDrop:
		return "monsters"
	case ui.envSearch, ui.envList, ui.envTypeDrop, ui.envRankDrop:
		return "ambienti"
	case ui.eqSearch, ui.eqList, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop:
		return "equipaggiamento"
	case ui.cardSearch, ui.cardList, ui.cardClassDrop, ui.cardTypeDrop:
		return "carte"
	case ui.classSearch, ui.classList, ui.classNameDrop, ui.classSubDrop:
		return "classe"
	case ui.detailTreasure:
		return "treasure"
	case ui.detail:
		return "details"
	default:
		return ""
	}
}

func (ui *tviewUI) toggleFullscreenForFocus(focus tview.Primitive) {
	target := ui.fullscreenTargetForFocus(focus)
	if target == "" {
		return
	}
	if ui.fullscreenActive && ui.fullscreenTarget == target {
		ui.fullscreenActive = false
		ui.fullscreenTarget = ""
		ui.rebuildMainLayout()
		ui.message = "Fullscreen off"
		ui.refreshStatus()
		return
	}
	ui.fullscreenActive = true
	ui.fullscreenTarget = target
	ui.rebuildMainLayout()
	ui.message = "Fullscreen " + target
	ui.refreshStatus()
}

func (ui *tviewUI) rebuildMainLayout() {
	var content tview.Primitive = ui.mainRow
	if ui.fullscreenActive {
		switch ui.fullscreenTarget {
		case "dadi":
			content = ui.dice
		case "png":
			content = ui.pngList
		case "encounter":
			content = ui.encList
		case "note":
			content = ui.notesList
		case "monsters":
			content = ui.monstersPanel
		case "ambienti":
			content = ui.environmentsPanel
		case "equipaggiamento":
			content = ui.equipmentPanel
		case "carte":
			content = ui.cardsPanel
		case "classe":
			content = ui.classesPanel
		case "treasure":
			content = ui.detailTreasure
		case "details":
			content = ui.detail
		}
	}
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(ui.status, 1, 0, false)
	ui.pages.RemovePage("main")
	ui.pages.AddPage("main", root, true, true)
	ui.pages.SwitchToPage("main")
}

func clonePNGs(src []PNG) []PNG {
	if len(src) == 0 {
		return nil
	}
	dst := make([]PNG, len(src))
	copy(dst, src)
	return dst
}

func cloneEncounter(src []EncounterEntry) []EncounterEntry {
	if len(src) == 0 {
		return nil
	}
	dst := make([]EncounterEntry, len(src))
	copy(dst, src)
	return dst
}

func (ui *tviewUI) captureSnapshot() uiSnapshot {
	return uiSnapshot{
		pngs:      clonePNGs(ui.pngs),
		encounter: cloneEncounter(ui.encounter),
		selected:  ui.selected,
	}
}

func (ui *tviewUI) beginUndoableChange() {
	ui.undoStack = append(ui.undoStack, ui.captureSnapshot())
	if len(ui.undoStack) > historyLimit {
		ui.undoStack = ui.undoStack[len(ui.undoStack)-historyLimit:]
	}
	ui.redoStack = nil
}

func (ui *tviewUI) applySnapshot(s uiSnapshot) {
	ui.pngs = clonePNGs(s.pngs)
	ui.encounter = cloneEncounter(s.encounter)
	ui.selected = s.selected
	if len(ui.pngs) == 0 {
		ui.selected = -1
	} else if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.selected = len(ui.pngs) - 1
	}
}

func (ui *tviewUI) undoLastChange() {
	if len(ui.undoStack) == 0 {
		ui.message = "Undo: nessuna modifica disponibile."
		ui.refreshStatus()
		return
	}
	current := ui.captureSnapshot()
	last := ui.undoStack[len(ui.undoStack)-1]
	ui.undoStack = ui.undoStack[:len(ui.undoStack)-1]
	ui.redoStack = append(ui.redoStack, current)
	if len(ui.redoStack) > historyLimit {
		ui.redoStack = ui.redoStack[len(ui.redoStack)-historyLimit:]
	}
	ui.applySnapshot(last)
	ui.persistPNGs()
	ui.persistEncounter()
	ui.message = "Undo eseguito."
	ui.refreshAll()
}

func (ui *tviewUI) redoLastChange() {
	if len(ui.redoStack) == 0 {
		ui.message = "Redo: nessuna modifica disponibile."
		ui.refreshStatus()
		return
	}
	current := ui.captureSnapshot()
	next := ui.redoStack[len(ui.redoStack)-1]
	ui.redoStack = ui.redoStack[:len(ui.redoStack)-1]
	ui.undoStack = append(ui.undoStack, current)
	if len(ui.undoStack) > historyLimit {
		ui.undoStack = ui.undoStack[len(ui.undoStack)-historyLimit:]
	}
	ui.applySnapshot(next)
	ui.persistPNGs()
	ui.persistEncounter()
	ui.message = "Redo eseguito."
	ui.refreshAll()
}

func (ui *tviewUI) persistPNGs() {
	_ = savePNGList(dataFile, ui.pngs, selectedPNGName(ui.pngs, ui.selected))
}

func (ui *tviewUI) persistEncounter() {
	_ = saveEncounter(encounterFile, ui.buildEncounterPersistEntries())
}

func (ui *tviewUI) persistPaure() {
	_ = saveFearState(fearStateFile, ui.paure)
}

func (ui *tviewUI) persistNotes() {
	_ = saveNotes(notesFile, ui.notes)
}

func (ui *tviewUI) adjustPaure(delta int) {
	next := clampFear(ui.paure + delta)
	if next == ui.paure {
		return
	}
	ui.paure = next
	ui.persistPaure()
	ui.refreshStatus()
}

func (ui *tviewUI) openAddNoteModal() {
	if ui.modalVisible {
		return
	}
	returnFocus := ui.app.GetFocus()
	editor := tview.NewTextArea()
	editor.SetBorder(true).SetTitle("Nuova Nota (Ctrl+S salva, Esc annulla)")

	editor.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			ui.refreshStatus()
			return nil
		}
		if ev.Key() == tcell.KeyCtrlS {
			text := strings.TrimSpace(editor.GetText())
			if text == "" {
				ui.message = "Nota vuota: annullata."
				ui.closeModal()
				ui.app.SetFocus(returnFocus)
				ui.refreshStatus()
				return nil
			}
			ui.notes = append(ui.notes, text)
			ui.persistNotes()
			ui.refreshNotes()
			ui.notesList.SetCurrentItem(len(ui.notes) - 1)
			ui.closeModal()
			ui.catalogMode = "note"
			ui.catalogPanel.SwitchToPage("note")
			ui.refreshCatalogTitles()
			ui.focusPanel(focusNotesList)
			ui.message = "Nota aggiunta."
			ui.refreshDetail()
			ui.refreshStatus()
			return nil
		}
		return ev
	})

	modal := ui.fullscreenModal(editor)

	ui.modalVisible = true
	ui.modalName = "add_note"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(editor)
}

func (ui *tviewUI) openEditNoteModal() {
	idx := ui.currentNoteIndex()
	if idx < 0 || idx >= len(ui.notes) {
		ui.message = "Nessuna nota da modificare."
		ui.refreshStatus()
		return
	}
	if ui.modalVisible {
		return
	}
	returnFocus := ui.app.GetFocus()
	editor := tview.NewTextArea()
	editor.SetBorder(true).SetTitle("Modifica Nota (Ctrl+S salva, Esc annulla)")
	editor.SetText(ui.notes[idx], true)

	editor.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			ui.refreshStatus()
			return nil
		}
		if ev.Key() == tcell.KeyCtrlS {
			text := strings.TrimSpace(editor.GetText())
			if text == "" {
				ui.message = "Nota vuota: modifica annullata."
				ui.closeModal()
				ui.app.SetFocus(returnFocus)
				ui.refreshStatus()
				return nil
			}
			ui.notes[idx] = text
			ui.persistNotes()
			ui.refreshNotes()
			// Reseleziona l'elemento modificato nel filtro corrente.
			for li, ni := range ui.filteredNotes {
				if ni == idx {
					ui.notesList.SetCurrentItem(li)
					break
				}
			}
			ui.closeModal()
			ui.catalogMode = "note"
			ui.catalogPanel.SwitchToPage("note")
			ui.refreshCatalogTitles()
			ui.focusPanel(focusNotesList)
			ui.message = "Nota aggiornata."
			ui.refreshDetail()
			ui.refreshStatus()
			return nil
		}
		return ev
	})

	modal := ui.fullscreenModal(editor)

	ui.modalVisible = true
	ui.modalName = "edit_note"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(editor)
}

func (ui *tviewUI) deleteSelectedNote() {
	idx := ui.currentNoteIndex()
	if idx < 0 || idx >= len(ui.notes) {
		ui.message = "Nessuna nota da eliminare."
		ui.refreshStatus()
		return
	}
	ui.notes = append(ui.notes[:idx], ui.notes[idx+1:]...)
	ui.persistNotes()
	ui.refreshNotes()
	if len(ui.filteredNotes) > 0 {
		next := 0
		for li, ni := range ui.filteredNotes {
			if ni >= idx {
				next = li
				break
			}
			next = li
		}
		ui.notesList.SetCurrentItem(next)
	}
	ui.message = "Nota eliminata."
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) toggleDetailsTreasureFocus() {
	if ui.activeBottomPane == "treasure" {
		ui.activeBottomPane = "details"
		ui.detailBottom.SwitchToPage("details")
		ui.app.SetFocus(ui.detail)
		ui.message = "Focus: Dettagli"
		ui.refreshStatus()
		return
	}
	ui.activeBottomPane = "treasure"
	ui.detailBottom.SwitchToPage("treasure")
	ui.app.SetFocus(ui.detailTreasure)
	ui.message = "Focus: Treasure"
	ui.refreshStatus()
}

func (ui *tviewUI) renderTreasure() {
	text := ui.treasureRaw
	if strings.TrimSpace(text) == "" {
		text = "Nessun treasure generato."
	}
	out := tview.Escape(text)
	lines := strings.Split(out, "\n")
	if len(lines) > 0 {
		lines[0] = "[yellow]" + lines[0] + "[-]"
		out = strings.Join(lines, "\n")
	}
	if strings.TrimSpace(ui.detailQuery) != "" {
		out = highlightMatches(out, ui.detailQuery)
	}
	ui.detailTreasure.SetText(out)
}

func (ui *tviewUI) openEquipmentTreasureInput() {
	categories := []string{"Comune", "Non Comune", "Raro", "Leggendario"}
	diceByCategory := map[string][]string{
		"Comune":      {"1d12", "2d12"},
		"Non Comune":  {"2d12", "3d12"},
		"Raro":        {"3d12", "4d12"},
		"Leggendario": {"4d12", "5d12"},
	}
	selectedCategory := categories[0]
	selectedDice := diceByCategory[selectedCategory][0]
	ready := false
	suppressDiceAdvance := false

	form := tview.NewForm()
	var categoryDrop *tview.DropDown
	var diceDrop *tview.DropDown
	advanceToGenerate := func() {
		form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
	}
	form.SetBorder(true).SetTitle("Genera Treasure da Bottino").SetTitleAlign(tview.AlignLeft)
	form.AddDropDown("Categoria", categories, 0, func(option string, _ int) {
		if option == "" {
			return
		}
		selectedCategory = option
		selectedDice = diceByCategory[selectedCategory][0]
		if diceDrop == nil {
			return
		}
		diceDrop.SetOptions(diceByCategory[selectedCategory], func(text string, _ int) {
			if text != "" {
				selectedDice = text
			}
			if ready && !suppressDiceAdvance {
				advanceToGenerate()
			}
		})
		diceDrop.SetListStyles(
			tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
			tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
		)
		suppressDiceAdvance = true
		diceDrop.SetCurrentOption(0)
		suppressDiceAdvance = false
		if ready {
			form.SetFocus(1)
		}
	})
	form.AddDropDown("Dadi", diceByCategory[selectedCategory], 0, func(option string, _ int) {
		if option != "" {
			selectedDice = option
		}
		if ready && !suppressDiceAdvance {
			advanceToGenerate()
		}
	})
	if item := form.GetFormItem(0); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			categoryDrop = dd
			categoryDrop.SetFinishedFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter, tcell.KeyTab:
					form.SetFocus(1)
				case tcell.KeyBacktab:
					form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Annulla"))
				}
			})
		}
	}
	if item := form.GetFormItem(1); item != nil {
		if dd, ok := item.(*tview.DropDown); ok {
			diceDrop = dd
			diceDrop.SetFinishedFunc(func(key tcell.Key) {
				switch key {
				case tcell.KeyEnter, tcell.KeyTab:
					form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Genera"))
				case tcell.KeyBacktab:
					form.SetFocus(0)
				}
			})
		}
	}
	applyDropStyle := func(dd *tview.DropDown) {
		if dd == nil {
			return
		}
		dd.SetFieldBackgroundColor(tcell.ColorBlack)
		dd.SetFieldTextColor(tcell.ColorWhite)
		dd.SetListStyles(
			tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
			tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold),
		)
	}
	applyDropStyle(categoryDrop)
	applyDropStyle(diceDrop)

	returnFocus := ui.app.GetFocus()
	form.AddButton("Genera", func() {
		total, breakdown, err := rollDiceExpression(selectedDice)
		if err != nil {
			ui.message = "Errore tiro treasure: " + err.Error()
			ui.refreshStatus()
			return
		}
		matches := ui.matchBottinoByTiro(total)
		ui.renderEquipmentTreasure(selectedCategory, selectedDice, total, breakdown, matches)
		ui.closeModal()
		ui.activeBottomPane = "treasure"
		ui.detailBottom.SwitchToPage("treasure")
		ui.app.SetFocus(ui.detailTreasure)
		ui.message = fmt.Sprintf("Treasure generato: %s %s = %02d", selectedCategory, selectedDice, total)
		ui.refreshStatus()
	})
	form.AddButton("Annulla", func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})
	form.SetCancelFunc(func() {
		ui.closeModal()
		ui.app.SetFocus(returnFocus)
		ui.refreshStatus()
	})

	modal := ui.fullscreenModal(form)

	ui.modalVisible = true
	ui.modalName = "equip_treasure"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form)
	ready = true
}

func (ui *tviewUI) matchBottinoByTiro(total int) []EquipmentItem {
	var matches []EquipmentItem
	for _, it := range ui.equipment {
		if !strings.EqualFold(strings.TrimSpace(it.Type), "bottino") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(it.Trait))
		if err != nil {
			continue
		}
		if n == total {
			matches = append(matches, it)
		}
	}
	return matches
}

func (ui *tviewUI) renderEquipmentTreasure(category, dice string, total int, breakdown string, matches []EquipmentItem) {
	var b strings.Builder
	b.WriteString("Treasure Equipaggiamento\n")
	fmt.Fprintf(&b, "Categoria: %s\n", category)
	fmt.Fprintf(&b, "Tiro: %s => %s\n", dice, breakdown)
	fmt.Fprintf(&b, "Valore Tiro: %02d\n", total)
	b.WriteString("\nRisultati:\n")
	if len(matches) == 0 {
		b.WriteString("- Nessun bottino con Tiro corrispondente.\n")
	} else {
		for _, it := range matches {
			fmt.Fprintf(&b, "- %s (Tiro %02d)\n", it.Name, total)
			if strings.TrimSpace(it.Characteristic) != "" && strings.TrimSpace(it.Characteristic) != "—" && strings.TrimSpace(it.Characteristic) != "-" {
				b.WriteString("  " + strings.TrimSpace(it.Characteristic) + "\n")
			}
		}
	}
	ui.treasureRaw = strings.TrimSpace(b.String())
	ui.renderTreasure()
	ui.detailTreasure.ScrollToBeginning()
}

func (ui *tviewUI) buildDiceDetail() string {
	var b strings.Builder
	b.WriteString("Dadi\n")
	if len(ui.diceLog) == 0 {
		b.WriteString("Nessun tiro registrato.\n\n")
		b.WriteString("Shortcut:\n")
		b.WriteString("- a: nuovo tiro\n")
		b.WriteString("- Invio: rilancia selezionato\n")
		b.WriteString("- e: modifica + rilancia\n")
		b.WriteString("- d: elimina selezionato\n")
		b.WriteString("- c: svuota storico\n")
		return strings.TrimSpace(b.String())
	}

	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	entry := ui.diceLog[cur]
	fmt.Fprintf(&b, "Tiro #%d\n", cur+1)
	b.WriteString("Espressione: " + entry.Expression + "\n")
	b.WriteString("Risultato: " + entry.Output + "\n")
	fmt.Fprintf(&b, "\nTotale tiri: %d", len(ui.diceLog))
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) openDiceRollInput() {
	input := tview.NewInputField().SetLabel(" Tiro ").SetFieldWidth(36)
	input.SetBorder(true).SetTitle("Dadi (es. 2d6+3, 1d20+5>15, d6,d8, 1d20+4 x3)")
	returnFocus := ui.app.GetFocus()

	modal := ui.fullscreenModal(input)

	ui.modalVisible = true
	ui.modalName = "dice_roll"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		raw := strings.TrimSpace(input.GetText())
		if raw == "" {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}

		exprs, err := expandDiceRollInput(raw)
		if err != nil {
			ui.message = "Errore dadi: " + err.Error()
			ui.refreshStatus()
			return
		}
		for _, expr := range exprs {
			_, breakdown, rollErr := rollDiceExpression(expr)
			if rollErr != nil {
				ui.message = "Errore dadi: " + rollErr.Error()
				ui.refreshStatus()
				continue
			}
			ui.appendDiceLog(DiceResult{Expression: expr, Output: breakdown})
		}
		ui.closeModal()
		ui.focusPanel(focusDice)
		ui.message = fmt.Sprintf("Registrati %d tiri.", len(exprs))
		ui.refreshDetail()
		ui.refreshStatus()
	})
}

func (ui *tviewUI) openDiceReRollInput() {
	if len(ui.diceLog) == 0 {
		ui.openDiceRollInput()
		return
	}

	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}

	input := tview.NewInputField().SetLabel(" Tiro ").SetFieldWidth(36)
	input.SetBorder(true).SetTitle("Modifica + Rilancia")
	input.SetText(ui.diceLog[cur].Expression)
	returnFocus := ui.app.GetFocus()

	modal := ui.fullscreenModal(input)

	ui.modalVisible = true
	ui.modalName = "dice_reroll"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		expr := strings.TrimSpace(input.GetText())
		if expr == "" {
			return
		}
		_, breakdown, err := rollDiceExpression(expr)
		if err != nil {
			ui.message = "Errore dadi: " + err.Error()
			ui.refreshStatus()
			return
		}
		ui.diceLog[cur] = DiceResult{Expression: expr, Output: breakdown}
		ui.renderDiceList()
		ui.dice.SetCurrentItem(cur)
		ui.closeModal()
		ui.focusPanel(focusDice)
		ui.message = "Tiro aggiornato."
		ui.refreshDetail()
		ui.refreshStatus()
	})
}

func (ui *tviewUI) rerollSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		ui.message = "Nessun tiro da rilanciare."
		ui.refreshStatus()
		return
	}
	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	expr := strings.TrimSpace(ui.diceLog[cur].Expression)
	if expr == "" {
		ui.message = "Espressione tiro vuota."
		ui.refreshStatus()
		return
	}
	_, breakdown, err := rollDiceExpression(expr)
	if err != nil {
		ui.message = "Errore dadi: " + err.Error()
		ui.refreshStatus()
		return
	}
	ui.diceLog[cur] = DiceResult{Expression: expr, Output: breakdown}
	ui.renderDiceList()
	ui.dice.SetCurrentItem(cur)
	ui.message = "Tiro rilanciato."
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) appendDiceLog(entry DiceResult) {
	ui.diceLog = append(ui.diceLog, entry)
	if len(ui.diceLog) > 200 {
		ui.diceLog = ui.diceLog[len(ui.diceLog)-200:]
	}
	ui.renderDiceList()
	if len(ui.diceLog) > 0 {
		ui.dice.SetCurrentItem(len(ui.diceLog) - 1)
	}
}

func (ui *tviewUI) renderDiceList() {
	ui.diceRenderLock = true
	defer func() { ui.diceRenderLock = false }()

	cur := 0
	if ui.dice != nil {
		cur = ui.dice.GetCurrentItem()
		ui.dice.Clear()
	}

	if len(ui.diceLog) == 0 {
		ui.dice.AddItem("(nessun tiro) premi 'a' per lanciare", "", 0, nil)
		ui.dice.SetCurrentItem(0)
		return
	}

	for i, row := range ui.diceLog {
		ui.dice.AddItem(fmt.Sprintf("%d) %s => %s", i+1, row.Expression, row.Output), "", 0, nil)
	}
	if cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	if cur < 0 {
		cur = 0
	}
	ui.dice.SetCurrentItem(cur)
}

func (ui *tviewUI) deleteSelectedDiceResult() {
	if len(ui.diceLog) == 0 {
		ui.message = "Nessun tiro da eliminare."
		ui.refreshStatus()
		return
	}
	cur := ui.dice.GetCurrentItem()
	if cur < 0 || cur >= len(ui.diceLog) {
		cur = len(ui.diceLog) - 1
	}
	ui.diceLog = append(ui.diceLog[:cur], ui.diceLog[cur+1:]...)
	ui.renderDiceList()
	if len(ui.diceLog) == 0 {
		ui.message = "Storico dadi svuotato."
	} else {
		if cur >= len(ui.diceLog) {
			cur = len(ui.diceLog) - 1
		}
		ui.dice.SetCurrentItem(cur)
		ui.message = "Tiro eliminato."
	}
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) clearDiceResults() {
	if len(ui.diceLog) == 0 {
		ui.message = "Storico dadi già vuoto."
		ui.refreshStatus()
		return
	}
	ui.diceLog = nil
	ui.renderDiceList()
	ui.message = "Storico dadi svuotato."
	ui.refreshDetail()
	ui.refreshStatus()
}

func expandDiceRollInput(input string) ([]string, error) {
	return diceroll.ExpandRollInput(input)
}

func rollDiceExpression(expr string) (int, string, error) {
	return diceroll.RollExpression(expr)
}
