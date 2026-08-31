package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/emailtemplates"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

const emailTypeForgotPassword = "ForgotPassword"

func isMediAdminSchema() bool {
	return strings.EqualFold(strings.TrimSpace(persistencemodels.Schema()), "MediAdmin")
}

func shouldQueueForgotPasswordEmail() bool {
	return isMediAdminSchema() || persistencemodels.IsMedLyfeSchema()
}

func forgotPasswordSubject(schema string) string {
	if strings.EqualFold(strings.TrimSpace(schema), persistencemodels.MedLyfeSchema) {
		return "Reset Your MedLyfe Health Password"
	}
	return "Reset your UrMediconnect password"
}

func formatLinkExpiry(d time.Duration) string {
	if d <= 0 {
		d = forgotPasswordKeyTTL
	}
	mins := int(d.Minutes())
	if mins < 1 {
		secs := int(d.Seconds())
		if secs == 1 {
			return "1 second"
		}
		return fmt.Sprintf("%d seconds", secs)
	}
	hours := int(d.Hours())
	if mins%60 == 0 && hours >= 1 {
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if mins == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", mins)
}

func (s *loginService) queueForgotPasswordEmail(ctx context.Context, userType int, userData interface{}, resetKey string) {
	if s == nil || s.emails == nil || userData == nil {
		return
	}
	if !shouldQueueForgotPasswordEmail() {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	to, displayName, username := forgotPasswordRecipient(userType, userData)
	if to == "" {
		slog.Warn("forgot password email skipped: recipient email is empty",
			slog.Int("userType", userType))
		return
	}

	portalURL := normalizePortalHomeURL(s.portalURLForUserType(userType))
	if portalURL == "" {
		slog.Warn("forgot password email skipped: portal URL env is empty",
			slog.Int("userType", userType))
		return
	}
	generateURL := buildClientResetPasswordURL(portalURL, resetKey)
	if generateURL == "" {
		slog.Error("forgot password email: reset password URL could not be built",
			slog.Int("userType", userType))
		return
	}

	schema := persistencemodels.Schema()
	body, err := emailtemplates.RenderForgotPassword(schema, emailtemplates.ForgotPasswordData{
		DisplayName:         displayName,
		Username:            username,
		GeneratePasswordURL: generateURL,
		PortalURL:           portalURL,
		LogoURL:             s.emailCfg.LogoURL,
		SupportPhone:        s.emailCfg.SupportPhone,
		SupportEmail:        s.emailCfg.SupportEmail,
		Year:                time.Now().Year(),
		LinkExpiry:          formatLinkExpiry(forgotPasswordKeyTTL),
	})
	if err != nil {
		slog.Error("forgot password email: render failed", slog.Any("err", err))
		return
	}

	err = s.emails.Enqueue(ctx, domain.QueuedEmail{
		Subject:     forgotPasswordSubject(schema),
		FromAddress: s.emailCfg.FromAddress,
		ToAddress:   to,
		CC:          joinEmailAddresses(s.emailCfg.CCAddress),
		BCC:         s.emailCfg.BCCAddress,
		BodyContent: body,
		EmailType:   emailTypeForgotPassword,
		CreatedBy:   0,
	})
	if err != nil {
		slog.Error("forgot password email: insert into tbl_Emails failed", slog.Any("err", err))
		return
	}
	slog.Info("forgot password email queued",
		slog.String("to", to),
		slog.Int("userType", userType),
		slog.String("schema", schema),
		slog.String("emailType", emailTypeForgotPassword))
}

func (s *loginService) portalURLForUserType(userType int) string {
	if s == nil {
		return ""
	}
	switch userType {
	case utils.UserTypeEmployee:
		return s.employeePortalURL
	case utils.UserTypeLab:
		return s.labPortalURL
	case utils.UserTypeClient, utils.UserTypeStore:
		return s.clientPortalURL
	default:
		return ""
	}
}

func forgotPasswordRecipient(userType int, userData interface{}) (to, displayName, username string) {
	switch userType {
	case utils.UserTypeEmployee:
		if e, ok := userData.(*domain.Employee); ok && e != nil {
			return strings.TrimSpace(e.CompanyEmailID), strings.TrimSpace(e.FullName), strings.TrimSpace(e.MobileNumber)
		}
	case utils.UserTypeLab:
		if l, ok := userData.(*domain.Lab); ok && l != nil {
			return derefString(l.ContactPerson1EmailID), strings.TrimSpace(l.LabName), derefString(l.ContactPerson1Number)
		}
	case utils.UserTypeClient:
		if c, ok := userData.(*domain.Client); ok && c != nil {
			return strings.TrimSpace(c.ContactPerson1EmailID), strings.TrimSpace(c.ClientName), strings.TrimSpace(c.ContactPerson1Number)
		}
	case utils.UserTypeStore:
		if st, ok := userData.(*domain.Store); ok && st != nil {
			return strings.TrimSpace(st.EmailID), strings.TrimSpace(st.StoreName), strings.TrimSpace(st.ContactNumber)
		}
	}
	return "", "", ""
}
