package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/config"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"gorm.io/gorm"
)

type LoginService interface {
	Login(req dto.LoginRequest) (*dto.LoginResponse, error)
	CreateForgotPasswordRecord(domainName, mobileNumber string) (int, error)
	GetLatestForgotPasswordKey(domainName, mobileNumber string) (*dto.ForgotPasswordKeyResponse, error)
	ForgotPasswordReset(forgetPasswordKey, newPassword string) (bool, error)
	ChangePassword(domainName, mobileNumber, oldPassword, newPassword string) (bool, error)
	GetProfile(domainName string, userID, mobileNumber *string) (interface{}, error)
}

type loginService struct {
	repo         repository.LoginRepository
	forgotRepo   repository.ForgotPasswordRepository
	clientRepo   repository.ClientRepository
	employeeRepo repository.EmployeeRepository
	labRepo      repository.LabRepository
	jwtSecret    string
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

func NewLoginService(
	repo repository.LoginRepository,
	forgotRepo repository.ForgotPasswordRepository,
	clientRepo repository.ClientRepository,
	employeeRepo repository.EmployeeRepository,
	labRepo repository.LabRepository,
	jwtCfg config.JWTConfig,
) LoginService {
	accessTTL, err := time.ParseDuration(jwtCfg.ExpiresIn)
	if err != nil {
		accessTTL = 24 * time.Hour
	}
	refreshTTL, err := time.ParseDuration(jwtCfg.RefreshExpiresIn)
	if err != nil {
		refreshTTL = 7 * 24 * time.Hour
	}
	return &loginService{
		repo:         repo,
		forgotRepo:   forgotRepo,
		clientRepo:   clientRepo,
		employeeRepo: employeeRepo,
		labRepo:      labRepo,
		jwtSecret:    jwtCfg.Secret,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
	}
}

// resolveUserByMobileNumber returns userId, userType (string for DB), and userData for client/employee/lab (matches Node.js)
func (s *loginService) resolveUserByMobileNumber(domainName, mobileNumber string) (userId int64, userType int, userData interface{}, err error) {
	fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: entry domain=%q mobileNumber=%q\n", domainName, mobileNumber)
	userType = utils.GetUserTypeFromDomain(domainName)
	fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: userType=%d\n", userType)
	if userType == 0 {
		fmt.Println("[LOGIN] Service.resolveUserByMobileNumber: invalid domain, returning error")
		return 0, 0, nil, apperrors.NewBadRequest("Invalid domain", nil)
	}
	switch userType {
	case utils.UserTypeClient:
		fmt.Println("[LOGIN] Service.resolveUserByMobileNumber: resolving client by contact number")
		client, err := s.clientRepo.FindByContactNumber(mobileNumber)
		if err != nil {
			fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: client not found: %v\n", err)
			return 0, 0, nil, apperrors.NewNotFound("User not found", err)
		}
		fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: client found ClientID=%d\n", client.ClientID)
		return client.ClientID, utils.UserTypeClient, client, nil
	case utils.UserTypeEmployee:
		fmt.Println("[LOGIN] Service.resolveUserByMobileNumber: resolving employee by mobile number")
		employee, err := s.employeeRepo.FindByMobileNumber(mobileNumber)
		if err != nil {
			fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: employee not found: %v\n", err)
			return 0, 0, nil, apperrors.NewNotFound("User not found", err)
		}
		fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: employee found UID=%d\n", employee.UID)
		return employee.UID, utils.UserTypeEmployee, employee, nil
	case utils.UserTypeLab:
		fmt.Println("[LOGIN] Service.resolveUserByMobileNumber: resolving lab by contact number")
		lab, err := s.labRepo.FindByContactNumber(mobileNumber)
		if err != nil {
			fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: lab not found: %v\n", err)
			return 0, 0, nil, apperrors.NewNotFound("User not found", err)
		}
		fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: lab found LabID=%d\n", lab.LabID)
		return lab.LabID, utils.UserTypeLab, lab, nil
	default:
		fmt.Printf("[LOGIN] Service.resolveUserByMobileNumber: unknown userType=%d, invalid domain\n", userType)
		return 0, 0, nil, apperrors.NewBadRequest("Invalid domain", nil)
	}
}

func (s *loginService) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	fmt.Println("[LOGIN] Service.Login: entry")
	var userID int64
	var userType int
	var userTypeStr string
	var userData interface{}

	if req.Domain != "" && req.MobileNumber != "" {
		fmt.Println("[LOGIN] Service.Login: using domain + mobileNumber path")
		uid, ut, ud, err := s.resolveUserByMobileNumber(req.Domain, req.MobileNumber)
		if err != nil {
			fmt.Printf("[LOGIN] Service.Login: resolveUserByMobileNumber failed: %v\n", err)
			return nil, err
		}
		userID, userType, userData = uid, ut, ud
		userTypeStr = strconv.Itoa(userType)
		fmt.Printf("[LOGIN] Service.Login: resolved userID=%d userTypeStr=%s\n", userID, userTypeStr)
	} else if req.UserID != 0 {
		fmt.Printf("[LOGIN] Service.Login: using legacy userId path userId=%d\n", req.UserID)
		login, err := s.repo.FindByUserID(req.UserID)
		if err != nil {
			fmt.Printf("[LOGIN] Service.Login: FindByUserID failed: %v\n", err)
			return nil, apperrors.NewUnauthorized("Invalid user ID or password", err)
		}
		userID = login.UserID
		userTypeStr = login.UserType
		userType = 0
		if login.UserType == "1" || login.UserType == "employee" {
			userType = utils.UserTypeEmployee
		} else if login.UserType == "2" || login.UserType == "client" {
			userType = utils.UserTypeClient
		} else if login.UserType == "3" || login.UserType == "lab" {
			userType = utils.UserTypeLab
		}
		userData = login
		fmt.Printf("[LOGIN] Service.Login: found login userID=%d userTypeStr=%s\n", userID, userTypeStr)
	} else {
		fmt.Println("[LOGIN] Service.Login: validation failed - need domain+mobileNumber or userId")
		return nil, apperrors.NewBadRequest("Either (domain + mobileNumber) or userId is required", nil)
	}

	fmt.Println("[LOGIN] Service.Login: encrypting password")
	encryptedPassword, err := utils.Encrypt(req.Password)
	if err != nil {
		fmt.Printf("[LOGIN] Service.Login: Encrypt failed: %v\n", err)
		return nil, apperrors.NewInternal("Error validating credentials", err)
	}

	fmt.Printf("[LOGIN] Service.Login: authenticating userID=%d userTypeStr=%s\n", userID, userTypeStr)
	ok, err := s.repo.Authenticate(userID, encryptedPassword, userTypeStr)
	if err != nil {
		fmt.Printf("[LOGIN] Service.Login: Authenticate error: %v\n", err)
		return nil, apperrors.NewInternal("Error validating credentials", err)
	}
	if !ok {
		fmt.Println("[LOGIN] Service.Login: authentication failed - password mismatch")
		return nil, apperrors.NewUnauthorized("Invalid credentials", errors.New("password mismatch"))
	}
	fmt.Println("[LOGIN] Service.Login: authentication OK")

	if userType == 0 {
		userType = utils.UserTypeClient
	}
	fmt.Println("[LOGIN] Service.Login: generating tokens")
	accessToken, refreshToken, err := utils.GenerateToken(userID, userType, s.jwtSecret, s.accessTTL, s.refreshTTL)
	if err != nil {
		fmt.Printf("[LOGIN] Service.Login: GenerateToken failed: %v\n", err)
		return nil, err
	}

	fmt.Println("[LOGIN] Service.Login: success")
	return &dto.LoginResponse{
		User:         userData,
		Token:        accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *loginService) CreateForgotPasswordRecord(domainName, mobileNumber string) (int, error) {
	userID, userType, _, err := s.resolveUserByMobileNumber(domainName, mobileNumber)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	expiry := now.Add(5 * time.Minute)

	payload := map[string]interface{}{
		"userId": userID, "userType": userType, "expiry": expiry.Format(time.RFC3339),
	}
	payloadBytes, _ := json.Marshal(payload)
	resetKey, err := utils.Encrypt(string(payloadBytes))
	if err != nil {
		return 0, apperrors.NewInternal("Failed to generate reset key", err)
	}

	rec := &domain.ForgotPassword{
		UserID:            userID,
		UserType:          strconv.Itoa(userType),
		ForgetPasswordKey: resetKey,
		CreatedOn:         now,
		ExpiryTimestamp:   expiry,
		IsPasswordChanged: false,
	}
	if err := s.forgotRepo.Create(rec); err != nil {
		return 0, err
	}
	return 1, nil
}

func (s *loginService) GetLatestForgotPasswordKey(domainName, mobileNumber string) (*dto.ForgotPasswordKeyResponse, error) {
	userID, userType, _, err := s.resolveUserByMobileNumber(domainName, mobileNumber)
	if err != nil {
		return nil, err
	}

	rec, err := s.forgotRepo.FindLatestValidKey(userID, strconv.Itoa(userType))
	if err != nil || rec == nil {
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Forgot password key not found or expired", err)
		}
		return nil, apperrors.NewNotFound("Forgot password key not found or expired", nil)
	}

	return &dto.ForgotPasswordKeyResponse{
		ForgetPasswordKey: rec.ForgetPasswordKey,
		Expiry:            rec.ExpiryTimestamp.Format(time.RFC3339),
	}, nil
}

func (s *loginService) ForgotPasswordReset(forgetPasswordKey, newPassword string) (bool, error) {
	decrypted, err := utils.Decrypt(forgetPasswordKey)
	if err != nil {
		return false, apperrors.NewBadRequest("Invalid forgot password key", err)
	}

	var payload struct {
		UserID   int64 `json:"userId"`
		UserType int   `json:"userType"`
	}
	if err := json.Unmarshal([]byte(decrypted), &payload); err != nil {
		return false, apperrors.NewBadRequest("Invalid forgot password key payload", err)
	}
	if payload.UserID == 0 {
		return false, apperrors.NewBadRequest("Invalid forgot password key", nil)
	}

	rec, err := s.forgotRepo.FindByKey(forgetPasswordKey, payload.UserID, strconv.Itoa(payload.UserType))
	if err != nil || rec == nil {
		return false, nil
	}

	encryptedNew, err := utils.Encrypt(newPassword)
	if err != nil {
		return false, apperrors.NewInternal("Failed to update password", err)
	}
	if err := s.repo.UpdatePassword(payload.UserID, encryptedNew); err != nil {
		return false, err
	}
	_ = s.forgotRepo.MarkAsUsed(rec)
	return true, nil
}

func (s *loginService) ChangePassword(domainName, mobileNumber, oldPassword, newPassword string) (bool, error) {
	userID, _, _, err := s.resolveUserByMobileNumber(domainName, mobileNumber)
	if err != nil {
		return false, err
	}

	oldEnc, err := utils.Encrypt(oldPassword)
	if err != nil {
		return false, apperrors.NewInternal("Error validating password", err)
	}
	newEnc, err := utils.Encrypt(newPassword)
	if err != nil {
		return false, apperrors.NewInternal("Failed to set new password", err)
	}

	rows, err := s.repo.ChangePassword(userID, oldEnc, newEnc)
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (s *loginService) GetProfile(domainName string, userIDStr, mobileNumber *string) (interface{}, error) {
	if userIDStr != nil && *userIDStr != "" {
		// Resolve by userId (would need domain->userType and then fetch client/lab by id)
		userType := utils.GetUserTypeFromDomain(domainName)
		if userType == utils.UserTypeClient {
			id, _ := strconv.ParseInt(*userIDStr, 10, 64)
			client, err := s.clientRepo.FindByID(id)
			if err != nil {
				return nil, apperrors.NewNotFound("Profile not found", err)
			}
			return client, nil
		}
		if userType == utils.UserTypeLab {
			id, _ := strconv.ParseInt(*userIDStr, 10, 64)
			lab, err := s.labRepo.FindByID(id)
			if err != nil {
				return nil, apperrors.NewNotFound("Profile not found", err)
			}
			return lab, nil
		}
		return nil, apperrors.NewBadRequest("Employee profile not supported", nil)
	}
	if mobileNumber != nil && *mobileNumber != "" {
		_, _, userData, err := s.resolveUserByMobileNumber(domainName, *mobileNumber)
		return userData, err
	}
	return nil, apperrors.NewBadRequest("Either userId or mobileNumber is required", nil)
}
