package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/emailtemplates"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

const emailTypeLabCreated = "LabCreated"

func labCreatedSubject(schema string) string {
	if strings.EqualFold(strings.TrimSpace(schema), "MedLyfe") {
		return "Welcome to MedLyfe Portal"
	}
	return "Welcome to the UrMediconnect Portal – Booking & Reports"
}

func (s *labService) queueLabCreatedEmail(ctx context.Context, l *domain.Lab) {
	if s == nil || s.emails == nil || l == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	to := derefString(l.ContactPerson1EmailID)
	if to == "" {
		slog.Warn("lab created email skipped: ContactPerson1EmailID is empty",
			slog.Int64("labID", l.LabID))
		return
	}

	portalURL := normalizePortalHomeURL(s.portalURL)
	if portalURL == "" {
		slog.Warn("lab created email skipped: LAB_DOMAIN_URL is empty",
			slog.Int64("labID", l.LabID))
		return
	}
	if l.LabID == 0 {
		slog.Warn("lab created email skipped: LabID is missing after insert",
			slog.String("labName", strings.TrimSpace(l.LabName)))
		return
	}

	resetKey, err := insertForgotPasswordKey(s.forgotRepo, l.LabID, utils.UserTypeLab, clientCreatedForgotPasswordTTL)
	if err != nil {
		slog.Error("lab created email: tbl_ForgotPassword insert failed",
			slog.Int64("labID", l.LabID),
			slog.Any("err", err))
		return
	}
	generateURL := buildClientResetPasswordURL(portalURL, resetKey)
	if generateURL == "" {
		slog.Error("lab created email: reset password URL could not be built",
			slog.Int64("labID", l.LabID))
		return
	}

	schema := persistencemodels.Schema()
	body, err := emailtemplates.RenderLabCreated(schema, emailtemplates.LabCreatedData{
		LabName:             strings.TrimSpace(l.LabName),
		Username:            derefString(l.ContactPerson1Number),
		GeneratePasswordURL: generateURL,
		PortalURL:           portalURL,
		LogoURL:             s.emailCfg.LogoURL,
		SupportPhone:        s.emailCfg.SupportPhone,
		SupportEmail:        s.emailCfg.SupportEmail,
		Year:                time.Now().Year(),
	})
	if err != nil {
		slog.Error("lab created email: render failed",
			slog.Int64("labID", l.LabID),
			slog.Any("err", err))
		return
	}

	cc := joinEmailAddresses(s.emailCfg.CCAddress, derefString(l.ContactPerson1EmailID1))
	createdBy := int64(0)
	if l.CreatedBy != nil {
		createdBy = *l.CreatedBy
	}
	err = s.emails.Enqueue(ctx, domain.QueuedEmail{
		Subject:     labCreatedSubject(schema),
		FromAddress: s.emailCfg.FromAddress,
		ToAddress:   to,
		CC:          cc,
		BCC:         s.emailCfg.BCCAddress,
		BodyContent: body,
		EmailType:   emailTypeLabCreated,
		CreatedBy:   createdBy,
	})
	if err != nil {
		slog.Error("lab created email: insert into tbl_Emails failed",
			slog.Int64("labID", l.LabID),
			slog.Any("err", err))
		return
	}
	slog.Info("lab created email queued",
		slog.Int64("labID", l.LabID),
		slog.String("to", to),
		slog.String("schema", schema),
		slog.String("emailType", emailTypeLabCreated))
}
