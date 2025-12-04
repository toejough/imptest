package main

import (
	"testing"
	"time"

	"github.com/toejough/imptest/POC-imptest-API/UAT/run"
	"github.com/toejough/imptest/POC-imptest-API/imptest"
)

//go:generate go run ../imptest/generator/main.go run.Tracker --name TrackerImp
//go:generate go run ../imptest/generator/main.go run.CoinFlipper --name CoinFlipperImp

func Test_PingPong_Match(t *testing.T) {
	trackerImp := NewTrackerImp(t)
	flipperImp := NewCoinFlipperImp(t)

	ping := run.NewPingPongPlayer(trackerImp.Mock, flipperImp.Mock, "Ping")
	pong := run.NewPingPongPlayer(trackerImp.Mock, flipperImp.Mock, "Pong")

	// Start both players
	pingInv := imptest.Start(t, ping.Play)
	pongInv := imptest.Start(t, pong.Play)

	// Register sequence (order might vary, but we can enforce one for the test or handle both)
	// For simplicity, we'll expect Ping then Pong or vice versa.
	// Since imptest is strict, we might need to be careful with concurrency.
	// However, since we control the scheduler via the mocks, we can serialize the events.
	// But wait, the players run in goroutines. They will race to Register.
	// To handle this deterministically in the test, we might need to rely on the fact that
	// imptest's ExpectCallTo blocks until the call happens.
	// Actually, `ExpectCallTo` *waits* for the call. So we can just say:
	// Use Within to handle non-deterministic order
	trackerImp.Within(1 * time.Second).ExpectCallTo.Register("Ping").Resolve()
	trackerImp.Within(1 * time.Second).ExpectCallTo.Register("Pong").Resolve()

	// Ping serves
	trackerImp.ExpectCallTo.IsServing("Ping").InjectResult(true)
	trackerImp.ExpectCallTo.Hit("Ping").Resolve()

	// Pong receives (not serving)
	trackerImp.ExpectCallTo.IsServing("Pong").InjectResult(false)
	// Pong waits for receive

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

	// Ping receives game over
	trackerImp.ExpectCallTo.Receive("Ping").InjectResult(false)
	// Ping should return now

	// Verify both finished
	pingInv.ExpectReturnedValues()
	pongInv.ExpectReturnedValues()
}
