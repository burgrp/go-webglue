# Authentication Guide

Implement authentication and authorization in go-webglue applications.

## Overview

go-webglue provides the `CallChecker` interface for implementing authentication. This interface allows you to:

- Validate requests before API calls
- Inject authenticated user data into API methods
- Implement role-based access control
- Add request logging and rate limiting

## CallChecker Interface

### Basic Implementation

```go
type CallChecker interface {
    CheckCall(request *http.Request, functionName string) ([]any, error)
}
```

Implement this interface on your API struct to intercept all API calls:

```go
type MyApi struct {
    db *Database
}

func (api *MyApi) CheckCall(req *http.Request, functionName string) ([]any, error) {
    // Extract token from header
    token := req.Header.Get("Authorization")

    // Validate token
    user, err := api.db.ValidateToken(token)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    // Inject user into all API method calls
    return []any{user}, nil
}

// Now all API methods receive the user
func (api *MyApi) GetProfile(user *User) (*Profile, error) {
    return api.db.GetProfile(user.ID)
}

func (api *MyApi) UpdateSettings(user *User, settings Settings) error {
    if !user.CanUpdateSettings {
        return errors.New("forbidden")
    }
    return api.db.UpdateSettings(user.ID, settings)
}
```

### How It Works

1. Client calls API method
2. `CheckCall` is invoked first
3. If `CheckCall` returns error, request fails
4. If successful, returned values are injected into the API method
5. API method executes with injected parameters

### Important Notes

- `CheckCall` itself **cannot** be called via API (protected)
- Returned values are type-matched to function parameters
- Multiple injected parameters are supported
- Injected params combined with JSON params from client

## Token-Based Authentication

### JWT Example

```go
package auth

import (
    "errors"
    "net/http"
    "strings"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID   int    `json:"userId"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

type AuthApi struct {
    secretKey []byte
    db        *Database
}

func (api *AuthApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    // Skip auth for login endpoint
    if funcName == "Login" {
        return nil, nil
    }

    // Extract token
    authHeader := req.Header.Get("Authorization")
    if authHeader == "" {
        return nil, errors.New("missing authorization header")
    }

    tokenString := strings.TrimPrefix(authHeader, "Bearer ")

    // Parse and validate JWT
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
        return api.secretKey, nil
    })

    if err != nil || !token.Valid {
        return nil, errors.New("invalid token")
    }

    claims := token.Claims.(*Claims)

    // Load full user from database
    user, err := api.db.GetUser(claims.UserID)
    if err != nil {
        return nil, errors.New("user not found")
    }

    // Inject user into API calls
    return []any{user}, nil
}

func (api *AuthApi) Login(username, password string) (string, error) {
    // Validate credentials
    user, err := api.db.ValidateCredentials(username, password)
    if err != nil {
        return "", errors.New("invalid credentials")
    }

    // Create token
    claims := Claims{
        UserID:   user.ID,
        Username: user.Username,
        Role:     user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString(api.secretKey)

    return tokenString, err
}
```

### Client-Side Token Storage

```javascript
// Login page
export default {
    title: "Login",

    async render() {
        let emailInput, passwordInput, errorDiv;

        const handleLogin = () => {
            asy(async () => {
                errorDiv.empty();

                let email = emailInput.val();
                let password = passwordInput.val();

                try {
                    // Call login API (doesn't require auth)
                    let token = await api.auth.login(email, password);

                    // Store token in localStorage
                    localStorage.setItem("webglue.headers.Authorization", `Bearer ${token}`);

                    // Redirect to home
                    goto("/");
                } catch (err) {
                    errorDiv.text(err.message);
                }
            });
        };

        return [
            DIV("errors", el => errorDiv = el),
            DIV("login-form", [
                TEXT(el => {
                    emailInput = el;
                    el.attr("placeholder", "Email");
                }),
                PASSWORD(el => {
                    passwordInput = el;
                    el.attr("placeholder", "Password");
                }),
                BUTTON().text("Login").click(handleLogin)
            ])
        ];
    }
}
```

## Session-Based Authentication

### Server Implementation

```go
type SessionApi struct {
    sessions map[string]*User // In production, use Redis
    mu       sync.RWMutex
}

func (api *SessionApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    if funcName == "Login" {
        return nil, nil
    }

    // Get session cookie
    cookie, err := req.Cookie("session_id")
    if err != nil {
        return nil, errors.New("not authenticated")
    }

    // Lookup session
    api.mu.RLock()
    user, exists := api.sessions[cookie.Value]
    api.mu.RUnlock()

    if !exists {
        return nil, errors.New("invalid session")
    }

    return []any{user}, nil
}

func (api *SessionApi) Login(w http.ResponseWriter, req *http.Request, email, password string) (string, error) {
    user, err := validateCredentials(email, password)
    if err != nil {
        return "", err
    }

    // Create session
    sessionID := generateSessionID()

    api.mu.Lock()
    api.sessions[sessionID] = user
    api.mu.Unlock()

    // Set cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    sessionID,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   86400, // 24 hours
    })

    return "success", nil
}

// Note: To access http.ResponseWriter, inject it via CheckCall
type SessionApi struct {
    // ...
}

func (api *SessionApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    // For Login, we need to access the response writer
    // This is handled differently - see parameter injection
    // ...
}
```

## Role-Based Access Control

### RBAC Implementation

```go
type Role string

const (
    RoleAdmin  Role = "admin"
    RoleEditor Role = "editor"
    RoleViewer Role = "viewer"
)

type User struct {
    ID       int
    Username string
    Role     Role
}

type RBACApi struct {
    db *Database
}

func (api *RBACApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    token := req.Header.Get("Authorization")
    user, err := validateToken(token)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    // Check permissions per function
    requiredRole := getRequiredRole(funcName)
    if !hasPermission(user.Role, requiredRole) {
        return nil, errors.New("forbidden")
    }

    return []any{user}, nil
}

func hasPermission(userRole, requiredRole Role) bool {
    roleLevel := map[Role]int{
        RoleViewer: 1,
        RoleEditor: 2,
        RoleAdmin:  3,
    }

    return roleLevel[userRole] >= roleLevel[requiredRole]
}

func getRequiredRole(funcName string) Role {
    permissions := map[string]Role{
        "GetUser":    RoleViewer,
        "UpdateUser": RoleEditor,
        "DeleteUser": RoleAdmin,
        "ListUsers":  RoleViewer,
    }

    role, exists := permissions[funcName]
    if !exists {
        return RoleViewer // Default
    }
    return role
}
```

### Declarative Permissions

```go
type PermissionConfig struct {
    Functions map[string][]Role
}

type PermissionApi struct {
    config PermissionConfig
}

func NewPermissionApi() *PermissionApi {
    return &PermissionApi{
        config: PermissionConfig{
            Functions: map[string][]Role{
                "GetUser":        {RoleViewer, RoleEditor, RoleAdmin},
                "UpdateUser":     {RoleEditor, RoleAdmin},
                "DeleteUser":     {RoleAdmin},
                "GetStatistics":  {RoleAdmin},
            },
        },
    }
}

func (api *PermissionApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    user, err := authenticate(req)
    if err != nil {
        return nil, err
    }

    allowedRoles, exists := api.config.Functions[funcName]
    if !exists {
        // Default: allow all authenticated users
        return []any{user}, nil
    }

    for _, role := range allowedRoles {
        if user.Role == role {
            return []any{user}, nil
        }
    }

    return nil, errors.New("forbidden")
}
```

## Protecting Specific Methods

### Method-Level Auth

```go
func (api *MyApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    // Public methods - no auth required
    publicMethods := []string{"Login", "Register", "GetPublicInfo"}
    for _, method := range publicMethods {
        if funcName == method {
            return nil, nil
        }
    }

    // All other methods require auth
    token := req.Header.Get("Authorization")
    user, err := validateToken(token)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    return []any{user}, nil
}
```

## Logout Implementation

### Server-Side

```go
func (api *AuthApi) Logout(user *User) error {
    // Invalidate token/session
    return api.db.InvalidateUserSessions(user.ID)
}
```

### Client-Side

```javascript
BUTTON().text("Logout").click(() => {
    asy(async () => {
        await api.auth.logout();

        // Remove token
        localStorage.removeItem("webglue.headers.Authorization");

        // Redirect to login
        goto("/login");
    });
})
```

## Checking Auth Status

### Protected Pages

```javascript
export default {
    async check(url, params) {
        // Try to get current user
        try {
            let user = await api.auth.getCurrentUser();
            // User is authenticated, continue
        } catch (err) {
            // Not authenticated, redirect to login
            return "/login";
        }
    },

    async render() {
        let user = await api.auth.getCurrentUser();

        return [
            H1().text(`Welcome, ${user.username}!`),
            // ... rest of page
        ];
    }
}
```

## Rate Limiting

### Request Rate Limiting

```go
type RateLimiter struct {
    requests map[string][]time.Time
    mu       sync.Mutex
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Allow(key string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-rl.window)

    // Clean old requests
    requests := rl.requests[key]
    validRequests := []time.Time{}
    for _, t := range requests {
        if t.After(cutoff) {
            validRequests = append(validRequests, t)
        }
    }

    // Check limit
    if len(validRequests) >= rl.limit {
        return false
    }

    // Add current request
    validRequests = append(validRequests, now)
    rl.requests[key] = validRequests

    return true
}

// Use in API
type RateLimitedApi struct {
    limiter *RateLimiter
}

func (api *RateLimitedApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    // Rate limit by IP
    ip := req.RemoteAddr

    if !api.limiter.Allow(ip) {
        return nil, errors.New("rate limit exceeded")
    }

    // Continue with auth...
    return nil, nil
}
```

## Security Best Practices

### Token Storage

**Recommended**:
```javascript
// Use localStorage for persistent tokens
localStorage.setItem("webglue.headers.Authorization", token);

// Use sessionStorage for session-only tokens
sessionStorage.setItem("webglue.headers.Authorization", token);
```

**Avoid**:
```javascript
// Don't store in cookies (CSRF risk without proper protection)
// Don't store in plain JavaScript variables (lost on refresh)
```

### Token Validation

```go
func (api *Api) CheckCall(req *http.Request, funcName string) ([]any, error) {
    token := extractToken(req)

    // Check token signature
    // Check expiration
    // Check revocation list
    // Verify audience/issuer

    user, err := validateToken(token)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    return []any{user}, nil
}
```

### Password Security

```go
import "golang.org/x/crypto/bcrypt"

func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func checkPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### HTTPS in Production

Always use HTTPS in production:

```go
// Production
http.ListenAndServeTLS(":443", "cert.pem", "key.pem", handler)

// Development
http.ListenAndServe(":8080", handler)
```

## Complete Example

See [examples.md](examples.md) for a complete authentication implementation with login, registration, and protected routes.

## Next Steps

- [Events Guide](events.md) - Secure event streams
- [Deployment](deployment.md) - Production security
- [Examples](examples.md) - Complete applications
