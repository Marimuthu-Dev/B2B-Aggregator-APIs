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

const emailTypeStoreCreated = "StoreCreated"

func (s *storeService) queueStoreCreatedEmail(ctx context.Context, st *domain.Store) {
	if s == nil || s.emails == nil || st == nil {
		return
	}
	if !persistencemodels.IsMedLyfeSchema() {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	to := strings.TrimSpace(st.EmailID)
	if to == "" {
		slog.Warn("store created email skipped: EmailID is empty",
			slog.Int64("storeID", st.StoreID))
		return
	}

	portalURL := normalizePortalHomeURL(s.portalURL)
	if portalURL == "" {
		slog.Warn("store created email skipped: CLIENT_DOMAIN_URL is empty",
			slog.Int64("storeID", st.StoreID))
		return
	}
	if st.StoreID == 0 {
		slog.Warn("store created email skipped: StoreID is missing after insert",
			slog.String("storeName", strings.TrimSpace(st.StoreName)))
		return
	}

	resetKey, err := insertForgotPasswordKey(s.forgotRepo, st.StoreID, utils.UserTypeStore, clientCreatedForgotPasswordTTL)
	if err != nil {
		slog.Error("store created email: tbl_ForgotPassword insert failed",
			slog.Int64("storeID", st.StoreID),
			slog.Any("err", err))
		return
	}
	generateURL := buildClientResetPasswordURL(portalURL, resetKey)
	if generateURL == "" {
		slog.Error("store created email: reset password URL could not be built",
			slog.Int64("storeID", st.StoreID))
		return
	}

	schema := persistencemodels.Schema()
	body, err := emailtemplates.RenderStoreCreated(schema, emailtemplates.StoreCreatedData{
		StoreName:           strings.TrimSpace(st.StoreName),
		Username:            strings.TrimSpace(st.ContactNumber),
		GeneratePasswordURL: generateURL,
		PortalURL:           portalURL,
		LogoURL:             s.emailCfg.LogoURL,
		SupportPhone:        s.emailCfg.SupportPhone,
		SupportEmail:        s.emailCfg.SupportEmail,
		Year:                time.Now().Year(),
	})
	if err != nil {
		slog.Error("store created email: render failed",
			slog.Int64("storeID", st.StoreID),
			slog.Any("err", err))
		return
	}

	err = s.emails.Enqueue(ctx, domain.QueuedEmail{
		Subject:     "Welcome to MedLyfe Health",
		FromAddress: s.emailCfg.FromAddress,
		ToAddress:   to,
		CC:          joinEmailAddresses(s.emailCfg.CCAddress),
		BCC:         s.emailCfg.BCCAddress,
		BodyContent: body,
		EmailType:   emailTypeStoreCreated,
		CreatedBy:   st.CreatedBy,
	})
	if err != nil {
		slog.Error("store created email: insert into tbl_Emails failed",
			slog.Int64("storeID", st.StoreID),
			slog.Any("err", err))
		return
	}
	slog.Info("store created email queued",
		slog.Int64("storeID", st.StoreID),
		slog.String("to", to),
		slog.String("schema", schema),
		slog.String("emailType", emailTypeStoreCreated))
}
