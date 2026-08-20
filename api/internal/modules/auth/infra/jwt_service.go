package infra

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type jwtClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
}

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

func (s *JWTService) Generate(userID uuid.UUID, email, role string) (*domain.TokenPair, error) {
	now := time.Now()

	accessToken, err := s.createToken(userID, email, role, domain.TokenTypeAccess, now, accessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.createToken(userID, email, role, domain.TokenTypeRefresh, now, refreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

func (s *JWTService) Validate(token string) (*domain.Claims, error) {
	return s.parseToken(token)
}

func (s *JWTService) RefreshTokens(refreshToken string) (*domain.TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != domain.TokenTypeRefresh {
		return nil, domain.ErrNotRefreshToken
	}

	return s.Generate(claims.UserID, claims.Email, claims.Role)
}

func (s *JWTService) createToken(userID uuid.UUID, email, role, tokenType string, now time.Time, ttl time.Duration) (string, error) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID:    userID.String(),
		Email:     email,
		Role:      role,
		TokenType: tokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) parseToken(tokenStr string) (*domain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		if isExpiredError(err) {
			return nil, domain.ErrExpiredToken
		}
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	return &domain.Claims{
		UserID:    userID,
		Email:     claims.Email,
		Role:      claims.Role,
		TokenType: claims.TokenType,
	}, nil
}

func isExpiredError(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}
