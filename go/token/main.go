package main

import (
	"fmt"
	"math/rand/v2" // Usiamo rand/v2 per una migliore generazione di numeri casuali
	"os"
	"time"

	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	// TextView per mostrare i messaggi all'utente
	messageTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(true).
		SetTextAlign(tview.AlignCenter)
	messageTextView.SetText("Premi un tasto per iniziare!")

	// Funzionalità 1: Saluto
	onSaluta := func() {
		messageTextView.SetText("[green]Ciao! Benvenuto nell'applicazione TUI.[-]")
	}

	// Funzionalità 2: Mostra Data e Ora
	onMostraDataOra := func() {
		currentTime := time.Now().Format("02/01/2006 15:04:05")
		messageTextView.SetText(fmt.Sprintf("[blue]La data e l'ora attuali sono: %s[-]", currentTime))
	}

	// Funzionalità 3: Mostra un numero casuale
	onNumeroCasuale := func() {
		randomNumber := rand.IntN(1000) + 1 // Un numero tra 1 e 1000
		messageTextView.SetText(fmt.Sprintf("[yellow]Il tuo numero casuale è: %d[-]", randomNumber))
	}

	// Funzionalità 4: Esci dall'applicazione
	onEsci := func() {
		app.Stop() // Ferma l'applicazione tview
	}

	// Crea un form per raggruppare i pulsanti in una colonna
	form := tview.NewForm().
		AddButton("Saluta", onSaluta).
		AddButton("Mostra Data/Ora", onMostraDataOra).
		AddButton("Numero Casuale", onNumeroCasuale).
		AddButton("Esci", onEsci).
		SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor) // Stile di sfondo per i bottoni

	form.SetBorder(true).SetTitle("Seleziona un'opzione").SetTitleAlign(tview.AlignCenter)

	// Crea un layout Flex per organizzare il messaggio e i pulsanti
	// Utilizziamo un Flex per dare spazio al messaggio e ai controlli
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(messageTextView, 0, 1, false).                 // Il messaggio occupa 1/3 dello spazio verticale
		AddItem(tview.NewBox().SetBorder(false), 1, 0, false). // Spazio vuoto
		AddItem(form, 0, 2, true)                              // Il form con i bottoni occupa 2/3 dello spazio verticale e ha il focus iniziale

	// Imposta il form come elemento iniziale con il focus
	app.SetRoot(flex, true).SetFocus(form)

	// Avvia l'applicazione tview. Se c'è un errore, lo stampa.
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
