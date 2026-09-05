package whatsapp

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/whatsapp"
)

// Deps bundles dependencies for the WhatsApp worker (mirrors email.Deps).
type Deps struct {
	Repo   *repository.WhatsAppRepository
	Sender *whatsapp.Service
	Config config.WhatsAppWorkerConfig
	Log    *slog.Logger
}

// RunLoop loads pending batches, sends via WhatsApp API, and sleeps until ctx is cancelled.
func RunLoop(ctx context.Context, d Deps) error {
	log := d.Log
	if log == nil {
		log = slog.Default()
	}
	if d.Repo == nil {
		return errors.New("whatsapp worker: repository is nil")
	}
	if d.Sender == nil {
		return errors.New("whatsapp worker: WhatsApp sender is nil")
	}
	log.Info("whatsapp worker loop running",
		slog.Int("batchSize", d.Config.BatchSize),
		slog.Duration("pollIntervalAfterWork", d.Config.PollInterval),
		slog.Duration("idleWaitWhenEmpty", d.Config.IdleWait),
		slog.Duration("sendTimeout", d.Config.SendTimeout),
	)
	for {
		if err := ctx.Err(); err != nil {
			log.Info("whatsapp worker stopping", slog.String("reason", err.Error()))
			return err
		}
		foundRows, err := RunOnce(ctx, d)
		if err != nil {
			log.Error("whatsapp worker batch failed", slog.String("error", err.Error()))
			// If rate limited, wait longer before retrying
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "rate limited") {
				log.Info("rate limit detected, waiting longer before retry")
				wait := 10 * time.Minute // Wait 10 minutes for rate limit to reset
				log.Info("next iteration scheduled",
					slog.Duration("wait", wait),
					slog.Bool("hadRowsInPreviousCycle", foundRows),
					slog.String("reason", "rate limit"),
				)
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					if !timer.Stop() {
						<-timer.C
					}
					log.Info("whatsapp worker stopping", slog.String("reason", ctx.Err().Error()))
					return ctx.Err()
				case <-timer.C:
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
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			log.Info("whatsapp worker stopping", slog.String("reason", ctx.Err().Error()))
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// RunOnce reads up to BatchSize pending rows and processes each (returns foundRows true if any were returned).
func RunOnce(ctx context.Context, d Deps) (foundRows bool, err error) {
	log := d.Log
	if log == nil {
		log = slog.Default()
	}
	whatsApps, err := d.Repo.SelectPendingBatch(ctx, d.Config.BatchSize)
	if err != nil {
		return false, err
	}
	if len(whatsApps) == 0 {
		return false, nil
	}
	
	rateLimited := false
	for _, w := range whatsApps {
		log.Info("processing whatsapp", slog.Int64("whatsappID", w.WhatsappID))
		sendCtx, cancel := context.WithTimeout(ctx, d.Config.SendTimeout)
		sendErr := d.Sender.SendMessage(sendCtx, w)
		cancel()
		if sendErr == nil {
			if err := d.Repo.MarkSent(ctx, w.WhatsAppID); err != nil {
				log.Error("mark sent failed",
					slog.Int64("whatsappID", w.WhatsAppID),
					slog.String("error", err.Error()),
				)
				continue
			}
			log.Info("whatsapp sent", slog.Int64("whatsappID", w.WhatsappID))
			continue
		}
		
		// Check if this is a rate limit error
		if strings.Contains(sendErr.Error(), "rate limited") || strings.Contains(sendErr.Error(), "TooManyRequests") {
			log.Warn("whatsapp rate limited by API",
				slog.Int64("whatsappID", w.WhatsappID),
				slog.String("error", sendErr.Error()),
			)
			rateLimited = true
			// Don't mark as failure - keep it for retry after rate limit expires
			continue
		}
		
		log.Error("send failed",
			slog.Int64("whatsappID", w.WhatsappID),
			slog.String("error", sendErr.Error()),
		)
		if err := d.Repo.MarkAfterFailure(ctx, w.WhatsAppID); err != nil {
			log.Error("mark after failure",
				slog.Int64("whatsappID", w.WhatsappID),
				slog.String("error", err.Error()),
			)
		}
	}
	
	// If rate limited, return an error to trigger longer wait in the main loop
	if rateLimited {
		return true, errors.New("WhatsApp API rate limit exceeded")
	}
	
	return true, nil
}
