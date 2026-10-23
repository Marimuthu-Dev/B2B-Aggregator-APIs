package service

import (
	"errors"
	"strings"
	"testing"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"gorm.io/gorm"
)

type mockLoginRepository struct {
	findByUserIDFn               func(userID int64) (*domain.Login, error)
	createFn                     func(l *domain.Login) error
	updatePasswordFn             func(userID int64, newPassword string) error
	updatePasswordWithUserTypeFn func(userID int64, newPassword, userType string) error
	authenticateFn               func(userID int64, encryptedPassword, userType string) (bool, error)
	changePasswordFn             func(userID int64, oldEncryptedPassword, newEncryptedPassword string) (int64, error)
}

func (m *mockLoginRepository) FindByUserID(userID int64) (*domain.Login, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockLoginRepository) Create(l *domain.Login) error {
	if m.createFn != nil {
		return m.createFn(l)
	}
	return nil
}

func (m *mockLoginRepository) UpdatePassword(userID int64, newPassword string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(userID, newPassword)
	}
	return nil
}

func (m *mockLoginRepository) UpdatePasswordWithUserType(userID int64, newPassword, userType string) error {
	if m.updatePasswordWithUserTypeFn != nil {
		return m.updatePasswordWithUserTypeFn(userID, newPassword, userType)
	}
	return nil
}

func (m *mockLoginRepository) Authenticate(userID int64, encryptedPassword, userType string) (bool, error) {
	if m.authenticateFn != nil {
		return m.authenticateFn(userID, encryptedPassword, userType)
	}
	return false, nil
}

func (m *mockLoginRepository) ChangePassword(userID int64, oldEncryptedPassword, newEncryptedPassword string) (int64, error) {
	if m.changePasswordFn != nil {
		return m.changePasswordFn(userID, oldEncryptedPassword, newEncryptedPassword)
	}
	return 0, nil
}

func TestStoreUpdateRequest_HasAtLeastOneField(t *testing.T) {
	pwd := "newSecret123"
	blankPwd := "   "
	emptyPwd := ""

	t.Run("only password provided", func(t *testing.T) {
		req := dto.StoreUpdateRequest{Password: &pwd}
		if !req.HasAtLeastOneField() {
			t.Errorf("expected HasAtLeastOneField to be true when valid password provided")
		}
	})

	t.Run("empty password provided", func(t *testing.T) {
		req := dto.StoreUpdateRequest{Password: &emptyPwd}
		if req.HasAtLeastOneField() {
			t.Errorf("expected HasAtLeastOneField to be false when empty password provided")
		}
	})

	t.Run("whitespace password provided", func(t *testing.T) {
		req := dto.StoreUpdateRequest{Password: &blankPwd}
		if req.HasAtLeastOneField() {
			t.Errorf("expected HasAtLeastOneField to be false when whitespace password provided")
		}
	})

	t.Run("no fields provided", func(t *testing.T) {
		req := dto.StoreUpdateRequest{}
		if req.HasAtLeastOneField() {
			t.Errorf("expected HasAtLeastOneField to be false when no fields provided")
		}
	})
}

func TestSyncStorePassword(t *testing.T) {
	t.Setenv("LOGIN_ENC_KEY", "12345678901234567890123456789012")
	t.Setenv("LOGIN_ENC_SALT", "12345678901234567890123456789012")

	const testStoreID = int64(7000001)

	t.Run("empty password is no-op", func(t *testing.T) {
		mockLogin := &mockLoginRepository{
			findByUserIDFn: func(userID int64) (*domain.Login, error) {
				t.Fatalf("FindByUserID should not be called for empty password")
				return nil, nil
			},
		}
		svc := &storeService{loginRepo: mockLogin}
		err := svc.syncStorePassword(testStoreID, "   ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("existing login updates password and user type", func(t *testing.T) {
		var updatedUserID int64
		var updatedEncryptedPwd string
		var updatedUserType string

		mockLogin := &mockLoginRepository{
			findByUserIDFn: func(userID int64) (*domain.Login, error) {
				if userID != testStoreID {
					t.Fatalf("expected userID %d, got %d", testStoreID, userID)
				}
				return &domain.Login{
					UserID:   userID,
					UserType: "4",
					Pwd:      "oldEncryptedPassword",
				}, nil
			},
			updatePasswordWithUserTypeFn: func(userID int64, newPassword, userType string) error {
				updatedUserID = userID
				updatedEncryptedPwd = newPassword
				updatedUserType = userType
				return nil
			},
		}

		svc := &storeService{loginRepo: mockLogin}
		rawPassword := "NewStorePass@2026"
		err := svc.syncStorePassword(testStoreID, rawPassword)
		if err != nil {
			t.Fatalf("syncStorePassword failed: %v", err)
		}

		if updatedUserID != testStoreID {
			t.Errorf("expected updatedUserID %d, got %d", testStoreID, updatedUserID)
		}
		if updatedUserType != utils.UserTypeToLoginString(utils.UserTypeStore) {
			t.Errorf("expected userType %s, got %s", utils.UserTypeToLoginString(utils.UserTypeStore), updatedUserType)
		}
		if updatedEncryptedPwd == "" || updatedEncryptedPwd == rawPassword {
			t.Errorf("expected encrypted password, got %s", updatedEncryptedPwd)
		}
	})

	t.Run("missing login creates new login record", func(t *testing.T) {
		var createdLogin *domain.Login

		mockLogin := &mockLoginRepository{
			findByUserIDFn: func(userID int64) (*domain.Login, error) {
				return nil, gorm.ErrRecordNotFound
			},
			createFn: func(l *domain.Login) error {
				createdLogin = l
				return nil
			},
		}

		svc := &storeService{loginRepo: mockLogin}
		rawPassword := "BrandNewPass@2026"
		err := svc.syncStorePassword(testStoreID, rawPassword)
		if err != nil {
			t.Fatalf("syncStorePassword failed: %v", err)
		}

		if createdLogin == nil {
			t.Fatalf("expected Create to be called, but it was not")
		}
		if createdLogin.UserID != testStoreID {
			t.Errorf("expected createdLogin.UserID %d, got %d", testStoreID, createdLogin.UserID)
		}
		if createdLogin.UserType != utils.UserTypeToLoginString(utils.UserTypeStore) {
			t.Errorf("expected createdLogin.UserType %s, got %s", utils.UserTypeToLoginString(utils.UserTypeStore), createdLogin.UserType)
		}
		if createdLogin.Pwd == "" || createdLogin.Pwd == rawPassword {
			t.Errorf("expected encrypted password, got %s", createdLogin.Pwd)
		}
	})

	t.Run("db error checking login fails safe", func(t *testing.T) {
		expectedErr := errors.New("db connection failure")
		mockLogin := &mockLoginRepository{
			findByUserIDFn: func(userID int64) (*domain.Login, error) {
				return nil, expectedErr
			},
		}

		svc := &storeService{loginRepo: mockLogin}
		err := svc.syncStorePassword(testStoreID, "somePassword")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Failed to check store login") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
