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
	trackerImp.Within(1 * time.Second).ExpectCallIs.Register().ExpectArgsAre("Ping").Resolve()
	trackerImp.Within(1 * time.Second).ExpectCallIs.Register().ExpectArgsAre("Pong").Resolve()

	// Ping and Pong are registered, and both will try to serve first.
	// Use Within to handle non-deterministic order
	trackerImp.Within(1 * time.Second).ExpectCallIs.IsServing().ExpectArgsAre("Ping").InjectResult(true)
	trackerImp.Within(1 * time.Second).ExpectCallIs.IsServing().ExpectArgsAre("Pong").InjectResult(false)

	// from here on, even though we are running ping and pong concurrently,
	// the sequence of calls is deterministic, so we can just expect them in order

	// 1st serve by Ping
	// serves
	trackerImp.ExpectCallIs.Hit().ExpectArgsAre("Ping").Resolve()

	// Pong receives the serve
	trackerImp.ExpectCallIs.Receive().ExpectArgsAre("Pong").InjectResult(true)
	// Pong flips -> Heads (Hit)
	flipperImp.ExpectCallIs.Flip().InjectResult(true)
	// Pong hits back
	trackerImp.ExpectCallIs.Hit().ExpectArgsAre("Pong").Resolve()

	// Ping receives
	trackerImp.ExpectCallIs.Receive().ExpectArgsAre("Ping").InjectResult(true)
	// Ping flips -> Heads (Hit)
	flipperImp.ExpectCallIs.Flip().InjectResult(true)
	// Ping hits back
	trackerImp.ExpectCallIs.Hit().ExpectArgsAre("Ping").Resolve()

	// Pong receives
	trackerImp.ExpectCallIs.Receive().ExpectArgsAre("Pong").InjectResult(true)
	// Pong flips -> Tails (Miss)
	flipperImp.ExpectCallIs.Flip().InjectResult(false)
	// Pong misses
	trackerImp.ExpectCallIs.Miss().ExpectArgsAre("Pong").Resolve()
	// Pong should return now
	pongInv.ExpectReturnedValuesAre()

	// Ping receives game over
	trackerImp.ExpectCallIs.Receive().ExpectArgsAre("Ping").InjectResult(false)
	// Ping should return now
	pingInv.ExpectReturnedValuesAre()
}
