package service

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/emailtemplates"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

const (
	emailTypeClientCreated  = "ClientCreated"
	clientResetPasswordPath = "reset-password"
)

func clientCreatedSubject(schema string) string {
	if strings.EqualFold(strings.TrimSpace(schema), "MedLyfe") {
		return "Welcome to MedLyfe Health"
	}
	return "Welcome to the UrMediconnect Portal – Booking & Reports"
}

func (s *clientService) queueClientCreatedEmail(ctx context.Context, c *domain.Client) {
	if s == nil || s.emails == nil || c == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	to := strings.TrimSpace(c.ContactPerson1EmailID)
	if to == "" {
		slog.Warn("client created email skipped: ContactPerson1EmailID is empty",
			slog.Int64("clientID", c.ClientID))
		return
	}

	portalURL := normalizePortalHomeURL(s.portalURL)
	if portalURL == "" {
		slog.Warn("client created email skipped: CLIENT_DOMAIN_URL is empty",
			slog.Int64("clientID", c.ClientID))
		return
	}
	if c.ClientID == 0 {
		slog.Warn("client created email skipped: ClientID is missing after insert",
			slog.String("clientName", strings.TrimSpace(c.ClientName)))
		return
	}

	resetKey, err := insertForgotPasswordKey(s.forgotRepo, c.ClientID, utils.UserTypeClient, clientCreatedForgotPasswordTTL)
	if err != nil {
		slog.Error("client created email: tbl_ForgotPassword insert failed",
			slog.Int64("clientID", c.ClientID),
			slog.Any("err", err))
		return
	}
	generateURL := buildClientResetPasswordURL(portalURL, resetKey)
	if generateURL == "" {
		slog.Error("client created email: reset password URL could not be built",
			slog.Int64("clientID", c.ClientID))
		return
	}

	schema := persistencemodels.Schema()
	body, err := emailtemplates.RenderClientCreated(schema, emailtemplates.ClientCreatedData{
		ClientName:          strings.TrimSpace(c.ClientName),
		Username:            strings.TrimSpace(c.ContactPerson1Number),
		GeneratePasswordURL: generateURL,
		PortalURL:           portalURL,
		LogoURL:             s.emailCfg.LogoURL,
		SupportPhone:        s.emailCfg.SupportPhone,
		SupportEmail:        s.emailCfg.SupportEmail,
		Year:                time.Now().Year(),
	})
	if err != nil {
		slog.Error("client created email: render failed",
			slog.Int64("clientID", c.ClientID),
			slog.Any("err", err))
		return
	}

	cc := joinEmailAddresses(s.emailCfg.CCAddress, derefString(c.ContactPerson2EmailID))
	err = s.emails.Enqueue(ctx, domain.QueuedEmail{
		Subject:     clientCreatedSubject(schema),
		FromAddress: s.emailCfg.FromAddress,
		ToAddress:   to,
		CC:          cc,
		BCC:         s.emailCfg.BCCAddress,
		BodyContent: body,
		EmailType:   emailTypeClientCreated,
		CreatedBy:   c.CreatedBy,
	})
	if err != nil {
		slog.Error("client created email: insert into tbl_Emails failed",
			slog.Int64("clientID", c.ClientID),
			slog.Any("err", err))
		return
	}
	slog.Info("client created email queued",
		slog.Int64("clientID", c.ClientID),
		slog.String("to", to),
		slog.String("schema", schema),
		slog.String("emailType", emailTypeClientCreated))
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func joinEmailAddresses(values ...string) string {
	seen := make(map[string]struct{})
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return strings.Join(out, ";")
}

func normalizePortalHomeURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(u), "http://") && !strings.HasPrefix(strings.ToLower(u), "https://") {
		u = "https://" + u
	}
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u
}

func buildClientResetPasswordURL(portalHome, forgetPasswordKey string) string {
	key := strings.TrimSpace(forgetPasswordKey)
	base := strings.TrimSuffix(normalizePortalHomeURL(portalHome), "/")
	if base == "" || key == "" {
		return ""
	}
	u, err := url.Parse(base + "/" + clientResetPasswordPath)
	if err != nil {
		return ""
	}
	q := url.Values{}
	q.Set("token", key)
	u.RawQuery = q.Encode()
	return u.String()
}
