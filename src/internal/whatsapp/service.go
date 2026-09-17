package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
)

const (
	defaultEndpoint       = "https://cpaaslink.com/api/whatsapp/public/apikey"
	defaultMMLiteEndpoint = "https://cpaaslink.com/api/whatsapp/public/mm-lite"
	apikeySuffix          = "/apikey"
	mmLiteSuffix          = "/mm-lite"
)

type Config struct {
	APIKey              string
	APIEndpoint         string
	DefaultTemplateName string
	CampaignName        string
	HTTPClient          *http.Client
	SendTimeout         time.Duration
}

type whatsappRequest struct {
	Number       []string `json:"number"`
	TemplateName string   `json:"template_name"`
	CampaignName string   `json:"campaign_name"`
	Variables    []string `json:"variables,omitempty"`
	Time         string   `json:"time,omitempty"`
}

type whatsappResponseData struct {
	MessageID   int64  `json:"messageid"`
	TotalNumber string `json:"totnumber"`
}

type whatsappResponse struct {
	Status      string               `json:"status"`
	Code        string               `json:"code"`
	Description string               `json:"description"`
	Data        whatsappResponseData `json:"data"`
}

type Service struct {
	cfg    Config
	client *http.Client
}

func NewService(cfg Config) (*Service, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("WhatsApp API key is required")
	}

	endpoint := strings.TrimSpace(cfg.APIEndpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	templateName := strings.TrimSpace(cfg.DefaultTemplateName)

	campaignName := strings.TrimSpace(cfg.CampaignName)
	if campaignName == "" {
		campaignName = "default_campaign"
	}

	return &Service{
		cfg: Config{
			APIKey:              cfg.APIKey,
			APIEndpoint:         endpoint,
			DefaultTemplateName: templateName,
			CampaignName:        campaignName,
			HTTPClient:          httpClient,
			SendTimeout:         cfg.SendTimeout,
		},
		client: httpClient,
	}, nil
}

func (s *Service) resolveEndpoint(templateType string) string {
	base := s.cfg.APIEndpoint
	if templateType == domain.WhatsAppTemplateTypeMMLite {
		if strings.HasSuffix(base, apikeySuffix) {
			return strings.TrimSuffix(base, apikeySuffix) + mmLiteSuffix
		}
		if strings.Contains(base, mmLiteSuffix) {
			return base
		}
		return strings.TrimRight(base, "/") + mmLiteSuffix
	}
	if strings.HasSuffix(base, mmLiteSuffix) {
		return strings.TrimSuffix(base, mmLiteSuffix) + apikeySuffix
	}
	return base
}

func (s *Service) resolveTemplateName(w domain.OutboxWhatsApp) string {
	templateName := strings.TrimSpace(w.TemplateName)
	if templateName == "" {
		templateName = s.cfg.DefaultTemplateName
	}
	return templateName
}

// SendMessage sends one WhatsApp message. The endpoint is selected per message
// based on OutboxWhatsApp.TemplateType (mm_lite routes to /mm-lite, anything else to /apikey).
func (s *Service) SendMessage(ctx context.Context, w domain.OutboxWhatsApp) error {
	fromMobile := strings.TrimSpace(w.FromMobile)
	toMobile := strings.TrimSpace(w.ToMobile)
	whatsappText := strings.TrimSpace(w.WhatsAppText)

	if fromMobile == "" {
		return errors.New("FromMobile is empty")
	}
	if toMobile == "" {
		return errors.New("ToMobile is empty")
	}
	if whatsappText == "" {
		return errors.New("WhatsAppText is empty")
	}

	templateName := s.resolveTemplateName(w)
	templateType := strings.ToLower(strings.TrimSpace(w.TemplateType))
	endpoint := s.resolveEndpoint(templateType)

	reqBody := whatsappRequest{
		Number:       []string{toMobile},
		TemplateName: templateName,
		CampaignName: s.cfg.CampaignName,
		Variables:    []string{whatsappText},
	}

	return s.postSend(ctx, reqBody, endpoint)
}

// SendBatch sends multiple WhatsApp messages in a single API call.
// All messages in a batch must share the same template name AND same template type (endpoint).
func (s *Service) SendBatch(ctx context.Context, messages []domain.OutboxWhatsApp) error {
	if len(messages) == 0 {
		return errors.New("no messages to send")
	}

	templateName := s.resolveTemplateName(messages[0])
	templateType := strings.ToLower(strings.TrimSpace(messages[0].TemplateType))

	for _, msg := range messages {
		msgTemplateName := strings.TrimSpace(msg.TemplateName)
		if msgTemplateName != "" && msgTemplateName != templateName {
			return fmt.Errorf("all messages in batch must use the same template name")
		}
		msgTemplateType := strings.ToLower(strings.TrimSpace(msg.TemplateType))
		if msgTemplateType != "" && msgTemplateType != templateType {
			return fmt.Errorf("all messages in batch must use the same template type (endpoint)")
		}
	}

	endpoint := s.resolveEndpoint(templateType)

	var numbers []string
	var variables []string

	for _, msg := range messages {
		toMobile := strings.TrimSpace(msg.ToMobile)
		whatsappText := strings.TrimSpace(msg.WhatsAppText)

		if toMobile == "" {
			return fmt.Errorf("ToMobile is empty for message %d", msg.WhatsAppID)
		}
		if whatsappText == "" {
			return fmt.Errorf("WhatsAppText is empty for message %d", msg.WhatsAppID)
		}

		numbers = append(numbers, toMobile)
		variables = append(variables, whatsappText)
	}

	reqBody := whatsappRequest{
		Number:       numbers,
		TemplateName: templateName,
		CampaignName: s.cfg.CampaignName,
		Variables:    variables,
	}

	return s.postSend(ctx, reqBody, endpoint)
}

func (s *Service) postSend(ctx context.Context, payload whatsappRequest, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("parse endpoint URL: %w", err)
	}

	q := u.Query()
	q.Set("apikey", s.cfg.APIKey)
	u.RawQuery = q.Encode()

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal whatsapp json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http send: %w", err)
	}
	defer resp.Body.Close()

	var response whatsappResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if response.Status == "Success" && response.Code == "011" {
		return nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("whatsapp API rate limited: code=%s description=%s", response.Code, response.Description)
	}

	return fmt.Errorf("whatsapp API failed: status=%s code=%s description=%s", response.Status, response.Code, response.Description)
}
