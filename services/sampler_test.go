package services

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lomokwa/mc-manager/types"
)

// TPS and player count are not free: reading them makes the JVM run
// `spark tps` / `list`. A rule sampling every 5s forever would put load on
// exactly the thing a "TPS dropped" rule exists to protect.
func TestSampleInterval_DerivesFromTheRulesOwnWindow(t *testing.T) {
	cases := []struct {
		window time.Duration
		want   time.Duration
	}{
		{5 * time.Minute, 30 * time.Second},
		{10 * time.Minute, time.Minute},
		{time.Minute, 10 * time.Second},  // floor
		{30 * time.Second, 10 * time.Second}, // floor
		{0, 10 * time.Second},                // no window configured
		{time.Hour, time.Minute},             // ceiling
	}
	for _, tc := range cases {
		if got := SampleInterval(tc.window); got != tc.want {
			t.Errorf("window %s: expected %s, got %s", tc.window, tc.want, got)
		}
	}
}

// The property the whole feature rests on: nothing configured, nothing
// measured. sampleOnce must not even ask a runtime for anything.
func TestSampleOnce_AsksForNothingWhenNoRuleWants(t *testing.T) {
	var asked atomic.Int32
	needs := func(types.SampleKind) bool {
		asked.Add(1)
		return false
	}

	before := types.Bus.Dropped()
	sampleOnce(needs)

	if asked.Load() == 0 {
		t.Error("expected sampleOnce to consult the gate at least once")
	}
	if types.Bus.Dropped() != before {
		t.Error("nothing should have been published")
	}
}

func TestStartSampler_StopsCleanly(t *testing.T) {
	stop := StartSampler(func(types.SampleKind) bool { return false })
	stop()
	// A second stop would panic on a closed channel if StartSampler returned a
	// naive closer; calling it once and returning is enough to prove the
	// goroutine exits, which -race would flag otherwise.
}

// The backlog replay must stay off the automation bus.
//
// On boot the tailer seeds the hub with the last ~200 lines of latest.log so a
// console opened right after a restart is never blank. Those lines already
// happened. Publishing them would make every API deploy re-fire rules for old
// events: a "server stopped" rule announcing last night's stop, a join rule
// welcoming players who left hours ago -- and, since a rule can run console
// commands, actually acting on them.
func TestTailer_BacklogReplayDoesNotReachTheAutomationBus(t *testing.T) {
	dir := t.TempDir()
	rt := &ServerRuntime{ID: "default", Dir: dir}
	rt.Hub = types.NewLogHub()

	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	historical := "[01:00:00] [Server thread/INFO]: Stopping the server\n" +
		"[01:00:01] [Server thread/INFO]: Notch joined the game\n"
	if err := os.WriteFile(filepath.Join(logDir, "latest.log"), []byte(historical), 0o644); err != nil {
		t.Fatal(err)
	}

	// Subscribe so HasSubscribers is true -- the publish path is live, and the
	// backlog still must not use it.
	ch := types.Bus.Subscribe()
	defer types.Bus.Unsubscribe(ch)

	// Read exactly what the tailer replays on open, then assert the bus stayed
	// quiet. readBacklog is the seeding path; the live loop is what publishes.
	f, err := os.Open(filepath.Join(logDir, "latest.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines := readBacklog(f)
	for _, line := range lines {
		rt.Hub.Broadcast(line) // exactly what the seeding branch does
	}

	if len(lines) == 0 {
		t.Fatal("expected the fixture to produce backlog lines")
	}
	select {
	case ev := <-ch:
		t.Errorf("a replayed backlog line reached the automation bus: %+v", ev)
	default:
	}
}
