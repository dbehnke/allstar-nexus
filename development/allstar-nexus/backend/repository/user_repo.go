package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
)

type UserRepo struct{ DB *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{DB: db} }

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, role string) (*models.User, error) {
	res, err := r.DB.ExecContext(ctx, `INSERT INTO users(email,password_hash,role) VALUES(?,?,?)`, email, passwordHash, role)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.User{ID: id, Email: email, PasswordHash: passwordHash, Role: role, CreatedAt: time.Now()}, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id,email,password_hash,role,created_at FROM users WHERE email = ?`, email)
	u := models.User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// Count returns total number of users.
func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// RoleCounts returns map[role]count.
func (r *UserRepo) RoleCounts(ctx context.Context) (map[string]int64, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT role, COUNT(1) FROM users GROUP BY role`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[string]int64{}
	for rows.Next() {
		var role string
		var cnt int64
		if err := rows.Scan(&role, &cnt); err != nil {
			return nil, err
		}
		res[role] = cnt
	}
	return res, rows.Err()
}

// NewUsersSince returns number of users created at or after 'since'.
func (r *UserRepo) NewUsersSince(ctx context.Context, since time.Time) (int64, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE created_at >= ?`, since)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// SafeUser minimal representation.
type SafeUser struct {
	ID    int64
	Email string
	Role  string
}
