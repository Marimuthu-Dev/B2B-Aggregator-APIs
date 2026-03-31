package fitness

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/fitnesscert"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/storage"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"

	"gorm.io/gorm"
)

// Deps bundles dependencies for the fitness certificate worker.
type Deps struct {
	DB         *gorm.DB
	Blob       *storage.AzureMoUBlobService
	LeadRepo   repository.LeadRepository
	ClientRepo repository.ClientRepository
	Config     config.FitnessCertWorkerConfig
	Log        *slog.Logger
}

// RunLoop processes pending leads on an interval until ctx is cancelled.
func RunLoop(ctx context.Context, d Deps) error {
	log := d.Log
	if log == nil {
		log = slog.Default()
	}
	if d.Blob == nil {
		return fmt.Errorf("fitness worker: blob service is nil")
	}
	for {
		if err := RunOnce(ctx, d); err != nil {
			log.Error("fitness worker batch failed", slog.Any("err", err))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d.Config.PollInterval):
		}
	}
}

// RunOnce loads up to BatchSize pending leads and processes each.
func RunOnce(ctx context.Context, d Deps) error {
	log := d.Log
	if log == nil {
		log = slog.Default()
	}
	leads, err := d.LeadRepo.FindLeadsPendingFitCertification(d.Config.BatchSize, d.Config.PendingLeadStatusID)
	if err != nil {
		return fmt.Errorf("find pending leads: %w", err)
	}
	for _, lead := range leads {
		if err := processLead(ctx, d, lead, log); err != nil {
			log.Error("fitness cert lead failed",
				slog.Int64("leadID", lead.LeadID),
				slog.Any("err", err))
		}
	}
	return nil
}

func processLead(ctx context.Context, d Deps, lead domain.Lead, log *slog.Logger) error {
	client, err := d.ClientRepo.FindByID(lead.ClientID)
	if err != nil {
		return fmt.Errorf("load client %d: %w", lead.ClientID, err)
	}
	if client.ClientTypeID == nil {
		return fmt.Errorf("client %d has no ClientTypeID; add certificate template after setting type", lead.ClientID)
	}
	ctID := *client.ClientTypeID

	dateStr := time.Now().In(timeutil.ISTLocation()).Format("02-Jan-2006")
	html, err := fitnesscert.RenderCertificateHTML(d.Config.TemplateDir, ctID, fitnesscert.TemplateData{
		Name:    lead.PatientName,
		Company: client.ClientName,
		Date:    dateStr,
	})
	if err != nil {
		return err
	}

	certPDF, err := fitnesscert.HTMLToPDF(ctx, html, d.Config.ChromiumPath)
	if err != nil {
		return err
	}

	container, blobName, err := storage.ParseAzureBlobContainerAndBlob(lead.ReportURL)
	if err != nil {
		return fmt.Errorf("parse report url: %w", err)
	}

	reportPDF, err := d.Blob.DownloadBlob(ctx, container, blobName)
	if err != nil {
		return fmt.Errorf("download report: %w", err)
	}

	merged, err := fitnesscert.MergeCertificateFirst(certPDF, reportPDF)
	if err != nil {
		return err
	}

	if err := d.Blob.UploadDiagnosticReportPDFBytes(ctx, container, blobName, merged); err != nil {
		return fmt.Errorf("upload merged pdf: %w", err)
	}

	updated, err := d.LeadRepo.MarkFitCertificationGenerated(lead.LeadID, d.Config.ActorUserID,
		d.Config.PendingLeadStatusID, d.Config.DoneLeadStatusID)
	if err != nil {
		return fmt.Errorf("mark fit certification generated: %w", err)
	}
	if !updated {
		log.Warn("lead no longer matched pending fit-cert criteria after upload; blob was overwritten",
			slog.Int64("leadID", lead.LeadID))
		return nil
	}
	log.Info("fitness certificate merged and lead updated", slog.Int64("leadID", lead.LeadID))
	return nil
}
