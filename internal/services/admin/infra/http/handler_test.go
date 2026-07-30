package adminhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakery/internal/inbound/api/contract"
	"bakery/internal/services/admin/usecase/admin"
	authuc "bakery/internal/services/auth/usecase/auth"
)

func TestHandleCreateUserReturnsConflictForExistingUsername(t *testing.T) {
	handler := New(createUserService{err: authuc.ErrUsernameTaken})
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(
		`{"username":"existing","password":"new-password","role":"shop"}`,
	))
	response := httptest.NewRecorder()

	handler.handleCreateUser(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
	var body contract.Error
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != authuc.ErrUsernameTaken.Message {
		t.Fatalf("error = %q, want %q", body.Error, authuc.ErrUsernameTaken.Message)
	}
}

type createUserService struct {
	adminuc.UseCase
	err error
}

func (s createUserService) CreateUser(context.Context, adminuc.CreateUserInput) (adminuc.User, error) {
	return adminuc.User{}, s.err
}
