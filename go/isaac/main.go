package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenW = 960
	screenH = 540

	roomMargin = 48
	doorHalf   = 34

	playerRadius = 14
	playerSpeed  = 3.2
	playerMaxHP  = 6

	fireCooldownFrames = 8
	bulletSpeed        = 7.2
	bulletRadius       = 4
	bulletDamage       = 1

	enemyChaserSpeed  = 1.4
	enemyWanderSpeed  = 1.1
	enemyShooterSpeed = 0.9
	enemyRadius       = 13
	enemyShotRadius   = 4
	enemyShotSpeed    = 3.6
	enemyShotDelay    = 90

	bossRadius       = 24
	bossSpeed        = 1.0
	bossSpeedP2      = 1.35
	bossSpeedP3      = 1.55
	bossShotSpeed    = 4.3
	bossShotRadius   = 5
	bossShotDamage   = 1
	bossShotDelay    = 48
	bossShotDelayP2  = 26
	bossSpreadRad    = 0.35
	bossSpreadCount  = 3
	bossRingCountP3  = 8
	bossWindupFrames = 16
	bossBaseHP       = 26
	bossPhase3HP     = 7
	bossRingDelayP3  = 120

	enemyDamageCooldown = 40
	contactDamage       = 1
	roomSwapCooldown    = 14
	itemRadius          = 10
	itemTextDuration    = 140

	dashDurationFrames = 8
	dashCooldownFrames = 48
	dashSpeedMult      = 2.8

	bombFuseFrames    = 90
	bombBlastRadius   = 74
	bombDamage        = 7
	bombStartCount    = 3
	bombPlaceCooldown = 10

	dropHeartChance = 0.16
	dropBombChance  = 0.12
	dropCoinChance  = 0.44
	dropKeyChance   = 0.11
	pickupRadius    = 9

	explosionFrames     = 14
	shopInteractRadius  = 26
	streakTimeoutFrames = 190
	lowHPThreshold      = 2
	spikeDamageTick     = 24
	transitionFramesMax = 18

	proceduralCombatRooms = 10
	layoutBound           = 4
)

type Vec2 struct {
	X float64
	Y float64
}

type Bullet struct {
	Pos    Vec2
	Vel    Vec2
	Active bool
	Pierce int
}

type EnemyShot struct {
	Pos      Vec2
	Vel      Vec2
	Active   bool
	FromBoss bool
}

type Bomb struct {
	Pos    Vec2
	Timer  int
	Active bool
}

type Explosion struct {
	Pos    Vec2
	Timer  int
	Radius float64
}

type EnemyType int

const (
	EnemyChaser EnemyType = iota
	EnemyWander
	EnemyShooter
	EnemyDasher
	EnemyBoss
)

type Enemy struct {
	Pos           Vec2
	Vel           Vec2
	HP            int
	Kind          EnemyType
	Alive         bool
	WanderTimer   int
	ShootCooldown int
	ShootWindup   int
	BossRingCD    int
}

type ItemType int

const (
	ItemDamage ItemType = iota
	ItemFireRate
	ItemSpeed
	ItemHeal
	ItemCrit
	ItemPierce
	ItemMultiShot
	ItemBombMaster
	ItemLuck
	ItemShield
)

type Item struct {
	Pos   Vec2
	Kind  ItemType
	Taken bool
}

type PickupType int

const (
	PickupHeart PickupType = iota
	PickupBomb
	PickupCoin
	PickupKey
)

type Pickup struct {
	Pos    Vec2
	Kind   PickupType
	Active bool
}

type RoomType int

const (
	RoomStart RoomType = iota
	RoomCombat
	RoomShop
	RoomBoss
)

type OfferType int

const (
	OfferHeart OfferType = iota
	OfferBombPack
	OfferDamage
	OfferKey
	OfferCrit
)

type ShopOffer struct {
	Pos       Vec2
	Kind      OfferType
	Price     int
	Purchased bool
}

type Chest struct {
	Pos    Vec2
	Opened bool
}

type Hazard struct {
	Pos Vec2
	R   float64
}

type Room struct {
	ID       int
	GridX    int
	GridY    int
	Type     RoomType
	Enemies  []Enemy
	Reward   Item
	Pickups  []Pickup
	Offers   []ShopOffer
	Chests   []Chest
	Hazards  []Hazard
	Template string
}

type RoomTemplate struct {
	Name       string
	Hazards    []Hazard
	ChestPos   []Vec2
	EnemySlots []Vec2
}

type Game struct {
	playerPos       Vec2
	playerHP        int
	playerInvFrames int
	fireCooldown    int
	swapCooldown    int
	itemTextFrames  int
	lastItemText    string
	bombPlaceCD     int
	spikeTick       int

	moveSpeed        float64
	shotCooldownBase int
	shotDamage       int
	critChance       float64
	critMult         float64
	luck             float64
	pierceCount      int
	multiShot        bool
	shieldCharges    int
	maxShieldCharges int
	bombRadiusMult   float64
	bombDamageBonus  int
	lastAimDir       Vec2

	dashDir      Vec2
	dashFrames   int
	dashCooldown int
	lastMoveDir  Vec2

	bombs int
	coins int
	keys  int

	bullets    []Bullet
	enemyShots []EnemyShot
	enemies    []Enemy
	bombList   []Bomb
	explosions []Explosion
	pickups    []Pickup
	offers     []ShopOffer
	chests     []Chest
	hazards    []Hazard

	rng            *rand.Rand
	runSeed        int64
	runFrames      int
	roomClear      bool
	rooms          map[int]*Room
	gridToRoomID   map[[2]int]int
	currentRoomID  int
	visitedRooms   map[int]bool
	statusText     string
	statusTextTick int
	paused         bool
	showMiniMap    bool
	transitionTick int
	shakeTick      int
	shakeMag       float64

	bossRoomID    int
	shopRoomID    int
	floor         int
	floorsCleared int

	score         int
	bestScore     int
	killCount     int
	killStreak    int
	streakTick    int
	runsCompleted int
	deaths        int
	shopRerolls   int

	runRoomsVisited int
	runDamageTaken  int
	runDamageDealt  int
}

type MetaSave struct {
	BestScore     int `json:"best_score"`
	RunsCompleted int `json:"runs_completed"`
	Deaths        int `json:"deaths"`
}

type RunTelemetry struct {
	Timestamp       string `json:"timestamp"`
	Seed            int64  `json:"seed"`
	Floor           int    `json:"floor"`
	Score           int    `json:"score"`
	RoomsVisited    int    `json:"rooms_visited"`
	EnemiesDefeated int    `json:"enemies_defeated"`
	DamageTaken     int    `json:"damage_taken"`
	DamageDealt     int    `json:"damage_dealt"`
	RunSeconds      int    `json:"run_seconds"`
	Rank            string `json:"rank"`
	Result          string `json:"result"`
}

func NewGame() *Game {
	g := &Game{}
	g.loadMeta()
	g.startNewRun()
	return g
}

func (g *Game) startNewRun() {
	if g.runFrames > 0 {
		g.saveRunTelemetry("new_run")
	}
	g.runSeed = time.Now().UnixNano()
	g.rng = rand.New(rand.NewSource(g.runSeed))
	g.floor = 1
	g.resetRun()
}

func (g *Game) resetRun() {
	if g.rng == nil {
		g.rng = rand.New(rand.NewSource(g.runSeed))
	}

	g.playerPos = Vec2{X: screenW / 2, Y: screenH / 2}
	g.playerHP = playerMaxHP
	g.playerInvFrames = 0
	g.fireCooldown = 0
	g.swapCooldown = 0
	g.itemTextFrames = 0
	g.lastItemText = ""
	g.bombPlaceCD = 0
	g.spikeTick = 0
	g.moveSpeed = playerSpeed
	g.shotCooldownBase = fireCooldownFrames
	g.shotDamage = bulletDamage
	g.critChance = 0.08
	g.critMult = 1.8
	g.luck = 0
	g.pierceCount = 0
	g.multiShot = false
	g.shieldCharges = 0
	g.maxShieldCharges = 0
	g.bombRadiusMult = 1.0
	g.bombDamageBonus = 0
	g.lastAimDir = Vec2{X: 1, Y: 0}
	g.dashDir = Vec2{}
	g.dashFrames = 0
	g.dashCooldown = 0
	g.lastMoveDir = Vec2{X: 1, Y: 0}
	g.bombs = bombStartCount
	g.coins = 0
	g.keys = 0
	g.bullets = g.bullets[:0]
	g.enemyShots = g.enemyShots[:0]
	g.bombList = g.bombList[:0]
	g.explosions = g.explosions[:0]
	g.pickups = g.pickups[:0]
	g.offers = g.offers[:0]
	g.chests = g.chests[:0]
	g.hazards = g.hazards[:0]
	g.statusText = ""
	g.statusTextTick = 0
	g.paused = false
	g.showMiniMap = true
	g.transitionTick = 0
	g.shakeTick = 0
	g.shakeMag = 0
	g.score = 0
	g.killCount = 0
	g.killStreak = 0
	g.streakTick = 0
	g.runFrames = 0
	g.shopRerolls = 0
	g.runRoomsVisited = 1
	g.runDamageTaken = 0
	g.runDamageDealt = 0
	g.floorsCleared = 0

	g.initRoomsProcedural()
	g.visitedRooms = map[int]bool{g.currentRoomID: true}
	g.loadCurrentRoom()
	g.updateRoomClear()
}

func (g *Game) initRoomsProcedural() {
	targetRooms := proceduralCombatRooms + 3 + minInt(5, g.floor-1)
	cells := g.generateLayoutCells(targetRooms)
	startCell := [2]int{0, 0}

	bossCell := startCell
	bestDist := -1
	for _, c := range cells {
		d := absInt(c[0]-startCell[0]) + absInt(c[1]-startCell[1])
		if d > bestDist {
			bestDist = d
			bossCell = c
		}
	}

	shopCandidates := make([][2]int, 0, len(cells))
	for _, c := range cells {
		if c == startCell || c == bossCell {
			continue
		}
		shopCandidates = append(shopCandidates, c)
	}
	shopCell := shopCandidates[g.rng.Intn(len(shopCandidates))]

	sort.Slice(cells, func(i, j int) bool {
		if cells[i][0] == cells[j][0] {
			return cells[i][1] < cells[j][1]
		}
		return cells[i][0] < cells[j][0]
	})

	g.rooms = make(map[int]*Room, len(cells))
	g.gridToRoomID = make(map[[2]int]int, len(cells))
	idForCell := make(map[[2]int]int, len(cells))

	nextID := 0
	for _, c := range cells {
		idForCell[c] = nextID
		nextID++
	}

	for _, c := range cells {
		id := idForCell[c]
		r := &Room{ID: id, GridX: c[0], GridY: c[1], Type: RoomCombat}

		switch {
		case c == startCell:
			r.Type = RoomStart
			g.currentRoomID = id
			g.populateStartRoom(r)
		case c == bossCell:
			r.Type = RoomBoss
			g.bossRoomID = id
			g.populateBossRoom(r)
		case c == shopCell:
			r.Type = RoomShop
			g.shopRoomID = id
			g.populateShopRoom(r)
		default:
			depth := absInt(c[0]-startCell[0]) + absInt(c[1]-startCell[1])
			g.populateCombatRoom(r, depth)
		}

		g.rooms[id] = r
		g.gridToRoomID[[2]int{c[0], c[1]}] = id
	}
}

func (g *Game) generateLayoutCells(target int) [][2]int {
	seen := map[[2]int]bool{{0, 0}: true}
	cells := [][2]int{{0, 0}}
	current := [2]int{0, 0}
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for len(cells) < target {
		d := dirs[g.rng.Intn(len(dirs))]
		next := [2]int{current[0] + d[0], current[1] + d[1]}
		if absInt(next[0]) > layoutBound || absInt(next[1]) > layoutBound {
			current = cells[g.rng.Intn(len(cells))]
			continue
		}
		if !seen[next] {
			seen[next] = true
			cells = append(cells, next)
		}
		if g.rng.Float64() < 0.55 {
			current = next
		} else {
			current = cells[g.rng.Intn(len(cells))]
		}
	}
	return cells
}

func (g *Game) populateStartRoom(r *Room) {
	r.Reward = Item{Pos: Vec2{X: screenW / 2, Y: screenH / 2}, Kind: ItemHeal}
	r.Chests = []Chest{{Pos: Vec2{X: 180, Y: 420}}}
	// Keep hazards away from the spawn area (screen center).
	r.Hazards = []Hazard{{Pos: Vec2{X: 700, Y: 340}, R: 17}}
	r.Enemies = []Enemy{
		{Pos: Vec2{X: 220, Y: 180}, HP: 3, Kind: EnemyChaser, Alive: true},
	}
}

func (g *Game) populateCombatRoom(r *Room, depth int) {
	templates := g.roomTemplates()
	tpl := templates[g.rng.Intn(len(templates))]
	r.Template = tpl.Name
	r.Hazards = append(r.Hazards[:0], tpl.Hazards...)
	for _, c := range tpl.ChestPos {
		r.Chests = append(r.Chests, Chest{Pos: c})
	}

	enemyCount := minInt(len(tpl.EnemySlots), 2+minInt(4, (depth+g.floor)/2))
	r.Enemies = make([]Enemy, 0, enemyCount)
	for i := 0; i < enemyCount; i++ {
		kindRoll := g.rng.Intn(100)
		kind := EnemyChaser
		hp := 2 + depth/2 + g.floor/2
		switch {
		case kindRoll < 28:
			kind = EnemyChaser
		case kindRoll < 52:
			kind = EnemyWander
		case kindRoll < 84:
			kind = EnemyShooter
		default:
			kind = EnemyDasher
		}
		slot := tpl.EnemySlots[i]
		x := slot.X + (g.rng.Float64()*24 - 12)
		y := slot.Y + (g.rng.Float64()*24 - 12)
		e := Enemy{Pos: Vec2{X: x, Y: y}, HP: hp, Kind: kind, Alive: true}
		if kind == EnemyShooter {
			e.ShootCooldown = enemyShotDelay - minInt(35, depth*4)
		}
		if kind == EnemyDasher {
			e.WanderTimer = 40 + g.rng.Intn(30)
		}
		r.Enemies = append(r.Enemies, e)
	}

	if g.rng.Float64() < 0.20+0.05*float64(minInt(6, g.floor)) {
		r.Chests = append(r.Chests, Chest{Pos: Vec2{X: 120 + g.rng.Float64()*720, Y: 100 + g.rng.Float64()*320}})
	}
	r.Reward = Item{Pos: Vec2{X: screenW / 2, Y: screenH / 2}, Kind: ItemType(g.rng.Intn(int(ItemShield) + 1))}
}

func (g *Game) roomTemplates() []RoomTemplate {
	return []RoomTemplate{
		{
			Name:       "Arena",
			Hazards:    []Hazard{{Pos: Vec2{X: 480, Y: 270}, R: 18}},
			ChestPos:   []Vec2{{X: 140, Y: 420}},
			EnemySlots: []Vec2{{X: 220, Y: 160}, {X: 730, Y: 160}, {X: 220, Y: 380}, {X: 730, Y: 380}, {X: 480, Y: 150}},
		},
		{
			Name:       "Crossfire",
			Hazards:    []Hazard{{Pos: Vec2{X: 350, Y: 220}, R: 14}, {Pos: Vec2{X: 610, Y: 320}, R: 14}},
			ChestPos:   []Vec2{{X: 820, Y: 120}},
			EnemySlots: []Vec2{{X: 180, Y: 270}, {X: 780, Y: 270}, {X: 480, Y: 140}, {X: 480, Y: 400}, {X: 680, Y: 180}},
		},
		{
			Name:       "Gauntlet",
			Hazards:    []Hazard{{Pos: Vec2{X: 300, Y: 180}, R: 16}, {Pos: Vec2{X: 480, Y: 270}, R: 16}, {Pos: Vec2{X: 660, Y: 360}, R: 16}},
			ChestPos:   []Vec2{{X: 120, Y: 120}},
			EnemySlots: []Vec2{{X: 200, Y: 120}, {X: 760, Y: 120}, {X: 200, Y: 420}, {X: 760, Y: 420}, {X: 480, Y: 270}},
		},
		{
			Name:       "Corners",
			Hazards:    []Hazard{{Pos: Vec2{X: 480, Y: 120}, R: 13}, {Pos: Vec2{X: 480, Y: 420}, R: 13}},
			ChestPos:   []Vec2{{X: 820, Y: 420}},
			EnemySlots: []Vec2{{X: 140, Y: 120}, {X: 820, Y: 120}, {X: 140, Y: 420}, {X: 820, Y: 420}, {X: 480, Y: 270}},
		},
		{
			Name:       "Midlane",
			Hazards:    []Hazard{{Pos: Vec2{X: 390, Y: 270}, R: 15}, {Pos: Vec2{X: 570, Y: 270}, R: 15}},
			ChestPos:   []Vec2{{X: 160, Y: 100}},
			EnemySlots: []Vec2{{X: 230, Y: 200}, {X: 730, Y: 200}, {X: 230, Y: 340}, {X: 730, Y: 340}, {X: 480, Y: 130}},
		},
		{
			Name:       "Open",
			Hazards:    nil,
			ChestPos:   []Vec2{{X: 820, Y: 100}},
			EnemySlots: []Vec2{{X: 240, Y: 150}, {X: 720, Y: 150}, {X: 240, Y: 390}, {X: 720, Y: 390}, {X: 480, Y: 270}},
		},
	}
}

func (g *Game) populateShopRoom(r *Room) {
	r.Reward = Item{Taken: true}
	offers := make([]ShopOffer, 0, 5)
	xs := []float64{300, 390, 480, 570, 660}
	for i := 0; i < 5; i++ {
		kind := OfferType(g.rng.Intn(5))
		price := 2 + g.rng.Intn(8)
		offers = append(offers, ShopOffer{Pos: Vec2{X: xs[i], Y: 240}, Kind: kind, Price: price})
	}
	r.Offers = offers
}

func (g *Game) populateBossRoom(r *Room) {
	r.Reward = Item{Taken: true}
	r.Enemies = []Enemy{{Pos: Vec2{X: screenW / 2, Y: screenH / 2}, HP: bossBaseHP, Kind: EnemyBoss, Alive: true, ShootCooldown: bossShotDelay, BossRingCD: bossRingDelayP3}}
}

func (g *Game) loadCurrentRoom() {
	room := g.currentRoom()
	g.enemies = g.enemies[:0]
	for _, e := range room.Enemies {
		clone := e
		if clone.Kind == EnemyWander && clone.WanderTimer == 0 && clone.Alive {
			clone.WanderTimer = 15 + g.rng.Intn(35)
		}
		if clone.Kind == EnemyShooter && clone.ShootCooldown <= 0 {
			clone.ShootCooldown = enemyShotDelay
		}
		if clone.Kind == EnemyBoss {
			if clone.ShootCooldown <= 0 {
				clone.ShootCooldown = bossShotDelay
			}
			if clone.BossRingCD <= 0 {
				clone.BossRingCD = bossRingDelayP3
			}
		}
		g.enemies = append(g.enemies, clone)
	}
	g.pickups = append(g.pickups[:0], room.Pickups...)
	g.offers = append(g.offers[:0], room.Offers...)
	g.chests = append(g.chests[:0], room.Chests...)
	g.hazards = append(g.hazards[:0], room.Hazards...)
	g.bullets = g.bullets[:0]
	g.enemyShots = g.enemyShots[:0]
	g.bombList = g.bombList[:0]
	g.explosions = g.explosions[:0]
	g.updateRoomClear()
}

func (g *Game) saveCurrentRoomState() {
	room := g.currentRoom()
	room.Enemies = append(room.Enemies[:0], g.enemies...)
	room.Pickups = append(room.Pickups[:0], g.pickups...)
	room.Offers = append(room.Offers[:0], g.offers...)
	room.Chests = append(room.Chests[:0], g.chests...)
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.startNewRun()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.showMiniMap = !g.showMiniMap
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
	}
	if g.playerHP <= 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.resetRun()
		}
		return nil
	}
	if g.paused {
		return nil
	}

	g.runFrames++
	if g.fireCooldown > 0 {
		g.fireCooldown--
	}
	if g.swapCooldown > 0 {
		g.swapCooldown--
	}
	if g.itemTextFrames > 0 {
		g.itemTextFrames--
	}
	if g.statusTextTick > 0 {
		g.statusTextTick--
	}
	if g.transitionTick > 0 {
		g.transitionTick--
	}
	if g.playerInvFrames > 0 {
		g.playerInvFrames--
	}
	if g.dashCooldown > 0 {
		g.dashCooldown--
	}
	if g.dashFrames > 0 {
		g.dashFrames--
	}
	if g.bombPlaceCD > 0 {
		g.bombPlaceCD--
	}
	if g.spikeTick > 0 {
		g.spikeTick--
	}
	if g.streakTick > 0 {
		g.streakTick--
		if g.streakTick == 0 {
			g.killStreak = 0
		}
	}

	g.updatePlayerMove()
	g.tryShoot()
	g.tryPlaceBomb()
	g.updateBullets()
	g.updateEnemies()
	g.updateEnemyShots()
	g.updateBombs()
	g.updateExplosions()
	g.applyHazardDamage()
	g.checkPlayerEnemyCollisions()
	g.checkPlayerEnemyShotCollisions()
	g.updateRoomClear()
	g.tryPickupItem()
	g.tryPickupDrops()
	g.tryOpenChest()
	g.tryBuyShopOffer()
	g.tryRerollShop()
	g.tryRoomTransition()
	g.tryDescendFloor()
	return nil
}

func (g *Game) updatePlayerMove() {
	dx, dy := g.moveInput()
	moveDir := Vec2{}
	if dx != 0 || dy != 0 {
		l := math.Hypot(dx, dy)
		moveDir = Vec2{X: dx / l, Y: dy / l}
		g.lastMoveDir = moveDir
	}
	dashPressed := inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight)
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton0) {
			dashPressed = true
			break
		}
	}
	if dashPressed && g.dashCooldown == 0 {
		d := moveDir
		if d == (Vec2{}) {
			d = g.lastMoveDir
		}
		if d != (Vec2{}) {
			g.dashDir = d
			g.dashFrames = dashDurationFrames
			g.dashCooldown = dashCooldownFrames
		}
	}
	speed := g.moveSpeed
	dir := moveDir
	if g.dashFrames > 0 {
		speed *= dashSpeedMult
		dir = g.dashDir
	}
	g.playerPos.X += dir.X * speed
	g.playerPos.Y += dir.Y * speed
	g.playerPos.X = clamp(g.playerPos.X, roomMargin+playerRadius, screenW-roomMargin-playerRadius)
	g.playerPos.Y = clamp(g.playerPos.Y, roomMargin+playerRadius, screenH-roomMargin-playerRadius)
}

func (g *Game) tryShoot() {
	if g.fireCooldown > 0 {
		return
	}
	dir := g.aimInput()
	if dir == (Vec2{}) {
		return
	}
	g.lastAimDir = dir

	g.fireCooldown = g.shotCooldownBase
	g.bullets = append(g.bullets, Bullet{
		Pos:    g.playerPos,
		Vel:    Vec2{X: dir.X * bulletSpeed, Y: dir.Y * bulletSpeed},
		Active: true,
		Pierce: g.pierceCount,
	})
	if g.multiShot {
		side := Vec2{X: -dir.Y, Y: dir.X}
		spread := 0.22
		v1 := Vec2{X: dir.X + side.X*spread, Y: dir.Y + side.Y*spread}
		v2 := Vec2{X: dir.X - side.X*spread, Y: dir.Y - side.Y*spread}
		l1 := math.Hypot(v1.X, v1.Y)
		l2 := math.Hypot(v2.X, v2.Y)
		if l1 > 0 {
			v1.X /= l1
			v1.Y /= l1
			g.bullets = append(g.bullets, Bullet{Pos: g.playerPos, Vel: Vec2{X: v1.X * bulletSpeed, Y: v1.Y * bulletSpeed}, Active: true, Pierce: maxInt(0, g.pierceCount-1)})
		}
		if l2 > 0 {
			v2.X /= l2
			v2.Y /= l2
			g.bullets = append(g.bullets, Bullet{Pos: g.playerPos, Vel: Vec2{X: v2.X * bulletSpeed, Y: v2.Y * bulletSpeed}, Active: true, Pierce: maxInt(0, g.pierceCount-1)})
		}
	}
}

func (g *Game) tryPlaceBomb() {
	if g.bombs <= 0 || g.bombPlaceCD > 0 || !inpututil.IsKeyJustPressed(ebiten.KeyE) {
		return
	}
	g.bombs--
	g.bombPlaceCD = bombPlaceCooldown
	g.bombList = append(g.bombList, Bomb{Pos: g.playerPos, Timer: bombFuseFrames, Active: true})
	g.emitEvent("bomb_place")
}

func (g *Game) moveInput() (float64, float64) {
	var dx, dy float64
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		dx -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		dx += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		dy -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		dy += 1
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		ax := ebiten.GamepadAxisValue(id, 0)
		ay := ebiten.GamepadAxisValue(id, 1)
		if math.Abs(ax) > 0.2 {
			dx += ax
		}
		if math.Abs(ay) > 0.2 {
			dy += ay
		}
	}
	return dx, dy
}

func (g *Game) aimInput() Vec2 {
	dir := Vec2{}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dir = Vec2{Y: -1}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		dir = Vec2{Y: 1}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dir = Vec2{X: -1}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dir = Vec2{X: 1}
	}
	if dir == (Vec2{}) && ebiten.IsKeyPressed(ebiten.KeySpace) {
		dir = g.lastAimDir
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		ax := ebiten.GamepadAxisValue(id, 2)
		ay := ebiten.GamepadAxisValue(id, 3)
		if math.Abs(ax) > 0.35 || math.Abs(ay) > 0.35 {
			dir = Vec2{X: ax, Y: ay}
		}
	}
	if dir == (Vec2{}) {
		return dir
	}
	l := math.Hypot(dir.X, dir.Y)
	if l == 0 {
		return Vec2{}
	}
	return Vec2{X: dir.X / l, Y: dir.Y / l}
}

func (g *Game) updateBullets() {
	for i := range g.bullets {
		b := &g.bullets[i]
		if !b.Active {
			continue
		}
		b.Pos.X += b.Vel.X
		b.Pos.Y += b.Vel.Y
		if b.Pos.X < roomMargin || b.Pos.X > screenW-roomMargin || b.Pos.Y < roomMargin || b.Pos.Y > screenH-roomMargin {
			b.Active = false
			continue
		}
		for ei := range g.enemies {
			e := &g.enemies[ei]
			if !e.Alive {
				continue
			}
			r := enemyRadius
			if e.Kind == EnemyBoss {
				r = bossRadius
			}
			if distance(b.Pos, e.Pos) <= bulletRadius+float64(r) {
				dmg := g.rollShotDamage()
				e.HP -= dmg
				g.runDamageDealt += dmg
				if e.HP <= 0 {
					e.Alive = false
					g.onEnemyKilled(*e)
				}
				if b.Pierce > 0 {
					b.Pierce--
				} else {
					b.Active = false
				}
				break
			}
		}
	}
	alive := g.bullets[:0]
	for _, b := range g.bullets {
		if b.Active {
			alive = append(alive, b)
		}
	}
	g.bullets = alive
}

func (g *Game) rollShotDamage() int {
	dmg := g.shotDamage
	if g.rng.Float64() < g.critChance {
		dmg = int(math.Ceil(float64(dmg) * g.critMult))
		g.lastItemText = "Critical hit!"
		g.itemTextFrames = 24
	}
	return dmg
}

func (g *Game) updateEnemies() {
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.Alive {
			continue
		}
		switch e.Kind {
		case EnemyChaser:
			g.updateChaser(e)
		case EnemyWander:
			g.updateWander(e)
		case EnemyShooter:
			g.updateShooter(e)
		case EnemyDasher:
			g.updateDasher(e)
		case EnemyBoss:
			g.updateBoss(e)
		}
		e.Pos.X += e.Vel.X
		e.Pos.Y += e.Vel.Y
		r := enemyRadius
		if e.Kind == EnemyBoss {
			r = bossRadius
		}
		if e.Pos.X < roomMargin+float64(r) || e.Pos.X > screenW-roomMargin-float64(r) {
			e.Vel.X *= -1
			e.Pos.X = clamp(e.Pos.X, roomMargin+float64(r), screenW-roomMargin-float64(r))
		}
		if e.Pos.Y < roomMargin+float64(r) || e.Pos.Y > screenH-roomMargin-float64(r) {
			e.Vel.Y *= -1
			e.Pos.Y = clamp(e.Pos.Y, roomMargin+float64(r), screenH-roomMargin-float64(r))
		}
	}
}

func (g *Game) updateChaser(e *Enemy) {
	dx := g.playerPos.X - e.Pos.X
	dy := g.playerPos.Y - e.Pos.Y
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	s := enemyChaserSpeed * g.enemyDifficultyScale()
	e.Vel = Vec2{X: dx / l * s, Y: dy / l * s}
}

func (g *Game) updateWander(e *Enemy) {
	e.WanderTimer--
	if e.WanderTimer <= 0 {
		a := g.rng.Float64() * 2 * math.Pi
		s := enemyWanderSpeed * g.enemyDifficultyScale()
		e.Vel = Vec2{X: math.Cos(a) * s, Y: math.Sin(a) * s}
		e.WanderTimer = 20 + g.rng.Intn(60)
	}
}

func (g *Game) updateShooter(e *Enemy) {
	dx := g.playerPos.X - e.Pos.X
	dy := g.playerPos.Y - e.Pos.Y
	l := math.Hypot(dx, dy)
	if l > 0 {
		e.Vel = Vec2{X: dx / l * enemyShooterSpeed, Y: dy / l * enemyShooterSpeed}
	}
	e.ShootCooldown--
	if e.ShootCooldown <= 0 {
		if l > 0 {
			g.enemyShots = append(g.enemyShots, EnemyShot{Pos: e.Pos, Vel: Vec2{X: dx / l * enemyShotSpeed, Y: dy / l * enemyShotSpeed}, Active: true, FromBoss: false})
		}
		e.ShootCooldown = maxInt(35, int(float64(enemyShotDelay)/g.enemyDifficultyScale()))
	}
}

func (g *Game) updateDasher(e *Enemy) {
	e.WanderTimer--
	if e.WanderTimer <= 0 {
		dx := g.playerPos.X - e.Pos.X
		dy := g.playerPos.Y - e.Pos.Y
		l := math.Hypot(dx, dy)
		if l > 0 {
			dashSpeed := 2.8 * g.enemyDifficultyScale()
			e.Vel = Vec2{X: dx / l * dashSpeed, Y: dy / l * dashSpeed}
		}
		e.WanderTimer = 35 + g.rng.Intn(40)
	} else {
		e.Vel.X *= 0.95
		e.Vel.Y *= 0.95
	}
}

func (g *Game) updateBoss(e *Enemy) {
	dx := g.playerPos.X - e.Pos.X
	dy := g.playerPos.Y - e.Pos.Y
	l := math.Hypot(dx, dy)
	speed := bossSpeed
	if g.isBossPhase2(*e) {
		speed = bossSpeedP2
	}
	if g.isBossPhase3(*e) {
		speed = bossSpeedP3
	}
	if l > 0 {
		e.Vel = Vec2{X: dx / l * speed, Y: dy / l * speed}
	}

	if e.ShootWindup > 0 {
		e.ShootWindup--
		if e.ShootWindup == 0 {
			g.fireBossShot(*e)
		}
	} else {
		e.ShootCooldown--
		if e.ShootCooldown <= 0 {
			e.ShootWindup = bossWindupFrames
			if g.isBossPhase2(*e) {
				e.ShootCooldown = bossShotDelayP2
			} else {
				e.ShootCooldown = bossShotDelay
			}
		}
	}
	if g.isBossPhase3(*e) {
		e.BossRingCD--
		if e.BossRingCD <= 0 {
			g.fireBossRing(*e)
			e.BossRingCD = bossRingDelayP3
		}
	}
}

func (g *Game) fireBossShot(boss Enemy) {
	a := math.Atan2(g.playerPos.Y-boss.Pos.Y, g.playerPos.X-boss.Pos.X)
	if g.isBossPhase2(boss) {
		start := -bossSpreadRad
		step := (bossSpreadRad * 2) / float64(maxInt(1, bossSpreadCount-1))
		for i := 0; i < bossSpreadCount; i++ {
			ang := a + start + float64(i)*step
			g.enemyShots = append(g.enemyShots, EnemyShot{Pos: boss.Pos, Vel: Vec2{X: math.Cos(ang) * bossShotSpeed, Y: math.Sin(ang) * bossShotSpeed}, Active: true, FromBoss: true})
		}
		return
	}
	g.enemyShots = append(g.enemyShots, EnemyShot{Pos: boss.Pos, Vel: Vec2{X: math.Cos(a) * bossShotSpeed, Y: math.Sin(a) * bossShotSpeed}, Active: true, FromBoss: true})
}

func (g *Game) fireBossRing(boss Enemy) {
	for i := 0; i < bossRingCountP3; i++ {
		a := 2 * math.Pi * float64(i) / float64(bossRingCountP3)
		g.enemyShots = append(g.enemyShots, EnemyShot{Pos: boss.Pos, Vel: Vec2{X: math.Cos(a) * (bossShotSpeed - 0.8), Y: math.Sin(a) * (bossShotSpeed - 0.8)}, Active: true, FromBoss: true})
	}
}

func (g *Game) isBossPhase2(boss Enemy) bool { return boss.HP > 0 && boss.HP <= bossBaseHP/2 }
func (g *Game) isBossPhase3(boss Enemy) bool { return boss.HP > 0 && boss.HP <= bossPhase3HP }

func (g *Game) updateEnemyShots() {
	for i := range g.enemyShots {
		s := &g.enemyShots[i]
		if !s.Active {
			continue
		}
		s.Pos.X += s.Vel.X
		s.Pos.Y += s.Vel.Y
		if s.Pos.X < roomMargin || s.Pos.X > screenW-roomMargin || s.Pos.Y < roomMargin || s.Pos.Y > screenH-roomMargin {
			s.Active = false
		}
	}
	alive := g.enemyShots[:0]
	for _, s := range g.enemyShots {
		if s.Active {
			alive = append(alive, s)
		}
	}
	g.enemyShots = alive
}

func (g *Game) updateBombs() {
	for i := range g.bombList {
		b := &g.bombList[i]
		if !b.Active {
			continue
		}
		b.Timer--
		if b.Timer <= 0 {
			b.Active = false
			g.explodeBomb(b.Pos)
		}
	}
	alive := g.bombList[:0]
	for _, b := range g.bombList {
		if b.Active {
			alive = append(alive, b)
		}
	}
	g.bombList = alive
}

func (g *Game) explodeBomb(pos Vec2) {
	radius := bombBlastRadius * g.bombRadiusMult
	damage := bombDamage + g.bombDamageBonus
	g.explosions = append(g.explosions, Explosion{Pos: pos, Timer: explosionFrames, Radius: radius})
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.Alive {
			continue
		}
		r := enemyRadius
		if e.Kind == EnemyBoss {
			r = bossRadius
		}
		if distance(pos, e.Pos) <= radius+float64(r) {
			e.HP -= damage
			g.runDamageDealt += damage
			if e.HP <= 0 {
				e.Alive = false
				g.onEnemyKilled(*e)
			}
		}
	}
	for i := range g.chests {
		c := &g.chests[i]
		if !c.Opened && distance(pos, c.Pos) <= radius+12 {
			g.openChest(c)
		}
	}
	if distance(pos, g.playerPos) <= radius+playerRadius {
		g.damagePlayer(1)
	}
	g.shakeTick = 8
	g.shakeMag = 5
	g.emitEvent("bomb_explode")
}

func (g *Game) updateExplosions() {
	alive := g.explosions[:0]
	for _, ex := range g.explosions {
		ex.Timer--
		if ex.Timer > 0 {
			alive = append(alive, ex)
		}
	}
	g.explosions = alive
}

func (g *Game) onEnemyKilled(enemy Enemy) {
	g.killCount++
	g.killStreak++
	g.streakTick = streakTimeoutFrames
	base := 10
	if enemy.Kind == EnemyWander {
		base = 15
	}
	if enemy.Kind == EnemyShooter {
		base = 20
	}
	if enemy.Kind == EnemyBoss {
		base = 300
		g.runsCompleted++
	}
	mult := 1.0 + math.Min(float64(g.killStreak-1)*0.12, 1.2)
	g.score += int(float64(base) * mult)
	if g.score > g.bestScore {
		g.bestScore = g.score
		g.saveMeta()
	}
	if enemy.Kind == EnemyBoss {
		g.saveMeta()
		return
	}
	r := g.rng.Float64()
	heartChance := clamp(dropHeartChance+g.luck*0.35, 0, 0.45)
	bombChance := clamp(dropBombChance+g.luck*0.20, 0, 0.30)
	coinChance := clamp(dropCoinChance+g.luck*0.25, 0, 0.70)
	keyChance := clamp(dropKeyChance+g.luck*0.15, 0, 0.25)
	switch {
	case r < heartChance:
		g.pickups = append(g.pickups, Pickup{Pos: enemy.Pos, Kind: PickupHeart, Active: true})
	case r < heartChance+bombChance:
		g.pickups = append(g.pickups, Pickup{Pos: enemy.Pos, Kind: PickupBomb, Active: true})
	case r < heartChance+bombChance+coinChance:
		g.pickups = append(g.pickups, Pickup{Pos: enemy.Pos, Kind: PickupCoin, Active: true})
	case r < heartChance+bombChance+coinChance+keyChance:
		g.pickups = append(g.pickups, Pickup{Pos: enemy.Pos, Kind: PickupKey, Active: true})
	}
}

func (g *Game) applyHazardDamage() {
	if g.spikeTick > 0 || g.dashFrames > 0 {
		return
	}
	for _, h := range g.hazards {
		if distance(g.playerPos, h.Pos) <= h.R+playerRadius {
			g.damagePlayer(1)
			g.spikeTick = spikeDamageTick
			return
		}
	}
}

func (g *Game) checkPlayerEnemyCollisions() {
	if g.playerInvFrames > 0 {
		return
	}
	for _, e := range g.enemies {
		if !e.Alive {
			continue
		}
		r := enemyRadius
		if e.Kind == EnemyBoss {
			r = bossRadius
		}
		if distance(g.playerPos, e.Pos) <= playerRadius+float64(r) {
			g.damagePlayer(contactDamage)
			return
		}
	}
}

func (g *Game) checkPlayerEnemyShotCollisions() {
	if g.playerInvFrames > 0 {
		return
	}
	for i := range g.enemyShots {
		s := &g.enemyShots[i]
		if !s.Active {
			continue
		}
		r := enemyShotRadius
		dmg := 1
		if s.FromBoss {
			r = bossShotRadius
			dmg = bossShotDamage
		}
		if distance(g.playerPos, s.Pos) <= playerRadius+float64(r) {
			s.Active = false
			g.damagePlayer(dmg)
			return
		}
	}
}

func (g *Game) damagePlayer(amount int) {
	if g.dashFrames > 0 {
		return
	}
	if g.shieldCharges > 0 {
		g.shieldCharges--
		g.statusText = "Shield blocked damage"
		g.statusTextTick = 40
		return
	}
	g.playerHP -= amount
	if g.playerHP < 0 {
		g.playerHP = 0
	}
	g.runDamageTaken += amount
	g.playerInvFrames = enemyDamageCooldown
	g.killStreak = 0
	g.streakTick = 0
	if g.playerHP == 0 {
		g.deaths++
		g.saveMeta()
		g.saveRunTelemetry("death")
	}
	g.shakeTick = 10
	g.shakeMag = 4
	g.emitEvent("player_hit")
}

func (g *Game) updateRoomClear() {
	for _, e := range g.enemies {
		if e.Alive {
			g.roomClear = false
			return
		}
	}
	g.roomClear = true
}

func (g *Game) currentRoom() *Room { return g.rooms[g.currentRoomID] }

func (g *Game) roomInDir(dx, dy int) (int, bool) {
	cur := g.currentRoom()
	id, ok := g.gridToRoomID[[2]int{cur.GridX + dx, cur.GridY + dy}]
	return id, ok
}

func (g *Game) roomNeedsClearForBoss(id int) bool {
	r := g.rooms[id]
	return r.Type == RoomStart || r.Type == RoomCombat
}

func (g *Game) allNonBossRoomsCleared() bool {
	for id, room := range g.rooms {
		if id == g.bossRoomID || !g.roomNeedsClearForBoss(id) {
			continue
		}
		enemies := room.Enemies
		if id == g.currentRoomID {
			enemies = g.enemies
		}
		for _, e := range enemies {
			if e.Alive {
				return false
			}
		}
	}
	return true
}

func (g *Game) tryPickupItem() {
	if !g.roomClear {
		return
	}
	room := g.currentRoom()
	if room.Reward.Taken || distance(g.playerPos, room.Reward.Pos) > playerRadius+itemRadius {
		return
	}
	room.Reward.Taken = true
	g.applyItem(room.Reward.Kind)
}

func (g *Game) tryPickupDrops() {
	for i := range g.pickups {
		p := &g.pickups[i]
		if !p.Active || distance(g.playerPos, p.Pos) > playerRadius+pickupRadius {
			continue
		}
		p.Active = false
		switch p.Kind {
		case PickupHeart:
			g.playerHP = minInt(playerMaxHP, g.playerHP+1)
			g.lastItemText = "Picked up: Heart"
		case PickupBomb:
			g.bombs = minInt(9, g.bombs+1)
			g.lastItemText = "Picked up: Bomb"
		case PickupCoin:
			g.coins++
			g.lastItemText = "Picked up: Coin"
		case PickupKey:
			g.keys++
			g.lastItemText = "Picked up: Key"
		}
		g.itemTextFrames = itemTextDuration
	}
	alive := g.pickups[:0]
	for _, p := range g.pickups {
		if p.Active {
			alive = append(alive, p)
		}
	}
	g.pickups = alive
}

func (g *Game) tryOpenChest() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyG) {
		return
	}
	for i := range g.chests {
		c := &g.chests[i]
		if c.Opened || distance(g.playerPos, c.Pos) > shopInteractRadius {
			continue
		}
		if g.keys <= 0 {
			g.statusText = "Need a key"
			g.statusTextTick = 80
			return
		}
		g.keys--
		g.openChest(c)
		return
	}
}

func (g *Game) openChest(c *Chest) {
	if c.Opened {
		return
	}
	c.Opened = true
	r := g.rng.Float64()
	switch {
	case r < 0.30:
		g.pickups = append(g.pickups, Pickup{Pos: c.Pos, Kind: PickupCoin, Active: true})
		g.pickups = append(g.pickups, Pickup{Pos: Vec2{X: c.Pos.X + 16, Y: c.Pos.Y}, Kind: PickupCoin, Active: true})
		g.lastItemText = "Chest: Coins"
	case r < 0.55:
		g.pickups = append(g.pickups, Pickup{Pos: c.Pos, Kind: PickupBomb, Active: true})
		g.lastItemText = "Chest: Bomb"
	case r < 0.75:
		g.pickups = append(g.pickups, Pickup{Pos: c.Pos, Kind: PickupHeart, Active: true})
		g.lastItemText = "Chest: Heart"
	case r < 0.90:
		g.pickups = append(g.pickups, Pickup{Pos: c.Pos, Kind: PickupKey, Active: true})
		g.lastItemText = "Chest: Key"
	default:
		g.score += 40
		g.lastItemText = "Chest: Treasure Score"
	}
	if g.score > g.bestScore {
		g.bestScore = g.score
		g.saveMeta()
	}
	g.itemTextFrames = itemTextDuration
}

func (g *Game) tryBuyShopOffer() {
	if g.currentRoom().Type != RoomShop || !inpututil.IsKeyJustPressed(ebiten.KeyF) {
		return
	}
	for i := range g.offers {
		o := &g.offers[i]
		if o.Purchased || distance(g.playerPos, o.Pos) > shopInteractRadius {
			continue
		}
		if g.coins < o.Price {
			g.statusText = "Not enough coins"
			g.statusTextTick = 90
			return
		}
		g.coins -= o.Price
		o.Purchased = true
		switch o.Kind {
		case OfferHeart:
			g.playerHP = minInt(playerMaxHP, g.playerHP+2)
			g.lastItemText = "Bought: Heart Bundle (+2 HP)"
		case OfferBombPack:
			g.bombs = minInt(9, g.bombs+3)
			g.lastItemText = "Bought: Bomb Pack (+3 Bombs)"
		case OfferDamage:
			g.shotDamage++
			g.lastItemText = "Bought: Damage Up"
		case OfferKey:
			g.keys += 2
			g.lastItemText = "Bought: 2 Keys"
		case OfferCrit:
			g.critChance = clamp(g.critChance+0.08, 0, 0.75)
			g.lastItemText = "Bought: Crit Up"
		}
		g.itemTextFrames = itemTextDuration
		g.saveMeta()
		return
	}
}

func (g *Game) tryRerollShop() {
	if g.currentRoom().Type != RoomShop || !inpututil.IsKeyJustPressed(ebiten.KeyH) {
		return
	}
	cost := 2 + g.shopRerolls
	if g.coins < cost {
		g.statusText = fmt.Sprintf("Need %d coins to reroll", cost)
		g.statusTextTick = 90
		return
	}
	g.coins -= cost
	g.shopRerolls++
	for i := range g.offers {
		if g.offers[i].Purchased {
			continue
		}
		g.offers[i].Kind = OfferType(g.rng.Intn(5))
		g.offers[i].Price = 2 + g.rng.Intn(8)
	}
	g.statusText = "Shop rerolled"
	g.statusTextTick = 80
}

func (g *Game) applyItem(kind ItemType) {
	switch kind {
	case ItemDamage:
		g.shotDamage++
		g.lastItemText = "Picked up: Blood Drop (+Damage)"
	case ItemFireRate:
		if g.shotCooldownBase > 4 {
			g.shotCooldownBase -= 2
		}
		g.lastItemText = "Picked up: Torn Page (+Fire Rate)"
	case ItemSpeed:
		g.moveSpeed += 0.35
		g.lastItemText = "Picked up: Running Shoe (+Speed)"
	case ItemHeal:
		g.playerHP = minInt(playerMaxHP, g.playerHP+1)
		g.lastItemText = "Picked up: Heart Patch (+1 HP)"
	case ItemCrit:
		g.critChance = clamp(g.critChance+0.10, 0, 0.8)
		g.lastItemText = "Picked up: Sharp Eye (+Crit)"
	case ItemPierce:
		g.pierceCount = minInt(3, g.pierceCount+1)
		g.lastItemText = "Picked up: Needle Tear (+Pierce)"
	case ItemMultiShot:
		g.multiShot = true
		g.lastItemText = "Picked up: Twin Eye (+MultiShot)"
	case ItemBombMaster:
		g.bombRadiusMult = clamp(g.bombRadiusMult+0.15, 1.0, 1.9)
		g.bombDamageBonus += 2
		g.lastItemText = "Picked up: Bomber Kit"
	case ItemLuck:
		g.luck = clamp(g.luck+0.08, 0, 0.6)
		g.critChance = clamp(g.critChance+0.03, 0, 0.9)
		g.lastItemText = "Picked up: Lucky Charm"
	case ItemShield:
		g.maxShieldCharges = minInt(3, g.maxShieldCharges+1)
		g.shieldCharges = g.maxShieldCharges
		g.lastItemText = "Picked up: Halo Shield"
	}
	g.itemTextFrames = itemTextDuration
}

func (g *Game) tryRoomTransition() {
	if !g.roomClear || g.swapCooldown > 0 {
		return
	}
	var nextID int
	var ok bool
	var spawn Vec2
	nearLeft := g.playerPos.X <= roomMargin+playerRadius+1 && math.Abs(g.playerPos.Y-screenH/2) <= doorHalf
	nearRight := g.playerPos.X >= screenW-roomMargin-playerRadius-1 && math.Abs(g.playerPos.Y-screenH/2) <= doorHalf
	nearUp := g.playerPos.Y <= roomMargin+playerRadius+1 && math.Abs(g.playerPos.X-screenW/2) <= doorHalf
	nearDown := g.playerPos.Y >= screenH-roomMargin-playerRadius-1 && math.Abs(g.playerPos.X-screenW/2) <= doorHalf
	switch {
	case nearLeft:
		nextID, ok = g.roomInDir(-1, 0)
		spawn = Vec2{X: screenW - roomMargin - playerRadius - 8, Y: screenH / 2}
	case nearRight:
		nextID, ok = g.roomInDir(1, 0)
		spawn = Vec2{X: roomMargin + playerRadius + 8, Y: screenH / 2}
	case nearUp:
		nextID, ok = g.roomInDir(0, -1)
		spawn = Vec2{X: screenW / 2, Y: screenH - roomMargin - playerRadius - 8}
	case nearDown:
		nextID, ok = g.roomInDir(0, 1)
		spawn = Vec2{X: screenW / 2, Y: roomMargin + playerRadius + 8}
	}
	if !ok {
		return
	}
	g.swapRoom(nextID, spawn)
}

func (g *Game) swapRoom(nextRoomID int, spawn Vec2) {
	g.saveCurrentRoomState()
	g.currentRoomID = nextRoomID
	g.visitedRooms[nextRoomID] = true
	g.runRoomsVisited++
	g.loadCurrentRoom()
	g.playerPos = spawn
	g.swapCooldown = roomSwapCooldown
	g.transitionTick = transitionFramesMax
}

func (g *Game) floorCleared() bool {
	if g.currentRoomID != g.bossRoomID {
		return false
	}
	return g.roomClear
}

func (g *Game) tryDescendFloor() {
	if !g.floorCleared() {
		return
	}
	if distance(g.playerPos, Vec2{X: screenW / 2, Y: screenH / 2}) > 28 {
		return
	}
	if !inpututil.IsKeyJustPressed(ebiten.KeyL) {
		return
	}
	g.startNextFloor()
}

func (g *Game) startNextFloor() {
	g.saveRunTelemetry("floor_clear")
	g.saveCurrentRoomState()
	g.floorsCleared++
	g.floor++
	g.shopRerolls = 0
	g.transitionTick = transitionFramesMax
	g.playerInvFrames = 0
	g.playerHP = minInt(playerMaxHP, g.playerHP+1)

	g.initRoomsProcedural()
	g.visitedRooms = map[int]bool{g.currentRoomID: true}
	g.runRoomsVisited++
	g.loadCurrentRoom()
	g.playerPos = Vec2{X: screenW / 2, Y: screenH / 2}
	g.statusText = fmt.Sprintf("Welcome to Floor %d", g.floor)
	g.statusTextTick = 120
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.shakeTick > 0 {
		g.shakeTick--
	}
	roomTint := color.RGBA{R: 64, G: 50, B: 45, A: 255}
	if g.currentRoom().Type == RoomShop {
		roomTint = color.RGBA{R: 70, G: 58, B: 47, A: 255}
	}
	if g.currentRoom().Type == RoomBoss {
		roomTint = color.RGBA{R: 72, G: 42, B: 40, A: 255}
	}
	screen.Fill(color.RGBA{R: 32, G: 26, B: 24, A: 255})
	vector.DrawFilledRect(screen, float32(roomMargin), float32(roomMargin), float32(screenW-2*roomMargin), float32(screenH-2*roomMargin), roomTint, false)
	vector.StrokeRect(screen, float32(roomMargin), float32(roomMargin), float32(screenW-2*roomMargin), float32(screenH-2*roomMargin), 6, color.RGBA{R: 100, G: 76, B: 68, A: 255}, false)
	g.drawDoors(screen)
	if g.showMiniMap {
		g.drawMiniMap(screen)
	}
	for _, h := range g.hazards {
		drawHazard(screen, h)
	}
	if g.roomClear && !g.currentRoom().Reward.Taken {
		drawItem(screen, g.currentRoom().Reward)
	}
	for _, c := range g.chests {
		drawChest(screen, c)
	}
	for _, o := range g.offers {
		drawOffer(screen, o)
	}
	for _, p := range g.pickups {
		if p.Active {
			drawPickup(screen, p)
		}
	}
	for _, b := range g.bombList {
		if b.Active {
			drawBomb(screen, b)
		}
	}
	for _, ex := range g.explosions {
		drawExplosion(screen, ex)
	}

	playerCol := color.RGBA{R: 220, G: 210, B: 190, A: 255}
	if g.playerInvFrames > 0 && (g.playerInvFrames/4)%2 == 0 {
		playerCol = color.RGBA{R: 250, G: 160, B: 160, A: 255}
	}
	if g.dashFrames > 0 {
		playerCol = color.RGBA{R: 205, G: 245, B: 210, A: 255}
	}
	vector.DrawFilledCircle(screen, float32(g.playerPos.X), float32(g.playerPos.Y), playerRadius, playerCol, false)

	for _, b := range g.bullets {
		vector.DrawFilledCircle(screen, float32(b.Pos.X), float32(b.Pos.Y), bulletRadius, color.RGBA{R: 180, G: 220, B: 255, A: 255}, false)
	}
	for _, s := range g.enemyShots {
		r := float32(enemyShotRadius)
		col := color.RGBA{R: 210, G: 125, B: 95, A: 255}
		if s.FromBoss {
			r = bossShotRadius
			col = color.RGBA{R: 230, G: 110, B: 90, A: 255}
		}
		vector.DrawFilledCircle(screen, float32(s.Pos.X), float32(s.Pos.Y), r, col, false)
	}
	for _, e := range g.enemies {
		if !e.Alive {
			continue
		}
		r := float32(enemyRadius)
		col := color.RGBA{R: 170, G: 70, B: 70, A: 255}
		if e.Kind == EnemyWander {
			col = color.RGBA{R: 190, G: 120, B: 70, A: 255}
		}
		if e.Kind == EnemyShooter {
			col = color.RGBA{R: 145, G: 95, B: 170, A: 255}
		}
		if e.Kind == EnemyBoss {
			r = bossRadius
			col = color.RGBA{R: 145, G: 42, B: 42, A: 255}
			if e.ShootWindup > 0 {
				ringR := float32(bossRadius + 8 + (bossWindupFrames - e.ShootWindup))
				vector.StrokeCircle(screen, float32(e.Pos.X), float32(e.Pos.Y), ringR, 2, color.RGBA{R: 245, G: 120, B: 90, A: 255}, false)
			}
		}
		vector.DrawFilledCircle(screen, float32(e.Pos.X), float32(e.Pos.Y), r, col, false)
	}
	g.drawBossHPBar(screen)

	status := fmt.Sprintf("F:%d HP:%d Bombs:%d Coins:%d Keys:%d Room:%d/%d E:%d Dmg:%d Rate:%d Spd:%.2f Crit:%d%% Score:%d Best:%d Streak:%d", g.floor, g.playerHP, g.bombs, g.coins, g.keys, g.currentRoomID+1, len(g.rooms), g.aliveEnemyCount(), g.shotDamage, g.shotCooldownBase, g.moveSpeed, int(g.critChance*100), g.score, g.bestScore, g.killStreak)
	ebitenutil.DebugPrintAt(screen, status, 18, 14)
	ebitenutil.DebugPrintAt(screen, "Move: WASD Shoot: Arrows Dash: Shift Bomb: E Chest: G Shop: F Reroll: H Pause: P Minimap: M New: N", 18, 34)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Seed:%d Time:%s Runs:%d Deaths:%d Rank:%s", g.runSeed, formatRunTime(g.runFrames), g.runsCompleted, g.deaths, g.runRank()), 18, 54)
	if g.playerHP <= lowHPThreshold && (g.runFrames/20)%2 == 0 {
		ebitenutil.DebugPrintAt(screen, "Low HP", 18, 74)
	}
	if g.roomClear {
		ebitenutil.DebugPrintAt(screen, "Room clear! Doors unlocked.", screenW/2-100, 74)
	}
	if g.statusTextTick > 0 {
		ebitenutil.DebugPrintAt(screen, g.statusText, screenW/2-150, 94)
	}
	if g.currentRoomID == g.bossRoomID {
		ebitenutil.DebugPrintAt(screen, "Boss Room", screenW/2-35, 114)
		if g.bossInPhase2() {
			ebitenutil.DebugPrintAt(screen, "Boss Phase 2!", screenW/2-42, 134)
		}
		if g.bossInPhase3() {
			ebitenutil.DebugPrintAt(screen, "Boss Phase 3!", screenW/2-42, 154)
		}
	}
	if g.currentRoom().Type == RoomShop {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Shop: F buy / H reroll (%d coins)", 2+g.shopRerolls), screenW/2-125, 174)
	}
	if g.allRoomsCleared() {
		ebitenutil.DebugPrintAt(screen, "Dungeon clear! Boss defeated.", screenW/2-90, 194)
	}
	if g.floorCleared() {
		ebitenutil.DebugPrintAt(screen, "Press L on the portal to descend", screenW/2-110, 214)
		vector.DrawFilledCircle(screen, screenW/2, screenH/2, 18, color.RGBA{R: 120, G: 180, B: 220, A: 180}, false)
	}
	if g.itemTextFrames > 0 {
		ebitenutil.DebugPrintAt(screen, g.lastItemText, screenW/2-140, screenH-28)
	}
	if g.paused {
		ebitenutil.DebugPrintAt(screen, "PAUSED", screenW/2-24, screenH/2)
	}
	if g.playerHP <= 0 {
		ebitenutil.DebugPrintAt(screen, "You died. Press R to restart seed, N for new run", screenW/2-170, screenH/2)
	}
	if g.transitionTick > 0 {
		alpha := uint8(float64(g.transitionTick) / float64(transitionFramesMax) * 160)
		vector.DrawFilledRect(screen, 0, 0, screenW, screenH, color.RGBA{R: 10, G: 10, B: 10, A: alpha}, false)
	}
}

func (g *Game) emitEvent(_ string) {
	// Placeholder hook for future SFX integration.
}

func (g *Game) drawDoors(screen *ebiten.Image) {
	if id, ok := g.roomInDir(0, -1); ok {
		vector.DrawFilledRect(screen, float32(screenW/2-doorHalf), float32(roomMargin-3), float32(doorHalf*2), 6, g.doorColorFor(id), false)
	}
	if id, ok := g.roomInDir(0, 1); ok {
		vector.DrawFilledRect(screen, float32(screenW/2-doorHalf), float32(screenH-roomMargin-3), float32(doorHalf*2), 6, g.doorColorFor(id), false)
	}
	if id, ok := g.roomInDir(-1, 0); ok {
		vector.DrawFilledRect(screen, float32(roomMargin-3), float32(screenH/2-doorHalf), 6, float32(doorHalf*2), g.doorColorFor(id), false)
	}
	if id, ok := g.roomInDir(1, 0); ok {
		vector.DrawFilledRect(screen, float32(screenW-roomMargin-3), float32(screenH/2-doorHalf), 6, float32(doorHalf*2), g.doorColorFor(id), false)
	}
}

func (g *Game) doorColorFor(targetRoom int) color.RGBA {
	if g.roomClear {
		return color.RGBA{R: 130, G: 140, B: 95, A: 255}
	}
	return color.RGBA{R: 120, G: 96, B: 80, A: 255}
}

func (g *Game) drawMiniMap(screen *ebiten.Image) {
	minGX, minGY, maxGX, _ := g.gridBounds()
	cell := float32(12)
	gap := float32(4)
	w := float32(maxGX-minGX+1)*(cell+gap) - gap
	x0 := float32(screenW) - w - 18
	y0 := float32(18)
	for id, room := range g.rooms {
		x := x0 + float32(room.GridX-minGX)*(cell+gap)
		y := y0 + float32(room.GridY-minGY)*(cell+gap)
		col := color.RGBA{R: 62, G: 58, B: 55, A: 255}
		if room.Type == RoomShop {
			col = color.RGBA{R: 120, G: 95, B: 70, A: 255}
		}
		if g.visitedRooms[id] {
			col = color.RGBA{R: 120, G: 112, B: 104, A: 255}
		}
		if id == g.currentRoomID {
			col = color.RGBA{R: 175, G: 210, B: 145, A: 255}
		}
		if id == g.bossRoomID {
			col = color.RGBA{R: 145, G: 70, B: 70, A: 255}
		}
		vector.DrawFilledRect(screen, x, y, cell, cell, col, false)
		vector.StrokeRect(screen, x, y, cell, cell, 1, color.RGBA{R: 30, G: 24, B: 24, A: 255}, false)
	}
}

func (g *Game) drawBossHPBar(screen *ebiten.Image) {
	if g.currentRoomID != g.bossRoomID {
		return
	}
	for _, e := range g.enemies {
		if e.Kind != EnemyBoss || !e.Alive {
			continue
		}
		barW := float32(260)
		barH := float32(12)
		x := float32(screenW)/2 - barW/2
		y := float32(70)
		vector.DrawFilledRect(screen, x, y, barW, barH, color.RGBA{R: 45, G: 25, B: 25, A: 255}, false)
		ratio := clamp(float64(e.HP)/float64(bossBaseHP), 0, 1)
		vector.DrawFilledRect(screen, x, y, barW*float32(ratio), barH, color.RGBA{R: 180, G: 68, B: 60, A: 255}, false)
		vector.StrokeRect(screen, x, y, barW, barH, 2, color.RGBA{R: 220, G: 175, B: 165, A: 255}, false)
		return
	}
}

func drawItem(screen *ebiten.Image, item Item) {
	col := color.RGBA{R: 210, G: 210, B: 150, A: 255}
	switch item.Kind {
	case ItemDamage:
		col = color.RGBA{R: 210, G: 90, B: 90, A: 255}
	case ItemFireRate:
		col = color.RGBA{R: 110, G: 170, B: 230, A: 255}
	case ItemSpeed:
		col = color.RGBA{R: 120, G: 210, B: 140, A: 255}
	case ItemHeal:
		col = color.RGBA{R: 230, G: 150, B: 170, A: 255}
	case ItemCrit:
		col = color.RGBA{R: 235, G: 215, B: 105, A: 255}
	}
	s := float32(itemRadius * 2)
	vector.DrawFilledRect(screen, float32(item.Pos.X-itemRadius), float32(item.Pos.Y-itemRadius), s, s, col, false)
	vector.StrokeRect(screen, float32(item.Pos.X-itemRadius), float32(item.Pos.Y-itemRadius), s, s, 2, color.RGBA{R: 40, G: 30, B: 25, A: 255}, false)
}

func drawOffer(screen *ebiten.Image, o ShopOffer) {
	if o.Purchased {
		vector.DrawFilledRect(screen, float32(o.Pos.X-14), float32(o.Pos.Y-14), 28, 28, color.RGBA{R: 55, G: 50, B: 48, A: 255}, false)
		return
	}
	col := color.RGBA{R: 195, G: 170, B: 115, A: 255}
	switch o.Kind {
	case OfferHeart:
		col = color.RGBA{R: 220, G: 125, B: 145, A: 255}
	case OfferBombPack:
		col = color.RGBA{R: 125, G: 125, B: 120, A: 255}
	case OfferDamage:
		col = color.RGBA{R: 225, G: 110, B: 95, A: 255}
	case OfferKey:
		col = color.RGBA{R: 210, G: 180, B: 120, A: 255}
	case OfferCrit:
		col = color.RGBA{R: 230, G: 205, B: 100, A: 255}
	}
	vector.DrawFilledRect(screen, float32(o.Pos.X-14), float32(o.Pos.Y-14), 28, 28, col, false)
	vector.StrokeRect(screen, float32(o.Pos.X-14), float32(o.Pos.Y-14), 28, 28, 2, color.RGBA{R: 35, G: 28, B: 25, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%dc", o.Price), int(o.Pos.X)-10, int(o.Pos.Y)+20)
}

func drawChest(screen *ebiten.Image, c Chest) {
	col := color.RGBA{R: 150, G: 105, B: 65, A: 255}
	if c.Opened {
		col = color.RGBA{R: 95, G: 78, B: 62, A: 255}
	}
	vector.DrawFilledRect(screen, float32(c.Pos.X-14), float32(c.Pos.Y-10), 28, 20, col, false)
	vector.StrokeRect(screen, float32(c.Pos.X-14), float32(c.Pos.Y-10), 28, 20, 2, color.RGBA{R: 45, G: 30, B: 20, A: 255}, false)
}

func drawHazard(screen *ebiten.Image, h Hazard) {
	vector.DrawFilledCircle(screen, float32(h.Pos.X), float32(h.Pos.Y), float32(h.R), color.RGBA{R: 100, G: 48, B: 48, A: 220}, false)
	vector.StrokeCircle(screen, float32(h.Pos.X), float32(h.Pos.Y), float32(h.R), 1.5, color.RGBA{R: 160, G: 82, B: 82, A: 255}, false)
}

func drawPickup(screen *ebiten.Image, p Pickup) {
	col := color.RGBA{R: 220, G: 145, B: 160, A: 255}
	r := float32(pickupRadius)
	switch p.Kind {
	case PickupBomb:
		col = color.RGBA{R: 125, G: 115, B: 100, A: 255}
	case PickupCoin:
		col = color.RGBA{R: 232, G: 194, B: 92, A: 255}
		r = pickupRadius - 1
	case PickupKey:
		col = color.RGBA{R: 210, G: 180, B: 120, A: 255}
	}
	vector.DrawFilledCircle(screen, float32(p.Pos.X), float32(p.Pos.Y), r, col, false)
	vector.StrokeCircle(screen, float32(p.Pos.X), float32(p.Pos.Y), r, 1.5, color.RGBA{R: 30, G: 25, B: 20, A: 255}, false)
}

func drawBomb(screen *ebiten.Image, b Bomb) {
	vector.DrawFilledCircle(screen, float32(b.Pos.X), float32(b.Pos.Y), 8, color.RGBA{R: 55, G: 52, B: 50, A: 255}, false)
	if b.Timer%20 < 10 {
		vector.DrawFilledCircle(screen, float32(b.Pos.X), float32(b.Pos.Y-8), 3, color.RGBA{R: 230, G: 140, B: 80, A: 255}, false)
	}
}

func drawExplosion(screen *ebiten.Image, ex Explosion) {
	ratio := float32(ex.Timer) / float32(explosionFrames)
	r := float32(ex.Radius) * (1 - ratio*0.6)
	col := color.RGBA{R: 250, G: 170, B: 90, A: uint8(180 * ratio)}
	vector.DrawFilledCircle(screen, float32(ex.Pos.X), float32(ex.Pos.Y), r, col, false)
}

func (g *Game) aliveEnemyCount() int {
	n := 0
	for _, e := range g.enemies {
		if e.Alive {
			n++
		}
	}
	return n
}

func (g *Game) allRoomsCleared() bool {
	for id, room := range g.rooms {
		enemies := room.Enemies
		if id == g.currentRoomID {
			enemies = g.enemies
		}
		for _, e := range enemies {
			if e.Alive {
				return false
			}
		}
	}
	return true
}

func (g *Game) bossInPhase2() bool {
	if g.currentRoomID != g.bossRoomID {
		return false
	}
	for _, e := range g.enemies {
		if e.Kind == EnemyBoss && e.Alive {
			return g.isBossPhase2(e)
		}
	}
	return false
}

func (g *Game) bossInPhase3() bool {
	if g.currentRoomID != g.bossRoomID {
		return false
	}
	for _, e := range g.enemies {
		if e.Kind == EnemyBoss && e.Alive {
			return g.isBossPhase3(e)
		}
	}
	return false
}

func (g *Game) enemyDifficultyScale() float64 {
	cleared := 0
	for id, room := range g.rooms {
		if id == g.bossRoomID || room.Type == RoomShop {
			continue
		}
		enemies := room.Enemies
		if id == g.currentRoomID {
			enemies = g.enemies
		}
		alive := false
		for _, e := range enemies {
			if e.Alive {
				alive = true
				break
			}
		}
		if !alive {
			cleared++
		}
	}
	metaScale := 1 + float64(g.runsCompleted)*0.02
	return (1 + float64(cleared)*0.05) * metaScale
}

func (g *Game) runRank() string {
	s := g.score
	switch {
	case s >= 1200:
		return "S"
	case s >= 850:
		return "A"
	case s >= 500:
		return "B"
	case s >= 250:
		return "C"
	default:
		return "D"
	}
}

func (g *Game) metaPath() string { return filepath.Join(".", "save_meta.json") }

func (g *Game) loadMeta() {
	data, err := os.ReadFile(g.metaPath())
	if err != nil {
		return
	}
	var m MetaSave
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	g.bestScore = m.BestScore
	g.runsCompleted = m.RunsCompleted
	g.deaths = m.Deaths
}

func (g *Game) saveMeta() {
	m := MetaSave{BestScore: g.bestScore, RunsCompleted: g.runsCompleted, Deaths: g.deaths}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(g.metaPath(), data, 0644)
}

func (g *Game) telemetryPath() string { return filepath.Join(".", "run_telemetry.jsonl") }

func (g *Game) saveRunTelemetry(result string) {
	entry := RunTelemetry{
		Timestamp:       time.Now().Format(time.RFC3339),
		Seed:            g.runSeed,
		Floor:           g.floor,
		Score:           g.score,
		RoomsVisited:    g.runRoomsVisited,
		EnemiesDefeated: g.killCount,
		DamageTaken:     g.runDamageTaken,
		DamageDealt:     g.runDamageDealt,
		RunSeconds:      g.runFrames / 60,
		Rank:            g.runRank(),
		Result:          result,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	f, err := os.OpenFile(g.telemetryPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

func (g *Game) gridBounds() (int, int, int, int) {
	first := true
	minGX, minGY, maxGX, maxGY := 0, 0, 0, 0
	for _, r := range g.rooms {
		if first {
			minGX, maxGX, minGY, maxGY = r.GridX, r.GridX, r.GridY, r.GridY
			first = false
			continue
		}
		if r.GridX < minGX {
			minGX = r.GridX
		}
		if r.GridX > maxGX {
			maxGX = r.GridX
		}
		if r.GridY < minGY {
			minGY = r.GridY
		}
		if r.GridY > maxGY {
			maxGY = r.GridY
		}
	}
	return minGX, minGY, maxGX, maxGY
}

func (g *Game) Layout(_, _ int) (int, int) { return screenW, screenH }

func formatRunTime(frames int) string {
	total := frames / 60
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func distance(a, b Vec2) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Mini Isaac Prototype (Go + Ebitengine)")
	if err := ebiten.RunGame(NewGame()); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
