package acsemail

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
)

// ACS Email Data Plane: https://learn.microsoft.com/en-us/rest/api/communication/dataplane/email/send
// The Go module github.com/Azure/azure-sdk-for-go/sdk/communication/azemail is not yet published
// on the Go module proxy; this client uses the same REST contract and HMAC authentication.
const (
	defaultAPIVersion = "2023-03-31"
	emailSendPath     = "/emails:send"
)

// Config holds connection and HTTP tuning for Azure Communication Services Email.
type Config struct {
	ConnectionString string
	APIVersion       string
	HTTPClient       *http.Client
}

type acsConn struct {
	endpointURL *url.URL
	accessKey   []byte
}

func parseConnectionString(cs string) (*acsConn, error) {
	cs = strings.TrimSpace(cs)
	if cs == "" {
		return nil, errors.New("ACS connection string is empty")
	}
	var endpointStr, accessKeyStr string
	for _, part := range strings.Split(cs, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "endpoint":
			endpointStr = val
		case "accesskey":
			accessKeyStr = val
		}
	}
	if endpointStr == "" || accessKeyStr == "" {
		return nil, errors.New("ACS connection string must include endpoint and accesskey")
	}
	u, err := url.Parse(endpointStr)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, errors.New("invalid ACS endpoint URL")
	}
	rawKey, err := base64.StdEncoding.DecodeString(accessKeyStr)
	if err != nil {
		return nil, fmt.Errorf("decode access key: %w", err)
	}
	return &acsConn{endpointURL: u, accessKey: rawKey}, nil
}

type mailAddress struct {
	Address     string `json:"address"`
	DisplayName string `json:"displayName,omitempty"`
}

type mailContent struct {
	Subject   string `json:"subject"`
	PlainText string `json:"plainText,omitempty"`
	HTML      string `json:"html,omitempty"`
}

type mailRecipients struct {
	To  []mailAddress `json:"to"`
	CC  []mailAddress `json:"cc,omitempty"`
	BCC []mailAddress `json:"bcc,omitempty"`
}

type sendEmailRequest struct {
	SenderAddress string         `json:"senderAddress"`
	Content       mailContent    `json:"content"`
	Recipients    mailRecipients `json:"recipients"`
}

// Service sends email via ACS REST API.
type Service struct {
	cfg    Config
	client *http.Client
	conn   *acsConn
	ver    string
}

// NewService builds a sender from config.
func NewService(cfg Config) (*Service, error) {
	conn, err := parseConnectionString(cfg.ConnectionString)
	if err != nil {
		return nil, err
	}
	ver := strings.TrimSpace(cfg.APIVersion)
	if ver == "" {
		ver = defaultAPIVersion
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Service{
		cfg:    cfg,
		client: httpClient,
		conn:   conn,
		ver:    ver,
	}, nil
}

// SendHTML sends one message using HTML body (BodyContent from DB).
func (s *Service) SendHTML(ctx context.Context, e domain.OutboxEmail) error {
	from := strings.TrimSpace(e.FromAddress)
	if from == "" {
		return errors.New("FromAddress is empty")
	}
	to := parseAddressList(e.ToAddress)
	if len(to) == 0 {
		return errors.New("ToAddress is empty")
	}

	reqBody := sendEmailRequest{
		SenderAddress: from,
		Content: mailContent{
			Subject: strings.TrimSpace(e.Subject),
			HTML:    e.BodyContent,
		},
		Recipients: mailRecipients{
			To:  to,
			CC:  parseAddressList(e.CC),
			BCC: parseAddressList(e.BCC),
		},
	}

	return s.postSend(ctx, reqBody)
}

func (s *Service) postSend(ctx context.Context, payload sendEmailRequest) error {
	u := *s.conn.endpointURL
	u.Path = strings.TrimRight(u.Path, "/") + emailSendPath
	q := u.Query()
	q.Set("api-version", s.ver)
	u.RawQuery = q.Encode()

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if err := signACSRequest(req, body, s.conn.accessKey); err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return nil
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return fmt.Errorf("acs email failed: status=%d, read body: %w", resp.StatusCode, err)
	}
	return fmt.Errorf("acs email failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(buf.String()))
}

func signACSRequest(req *http.Request, body []byte, accessKey []byte) error {
	u := req.URL
	pathAndQuery := u.Path
	if u.RawQuery != "" {
		pathAndQuery += "?" + u.RawQuery
	}

	timestamp := strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT")

	hash := sha256.Sum256(body)
	hashB64 := base64.StdEncoding.EncodeToString(hash[:])

	host := u.Host
	if host == "" {
		return errors.New("request URL missing host")
	}

	stringToSign := fmt.Sprintf(
		"%s\n%s\n%s;%s;%s",
		req.Method,
		pathAndQuery,
		timestamp,
		host,
		hashB64,
	)

	hm := hmac.New(sha256.New, accessKey)
	if _, err := hm.Write([]byte(stringToSign)); err != nil {
		return fmt.Errorf("hmac write: %w", err)
	}
	signature := base64.StdEncoding.EncodeToString(hm.Sum(nil))
	authorization := fmt.Sprintf("HMAC-SHA256 SignedHeaders=x-ms-date;host;x-ms-content-sha256&Signature=%s", signature)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ms-date", timestamp)
	req.Header.Set("x-ms-content-sha256", hashB64)
	req.Header.Set("Authorization", authorization)
	return nil
}

func parseAddressList(raw string) []mailAddress {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == ','
	})
	var out []mailAddress
	for _, p := range parts {
		addr := strings.TrimSpace(p)
		if addr == "" {
			continue
		}
		out = append(out, mailAddress{Address: addr})
	}
	return out
}
