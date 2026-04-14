package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"personal-knowledge-base/backend/internal/store"
)

type userAuthStore interface {
	CreateUser(ctx context.Context, email, passwordHash string) (store.User, error)
	FindUserByEmail(ctx context.Context, email string) (store.User, error)
}

type tokenService interface {
	GenerateToken(userID uuid.UUID) (string, error)
}

type AuthCookieConfig struct {
	Name   string
	Secure bool
}

type AuthHandler struct {
	users        userAuthStore
	tokens       tokenService
	cookieConfig AuthCookieConfig
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
	} `json:"user"`
}

func NewAuthHandler(users userAuthStore, tokens tokenService, cookieConfig AuthCookieConfig) *AuthHandler {
	if cookieConfig.Name == "" {
		cookieConfig.Name = "pkb_auth_token"
	}

	return &AuthHandler{
		users:        users,
		tokens:       tokens,
		cookieConfig: cookieConfig,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAndValidateAuthRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to create account"})
		return
	}

	user, err := h.users.CreateUser(r.Context(), req.Email, string(passwordHashBytes))
	if err != nil {
		if errors.Is(err, store.ErrUserAlreadyExists) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}

		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to create account"})
		return
	}

	token, err := h.tokens.GenerateToken(user.ID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to create account"})
		return
	}

	h.setAuthCookie(w, token)
	respondJSON(w, http.StatusCreated, makeAuthResponse(user.ID, user.Email))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAndValidateAuthRequest(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, err := h.users.FindUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}

		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to login"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	token, err := h.tokens.GenerateToken(user.ID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to login"})
		return
	}

	h.setAuthCookie(w, token)
	respondJSON(w, http.StatusOK, makeAuthResponse(user.ID, user.Email))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieConfig.Name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieConfig.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	respondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func decodeAndValidateAuthRequest(r *http.Request) (authRequest, error) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return authRequest{}, errors.New("invalid JSON payload")
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return authRequest{}, errors.New("valid email is required")
	}
	if len(req.Password) < 8 {
		return authRequest{}, errors.New("password must be at least 8 characters")
	}

	return req, nil
}

func makeAuthResponse(userID uuid.UUID, email string) authResponse {
	response := authResponse{}
	response.User.ID = userID
	response.User.Email = email
	return response
}

func (h *AuthHandler) setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieConfig.Name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieConfig.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24,
	})
}
