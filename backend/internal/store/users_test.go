package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockUserDB struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (m *mockUserDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFn == nil {
		return &mockRow{err: errors.New("not implemented")}
	}

	return m.queryRowFn(ctx, sql, args...)
}

func TestUserStoreCreateUserReturnsUser(t *testing.T) {
	userID := uuid.MustParse("688f69f3-0cd7-4871-8ca0-05206d641f4b")
	createdAt := time.Unix(100, 0).UTC()
	store := &UserStore{
		db: &mockUserDB{
			queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return &mockRow{
					values: []any{userID, "user@example.com", "hash", createdAt},
				}
			},
		},
	}

	user, err := store.CreateUser(context.Background(), "USER@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if user.ID != userID || user.Email != "user@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestUserStoreCreateUserReturnsConflict(t *testing.T) {
	store := &UserStore{
		db: &mockUserDB{
			queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return &mockRow{
					err: &pgconn.PgError{Code: "23505"},
				}
			},
		},
	}

	_, err := store.CreateUser(context.Background(), "user@example.com", "hash")
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestUserStoreFindUserByEmailReturnsInvalidCredentials(t *testing.T) {
	store := &UserStore{
		db: &mockUserDB{
			queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return &mockRow{
					err: pgx.ErrNoRows,
				}
			},
		},
	}

	_, err := store.FindUserByEmail(context.Background(), "missing@example.com")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
