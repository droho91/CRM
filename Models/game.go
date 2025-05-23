package models

import "math"

// ────────────────────────────
// ‼️  CONFIGURABLE CONSTANTS ‼️
// ────────────────────────────
const (
	BaseReqEXP   = 100  // EXP needed for level-1→2
	ReqExpFactor = 1.10 // +10 % each level
	StatBoost    = 0.10 // +10 % ATK/DEF/HP per level
)

// ────────────────────────────
//      Static specifications
// ────────────────────────────
type TowerSpec struct {
	Name         string
	HP, ATK, DEF int
}
type TroopSpec struct {
	Name               string
	HP, ATK, DEF, MANA int
}

var (
	KingTowerSpec  = TowerSpec{"King Tower", 2000, 500, 300}
	GuardTowerSpec = TowerSpec{"Guard Tower", 1000, 300, 100}

	AllTroopSpecs = []TroopSpec{
		{"Pawn", 50, 150, 100, 3},
		{"Bishop", 100, 200, 150, 4},
		{"Rook", 250, 200, 200, 5},
		{"Knight", 200, 300, 150, 5},
		{"Prince", 500, 400, 300, 6},
	}
)

// ────────────────────────────
//        Runtime objects
// ────────────────────────────
type Tower struct {
	Spec  TowerSpec
	Label string
	HP    int
}

type Troop struct {
	Spec TroopSpec
	HP   int
	// Used bool
}

type PlayerState struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	EXP      int     `json:"exp"`
	Level    int     `json:"level"`
	Mana     int     `json:"mana"`
	Towers   []Tower `json:"-"` // runtime only
	Troops   []Troop `json:"-"` // runtime only
}

// ---------- constructors ----------
func NewTower(spec TowerSpec, lvl int) Tower {
	m := levelMultiplier(lvl)
	return Tower{
		Spec:  spec,
		Label: spec.Name,
		HP:    int(float64(spec.HP) * m),
	}

}

func (t Tower) WithLabel(label string) Tower {
	t.Label = label
	return t
}

func NewTroop(spec TroopSpec, lvl int) Troop {
	m := levelMultiplier(lvl)
	return Troop{Spec: TroopSpec{
		Name: spec.Name,
		HP:   int(float64(spec.HP) * m),
		ATK:  int(float64(spec.ATK) * m),
		DEF:  int(float64(spec.DEF) * m),
		MANA: spec.MANA,
	}, HP: int(float64(spec.HP) * m)}
}

// ---------- helpers ----------
func levelMultiplier(lvl int) float64 {
	return 1.0 + StatBoost*float64(lvl-1)
}
func RequiredEXP(level int) int {
	return int(float64(BaseReqEXP) * math.Pow(ReqExpFactor, float64(level-1)))
}
func (p *PlayerState) GainEXP(amount int) {
	p.EXP += amount
	for p.EXP >= RequiredEXP(p.Level) {
		p.EXP -= RequiredEXP(p.Level)
		p.Level++
	}
}
func (p *PlayerState) HasAliveTowers() bool {
	for _, t := range p.Towers {
		if t.HP > 0 {
			return true
		}
	}
	return false
}
