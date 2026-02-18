# Mini Isaac Prototype (Go)

Prototipo ispirato a The Binding of Isaac, scritto in Go con Ebitengine.

## Features

- layout procedurale a ogni run (start/combat/shop/boss)
- contenuto stanza procedurale con template (arena/crossfire/gauntlet/corners/midlane/open)
- movimento player (WASD)
- shooting in 4 direzioni (frecce)
- supporto gamepad (movimento + mira + dash)
- dash con invulnerabilita' breve (`Shift`)
- bomb system (`E`) con fuse + esplosione ad area
- nemici: chaser, wander, shooter, dasher, boss multi-fase
- boss con telegraph shot, spread phase 2 e ring phase 3
- drop casuali (heart / bomb / coin / key)
- chest system apribile con chiavi (`G`) o con bombe
- hazard a terra (spike zones)
- economia base con coins/keys/bombs e shop room
- shop interaction (`F`) con offerte random + reroll (`H`)
- reward item per stanza con sinergie (damage, fire rate, speed, heal, crit, pierce, multishot, bomb master, luck, shield)
- minimappa stanze visitate (toggle `M`)
- score + best score + kill streak + rank run
- seed run visibile + timer run
- livelli multipli: dopo aver sconfitto il boss scendi al piano successivo (`L`)
- pausa (`P`) e nuova run (`N`)
- meta save locale (`save_meta.json`) per best/runs/deaths
- telemetria run locale append-only (`run_telemetry.jsonl`)

## Run

```bash
cd isaac
go mod tidy
go run .
```

## Controls

- `W A S D`: movimento
- `Arrow keys`: sparo
- `Space`: sparo verso destra (fallback)
- `Shift`: dash
- `E`: piazza bomba
- `G`: apri chest se hai una key
- `F`: acquista in shop quando sei vicino a un'offerta
- `H`: rerolla offerte shop (costo crescente)
- `L`: scendi al piano successivo quando il boss e' sconfitto
- passa sopra item/drop per raccoglierli
- attraversa una porta quando la stanza e' pulita per cambiare stanza
- `P`: pausa
- `M`: mostra/nascondi minimappa
- `N`: nuova run (nuovo seed)
- `R`: restart stesso seed dopo morte
- `Esc`: uscita
