use anyhow::{Context, Result};
use clap::{Args, Parser, Subcommand, ValueEnum};
use crossterm::event::{self, Event, KeyCode, KeyEvent, KeyEventKind};
use crossterm::terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen};
use crossterm::{execute, ExecutableCommand};
use rand::Rng;
use ratatui::layout::{Alignment, Constraint, Direction, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::Span;
use ratatui::widgets::{Block, Borders, Clear, List, ListItem, ListState, Paragraph, Wrap};
use ratatui::{DefaultTerminal, Frame};
use serde::{Deserialize, Serialize};
use serde_yaml::Value;
use std::collections::BTreeMap;
use std::fmt;
use std::fs;
use std::io;
use std::path::PathBuf;
use std::time::Duration;

const DEFAULT_ENCOUNTERS_PATH: &str = "encounters.yaml";
const DEFAULT_DICE_PATH: &str = "dice.yaml";
const DEFAULT_BUILD_PATH: &str = "character_build.yaml";
const HELP_TEXT: &str = "q esci | / cerca | tab focus | 0/1/2/3 pannelli | [/] browse | 4..9 browse diretto | a roll dice | Enter aggiungi encounter | N nuovo personaggio | e edit encounter | c condizione | x clear cond | d del encounter / toggle detail | ←/→ hp -/+ | s/l save/load | w/o save/load build | i/I init one/all | S sort init | * turn mode | n/p next/prev turn | u/r undo/redo | M/L treasure | f fullscreen | PgUp/PgDn scroll desc | j/k naviga";

#[derive(Parser, Debug)]
#[command(name = "r5e", version, about = "Rust conversion di lazy5e")]
struct Cli {
    #[command(subcommand)]
    command: Option<Command>,
}

#[derive(Subcommand, Debug)]
enum Command {
    Browse(BrowseArgs),
    Encounters {
        #[command(subcommand)]
        command: EncountersCmd,
    },
    Dice {
        #[command(subcommand)]
        command: DiceCmd,
    },
}

#[derive(Args, Debug)]
struct BrowseArgs {
    #[arg(value_enum)]
    mode: BrowseMode,

    #[arg(long)]
    name: Option<String>,

    #[arg(long)]
    env: Option<String>,

    #[arg(long)]
    source: Option<String>,

    #[arg(long)]
    cr: Option<String>,

    #[arg(long = "type")]
    kind: Option<String>,

    #[arg(long, default_value_t = 25)]
    limit: usize,
}

#[derive(Subcommand, Debug)]
enum EncountersCmd {
    Show(FileArg),
}

#[derive(Subcommand, Debug)]
enum DiceCmd {
    Show(FileArg),
}

#[derive(Args, Debug)]
struct FileArg {
    #[arg(long)]
    path: Option<PathBuf>,
}

#[derive(ValueEnum, Clone, Copy, Debug, Eq, PartialEq, Hash)]
enum BrowseMode {
    Monsters,
    Items,
    Spells,
    Characters,
    Races,
    Feats,
    Books,
    Adventures,
}

impl BrowseMode {
    fn label(self) -> &'static str {
        match self {
            Self::Monsters => "Monsters",
            Self::Items => "Items",
            Self::Spells => "Spells",
            Self::Characters => "Characters",
            Self::Races => "Races",
            Self::Feats => "Feats",
            Self::Books => "Books",
            Self::Adventures => "Adventures",
        }
    }

    fn cycle(self, delta: i8) -> Self {
        const ALL: [BrowseMode; 8] = [
            BrowseMode::Monsters,
            BrowseMode::Items,
            BrowseMode::Spells,
            BrowseMode::Characters,
            BrowseMode::Races,
            BrowseMode::Feats,
            BrowseMode::Books,
            BrowseMode::Adventures,
        ];
        let idx = ALL.iter().position(|m| *m == self).unwrap_or(0) as i32;
        let next = (idx + delta as i32).rem_euclid(ALL.len() as i32) as usize;
        ALL[next]
    }

    fn from_shortcut(c: char) -> Option<Self> {
        match c {
            '4' => Some(Self::Monsters),
            '5' => Some(Self::Items),
            '6' => Some(Self::Spells),
            '7' => Some(Self::Characters),
            '8' => Some(Self::Races),
            '9' => Some(Self::Feats),
            _ => None,
        }
    }
}

#[derive(Debug, Clone)]
struct Record {
    id: i32,
    name: String,
    source: String,
    cr: String,
    kind: String,
    environment: Vec<String>,
    description: String,
    stat_block: String,
    base_hp: i32,
    initiative_mod: i32,
}

#[derive(Debug, Clone)]
struct EncounterRow {
    name: String,
    monster_id: i32,
    ordinal: i32,
    custom: bool,
    custom_name: String,
    current_hp: i32,
    base_hp: i32,
    has_init_roll: bool,
    init_roll: i32,
    conditions: BTreeMap<String, i32>,
    character: Option<CharacterBuild>,
}

#[derive(Debug, Deserialize)]
struct PersistedEncounters {
    #[serde(default)]
    version: Option<u32>,
    #[serde(default)]
    items: Vec<PersistedEncounterItem>,
    #[serde(default)]
    turn_mode: bool,
    #[serde(default)]
    turn_index: i32,
    #[serde(default)]
    turn_round: i32,
}

#[derive(Debug, Deserialize)]
struct PersistedEncounterItem {
    #[serde(default)]
    monster_id: i32,
    #[serde(default)]
    ordinal: i32,
    #[serde(default)]
    custom: bool,
    #[serde(default)]
    custom_name: String,
    #[serde(default)]
    init_rolled: bool,
    #[serde(default)]
    init_roll: i32,
    #[serde(default)]
    conditions: BTreeMap<String, i32>,
    #[serde(default)]
    character: Option<CharacterBuild>,
    #[serde(default)]
    current_hp: i32,
    #[serde(default)]
    base_hp: i32,
}

#[derive(Debug, Serialize)]
struct PersistedEncountersOut {
    version: u32,
    items: Vec<PersistedEncounterItemOut>,
    #[serde(skip_serializing_if = "is_false")]
    turn_mode: bool,
    #[serde(skip_serializing_if = "is_zero_i32")]
    turn_index: i32,
    #[serde(skip_serializing_if = "is_one_i32")]
    turn_round: i32,
}

#[derive(Debug, Serialize)]
struct PersistedEncounterItemOut {
    monster_id: i32,
    ordinal: i32,
    #[serde(skip_serializing_if = "is_false")]
    custom: bool,
    #[serde(skip_serializing_if = "String::is_empty")]
    custom_name: String,
    #[serde(skip_serializing_if = "is_false")]
    init_rolled: bool,
    #[serde(skip_serializing_if = "is_zero_i32")]
    init_roll: i32,
    #[serde(skip_serializing_if = "BTreeMap::is_empty")]
    conditions: BTreeMap<String, i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    character: Option<CharacterBuild>,
    current_hp: i32,
    base_hp: i32,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
struct CharacterBuild {
    #[serde(default)]
    name: String,
    #[serde(default)]
    race: String,
    #[serde(default)]
    class: String,
    #[serde(default)]
    level: i32,
    #[serde(default)]
    hp: i32,
}

#[derive(Debug, Deserialize)]
struct PersistedDice {
    #[serde(default)]
    version: Option<u32>,
    #[serde(default)]
    items: Vec<DiceEntry>,
}

#[derive(Debug, Deserialize)]
struct DiceResult {
    #[serde(default)]
    expression: String,
    #[serde(default)]
    output: String,
}

#[derive(Debug, Deserialize)]
#[serde(untagged)]
enum DiceEntry {
    Structured(DiceResult),
    Legacy(String),
}

#[derive(Debug, Serialize)]
struct PersistedDiceOut {
    version: u32,
    items: Vec<String>,
}

#[derive(Debug, Clone)]
struct EncounterSnapshot {
    items: Vec<EncounterRow>,
    selected: Option<usize>,
    turn_mode: bool,
    turn_index: usize,
    turn_round: i32,
}

#[derive(Debug, Clone)]
struct DiceSnapshot {
    items: Vec<String>,
    selected: Option<usize>,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum ActivePanel {
    Browse,
    Encounters,
    Dice,
    Detail,
}

impl ActivePanel {
    fn cycle(self) -> Self {
        match self {
            Self::Dice => Self::Encounters,
            Self::Encounters => Self::Browse,
            Self::Browse => Self::Detail,
            Self::Detail => Self::Dice,
        }
    }
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum InputMode {
    Normal,
    Search,
    Dice,
    EncounterEdit,
    ConditionEdit,
    CharacterCreate,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum DetailMode {
    Description,
    Treasure,
}

struct App {
    browse_mode: BrowseMode,
    monsters: Vec<Record>,
    items: Vec<Record>,
    spells: Vec<Record>,
    classes: Vec<Record>,
    races: Vec<Record>,
    feats: Vec<Record>,
    books: Vec<Record>,
    adventures: Vec<Record>,

    filtered: Vec<usize>,
    browse_state: ListState,
    encounters: Vec<EncounterRow>,
    encounter_state: ListState,
    dice: Vec<String>,
    dice_state: ListState,

    search_query: String,
    input_mode: InputMode,
    input_buffer: String,
    active_panel: ActivePanel,
    status: String,
    encounters_path: PathBuf,
    dice_path: PathBuf,
    build_path: PathBuf,
    turn_mode: bool,
    turn_index: usize,
    turn_round: i32,
    encounter_undo: Vec<EncounterSnapshot>,
    encounter_redo: Vec<EncounterSnapshot>,
    dice_undo: Vec<DiceSnapshot>,
    dice_redo: Vec<DiceSnapshot>,
    fullscreen_panel: Option<ActivePanel>,
    detail_mode: DetailMode,
    description_scroll: usize,
    treasure_text: String,
}

impl App {
    fn new() -> Result<Self> {
        let encounters_path = std::env::var("ENCOUNTERS_YAML")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| PathBuf::from(DEFAULT_ENCOUNTERS_PATH));
        let dice_path = std::env::var("DICE_YAML")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| PathBuf::from(DEFAULT_DICE_PATH));
        let build_path = std::env::var("BUILD_YAML")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| PathBuf::from(DEFAULT_BUILD_PATH));

        let monsters = parse_dataset("monsters", include_str!("../data/monster.yaml"))?;
        let items = parse_dataset("items", include_str!("../data/item.yaml"))?;
        let spells = parse_dataset("spells", include_str!("../data/spell.yaml"))?;
        let classes = parse_dataset("classes", include_str!("../data/class.yaml"))?;
        let races = parse_dataset("races", include_str!("../data/race.yaml"))?;
        let feats = parse_dataset("feats", include_str!("../data/feat.yaml"))?;
        let books = parse_dataset("books", include_str!("../data/book.yaml"))?;
        let adventures = parse_dataset("adventures", include_str!("../data/adventure.yaml"))?;

        let mut app = Self {
            browse_mode: BrowseMode::Monsters,
            monsters,
            items,
            spells,
            classes,
            races,
            feats,
            books,
            adventures,
            filtered: Vec::new(),
            browse_state: ListState::default(),
            encounters: Vec::new(),
            encounter_state: ListState::default(),
            dice: Vec::new(),
            dice_state: ListState::default(),
            search_query: String::new(),
            input_mode: InputMode::Normal,
            input_buffer: String::new(),
            active_panel: ActivePanel::Browse,
            status: HELP_TEXT.to_string(),
            encounters_path,
            dice_path,
            build_path,
            turn_mode: false,
            turn_index: 0,
            turn_round: 1,
            encounter_undo: Vec::new(),
            encounter_redo: Vec::new(),
            dice_undo: Vec::new(),
            dice_redo: Vec::new(),
            fullscreen_panel: None,
            detail_mode: DetailMode::Description,
            description_scroll: 0,
            treasure_text: "Nessun tesoro generato. Premi M (normal) o L (lair).".to_string(),
        };

        app.load_encounters();
        app.load_dice();
        app.refresh_filter();
        Ok(app)
    }

    fn current_dataset(&self) -> &[Record] {
        match self.browse_mode {
            BrowseMode::Monsters => &self.monsters,
            BrowseMode::Items => &self.items,
            BrowseMode::Spells => &self.spells,
            BrowseMode::Characters => &self.classes,
            BrowseMode::Races => &self.races,
            BrowseMode::Feats => &self.feats,
            BrowseMode::Books => &self.books,
            BrowseMode::Adventures => &self.adventures,
        }
    }

    fn refresh_filter(&mut self) {
        let query = self.search_query.to_lowercase();
        let mut next_filtered = Vec::new();
        for (idx, rec) in self.current_dataset().iter().enumerate() {
            if query.is_empty() {
                next_filtered.push(idx);
                continue;
            }
            let hay = format!(
                "{} {} {} {} {}",
                rec.name,
                rec.source,
                rec.cr,
                rec.kind,
                rec.description
            )
            .to_lowercase();
            if hay.contains(&query) {
                next_filtered.push(idx);
            }
        }
        self.filtered = next_filtered;
        if self.filtered.is_empty() {
            self.browse_state.select(None);
            self.status = format!("nessun risultato per \"{}\"", self.search_query);
        } else {
            let current = self.browse_state.selected().unwrap_or(0);
            self.browse_state
                .select(Some(current.min(self.filtered.len().saturating_sub(1))));
            self.status = format!("{} risultati", self.filtered.len());
        }
    }

    fn load_encounters(&mut self) {
        let content = match fs::read_to_string(&self.encounters_path) {
            Ok(v) => v,
            Err(_) => return,
        };
        let persisted: PersistedEncounters = match serde_yaml::from_str(&content) {
            Ok(v) => v,
            Err(e) => {
                self.status = format!("errore parse encounters: {e}");
                return;
            }
        };

        self.encounters.clear();
        for it in persisted.items {
            let custom_name = it.custom_name;
            let name = if it.custom && !custom_name.is_empty() {
                custom_name.clone()
            } else if let Some(ch) = &it.character {
                ch.name.clone()
            } else {
                self.monsters
                    .get(it.monster_id as usize)
                    .map(|m| m.name.clone())
                    .unwrap_or_else(|| format!("monster #{}", it.monster_id))
            };
            self.encounters.push(EncounterRow {
                name,
                monster_id: it.monster_id,
                ordinal: it.ordinal,
                custom: it.custom,
                custom_name,
                current_hp: it.current_hp,
                base_hp: it.base_hp,
                has_init_roll: it.init_rolled,
                init_roll: it.init_roll,
                conditions: it.conditions,
                character: it.character,
            });
        }
        self.turn_mode = persisted.turn_mode;
        self.turn_round = persisted.turn_round.max(1);
        self.turn_index = if self.encounters.is_empty() {
            0
        } else {
            (persisted.turn_index.max(0) as usize).min(self.encounters.len().saturating_sub(1))
        };
        if !self.encounters.is_empty() {
            if self.turn_mode {
                self.encounter_state.select(Some(self.turn_index));
            } else {
                self.encounter_state.select(Some(0));
            }
        } else {
            self.encounter_state.select(None);
        }
        self.encounter_undo.clear();
        self.encounter_redo.clear();
        if let Some(v) = persisted.version {
            self.status = format!("encounters caricati (v{v})");
        }
    }

    fn load_dice(&mut self) {
        let content = match fs::read_to_string(&self.dice_path) {
            Ok(v) => v,
            Err(_) => return,
        };
        let persisted: PersistedDice = match serde_yaml::from_str(&content) {
            Ok(v) => v,
            Err(e) => {
                self.status = format!("errore parse dice: {e}");
                return;
            }
        };
        self.dice.clear();
        for d in persisted.items {
            match d {
                DiceEntry::Structured(v) => self.dice.push(format!("{} => {}", v.expression, v.output)),
                DiceEntry::Legacy(v) => self.dice.push(v),
            }
        }
        if !self.dice.is_empty() {
            self.dice_state.select(Some(0));
        } else {
            self.dice_state.select(None);
        }
        if let Some(v) = persisted.version {
            self.status = format!("dice caricati (v{v})");
        }
    }

    fn save_encounters(&mut self) -> Result<()> {
        let out = PersistedEncountersOut {
            version: 1,
            items: self
                .encounters
                .iter()
                .map(|it| PersistedEncounterItemOut {
                    monster_id: it.monster_id,
                    ordinal: it.ordinal,
                    custom: it.custom,
                    custom_name: it.custom_name.clone(),
                    init_rolled: it.has_init_roll,
                    init_roll: it.init_roll,
                    conditions: it.conditions.clone(),
                    character: it.character.clone(),
                    current_hp: it.current_hp,
                    base_hp: it.base_hp,
                })
                .collect(),
            turn_mode: self.turn_mode,
            turn_index: self.turn_index as i32,
            turn_round: self.turn_round,
        };
        let yaml = serde_yaml::to_string(&out)?;
        fs::write(&self.encounters_path, yaml).with_context(|| {
            format!(
                "errore scrittura encounters {}",
                self.encounters_path.display()
            )
        })?;
        self.status = format!("salvato {}", self.encounters_path.display());
        Ok(())
    }

    fn save_dice(&mut self) -> Result<()> {
        let out = PersistedDiceOut {
            version: 1,
            items: self.dice.clone(),
        };
        let yaml = serde_yaml::to_string(&out)?;
        fs::write(&self.dice_path, yaml)
            .with_context(|| format!("errore scrittura dice {}", self.dice_path.display()))?;
        self.status = format!("salvato {}", self.dice_path.display());
        Ok(())
    }

    fn load_encounters_with_undo(&mut self) {
        self.push_encounter_undo();
        self.load_encounters();
    }

    fn load_dice_with_undo(&mut self) {
        self.push_dice_undo();
        self.load_dice();
    }

    fn encounter_snapshot(&self) -> EncounterSnapshot {
        EncounterSnapshot {
            items: self.encounters.clone(),
            selected: self.encounter_state.selected(),
            turn_mode: self.turn_mode,
            turn_index: self.turn_index,
            turn_round: self.turn_round,
        }
    }

    fn restore_encounter_snapshot(&mut self, snapshot: EncounterSnapshot) {
        self.encounters = snapshot.items;
        self.turn_mode = snapshot.turn_mode;
        self.turn_index = snapshot.turn_index.min(self.encounters.len().saturating_sub(1));
        self.turn_round = snapshot.turn_round.max(1);
        if self.encounters.is_empty() {
            self.encounter_state.select(None);
            self.turn_index = 0;
        } else {
            let selected = snapshot
                .selected
                .unwrap_or(0)
                .min(self.encounters.len().saturating_sub(1));
            self.encounter_state.select(Some(selected));
        }
    }

    fn push_encounter_undo(&mut self) {
        self.encounter_undo.push(self.encounter_snapshot());
        if self.encounter_undo.len() > 100 {
            self.encounter_undo.remove(0);
        }
        self.encounter_redo.clear();
    }

    fn undo_encounter(&mut self) {
        let Some(prev) = self.encounter_undo.pop() else {
            self.status = "nessuna operazione encounter da annullare".to_string();
            return;
        };
        self.encounter_redo.push(self.encounter_snapshot());
        self.restore_encounter_snapshot(prev);
        self.status = "undo encounter".to_string();
    }

    fn redo_encounter(&mut self) {
        let Some(next) = self.encounter_redo.pop() else {
            self.status = "nessuna operazione encounter da ripristinare".to_string();
            return;
        };
        self.encounter_undo.push(self.encounter_snapshot());
        self.restore_encounter_snapshot(next);
        self.status = "redo encounter".to_string();
    }

    fn dice_snapshot(&self) -> DiceSnapshot {
        DiceSnapshot {
            items: self.dice.clone(),
            selected: self.dice_state.selected(),
        }
    }

    fn restore_dice_snapshot(&mut self, snapshot: DiceSnapshot) {
        self.dice = snapshot.items;
        if self.dice.is_empty() {
            self.dice_state.select(None);
        } else {
            self.dice_state
                .select(Some(snapshot.selected.unwrap_or(0).min(self.dice.len() - 1)));
        }
    }

    fn push_dice_undo(&mut self) {
        self.dice_undo.push(self.dice_snapshot());
        if self.dice_undo.len() > 100 {
            self.dice_undo.remove(0);
        }
        self.dice_redo.clear();
    }

    fn undo_dice(&mut self) {
        let Some(prev) = self.dice_undo.pop() else {
            self.status = "nessuna operazione dice da annullare".to_string();
            return;
        };
        self.dice_redo.push(self.dice_snapshot());
        self.restore_dice_snapshot(prev);
        self.status = "undo dice".to_string();
    }

    fn redo_dice(&mut self) {
        let Some(next) = self.dice_redo.pop() else {
            self.status = "nessuna operazione dice da ripristinare".to_string();
            return;
        };
        self.dice_undo.push(self.dice_snapshot());
        self.restore_dice_snapshot(next);
        self.status = "redo dice".to_string();
    }

    fn toggle_turn_mode(&mut self) {
        if self.encounters.is_empty() {
            self.status = "turn mode: encounters vuoto".to_string();
            return;
        }
        self.push_encounter_undo();
        self.turn_mode = !self.turn_mode;
        if self.turn_mode {
            self.turn_round = 1;
            self.turn_index = self.encounter_state.selected().unwrap_or(0);
            self.status = "turn mode attivo".to_string();
        } else {
            self.status = "turn mode disattivato".to_string();
        }
    }

    fn turn_step(&mut self, delta: i32) {
        if !self.turn_mode || self.encounters.is_empty() {
            return;
        }
        let len = self.encounters.len() as i32;
        let prev = self.turn_index as i32;
        let mut next = (prev + delta).rem_euclid(len);
        if delta > 0 && next == 0 && prev == len - 1 {
            self.turn_round += 1;
            self.decay_conditions_round();
        } else if delta < 0 && next == len - 1 && prev == 0 {
            self.turn_round = (self.turn_round - 1).max(1);
        }
        if next < 0 {
            next += len;
        }
        self.turn_index = next as usize;
        self.encounter_state.select(Some(self.turn_index));
        self.status = format!("turn round {} entry {}", self.turn_round, self.turn_index + 1);
    }

    fn decay_conditions_round(&mut self) {
        for row in &mut self.encounters {
            let mut remove = Vec::new();
            for (k, v) in &mut row.conditions {
                if *v > 0 {
                    *v -= 1;
                }
                if *v <= 0 {
                    remove.push(k.clone());
                }
            }
            for k in remove {
                row.conditions.remove(&k);
            }
        }
    }

    fn roll_selected_init(&mut self) {
        let Some(idx) = self.encounter_state.selected() else {
            return;
        };
        let Some(snapshot_row) = self.encounters.get(idx).cloned() else {
            return;
        };
        self.push_encounter_undo();
        let modif = self.initiative_mod_for_entry(snapshot_row.custom, snapshot_row.monster_id);
        let roll = rand::rng().random_range(1..=20);
        let Some(row) = self.encounters.get_mut(idx) else {
            return;
        };
        row.has_init_roll = true;
        row.init_roll = roll + modif;
        self.status = format!("initiative {} #{} = {}", row.name, row.ordinal, row.init_roll);
    }

    fn roll_all_init(&mut self) {
        if self.encounters.is_empty() {
            return;
        }
        self.push_encounter_undo();
        for idx in 0..self.encounters.len() {
            let modif = {
                let row = &self.encounters[idx];
                self.initiative_mod_for_entry(row.custom, row.monster_id)
            };
            let roll = rand::rng().random_range(1..=20);
            self.encounters[idx].has_init_roll = true;
            self.encounters[idx].init_roll = roll + modif;
        }
        self.status = format!("initiative tirata per {} entry", self.encounters.len());
    }

    fn sort_encounters_by_init(&mut self) {
        if self.encounters.is_empty() {
            return;
        }
        self.push_encounter_undo();
        self.encounters.sort_by(|a, b| {
            b.has_init_roll
                .cmp(&a.has_init_roll)
                .then_with(|| b.init_roll.cmp(&a.init_roll))
                .then_with(|| a.name.cmp(&b.name))
        });
        self.encounter_state.select(Some(0));
        if self.turn_mode {
            self.turn_index = 0;
            self.turn_round = 1;
        }
        self.status = "encounters ordinati per iniziativa".to_string();
    }

    fn initiative_mod_for_entry(&self, custom: bool, monster_id: i32) -> i32 {
        if custom {
            return 0;
        }
        self.monsters
            .get(monster_id.max(0) as usize)
            .map(|m| m.initiative_mod)
            .unwrap_or(0)
    }

    fn add_selected_monster_to_encounter(&mut self) {
        if self.browse_mode != BrowseMode::Monsters {
            self.status = "aggiunta encounter disponibile solo da Monsters".to_string();
            return;
        }
        let Some(rec) = self.selected_record().cloned() else {
            return;
        };
        self.push_encounter_undo();
        let ordinal = self
            .encounters
            .iter()
            .filter(|e| !e.custom && e.monster_id == rec.id)
            .map(|e| e.ordinal)
            .max()
            .unwrap_or(0)
            + 1;
        self.encounters.push(EncounterRow {
            name: rec.name.clone(),
            monster_id: rec.id,
            ordinal,
            custom: false,
            custom_name: String::new(),
            current_hp: rec.base_hp,
            base_hp: rec.base_hp,
            has_init_roll: false,
            init_roll: 0,
            conditions: BTreeMap::new(),
            character: None,
        });
        self.encounter_state
            .select(Some(self.encounters.len().saturating_sub(1)));
        if self.turn_mode {
            self.turn_index = self.encounters.len().saturating_sub(1);
        }
        self.active_panel = ActivePanel::Encounters;
        self.status = format!("aggiunto {} #{}", rec.name, ordinal);
    }

    fn delete_selected_encounter(&mut self) {
        let Some(idx) = self.encounter_state.selected() else {
            return;
        };
        if idx >= self.encounters.len() {
            return;
        }
        self.push_encounter_undo();
        let removed_before_turn = self.turn_mode && idx < self.turn_index;
        let removed = self.encounters.remove(idx);
        if self.encounters.is_empty() {
            self.encounter_state.select(None);
            self.turn_index = 0;
            self.turn_mode = false;
        } else {
            if removed_before_turn {
                self.turn_index = self.turn_index.saturating_sub(1);
            }
            self.encounter_state
                .select(Some(idx.min(self.encounters.len().saturating_sub(1))));
            self.turn_index = self
                .turn_index
                .min(self.encounters.len().saturating_sub(1));
        }
        self.status = format!("eliminato {} #{}", removed.name, removed.ordinal);
    }

    fn adjust_selected_encounter_hp(&mut self, delta: i32) {
        let Some(idx) = self.encounter_state.selected() else {
            return;
        };
        self.push_encounter_undo();
        let Some(row) = self.encounters.get_mut(idx) else {
            return;
        };
        row.current_hp = (row.current_hp + delta).max(0);
        self.status = format!("{} #{} hp={}", row.name, row.ordinal, row.current_hp);
    }

    fn delete_selected_dice(&mut self) {
        let Some(idx) = self.dice_state.selected() else {
            return;
        };
        if idx >= self.dice.len() {
            return;
        }
        self.push_dice_undo();
        self.dice.remove(idx);
        if self.dice.is_empty() {
            self.dice_state.select(None);
        } else {
            self.dice_state
                .select(Some(idx.min(self.dice.len().saturating_sub(1))));
        }
        self.status = "dice eliminato".to_string();
    }

    fn open_encounter_edit(&mut self) {
        let Some(idx) = self.encounter_state.selected() else {
            return;
        };
        let Some(row) = self.encounters.get(idx) else {
            return;
        };
        self.input_mode = InputMode::EncounterEdit;
        self.input_buffer = format!("{};{};{}", row.name, row.current_hp, row.base_hp);
    }

    fn open_condition_edit(&mut self) {
        if self.encounter_state.selected().is_none() {
            return;
        }
        self.input_mode = InputMode::ConditionEdit;
        self.input_buffer.clear();
    }

    fn open_character_create(&mut self) {
        self.input_mode = InputMode::CharacterCreate;
        self.input_buffer.clear();
    }

    fn apply_character_create(&mut self, payload: &str) {
        let Ok(build) = parse_character_payload(payload) else {
            self.status = "nuovo personaggio: formato nome;razza;classe;livello;hp".to_string();
            return;
        };
        self.push_encounter_undo();
        let hp = build.hp.max(1);
        self.encounters.push(EncounterRow {
            name: build.name.clone(),
            monster_id: 0,
            ordinal: 1,
            custom: true,
            custom_name: build.name.clone(),
            current_hp: hp,
            base_hp: hp,
            has_init_roll: false,
            init_roll: 0,
            conditions: BTreeMap::new(),
            character: Some(build),
        });
        self.encounter_state
            .select(Some(self.encounters.len().saturating_sub(1)));
        self.active_panel = ActivePanel::Encounters;
        self.status = "personaggio aggiunto a encounters".to_string();
    }

    fn save_selected_build(&mut self) -> Result<()> {
        let Some(idx) = self.encounter_state.selected() else {
            self.status = "nessun encounter selezionato".to_string();
            return Ok(());
        };
        let Some(row) = self.encounters.get(idx) else {
            return Ok(());
        };
        let build = if let Some(ch) = &row.character {
            ch.clone()
        } else {
            CharacterBuild {
                name: row.name.clone(),
                race: String::new(),
                class: String::new(),
                level: 1,
                hp: row.base_hp,
            }
        };
        let yaml = serde_yaml::to_string(&build)?;
        fs::write(&self.build_path, yaml)
            .with_context(|| format!("errore save build {}", self.build_path.display()))?;
        self.status = format!("build salvato {}", self.build_path.display());
        Ok(())
    }

    fn load_build_into_encounter(&mut self) -> Result<()> {
        let content = fs::read_to_string(&self.build_path)
            .with_context(|| format!("errore load build {}", self.build_path.display()))?;
        let build: CharacterBuild = serde_yaml::from_str(&content)
            .with_context(|| format!("build yaml non valido {}", self.build_path.display()))?;
        if build.name.trim().is_empty() {
            self.status = "build non valido (name vuoto)".to_string();
            return Ok(());
        }
        self.push_encounter_undo();
        let hp = build.hp.max(1);
        self.encounters.push(EncounterRow {
            name: build.name.clone(),
            monster_id: 0,
            ordinal: 1,
            custom: true,
            custom_name: build.name.clone(),
            current_hp: hp,
            base_hp: hp,
            has_init_roll: false,
            init_roll: 0,
            conditions: BTreeMap::new(),
            character: Some(build),
        });
        self.encounter_state
            .select(Some(self.encounters.len().saturating_sub(1)));
        self.active_panel = ActivePanel::Encounters;
        self.status = format!("build caricato {}", self.build_path.display());
        Ok(())
    }

    fn apply_encounter_edit(&mut self, payload: &str) {
        let Some(idx) = self.encounter_state.selected() else {
            return;
        };
        let parts: Vec<&str> = payload.split(';').collect();
        if parts.len() < 3 {
            self.status = "edit encounter: formato atteso nome;current_hp;base_hp".to_string();
            return;
        }
        let name = parts[0].trim();
        let Ok(current_hp) = parts[1].trim().parse::<i32>() else {
            self.status = "edit encounter: current_hp non valido".to_string();
            return;
        };
        let Ok(base_hp) = parts[2].trim().parse::<i32>() else {
            self.status = "edit encounter: base_hp non valido".to_string();
            return;
        };
        if name.is_empty() || current_hp < 0 || base_hp <= 0 {
            self.status = "edit encounter: valori non validi".to_string();
            return;
        }
        self.push_encounter_undo();
        let Some(row) = self.encounters.get_mut(idx) else {
            return;
        };
        row.name = name.to_string();
        row.current_hp = current_hp;
        row.base_hp = base_hp;
        if row.custom {
            row.custom_name = row.name.clone();
        }
        if let Some(ch) = &mut row.character {
            ch.name = row.name.clone();
            ch.hp = row.base_hp;
            if parts.len() >= 6 {
                ch.race = parts[3].trim().to_string();
                ch.class = parts[4].trim().to_string();
                if let Ok(level) = parts[5].trim().parse::<i32>() {
                    ch.level = level.max(1);
                }
            }
        }
        self.status = format!("encounter aggiornato: {}", row.name);
    }

    fn apply_condition_edit(&mut self, payload: &str) {
        let Some(idx) = self.encounter_state.selected() else {
            return;
        };
        let parsed = parse_condition_payload(payload);
        let Ok((code, rounds)) = parsed else {
            self.status = "condizione non valida".to_string();
            return;
        };
        self.push_encounter_undo();
        let Some(row) = self.encounters.get_mut(idx) else {
            return;
        };
        if row.conditions.contains_key(&code) {
            row.conditions.remove(&code);
            self.status = format!("condizione rimossa: {}", code);
        } else {
            row.conditions.insert(code.clone(), rounds);
            self.status = format!("condizione aggiunta: {}({})", code, rounds);
        }
    }

    fn clear_selected_conditions(&mut self) {
        let Some(idx) = self.encounter_state.selected() else {
            return;
        };
        let Some(row) = self.encounters.get(idx) else {
            return;
        };
        if row.conditions.is_empty() {
            return;
        }
        self.push_encounter_undo();
        if let Some(row) = self.encounters.get_mut(idx) {
            row.conditions.clear();
        }
        self.status = "condizioni rimosse".to_string();
    }

    fn toggle_detail_mode(&mut self) {
        self.detail_mode = match self.detail_mode {
            DetailMode::Description => DetailMode::Treasure,
            DetailMode::Treasure => DetailMode::Description,
        };
        self.description_scroll = 0;
    }

    fn detail_text(&self) -> String {
        match self.detail_mode {
            DetailMode::Description => self.description_text(),
            DetailMode::Treasure => self.treasure_text.clone(),
        }
    }

    fn description_text(&self) -> String {
        if let Some(rec) = self.selected_record() {
            let mut parts = Vec::new();
            parts.push(format!(
                "{}\nsource: {}\ncr: {}\ntype: {}\nenv: {}",
                rec.name,
                rec.source,
                rec.cr,
                rec.kind,
                rec.environment.join(", ")
            ));
            if !rec.stat_block.is_empty() {
                parts.push(format!("STATISTICHE\n{}", rec.stat_block));
            }
            if !rec.description.is_empty() {
                parts.push(format!("TESTO YAML\n{}", rec.description));
            }
            parts.join("\n\n")
        } else {
            "Nessun elemento selezionato".to_string()
        }
    }

    fn generate_treasure(&mut self, lair: bool) {
        let Some(rec) = self.selected_record() else {
            self.status = "nessun elemento selezionato".to_string();
            return;
        };
        let cr = rec.cr.clone();
        let name = rec.name.clone();
        let band = cr_to_band(&cr);
        let mut rng = rand::rng();
        let mul = if lair { 5 } else { 1 };
        let cp = match band {
            0 => rng.random_range(0..=80) * mul,
            1 => rng.random_range(0..=500) * mul,
            2 => rng.random_range(0..=1200) * mul,
            _ => rng.random_range(0..=3000) * mul,
        };
        let sp = match band {
            0 => rng.random_range(0..=30) * mul,
            1 => rng.random_range(0..=250) * mul,
            2 => rng.random_range(0..=800) * mul,
            _ => rng.random_range(0..=2000) * mul,
        };
        let gp = match band {
            0 => rng.random_range(5..=60) * mul,
            1 => rng.random_range(50..=500) * mul,
            2 => rng.random_range(400..=3500) * mul,
            _ => rng.random_range(3000..=25000) * mul,
        };
        let pp = match band {
            0 => 0,
            1 => rng.random_range(0..=20) * mul,
            2 => rng.random_range(0..=200) * mul,
            _ => rng.random_range(50..=1200) * mul,
        };
        let kind = if lair { "Lair Treasure" } else { "Treasure" };
        self.treasure_text = format!(
            "{}\n\nTarget: {}\nCR: {}\nBand: {}\n\nCoins:\n- {} cp\n- {} sp\n- {} gp\n- {} pp\n\nNote:\n- Generazione rapida ispirata alle bande CR.\n- Premi d per tornare alla Description.",
            kind,
            name,
            cr,
            band_label(band),
            cp,
            sp,
            gp,
            pp
        );
        self.detail_mode = DetailMode::Treasure;
        self.description_scroll = 0;
        self.status = format!("{} generato", kind.to_lowercase());
    }

    fn scrolled_text(&self, text: &str) -> String {
        let lines: Vec<&str> = text.lines().collect();
        if lines.is_empty() {
            return String::new();
        }
        let start = self.description_scroll.min(lines.len().saturating_sub(1));
        lines[start..].join("\n")
    }

    fn selected_record(&self) -> Option<&Record> {
        let idx = self.browse_state.selected()?;
        let rec_idx = *self.filtered.get(idx)?;
        self.current_dataset().get(rec_idx)
    }

    fn on_key(&mut self, key: KeyEvent) -> Result<bool> {
        if key.kind != KeyEventKind::Press {
            return Ok(false);
        }

        match self.input_mode {
            InputMode::Normal => self.handle_normal(key),
            InputMode::Search => {
                self.handle_prompt(key, InputMode::Search)?;
                Ok(false)
            }
            InputMode::Dice => {
                self.handle_prompt(key, InputMode::Dice)?;
                Ok(false)
            }
            InputMode::EncounterEdit => {
                self.handle_prompt(key, InputMode::EncounterEdit)?;
                Ok(false)
            }
            InputMode::ConditionEdit => {
                self.handle_prompt(key, InputMode::ConditionEdit)?;
                Ok(false)
            }
            InputMode::CharacterCreate => {
                self.handle_prompt(key, InputMode::CharacterCreate)?;
                Ok(false)
            }
        }
    }

    fn handle_normal(&mut self, key: KeyEvent) -> Result<bool> {
        match key.code {
            KeyCode::Char('q') => {
                let _ = self.save_encounters();
                let _ = self.save_dice();
                return Ok(true);
            }
            KeyCode::Tab => self.active_panel = self.active_panel.cycle(),
            KeyCode::Char('0') => self.active_panel = ActivePanel::Dice,
            KeyCode::Char('1') => self.active_panel = ActivePanel::Encounters,
            KeyCode::Char('2') => self.active_panel = ActivePanel::Browse,
            KeyCode::Char('3') => self.active_panel = ActivePanel::Detail,
            KeyCode::Char('j') | KeyCode::Down => self.move_selection(1),
            KeyCode::Char('k') | KeyCode::Up => self.move_selection(-1),
            KeyCode::Char('[') => {
                self.browse_mode = self.browse_mode.cycle(-1);
                self.refresh_filter();
            }
            KeyCode::Char(']') => {
                self.browse_mode = self.browse_mode.cycle(1);
                self.refresh_filter();
            }
            KeyCode::Char(c) if BrowseMode::from_shortcut(c).is_some() => {
                self.browse_mode = BrowseMode::from_shortcut(c).unwrap_or(self.browse_mode);
                self.refresh_filter();
            }
            KeyCode::Char('/') => {
                self.input_mode = InputMode::Search;
                self.input_buffer = self.search_query.clone();
            }
            KeyCode::Char('a') => {
                self.input_mode = InputMode::Dice;
                self.input_buffer.clear();
            }
            KeyCode::Char('N') => self.open_character_create(),
            KeyCode::Char('e') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.open_encounter_edit();
                }
            }
            KeyCode::Char('c') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.open_condition_edit();
                }
            }
            KeyCode::Char('x') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.clear_selected_conditions();
                }
            }
            KeyCode::Enter => {
                if self.active_panel == ActivePanel::Browse {
                    self.add_selected_monster_to_encounter();
                }
            }
            KeyCode::Char('d') => match self.active_panel {
                ActivePanel::Encounters => self.delete_selected_encounter(),
                ActivePanel::Dice => self.delete_selected_dice(),
                ActivePanel::Detail => self.toggle_detail_mode(),
                _ => {}
            },
            KeyCode::Left => {
                if self.active_panel == ActivePanel::Encounters {
                    self.adjust_selected_encounter_hp(-1);
                }
            }
            KeyCode::Right => {
                if self.active_panel == ActivePanel::Encounters {
                    self.adjust_selected_encounter_hp(1);
                }
            }
            KeyCode::Char('s') => match self.active_panel {
                ActivePanel::Encounters => {
                    self.save_encounters()?;
                }
                ActivePanel::Dice => {
                    self.save_dice()?;
                }
                _ => {}
            },
            KeyCode::Char('w') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.save_selected_build()?;
                }
            }
            KeyCode::Char('o') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.load_build_into_encounter()?;
                }
            }
            KeyCode::Char('l') => match self.active_panel {
                ActivePanel::Encounters => self.load_encounters_with_undo(),
                ActivePanel::Dice => self.load_dice_with_undo(),
                _ => {}
            },
            KeyCode::Char('*') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.toggle_turn_mode();
                }
            }
            KeyCode::Char('n') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.turn_step(1);
                }
            }
            KeyCode::Char('p') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.turn_step(-1);
                }
            }
            KeyCode::Char('i') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.roll_selected_init();
                }
            }
            KeyCode::Char('I') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.roll_all_init();
                }
            }
            KeyCode::Char('S') => {
                if self.active_panel == ActivePanel::Encounters {
                    self.sort_encounters_by_init();
                }
            }
            KeyCode::Char('u') => match self.active_panel {
                ActivePanel::Encounters => self.undo_encounter(),
                ActivePanel::Dice => self.undo_dice(),
                _ => {}
            },
            KeyCode::Char('r') => match self.active_panel {
                ActivePanel::Encounters => self.redo_encounter(),
                ActivePanel::Dice => self.redo_dice(),
                _ => {}
            },
            KeyCode::Char('M') => {
                if self.active_panel == ActivePanel::Detail || self.active_panel == ActivePanel::Browse
                {
                    self.generate_treasure(false);
                }
            }
            KeyCode::Char('L') => {
                if self.active_panel == ActivePanel::Detail || self.active_panel == ActivePanel::Browse
                {
                    self.generate_treasure(true);
                }
            }
            KeyCode::Char('f') => {
                if self.fullscreen_panel.is_some() {
                    self.fullscreen_panel = None;
                    self.status = "fullscreen disattivato".to_string();
                } else {
                    self.fullscreen_panel = Some(self.active_panel);
                    self.status = format!("fullscreen {:?}", self.active_panel);
                }
            }
            KeyCode::PageUp => {
                if self.active_panel == ActivePanel::Detail {
                    self.description_scroll = self.description_scroll.saturating_sub(8);
                }
            }
            KeyCode::PageDown => {
                if self.active_panel == ActivePanel::Detail {
                    self.description_scroll = self.description_scroll.saturating_add(8);
                }
            }
            _ => {}
        }
        Ok(false)
    }

    fn handle_prompt(&mut self, key: KeyEvent, mode: InputMode) -> Result<()> {
        match key.code {
            KeyCode::Esc => {
                self.input_mode = InputMode::Normal;
                self.input_buffer.clear();
            }
            KeyCode::Enter => {
                let value = self.input_buffer.trim().to_string();
                self.input_mode = InputMode::Normal;
                self.input_buffer.clear();
                match mode {
                    InputMode::Search => {
                        self.search_query = value;
                        self.refresh_filter();
                    }
                    InputMode::Dice => {
                        if value.is_empty() {
                            self.status = "roll annullato".to_string();
                        } else {
                            self.push_dice_undo();
                            let line = roll_expression(&value)?;
                            self.dice.push(line.clone());
                            self.dice_state.select(Some(self.dice.len().saturating_sub(1)));
                            self.status = format!("dice: {line}");
                        }
                    }
                    InputMode::EncounterEdit => {
                        self.apply_encounter_edit(&value);
                    }
                    InputMode::ConditionEdit => {
                        self.apply_condition_edit(&value);
                    }
                    InputMode::CharacterCreate => {
                        self.apply_character_create(&value);
                    }
                    InputMode::Normal => {}
                }
            }
            KeyCode::Backspace => {
                self.input_buffer.pop();
            }
            KeyCode::Char(c) => self.input_buffer.push(c),
            _ => {}
        }
        Ok(())
    }

    fn move_selection(&mut self, delta: i32) {
        match self.active_panel {
            ActivePanel::Browse => {
                if self.filtered.is_empty() {
                    return;
                }
                let len = self.filtered.len();
                let current = self.browse_state.selected().unwrap_or(0) as i32;
                let next = (current + delta).clamp(0, (len - 1) as i32) as usize;
                self.browse_state.select(Some(next));
            }
            ActivePanel::Encounters => {
                if self.encounters.is_empty() {
                    return;
                }
                let len = self.encounters.len();
                let current = self.encounter_state.selected().unwrap_or(0) as i32;
                let next = (current + delta).clamp(0, (len - 1) as i32) as usize;
                self.encounter_state.select(Some(next));
                if self.turn_mode {
                    self.turn_index = next;
                }
            }
            ActivePanel::Dice => {
                if self.dice.is_empty() {
                    return;
                }
                let len = self.dice.len();
                let current = self.dice_state.selected().unwrap_or(0) as i32;
                let next = (current + delta).clamp(0, (len - 1) as i32) as usize;
                self.dice_state.select(Some(next));
            }
            ActivePanel::Detail => {}
        }
    }
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    match cli.command {
        None => run_tui(),
        Some(Command::Browse(args)) => browse(args),
        Some(Command::Encounters { command }) => match command {
            EncountersCmd::Show(arg) => show_encounters(arg),
        },
        Some(Command::Dice { command }) => match command {
            DiceCmd::Show(arg) => show_dice(arg),
        },
    }
}

fn run_tui() -> Result<()> {
    let mut app = App::new()?;

    enable_raw_mode().context("enable raw mode")?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen).context("enter alt screen")?;
    let mut terminal = ratatui::Terminal::new(ratatui::backend::CrosstermBackend::new(stdout))
        .context("init terminal")?;

    let result = tui_loop(&mut terminal, &mut app);

    disable_raw_mode().ok();
    io::stdout().execute(LeaveAlternateScreen).ok();

    result
}

fn tui_loop(terminal: &mut DefaultTerminal, app: &mut App) -> Result<()> {
    loop {
        terminal.draw(|frame| render_ui(frame, app))?;

        if event::poll(Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if app.on_key(key)? {
                    return Ok(());
                }
            }
        }
    }
}

fn render_ui(frame: &mut Frame<'_>, app: &mut App) {
    let root = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(1), Constraint::Length(1)])
        .split(frame.area());

    if let Some(panel) = app.fullscreen_panel {
        render_fullscreen_panel(frame, app, root[0], panel);
        let status = Paragraph::new(app.status.as_str())
            .block(Block::default().borders(Borders::TOP))
            .alignment(Alignment::Left)
            .style(Style::default().fg(Color::White));
        frame.render_widget(status, root[1]);
        return;
    }

    let columns = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Percentage(50), Constraint::Percentage(50)])
        .split(root[0]);

    let left_column = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(8),
            Constraint::Length(10),
            Constraint::Min(8),
        ])
        .split(columns[0]);

    let browse_items: Vec<ListItem<'_>> = app
        .filtered
        .iter()
        .map(|idx| &app.current_dataset()[*idx])
        .map(|r| {
            let line = format!(
                "{}{}{}{}",
                r.name,
                maybe_tag("type", &r.kind),
                maybe_tag("cr", &r.cr),
                maybe_tag("src", &r.source),
            );
            ListItem::new(line)
        })
        .collect();

    let browse_title = format!("[2]-Monsters [{}]", app.browse_mode.label());
    let browse_block = block_with_focus(&browse_title, app.active_panel == ActivePanel::Browse);
    let browse_list = List::new(browse_items).block(browse_block).highlight_style(
        Style::default()
            .fg(Color::Yellow)
            .add_modifier(Modifier::BOLD),
    );
    frame.render_stateful_widget(browse_list, left_column[2], &mut app.browse_state);

    let detail_title = match app.detail_mode {
        DetailMode::Description => "[3]-Description",
        DetailMode::Treasure => "[3]-Treasure",
    };
    let detail = Paragraph::new(app.scrolled_text(&app.detail_text()))
        .block(block_with_focus(detail_title, app.active_panel == ActivePanel::Detail))
        .wrap(Wrap { trim: false });
    frame.render_widget(detail, columns[1]);

    let enc_items: Vec<ListItem<'_>> = app
        .encounters
        .iter()
        .enumerate()
        .map(|(idx, e)| {
            let turn_mark = if app.turn_mode && idx == app.turn_index {
                ">> "
            } else {
                "   "
            };
            let init = if e.has_init_roll {
                format!(" init={}", e.init_roll)
            } else {
                String::new()
            };
            let cond = if e.conditions.is_empty() {
                String::new()
            } else {
                let txt = e
                    .conditions
                    .iter()
                    .map(|(k, v)| format!("{k}{v}"))
                    .collect::<Vec<_>>()
                    .join(",");
                format!(" cond=[{}]", txt)
            };
            let ch = if let Some(c) = &e.character {
                let mut parts = Vec::new();
                if !c.class.trim().is_empty() {
                    parts.push(c.class.trim().to_string());
                }
                if c.level > 0 {
                    parts.push(format!("Lv{}", c.level));
                }
                if !c.race.trim().is_empty() {
                    parts.push(c.race.trim().to_string());
                }
                if parts.is_empty() {
                    String::new()
                } else {
                    format!(" [{}]", parts.join(" "))
                }
            } else {
                String::new()
            };
            ListItem::new(format!(
                "{}{}{} #{} [id={}] hp={}/{}{}{}",
                turn_mark, e.name, ch, e.ordinal, e.monster_id, e.current_hp, e.base_hp, init, cond
            ))
        })
        .collect();
    let enc_list = List::new(enc_items)
        .block(block_with_focus("[1]-Encounters", app.active_panel == ActivePanel::Encounters))
        .highlight_style(Style::default().fg(Color::Yellow));
    frame.render_stateful_widget(enc_list, left_column[1], &mut app.encounter_state);

    let dice_items: Vec<ListItem<'_>> = app
        .dice
        .iter()
        .map(|d| ListItem::new(d.as_str()))
        .collect();
    let dice_list = List::new(dice_items)
        .block(block_with_focus("[0]-Dice", app.active_panel == ActivePanel::Dice))
        .highlight_style(Style::default().fg(Color::Yellow));
    frame.render_stateful_widget(dice_list, left_column[0], &mut app.dice_state);

    let status = Paragraph::new(app.status.as_str())
        .block(Block::default().borders(Borders::TOP))
        .alignment(Alignment::Left)
        .style(Style::default().fg(Color::White));
    frame.render_widget(status, root[1]);

    if app.input_mode != InputMode::Normal {
        let title = match app.input_mode {
            InputMode::Search => "Cerca (/)",
            InputMode::Dice => "Roll Dice (a)",
            InputMode::EncounterEdit => "Edit Encounter (e): nome;current_hp;base_hp",
            InputMode::ConditionEdit => "Condizione (c): CODE[:rounds] (es. B:2)",
            InputMode::CharacterCreate => "Nuovo Personaggio (N): nome;razza;classe;livello;hp",
            InputMode::Normal => "",
        };
        let area = centered_rect(70, 3, frame.area());
        frame.render_widget(Clear, area);
        let prompt = Paragraph::new(app.input_buffer.as_str())
            .block(Block::default().title(title).borders(Borders::ALL));
        frame.render_widget(prompt, area);
    }
}

fn render_fullscreen_panel(frame: &mut Frame<'_>, app: &mut App, area: Rect, panel: ActivePanel) {
    match panel {
        ActivePanel::Browse => {
            let browse_items: Vec<ListItem<'_>> = app
                .filtered
                .iter()
                .map(|idx| &app.current_dataset()[*idx])
                .map(|r| {
                    let line = format!(
                        "{}{}{}{}",
                        r.name,
                        maybe_tag("type", &r.kind),
                        maybe_tag("cr", &r.cr),
                        maybe_tag("src", &r.source),
                    );
                    ListItem::new(line)
                })
                .collect();
            let browse_title = format!("[2]-Monsters [{}] [fullscreen]", app.browse_mode.label());
            let browse_list = List::new(browse_items)
                .block(block_with_focus(&browse_title, true))
                .highlight_style(Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD));
            frame.render_stateful_widget(browse_list, area, &mut app.browse_state);
        }
        ActivePanel::Encounters => {
            let enc_items: Vec<ListItem<'_>> = app
                .encounters
                .iter()
                .enumerate()
                .map(|(idx, e)| {
                    let turn_mark = if app.turn_mode && idx == app.turn_index {
                        ">> "
                    } else {
                        "   "
                    };
                    let init = if e.has_init_roll {
                        format!(" init={}", e.init_roll)
                    } else {
                        String::new()
                    };
                    let cond = if e.conditions.is_empty() {
                        String::new()
                    } else {
                        let txt = e
                            .conditions
                            .iter()
                            .map(|(k, v)| format!("{k}{v}"))
                            .collect::<Vec<_>>()
                            .join(",");
                        format!(" cond=[{}]", txt)
                    };
                    let ch = if let Some(c) = &e.character {
                        let mut parts = Vec::new();
                        if !c.class.trim().is_empty() {
                            parts.push(c.class.trim().to_string());
                        }
                        if c.level > 0 {
                            parts.push(format!("Lv{}", c.level));
                        }
                        if !c.race.trim().is_empty() {
                            parts.push(c.race.trim().to_string());
                        }
                        if parts.is_empty() {
                            String::new()
                        } else {
                            format!(" [{}]", parts.join(" "))
                        }
                    } else {
                        String::new()
                    };
                    ListItem::new(format!(
                        "{}{}{} #{} [id={}] hp={}/{}{}{}",
                        turn_mark, e.name, ch, e.ordinal, e.monster_id, e.current_hp, e.base_hp, init, cond
                    ))
                })
                .collect();
            let enc_list = List::new(enc_items)
                .block(block_with_focus("[1]-Encounters [fullscreen]", true))
                .highlight_style(Style::default().fg(Color::Yellow));
            frame.render_stateful_widget(enc_list, area, &mut app.encounter_state);
        }
        ActivePanel::Dice => {
            let dice_items: Vec<ListItem<'_>> = app
                .dice
                .iter()
                .map(|d| ListItem::new(d.as_str()))
                .collect();
            let dice_list = List::new(dice_items)
                .block(block_with_focus("[0]-Dice [fullscreen]", true))
                .highlight_style(Style::default().fg(Color::Yellow));
            frame.render_stateful_widget(dice_list, area, &mut app.dice_state);
        }
        ActivePanel::Detail => {
            let detail_title = match app.detail_mode {
                DetailMode::Description => "[3]-Description [fullscreen]",
                DetailMode::Treasure => "[3]-Treasure [fullscreen]",
            };
            let detail = Paragraph::new(app.scrolled_text(&app.detail_text()))
                .block(block_with_focus(detail_title, true))
                .wrap(Wrap { trim: false });
            frame.render_widget(detail, area);
        }
    }
}

fn block_with_focus(title: &str, focused: bool) -> Block<'_> {
    let style = if focused {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default()
    };
    Block::default()
        .title(Span::styled(title, style.add_modifier(Modifier::BOLD)))
        .borders(Borders::ALL)
}

fn centered_rect(width_percent: u16, height_rows: u16, area: Rect) -> Rect {
    let vertical = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(1),
            Constraint::Length(height_rows),
            Constraint::Min(1),
        ])
        .split(area);
    let horizontal = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage((100 - width_percent) / 2),
            Constraint::Percentage(width_percent),
            Constraint::Percentage((100 - width_percent) / 2),
        ])
        .split(vertical[1]);
    horizontal[1]
}

fn maybe_tag(label: &str, value: &str) -> String {
    if value.is_empty() {
        String::new()
    } else {
        format!(" | {label}={value}")
    }
}

fn is_false(v: &bool) -> bool {
    !*v
}

fn is_zero_i32(v: &i32) -> bool {
    *v == 0
}

fn is_one_i32(v: &i32) -> bool {
    *v == 1
}

fn parse_dataset(root: &str, raw: &str) -> Result<Vec<Record>> {
    let yaml: Value = serde_yaml::from_str(raw)?;
    let Some(items) = yaml.get(root).and_then(Value::as_sequence) else {
        return Ok(Vec::new());
    };

    let mut out = Vec::with_capacity(items.len());
    for (idx, item) in items.iter().enumerate() {
        let environment = item
            .get("environment")
            .and_then(Value::as_sequence)
            .map(|arr| {
                arr.iter()
                    .filter_map(Value::as_str)
                    .map(ToString::to_string)
                    .collect::<Vec<_>>()
            })
            .unwrap_or_default();

        let description = if root == "monsters" {
            extract_monster_yaml_text(item)
        } else {
            extract_description(item)
        };
        let stat_block = if root == "monsters" {
            extract_monster_stat_block(item)
        } else {
            String::new()
        };
        out.push(Record {
            id: idx as i32,
            name: field_string(item, "name"),
            source: field_string(item, "source"),
            cr: field_string(item, "cr"),
            kind: field_string(item, "type"),
            environment,
            description,
            stat_block,
            base_hp: extract_base_hp(item),
            initiative_mod: extract_initiative_mod(item),
        });
    }
    Ok(out)
}

fn extract_base_hp(item: &Value) -> i32 {
    match item.get("hp") {
        Some(Value::Number(n)) => n.as_i64().unwrap_or(1).max(1) as i32,
        Some(Value::Mapping(map)) => {
            for key in ["average", "special", "hp"] {
                let Some(v) = map.get(&Value::String(key.to_string())) else {
                    continue;
                };
                match v {
                    Value::Number(n) => return n.as_i64().unwrap_or(1).max(1) as i32,
                    Value::String(s) => {
                        if let Ok(parsed) = s.parse::<i32>() {
                            return parsed.max(1);
                        }
                    }
                    _ => {}
                }
            }
            1
        }
        Some(Value::String(s)) => s.parse::<i32>().unwrap_or(1).max(1),
        _ => 1,
    }
}

fn extract_initiative_mod(item: &Value) -> i32 {
    let dex_score = match item.get("dex") {
        Some(Value::Number(n)) => n.as_i64().unwrap_or(10) as i32,
        Some(Value::String(s)) => s.parse::<i32>().unwrap_or(10),
        _ => 10,
    };
    (dex_score - 10).div_euclid(2)
}

fn cr_to_band(cr: &str) -> i32 {
    let v = cr.trim();
    if v.is_empty() || v == "0" || v == "1/8" || v == "1/4" || v == "1/2" {
        return 0;
    }
    if let Ok(n) = v.parse::<i32>() {
        if n <= 4 {
            0
        } else if n <= 10 {
            1
        } else if n <= 16 {
            2
        } else {
            3
        }
    } else {
        0
    }
}

fn band_label(band: i32) -> &'static str {
    match band {
        0 => "0-4",
        1 => "5-10",
        2 => "11-16",
        _ => "17+",
    }
}

fn extract_description(item: &Value) -> String {
    for key in ["description", "entries", "entry", "fluff"] {
        if let Some(v) = item.get(key) {
            let text = flatten_text(v);
            if !text.is_empty() {
                return text;
            }
        }
    }
    String::new()
}

fn extract_monster_stat_block(item: &Value) -> String {
    let size = field_string(item, "size");
    let alignment = field_string(item, "alignment");
    let ac = extract_ac(item);
    let hp = extract_hp_line(item);
    let speed = extract_speed(item);
    let str_score = field_int_or_str(item, "str");
    let dex_score = field_int_or_str(item, "dex");
    let con_score = field_int_or_str(item, "con");
    let int_score = field_int_or_str(item, "int");
    let wis_score = field_int_or_str(item, "wis");
    let cha_score = field_int_or_str(item, "cha");
    let passive = field_int_or_str(item, "passive");
    let senses = value_to_compact(item.get("senses"));
    let languages = value_to_compact(item.get("languages"));

    let mut lines = Vec::new();
    if !size.is_empty() || !alignment.is_empty() {
        lines.push(format!("size: {} | alignment: {}", size, alignment));
    }
    if !ac.is_empty() {
        lines.push(format!("ac: {}", ac));
    }
    if !hp.is_empty() {
        lines.push(format!("hp: {}", hp));
    }
    if !speed.is_empty() {
        lines.push(format!("speed: {}", speed));
    }
    lines.push(format!(
        "str {} | dex {} | con {} | int {} | wis {} | cha {}",
        str_score, dex_score, con_score, int_score, wis_score, cha_score
    ));
    if !passive.is_empty() || !senses.is_empty() {
        lines.push(format!("passive: {} | senses: {}", passive, senses));
    }
    if !languages.is_empty() {
        lines.push(format!("languages: {}", languages));
    }
    lines.join("\n")
}

fn extract_monster_yaml_text(item: &Value) -> String {
    let sections = [
        ("description", "Description"),
        ("entries", "Description"),
        ("entry", "Description"),
        ("trait", "Traits"),
        ("traits", "Traits"),
        ("action", "Actions"),
        ("actions", "Actions"),
        ("bonus", "Bonus Actions"),
        ("bonus_actions", "Bonus Actions"),
        ("reaction", "Reactions"),
        ("reactions", "Reactions"),
        ("legendary", "Legendary Actions"),
        ("legendary_actions", "Legendary Actions"),
        ("mythic", "Mythic Actions"),
        ("spellcasting", "Spellcasting"),
        ("lair_actions", "Lair Actions"),
        ("regional_effects", "Regional Effects"),
        ("fluff", "Lore"),
    ];
    let mut grouped: BTreeMap<&str, Vec<String>> = BTreeMap::new();
    for (key, label) in sections {
        if let Some(v) = item.get(key) {
            let text = flatten_text(v);
            if !text.trim().is_empty() {
                grouped.entry(label).or_default().push(text);
            }
        }
    }

    // Keep display order intentional instead of alphabetical.
    let ordered_labels = [
        "Description",
        "Traits",
        "Spellcasting",
        "Actions",
        "Bonus Actions",
        "Reactions",
        "Legendary Actions",
        "Mythic Actions",
        "Lair Actions",
        "Regional Effects",
        "Lore",
    ];
    let mut chunks = Vec::new();
    for label in ordered_labels {
        let Some(parts) = grouped.get(label) else {
            continue;
        };
        let section_text = parts
            .iter()
            .filter(|s| !s.trim().is_empty())
            .cloned()
            .collect::<Vec<_>>()
            .join("\n\n");
        if section_text.is_empty() {
            continue;
        }
        chunks.push(format!("{label}\n{section_text}"));
    }

    if chunks.is_empty() {
        extract_description(item)
    } else {
        chunks.join("\n\n")
    }
}

fn flatten_text(v: &Value) -> String {
    match v {
        Value::String(s) => s.clone(),
        Value::Sequence(seq) => seq
            .iter()
            .map(flatten_text)
            .filter(|s| !s.is_empty())
            .collect::<Vec<_>>()
            .join("\n"),
        Value::Mapping(map) => {
            let mut out = Vec::new();
            for (k, val) in map {
                if let Some(key) = k.as_str() {
                    if key.eq_ignore_ascii_case("name") || key.eq_ignore_ascii_case("type") {
                        continue;
                    }
                }
                let t = flatten_text(val);
                if !t.is_empty() {
                    out.push(t);
                }
            }
            out.join("\n")
        }
        _ => String::new(),
    }
}

fn field_string(item: &Value, key: &str) -> String {
    match item.get(key) {
        Some(Value::String(s)) => s.clone(),
        Some(Value::Number(n)) => n.to_string(),
        _ => String::new(),
    }
}

fn field_int_or_str(item: &Value, key: &str) -> String {
    match item.get(key) {
        Some(Value::String(s)) => s.clone(),
        Some(Value::Number(n)) => n.to_string(),
        Some(Value::Mapping(_)) | Some(Value::Sequence(_)) => value_to_compact(item.get(key)),
        _ => String::new(),
    }
}

fn value_to_compact(v: Option<&Value>) -> String {
    match v {
        Some(Value::String(s)) => s.clone(),
        Some(Value::Number(n)) => n.to_string(),
        Some(Value::Sequence(seq)) => seq
            .iter()
            .map(|x| value_to_compact(Some(x)))
            .filter(|s| !s.is_empty())
            .collect::<Vec<_>>()
            .join(", "),
        Some(Value::Mapping(map)) => map
            .iter()
            .map(|(k, val)| {
                let key = k.as_str().unwrap_or("").to_string();
                let value = value_to_compact(Some(val));
                if key.is_empty() {
                    value
                } else if value.is_empty() {
                    key
                } else {
                    format!("{key} {value}")
                }
            })
            .filter(|s| !s.is_empty())
            .collect::<Vec<_>>()
            .join(", "),
        _ => String::new(),
    }
}

fn extract_ac(item: &Value) -> String {
    match item.get("ac") {
        Some(Value::String(s)) => s.clone(),
        Some(Value::Number(n)) => n.to_string(),
        Some(Value::Sequence(seq)) => seq
            .iter()
            .map(|v| value_to_compact(Some(v)))
            .filter(|s| !s.is_empty())
            .collect::<Vec<_>>()
            .join(" | "),
        Some(v) => value_to_compact(Some(v)),
        None => String::new(),
    }
}

fn extract_hp_line(item: &Value) -> String {
    match item.get("hp") {
        Some(Value::Number(n)) => n.to_string(),
        Some(Value::String(s)) => s.clone(),
        Some(Value::Mapping(map)) => {
            let avg = map
                .get(&Value::String("average".to_string()))
                .map(|v| value_to_compact(Some(v)))
                .unwrap_or_default();
            let formula = map
                .get(&Value::String("formula".to_string()))
                .map(|v| value_to_compact(Some(v)))
                .unwrap_or_default();
            if !avg.is_empty() && !formula.is_empty() {
                format!("{avg} ({formula})")
            } else {
                value_to_compact(Some(&Value::Mapping(map.clone())))
            }
        }
        Some(v) => value_to_compact(Some(v)),
        None => String::new(),
    }
}

fn extract_speed(item: &Value) -> String {
    match item.get("speed") {
        Some(Value::String(s)) => s.clone(),
        Some(v) => value_to_compact(Some(v)),
        None => String::new(),
    }
}

fn browse(args: BrowseArgs) -> Result<()> {
    let (root, raw) = embedded_yaml(args.mode);
    let data = parse_dataset(root, raw)?;

    let name_filter = args.name.as_deref().map(str::to_lowercase);
    let env_filter = args.env.as_deref().map(str::to_lowercase);
    let src_filter = args.source.as_deref().map(str::to_lowercase);
    let cr_filter = args.cr.as_deref().map(str::to_lowercase);
    let type_filter = args.kind.as_deref().map(str::to_lowercase);

    let filtered: Vec<&Record> = data
        .iter()
        .filter(|r| {
            let name_ok = name_filter
                .as_deref()
                .is_none_or(|q| r.name.to_lowercase().contains(q));
            let env_ok = env_filter.as_deref().is_none_or(|q| {
                r.environment
                    .join(",")
                    .to_lowercase()
                    .contains(q)
            });
            let src_ok = src_filter
                .as_deref()
                .is_none_or(|q| r.source.to_lowercase().contains(q));
            let cr_ok = cr_filter
                .as_deref()
                .is_none_or(|q| r.cr.to_lowercase().contains(q));
            let type_ok = type_filter
                .as_deref()
                .is_none_or(|q| r.kind.to_lowercase().contains(q));
            name_ok && env_ok && src_ok && cr_ok && type_ok
        })
        .take(args.limit)
        .collect();

    println!("trovati: {} (limite {})", filtered.len(), args.limit);
    for (i, r) in filtered.iter().enumerate() {
        println!(
            "{:>3}. {}{}{}{}",
            i + 1,
            r.name,
            maybe_tag("type", &r.kind),
            maybe_tag("cr", &r.cr),
            maybe_tag("src", &r.source)
        );
    }
    Ok(())
}

fn show_encounters(arg: FileArg) -> Result<()> {
    let path = arg.path.unwrap_or_else(|| {
        std::env::var("ENCOUNTERS_YAML")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| PathBuf::from(DEFAULT_ENCOUNTERS_PATH))
    });
    let content = fs::read_to_string(&path)
        .with_context(|| format!("impossibile leggere {}", path.display()))?;
    let persisted: PersistedEncounters = serde_yaml::from_str(&content)?;

    println!("file: {}", path.display());
    println!("version: {}", persisted.version.unwrap_or_default());
    println!("items: {}", persisted.items.len());
    for (i, it) in persisted.items.iter().take(20).enumerate() {
        let label = if it.custom && !it.custom_name.is_empty() {
            it.custom_name.as_str()
        } else {
            "monster"
        };
        println!(
            "{:>3}. {} (id={} ord={} hp={}/{})",
            i + 1,
            label,
            it.monster_id,
            it.ordinal,
            it.current_hp,
            it.base_hp
        );
    }
    Ok(())
}

fn show_dice(arg: FileArg) -> Result<()> {
    let path = arg.path.unwrap_or_else(|| {
        std::env::var("DICE_YAML")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| PathBuf::from(DEFAULT_DICE_PATH))
    });
    let content = fs::read_to_string(&path)
        .with_context(|| format!("impossibile leggere {}", path.display()))?;
    let persisted: PersistedDice = serde_yaml::from_str(&content)?;

    println!("file: {}", path.display());
    println!("version: {}", persisted.version.unwrap_or_default());
    println!("items: {}", persisted.items.len());
    for (i, it) in persisted.items.iter().take(20).enumerate() {
        match it {
            DiceEntry::Structured(v) => println!("{:>3}. {} => {}", i + 1, v.expression, v.output),
            DiceEntry::Legacy(v) => println!("{:>3}. {}", i + 1, v),
        }
    }
    Ok(())
}

fn embedded_yaml(mode: BrowseMode) -> (&'static str, &'static str) {
    match mode {
        BrowseMode::Monsters => ("monsters", include_str!("../data/monster.yaml")),
        BrowseMode::Items => ("items", include_str!("../data/item.yaml")),
        BrowseMode::Spells => ("spells", include_str!("../data/spell.yaml")),
        BrowseMode::Characters => ("classes", include_str!("../data/class.yaml")),
        BrowseMode::Races => ("races", include_str!("../data/race.yaml")),
        BrowseMode::Feats => ("feats", include_str!("../data/feat.yaml")),
        BrowseMode::Books => ("books", include_str!("../data/book.yaml")),
        BrowseMode::Adventures => ("adventures", include_str!("../data/adventure.yaml")),
    }
}

#[derive(Debug)]
struct DiceRoll {
    expression: String,
    rolls: Vec<i32>,
    modifier: i32,
}

impl fmt::Display for DiceRoll {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let total: i32 = self.rolls.iter().sum::<i32>() + self.modifier;
        if self.modifier == 0 {
            write!(
                f,
                "{} => {} = {}",
                self.expression,
                self.rolls
                    .iter()
                    .map(i32::to_string)
                    .collect::<Vec<_>>()
                    .join("+"),
                total
            )
        } else {
            write!(
                f,
                "{} => {}{:+} = {}",
                self.expression,
                self.rolls
                    .iter()
                    .map(i32::to_string)
                    .collect::<Vec<_>>()
                    .join("+"),
                self.modifier,
                total
            )
        }
    }
}

fn roll_expression(input: &str) -> Result<String> {
    let expr = input.trim().to_lowercase();
    let (num, sides, modifier) = parse_dice_expr(&expr)?;
    let mut rng = rand::rng();
    let mut rolls = Vec::with_capacity(num as usize);
    for _ in 0..num {
        rolls.push(rng.random_range(1..=sides));
    }
    Ok(DiceRoll {
        expression: expr,
        rolls,
        modifier,
    }
    .to_string())
}

fn parse_dice_expr(expr: &str) -> Result<(i32, i32, i32)> {
    let Some(d_pos) = expr.find('d') else {
        anyhow::bail!("espressione non valida: manca 'd'");
    };

    let left = &expr[..d_pos];
    let right = &expr[d_pos + 1..];
    let count = if left.is_empty() { 1 } else { left.parse::<i32>()? };

    let mut sides_part = right;
    let mut modifier = 0;
    if let Some(pos) = right.find('+') {
        sides_part = &right[..pos];
        modifier = right[pos + 1..].parse::<i32>()?;
    } else if let Some(pos) = right[1..].find('-') {
        let p = pos + 1;
        sides_part = &right[..p];
        modifier = -right[p + 1..].parse::<i32>()?;
    }

    let sides = sides_part.parse::<i32>()?;
    if count <= 0 || sides <= 0 {
        anyhow::bail!("espressione non valida: count/sides devono essere > 0");
    }
    Ok((count, sides, modifier))
}

fn parse_character_payload(payload: &str) -> Result<CharacterBuild> {
    let parts: Vec<&str> = payload.split(';').collect();
    if parts.len() < 5 {
        anyhow::bail!("formato atteso nome;razza;classe;livello;hp");
    }
    let name = parts[0].trim();
    let race = parts[1].trim();
    let class = parts[2].trim();
    let level = parts[3].trim().parse::<i32>()?;
    let hp = parts[4].trim().parse::<i32>()?;
    if name.is_empty() || level <= 0 || hp <= 0 {
        anyhow::bail!("dati non validi");
    }
    Ok(CharacterBuild {
        name: name.to_string(),
        race: race.to_string(),
        class: class.to_string(),
        level,
        hp,
    })
}

fn parse_condition_payload(payload: &str) -> Result<(String, i32)> {
    let text = payload.trim();
    if text.is_empty() {
        anyhow::bail!("vuoto");
    }
    let mut parts = text.split(':');
    let code = parts.next().unwrap_or("").trim().to_uppercase();
    if code.is_empty() {
        anyhow::bail!("code vuoto");
    }
    let rounds = parts
        .next()
        .map(str::trim)
        .filter(|s| !s.is_empty())
        .and_then(|s| s.parse::<i32>().ok())
        .unwrap_or(1)
        .max(1);
    Ok((code, rounds))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_embedded_monsters() {
        let entries = parse_dataset("monsters", include_str!("../data/monster.yaml")).expect("yaml valido");
        assert!(!entries.is_empty());
    }

    #[test]
    fn parse_dice_ok() {
        let (n, s, m) = parse_dice_expr("2d6+3").expect("parse");
        assert_eq!((n, s, m), (2, 6, 3));
    }

    #[test]
    fn parse_dice_bad() {
        assert!(parse_dice_expr("abc").is_err());
    }

    #[test]
    fn parse_character_payload_ok() {
        let c = parse_character_payload("Aria;Elf;Wizard;5;28").expect("parse char");
        assert_eq!(c.name, "Aria");
        assert_eq!(c.race, "Elf");
        assert_eq!(c.class, "Wizard");
        assert_eq!(c.level, 5);
        assert_eq!(c.hp, 28);
    }

    #[test]
    fn parse_character_payload_bad() {
        assert!(parse_character_payload("x;y;z;0;1").is_err());
        assert!(parse_character_payload("x;y;z;2;0").is_err());
        assert!(parse_character_payload("x;y;z").is_err());
    }

    #[test]
    fn parse_condition_payload_ok() {
        let (code, rounds) = parse_condition_payload("b:3").expect("cond");
        assert_eq!(code, "B");
        assert_eq!(rounds, 3);

        let (code2, rounds2) = parse_condition_payload(" stunned ").expect("cond2");
        assert_eq!(code2, "STUNNED");
        assert_eq!(rounds2, 1);
    }

    #[test]
    fn parse_condition_payload_bad() {
        assert!(parse_condition_payload("").is_err());
        assert!(parse_condition_payload(" :2").is_err());
    }

    #[test]
    fn cr_band_mapping() {
        assert_eq!(cr_to_band("1/2"), 0);
        assert_eq!(cr_to_band("4"), 0);
        assert_eq!(cr_to_band("5"), 1);
        assert_eq!(cr_to_band("11"), 2);
        assert_eq!(cr_to_band("20"), 3);
    }

    #[test]
    fn monster_stat_block_contains_core_fields() {
        let entries =
            parse_dataset("monsters", include_str!("../data/monster.yaml")).expect("yaml valido");
        let first = entries.first().expect("at least one monster");
        assert!(!first.stat_block.is_empty());
        let low = first.stat_block.to_lowercase();
        assert!(low.contains("str"));
        assert!(low.contains("dex"));
        assert!(low.contains("hp"));
    }

    #[test]
    fn monster_yaml_text_is_present() {
        let entries =
            parse_dataset("monsters", include_str!("../data/monster.yaml")).expect("yaml valido");
        let first = entries.first().expect("at least one monster");
        assert!(!first.description.is_empty());
    }
}
