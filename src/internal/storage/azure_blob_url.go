package storage

import (
	"errors"
	"net/url"
	"strings"
)

// ParseAzureBlobContainerAndBlob extracts container and blob object path from a typical Azure Blob HTTPS URL.
// The blob name may contain '/' (virtual directories).
func ParseAzureBlobContainerAndBlob(rawURL string) (container, blobName string, err error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "https" {
		return "", "", errors.New("invalid blob url: expected https")
	}
	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return "", "", errors.New("invalid blob url: empty path")
	}
	idx := strings.IndexByte(path, '/')
	if idx <= 0 || idx >= len(path)-1 {
		return "", "", errors.New("invalid blob url: missing blob path")
	}
	container = path[:idx]
	blobName = path[idx+1:]
	blobName, err = url.PathUnescape(blobName)
	if err != nil {
		return "", "", err
	}
	return container, blobName, nil
}
