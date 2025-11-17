package models

import (
	"time"
)

// User represents an API user
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	APIKey       string    `json:"api_key,omitempty" db:"api_key"`
	Quota        int       `json:"quota" db:"quota"` // Max videos per day
	UsedQuota    int       `json:"used_quota" db:"used_quota"`
	QuotaResetAt time.Time `json:"quota_reset_at" db:"quota_reset_at"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserRole represents user roles
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Role   UserRole `json:"role"`
}
