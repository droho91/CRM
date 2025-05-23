package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	// ↙️  Đổi nếu go.mod không phải “module CRM”
	"CRM/models"
)

/*─────────────────────────────────────────────────────────────
                       CẤU TRÚC DỮ LIỆU
─────────────────────────────────────────────────────────────*/

const MaxPlayers = 2

// gói một kết nối + trạng thái runtime
type clientConn struct {
	conn   net.Conn
	rw     *bufio.ReadWriter
	player *models.PlayerState
	mu     sync.Mutex // bảo vệ player
}

/* ===== Helper methods để dùng như trước ===== */
func (c *clientConn) WriteString(s string) (int, error) { return c.rw.WriteString(s) }
func (c *clientConn) Flush() error                      { return c.rw.Flush() }
func (c *clientConn) ReadString(delim byte) (string, error) {
	return c.rw.ReadString(delim)
}
func (c *clientConn) Close() error { return c.conn.Close() }
func (c *clientConn) Write(p []byte) (int, error) {
	return c.rw.Write(p) // để fmt.Fprint() hoạt động
}

type gameServer struct {
	players [MaxPlayers]*clientConn
	turn    int // 0 hoặc 1 – chỉ số người chơi hiện tại
	cmdCh   chan deployCmd
}

// DB persistent
var accountDB map[string]*models.PlayerState

/*─────────────────────────────────────────────────────────────
                           MAIN ENTRY
─────────────────────────────────────────────────────────────*/

func StartServer() {
	var err error
	accountDB, err = loadPlayers()
	if err != nil {
		log.Fatalf("loading player DB: %v", err)
	}

	rand.Seed(time.Now().UnixNano())

	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	fmt.Println("TCR server started on :9090 – waiting for 2 players…")

	gs := &gameServer{
		cmdCh: make(chan deployCmd, 32), // buffered so multiple sends don’t block

	}
	for i := 0; i < MaxPlayers; i++ {
		c, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go gs.registerPlayer(i, c)
	}

	select {} // block mãi
}

/*─────────────────────────────────────────────────────────────
                       ĐĂNG KÝ NGƯỜI CHƠI
─────────────────────────────────────────────────────────────*/

func (gs *gameServer) registerPlayer(idx int, c net.Conn) {
	rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))

	// hỏi username
	rw.WriteString("WELCOME! Enter username:\n")
	rw.Flush()
	username, _ := rw.ReadString('\n')
	username = strings.TrimSpace(username)

	// lấy / tạo account
	acc, ok := accountDB[username]
	if !ok {
		acc = &models.PlayerState{Username: username, EXP: 0, Level: 1}
		accountDB[username] = acc
	}

	// hỏi password (bỏ qua)
	rw.WriteString("Enter password:\n")
	rw.Flush()
	_, _ = rw.ReadString('\n')

	// tạo player runtime
	player := &models.PlayerState{
		Username: username,
		Level:    acc.Level,
		EXP:      acc.EXP,
		Mana:     5,
		Towers: []models.Tower{
			models.NewTower(models.GuardTowerSpec, acc.Level).WithLabel(fmt.Sprintf("G%sL", username)),
			models.NewTower(models.GuardTowerSpec, acc.Level).WithLabel(fmt.Sprintf("G%sR", username)),
			models.NewTower(models.KingTowerSpec, acc.Level).WithLabel(fmt.Sprintf("K%s", username)),
		},

		Troops: randomTroops(3, acc.Level),
	}
	gs.players[idx] = &clientConn{conn: c, rw: rw, player: player}

	fmt.Fprintln(rw, "Waiting for other player…")
	rw.Flush()

	// đủ 2 người → bắt đầu
	if gs.players[0] != nil && gs.players[1] != nil {
		go gs.runGameContinuous()
	}
}

/*─────────────────────────────────────────────────────────────
                          VÒNG GAME
─────────────────────────────────────────────────────────────*/
// Turn-based game
// func (gs *gameServer) runGame() {
// 	// ─── Start mana regen ticker ───
// 	regenTicker := time.NewTicker(time.Second)
// 	done := make(chan struct{})

// 	go func() {
// 		for {
// 			select {
// 			case <-regenTicker.C:
// 				for _, cl := range gs.players {
// 					if cl.player.Mana < 10 {
// 						cl.player.Mana++
// 					}
// 				}
// 			case <-done:
// 				return
// 			}
// 		}
// 	}()
// 	defer func() { close(done); regenTicker.Stop() }()

// 	for _, cl := range gs.players {
// 		cl.WriteString("=== GAME START ===\n")
// 		cl.Flush()
// 	}

// 	for {
// 		current := gs.players[gs.turn]
// 		opponent := gs.players[1-gs.turn]

// 		current.WriteString("YOUR TURN\n")
// 		opponent.WriteString("OPPONENT TURN\n")
// 		current.Flush()
// 		opponent.Flush()

// 		// show hand which troops are available for player
// 		showHand := func(cl *clientConn) {
// 			cl.WriteString("Your hand: ")
// 			for i, tr := range cl.player.Troops {
// 				cl.WriteString(fmt.Sprintf("[%d]%s ", i, tr.Spec.Name))
// 			}
// 			cl.WriteString("\n")
// 			cl.Flush()
// 		}
// 		showHand(current)

// 		// gs.broadcastState()

// 		current.WriteString(fmt.Sprintf("Mana: %d/10\n", current.player.Mana))

// 		current.WriteString("Choose troop index (0-2):\n ")
// 		current.Flush()
// 		line, _ := current.ReadString('\n')
// 		idx := parseIndex(strings.TrimSpace(line))

// 		// Ask for lane
// 		current.WriteString("Choose lane (L/R):\n")
// 		current.Flush()
// 		laneStr, _ := current.ReadString('\n')
// 		lane := strings.ToUpper(strings.TrimSpace(laneStr))
// 		if lane != "L" && lane != "R" { // mặc định
// 			lane = "L"
// 		}

// 		if idx < 0 || idx >= len(current.player.Troops) {
// 			current.WriteString("Invalid index – turn skipped\n")
// 			current.Flush()
// 			continue
// 		}

// 		if idx < 0 || idx >= len(current.player.Troops) {
// 			current.WriteString("Invalid index – turn skipped\n")
// 			current.Flush()
// 			continue
// 		}

// 		need := current.player.Troops[idx].Spec.MANA
// 		if current.player.Mana < need {
// 			current.WriteString(fmt.Sprintf("Not enough mana (need %d, have %d)\n", need, current.player.Mana))
// 			current.Flush()
// 			// không trừ lượt: bạn có thể cho chọn lại; ở đây skip lượt
// 		} else {
// 			current.player.Mana -= need
// 			gs.processAttack(idx, lane)
// 		}

//			if gs.checkWin() {
//				return
//			}
//			gs.turn = 1 - gs.turn
//		}
//	}

// ─────────────────────────────────────────────────────────────
// continuous play
func (gs *gameServer) runGameContinuous() {
	for _, cl := range gs.players {
		cl.WriteString("=== CONTINUOUS PLAY – 3-MINUTE MATCH ===\n")
		cl.Flush()
	}

	manaTicker := time.NewTicker(time.Second)
	defer manaTicker.Stop()

	endTimer := time.After(3 * time.Minute) // ← single timer!

	// spawn input goroutines
	for idx, cl := range gs.players {
		go gs.readCommands(idx, cl)
	}

	for {
		select {
		case cmd := <-gs.cmdCh:
			gs.handleDeploy(cmd)

		case <-manaTicker.C:
			for _, cl := range gs.players {
				if cl.player.Mana < 10 {
					cl.player.Mana++
				}
			}

		case <-endTimer: // ← fires once after 3 min
			gs.endByTimeout()
			return
		}

		if gs.checkWin() {
			return
		}
	}
}

/*─────────────────────────────────────────────────────────────
                       XỬ LÝ TẤN CÔNG
─────────────────────────────────────────────────────────────*/

// func (gs *gameServer) processAttack(troopIdx int) {
// 	att := gs.players[gs.turn].player
// 	def := gs.players[1-gs.turn].player
// 	troop := &att.Troops[troopIdx]

// 	if troop.Used {
// 		gs.players[gs.turn].WriteString("Troop already used – turn wasted\n")
// 		gs.players[gs.turn].Flush()
// 		return
// 	}
// 	troop.Used = true

// 	targets := []*models.Tower{&def.Towers[0], &def.Towers[1], &def.Towers[2]}
// 	for _, t := range targets {
// 		if t.HP <= 0 {
// 			continue
// 		}
// 		dmg := troop.Spec.ATK - t.Spec.DEF
// 		if dmg < 0 {
// 			dmg = 0
// 		}
// 		t.HP -= dmg
// 		if t.HP < 0 {
// 			t.HP = 0
// 		}
// 		gs.broadcast(fmt.Sprintf("%s’s %s dealt %d dmg to %s (HP now %d)\n",
// 			att.Username, troop.Spec.Name, dmg, t.Spec.Name, t.HP))
// 		if t.HP > 0 {
// 			break
// 		}
// 	}
// }

// --- xử lý tấn công cho 1 quân lính ---Turn based game
func (gs *gameServer) processAttack(troopIdx int, lane string) {
	attacker := gs.players[gs.turn].player
	defender := gs.players[1-gs.turn].player
	troop := &attacker.Troops[troopIdx]

	// --- xác định mục tiêu dựa trên lane ---
	var target *models.Tower
	if lane == "L" {
		if defender.Towers[0].HP > 0 { // Guard-L còn sống
			target = &defender.Towers[0]
		} else {
			target = &defender.Towers[2] // King Tower
		}
	} else { // "R"
		if defender.Towers[1].HP > 0 { // Guard-R còn sống
			target = &defender.Towers[1]
		} else {
			target = &defender.Towers[2]
		}
	}

	// --- gây sát thương ---
	dmg := troop.Spec.ATK - target.Spec.DEF
	if dmg < 0 {
		dmg = 0
	}
	target.HP -= dmg
	if target.HP < 0 {
		target.HP = 0
	}

	gs.broadcast(fmt.Sprintf("%s’s %s hit %s lane → %s (%d dmg, HP %d)\n",
		attacker.Username, troop.Spec.Name, lane, target.Label, dmg, target.HP))

	// --- rút quân mới thay thế ô đã dùng ---
	newSpec := models.AllTroopSpecs[rand.Intn(len(models.AllTroopSpecs))]
	attacker.Troops[troopIdx] = models.NewTroop(newSpec, attacker.Level)
}

/*─────────────────────────────────────────────────────────────
                      KIỂM TRA THẮNG – EXP
─────────────────────────────────────────────────────────────*/

func (gs *gameServer) checkWin() bool {
	k0 := gs.players[0].player.Towers[2].HP
	k1 := gs.players[1].player.Towers[2].HP

	if k0 <= 0 || k1 <= 0 {
		var win, lose *clientConn
		if k0 <= 0 {
			win, lose = gs.players[1], gs.players[0]
		} else {
			win, lose = gs.players[0], gs.players[1]
		}

		win.WriteString("=== YOU WIN! ===\n")
		lose.WriteString("=== YOU LOSE! ===\n")
		win.Flush()
		lose.Flush()

		gs.awardEXP(win.player.Username, 30)
		gs.awardEXP(lose.player.Username, 0)

		win.Close()
		lose.Close()
		return true
	}
	return false
}

func (gs *gameServer) awardEXP(user string, amt int) {
	acc := accountDB[user]
	before := acc.Level
	acc.GainEXP(amt)
	if acc.Level > before {
		gs.broadcast(fmt.Sprintf("%s leveled up to %d!\n", user, acc.Level))
	}
	_ = savePlayers(accountDB)
}

/*─────────────────────────────────────────────────────────────
                          UTILITIES
─────────────────────────────────────────────────────────────*/

func (gs *gameServer) broadcast(msg string) {
	for _, cl := range gs.players {
		cl.SafeWrite(msg)
	}
}

func (gs *gameServer) broadcastState() {
	state := map[string]any{
		gs.players[0].player.Username: map[string]any{
			"level":  gs.players[0].player.Level,
			"exp":    gs.players[0].player.EXP,
			"towers": gs.players[0].player.Towers,
			"troops": gs.players[0].player.Troops,
		},
		gs.players[1].player.Username: map[string]any{
			"level":  gs.players[1].player.Level,
			"exp":    gs.players[1].player.EXP,
			"towers": gs.players[1].player.Towers,
			"troops": gs.players[1].player.Troops,
		},
	}
	js, _ := json.MarshalIndent(state, "", "  ")
	gs.broadcast("GAME STATE:\n" + string(js) + "\n")
}

func parseIndex(s string) int {
	if len(s) == 0 {
		return -1
	}
	return int(s[0] - '0')
}

func randomTroops(n, lvl int) []models.Troop {
	out := make([]models.Troop, n)
	for i := 0; i < n; i++ {
		spec := models.AllTroopSpecs[rand.Intn(len(models.AllTroopSpecs))]
		out[i] = models.NewTroop(spec, lvl)
	}
	return out
}

// ___─────────────────────────────────────────────────────────────
// --- xử lý lệnh deploy quân lính ---Continuous game
type deployCmd struct {
	pIdx int    // 0 or 1
	slot int    // 0-2
	lane string // "L" or "R"
}

func (gs *gameServer) readCommands(idx int, cl *clientConn) {
	for {

		cl.SafePrintf("Mana %d/10 – deploy: <slot 0-2> <L/R>\n", cl.player.Mana)
		cl.Flush()

		line, err := cl.ReadString('\n')
		if err != nil {
			return
		} // client closed
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) != 2 {
			continue
		}

		slot := parseIndex(parts[0])
		lane := strings.ToUpper(parts[1])
		if slot < 0 || slot > 2 || (lane != "L" && lane != "R") {
			continue
		}

		gs.cmdCh <- deployCmd{pIdx: idx, slot: slot, lane: lane}
	}
}

func (gs *gameServer) handleDeploy(cmd deployCmd) {
	p := gs.players[cmd.pIdx].player
	if cmd.slot < 0 || cmd.slot >= len(p.Troops) {
		return
	}

	cost := p.Troops[cmd.slot].Spec.MANA
	if p.Mana < cost {
		gs.players[cmd.pIdx].WriteString("Not enough mana!\n")
		gs.players[cmd.pIdx].Flush()
		return
	}

	p.Mana -= cost
	gs.processAttackFor(cmd.pIdx, cmd.slot, cmd.lane)
}

// --- xử lý tấn công cho 1 quân lính ---Continuous game
/*
─────────────────────────────────────────────────────────────

	Deploy action – one hit, then draw a new troop

─────────────────────────────────────────────────────────────
*/
func (gs *gameServer) processAttackFor(pIdx, slot int, lane string) {
	attacker := gs.players[pIdx].player
	defender := gs.players[1-pIdx].player
	troop := &attacker.Troops[slot]

	/* ── copy fields needed for the log BEFORE we mutate anything ── */
	userName := attacker.Username
	cardName := troop.Spec.Name

	/* pick target by lane */
	var target *models.Tower
	if lane == "L" {
		if defender.Towers[0].HP > 0 {
			target = &defender.Towers[0]
		} else {
			target = &defender.Towers[2]
		}
	} else {
		if defender.Towers[1].HP > 0 {
			target = &defender.Towers[1]
		} else {
			target = &defender.Towers[2]
		}
	}

	/* damage */
	dmg := troop.Spec.ATK - target.Spec.DEF
	if dmg < 0 {
		dmg = 0
	}
	target.HP -= dmg
	if target.HP < 0 {
		target.HP = 0
	}

	/* single, correct broadcast */
	gs.broadcast(fmt.Sprintf(
		"%s’s %s hit %s lane → %s (%d dmg, HP %d)\n",
		userName, cardName, lane, target.Label, dmg, target.HP,
	))

	/* now it’s safe to draw a new card for that slot */
	newSpec := models.AllTroopSpecs[rand.Intn(len(models.AllTroopSpecs))]
	attacker.Troops[slot] = models.NewTroop(newSpec, attacker.Level)
}

func (gs *gameServer) endByTimeout() {
	// Choose winner by total towers
	// alive := func(t []models.Tower) int {
	// 	c := 0
	// 	for _, tw := range t {
	// 		if tw.HP > 0 {
	// 			c++
	// 		}
	// 	}
	// 	return c
	// }

	// a := alive(gs.players[0].player.Towers)
	// b := alive(gs.players[1].player.Towers)

	// if a < b {
	// 	gs.declareWinner(1, 0) // P2 wins
	// } else if b < a {
	// 	gs.declareWinner(0, 1) // P1 wins
	// } else {
	// 	gs.broadcast("=== DRAW – time up ===\n")
	// }

	// Choose winner by total HP
	total := func(ts []models.Tower) int {
		sum := 0
		for _, t := range ts {
			sum += t.HP
		}
		return sum
	}
	hpA := total(gs.players[0].player.Towers)
	hpB := total(gs.players[1].player.Towers)

	switch {
	case hpA < hpB:
		gs.declareWinner(0, 1)
	case hpB < hpA:
		gs.declareWinner(1, 0)
	default:
		gs.broadcast("=== DRAW – time up ===\n")
	}

}

/*
─────────────────────────────────────────────────────────────

	Declare winner / loser, give EXP, close sockets

─────────────────────────────────────────────────────────────
*/

func (gs *gameServer) declareWinner(winnerIdx, loserIdx int) {
	winner := gs.players[winnerIdx]
	loser := gs.players[loserIdx]

	winner.WriteString("=== YOU WIN! ===\n")
	loser.WriteString("=== YOU LOSE! ===\n")
	winner.Flush()
	loser.Flush()

	gs.awardEXP(winner.player.Username, 30)
	gs.awardEXP(loser.player.Username, 0)

	_ = winner.Close()
	_ = loser.Close()
}

// --- thread-safe write methods ---
// (không cần thiết, nhưng tốt hơn là không lock cả conn)
// (để tránh deadlock nếu có nhiều goroutine cùng gọi WriteString)
// (vì WriteString() gọi Flush() bên trong)
// (có thể lock cả conn, nếu có goroutine khác đang đọc)
// (nhưng không cần thiết, vì Flush() không block)
// (vì Flush() không block, chỉ block khi có nhiều goroutine cùng gọi)
// (nên dùng mutex để lock cả conn, nếu có nhiều goroutine cùng gọi)
// (để tránh deadlock nếu có nhiều goroutine cùng gọi WriteString)
// (vì WriteString() gọi Flush() bên trong)
func (c *clientConn) SafeWrite(s string) {
	c.mu.Lock()
	c.rw.WriteString(s)
	c.rw.Flush()
	c.mu.Unlock()
}

func (c *clientConn) SafePrintf(format string, a ...interface{}) {
	c.mu.Lock()
	fmt.Fprintf(c.rw, format, a...)
	c.rw.Flush()
	c.mu.Unlock()
}
