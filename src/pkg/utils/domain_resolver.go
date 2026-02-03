package utils

// UserType constants matching Node.js (1=employee, 2=client, 3=lab)
const (
	UserTypeEmployee = 1
	UserTypeClient   = 2
	UserTypeLab      = 3
)

// GetUserTypeFromDomain returns userType (1=employee, 2=client, 3=lab) from domain string
func GetUserTypeFromDomain(domain string) int {
	switch domain {
	case "employee":
		return UserTypeEmployee
	case "client":
		return UserTypeClient
	case "lab":
		return UserTypeLab
	default:
		return 0
	}
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
