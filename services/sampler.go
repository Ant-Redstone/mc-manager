package services

import (
	"time"

	"github.com/lomokwa/mc-manager/types"
)

const (
	// Sampling floor and ceiling. The floor stops a short window from turning
	// into a tight poll; the ceiling stops a long one from being so lazy the
	// rule feels broken.
	minSampleInterval = 10 * time.Second
	maxSampleInterval = time.Minute
)

// SampleInterval picks how often to measure, given the window a rule cares
// about.
//
// This is the one number in the feature that costs the JVM rather than the API:
// reading TPS means running `spark tps` on the game server, and reading the
// player count means running `list`. "TPS below 15 for 5 minutes" decides
// identically at 30s samples and costs a tenth as much -- so the interval comes
// from the rule's own window instead of a fixed fast poll.
func SampleInterval(window time.Duration) time.Duration {
	if window <= 0 {
		return minSampleInterval
	}
	interval := window / 10
	if interval < minSampleInterval {
		return minSampleInterval
	}
	if interval > maxSampleInterval {
		return maxSampleInterval
	}
	return interval
}

// StartSampler polls only the measurements some enabled rule asks for, and
// returns a stop function.
//
// needs is the engine's NeedsSampling, read live on every tick: turning a rule
// off has to stop its cost immediately, not at the next restart.
func StartSampler(needs func(types.SampleKind) bool) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(minSampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sampleOnce(needs)
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}

func sampleOnce(needs func(types.SampleKind) bool) {
	wantPlayers := needs(types.SamplePlayerCount)
	if !wantPlayers {
		// TPS and disk are gated the same way once implemented; today this is
		// the only measurement taken, so a tick with nothing to do costs one
		// map read and returns.
		return
	}

	for _, rt := range AllRuntimes() {
		if !rt.IsServerRunning() {
			continue // asking a stopped server for its player list is pointless
		}
		// Reuses GetOnlinePlayers' existing 10s cache rather than opening a
		// second path to `list`.
		names, err := rt.GetOnlinePlayers()
		if err != nil {
			continue
		}
		types.Bus.Publish(types.SampleEvent{
			ServerID: rt.ID,
			Kind:     types.SamplePlayerCount,
			Value:    float64(len(names)),
			At:       time.Now(),
		})
	}
}
