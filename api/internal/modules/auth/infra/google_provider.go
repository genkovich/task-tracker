package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
)

const googleAPITimeout = 10 * time.Second

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string // overridable for tests, defaults to Google
	TokenURL     string // overridable for tests, defaults to Google
	UserInfoURL  string // overridable for tests, defaults to Google
}

type GoogleOAuthProvider struct {
	cfg    GoogleOAuthConfig
	client *http.Client
}

func NewGoogleOAuthProvider(cfg GoogleOAuthConfig) *GoogleOAuthProvider {
	if cfg.AuthURL == "" {
		cfg.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = "https://oauth2.googleapis.com/token"
	}
	if cfg.UserInfoURL == "" {
		cfg.UserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	}
	return &GoogleOAuthProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: googleAPITimeout},
	}
}

func (p *GoogleOAuthProvider) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":     {p.cfg.ClientID},
		"redirect_uri":  {p.cfg.RedirectURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return p.cfg.AuthURL + "?" + params.Encode()
}

func (p *GoogleOAuthProvider) ExchangeCode(ctx context.Context, code string) (*domain.OAuthTokens, error) {
	ctx, cancel := context.WithTimeout(ctx, googleAPITimeout)
	defer cancel()

	data := url.Values{
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"redirect_uri":  {p.cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrGoogleAPIFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrGoogleAPIFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrGoogleAPIFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: token exchange returned %d: %s", domain.ErrInvalidCode, resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrGoogleAPIFailed, err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("%w: empty access_token in response", domain.ErrGoogleAPIFailed)
	}

	return &domain.OAuthTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

func (p *GoogleOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*domain.GoogleUserInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, googleAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrGoogleAPIFailed, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrGoogleAPIFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: userinfo returned %d", domain.ErrGoogleAPIFailed, resp.StatusCode)
	}

	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrGoogleAPIFailed, err)
	}

	return &domain.GoogleUserInfo{
		Sub:           info.Sub,
		Email:         info.Email,
		Name:          info.Name,
		GivenName:     info.GivenName,
		FamilyName:    info.FamilyName,
		Picture:       info.Picture,
		EmailVerified: info.EmailVerified,
	}, nil
}
