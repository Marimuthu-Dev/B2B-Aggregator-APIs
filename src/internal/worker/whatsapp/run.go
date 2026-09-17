package whatsapp

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/whatsapp"
)

type Deps struct {
	Repo         *repository.WhatsAppRepository
	TemplateRepo *repository.WhatsAppTemplateRepository
	Sender       *whatsapp.Service
	Config       config.WhatsAppWorkerConfig
	Log          *slog.Logger
}

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
		slog.Bool("templateRepoAvailable", d.TemplateRepo != nil),
	)
	for {
		if err := ctx.Err(); err != nil {
			log.Info("whatsapp worker stopping", slog.String("reason", err.Error()))
			return err
		}
		foundRows, err := RunOnce(ctx, d)
		if err != nil {
			log.Error("whatsapp worker batch failed", slog.String("error", err.Error()))
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "rate limited") {
				log.Info("rate limit detected, waiting longer before retry")
				wait := 10 * time.Minute
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

// RunOnce reads up to BatchSize pending rows and processes each.
// When TemplateRepo is configured, missing TemplateName / TemplateType are looked up
// from tbl_WhatsAppTemplates using the row's TemplateID before sending.
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
		resolved := resolveMessageMetadata(ctx, d.TemplateRepo, &w, d.Config.DefaultTemplateName)

		log.Info("processing whatsapp",
			slog.Int64("whatsappID", w.WhatsAppID),
			slog.Int64("templateID", w.TemplateID),
			slog.String("templateName", w.TemplateName),
			slog.String("templateType", w.TemplateType),
			slog.String("toMobile", w.ToMobile),
			slog.String("resolvedFrom", resolved),
		)

		sendCtx, cancel := context.WithTimeout(ctx, d.Config.SendTimeout)
		sendErr := d.Sender.SendMessage(sendCtx, w)
		cancel()
		if sendErr == nil {
			if err := d.Repo.MarkSent(ctx, w.WhatsAppID); err != nil {
				log.Error("mark sent failed",
					slog.Int64("whatsappID", w.WhatsAppID),
					slog.String("templateName", w.TemplateName),
					slog.String("error", err.Error()),
				)
				continue
			}
			log.Info("whatsapp sent",
				slog.Int64("whatsappID", w.WhatsAppID),
				slog.String("templateName", w.TemplateName),
				slog.String("templateType", w.TemplateType),
			)
			continue
		}

		if strings.Contains(sendErr.Error(), "rate limited") || strings.Contains(sendErr.Error(), "TooManyRequests") {
			log.Warn("whatsapp rate limited by API",
				slog.Int64("whatsappID", w.WhatsAppID),
				slog.String("templateName", w.TemplateName),
				slog.String("error", sendErr.Error()),
			)
			rateLimited = true
			continue
		}

		log.Error("send failed",
			slog.Int64("whatsappID", w.WhatsAppID),
			slog.Int64("templateID", w.TemplateID),
			slog.String("templateName", w.TemplateName),
			slog.String("templateType", w.TemplateType),
			slog.String("error", sendErr.Error()),
		)
		if err := d.Repo.MarkAfterFailure(ctx, w.WhatsAppID); err != nil {
			log.Error("mark after failure",
				slog.Int64("whatsappID", w.WhatsAppID),
				slog.String("error", err.Error()),
			)
		}
	}

	if rateLimited {
		return true, errors.New("WhatsApp API rate limit exceeded")
	}

	return true, nil
}

// resolveMessageMetadata ensures TemplateName and TemplateType are populated on *msg.
// It returns a short human-readable source describing how the metadata was resolved.
// Resolution order:
//  1. Row already has both TemplateName + TemplateType -> "row"
//  2. Row has TemplateID (but missing name/type) and TemplateRepo is available -> "tpl_by_id"
//  3. Fallback to config.DefaultTemplateName with 'regular' type -> "config_default"
func resolveMessageMetadata(
	ctx context.Context,
	tplRepo *repository.WhatsAppTemplateRepository,
	msg *domain.OutboxWhatsApp,
	defaultTemplateName string,
) string {
	hasName := strings.TrimSpace(msg.TemplateName) != ""
	hasType := strings.TrimSpace(msg.TemplateType) != ""

	if hasName && hasType {
		return "row"
	}

	if msg.TemplateID != 0 && tplRepo != nil {
		if tpl, err := tplRepo.FindByID(ctx, msg.TemplateID); err == nil && tpl != nil {
			if !hasName {
				msg.TemplateName = tpl.TemplateName
			}
			if !hasType {
				msg.TemplateType = tpl.TemplateType
			}
			return "tpl_by_id"
		}
	}

	if !hasName {
		msg.TemplateName = strings.TrimSpace(defaultTemplateName)
	}
	if !hasType {
		msg.TemplateType = domain.WhatsAppTemplateTypeRegular
	}
	return "config_default"
}
