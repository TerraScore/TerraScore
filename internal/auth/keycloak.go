package auth

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/terrascore/api/internal/platform"
)

// KeycloakClient communicates with the Keycloak Admin REST API.
type KeycloakClient struct {
	baseURL      string
	realm        string
	adminUser    string
	adminPass    string
	clientID     string
	clientSecret string
	httpClient   *http.Client

	// Cached admin token
	mu          sync.Mutex
	adminToken  string
	tokenExpiry time.Time

	// Cached JWKS public keys
	jwksMu   sync.RWMutex
	jwksKeys map[string]*rsa.PublicKey
	jwksExp  time.Time
}

// NewKeycloakClient creates a new Keycloak admin client.
func NewKeycloakClient(cfg platform.KeycloakConfig) *KeycloakClient {
	return &KeycloakClient{
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		realm:        cfg.Realm,
		adminUser:    cfg.AdminUser,
		adminPass:    cfg.AdminPass,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		jwksKeys:     make(map[string]*rsa.PublicKey),
	}
}

// TokenResponse from Keycloak.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// KeycloakUser represents a user in Keycloak.
type KeycloakUser struct {
	ID              string              `json:"id,omitempty"`
	Username        string              `json:"username"`
	Email           string              `json:"email,omitempty"`
	FirstName       string              `json:"firstName,omitempty"`
	LastName        string              `json:"lastName,omitempty"`
	Enabled         bool                `json:"enabled"`
	Attributes      map[string][]string `json:"attributes,omitempty"`
	RequiredActions []string            `json:"requiredActions"`
}

// getAdminToken retrieves or refreshes the admin access token.
func (kc *KeycloakClient) getAdminToken(ctx context.Context) (string, error) {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.adminToken != "" && time.Now().Before(kc.tokenExpiry) {
		return kc.adminToken, nil
	}

	data := url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {kc.adminUser},
		"password":   {kc.adminPass},
	}

	resp, err := kc.httpClient.PostForm(
		fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", kc.baseURL),
		data,
	)
	if err != nil {
		return "", fmt.Errorf("requesting admin token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("admin token request failed (%d): %s", resp.StatusCode, body)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decoding admin token: %w", err)
	}

	kc.adminToken = tokenResp.AccessToken
	kc.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-30) * time.Second)

	return kc.adminToken, nil
}

// CreateUser creates a user in Keycloak and returns their ID.
func (kc *KeycloakClient) CreateUser(ctx context.Context, user KeycloakUser) (string, error) {
	token, err := kc.getAdminToken(ctx)
	if err != nil {
		return "", err
	}

	body, _ := json.Marshal(user)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/admin/realms/%s/users", kc.baseURL, kc.realm),
		bytes.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := kc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("creating keycloak user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return "", platform.NewConflict("user already exists in Keycloak")
	}
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak create user failed (%d): %s", resp.StatusCode, respBody)
	}

	// Extract user ID from Location header
	location := resp.Header.Get("Location")
	parts := strings.Split(location, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("no user ID in location header")
	}

	return parts[len(parts)-1], nil
}

// AssignRealmRole assigns a realm role to a user.
func (kc *KeycloakClient) AssignRealmRole(ctx context.Context, userID, roleName string) error {
	token, err := kc.getAdminToken(ctx)
	if err != nil {
		return err
	}

	// First get the role representation
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/admin/realms/%s/roles/%s", kc.baseURL, kc.realm, roleName),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("getting role: %w", err)
	}
	defer resp.Body.Close()

	var role json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return fmt.Errorf("decoding role: %w", err)
	}

	// Assign the role
	body, _ := json.Marshal([]json.RawMessage{role})
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/admin/realms/%s/users/%s/role-mappings/realm", kc.baseURL, kc.realm, userID),
		bytes.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp2, err := kc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("assigning role: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("assign role failed (%d): %s", resp2.StatusCode, respBody)
	}

	return nil
}

// GetToken exchanges credentials for tokens (direct grant).
func (kc *KeycloakClient) GetToken(ctx context.Context, username, password string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type": {"password"},
		"client_id":  {kc.clientID},
		"username":   {username},
		"password":   {password},
	}
	if kc.clientSecret != "" {
		data.Set("client_secret", kc.clientSecret)
	}

	resp, err := kc.httpClient.PostForm(
		fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", kc.baseURL, kc.realm),
		data,
	)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed (%d): %s", resp.StatusCode, body)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decoding token: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken refreshes an access token.
func (kc *KeycloakClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {kc.clientID},
		"refresh_token": {refreshToken},
	}
	if kc.clientSecret != "" {
		data.Set("client_secret", kc.clientSecret)
	}

	resp, err := kc.httpClient.PostForm(
		fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", kc.baseURL, kc.realm),
		data,
	)
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, body)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decoding refresh token: %w", err)
	}

	return &tokenResp, nil
}

// SetTemporaryPassword sets a temporary password for a Keycloak user.
func (kc *KeycloakClient) SetTemporaryPassword(ctx context.Context, userID, password string) error {
	token, err := kc.getAdminToken(ctx)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(map[string]any{
		"type":      "password",
		"value":     password,
		"temporary": false,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/admin/realms/%s/users/%s/reset-password", kc.baseURL, kc.realm, userID),
		bytes.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := kc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("setting password: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set password failed (%d): %s", resp.StatusCode, respBody)
	}

	return nil
}

// JWKS key structure
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// GetPublicKey returns the RSA public key for the given key ID, fetching JWKS if needed.
func (kc *KeycloakClient) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	kc.jwksMu.RLock()
	key, ok := kc.jwksKeys[kid]
	expired := time.Now().After(kc.jwksExp)
	kc.jwksMu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	// Refresh JWKS
	if err := kc.refreshJWKS(); err != nil {
		return nil, err
	}

	kc.jwksMu.RLock()
	defer kc.jwksMu.RUnlock()

	key, ok = kc.jwksKeys[kid]
	if !ok {
		return nil, fmt.Errorf("key %s not found in JWKS", kid)
	}

	return key, nil
}

func (kc *KeycloakClient) refreshJWKS() error {
	resp, err := kc.httpClient.Get(
		fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", kc.baseURL, kc.realm),
	)
	if err != nil {
		return fmt.Errorf("fetching JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decoding JWKS: %w", err)
	}

	kc.jwksMu.Lock()
	defer kc.jwksMu.Unlock()

	for _, k := range jwks.Keys {
		if k.Kty != "RSA" || k.Use != "sig" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}

		n := new(big.Int).SetBytes(nBytes)
		e := int(new(big.Int).SetBytes(eBytes).Int64())

		kc.jwksKeys[k.Kid] = &rsa.PublicKey{N: n, E: e}
	}

	kc.jwksExp = time.Now().Add(1 * time.Hour)

	return nil
}

// ValidateToken parses and validates a JWT, returning claims.
func (kc *KeycloakClient) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}
		return kc.GetPublicKey(kid)
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &claims, nil
}
