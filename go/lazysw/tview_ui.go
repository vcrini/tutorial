package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/vcrini/diceroll"
)

const helpText = " [black:gold]q[-:-] esci  [black:gold]?[-:-] help  [black:gold]f[-:-] fullscreen  [black:gold]tab/shift+tab[-:-] focus  [black:gold]0/1/2/3[-:-] pannelli  [black:gold][[ / ]][-:-] Mostri/Ambienti/Equip./Carte/Classe  [black:gold]a[-:-] roll dadi  [black:gold]b[-:-] treasure equip  [black:gold]i/I/S[-:-] init one/all/sort (Encounter)  [black:gold]c/x/C/o[-:-] condizioni encounter  [black:gold]/[-:-] ricerca raw  [black:gold]PgUp/PgDn[-:-] scroll dettagli  [black:gold]u/t/g[-:-] filtri pannello  [black:gold]v[-:-] reset filtri "

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

type tviewUI struct {
	app    *tview.Application
	pages  *tview.Pages
	status *tview.TextView

	dice           *tview.List
	diceLog        []DiceResult
	diceRenderLock bool

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
	detailBottom   *tview.Pages
	detail         *tview.TextView
	detailTreasure *tview.TextView

	monstersPanel     *tview.Flex
	environmentsPanel *tview.Flex
	equipmentPanel    *tview.Flex
	cardsPanel        *tview.Flex
	classesPanel      *tview.Flex
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
	encounter        []EncounterEntry
	filtered         []int
	filteredEnv      []int
	filteredEq       []int
	filteredCards    []int
	filteredClasses  []int
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

	encounterShowConditionEffects bool
}

func runTViewUI() error {
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
	return ui.app.SetRoot(ui.pages, true).EnableMouse(true).Run()
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
		encounter:        encounter,
		message:          "Pronto.",
		catalogMode:      "mostri",
		activeBottomPane: "details",
	}
	ui.build()
	return ui, nil
}

func (ui *tviewUI) build() {
	ui.dice = tview.NewList().ShowSecondaryText(false)
	ui.dice.SetBorder(true).SetTitle(" [0]-Dadi ")
	ui.dice.SetChangedFunc(func(int, string, string, rune) {
		if ui.diceRenderLock {
			return
		}
		ui.refreshDetail()
	})

	ui.pngList = tview.NewList().ShowSecondaryText(false)
	ui.pngList.SetBorder(true).SetTitle(" [1]-PNG ")
	ui.pngList.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(ui.pngs) {
			ui.selected = index
			ui.persistPNGs()
		}
		ui.refreshDetail()
	})

	ui.encList = tview.NewList().ShowSecondaryText(false)
	ui.encList.SetBorder(true).SetTitle(" [2]-Encounter ")
	ui.encList.SetChangedFunc(func(int, string, string, rune) {
		ui.refreshDetail()
	})

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

	ui.rankDrop = tview.NewDropDown().SetLabel(" Taglia ")
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

	ui.monList = tview.NewList().ShowSecondaryText(false)
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

	ui.envList = tview.NewList().ShowSecondaryText(false)
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

	ui.eqList = tview.NewList().ShowSecondaryText(false)
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

	ui.cardList = tview.NewList().ShowSecondaryText(false)
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

	ui.classList = tview.NewList().ShowSecondaryText(false)
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
		AddPage("classe", ui.classesPanel, true, false)
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
		ui.detailTreasure,
		ui.detail,
	}
	ui.focusIdx = focusMonList
	ui.app.SetFocus(ui.monList)
	ui.app.SetInputCapture(ui.handleGlobalKeys)
	ui.renderDiceList()
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
	case tcell.KeyPgUp:
		if focus == ui.detail || focus == ui.detailTreasure || focus == ui.dice || focus == ui.pngList || focus == ui.encList || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.envList || focus == ui.envSearch || focus == ui.envTypeDrop || focus == ui.envRankDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop || focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop {
			ui.scrollDetailByPage(-1)
			return nil
		}
	case tcell.KeyPgDn:
		if focus == ui.detail || focus == ui.detailTreasure || focus == ui.dice || focus == ui.pngList || focus == ui.encList || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.envList || focus == ui.envSearch || focus == ui.envTypeDrop || focus == ui.envRankDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop || focus == ui.classList || focus == ui.classSearch || focus == ui.classNameDrop || focus == ui.classSubDrop {
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
	case 'q':
		ui.app.Stop()
		return nil
	case '1':
		ui.focusPanel(focusPNG)
		return nil
	case '2':
		ui.focusPanel(focusEncounter)
		return nil
	case '3':
		ui.focusPanel(ui.activeCatalogListFocus())
		return nil
	case '0':
		ui.focusPanel(focusDice)
		return nil
	case '[':
		if focus == ui.encList {
			ui.adjustEncounterConditionRounds(-1)
			return nil
		}
		ui.switchCatalog(-1)
		return nil
	case ']':
		if focus == ui.encList {
			ui.adjustEncounterConditionRounds(1)
			return nil
		}
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
		if focus == ui.encList {
			ui.openEncounterConditionModal()
			return nil
		}
		ui.openCreatePNGModal()
		return nil
	case 'x':
		if focus == ui.encList {
			ui.openEncounterConditionRemoveModal()
			return nil
		}
		ui.openDeletePNGConfirm()
		return nil
	case 'C':
		if focus == ui.encList {
			ui.clearEncounterConditions()
			return nil
		}
	case 'm':
		if focus == ui.pngList {
			ui.openRenamePNGModal()
			return nil
		}
	case 'a':
		if focus == ui.dice {
			ui.openDiceRollInput()
			return nil
		}
		if focus == ui.pngList {
			ui.addSelectedPNGToEncounter()
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
		if focus == ui.dice {
			ui.openDiceReRollInput()
			return nil
		}
	case 'b':
		if ui.catalogMode == "equipaggiamento" && (focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.detail || focus == ui.detailTreasure) {
			ui.openEquipmentTreasureInput()
			return nil
		}
	case 'u':
		if ui.catalogMode == "mostri" {
			ui.focusPanel(focusMonSearch)
		} else if ui.catalogMode == "ambienti" {
			ui.focusPanel(focusEnvSearch)
		} else if ui.catalogMode == "equipaggiamento" {
			ui.focusPanel(focusEqSearch)
		} else if ui.catalogMode == "carte" {
			ui.focusPanel(focusCardSearch)
		} else {
			ui.focusPanel(focusClassSearch)
		}
		return nil
	case 't':
		if ui.catalogMode == "mostri" {
			ui.focusPanel(focusMonRole)
		} else if ui.catalogMode == "ambienti" {
			ui.focusPanel(focusEnvType)
		} else if ui.catalogMode == "equipaggiamento" {
			ui.focusPanel(focusEqItemType)
		} else if ui.catalogMode == "carte" {
			ui.focusPanel(focusCardClass)
		} else {
			ui.focusPanel(focusClassName)
		}
		return nil
	case 'g':
		if ui.catalogMode == "mostri" {
			ui.focusPanel(focusMonRank)
		} else if ui.catalogMode == "ambienti" {
			ui.focusPanel(focusEnvRank)
		} else if ui.catalogMode == "equipaggiamento" {
			ui.focusPanel(focusEqRank)
		} else if ui.catalogMode == "carte" {
			ui.focusPanel(focusCardType)
		} else {
			ui.focusPanel(focusClassSubclass)
		}
		return nil
	case 'v':
		if ui.catalogMode == "mostri" {
			ui.resetMonsterFilters()
		} else if ui.catalogMode == "ambienti" {
			ui.resetEnvironmentFilters()
		} else if ui.catalogMode == "equipaggiamento" {
			ui.resetEquipmentFilters()
		} else if ui.catalogMode == "carte" {
			ui.resetCardFilters()
		} else {
			ui.resetClassFilters()
		}
		return nil
	case 'd':
		if focus == ui.dice {
			ui.deleteSelectedDiceResult()
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
	case 'h':
		if focus == ui.encList {
			ui.adjustEncounterWounds(1)
			return nil
		}
	case 'l':
		if focus == ui.encList {
			ui.adjustEncounterWounds(-1)
			return nil
		}
	case 'j':
		if focus == ui.encList {
			ui.adjustEncounterWounds(1)
			return nil
		}
	case 'k':
		if focus == ui.encList {
			ui.adjustEncounterWounds(-1)
			return nil
		}
	case 'i':
		if focus == ui.encList {
			ui.rollEncounterInitiativeSelected()
			return nil
		}
	case 'I':
		if focus == ui.encList {
			ui.rollEncounterInitiativeAll()
			return nil
		}
	case 'S':
		if focus == ui.encList {
			ui.sortEncounterByInitiative()
			return nil
		}
	case 'o':
		if focus == ui.encList {
			ui.encounterShowConditionEffects = !ui.encounterShowConditionEffects
			if ui.encounterShowConditionEffects {
				ui.message = "Dettagli effetti condizioni: ON."
			} else {
				ui.message = "Dettagli effetti condizioni: OFF."
			}
			ui.refreshDetail()
			ui.refreshStatus()
			return nil
		}
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
	default:
		return "Mostri"
	}
}

func (ui *tviewUI) refreshCatalogTitles() {
	order := []string{"mostri", "ambienti", "equipaggiamento", "carte", "classe"}
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
		}
	}
}

func (ui *tviewUI) switchCatalog(delta int) {
	if delta == 0 {
		return
	}
	order := []string{"mostri", "ambienti", "equipaggiamento", "carte", "classe"}
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
	if next == "ambienti" {
		ui.message = "Catalogo: Ambienti"
	} else if next == "equipaggiamento" {
		ui.message = "Catalogo: Equipaggiamento"
	} else if next == "carte" {
		ui.message = "Catalogo: Carte"
	} else if next == "classe" {
		ui.message = "Catalogo: Classe"
	} else {
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
	for i, p := range ui.pngs {
		prefix := "  "
		if i == ui.selected {
			prefix = "* "
		}
		ui.pngList.AddItem(fmt.Sprintf("%s%s", prefix, p.Name), "", 0, nil)
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
		if ui.rankFilter != "" && ui.rankFilter != "Tutti" && strconv.Itoa(m.Size) != ui.rankFilter {
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
		wc := ""
		if m.WildCard {
			wc = " WC"
		}
		ui.monList.AddItem(fmt.Sprintf("%s [S%d%s] Ferite:%d", m.Name, m.Size, wc, monsterWoundsCap(m)), "", 0, nil)
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
		base := encounterWoundsCap(e)
		label := ui.encounterLabelAt(i)
		if badge := encounterConditionsBadge(e); badge != "" {
			label = badge + " " + label
		}
		remaining := base - e.Wounds
		if remaining < 0 {
			remaining = 0
		}
		initLabel := "--"
		if e.HasInit {
			initLabel = e.InitiativeCard
		}
		ui.encList.AddItem(fmt.Sprintf("%s [Ini %s | Ferite %d/%d]", label, initLabel, remaining, base), "", 0, nil)
	}
	if current >= len(ui.encounter) {
		current = len(ui.encounter) - 1
	}
	if current < 0 {
		current = 0
	}
	ui.encList.SetCurrentItem(current)
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
		base := encounterWoundsCap(e)
		remaining := base - e.Wounds
		if remaining < 0 {
			remaining = 0
		}
		initLabel := "--"
		if e.HasInit {
			initLabel = e.InitiativeCard
		}
		extra := fmt.Sprintf("Iniziativa: %s | Stato: %d/%d ferite residue (%s)", initLabel, remaining, base, encounterStateLabel(e))
		if cond := encounterConditionsLong(e); cond != "" {
			extra += " | Condizioni: " + cond
			if ui.encounterShowConditionEffects {
				if effects := encounterConditionEffectsLong(e); effects != "" {
					extra += "\nEffetti condizioni:\n" + effects
				}
			}
		}
		ui.detailRaw = ui.buildMonsterDetails(e.Monster, ui.encounterLabelAt(idx), extra)
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
	b.WriteString(p.Name)
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
		b.WriteString(fmt.Sprintf("\nLivello: %d | Rango: %d", p.Level, rank))
		b.WriteString(fmt.Sprintf("\nBonus Competenza (livello): +%d", compBonus))
		b.WriteString(fmt.Sprintf("\nEsperienze aggiuntive (livello): +%d", expBonus))
	}
	if def := ui.findClassDefinition(p.Class, p.Subclass); def != nil {
		if def.Evasion > 0 {
			b.WriteString(fmt.Sprintf("\nEvasione iniziale: %d", def.Evasion))
		}
		if def.HP > 0 {
			b.WriteString(fmt.Sprintf("\nPF iniziali: %d", def.HP))
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
	focusLabel := "PNG"
	switch ui.app.GetFocus() {
	case ui.dice:
		focusLabel = "Dadi"
	case ui.encList:
		focusLabel = "Encounter"
	case ui.search:
		focusLabel = "Nome Mostri"
	case ui.roleDrop:
		focusLabel = "Ruolo Mostri"
	case ui.rankDrop:
		focusLabel = "Taglia Mostri"
	case ui.monList:
		focusLabel = "Mostri"
	case ui.envSearch:
		focusLabel = "Nome Ambienti"
	case ui.envTypeDrop:
		focusLabel = "Tipo Ambienti"
	case ui.envRankDrop:
		focusLabel = "Rango Ambienti"
	case ui.envList:
		focusLabel = "Ambienti"
	case ui.eqSearch:
		focusLabel = "Nome Equip."
	case ui.eqTypeDrop:
		focusLabel = "Categoria Equip."
	case ui.eqItemTypeDrop:
		focusLabel = "Tipo Equip."
	case ui.eqRankDrop:
		focusLabel = "Rango Equip."
	case ui.eqList:
		focusLabel = "Equipaggiamento"
	case ui.cardSearch:
		focusLabel = "Nome Carte"
	case ui.cardClassDrop:
		focusLabel = "Classe Carte"
	case ui.cardTypeDrop:
		focusLabel = "Tipo Carte"
	case ui.cardList:
		focusLabel = "Carte"
	case ui.classSearch:
		focusLabel = "Nome Classe"
	case ui.classNameDrop:
		focusLabel = "Classe"
	case ui.classSubDrop:
		focusLabel = "Sottoclasse"
	case ui.classList:
		focusLabel = "Classi"
	case ui.detailTreasure:
		focusLabel = "Treasure"
	case ui.detail:
		focusLabel = "Dettagli"
	}
	msg := ui.message
	if msg == "" {
		msg = "Pronto."
	}
	catalogLabel := "Mostri"
	if ui.catalogMode == "ambienti" {
		catalogLabel = "Ambienti"
	} else if ui.catalogMode == "equipaggiamento" {
		catalogLabel = "Equipaggiamento"
	} else if ui.catalogMode == "carte" {
		catalogLabel = "Carte"
	} else if ui.catalogMode == "classe" {
		catalogLabel = "Classe"
	}
	ui.status.SetText(fmt.Sprintf("focus:[black:gold] %s [-:-] | catalogo:[black:gold] %s [-:-] | %s [black:gold]msg[-:-] %s", focusLabel, catalogLabel, helpText, msg))
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
	sizeSet := map[int]struct{}{}

	for _, m := range ui.monsters {
		role := strings.TrimSpace(m.Role)
		if role != "" {
			roleSet[role] = struct{}{}
		}
		if m.Size != 0 {
			sizeSet[m.Size] = struct{}{}
		}
	}

	roles := make([]string, 0, len(roleSet)+1)
	roles = append(roles, "Tutti")
	for role := range roleSet {
		roles = append(roles, role)
	}
	sort.Strings(roles[1:])

	sizesInt := make([]int, 0, len(sizeSet))
	for size := range sizeSet {
		sizesInt = append(sizesInt, size)
	}
	sort.Ints(sizesInt)

	ranks := make([]string, 0, len(sizesInt)+1)
	ranks = append(ranks, "Tutti")
	for _, size := range sizesInt {
		ranks = append(ranks, strconv.Itoa(size))
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

func (ui *tviewUI) buildEnvironmentDetails(e Environment) string {
	var b strings.Builder
	b.WriteString(e.Name + "\n")
	b.WriteString(fmt.Sprintf("Tipo: %s | Rango: %d\n", e.Kind, e.Rank))
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
	b.WriteString(fmt.Sprintf("Categoria: %s | Tipo: %s | Rango: %d", it.Category, it.Type, it.Rank))
	if hasValue(it.Levels) {
		b.WriteString(fmt.Sprintf(" | Livelli: %s", it.Levels))
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
	b.WriteString(fmt.Sprintf("Classe: %s | Tipo: %s\n", strings.TrimSpace(c.Class), strings.TrimSpace(c.Type)))
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
	b.WriteString(fmt.Sprintf("%s - %s\n", c.Name, c.Subclass))
	b.WriteString(fmt.Sprintf("Rango: %d\n", c.Rank))
	if strings.TrimSpace(c.Domains) != "" {
		b.WriteString("Domini: " + strings.TrimSpace(c.Domains) + "\n")
	}
	if c.Evasion > 0 {
		b.WriteString(fmt.Sprintf("Evasione iniziale: %d\n", c.Evasion))
	}
	if c.HP > 0 {
		b.WriteString(fmt.Sprintf("Punti Ferita iniziali: %d\n", c.HP))
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
	wc := "No"
	if m.WildCard {
		wc = "Si"
	}
	pace := strings.TrimSpace(m.Pace)
	if pace == "" {
		pace = "-"
	}
	b.WriteString(fmt.Sprintf("Ruolo: %s | Taglia: %d | Wild Card: %s | Passo: %s\n", m.Role, m.Size, wc, pace))
	if extraLine != "" {
		b.WriteString(extraLine + "\n")
	}
	parry := strings.TrimSpace(m.Parry)
	if parry == "" {
		parry = "-"
	}
	toughness := strings.TrimSpace(m.Toughness)
	if toughness == "" {
		toughness = "-"
	}
	b.WriteString(fmt.Sprintf("Parata: %s | Robustezza: %s | Ferite max: %d\n", parry, toughness, monsterWoundsCap(m)))
	if m.Attack.Name != "" {
		bonus := strings.TrimSpace(m.Attack.Bonus)
		bonus = strings.ReplaceAll(bonus, "−", "-")
		bonus = strings.ReplaceAll(bonus, "–", "-")
		if bonus != "" && !strings.HasPrefix(bonus, "+") && !strings.HasPrefix(bonus, "-") {
			bonus = "+" + bonus
		}
		if bonus != "" {
			b.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s (%s)\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType, bonus))
		} else {
			b.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType))
		}
	}
	if strings.TrimSpace(m.MotivationsTactics) != "" {
		b.WriteString("\nMotivazioni/Tattiche:\n" + strings.TrimSpace(m.MotivationsTactics) + "\n")
	}
	if len(m.Skills) > 0 {
		b.WriteString("\nAbilita:\n")
		for _, sk := range m.Skills {
			sk = strings.TrimSpace(sk)
			if sk == "" {
				continue
			}
			b.WriteString("- " + sk + "\n")
		}
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

func (ui *tviewUI) encounterLabelAt(idx int) string {
	if idx < 0 || idx >= len(ui.encounter) {
		return ""
	}
	name := ui.encounter[idx].Monster.Name
	seen := 0
	for i := 0; i <= idx; i++ {
		if ui.encounter[i].Monster.Name == name {
			seen++
		}
	}
	return fmt.Sprintf("%s #%d", name, seen)
}

func monsterWoundsCap(m Monster) int {
	if m.WoundsMax > 0 {
		return m.WoundsMax
	}
	if m.PF > 0 {
		return m.PF
	}
	return 3
}

func encounterWoundsCap(e EncounterEntry) int {
	if e.WoundsMax > 0 {
		return e.WoundsMax
	}
	if e.BasePF > 0 {
		return e.BasePF
	}
	return monsterWoundsCap(e.Monster)
}

func encounterStateLabel(e EncounterEntry) string {
	max := encounterWoundsCap(e)
	if e.Wounds >= max {
		return "KO/Incapacitato"
	}
	if e.Wounds > 0 {
		return "Ferito"
	}
	return "Operativo"
}

func (ui *tviewUI) openCreatePNGModal() {
	input := tview.NewInputField().SetLabel(" Nome PNG ").SetFieldWidth(24)
	input.SetText(uniqueRandomPNGName(ui.pngs))
	input.SetBorder(true).SetTitle("Crea PNG")
	returnFocus := ui.app.GetFocus()

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 42, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "create_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		name := strings.TrimSpace(input.GetText())
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
		ui.pngs = append(ui.pngs, PNG{Name: name})
		ui.selected = len(ui.pngs) - 1
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(0)
		ui.message = fmt.Sprintf("Creato PNG %s.", name)
		ui.refreshAll()
	})
}

func (ui *tviewUI) openRenamePNGModal() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}

	currentName := ui.pngs[ui.selected].Name
	input := tview.NewInputField().SetLabel(" Nuovo nome ").SetFieldWidth(28)
	input.SetText(currentName)
	input.SetBorder(true).SetTitle("Rinomina PNG")
	returnFocus := ui.app.GetFocus()

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 48, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "rename_png"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(input)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			ui.closeModal()
			ui.app.SetFocus(returnFocus)
			return
		}
		newName := strings.TrimSpace(input.GetText())
		if newName == "" {
			ui.message = "Nome PNG non valido."
			ui.refreshStatus()
			return
		}
		for i, p := range ui.pngs {
			if i == ui.selected {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(p.Name), newName) {
				ui.message = "Nome già esistente."
				ui.refreshStatus()
				return
			}
		}
		ui.pngs[ui.selected].Name = newName
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(focusPNG)
		ui.message = fmt.Sprintf("PNG rinominato in %s.", newName)
		ui.refreshAll()
	})
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
					advanceToGenerate()
				case tcell.KeyBacktab:
					form.SetFocus(form.GetFormItemCount() + form.GetButtonIndex("Annulla"))
				}
			})
		}
	}

	form.AddButton("Genera", func() {
		baseName := uniqueRandomPNGName(ui.pngs)
		preset := classPresetFor(c.Name)
		inv := buildSuggestedInventory(preset)
		look := buildSuggestedLook(preset)
		png := PNG{
			Name:        fmt.Sprintf("%s (%s | %s L%d)", baseName, c.Subclass, c.Name, selectedLevel),
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
		AddItem(info, 2, 0, false).
		AddItem(form, 0, 1, true)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(container, 66, 0, true).
			AddItem(nil, 0, 1, false), 11, 0, true).
		AddItem(nil, 0, 1, false)

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

func (ui *tviewUI) openDeletePNGConfirm() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	name := ui.pngs[ui.selected].Name
	ui.openConfirmModal("Conferma", fmt.Sprintf("Eliminare PNG '%s'?", name), func() {
		ui.deleteSelectedPNG()
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

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(text, 56, 0, true).
			AddItem(nil, 0, 1, false), 8, 0, true).
		AddItem(nil, 0, 1, false)

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
	if focus == ui.detail {
		input.SetText(ui.detailQuery)
	}
	if focus == ui.detailTreasure {
		input.SetText(ui.detailQuery)
	}

	returnFocus := focus
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 48, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

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
	step := h / 2
	if step < 1 {
		step = 1
	}
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

func (ui *tviewUI) addSelectedMonsterToEncounter() {
	idx := ui.currentMonsterIndex()
	if idx < 0 {
		ui.message = "Nessun mostro selezionato."
		ui.refreshStatus()
		return
	}
	mon := ui.monsters[idx]
	woundsMax := monsterWoundsCap(mon)
	ui.encounter = append(ui.encounter, EncounterEntry{
		Monster:    mon,
		WoundsMax:  woundsMax,
		BasePF:     woundsMax,
		Stress:     0,
		BaseStress: 0,
	})
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Aggiunto %s a encounter.", mon.Name)
	ui.refreshEncounter()
	ui.refreshStatus()
}

func (ui *tviewUI) addSelectedPNGToEncounter() {
	if ui.selected < 0 || ui.selected >= len(ui.pngs) {
		ui.message = "Nessun PNG selezionato."
		ui.refreshStatus()
		return
	}
	p := ui.pngs[ui.selected]
	name := strings.TrimSpace(p.Name)
	if name == "" {
		ui.message = "Nome PNG non valido."
		ui.refreshStatus()
		return
	}

	woundsMax := 3
	if def := ui.findClassDefinition(p.Class, p.Subclass); def != nil && def.HP > 0 {
		woundsMax = def.HP
	}
	if p.Level > 1 {
		woundsMax += (p.Level - 1) / 2
	}
	if woundsMax < 1 {
		woundsMax = 1
	}

	mon := Monster{
		Name:               "PNG: " + name,
		Role:               "PNG",
		WildCard:           true,
		Size:               0,
		Pace:               "6",
		Parry:              "-",
		Toughness:          "-",
		WoundsMax:          woundsMax,
		Description:        strings.TrimSpace(p.Description),
		MotivationsTactics: strings.TrimSpace(p.Traits),
	}
	if strings.TrimSpace(p.Primary) != "" {
		mon.Skills = append(mon.Skills, "Primario: "+strings.TrimSpace(p.Primary))
	}
	if strings.TrimSpace(p.Secondary) != "" {
		mon.Skills = append(mon.Skills, "Secondario: "+strings.TrimSpace(p.Secondary))
	}
	if strings.TrimSpace(p.Armor) != "" {
		mon.Skills = append(mon.Skills, "Armatura: "+strings.TrimSpace(p.Armor))
	}

	ui.encounter = append(ui.encounter, EncounterEntry{
		Monster:    mon,
		WoundsMax:  woundsMax,
		BasePF:     woundsMax,
		Stress:     0,
		BaseStress: 0,
	})
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Aggiunto PNG %s a encounter.", name)
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
	Size          int
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
	sizeOptions := make([]string, 0, len(ui.rankOpts))
	defaultSizeIdx := 0
	currentMonsterIdx := ui.currentMonsterIndex()
	currentSize := 0
	if currentMonsterIdx >= 0 && currentMonsterIdx < len(ui.monsters) {
		currentSize = ui.monsters[currentMonsterIdx].Size
	}
	for _, opt := range ui.rankOpts {
		if strings.EqualFold(strings.TrimSpace(opt), "Tutti") {
			continue
		}
		sizeOptions = append(sizeOptions, opt)
		if currentSize != 0 && opt == strconv.Itoa(currentSize) {
			defaultSizeIdx = len(sizeOptions) - 1
		}
	}
	if len(sizeOptions) == 0 {
		ui.message = "Nessuna taglia disponibile nei Mostri."
		ui.refreshStatus()
		return
	}

	defaultPG := len(ui.pngs)
	if defaultPG < 1 {
		defaultPG = 1
	}
	selectedSize, _ := strconv.Atoi(sizeOptions[defaultSizeIdx])
	if selectedSize == 0 {
		selectedSize = 1
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
	form.AddDropDown("Taglia gruppo", sizeOptions, defaultSizeIdx, func(option string, _ int) {
		if option == "" {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(option)); err == nil && v != 0 {
			selectedSize = v
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
		summary := ui.generateRandomEncounterFromMonsters(selectedSize, selectedPG, mod)
		if summary.AddedEntries == 0 {
			ui.message = fmt.Sprintf("Nessun mostro generato (S%d, %d PG).", selectedSize, selectedPG)
			ui.refreshStatus()
			return
		}
		ui.closeModal()
		ui.focusPanel(focusEncounter)
		ui.message = fmt.Sprintf("Encounter random S%d: +%d nemici (%d PB spesi, %d residui).", selectedSize, summary.AddedEntries, summary.Spent, summary.Remaining)
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

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(container, 80, 0, true).
			AddItem(nil, 0, 1, false), 13, 0, true).
		AddItem(nil, 0, 1, false)

	ui.modalVisible = true
	ui.modalName = "monster_random_encounter"
	ui.pages.AddAndSwitchToPage(ui.modalName, modal, true)
	ui.app.SetFocus(form.GetFormItem(0))
	ready = true
}

func (ui *tviewUI) generateRandomEncounterFromMonsters(size int, pgCount int, budgetMod int) generatedEncounterSummary {
	summary := generatedEncounterSummary{
		Size:          size,
		PGCount:       pgCount,
		BaseBudget:    3*pgCount + 2,
		BudgetMod:     budgetMod,
		ByMonsterName: map[string]int{},
	}
	if pgCount < 1 {
		return summary
	}
	finalBudget := summary.BaseBudget + budgetMod
	if finalBudget < 1 {
		finalBudget = 1
	}
	summary.FinalBudget = finalBudget

	type candidate struct {
		mon  Monster
		cost int
	}
	candidates := make([]candidate, 0, len(ui.monsters))
	for _, m := range ui.monsters {
		if m.Size != size {
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
		for i := 0; i < qty; i++ {
			woundsMax := monsterWoundsCap(pick.mon)
			ui.encounter = append(ui.encounter, EncounterEntry{
				Monster:    pick.mon,
				WoundsMax:  woundsMax,
				BasePF:     woundsMax,
				Stress:     0,
				BaseStress: 0,
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
	b.WriteString(fmt.Sprintf("Taglia gruppo: %d | PG: %d\n", s.Size, s.PGCount))
	b.WriteString(fmt.Sprintf("Punti Battaglia: %d %+d = %d\n", s.BaseBudget, s.BudgetMod, s.FinalBudget))
	b.WriteString(fmt.Sprintf("Spesi: %d | Residui: %d\n", s.Spent, s.Remaining))
	b.WriteString(fmt.Sprintf("Nemici aggiunti: %d (gruppi estratti: %d)\n", s.AddedEntries, s.AddedGroups))
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
		b.WriteString(fmt.Sprintf("- %s x%d\n", name, s.ByMonsterName[name]))
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
	name := ui.encounter[idx].Monster.Name
	ui.encounter = append(ui.encounter[:idx], ui.encounter[idx+1:]...)
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Rimosso %s da encounter.", name)
	ui.refreshAll()
}

func (ui *tviewUI) adjustEncounterWounds(delta int) {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	e := &ui.encounter[idx]
	prevWounds := e.Wounds
	e.Wounds += delta
	if e.Wounds < 0 {
		e.Wounds = 0
	}
	base := encounterWoundsCap(*e)
	if base > 0 && e.Wounds > base {
		e.Wounds = base
	}
	appliedShaken := applyShakenOnWoundReduction(prevWounds, e)
	ui.persistEncounter()
	if appliedShaken {
		ui.message = fmt.Sprintf("Ferite %s: %d/%d | Stato aggiunto: Scosso", e.Monster.Name, e.Wounds, base)
	} else {
		ui.message = fmt.Sprintf("Ferite %s: %d/%d", e.Monster.Name, e.Wounds, base)
	}
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) rollEncounterInitiativeSelected() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	e := &ui.encounter[idx]
	card, ok := ui.drawInitiativeCard(idx)
	if !ok {
		ui.message = "Mazzo esaurito per iniziative uniche."
		ui.refreshStatus()
		return
	}
	e.InitiativeCard = card
	e.HasInit = true
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Iniziativa %s: %s", e.Monster.Name, e.InitiativeCard)
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) rollEncounterInitiativeAll() {
	if len(ui.encounter) == 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	deck := buildInitiativeDeck()
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	for i := range ui.encounter {
		if i < len(deck) {
			ui.encounter[i].InitiativeCard = deck[i]
			ui.encounter[i].HasInit = true
			continue
		}
		ui.encounter[i].InitiativeCard = ""
		ui.encounter[i].HasInit = false
	}
	ui.persistEncounter()
	if len(ui.encounter) > len(deck) {
		ui.message = "Iniziativa tirata per i primi 52; mazzo esaurito."
	} else {
		ui.message = "Iniziativa tirata per tutti."
	}
	ui.refreshEncounter()
	ui.refreshDetail()
	ui.refreshStatus()
}

func (ui *tviewUI) sortEncounterByInitiative() {
	if len(ui.encounter) == 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	currentLabel := ui.encounterLabelAt(ui.currentEncounterIndex())
	sort.SliceStable(ui.encounter, func(i, j int) bool {
		ai, aj := ui.encounter[i], ui.encounter[j]
		if ai.HasInit != aj.HasInit {
			return ai.HasInit
		}
		if ai.HasInit && aj.HasInit {
			cmp := compareInitiativeCards(ai.InitiativeCard, aj.InitiativeCard)
			if cmp != 0 {
				return cmp < 0
			}
		}
		return ai.Monster.Name < aj.Monster.Name
	})
	ui.persistEncounter()
	ui.refreshEncounter()
	if currentLabel != "" {
		for i := range ui.encounter {
			if ui.encounterLabelAt(i) == currentLabel {
				ui.encList.SetCurrentItem(i)
				break
			}
		}
	}
	ui.refreshDetail()
	ui.message = "Encounter ordinato per iniziativa."
	ui.refreshStatus()
}

func (ui *tviewUI) openEncounterConditionModal() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	temp := cloneStringIntMap(ui.encounter[idx].Conditions)
	if temp == nil {
		temp = map[string]int{}
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Encounter Conditions (Space=toggle, Enter=apply, Esc=cancel) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)

	render := func() {
		cur := list.GetCurrentItem()
		list.Clear()
		for _, d := range encounterConditionDefs {
			r := temp[d.Code]
			mark := "[ ]"
			if r > 0 {
				mark = fmt.Sprintf("[x%d]", r)
			}
			list.AddItem(fmt.Sprintf("%s %s (%s)", mark, d.Name, d.Code), "", 0, nil)
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

	toggle := func() {
		cur := list.GetCurrentItem()
		if cur < 0 || cur >= len(encounterConditionDefs) {
			return
		}
		code := encounterConditionDefs[cur].Code
		if temp[code] > 0 {
			delete(temp, code)
		} else {
			temp[code] = 1
		}
		render()
	}

	closeModal := func(apply bool) {
		ui.pages.RemovePage("encounter-conditions")
		ui.app.SetFocus(ui.encList)
		if !apply {
			return
		}
		ui.encounter[idx].Conditions = cloneStringIntMap(temp)
		ui.persistEncounter()
		ui.refreshEncounter()
		ui.encList.SetCurrentItem(idx)
		ui.refreshDetail()
		ui.message = fmt.Sprintf("Condizioni aggiornate su %s.", ui.encounterLabelAt(idx))
		ui.refreshStatus()
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
			AddItem(list, 20, 0, true).
			AddItem(nil, 0, 1, false), 74, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("encounter-conditions", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) clearEncounterConditions() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	if len(ui.encounter[idx].Conditions) == 0 {
		ui.message = "Nessuna condizione da rimuovere."
		ui.refreshStatus()
		return
	}
	ui.encounter[idx].Conditions = nil
	ui.persistEncounter()
	ui.refreshEncounter()
	ui.encList.SetCurrentItem(idx)
	ui.refreshDetail()
	ui.message = fmt.Sprintf("Condizioni rimosse da %s.", ui.encounterLabelAt(idx))
	ui.refreshStatus()
}

func (ui *tviewUI) removeEncounterConditionByCode(index int, code string) bool {
	if index < 0 || index >= len(ui.encounter) {
		return false
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || len(ui.encounter[index].Conditions) == 0 {
		return false
	}
	if _, ok := ui.encounter[index].Conditions[code]; !ok {
		return false
	}
	delete(ui.encounter[index].Conditions, code)
	if len(ui.encounter[index].Conditions) == 0 {
		ui.encounter[index].Conditions = nil
	}
	return true
}

func (ui *tviewUI) openEncounterConditionRemoveModal() {
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	entry := ui.encounter[idx]
	if len(entry.Conditions) == 0 {
		ui.message = "Nessuna condizione da rimuovere."
		ui.refreshStatus()
		return
	}

	active := make([]encounterConditionDef, 0, len(entry.Conditions))
	for _, d := range encounterConditionDefs {
		if entry.Conditions[d.Code] > 0 {
			active = append(active, d)
		}
	}
	if len(active) == 0 {
		ui.message = "Nessuna condizione da rimuovere."
		ui.refreshStatus()
		return
	}

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Remove One Condition (Enter=remove, Esc=cancel) ")
	list.SetBorderColor(tcell.ColorGold)
	list.SetTitleColor(tcell.ColorGold)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorGold)
	list.ShowSecondaryText(false)
	for _, d := range active {
		rounds := entry.Conditions[d.Code]
		list.AddItem(fmt.Sprintf("%s (%s%d)", d.Name, d.Code, rounds), "", 0, nil)
	}

	closeModal := func() {
		ui.pages.RemovePage("encounter-condition-remove")
		ui.app.SetFocus(ui.encList)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeModal()
			return nil
		case tcell.KeyEnter:
			cur := list.GetCurrentItem()
			if cur < 0 || cur >= len(active) {
				closeModal()
				return nil
			}
			code := active[cur].Code
			if ui.removeEncounterConditionByCode(idx, code) {
				ui.persistEncounter()
				ui.refreshEncounter()
				ui.encList.SetCurrentItem(idx)
				ui.refreshDetail()
				ui.message = fmt.Sprintf("Rimossa condizione %s da %s.", conditionNameByCode(code), ui.encounterLabelAt(idx))
				ui.refreshStatus()
			}
			closeModal()
			return nil
		default:
			return event
		}
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 12, 0, true).
			AddItem(nil, 0, 1, false), 58, 0, true).
		AddItem(nil, 0, 1, false)
	ui.pages.AddPage("encounter-condition-remove", modal, true, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) adjustEncounterConditionRounds(delta int) {
	if delta == 0 {
		return
	}
	idx := ui.currentEncounterIndex()
	if idx < 0 {
		ui.message = "Encounter vuoto."
		ui.refreshStatus()
		return
	}
	if len(ui.encounter[idx].Conditions) == 0 {
		ui.message = "Nessuna condizione attiva."
		ui.refreshStatus()
		return
	}
	for code, r := range ui.encounter[idx].Conditions {
		n := r + delta
		if n <= 0 {
			delete(ui.encounter[idx].Conditions, code)
		} else {
			ui.encounter[idx].Conditions[code] = n
		}
	}
	if len(ui.encounter[idx].Conditions) == 0 {
		ui.encounter[idx].Conditions = nil
	}
	ui.persistEncounter()
	ui.refreshEncounter()
	ui.encList.SetCurrentItem(idx)
	ui.refreshDetail()
	if delta > 0 {
		ui.message = fmt.Sprintf("Round condizioni +1 su %s.", ui.encounterLabelAt(idx))
	} else {
		ui.message = fmt.Sprintf("Round condizioni -1 su %s.", ui.encounterLabelAt(idx))
	}
	ui.refreshStatus()
}

var initiativeRanks = []string{"A", "K", "Q", "J", "10", "9", "8", "7", "6", "5", "4", "3", "2"}
var initiativeSuits = []string{"♥", "♦", "♣", "♠"}

func buildInitiativeDeck() []string {
	deck := make([]string, 0, len(initiativeRanks)*len(initiativeSuits))
	for _, r := range initiativeRanks {
		for _, s := range initiativeSuits {
			deck = append(deck, r+s)
		}
	}
	return deck
}

func (ui *tviewUI) drawInitiativeCard(excludeIdx int) (string, bool) {
	used := make(map[string]struct{}, len(ui.encounter))
	for i, e := range ui.encounter {
		if i == excludeIdx {
			continue
		}
		if e.HasInit && strings.TrimSpace(e.InitiativeCard) != "" {
			used[e.InitiativeCard] = struct{}{}
		}
	}
	deck := buildInitiativeDeck()
	available := make([]string, 0, len(deck))
	for _, c := range deck {
		if _, ok := used[c]; !ok {
			available = append(available, c)
		}
	}
	if len(available) == 0 {
		return "", false
	}
	return available[rand.IntN(len(available))], true
}

func compareInitiativeCards(a, b string) int {
	ar, as, aok := parseInitiativeCard(a)
	br, bs, bok := parseInitiativeCard(b)
	if !aok && !bok {
		return strings.Compare(a, b)
	}
	if !aok {
		return 1
	}
	if !bok {
		return -1
	}
	if as != bs {
		return as - bs
	}
	return ar - br
}

func parseInitiativeCard(card string) (rankIdx int, suitIdx int, ok bool) {
	card = strings.TrimSpace(card)
	if card == "" {
		return 0, 0, false
	}
	for ri, r := range initiativeRanks {
		for si, s := range initiativeSuits {
			if card == r+s || card == s+r {
				return ri, si, true
			}
		}
	}
	return 0, 0, false
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

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(text, 0, 5, true).
			AddItem(nil, 0, 1, false), 0, 5, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddAndSwitchToPage("help", modal, true)
	ui.app.SetFocus(text)
}

func (ui *tviewUI) buildHelpContent(focus tview.Primitive) string {
	var b strings.Builder
	b.WriteString("LazySW - scorciatoie\n\n")

	panel := "Dettagli"
	var panelLines []string
	switch focus {
	case ui.dice:
		panel = "Dadi"
		panelLines = []string{
			"- a: nuovo tiro (anche multiplo, es. 1d20+3 x2)",
			"- Invio: rilancia il tiro selezionato",
			"- e: modifica + rilancia il tiro selezionato",
			"- d: elimina il tiro selezionato",
			"- c: svuota storico tiri",
		}
	case ui.pngList:
		panel = "PNG"
		panelLines = []string{
			"- c: crea PNG",
			"- m: rinomina PNG selezionato",
			"- x: elimina PNG selezionato",
			"- a: aggiungi PNG selezionato a Encounter",
		}
	case ui.encList:
		panel = "Encounter"
		panelLines = []string{
			"- d: rimuovi mostro selezionato",
			"- h / l: ferite +1 / -1 sul selezionato",
			"- j / k: ferite +1 / -1 sul selezionato",
			"- c: aggiungi/togli condizioni (multi select)",
			"- x: rimuovi una condizione dall'entry",
			"- C: rimuovi tutte le condizioni dall'entry",
			"- [ / ]: diminuisci/aumenta round condizioni",
			"- o: toggle effetti estesi condizioni nei dettagli",
			"- i: tira iniziativa sul selezionato",
			"- I: tira iniziativa per tutti",
			"- S: ordina encounter per iniziativa",
		}
	case ui.search, ui.roleDrop, ui.rankDrop, ui.monList:
		panel = "Mostri"
		panelLines = []string{
			"- a: aggiungi mostro selezionato a Encounter",
			"- n: genera Encounter random (Punti Battaglia)",
			"- u / t / g: focus filtro Nome / Ruolo / Taglia",
			"- v: reset filtri Mostri",
		}
	case ui.envSearch, ui.envTypeDrop, ui.envRankDrop, ui.envList:
		panel = "Ambienti"
		panelLines = []string{
			"- u / t / g: focus filtro Nome / Tipo / Rango",
			"- v: reset filtri Ambienti (Nome/Tipo/Rango)",
		}
	case ui.eqSearch, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqList:
		panel = "Equipaggiamento"
		panelLines = []string{
			"- u / t / g: focus filtro Nome / Tipo / Rango (TAB per Categoria)",
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
			"- u / t / g: focus filtro Nome / Classe / Tipo",
			"- v: reset filtri Carte (Nome/Classe/Tipo)",
		}
	case ui.classSearch, ui.classNameDrop, ui.classSubDrop, ui.classList:
		panel = "Classe"
		panelLines = []string{
			"- u / t / g: focus filtro Cerca / Classe / Sottoclasse",
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
	b.WriteString("- 0 / 1 / 2 / 3: focus Dadi / PNG / Encounter / Catalogo\n")
	b.WriteString("- i / I / S (su Encounter): iniziativa selezionato/tutti/ordina\n")
	b.WriteString("- [ / ]: alterna Catalogo (oppure round condizioni su Encounter)\n")
	b.WriteString("- /: ricerca rapida sul pannello corrente\n")
	b.WriteString("- f: fullscreen pannello corrente\n")
	b.WriteString("- PgUp / PgDn: scroll Dettagli\n")
	b.WriteString("\nEsc/?/q per chiudere")
	return b.String()
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

func (ui *tviewUI) persistPNGs() {
	_ = savePNGList(dataFile, ui.pngs, selectedPNGName(ui.pngs, ui.selected))
}

func (ui *tviewUI) persistEncounter() {
	entries := make([]struct {
		Name             string         `yaml:"name"`
		Wounds           int            `yaml:"wounds"`
		PF               int            `yaml:"pf"`
		InitiativeCard   string         `yaml:"initiative_card,omitempty"`
		LegacyInitiative int            `yaml:"initiative,omitempty"`
		HasInit          bool           `yaml:"has_initiative,omitempty"`
		Conditions       map[string]int `yaml:"conditions,omitempty"`
		Stress           int            `yaml:"stress,omitempty"`
		BaseStress       int            `yaml:"base_stress,omitempty"`
	}, 0, len(ui.encounter))
	for _, e := range ui.encounter {
		base := encounterWoundsCap(e)
		entries = append(entries, struct {
			Name             string         `yaml:"name"`
			Wounds           int            `yaml:"wounds"`
			PF               int            `yaml:"pf"`
			InitiativeCard   string         `yaml:"initiative_card,omitempty"`
			LegacyInitiative int            `yaml:"initiative,omitempty"`
			HasInit          bool           `yaml:"has_initiative,omitempty"`
			Conditions       map[string]int `yaml:"conditions,omitempty"`
			Stress           int            `yaml:"stress,omitempty"`
			BaseStress       int            `yaml:"base_stress,omitempty"`
		}{Name: e.Monster.Name, Wounds: e.Wounds, PF: base, InitiativeCard: e.InitiativeCard, HasInit: e.HasInit, Conditions: cloneStringIntMap(e.Conditions)})
	}
	_ = saveEncounter(encounterFile, entries)
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

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 72, 0, true).
			AddItem(nil, 0, 1, false), 13, 0, true).
		AddItem(nil, 0, 1, false)

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
	b.WriteString(fmt.Sprintf("Categoria: %s\n", category))
	b.WriteString(fmt.Sprintf("Tiro: %s => %s\n", dice, breakdown))
	b.WriteString(fmt.Sprintf("Valore Tiro: %02d\n", total))
	b.WriteString("\nRisultati:\n")
	if len(matches) == 0 {
		b.WriteString("- Nessun bottino con Tiro corrispondente.\n")
	} else {
		for _, it := range matches {
			b.WriteString(fmt.Sprintf("- %s (Tiro %02d)\n", it.Name, total))
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
	b.WriteString(fmt.Sprintf("Tiro #%d\n", cur+1))
	b.WriteString("Espressione: " + entry.Expression + "\n")
	b.WriteString("Risultato: " + entry.Output + "\n")
	b.WriteString(fmt.Sprintf("\nTotale tiri: %d", len(ui.diceLog)))
	return strings.TrimSpace(b.String())
}

func (ui *tviewUI) openDiceRollInput() {
	input := tview.NewInputField().SetLabel(" Tiro ").SetFieldWidth(36)
	input.SetBorder(true).SetTitle("Dadi (es. 2d6+3, 1d20+5>15, d6,d8, 1d20+4 x3)")
	returnFocus := ui.app.GetFocus()

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 72, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

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

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 64, 0, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

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
