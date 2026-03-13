package auth

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/user"
	"toggly.com/m/pkg/crypto"
	"toggly.com/m/pkg/response"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

const minPasswordLen = 8

// AuthResponse is returned by Register, Login, and Refresh.
type AuthResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int64        `json:"expiresIn"` // seconds until access token expiry
	User         *domain.User `json:"user"`
}

// Service handles auth business logic.
type Service struct {
	userRepo    *user.Repository
	secret      string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewService returns a new auth service.
func NewService(userRepo *user.Repository, secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		userRepo:   userRepo,
		secret:     secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Register creates a new user and returns tokens and user.
func (s *Service) Register(ctx context.Context, email, password, name string) (*AuthResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if email == "" {
		return nil, response.BadRequest("email is required")
	}
	if !emailRegex.MatchString(email) {
		return nil, response.BadRequest("invalid email format")
	}
	if len(password) < minPasswordLen {
		return nil, response.BadRequest("password must be at least 8 characters")
	}
	if name == "" {
		return nil, response.BadRequest("name is required")
	}
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return nil, response.Conflict("email already registered")
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := &domain.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return s.issueTokens(ctx, u)
}

// Login authenticates by email/password and returns tokens and user.
func (s *Service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return nil, response.BadRequest("email and password are required")
	}
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil || !crypto.CheckPassword(u.PasswordHash, password) {
		return nil, response.Unauthorized("invalid email or password")
	}
	return s.issueTokens(ctx, u)
}

// Refresh validates the refresh token and returns new token pair and user.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	if refreshToken == "" {
		return nil, response.BadRequest("refresh token is required")
	}
	userID, err := ParseRefreshToken(refreshToken, s.secret)
	if err != nil {
		return nil, response.Unauthorized("invalid or expired refresh token")
	}
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, response.Unauthorized("invalid refresh token")
	}
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, response.Unauthorized("user not found")
	}
	return s.issueTokens(ctx, u)
}

func (s *Service) issueTokens(ctx context.Context, u *domain.User) (*AuthResponse, error) {
	userID := u.ID.String()
	access, err := CreateAccessToken(userID, s.secret, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("create access token: %w", err)
	}
	refresh, err := CreateRefreshToken(userID, s.secret, s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}
	return &AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
		User:         u,
	}, nil
}
