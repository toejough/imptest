package run

type Tracker interface {
	Register(name string)
	IsServing(name string) bool
	Hit(name string)
	Miss(name string)
	// Receive returns true if a ball is received, false if the game is over
	Receive(name string) bool
}

type CoinFlipper interface {
	Flip() bool
}

type PingPongPlayer struct {
	tracker Tracker
	flipper CoinFlipper
	name    string
	Done    chan struct{}
}

func NewPingPongPlayer(tracker Tracker, flipper CoinFlipper, name string) *PingPongPlayer {
	return &PingPongPlayer{
		tracker: tracker,
		flipper: flipper,
		name:    name,
		Done:    make(chan struct{}),
	}
}

func (p *PingPongPlayer) Play() {
	defer close(p.Done)
	p.tracker.Register(p.name)

	if p.tracker.IsServing(p.name) {
		p.tracker.Hit(p.name)
	}

	for {
		// Wait to receive
		gotBall := p.tracker.Receive(p.name)
		if !gotBall {
			// Game over
			return
		}

		// Got ball, flip coin
		if p.flipper.Flip() {
			// Heads = Hit
			p.tracker.Hit(p.name)
		} else {
			// Tails = Miss
			p.tracker.Miss(p.name)
			return
		}
	}
}
