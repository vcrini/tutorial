## LazySW

LazySW è un'applicazione da terminale per supportare il gioco con **Savage Worlds Adventure Edition (SWADE)**:

- **Catalogo** mostri, equipaggiamento, classi/regole e ambienti, caricati da file YAML nella cartella `config`.
- **Gestione PNG** (personaggi non giocanti) con nome generato, classe e descrizione riassunta.
- **Gestione encounter** con ferite, condizioni (Scosso, Vulnerabile, ecc.) e iniziativa.
- **Roller di dadi** integrato (inclusa notazione `D6` in stile Savage Worlds, dadi esplosivi, vantaggio/svantaggio).

L'interfaccia è una **TUI** basata su `tview` e funziona in un terminale "cursor addressable" (es. iTerm2, Terminal.app, Alacritty, kitty, ecc.).

## Requisiti

- Go 1.22 o successivo.
- Un terminale che supporti colori ANSI e controllo del cursore.

## Installazione

Nella root del progetto `go/lazysw`:

```bash
go build ./...
```

oppure per l'esecuzione diretta:

```bash
go run .
```

Se usi un gestore di versioni Go (es. `gvm`, `asdf`), assicurati che la versione attiva sia compatibile con il `go.mod` del progetto.

## Avvio

Da dentro la cartella `go/lazysw`:

```bash
go run .
```

Se il programma termina subito con un errore tipo `terminal not cursor addressable`, significa che l'ambiente non espone un vero TTY interattivo (es. esecuzione da IDE/sandbox). In questo caso avvialo da un terminale di sistema (macOS, Linux, ecc.).

## Modalità CLI / headless

Per usare LazySW senza interfaccia TUI (ad esempio in un terminale non "cursor addressable" o da script), puoi usare la modalità CLI:

```bash
./lazysw cli dice "2d6+1"
./lazysw cli monsters
./lazysw cli monsters drago
```

Comandi disponibili:

- **`cli dice <espressione>`**: esegue un tiro di dadi con la stessa notazione supportata dal pannello dadi (incluso `D6` stile Savage Worlds) e stampa totale e dettaglio.
- **`cli monsters [filtro]`**: stampa l'elenco dei mostri dal file `config/mostri.yml`; se passi un filtro, viene usato come substring case-insensitive sul nome del mostro.

## Struttura principale

- `main.go`: entrypoint, avvia la TUI (`runTViewUI`).
- `cli.go`: entrypoint per la modalità CLI/headless (`runCLI`).
- `tview_ui.go`: definisce l'interfaccia utente (liste, pannelli, scorciatoie da tastiera, help contestuale).
- `data.go`: tipi e funzioni per caricare/salvare:
  - PNG (`pngs.yml`, JSON legacy),
  - mostri (`config/mostri.yml`),
  - equipaggiamento (`config/equipaggiamento.yaml`),
  - classi/regole (`config/classi.yaml`),
  - nomi (`config/names.yaml`),
  - encounter (`encounter.yml`),
  - storico tiri (`dice_history.yml`).
- `encounter.go`, `view.go`, `menu.go`, `selection.go`, `monster_history.go`, `monsters.go`: logica di dominio per encounter, selezioni, viste e storico mostri.

I vari file `config/*.yml` / `config/*.yaml` contengono i dati in formato YAML (mostri, equipaggiamento, classi, nomi).

## Comandi di base (tastiera)

Questa è una sintesi delle scorciatoie più importanti (vedi help interno `?` per la lista completa):

- **Navigazione generale**
  - `tab` / `shift+tab`: cambia focus tra i pannelli.
  - `0` / `1` / `2` / `3`: cambia layout/pannello principale.
  - `PgUp` / `PgDn`: scroll sui dettagli.
  - `q`: esci dall'applicazione.
  - `?`: mostra l'help contestuale.
  - `f`: toggle fullscreen.

- **Pannello dadi**
  - `a`: nuovo tiro di dado (espressione libera).
  - `Invio`: rilancia il tiro selezionato.
  - `e`: modifica + rilancia il tiro selezionato.
  - `d`: elimina il tiro selezionato.
  - `c`: svuota lo storico dei tiri.

- **Pannello encounter**
  - `i` / `I` / `S`: gestisci iniziativa (uno, tutti, ordina).
  - `h` / `l` o `j` / `k`: incrementa/decrementa ferite.
  - `c`: aggiungi/togli condizioni.
  - `x`: rimuovi una condizione dall'entry.

- **Pannello PNG e cataloghi**
  - `c`: crea un nuovo PNG (pannello PNG).
  - `m`: rinomina il PNG selezionato.
  - `x`: elimina il PNG selezionato.
  - `a`: aggiungi PNG/mostro selezionato all'encounter.
  - `/`: ricerca raw nel pannello corrente.
  - `u` / `t` / `g`: gestisci filtri sul pannello attivo.

La legenda completa della notazione dei dadi è visibile nell'help relativa al pannello dadi.

## File di dati e persistenza

Per impostazione predefinita:

- I file di **configurazione** (mostri, equipaggiamento, classi, nomi) vivono sotto `config/` e sono versionati nel repository.
- I file di **stato locale** (PNG, encounter, storico tiri) vivono in una directory "persistente" specifica dell'utente, gestita da `persistentPath` (vedi `data.go`), così da non sporcare la cartella del codice.

Puoi personalizzare i contenuti YAML in `config/` per il tuo setting, mantenendo la stessa struttura dei tipi Go in `data.go`.

## Test

Sono presenti test unitari in `main_test.go` per:

- helper di formattazione e capitalizzazione,
- salvataggio/caricamento PNG (incluso formato legacy),
- condizioni encounter e loro rappresentazione testuale,
- salvataggio/caricamento storico dei tiri,
- parsing della notazione `D*` con dado destino.

Per eseguirli:

```bash
go test ./...
```

## Note legali

LazySW è uno strumento **non ufficiale** pensato per facilitare il gioco con Savage Worlds Adventure Edition.

- **Savage Worlds**, **SWADE** e tutti i relativi loghi e marchi sono proprietà di **Pinnacle Entertainment Group**.
- Questo progetto non è affiliato, approvato o sponsorizzato da Pinnacle Entertainment Group.
- Verifica sempre la licenza ufficiale (SRD / Fan License / Community License) prima di distribuire pubblicamente materiale derivato e adatta i contenuti YAML (descrizioni, nomi, ecc.) in modo coerente con i termini previsti.

