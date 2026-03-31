package storage

import (
	"errors"
	"strings"
)

// ParseAzureConnectionString extracts AccountName and AccountKey from a standard Azure Storage connection string.
// Values may contain '='; only the first '=' in each segment splits key and value.
func ParseAzureConnectionString(conn string) (accountName, accountKey string, err error) {
	conn = strings.TrimSpace(conn)
	if conn == "" {
		return "", "", errors.New("empty connection string")
	}
	var nameOK, keyOK bool
	for _, part := range strings.Split(conn, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "AccountName":
			accountName, nameOK = v, true
		case "AccountKey":
			accountKey, keyOK = v, true
		}
	}
	if !nameOK || !keyOK {
		return "", "", errors.New("connection string must include AccountName and AccountKey")
	}
	return accountName, accountKey, nil
}
