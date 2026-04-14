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

type AuthHandler struct {
	users  userAuthStore
	tokens tokenService
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
	} `json:"user"`
}

func NewAuthHandler(users userAuthStore, tokens tokenService) *AuthHandler {
	return &AuthHandler{
		users:  users,
		tokens: tokens,
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

	respondJSON(w, http.StatusCreated, makeAuthResponse(user.ID, user.Email, token))
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

	respondJSON(w, http.StatusOK, makeAuthResponse(user.ID, user.Email, token))
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

func makeAuthResponse(userID uuid.UUID, email, token string) authResponse {
	response := authResponse{Token: token}
	response.User.ID = userID
	response.User.Email = email
	return response
}
