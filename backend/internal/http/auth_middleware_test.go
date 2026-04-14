package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"personal-knowledge-base/backend/internal/store"
)

type mockTokenVerifier struct {
	parseTokenFn func(token string) (uuid.UUID, error)
}

func (m mockTokenVerifier) ParseToken(token string) (uuid.UUID, error) {
	if m.parseTokenFn == nil {
		return uuid.Nil, errors.New("not implemented")
	}

	return m.parseTokenFn(token)
}

func TestWithAuthRejectsMissingCredentials(t *testing.T) {
	handler := withAuth(mockTokenVerifier{}, "pkb_auth_token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWithAuthUsesCookieToken(t *testing.T) {
	expectedUserID := uuid.MustParse("f4be9226-af20-4738-b79e-9f66c972550d")
	handler := withAuth(
		mockTokenVerifier{
			parseTokenFn: func(token string) (uuid.UUID, error) {
				if token != "cookie-token" {
					t.Fatalf("unexpected token: %q", token)
				}
				return expectedUserID, nil
			},
		},
		"pkb_auth_token",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := store.UserIDFromContext(r.Context())
			if err != nil {
				t.Fatalf("expected user ID in context: %v", err)
			}
			if userID != expectedUserID {
				t.Fatalf("expected user ID %s, got %s", expectedUserID, userID)
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	req.AddCookie(&http.Cookie{Name: "pkb_auth_token", Value: "cookie-token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithAuthInjectsUserIDIntoContext(t *testing.T) {
	expectedUserID := uuid.MustParse("9ce2ebd0-2336-4845-bf90-4f55753fb8fb")
	handler := withAuth(
		mockTokenVerifier{
			parseTokenFn: func(token string) (uuid.UUID, error) {
				if token != "valid-token" {
					t.Fatalf("unexpected token: %q", token)
				}
				return expectedUserID, nil
			},
		},
		"pkb_auth_token",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := store.UserIDFromContext(r.Context())
			if err != nil {
				t.Fatalf("expected user ID in context: %v", err)
			}
			if userID != expectedUserID {
				t.Fatalf("expected user ID %s, got %s", expectedUserID, userID)
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
