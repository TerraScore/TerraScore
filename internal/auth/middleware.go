package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/terrascore/api/internal/platform"
)

type userContextKey struct{}

// UserContext holds authenticated user info extracted from JWT.
type UserContext struct {
	KeycloakID string
	Username   string
	Email      string
	Roles      []string
}

// GetUser returns the authenticated user from context.
func GetUser(ctx context.Context) *UserContext {
	if u, ok := ctx.Value(userContextKey{}).(*UserContext); ok {
		return u
	}
	return nil
}

// SetUser injects a UserContext into the context. For testing only.
func SetUser(ctx context.Context, user *UserContext) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// JWTAuth middleware validates the Keycloak JWT and injects UserContext.
func JWTAuth(kc *KeycloakClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "invalid authorization format")
				return
			}

			claims, err := kc.ValidateToken(parts[1])
			if err != nil {
				platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "invalid token")
				return
			}

			userCtx := extractUserContext(*claims)
			ctx := context.WithValue(r.Context(), userContextKey{}, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks that the authenticated user has the specified role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
				return
			}

			for _, required := range roles {
				for _, userRole := range user.Roles {
					if userRole == required {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			platform.JSONError(w, http.StatusForbidden, platform.CodeForbidden, "insufficient permissions")
		})
	}
}

func extractUserContext(claims jwt.MapClaims) *UserContext {
	uc := &UserContext{}

	if sub, ok := claims["sub"].(string); ok {
		uc.KeycloakID = sub
	}
	if username, ok := claims["preferred_username"].(string); ok {
		uc.Username = username
	}
	if email, ok := claims["email"].(string); ok {
		uc.Email = email
	}

	// Extract realm roles
	if realmRoles, ok := claims["realm_roles"].([]any); ok {
		for _, r := range realmRoles {
			if role, ok := r.(string); ok {
				uc.Roles = append(uc.Roles, role)
			}
		}
	}

	// Also check realm_access.roles (standard Keycloak claim)
	if realmAccess, ok := claims["realm_access"].(map[string]any); ok {
		if roles, ok := realmAccess["roles"].([]any); ok {
			for _, r := range roles {
				if role, ok := r.(string); ok {
					uc.Roles = append(uc.Roles, role)
				}
			}
		}
	}

	return uc
}
