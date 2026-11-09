// Package auth holds authentication and authorization: user roles, JWT
// generation/validation, middleware and HTTP handlers.
package auth

// Role constants, matching the values stored on a user record and in JWTs.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)