package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthUserInfo represents normalized user profile details from an OAuth provider.
type OAuthUserInfo struct {
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	AvatarURL      string `json:"avatar_url"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ==========================================
// GitHub OAuth
// ==========================================

// BuildGitHubAuthURL constructs the redirect URL for GitHub OAuth.
func BuildGitHubAuthURL(clientID, redirectURI, state string) string {
	v := url.Values{}
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("scope", "read:user user:email")
	v.Set("state", state)
	return "https://github.com/login/oauth/authorize?" + v.Encode()
}

// ExchangeGitHubCode exchanges the authorization code for a GitHub access token.
func ExchangeGitHubCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (string, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("github token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse github token response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("github oauth error: %s (%s)", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", errors.New("github returned empty access token")
	}

	return result.AccessToken, nil
}

// FetchGitHubUser retrieves the user's profile and primary verified email from GitHub.
func FetchGitHubUser(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github user request failed: %w", err)
	}
	defer resp.Body.Close()

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("decode github user failed: %w", err)
	}

	userName := ghUser.Name
	if userName == "" {
		userName = ghUser.Login
	}

	userEmail := ghUser.Email
	// If email is private on GitHub profile, query the /user/emails endpoint
	if userEmail == "" {
		emailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
		if err == nil {
			emailReq.Header.Set("Authorization", "Bearer "+accessToken)
			emailReq.Header.Set("Accept", "application/vnd.github.v3+json")
			if emailResp, err := httpClient.Do(emailReq); err == nil {
				defer emailResp.Body.Close()
				var emails []struct {
					Email    string `json:"email"`
					Primary  bool   `json:"primary"`
					Verified bool   `json:"verified"`
				}
				if json.NewDecoder(emailResp.Body).Decode(&emails) == nil {
					for _, e := range emails {
						if e.Primary && e.Verified {
							userEmail = e.Email
							break
						}
					}
					if userEmail == "" && len(emails) > 0 {
						userEmail = emails[0].Email
					}
				}
			}
		}
	}

	if userEmail == "" {
		userEmail = fmt.Sprintf("%s@users.noreply.github.com", ghUser.Login)
	}

	return &OAuthUserInfo{
		ProviderUserID: fmt.Sprintf("%d", ghUser.ID),
		Email:          userEmail,
		Name:           userName,
		AvatarURL:      ghUser.AvatarURL,
	}, nil
}

// ==========================================
// Google OAuth
// ==========================================

// BuildGoogleAuthURL constructs the redirect URL for Google OAuth.
func BuildGoogleAuthURL(clientID, redirectURI, state string) string {
	v := url.Values{}
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("scope", "openid profile email")
	v.Set("access_type", "offline")
	v.Set("state", state)
	return "https://accounts.google.com/o/oauth2/v2/auth?" + v.Encode()
}

// ExchangeGoogleCode exchanges the authorization code for a Google access token.
func ExchangeGoogleCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (string, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("google token request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse google token response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("google oauth error: %s (%s)", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", errors.New("google returned empty access token")
	}

	return result.AccessToken, nil
}

// FetchGoogleUser retrieves the user's profile from Google.
func FetchGoogleUser(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	var gUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("decode google user failed: %w", err)
	}

	if gUser.Email == "" {
		return nil, errors.New("google user has no email")
	}

	name := gUser.Name
	if name == "" {
		name = strings.Split(gUser.Email, "@")[0]
	}

	return &OAuthUserInfo{
		ProviderUserID: gUser.ID,
		Email:          gUser.Email,
		Name:           name,
		AvatarURL:      gUser.Picture,
	}, nil
}
