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

// WhatsApp API Documentation: https://cpaaslink.com/developer
// Based on the provided documentation screenshots for cpaaslink.com WhatsApp API

const (
	defaultEndpoint = "https://cpaaslink.com/api/whatsapp/public/apikey"
)

// Config holds connection and HTTP tuning for WhatsApp API.
type Config struct {
	APIKey        string
	APIEndpoint   string
	TemplateName  string
	CampaignName  string
	HTTPClient    *http.Client
	SendTimeout   time.Duration
}

// WhatsApp API Request/Response structures
type whatsappRequest struct {
	Number       []string `json:"number"`
	TemplateName string   `json:"template_name"`
	CampaignName string   `json:"campaign_name"`
	Variables    []string `json:"variables,omitempty"`
	Time         string   `json:"time,omitempty"` // Optional: schedule time for future delivery
}

type whatsappResponseData struct {
	MessageID int64  `json:"messageid"`
	TotalNumber string `json:"totnumber"`
}

type whatsappResponse struct {
	Status      string                `json:"status"`
	Code        string                `json:"code"`
	Description string                `json:"description"`
	Data        whatsappResponseData  `json:"data"`
}

// Service sends WhatsApp messages via cpaaslink.com API.
type Service struct {
	cfg    Config
	client *http.Client
}

// NewService builds a WhatsApp sender from config.
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
	
	templateName := strings.TrimSpace(cfg.TemplateName)
	if templateName == "" {
		return nil, errors.New("WhatsApp template name is required")
	}
	
	campaignName := strings.TrimSpace(cfg.CampaignName)
	if campaignName == "" {
		campaignName = "default_campaign"
	}
	
	return &Service{
		cfg: Config{
			APIKey:       cfg.APIKey,
			APIEndpoint:  endpoint,
			TemplateName: templateName,
			CampaignName: campaignName,
			HTTPClient:   httpClient,
			SendTimeout:  cfg.SendTimeout,
		},
		client: httpClient,
	}, nil
}

// SendMessage sends one WhatsApp message using the provided data.
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

	// Build the API request
	reqBody := whatsappRequest{
		Number:       []string{toMobile},
		TemplateName: s.cfg.TemplateName,
		CampaignName: s.cfg.CampaignName,
		Variables:    []string{whatsappText}, // Using WhatsAppText as the variable
	}

	return s.postSend(ctx, reqBody)
}

// SendBatch sends multiple WhatsApp messages in a single API call.
func (s *Service) SendBatch(ctx context.Context, messages []domain.OutboxWhatsApp) error {
	if len(messages) == 0 {
		return errors.New("no messages to send")
	}

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
		TemplateName: s.cfg.TemplateName,
		CampaignName: s.cfg.CampaignName,
		Variables:    variables,
	}

	return s.postSend(ctx, reqBody)
}

func (s *Service) postSend(ctx context.Context, payload whatsappRequest) error {
	// Build URL with API key as query parameter
	u, err := url.Parse(s.cfg.APIEndpoint)
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

	// Check for successful response
	if response.Status == "Success" && response.Code == "011" {
		return nil
	}

	// Handle rate limiting or other errors
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("whatsapp API rate limited: code=%s description=%s", response.Code, response.Description)
	}

	return fmt.Errorf("whatsapp API failed: status=%s code=%s description=%s", response.Status, response.Code, response.Description)
}
