package fitness

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
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
	log.Info("fitness worker loop running",
		slog.String("pollInterval", d.Config.PollInterval.String()),
		slog.Int("batchSize", d.Config.BatchSize),
	)
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
	log.Info("fitness worker batch: querying pending leads",
		slog.Int("batchSize", d.Config.BatchSize),
		slog.Int("pendingLeadStatusID", int(d.Config.PendingLeadStatusID)),
	)
	leads, err := d.LeadRepo.FindLeadsPendingFitCertification(d.Config.BatchSize, d.Config.PendingLeadStatusID)
	if err != nil {
		return fmt.Errorf("find pending leads: %w", err)
	}
	log.Info("fitness worker batch: query complete", slog.Int("leadCount", len(leads)))
	if len(leads) == 0 {
		log.Info("fitness worker batch: no pending leads")
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
	log.Info("fitness cert: start lead",
		slog.Int64("leadID", lead.LeadID),
		slog.Int64("clientID", lead.ClientID),
		slog.String("patientName", lead.PatientName),
	)
	client, err := d.ClientRepo.FindByID(lead.ClientID)
	if err != nil {
		return fmt.Errorf("load client %d: %w", lead.ClientID, err)
	}
	log.Info("fitness cert: loaded client",
		slog.Int64("clientID", client.ClientID),
		slog.String("clientName", client.ClientName),
	)
	if client.ClientTypeID == nil {
		if !lead.IsFitCertificateTobeGenerated {
			log.Info("fitness cert: skipping certificate (IsFitCertificateTobeGenerated=false, client has no ClientTypeID)",
				slog.Int64("leadID", lead.LeadID),
			)
			updated, err := d.LeadRepo.MarkReportReadyToDownload(lead.LeadID, d.Config.ActorUserID,
				d.Config.PendingLeadStatusID, d.Config.DoneLeadStatusID)
			if err != nil {
				return fmt.Errorf("mark report ready to download: %w", err)
			}
			if !updated {
				log.Warn("lead no longer matched pending report-ready criteria", slog.Int64("leadID", lead.LeadID))
				return nil
			}
			log.Info("report marked ready to download without certificate generation", slog.Int64("leadID", lead.LeadID))
			return nil
		}
		return fmt.Errorf("client %d has no ClientTypeID; add certificate template after setting type", lead.ClientID)
	}
	ctID := *client.ClientTypeID
	log.Info("fitness cert: client type", slog.Int("clientTypeID", int(ctID)))
	// Skip PDF certificate when (client type is not 1 or 2) OR (IsFitCertificateTobeGenerated is false).
	if !requiresFitnessCertificate(ctID) || !lead.IsFitCertificateTobeGenerated {
		log.Info("fitness cert: skipping certificate (client type not 1/2 or IsFitCertificateTobeGenerated=false)",
			slog.Int64("leadID", lead.LeadID),
			slog.Int("clientTypeID", int(ctID)),
			slog.Bool("isFitCertificateTobeGenerated", lead.IsFitCertificateTobeGenerated),
		)
		updated, err := d.LeadRepo.MarkReportReadyToDownload(lead.LeadID, d.Config.ActorUserID,
			d.Config.PendingLeadStatusID, d.Config.DoneLeadStatusID)
		if err != nil {
			return fmt.Errorf("mark report ready to download: %w", err)
		}
		if !updated {
			log.Warn("lead no longer matched pending report-ready criteria",
				slog.Int64("leadID", lead.LeadID))
			return nil
		}
		log.Info("report marked ready to download without certificate generation", slog.Int64("leadID", lead.LeadID))
		return nil
	}

	tplPath := filepath.Join(d.Config.TemplateDir, fmt.Sprintf("certificate_%d.html", ctID))
	log.Info("fitness cert: selected HTML template",
		slog.String("templatePath", tplPath),
		slog.Int("clientTypeID", int(ctID)),
	)

	dateStr := time.Now().In(timeutil.ISTLocation()).Format("02-Jan-2006")
	html, err := fitnesscert.RenderCertificateHTML(d.Config.TemplateDir, ctID, fitnesscert.TemplateData{
		Name:    lead.PatientName,
		Company: client.ClientName,
		Date:    dateStr,
	})
	if err != nil {
		return fmt.Errorf("render certificate html from %s: %w", tplPath, err)
	}
	log.Info("fitness cert: HTML render OK",
		slog.String("templatePath", tplPath),
		slog.Int("htmlBytes", len(html)),
	)

	chromium := d.Config.ChromiumPath
	if chromium == "" {
		chromium = "(search PATH)"
	}
	log.Info("fitness cert: HTML to PDF via Chromium",
		slog.String("chromium", chromium),
		slog.String("templateDir", d.Config.TemplateDir),
	)
	certPDF, err := fitnesscert.HTMLToPDF(ctx, html, d.Config.ChromiumPath, d.Config.TemplateDir)
	if err != nil {
		return fmt.Errorf("html to pdf: %w", err)
	}
	log.Info("fitness cert: certificate PDF bytes ready", slog.Int("certPDFBytes", len(certPDF)))

	log.Info("fitness cert: parsing report blob URL", slog.String("reportURL", lead.ReportURL))
	container, blobName, err := storage.ParseAzureBlobContainerAndBlob(lead.ReportURL)
	if err != nil {
		return fmt.Errorf("parse report url: %w", err)
	}
	log.Info("fitness cert: blob target",
		slog.String("container", container),
		slog.String("blob", blobName),
	)

	log.Info("fitness cert: downloading existing diagnostic report PDF")
	reportPDF, err := d.Blob.DownloadBlob(ctx, container, blobName)
	if err != nil {
		return fmt.Errorf("download report: %w", err)
	}
	log.Info("fitness cert: downloaded report PDF", slog.Int("reportPDFBytes", len(reportPDF)))

	log.Info("fitness cert: merging PDFs (certificate first, then report)",
		slog.Int("certPDFBytes", len(certPDF)),
		slog.Int("reportPDFBytes", len(reportPDF)),
	)
	merged, err := fitnesscert.MergeCertificateFirst(certPDF, reportPDF)
	if err != nil {
		return fmt.Errorf("merge certificate and report pdf: %w", err)
	}
	log.Info("fitness cert: merge complete", slog.Int("mergedPDFBytes", len(merged)))

	log.Info("fitness cert: uploading merged PDF to blob")
	if err := d.Blob.UploadDiagnosticReportPDFBytes(ctx, container, blobName, merged); err != nil {
		return fmt.Errorf("upload merged pdf: %w", err)
	}
	log.Info("fitness cert: upload complete", slog.String("container", container), slog.String("blob", blobName))

	log.Info("fitness cert: updating lead status in database",
		slog.Int64("leadID", lead.LeadID),
		slog.Int("doneLeadStatusID", int(d.Config.DoneLeadStatusID)),
	)
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
	log.Info("fitness cert: success - merged PDF uploaded and lead marked done",
		slog.Int64("leadID", lead.LeadID),
		slog.Int("mergedPDFBytes", len(merged)),
	)
	return nil
}

func requiresFitnessCertificate(clientTypeID int8) bool {
	return clientTypeID == 1 || clientTypeID == 2
}
