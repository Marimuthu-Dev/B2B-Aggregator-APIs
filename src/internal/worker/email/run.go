package email

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strconv"
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
		log.Info("starting new run cycle, checking for pending emails")
		foundRows, err := RunOnce(ctx, d)
		if err != nil {
			log.Error("email worker batch failed", slog.String("error", err.Error()))
			// If rate limited, wait longer before retrying
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "TooManyRequests") {
				log.Info("rate limit detected, waiting longer before retry")
				wait := d.Config.RateLimitWait
				if wait <= 0 {
					wait = 60 * time.Minute
				}
				// Try to extract seconds from ACS error message e.g., "Please try again after 3703 seconds"
				re := regexp.MustCompile(`(?i)after (\d+) seconds`)
				if matches := re.FindStringSubmatch(err.Error()); len(matches) > 1 {
					if sec, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
						// Add 5 seconds buffer to ensure the limit has expired
						wait = time.Duration(sec+5) * time.Second
					}
				}
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
						timer.Stop()
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
	log.Info("querying database for pending emails", slog.Int("batchSize", d.Config.BatchSize))
	emails, err := d.Repo.SelectPendingBatch(ctx, d.Config.BatchSize)
	if err != nil {
		return false, err
	}
	if len(emails) == 0 {
		log.Info("no pending emails found in this cycle")
		return false, nil
	}
	
	log.Info("successfully fetched pending emails", slog.Int("count", len(emails)))
	
	rateLimited := false
	var rateLimitErr error
	for i, e := range emails {
		if i > 0 && d.Config.InterSendDelay > 0 {
			select {
			case <-ctx.Done():
				return true, ctx.Err()
			case <-time.After(d.Config.InterSendDelay):
			}
		}

		log.Info("processing email", slog.Int64("emailID", e.EmailID))
		log.Info("attempting to send email via ACS", slog.Int64("emailID", e.EmailID))
		sendCtx, cancel := context.WithTimeout(ctx, d.Config.SendTimeout)
		sendErr := d.Sender.SendHTML(sendCtx, e)
		cancel()
		if sendErr == nil {
			log.Info("ACS successfully accepted email, marking as sent in database", slog.Int64("emailID", e.EmailID))
			if err := d.Repo.MarkSent(ctx, e.EmailID); err != nil {
				log.Error("mark sent failed in database",
					slog.Int64("emailID", e.EmailID),
					slog.String("error", err.Error()),
				)
				continue
			}
			log.Info("email processing completed successfully", slog.Int64("emailID", e.EmailID))
			continue
		}
		
		// Check if this is a rate limit error
		if strings.Contains(sendErr.Error(), "rate limited") || strings.Contains(sendErr.Error(), "TooManyRequests") {
			log.Warn("email rate limited by ACS",
				slog.Int64("emailID", e.EmailID),
				slog.String("error", sendErr.Error()),
			)
			rateLimited = true
			rateLimitErr = sendErr
			// Don't mark as failure - keep it for retry after rate limit expires
			break
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
		return true, errors.New("ACS rate limit exceeded: " + rateLimitErr.Error())
	}
	
	return true, nil
}
