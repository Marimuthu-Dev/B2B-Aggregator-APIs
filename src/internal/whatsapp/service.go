package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	MessageID   any    `json:"messageid"`
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
	toMobile := NormalizePhoneNumber(strings.TrimSpace(w.ToMobile))
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

	var variables []string
	if strings.Contains(whatsappText, "|") {
		variables = strings.Split(whatsappText, "|")
	} else {
		variables = []string{whatsappText}
	}

	reqBody := whatsappRequest{
		Number:       []string{toMobile},
		TemplateName: templateName,
		CampaignName: s.cfg.CampaignName,
		Variables:    variables,
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
		toMobile := NormalizePhoneNumber(strings.TrimSpace(msg.ToMobile))
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

	// For batch, CPAAS expects each parameter in the variables array.
	// But it could depend on the API. Let's assume it accepts an array of strings per number,
	// wait, since `Variables` is `[]string`, for batch it usually means comma-separated params per number.
	// We'll join by comma if it contains pipe, else just use the text.
	var finalVariables []string
	for _, v := range variables {
		if strings.Contains(v, "|") {
			finalVariables = append(finalVariables, strings.ReplaceAll(v, "|", ","))
		} else {
			finalVariables = append(finalVariables, v)
		}
	}

	reqBody := whatsappRequest{
		Number:       numbers,
		TemplateName: templateName,
		CampaignName: s.cfg.CampaignName,
		Variables:    finalVariables,
	}

	return s.postSend(ctx, reqBody, endpoint)
}

type APIError struct {
	Status      string
	Code        string
	Description string
	Curl        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("whatsapp API failed: status=%s code=%s description=%s | curl: %s", e.Status, e.Code, e.Description, e.Curl)
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

	fullURL := u.String()
	
	// Temporarily unmasking API key in curl for debugging delivery issues as requested by the user
	curlCmd := fmt.Sprintf("curl -X POST '%s' -H 'Content-Type: application/json' -d '%s'", fullURL, strings.ReplaceAll(string(body), "'", "'\\''"))

	slog.Info("whatsapp 3rd party api call curl",
		slog.String("templateName", payload.TemplateName),
		slog.String("curl", curlCmd),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http send: %w | curl: %s", err, curlCmd)
	}
	defer resp.Body.Close()

	// Read and log raw response body for deep debugging
	respBodyBytes := new(bytes.Buffer)
	_, _ = respBodyBytes.ReadFrom(resp.Body)
	rawResponse := respBodyBytes.String()

	slog.Info("whatsapp 3rd party api raw response",
		slog.String("templateName", payload.TemplateName),
		slog.String("responseBody", rawResponse),
	)

	var response whatsappResponse
	if err := json.NewDecoder(strings.NewReader(rawResponse)).Decode(&response); err != nil {
		return fmt.Errorf("decode response: %w | rawResponse: %s | curl: %s", err, rawResponse, curlCmd)
	}

	if response.Status == "Success" && response.Code == "011" {
		return nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("whatsapp API rate limited: code=%s description=%s | curl: %s", response.Code, response.Description, curlCmd)
	}

	return &APIError{
		Status:      response.Status,
		Code:        response.Code,
		Description: response.Description,
		Curl:        curlCmd,
	}
}
