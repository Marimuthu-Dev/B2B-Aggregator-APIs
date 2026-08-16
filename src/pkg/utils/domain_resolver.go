package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// UserType constants matching Node.js (1=employee, 2=client, 3=lab) plus store (4).
const (
	UserTypeEmployee = 1
	UserTypeClient   = 2
	UserTypeLab      = 3
	UserTypeStore    = 4
)

// UserTypeToLoginString is the tbl_Login.UserType value (numeric string).
func UserTypeToLoginString(userType int) string {
	switch userType {
	case UserTypeEmployee:
		return "1"
	case UserTypeClient:
		return "2"
	case UserTypeLab:
		return "3"
	case UserTypeStore:
		return "4"
	default:
		return ""
	}
}

// normalizeLoginDomain trims, lowercases, extracts host from full URLs, strips ports, and trailing dots.
func normalizeLoginDomain(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Hostname()
		}
	} else if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	return strings.TrimSuffix(s, ".")
}

// GetUserTypeFromDomain returns userType (1=employee, 2=client, 3=lab, 4=store) from domain string.
// Accepts short names (employee, client, lab, store), numeric ("1","2","3","4"), staging URLs, or prod hostnames.
// Full URLs (e.g. https://ops.urmediconnect.com) and host:port are normalized to the hostname.
// Returns 0 for invalid/unknown domain (matches Node.js getUserTypeFromDomain).
func GetUserTypeFromDomain(domain string) int {
	domain = normalizeLoginDomain(domain)
	var userType int
	switch domain {
	case "1", "employee", "um-staging-ops-web.azurewebsites.net", "um-prod-web.azurewebsites.net", "ops.urmediconnect.com", "ops.medlyfehealth.com":
		userType = UserTypeEmployee
	case "2", "client", "um-staging-client-web.azurewebsites.net", "client.urmediconnect.com", "client.medlyfehealth.com":
		userType = UserTypeClient
	case "3", "lab", "um-staging-lab-web.azurewebsites.net", "lab.urmediconnect.com", "lab.medlyfehealth.com":
		userType = UserTypeLab
	case "4", "store", "store.urmediconnect.com", "store.medlyfehealth.com":
		userType = UserTypeStore
	default:
		userType = 0 // invalid domain, same as Node.js
	}
	fmt.Printf("[LOGIN] Utils.GetUserTypeFromDomain: domain=%q -> userType=%d\n", domain, userType)
	return userType
}

// UserTypeToString returns the domain string for a userType
func UserTypeToString(userType int) string {
	switch userType {
	case UserTypeEmployee:
		return "employee"
	case UserTypeClient:
		return "client"
	case UserTypeLab:
		return "lab"
	case UserTypeStore:
		return "store"
	default:
		return ""
	}
}
