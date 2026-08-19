package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"examplatform/internal/models"
	"examplatform/internal/uuidx"
)

var ErrEmailTaken = errors.New("an account with this email already exists")
var ErrInvalidCredentials = errors.New("invalid email or password")

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) CreateUser(ctx context.Context, name, email, passwordHash string, role models.Role) (*models.User, error) {
	id := uuidx.New()
	const q = `
		INSERT INTO users (id, name, email, password_hash, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, email, role, created_at`
	u := &models.User{}
	err := r.DB.QueryRow(ctx, q, id, name, email, passwordHash, role).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `SELECT id, name, email, password_hash, role, created_at FROM users WHERE email = $1`
	u := &models.User{}
	err := r.DB.QueryRow(ctx, q, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func isUniqueViolation(err error) bool {
	// pgx wraps *pgconn.PgError; SQLSTATE 23505 = unique_violation.
	type pgError interface{ SQLState() string }
	var pgErr pgError
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
