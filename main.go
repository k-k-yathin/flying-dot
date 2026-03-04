package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/nsf/termbox-go"
)

type gameState int

const (
	statePlaying gameState = iota
	stateGameOver
)

type obstacle struct {
	x    int
	gapY int
}

type game struct {
	width      int
	height     int
	heliX      int
	heliY      int
	velocity   float64
	state      gameState
	score      int
	bestScore  int
	obstacles  []obstacle
	gapHeight  int
	gravity    float64
	thrust     float64
	tickMillis int
}

func newGame(w, h, bestScore int) *game {
	g := &game{
		width:      w,
		height:     h,
		heliX:      w / 6,
		heliY:      h / 2,
		velocity:   0,
		state:      statePlaying,
		score:      0,
		bestScore: bestScore,
		obstacles: []obstacle{},
		// Easier game: bigger gap, slower fall, fast climb.
		gapHeight:  8,
		gravity:    0.18,  // slower fall
		thrust:     -2.4,  // faster upward jump
		tickMillis: 60,
	}
	g.initObstacles()
	return g
}

func (g *game) groundY() int {
	// Last row is ground
	return g.height - 2
}

func (g *game) initObstacles() {
	g.obstacles = g.obstacles[:0]
	spacing := 34
	// Pre-generate a long stream of obstacles; the update loop
	// will keep recycling them so they are effectively unlimited.
	for x := g.width; x < g.width+40*spacing; x += spacing {
		g.obstacles = append(g.obstacles, g.newObstacleAt(x))
	}
}

func (g *game) newObstacleAt(x int) obstacle {
	minGapY := 2
	maxGapY := g.groundY() - g.gapHeight - 1
	if maxGapY <= minGapY {
		maxGapY = minGapY + 1
	}
	gapY := rand.Intn(maxGapY-minGapY) + minGapY
	return obstacle{x: x, gapY: gapY}
}

func (g *game) update() {
	if g.state != statePlaying {
		return
	}

	// Physics
	g.velocity += g.gravity
	g.heliY += int(g.velocity)

	// Bounds
	if g.heliY < 1 {
		g.heliY = 1
		g.velocity = 0
	}

	// Move obstacles
	for i := range g.obstacles {
		g.obstacles[i].x--
	}

	// Recycle obstacles and update score when passed
	passed := 0
	for i := range g.obstacles {
		if g.obstacles[i].x+1 == g.heliX {
			g.score++
			if g.score > g.bestScore {
				g.bestScore = g.score
			}
		}
	}

	for len(g.obstacles) > 0 && g.obstacles[0].x < -1 {
		g.obstacles = g.obstacles[1:]
		passed++
	}
	for i := 0; i < passed; i++ {
		lastX := g.width - 1
		if len(g.obstacles) > 0 && g.obstacles[len(g.obstacles)-1].x > lastX {
			lastX = g.obstacles[len(g.obstacles)-1].x
		}
		g.obstacles = append(g.obstacles, g.newObstacleAt(lastX+34))
	}

	// Collisions
	if g.heliY >= g.groundY() {
		g.heliY = g.groundY()
		g.gameOver()
		return
	}

	for _, o := range g.obstacles {
		if o.x == g.heliX {
			if g.heliY < o.gapY || g.heliY > o.gapY+g.gapHeight-1 {
				g.gameOver()
				return
			}
		}
	}
}

func (g *game) flap() {
	if g.state != statePlaying {
		return
	}
	g.velocity = g.thrust
}

func (g *game) gameOver() {
	g.state = stateGameOver
}

func (g *game) reset(bestScore int) {
	*g = *newGame(g.width, g.height, bestScore)
}

func (g *game) draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	// Draw title "FLYING DOT" centered at top
	title := "≈≈≈ FLYING DOT ≈≈≈"
	titleStartX := (g.width - len(title)) / 2
	if titleStartX < 0 {
		titleStartX = 0
	}
	for i, ch := range title {
		x := titleStartX + i
		if x >= 0 && x < g.width {
			termbox.SetCell(x, 0, ch, termbox.ColorMagenta|termbox.AttrBold, termbox.ColorDefault)
		}
	}

	// Draw HUD just below title
	status := " SPACE = up | Q = quit"
	gameOverText := ""
	if g.state == stateGameOver {
		gameOverText = "  GAME OVER - R = restart"
	}
	text := "Score: " + itoa(g.score) + "  Best: " + itoa(g.bestScore) + "  " + status + gameOverText
	for i, ch := range text {
		if i >= g.width {
			break
		}
		termbox.SetCell(i, 1, ch, termbox.ColorYellow, termbox.ColorDefault)
	}

	// Draw ground
	for x := 0; x < g.width; x++ {
		termbox.SetCell(x, g.groundY()+1, '_', termbox.ColorGreen, termbox.ColorDefault)
	}

	// Draw obstacles
	for _, o := range g.obstacles {
		if o.x < 0 || o.x >= g.width {
			continue
		}
		for y := 1; y <= g.groundY(); y++ {
			if y >= o.gapY && y < o.gapY+g.gapHeight {
				continue
			}
			termbox.SetCell(o.x, y, '#', termbox.ColorRed, termbox.ColorDefault)
		}
	}

	// Draw helicopter (small icon, no trailing dot)
	if g.heliX >= 0 && g.heliX < g.width && g.heliY >= 1 && g.heliY <= g.groundY() {
		termbox.SetCell(g.heliX, g.heliY, '^', termbox.ColorCyan|termbox.AttrBold, termbox.ColorDefault)
	}

	if g.state == stateGameOver {
		msg := "GAME OVER - press R to restart or Q to quit"
		startX := (g.width - len(msg)) / 2
		y := g.height / 2
		for i, ch := range msg {
			x := startX + i
			if x >= 0 && x < g.width && y >= 0 && y < g.height {
				termbox.SetCell(x, y, ch, termbox.ColorWhite|termbox.AttrBold, termbox.ColorRed)
			}
		}
	}

	termbox.Flush()
}

// Simple integer to string to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}

func eventLoop() {
	if err := termbox.Init(); err != nil {
		log.Fatalf("failed to init termbox: %v", err)
	}
	defer termbox.Close()

	rand.Seed(time.Now().UnixNano())

	width, height := termbox.Size()
	if height < 10 || width < 30 {
		termbox.Close()
		log.Fatalf("terminal too small, need at least 30x10 (got %dx%d)", width, height)
	}

	g := newGame(width, height, 0)
	g.draw()

	events := make(chan termbox.Event, 20)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	ticker := time.NewTicker(time.Duration(g.tickMillis) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-events:
			switch ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyEsc:
					return
				case termbox.KeySpace:
					g.flap()
				default:
					// letter keys
					switch ev.Ch {
					case 'q', 'Q':
						return
					case 'r', 'R':
						if g.state == stateGameOver {
							best := g.bestScore
							g.reset(best)
							g.draw()
						}
					case ' ':
						g.flap()
					}
				}
			case termbox.EventResize:
				width, height = termbox.Size()
				g.width = width
				g.height = height
				g.reset(g.bestScore)
				g.draw()
			case termbox.EventInterrupt, termbox.EventError:
				return
			}
		case <-ticker.C:
			g.update()
			g.draw()
		}
	}
}

func main() {
	eventLoop()
}

