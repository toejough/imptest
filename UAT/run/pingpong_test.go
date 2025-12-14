package run_test

import (
	"testing"
	"time"

	"github.com/toejough/imptest/UAT/run"
)

//go:generate go run ../../impgen/main.go run.Tracker --name TrackerImp
//go:generate go run ../../impgen/main.go run.CoinFlipper --name CoinFlipperImp
//go:generate go run ../../impgen/main.go run.PingPongPlayer.Play --name PingPongPlayerImp --call

func Test_PingPong_Match(t *testing.T) {
	t.Parallel()
	trackerImp := NewTrackerImp(t)
	flipperImp := NewCoinFlipperImp(t)

	ping := run.NewPingPongPlayer(trackerImp.Mock, flipperImp.Mock, "Ping")
	pong := run.NewPingPongPlayer(trackerImp.Mock, flipperImp.Mock, "Pong")

	// Start both players
	pingInv := NewPingPongPlayerImp(t, ping.Play).Start()
	pongInv := NewPingPongPlayerImp(t, pong.Play).Start()
	// pingInv := imptest.Start(t, ping.Play)
	// pongInv := imptest.Start(t, pong.Play)

	// Expect Ping then Pong or vice versa.
	// Use Within to handle non-deterministic order
	trackerImp.Within(1 * time.Second).ExpectCallTo.Register("Ping").Resolve()
	trackerImp.Within(1 * time.Second).ExpectCallTo.Register("Pong").Resolve()

	// Ping and Pong are registered, and both will try to serve first.
	// Use Within to handle non-deterministic order
	trackerImp.Within(1 * time.Second).ExpectCallTo.IsServing("Ping").InjectResult(true)
	trackerImp.Within(1 * time.Second).ExpectCallTo.IsServing("Pong").InjectResult(false)

	// from here on, even though we are running ping and pong concurrently,
	// the sequence of calls is deterministic, so we can just expect them in order

	// 1st serve by Ping
	// serves
	trackerImp.ExpectCallTo.Hit("Ping").Resolve()

	// Pong receives the serve
	trackerImp.ExpectCallTo.Receive("Pong").InjectResult(true)
	// Pong flips -> Heads (Hit)
	flipperImp.ExpectCallTo.Flip().InjectResult(true)
	// Pong hits back
	trackerImp.ExpectCallTo.Hit("Pong").Resolve()

	// Ping receives
	trackerImp.ExpectCallTo.Receive("Ping").InjectResult(true)
	// Ping flips -> Heads (Hit)
	flipperImp.ExpectCallTo.Flip().InjectResult(true)
	// Ping hits back
	trackerImp.ExpectCallTo.Hit("Ping").Resolve()

	// Pong receives
	trackerImp.ExpectCallTo.Receive("Pong").InjectResult(true)
	// Pong flips -> Tails (Miss)
	flipperImp.ExpectCallTo.Flip().InjectResult(false)
	// Pong misses
	trackerImp.ExpectCallTo.Miss("Pong").Resolve()
	// Pong should return now
	pongInv.ExpectReturnedValues()

	// Ping receives game over
	trackerImp.ExpectCallTo.Receive("Ping").InjectResult(false)
	// Ping should return now
	pingInv.ExpectReturnedValues()
}
