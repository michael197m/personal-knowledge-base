package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrUserNotInContext = errors.New("authenticated user not found in request context")

type userIDContextKey struct{}

func ContextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	value := ctx.Value(userIDContextKey{})
	if value == nil {
		return uuid.Nil, ErrUserNotInContext
	}

	userID, ok := value.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, ErrUserNotInContext
	}

	return userID, nil
}
