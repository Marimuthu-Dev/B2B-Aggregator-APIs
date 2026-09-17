package handlers

import (
	"errors"
	"testing"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

func TestResolveStoreListScope(t *testing.T) {
	otherClient := int64(99)

	t.Run("employee without clientId lists all stores", func(t *testing.T) {
		scope, err := resolveStoreListScope(utils.UserTypeEmployee, 10, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope.ClientID != nil {
			t.Fatalf("ClientID = %v, want nil", *scope.ClientID)
		}
		if scope.StoreID != nil {
			t.Fatalf("StoreID = %v, want nil", *scope.StoreID)
		}
	})

	t.Run("employee with clientId keeps that filter", func(t *testing.T) {
		scope, err := resolveStoreListScope(utils.UserTypeEmployee, 10, &otherClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope.ClientID == nil || *scope.ClientID != otherClient {
			t.Fatalf("ClientID = %v, want %d", scope.ClientID, otherClient)
		}
		if scope.StoreID != nil {
			t.Fatalf("StoreID = %v, want nil", *scope.StoreID)
		}
	})

	t.Run("client forces JWT userId as ClientID", func(t *testing.T) {
		const jwtUserID int64 = 42
		scope, err := resolveStoreListScope(utils.UserTypeClient, jwtUserID, &otherClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope.ClientID == nil || *scope.ClientID != jwtUserID {
			t.Fatalf("ClientID = %v, want JWT userId %d", scope.ClientID, jwtUserID)
		}
		if scope.StoreID != nil {
			t.Fatalf("StoreID = %v, want nil", *scope.StoreID)
		}
	})

	t.Run("lab is forbidden", func(t *testing.T) {
		_, err := resolveStoreListScope(utils.UserTypeLab, 7, nil)
		if err == nil {
			t.Fatal("expected forbidden error")
		}
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Kind != apperrors.KindForbidden {
			t.Fatalf("error = %v, want forbidden AppError", err)
		}
	})

	t.Run("store forces JWT userId as StoreID", func(t *testing.T) {
		const jwtUserID int64 = 55
		scope, err := resolveStoreListScope(utils.UserTypeStore, jwtUserID, &otherClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope.ClientID != nil {
			t.Fatalf("ClientID = %v, want nil (query clientId ignored)", *scope.ClientID)
		}
		if scope.StoreID == nil || *scope.StoreID != jwtUserID {
			t.Fatalf("StoreID = %v, want JWT userId %d", scope.StoreID, jwtUserID)
		}
	})
}
