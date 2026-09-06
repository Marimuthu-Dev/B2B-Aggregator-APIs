package email

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/acsemail"
	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/repository"
)

// Deps bundles dependencies for the email outbox worker (mirrors fitness.Deps).
type Deps struct {
	Repo   *repository.EmailOutboxRepository
	Sender *acsemail.Service
	Config config.EmailWorkerConfig
	Log    *slog.Logger
}

// RunLoop loads pending batches, sends via ACS, and sleeps until ctx is cancelled.
func RunLoop(ctx context.Context, d Deps) error {
	log := d.Log
	if log == nil {
		log = slog.Default()
	}
	if d.Repo == nil {
		return errors.New("email worker: repository is nil")
	}
	if d.Sender == nil {
		return errors.New("email worker: ACS sender is nil")
	}
	log.Info("email worker loop running",
		slog.Int("batchSize", d.Config.BatchSize),
		slog.Duration("pollIntervalAfterWork", d.Config.PollInterval),
		slog.Duration("idleWaitWhenEmpty", d.Config.IdleWait),
		slog.Duration("sendTimeout", d.Config.SendTimeout),
	)
	for {
		if err := ctx.Err(); err != nil {
			log.Info("email worker stopping", slog.String("reason", err.Error()))
			return err
		}
		foundRows, err := RunOnce(ctx, d)
		if err != nil {
			log.Error("email worker batch failed", slog.String("error", err.Error()))
			// If rate limited, wait longer before retrying
			if strings.Contains(err.Error(), "rate limit") {
				log.Info("rate limit detected, waiting longer before retry")
				wait := 10 * time.Minute // Wait 10 minutes for rate limit to reset
				log.Info("next iteration scheduled",
					slog.Duration("wait", wait),
					slog.Bool("hadRowsInPreviousCycle", foundRows),
					slog.String("reason", "rate limit"),
				)
				// Use a ticker to print a heartbeat every minute to prevent Azure WebJob idle timeout (WEBJOBS_IDLE_TIMEOUT)
				ticker := time.NewTicker(1 * time.Minute)
				timer := time.NewTimer(wait)
			WaitLoop:
				for {
					select {
					case <-ctx.Done():
						timer.Stop()
						ticker.Stop()
						log.Info("email worker stopping", slog.String("reason", ctx.Err().Error()))
						return ctx.Err()
					case <-ticker.C:
						log.Info("heartbeat: waiting for rate limit to reset...")
					case <-timer.C:
						ticker.Stop()
						break WaitLoop
					}
				}
				continue
			}
		}
		wait := d.Config.IdleWait
		if foundRows {
			wait = d.Config.PollInterval
		}
		log.Info("next iteration scheduled",
			slog.Duration("wait", wait),
			slog.Bool("hadRowsInPreviousCycle", foundRows),
		)
		// Use a ticker to print a heartbeat every minute to prevent Azure WebJob idle timeout (WEBJOBS_IDLE_TIMEOUT)
		ticker := time.NewTicker(1 * time.Minute)
		timer := time.NewTimer(wait)
	NormalWaitLoop:
		for {
			select {
			case <-ctx.Done():
				timer.Stop()
				ticker.Stop()
				log.Info("email worker stopping", slog.String("reason", ctx.Err().Error()))
				return ctx.Err()
			case <-ticker.C:
				log.Info("heartbeat: waiting for next polling cycle...")
			case <-timer.C:
				ticker.Stop()
				break NormalWaitLoop
			}
		}
	}
}

// RunOnce reads up to BatchSize pending rows and processes each (returns foundRows true if any were returned).
func RunOnce(ctx context.Context, d Deps) (foundRows bool, err error) {
	log := d.Log
	if log == nil {
		log = slog.Default()
	}
	emails, err := d.Repo.SelectPendingBatch(ctx, d.Config.BatchSize)
	if err != nil {
		return false, err
	}
	if len(emails) == 0 {
		return false, nil
	}
	
	rateLimited := false
	for _, e := range emails {
		log.Info("processing email", slog.Int64("emailID", e.EmailID))
		sendCtx, cancel := context.WithTimeout(ctx, d.Config.SendTimeout)
		sendErr := d.Sender.SendHTML(sendCtx, e)
		cancel()
		if sendErr == nil {
			if err := d.Repo.MarkSent(ctx, e.EmailID); err != nil {
				log.Error("mark sent failed",
					slog.Int64("emailID", e.EmailID),
					slog.String("error", err.Error()),
				)
				continue
			}
			log.Info("email sent", slog.Int64("emailID", e.EmailID))
			continue
		}
		
		// Check if this is a rate limit error
		if strings.Contains(sendErr.Error(), "rate limited") || strings.Contains(sendErr.Error(), "TooManyRequests") {
			log.Warn("email rate limited by ACS",
				slog.Int64("emailID", e.EmailID),
				slog.String("error", sendErr.Error()),
			)
			rateLimited = true
			// Don't mark as failure - keep it for retry after rate limit expires
			continue
		}
		
		log.Error("send failed",
			slog.Int64("emailID", e.EmailID),
			slog.String("error", sendErr.Error()),
		)
		if err := d.Repo.MarkAfterFailure(ctx, e.EmailID); err != nil {
			log.Error("mark after failure",
				slog.Int64("emailID", e.EmailID),
				slog.String("error", err.Error()),
			)
		}
	}
	
	// If rate limited, return an error to trigger longer wait in the main loop
	if rateLimited {
		return true, errors.New("ACS rate limit exceeded")
	}
	
	return true, nil
}
