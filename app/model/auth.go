package model

import "time"

// Request untuk register
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,username"`
	Email    string `json:"email"    validate:"required,email,max=120"`
	Password string `json:"password" validate:"required,max=72,strongpassword"`
}

// Request untuk login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Request untuk refresh & logout
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Response setelah login/refresh berhasil
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Data refresh token yang disimpan di database
type RefreshToken struct {
	ID        int64
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// Identitas user yang dibawa access token (JWT)
type AuthUser struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
