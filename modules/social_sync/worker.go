package socialsync

import (
	"context"
	"log/slog"
	"time"
)

type Worker struct {
	Service     *SyncService
	Interval    time.Duration
	Logger      *slog.Logger
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

func NewWorker(service *SyncService, interval time.Duration, logger *slog.Logger) *Worker {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		Service:     service,
		Interval:    interval,
		Logger:      logger,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	go func() {
		defer close(w.stoppedChan)
		w.Logger.InfoContext(ctx, "starting background social sync worker", "interval", w.Interval.String())

		// Perform initial sync on startup
		if err := w.Service.Sync(ctx); err != nil {
			w.Logger.WarnContext(ctx, "initial social sync completed with warnings", "error", err)
		}

		ticker := time.NewTicker(w.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				w.Logger.InfoContext(ctx, "stopping background social sync worker (context cancelled)")
				return
			case <-w.stopChan:
				w.Logger.InfoContext(ctx, "stopping background social sync worker")
				return
			case <-ticker.C:
				if err := w.Service.Sync(ctx); err != nil {
					w.Logger.WarnContext(ctx, "scheduled social sync completed with warnings", "error", err)
				}
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stopChan)
	<-w.stoppedChan
}
