package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const helpText = " [black:gold]q[-:-] esci  [black:gold]?[-:-] help  [black:gold]f[-:-] fullscreen  [black:gold]tab/shift+tab[-:-] focus  [black:gold]1/2/3[-:-] pannelli  [black:gold][[ / ]][-:-] Mostri/Ambienti/Equip./Carte  [black:gold]/[-:-] ricerca raw  [black:gold]PgUp/PgDn[-:-] scroll dettagli  [black:gold]u/t/g[-:-] filtri pannello  [black:gold]v[-:-] reset filtri "

const (
	focusPNG = iota
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
	focusDetail
)

type tviewUI struct {
	app    *tview.Application
	pages  *tview.Pages
	status *tview.TextView

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
	detail         *tview.TextView

	monstersPanel     *tview.Flex
	environmentsPanel *tview.Flex
	equipmentPanel    *tview.Flex
	cardsPanel        *tview.Flex
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
	encounter        []EncounterEntry
	filtered         []int
	filteredEnv      []int
	filteredEq       []int
	filteredCards    []int
	roleOpts         []string
	rankOpts         []string
	envTypeOpts      []string
	envRankOpts      []string
	eqTypeOpts       []string
	eqItemTypeOpts   []string
	eqRankOpts       []string
	cardClassOpts    []string
	cardTypeOpts     []string
	roleFilter       string
	rankFilter       string
	envTypeFilter    string
	envRankFilter    string
	eqTypeFilter     string
	eqItemTypeFilter string
	eqRankFilter     string
	cardClassFilter  string
	cardTypeFilter   string
	catalogMode      string

	detailRaw   string
	detailQuery string

	helpVisible     bool
	helpReturnFocus tview.Primitive

	modalVisible bool
	modalName    string

	fullscreenActive bool
	fullscreenTarget string
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
		app:          tview.NewApplication(),
		pngs:         pngs,
		selected:     selected,
		monsters:     monsters,
		environments: environments,
		equipment:    equipment,
		cards:        cards,
		encounter:    encounter,
		message:      "Pronto.",
		catalogMode:  "mostri",
	}
	ui.build()
	return ui, nil
}

func (ui *tviewUI) build() {
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

	ui.catalogPanel = tview.NewPages().
		AddPage("mostri", ui.monstersPanel, true, true).
		AddPage("ambienti", ui.environmentsPanel, true, false).
		AddPage("equipaggiamento", ui.equipmentPanel, true, false).
		AddPage("carte", ui.cardsPanel, true, false)
	ui.refreshCatalogTitles()

	ui.leftPanel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.pngList, 0, 1, true).
		AddItem(ui.encList, 0, 1, false).
		AddItem(ui.catalogPanel, 0, 1, false)

	ui.detail = tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	ui.detail.SetBorder(true).SetTitle(" Dettagli ")

	ui.mainRow = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.leftPanel, 0, 1, false).
		AddItem(ui.detail, 0, 1, false)

	ui.status = tview.NewTextView().SetDynamicColors(true).SetText(helpText)
	ui.status.SetBackgroundColor(tcell.ColorBlack)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.mainRow, 0, 1, true).
		AddItem(ui.status, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", root, true, true)
	ui.focus = []tview.Primitive{
		ui.pngList, ui.encList, ui.search, ui.roleDrop, ui.rankDrop, ui.monList,
		ui.envSearch, ui.envTypeDrop, ui.envRankDrop, ui.envList,
		ui.eqSearch, ui.eqTypeDrop, ui.eqItemTypeDrop, ui.eqRankDrop, ui.eqList,
		ui.cardSearch, ui.cardClassDrop, ui.cardTypeDrop, ui.cardList,
		ui.detail,
	}
	ui.focusIdx = focusMonList
	ui.app.SetFocus(ui.monList)
	ui.app.SetInputCapture(ui.handleGlobalKeys)
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
	case tcell.KeyTAB:
		ui.focusNext()
		return nil
	case tcell.KeyBacktab:
		ui.focusPrev()
		return nil
	case tcell.KeyLeft:
		if focus == ui.pngList {
			ui.adjustSelectedToken(-1)
			return nil
		}
	case tcell.KeyRight:
		if focus == ui.pngList {
			ui.adjustSelectedToken(1)
			return nil
		}
	case tcell.KeyPgUp:
		if focus == ui.detail || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.envList || focus == ui.envSearch || focus == ui.envTypeDrop || focus == ui.envRankDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop {
			ui.scrollDetailByPage(-1)
			return nil
		}
	case tcell.KeyPgDn:
		if focus == ui.detail || focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop || focus == ui.envList || focus == ui.envSearch || focus == ui.envTypeDrop || focus == ui.envRankDrop || focus == ui.eqList || focus == ui.eqSearch || focus == ui.eqTypeDrop || focus == ui.eqItemTypeDrop || focus == ui.eqRankDrop || focus == ui.cardList || focus == ui.cardSearch || focus == ui.cardClassDrop || focus == ui.cardTypeDrop {
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
		ui.focusPanel(0)
		return nil
	case '2':
		ui.focusPanel(1)
		return nil
	case '3':
		ui.focusPanel(ui.activeCatalogListFocus())
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
		ui.openCreatePNGModal()
		return nil
	case 'x':
		ui.openDeletePNGConfirm()
		return nil
	case 'r':
		if focus == ui.pngList {
			ui.openResetTokensConfirm()
			return nil
		}
	case 'a':
		if ui.catalogMode == "mostri" && (focus == ui.monList || focus == ui.search || focus == ui.roleDrop || focus == ui.rankDrop) {
			ui.addSelectedMonsterToEncounter()
			return nil
		}
	case 'u':
		if ui.catalogMode == "mostri" {
			ui.focusPanel(focusMonSearch)
		} else if ui.catalogMode == "ambienti" {
			ui.focusPanel(focusEnvSearch)
		} else if ui.catalogMode == "equipaggiamento" {
			ui.focusPanel(focusEqSearch)
		} else {
			ui.focusPanel(focusCardSearch)
		}
		return nil
	case 't':
		if ui.catalogMode == "mostri" {
			ui.focusPanel(focusMonRole)
		} else if ui.catalogMode == "ambienti" {
			ui.focusPanel(focusEnvType)
		} else if ui.catalogMode == "equipaggiamento" {
			ui.focusPanel(focusEqItemType)
		} else {
			ui.focusPanel(focusCardClass)
		}
		return nil
	case 'g':
		if ui.catalogMode == "mostri" {
			ui.focusPanel(focusMonRank)
		} else if ui.catalogMode == "ambienti" {
			ui.focusPanel(focusEnvRank)
		} else if ui.catalogMode == "equipaggiamento" {
			ui.focusPanel(focusEqRank)
		} else {
			ui.focusPanel(focusCardType)
		}
		return nil
	case 'v':
		if ui.catalogMode == "mostri" {
			ui.resetMonsterFilters()
		} else if ui.catalogMode == "ambienti" {
			ui.resetEnvironmentFilters()
		} else if ui.catalogMode == "equipaggiamento" {
			ui.resetEquipmentFilters()
		} else {
			ui.resetCardFilters()
		}
		return nil
	case 'd':
		if focus == ui.encList {
			ui.removeSelectedEncounter()
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
	default:
		return "Mostri"
	}
}

func (ui *tviewUI) refreshCatalogTitles() {
	order := []string{"mostri", "ambienti", "equipaggiamento", "carte"}
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
		}
	}
}

func (ui *tviewUI) switchCatalog(delta int) {
	if delta == 0 {
		return
	}
	order := []string{"mostri", "ambienti", "equipaggiamento", "carte"}
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
		ui.pngList.AddItem(fmt.Sprintf("%s%s [token %d]", prefix, p.Name, p.Token), "", 0, nil)
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
		remaining := base - e.Wounds
		if remaining < 0 {
			remaining = 0
		}
		label := ui.encounterLabelAt(i)
		ui.encList.AddItem(fmt.Sprintf("%s [PF %d/%d]", label, remaining, base), "", 0, nil)
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
		remaining := base - e.Wounds
		if remaining < 0 {
			remaining = 0
		}
		extra := fmt.Sprintf("PF correnti: %d/%d | Ferite: %d", remaining, base, e.Wounds)
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
	ui.detailRaw = fmt.Sprintf("%s\nToken: %d", p.Name, p.Token)
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
	case ui.encList:
		focusLabel = "Encounter"
	case ui.search:
		focusLabel = "Nome Mostri"
	case ui.roleDrop:
		focusLabel = "Ruolo Mostri"
	case ui.rankDrop:
		focusLabel = "Rango Mostri"
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
	b.WriteString(fmt.Sprintf("Ruolo: %s | Rango: %d\n", m.Role, m.Rank))
	if extraLine != "" {
		b.WriteString(extraLine + "\n")
	}
	b.WriteString(fmt.Sprintf("PF: %d | Stress: %d | Difficoltà: %d\n", m.PF, m.Stress, m.Difficulty))
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
			b.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s (%s)\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType, bonus))
		} else {
			b.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s\n", m.Attack.Name, m.Attack.Range, m.Attack.Damage, m.Attack.DamageType))
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
	name := ui.encounter[idx].Monster.Name
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
	p.Token += delta
	if p.Token < minToken {
		p.Token = minToken
	}
	if p.Token > maxToken {
		p.Token = maxToken
	}
	ui.persistPNGs()
	ui.message = fmt.Sprintf("Token di %s: %d", p.Name, p.Token)
	ui.refreshPNGs()
	ui.refreshDetail()
	ui.refreshStatus()
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
		ui.pngs = append(ui.pngs, PNG{Name: name, Token: defaultToken})
		ui.selected = len(ui.pngs) - 1
		ui.persistPNGs()
		ui.closeModal()
		ui.focusPanel(0)
		ui.message = fmt.Sprintf("Creato PNG %s.", name)
		ui.refreshAll()
	})
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
	if focus == ui.detail {
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

func (ui *tviewUI) scrollDetailByPage(direction int) {
	row, col := ui.detail.GetScrollOffset()
	_, _, _, h := ui.detail.GetInnerRect()
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
	ui.detail.ScrollTo(row, col)
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

func (ui *tviewUI) resetTokens() {
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
	ui.encounter = append(ui.encounter, EncounterEntry{Monster: mon, BasePF: mon.PF})
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Aggiunto %s a encounter.", mon.Name)
	ui.refreshEncounter()
	ui.refreshStatus()
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
	e.Wounds += delta
	if e.Wounds < 0 {
		e.Wounds = 0
	}
	base := e.BasePF
	if base == 0 {
		base = e.Monster.PF
	}
	if base > 0 && e.Wounds > base {
		e.Wounds = base
	}
	ui.persistEncounter()
	ui.message = fmt.Sprintf("Ferite %s: %d", e.Monster.Name, e.Wounds)
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

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(text, 0, 1, true).
			AddItem(nil, 0, 1, false), 0, 1, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddAndSwitchToPage("help", modal, true)
	ui.app.SetFocus(text)
}

func (ui *tviewUI) buildHelpContent(focus tview.Primitive) string {
	var b strings.Builder
	b.WriteString("LazyDaggerheart - scorciatoie\n\n")

	panel := "Dettagli"
	var panelLines []string
	switch focus {
	case ui.pngList:
		panel = "PNG"
		panelLines = []string{
			"- c: crea PNG",
			"- x: elimina PNG selezionato",
			"- r: reset token di tutti i PNG",
			"- ← / →: diminuisci/aumenta token selezionato",
		}
	case ui.encList:
		panel = "Encounter"
		panelLines = []string{
			"- d: rimuovi mostro selezionato",
			"- h / l: ferite +1 / -1 sul selezionato",
		}
	case ui.search, ui.roleDrop, ui.rankDrop, ui.monList:
		panel = "Mostri"
		panelLines = []string{
			"- a: aggiungi mostro selezionato a Encounter",
			"- u / t / g: focus filtro Nome / Ruolo / Rango",
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
		}
	case ui.cardSearch, ui.cardClassDrop, ui.cardTypeDrop, ui.cardList:
		panel = "Carte"
		panelLines = []string{
			"- u / t / g: focus filtro Nome / Classe / Tipo",
			"- v: reset filtri Carte (Nome/Classe/Tipo)",
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
	b.WriteString("- 1 / 2 / 3: focus PNG / Encounter / Catalogo\n")
	b.WriteString("- [ / ]: alterna Mostri / Ambienti / Equipaggiamento / Carte\n")
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
		Name   string `yaml:"name"`
		Wounds int    `yaml:"wounds"`
		PF     int    `yaml:"pf"`
	}, 0, len(ui.encounter))
	for _, e := range ui.encounter {
		base := e.BasePF
		if base == 0 {
			base = e.Monster.PF
		}
		entries = append(entries, struct {
			Name   string `yaml:"name"`
			Wounds int    `yaml:"wounds"`
			PF     int    `yaml:"pf"`
		}{Name: e.Monster.Name, Wounds: e.Wounds, PF: base})
	}
	_ = saveEncounter(encounterFile, entries)
}
