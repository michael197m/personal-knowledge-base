package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"personal-knowledge-base/backend/internal/store"
)

type mockUserAuthStore struct {
	createUserFn      func(ctx context.Context, email, passwordHash string) (store.User, error)
	findUserByEmailFn func(ctx context.Context, email string) (store.User, error)
}

func (m *mockUserAuthStore) CreateUser(ctx context.Context, email, passwordHash string) (store.User, error) {
	if m.createUserFn == nil {
		return store.User{}, nil
	}

	return m.createUserFn(ctx, email, passwordHash)
}

func (m *mockUserAuthStore) FindUserByEmail(ctx context.Context, email string) (store.User, error) {
	if m.findUserByEmailFn == nil {
		return store.User{}, store.ErrInvalidCredentials
	}

	return m.findUserByEmailFn(ctx, email)
}

type mockTokenService struct {
	generateTokenFn func(userID uuid.UUID) (string, error)
}

func (m *mockTokenService) GenerateToken(userID uuid.UUID) (string, error) {
	if m.generateTokenFn == nil {
		return "token", nil
	}

	return m.generateTokenFn(userID)
}

func TestAuthHandlerRegisterReturnsCreatedUser(t *testing.T) {
	userID := uuid.MustParse("fd8c7e88-26b8-43fc-8113-5f7398e9c3a7")
	handler := NewAuthHandler(
		&mockUserAuthStore{
			createUserFn: func(ctx context.Context, email, passwordHash string) (store.User, error) {
				return store.User{
					ID:           userID,
					Email:        email,
					PasswordHash: passwordHash,
					CreatedAt:    time.Now().UTC(),
				}, nil
			},
		},
		&mockTokenService{
			generateTokenFn: func(uid uuid.UUID) (string, error) {
				if uid != userID {
					t.Fatalf("expected token generation for %s, got %s", userID, uid)
				}
				return "signed-token", nil
			},
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"User@Example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["token"] != "signed-token" {
		t.Fatalf("unexpected token payload: %#v", payload)
	}
}

func TestAuthHandlerRegisterRejectsDuplicateUser(t *testing.T) {
	handler := NewAuthHandler(
		&mockUserAuthStore{
			createUserFn: func(ctx context.Context, email, passwordHash string) (store.User, error) {
				return store.User{}, store.ErrUserAlreadyExists
			},
		},
		&mockTokenService{},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"user@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestAuthHandlerLoginRejectsInvalidCredentials(t *testing.T) {
	handler := NewAuthHandler(
		&mockUserAuthStore{
			findUserByEmailFn: func(ctx context.Context, email string) (store.User, error) {
				return store.User{}, store.ErrInvalidCredentials
			},
		},
		&mockTokenService{},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"user@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthHandlerLoginReturnsToken(t *testing.T) {
	userID := uuid.MustParse("7d69a1c0-9051-4280-b433-0a3a59b6965a")
	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	passwordHash := string(passwordHashBytes)

	handler := NewAuthHandler(
		&mockUserAuthStore{
			findUserByEmailFn: func(ctx context.Context, email string) (store.User, error) {
				return store.User{
					ID:           userID,
					Email:        email,
					PasswordHash: passwordHash,
				}, nil
			},
		},
		&mockTokenService{
			generateTokenFn: func(uid uuid.UUID) (string, error) {
				if uid != userID {
					t.Fatalf("unexpected user ID for token generation: %s", uid)
				}
				return "login-token", nil
			},
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"user@example.com","password":"password"}`))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["token"] != "login-token" {
		t.Fatalf("unexpected token payload: %#v", payload)
	}
}

func TestAuthHandlerRejectsInvalidPayload(t *testing.T) {
	handler := NewAuthHandler(
		&mockUserAuthStore{},
		&mockTokenService{
			generateTokenFn: func(uid uuid.UUID) (string, error) {
				return "", errors.New("should not be called")
			},
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"bad","password":"short"}`))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
