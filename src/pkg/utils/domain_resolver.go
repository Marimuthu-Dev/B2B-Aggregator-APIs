package utils

import "fmt"

// UserType constants matching Node.js (1=employee, 2=client, 3=lab)
const (
	UserTypeEmployee = 1
	UserTypeClient   = 2
	UserTypeLab      = 3
)

// GetUserTypeFromDomain returns userType (1=employee, 2=client, 3=lab) from domain string.
// Accepts short names (employee, client, lab), numeric ("1","2","3"), or staging URLs.
// Returns 0 for invalid/unknown domain (matches Node.js getUserTypeFromDomain).
func GetUserTypeFromDomain(domain string) int {
	var userType int
	switch domain {
	case "1", "employee", "um-staging-ops-web.azurewebsites.net":
		userType = UserTypeEmployee
	case "2", "client", "um-staging-client-web.azurewebsites.net":
		userType = UserTypeClient
	case "3", "lab", "um-staging-lab-web.azurewebsites.net":
		userType = UserTypeLab
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
	default:
		return ""
	}
}
