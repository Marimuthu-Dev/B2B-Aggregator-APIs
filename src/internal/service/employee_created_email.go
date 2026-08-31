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

const emailTypeEmployeeCreated = "EmployeeCreated"

func employeeCreatedSubject(schema string) string {
	if strings.EqualFold(strings.TrimSpace(schema), "MedLyfe") {
		return "Welcome to MedLyfe Health"
	}
	return "Welcome to the UrMediconnect Portal – Booking & Reports"
}

func (s *employeeService) queueEmployeeCreatedEmail(ctx context.Context, e *domain.Employee) {
	if s == nil || s.emails == nil || e == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	to := strings.TrimSpace(e.CompanyEmailID)
	if to == "" {
		slog.Warn("employee created email skipped: CompanyEmailID is empty",
			slog.Int64("employeeUID", e.UID))
		return
	}

	portalURL := normalizePortalHomeURL(s.portalURL)
	if portalURL == "" {
		slog.Warn("employee created email skipped: EMPLOYEE_DOMAIN_URL is empty",
			slog.Int64("employeeUID", e.UID))
		return
	}
	if e.UID == 0 {
		slog.Warn("employee created email skipped: UID is missing after insert",
			slog.String("fullName", strings.TrimSpace(e.FullName)))
		return
	}

	resetKey, err := insertForgotPasswordKey(s.forgotRepo, e.UID, utils.UserTypeEmployee, clientCreatedForgotPasswordTTL)
	if err != nil {
		slog.Error("employee created email: tbl_ForgotPassword insert failed",
			slog.Int64("employeeUID", e.UID),
			slog.Any("err", err))
		return
	}
	generateURL := buildClientResetPasswordURL(portalURL, resetKey)
	if generateURL == "" {
		slog.Error("employee created email: reset password URL could not be built",
			slog.Int64("employeeUID", e.UID))
		return
	}

	schema := persistencemodels.Schema()
	body, err := emailtemplates.RenderEmployeeCreated(schema, emailtemplates.EmployeeCreatedData{
		FullName:            strings.TrimSpace(e.FullName),
		Username:            strings.TrimSpace(e.MobileNumber),
		GeneratePasswordURL: generateURL,
		PortalURL:           portalURL,
		LogoURL:             s.emailCfg.LogoURL,
		SupportPhone:        s.emailCfg.SupportPhone,
		SupportEmail:        s.emailCfg.SupportEmail,
		Year:                time.Now().Year(),
	})
	if err != nil {
		slog.Error("employee created email: render failed",
			slog.Int64("employeeUID", e.UID),
			slog.Any("err", err))
		return
	}

	err = s.emails.Enqueue(ctx, domain.QueuedEmail{
		Subject:     employeeCreatedSubject(schema),
		FromAddress: s.emailCfg.FromAddress,
		ToAddress:   to,
		CC:          joinEmailAddresses(s.emailCfg.CCAddress),
		BCC:         s.emailCfg.BCCAddress,
		BodyContent: body,
		EmailType:   emailTypeEmployeeCreated,
		CreatedBy:   e.CreatedBy,
	})
	if err != nil {
		slog.Error("employee created email: insert into tbl_Emails failed",
			slog.Int64("employeeUID", e.UID),
			slog.Any("err", err))
		return
	}
	slog.Info("employee created email queued",
		slog.Int64("employeeUID", e.UID),
		slog.String("to", to),
		slog.String("schema", schema),
		slog.String("emailType", emailTypeEmployeeCreated))
}
