package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

const helpText = " [black:gold] q [-:-] esci  [black:gold] / [-:-] cerca nome  [black:gold] tab [-:-] focus  [black:gold] j/k [-:-] naviga  [black:gold] PgUp/PgDn [-:-] scroll dettagli "

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

type UI struct {
	app        *tview.Application
	monsters   []Monster
	filtered   []int
	envOptions []string
	crOptions  []string

	nameFilter string
	envFilter  string
	crFilter   string

	nameInput *tview.InputField
	envDrop   *tview.DropDown
	crDrop    *tview.DropDown
	list      *tview.List
	detail    *tview.TextView
	status    *tview.TextView
	pages     *tview.Pages

	focusOrder []tview.Primitive
}

func main() {
	yamlPath := strings.TrimSpace(os.Getenv("MONSTERS_YAML"))
	var (
		monsters []Monster
		envs     []string
		crs      []string
		err      error
	)

	if yamlPath != "" {
		monsters, envs, crs, err = loadMonstersFromPath(yamlPath)
		if err != nil {
			log.Fatalf("errore caricamento YAML esterno (%s): %v", yamlPath, err)
		}
	} else {
		monsters, envs, crs, err = loadMonstersFromBytes(embeddedMonstersYAML)
		if err != nil {
			log.Fatalf("errore caricamento YAML embedded: %v", err)
		}
	}

	ui := newUI(monsters, envs, crs)
	if err := ui.run(); err != nil {
		log.Fatal(err)
	}
}

func newUI(monsters []Monster, envs, crs []string) *UI {
	setTheme()

	ui := &UI{
		app:        tview.NewApplication(),
		monsters:   monsters,
		envOptions: append([]string{"All"}, envs...),
		crOptions:  append([]string{"All"}, crs...),
		filtered:   make([]int, 0, len(monsters)),
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

	ui.list = tview.NewList()
	ui.list.SetBorder(true)
	ui.list.SetTitle(" Monsters ")
	ui.list.SetTitleColor(tcell.ColorGold)
	ui.list.SetBorderColor(tcell.ColorGold)
	ui.list.SetMainTextColor(tcell.ColorWhite)
	ui.list.SetSecondaryTextColor(tcell.ColorLightGray)
	ui.list.SetSelectedTextColor(tcell.ColorBlack)
	ui.list.SetSelectedBackgroundColor(tcell.ColorGold)
	ui.list.ShowSecondaryText(true)
	ui.list.SetChangedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByListIndex(index)
	})
	ui.list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		ui.renderDetailByListIndex(index)
	})

	ui.detail = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	ui.detail.SetBorder(true)
	ui.detail.SetTitle(" Details ")
	ui.detail.SetTitleColor(tcell.ColorGold)
	ui.detail.SetBorderColor(tcell.ColorGold)
	ui.detail.SetTextColor(tcell.ColorWhite)
	ui.detail.SetWrap(true)

	ui.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText)
	ui.status.SetBackgroundColor(tcell.ColorBlack)

	filterRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.nameInput, 30, 0, true).
		AddItem(ui.envDrop, 26, 0, false).
		AddItem(ui.crDrop, 18, 0, false)

	mainRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(ui.list, 0, 1, false).
		AddItem(ui.detail, 0, 1, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(filterRow, 4, 0, true).
		AddItem(mainRow, 0, 1, false).
		AddItem(ui.status, 1, 0, false)

	ui.pages = tview.NewPages().AddPage("main", root, true, true)
	ui.app.SetRoot(ui.pages, true)
	ui.focusOrder = []tview.Primitive{ui.nameInput, ui.envDrop, ui.crDrop, ui.list, ui.detail}
	ui.app.SetFocus(ui.list)
	ui.envDrop.SetCurrentOption(0)
	ui.crDrop.SetCurrentOption(0)

	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		focus := ui.app.GetFocus()
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == 'q':
			ui.app.Stop()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == '/':
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
		case focus != ui.nameInput && event.Key() == tcell.KeyRune && event.Rune() == 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case focus != ui.nameInput && event.Key() == tcell.KeyRune && event.Rune() == 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		default:
			return event
		}
	})

	ui.applyFilters()
	return ui
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

	_, _, _, height := ui.detail.GetInnerRect()
	if height <= 0 {
		height = 10
	}

	row, _ := ui.detail.GetScrollOffset()
	step := height - 1
	if step < 1 {
		step = 1
	}

	nextRow := row + (step * direction)
	if nextRow < 0 {
		nextRow = 0
	}
	ui.detail.ScrollTo(nextRow, 0)
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
		ui.filtered = append(ui.filtered, i)
	}

	ui.renderList()
}

func (ui *UI) renderList() {
	ui.list.Clear()

	for _, idx := range ui.filtered {
		m := ui.monsters[idx]
		env := "n/a"
		if len(m.Environment) > 0 {
			env = strings.Join(m.Environment, ", ")
		}
		secondary := fmt.Sprintf("[lightgray]%s | %s | %s", blankIfEmpty(m.Type, "type?"), blankIfEmpty(m.Source, "src?"), env)
		ui.list.AddItem(m.Name, secondary, 0, nil)
	}

	ui.status.SetText(fmt.Sprintf(" [black:gold] %d risultati [-:-] %s", len(ui.filtered), helpText))

	if len(ui.filtered) == 0 {
		ui.detail.SetText("Nessun mostro con i filtri correnti.")
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
		ui.detail.SetText("Seleziona un mostro dalla lista.")
		return
	}

	m := ui.monsters[ui.filtered[listIndex]]
	raw, _ := json.MarshalIndent(m.Raw, "", "  ")

	builder := &strings.Builder{}
	fmt.Fprintf(builder, "[yellow]%s[-]\n", m.Name)
	fmt.Fprintf(builder, "[white]Source:[-] %s\n", blankIfEmpty(m.Source, "n/a"))
	fmt.Fprintf(builder, "[white]Type:[-] %s\n", blankIfEmpty(m.Type, "n/a"))
	fmt.Fprintf(builder, "[white]CR:[-] %s\n", blankIfEmpty(m.CR, "n/a"))
	if len(m.Environment) > 0 {
		fmt.Fprintf(builder, "[white]Environment:[-] %s\n", strings.Join(m.Environment, ", "))
	} else {
		fmt.Fprintf(builder, "[white]Environment:[-] n/a\n")
	}
	fmt.Fprintf(builder, "\n[deepskyblue]Raw JSON[-]\n%s", string(raw))

	ui.detail.SetText(builder.String())
}

func loadMonstersFromPath(path string) ([]Monster, []string, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, err
	}
	return loadMonstersFromBytes(b)
}

func loadMonstersFromBytes(b []byte) ([]Monster, []string, []string, error) {
	var ds dataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, nil, nil, err
	}
	if len(ds.Monsters) == 0 {
		return nil, nil, nil, errors.New("nessun mostro trovato nel yaml")
	}

	monsters := make([]Monster, 0, len(ds.Monsters))
	envSet := map[string]struct{}{}
	crSet := map[string]struct{}{}

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
	}

	sort.Slice(monsters, func(i, j int) bool {
		return strings.ToLower(monsters[i].Name) < strings.ToLower(monsters[j].Name)
	})

	return monsters, keysSorted(envSet), sortCR(keysSorted(crSet)), nil
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
