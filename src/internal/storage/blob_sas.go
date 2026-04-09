package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

// sasStartSkew is subtracted from "now" so SAS is valid if local/Azure clocks differ slightly.
const sasStartSkew = 5 * time.Minute

var (
	// Azure container names: 3–63 chars, lowercase letters, digits, hyphen; start/end alnum.
	reAzureContainerName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])$`)
	reNumericClientID    = regexp.MustCompile(`^[0-9]+$`)
	reClientMoUBlobName  = regexp.MustCompile(`^client-[0-9]+-mou\.pdf$`)
)

// validateAzureContainerName checks AZURE_LEGAL_CONTAINER_NAME-style values.
func validateAzureContainerName(name string) error {
	n := strings.TrimSpace(name)
	if len(n) < 3 || len(n) > 63 {
		return fmt.Errorf("container name must be 3–63 characters")
	}
	if strings.Contains(n, "--") || !reAzureContainerName.MatchString(n) {
		return fmt.Errorf("container name must be lowercase alphanumeric with single hyphens (Azure naming rules)")
	}
	return nil
}

// validateBlobNameForSAS rejects empty names and path traversal; allows virtual directories (slashes).
func validateBlobNameForSAS(blobName string) error {
	b := strings.TrimSpace(blobName)
	if b == "" {
		return errors.New("blob name is required")
	}
	if strings.HasPrefix(b, "/") || strings.Contains(b, "\\") || strings.Contains(b, "..") {
		return errors.New("blob name must not use path traversal or leading slashes")
	}
	return nil
}

// validateClientIDForMoUBlob ensures clientId is numeric for client-<id>-mou.pdf.
func validateClientIDForMoUBlob(clientID string) error {
	id := strings.TrimSpace(clientID)
	if id == "" {
		return errors.New("clientId is required")
	}
	if !reNumericClientID.MatchString(id) {
		return errors.New("clientId must contain digits only")
	}
	return nil
}

// ClientMoUBlobObjectName returns the blob path client-<clientId>-mou.pdf (clientId trimmed, no validation).
func ClientMoUBlobObjectName(clientID string) string {
	return fmt.Sprintf("client-%s-mou.pdf", strings.TrimSpace(clientID))
}

// buildBlobReadOnlyHTTPSASURLWithTimes creates a read-only blob SAS URL over HTTPS using a SharedKeyCredential.
// Signing uses github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas only — never a connection string.
// The returned query string uses sas.QueryParameters.Encode() (no JSON/HTML escaping).
func buildBlobReadOnlyHTTPSASURLWithTimes(
	cred *azblob.SharedKeyCredential,
	endpointSuffix, containerName, blobName string,
	startUTC, expiryUTC time.Time,
	log *slog.Logger,
) (fullURL string, expiresAt time.Time, err error) {
	if cred == nil {
		return "", time.Time{}, errors.New("shared key credential is required for SAS signing")
	}
	account := strings.TrimSpace(cred.AccountName())
	if account == "" {
		return "", time.Time{}, errors.New("credential has empty account name")
	}
	suffix := strings.TrimSpace(endpointSuffix)
	if suffix == "" {
		suffix = "core.windows.net"
	}
	containerName = strings.TrimSpace(containerName)
	blobName = strings.TrimSpace(blobName)
	if err := validateAzureContainerName(containerName); err != nil {
		return "", time.Time{}, fmt.Errorf("container: %w", err)
	}
	if err := validateBlobNameForSAS(blobName); err != nil {
		return "", time.Time{}, fmt.Errorf("blob: %w", err)
	}
	if expiryUTC.IsZero() || !expiryUTC.After(startUTC) {
		return "", time.Time{}, errors.New("expiry must be after SAS start time")
	}

	// Use the same blob URL + signing path as the official SDK tests: blob.Client.GetSASURL.
	// That keeps container/blob path parsing and the HMAC string-to-sign aligned (avoids subtle 403s).
	account = strings.ToLower(strings.TrimSpace(account))
	serviceRoot := fmt.Sprintf("https://%s.blob.%s", account, suffix)
	blobPath := runtime.JoinPaths(serviceRoot, containerName, blobName)

	bc, err := blob.NewClientWithSharedKeyCredential(blobPath, cred, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("blob client for SAS: %w", err)
	}

	start := startUTC.UTC()
	expiry := expiryUTC.UTC()
	fullURL, err = bc.GetSASURL(sas.BlobPermissions{Read: true}, expiry, &blob.GetSASURLOptions{StartTime: &start})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign blob SAS (GetSASURL): %w", err)
	}

	if log != nil {
		log.Info("issued blob read SAS",
			slog.String("account", account),
			slog.String("container", containerName),
			slog.String("blob", blobName),
			slog.Time("sas_start_utc", startUTC.UTC()),
			slog.Time("sas_expiry_utc", expiryUTC.UTC()),
		)
		// Full URL contains the sig; log at Debug only to reduce secret leakage in production logs.
		log.Debug("blob SAS URL", slog.String("url", fullURL))
	}

	return fullURL, expiryUTC.UTC(), nil
}

// buildBlobReadOnlySASURLRelativeToNow applies a -5 minute start skew and an expiry duration from "now" (UTC).
func buildBlobReadOnlySASURLRelativeToNow(
	cred *azblob.SharedKeyCredential,
	endpointSuffix, containerName, blobName string,
	validFor time.Duration,
	log *slog.Logger,
) (fullURL string, expiresAt time.Time, err error) {
	if validFor <= 0 {
		validFor = 15 * time.Minute
	}
	now := time.Now().UTC()
	start := now.Add(-sasStartSkew)
	expiresAt = now.Add(validFor)
	return buildBlobReadOnlyHTTPSASURLWithTimes(cred, endpointSuffix, containerName, blobName, start, expiresAt, log)
}

// GenerateClientMoUBlobReadSASURL builds a read-only HTTPS SAS URL for blob client-<clientId>-mou.pdf.
// accountName and accountKey must come from AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_KEY (not from a connection string).
// containerName should match AZURE_LEGAL_CONTAINER_NAME.
//
// Example (standalone tool or test):
//
//	sasURL, err := storage.GenerateClientMoUBlobReadSASURL(
//	    os.Getenv("AZURE_STORAGE_ACCOUNT"),
//	    os.Getenv("AZURE_STORAGE_KEY"),
//	    "core.windows.net",
//	    os.Getenv("AZURE_LEGAL_CONTAINER_NAME"),
//	    "12345",
//	    slog.Default(),
//	)
func GenerateClientMoUBlobReadSASURL(accountName, accountKey, endpointSuffix, containerName, clientID string, log *slog.Logger) (string, error) {
	accountName = strings.ToLower(strings.TrimSpace(accountName))
	accountKey = strings.TrimSpace(accountKey)
	if accountName == "" || accountKey == "" {
		return "", errors.New("accountName and accountKey are required")
	}
	if err := validateClientIDForMoUBlob(clientID); err != nil {
		return "", err
	}
	if err := validateAzureContainerName(containerName); err != nil {
		return "", err
	}
	blobName := ClientMoUBlobObjectName(clientID)
	if !reClientMoUBlobName.MatchString(blobName) {
		return "", fmt.Errorf("unexpected blob name shape: %q", blobName)
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return "", fmt.Errorf("shared key credential: %w", err)
	}
	now := time.Now().UTC()
	start := now.Add(-sasStartSkew)
	expiry := now.Add(15 * time.Minute)
	u, _, err := buildBlobReadOnlyHTTPSASURLWithTimes(cred, endpointSuffix, containerName, blobName, start, expiry, log)
	return u, err
}

// GenerateBlobSAS returns a read-only HTTPS SAS URL for client-<clientId>-mou.pdf using the service’s
// storage account, key, container, and endpoint suffix. clientId must be numeric digits only.
func (s *AzureMoUBlobService) GenerateBlobSAS(clientID string) (string, error) {
	if s == nil || s.sharedKey == nil {
		return "", errors.New("azure blob service or credential not initialized")
	}
	if err := validateClientIDForMoUBlob(clientID); err != nil {
		return "", err
	}
	blobName := ClientMoUBlobObjectName(clientID)
	now := time.Now().UTC()
	start := now.Add(-sasStartSkew)
	expiry := now.Add(15 * time.Minute)
	u, _, err := buildBlobReadOnlyHTTPSASURLWithTimes(
		s.sharedKey, s.endpointSuffix, s.container, blobName, start, expiry, s.log,
	)
	return u, err
}
