package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type userDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type UserStore struct {
	db userDB
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	var user User
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, normalizedEmail, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrUserAlreadyExists
		}

		return User{}, err
	}

	return user, nil
}

func (s *UserStore) FindUserByEmail(ctx context.Context, email string) (User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	var user User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`, normalizedEmail).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}

		return User{}, err
	}

	return user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
