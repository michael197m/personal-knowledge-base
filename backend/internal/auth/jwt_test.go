package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceGenerateAndParseToken(t *testing.T) {
	service := NewService("test-secret", time.Hour)
	userID := uuid.MustParse("0115bc99-b3c8-47b6-8ed3-f2f782f19b75")

	token, err := service.GenerateToken(userID)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	parsedUserID, err := service.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}

	if parsedUserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, parsedUserID)
	}
}

func TestServiceParseTokenRejectsInvalidSignature(t *testing.T) {
	signingService := NewService("secret-a", time.Hour)
	verifyingService := NewService("secret-b", time.Hour)
	userID := uuid.MustParse("e4248426-2480-4f62-adbf-340c9cf730ea")

	token, err := signingService.GenerateToken(userID)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if _, err := verifyingService.ParseToken(token); err == nil {
		t.Fatal("expected ParseToken to reject token signed with another key")
	}
}
