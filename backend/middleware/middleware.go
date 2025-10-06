package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/auth"
	"github.com/dbehnke/allstar-nexus/backend/repository"
)

type key int

const userKey key = 0

// simpleTokenBucket holds state for a single IP.
type simpleTokenBucket struct {
	tokens     int
	lastRefill time.Time
}

// RateLimiter middleware (fixed window token bucket per IP, refill each minute) for low volume prototypes.
func RateLimiter(maxPerMinute int) func(http.Handler) http.Handler {
	if maxPerMinute <= 0 {
		maxPerMinute = 60
	}
	buckets := make(map[string]*simpleTokenBucket)
	mu := sync.Mutex{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()
			mu.Lock()
			b, ok := buckets[ip]
			if !ok {
				b = &simpleTokenBucket{tokens: maxPerMinute, lastRefill: now}
				buckets[ip] = b
			}
			// Refill per 60s window
			if now.Sub(b.lastRefill) >= time.Minute {
				b.tokens = maxPerMinute
				b.lastRefill = now
			}
			if b.tokens <= 0 {
				mu.Unlock()
				w.Header().Set("Retry-After", "60")
				writeJSONError(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
				return
			}
			b.tokens--
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	// honor X-Forwarded-For first value if present (simple parse)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// UserFromContext extracts SafeUser from context.
func UserFromContext(ctx context.Context) (*repository.SafeUser, bool) {
	u, ok := ctx.Value(userKey).(*repository.SafeUser)
	return u, ok
}

// Auth returns middleware that validates bearer token and loads user.
func Auth(secret string, userLoader func(email string) (*repository.SafeUser, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			tok := strings.TrimPrefix(authz, "Bearer ")
			email, role, exp, err := auth.ParseJWT(tok, secret)
			if err != nil || time.Now().After(exp) {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			su, err := userLoader(email)
			if err != nil {
				writeJSONError(w, 500, "server_error", "failed to load user")
				return
			}
			if su == nil || su.Role != role {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "user not found or role mismatch")
				return
			}
			ctx := context.WithValue(r.Context(), userKey, su)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole ensures user has one of required roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
				return
			}
			if _, ok := allowed[u.Role]; !ok {
				writeJSONError(w, http.StatusForbidden, "forbidden", "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// writeJSONError returns standardized error envelope.
func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	type errEnv struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	var env errEnv
	env.OK = false
	env.Error.Code = code
	env.Error.Message = msg
	_ = json.NewEncoder(w).Encode(&env)
}
