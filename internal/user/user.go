// Package user holds the user domain: model, roles, hashing and the
// persistence contract. Passwords are stored as bcrypt hashes only.
package user

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role is the access level of a user.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func (r Role) String() string { return string(r) }

// User identifies an account. PasswordHash is a bcrypt hash; the plaintext
// password is never retained.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Repository is the persistence contract for users.
type Repository interface {
	// Init creates the users table if it does not exist.
	Init(ctx context.Context) error
	// Create inserts a new user and sets u.ID / u.CreatedAt.
	Create(ctx context.Context, u *User) error
	// GetByUsername returns the user, or nil if not found.
	GetByUsername(ctx context.Context, username string) (*User, error)
	// GetByID returns the user, or nil if not found.
	GetByID(ctx context.Context, id int64) (*User, error)
}

// HashPassword returns a bcrypt hash of the plaintext password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword reports whether plain matches the stored hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}