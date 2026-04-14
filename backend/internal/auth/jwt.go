package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type Service struct {
	secret []byte
	ttl    time.Duration
}

type tokenClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

func NewService(secret string, ttl time.Duration) *Service {
	return &Service{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (s *Service) GenerateToken(userID uuid.UUID) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	claimsJSON, err := json.Marshal(tokenClaims{
		Sub: userID.String(),
		Iat: now.Unix(),
		Exp: now.Add(s.ttl).Unix(),
	})
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := headerPart + "." + claimsPart

	signature := signHMACSHA256([]byte(unsigned), s.secret)
	signaturePart := base64.RawURLEncoding.EncodeToString(signature)

	return unsigned + "." + signaturePart, nil
}

func (s *Service) ParseToken(token string) (uuid.UUID, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expectedSig := signHMACSHA256([]byte(unsigned), s.secret)

	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	if !hmac.Equal(gotSig, expectedSig) {
		return uuid.Nil, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	if claims.Sub == "" {
		return uuid.Nil, ErrInvalidToken
	}
	if time.Now().UTC().Unix() >= claims.Exp {
		return uuid.Nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid subject", ErrInvalidToken)
	}

	return userID, nil
}

func signHMACSHA256(message, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(message)
	return mac.Sum(nil)
}
