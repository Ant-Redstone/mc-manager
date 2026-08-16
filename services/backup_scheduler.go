package services

import (
	"log/slog"
	"time"
)

// reloadCh signals the scheduler goroutine to re-read config and restart its
// ticker. Buffered so UpdateBackupConfigHandler never blocks sending to it.
var reloadCh = make(chan struct{}, 1)

// StartBackupScheduler launches the background goroutine that runs automatic
// backups according to the persisted BackupConfig. It should be called once
// at startup, after db.Init.
func StartBackupScheduler() {
	go runBackupScheduler()
}

// NotifyBackupConfigChanged tells the running scheduler to reload its
// config immediately, instead of waiting for the next tick.
func NotifyBackupConfigChanged() {
	select {
	case reloadCh <- struct{}{}:
	default:
		// A reload is already pending; no need to queue another.
	}
}

func runBackupScheduler() {
	var ticker *time.Ticker

	stopTicker := func() {
		if ticker != nil {
			ticker.Stop()
			ticker = nil
		}
	}
	defer stopTicker()

	applyConfig := func() *time.Ticker {
		cfg, err := LoadBackupConfig()
		if err != nil {
			slog.Error("backup scheduler: failed to load config", "err", err)
			return nil
		}
		if !cfg.Enabled {
			return nil
		}
		return time.NewTicker(time.Duration(cfg.IntervalMinutes) * time.Minute)
	}

	stopTicker()
	ticker = applyConfig()

	for {
		var tickCh <-chan time.Time
		if ticker != nil {
			tickCh = ticker.C
		}

		select {
		case <-reloadCh:
			stopTicker()
			ticker = applyConfig()

		case <-tickCh:
			runScheduledBackup()
		}
	}
}

// runScheduledBackup creates a backup and prunes old ones according to the
// current config. It's a no-op (skips, doesn't crash the loop) if a manual
// backup is already in progress — backupMu just makes it wait its turn.
func runScheduledBackup() {
	slog.Info("backup scheduler: starting scheduled backup")

	info, err := CreateBackup()
	if err != nil {
		slog.Error("backup scheduler: scheduled backup failed", "err", err)
		return
	}
	slog.Info("backup scheduler: backup created", "backup", info.Name, "bytes", info.Size)

	cfg, err := LoadBackupConfig()
	if err != nil {
		slog.Error("backup scheduler: failed to load config for pruning", "err", err)
		return
	}

	if err := PruneBackups(cfg.Keep); err != nil {
		slog.Error("backup scheduler: pruning failed", "err", err)
	}
}
