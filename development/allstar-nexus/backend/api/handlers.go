package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/auth"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"github.com/dbehnke/allstar-nexus/internal/ami"
)

type API struct {
	Users        *repository.UserRepo
	Secret       string
	TTL          time.Duration
	LinkStats    *repository.LinkStatsRepo
	AMIConnector *ami.Connector
	AstDBPath    string
}

func New(db *sql.DB, secret string, ttl time.Duration) *API {
	return &API{
		Users:        repository.NewUserRepo(db),
		LinkStats:    repository.NewLinkStatsRepo(db),
		Secret:       secret,
		TTL:          ttl,
		AMIConnector: nil,
		AstDBPath:    "",
	}
}

// SetAMIConnector sets the AMI connector for the API (called after initialization)
func (a *API) SetAMIConnector(conn *ami.Connector) {
	a.AMIConnector = conn
}

// SetAstDBPath sets the path to the astdb.txt file
func (a *API) SetAstDBPath(path string) {
	a.AstDBPath = path
}

func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role,omitempty"` // only honored if first user
	}
	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}
	// Normalize email
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	if body.Email == "" || body.Password == "" {
		writeError(w, 400, "validation_error", "email and password required")
		return
	}
	if err := validatePassword(body.Password); err != nil {
		writeError(w, 400, "password_invalid", err.Error())
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeError(w, 500, "hash_error", "unable to hash password")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	count, err := a.Users.Count(ctx)
	if err != nil {
		writeError(w, 500, "db_error", "unable to count users")
		return
	}
	role := models.RoleUser
	if count == 0 { // bootstrap: first user becomes superadmin
		role = models.RoleSuperAdmin
	} else if body.Role != "" && count == 1 && body.Role == models.RoleAdmin { // optional second admin early
		role = models.RoleAdmin
	}
	u, err := a.Users.Create(ctx, body.Email, hash, role)
	if err != nil {
		// Detect unique email violation (SQLite typical message)
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "unique") && strings.Contains(msg, "users.email") {
			writeError(w, 409, "duplicate_email", "email already registered")
			return
		}
		writeError(w, 500, "db_error", "could not create user")
		return
	}
	writeJSON(w, 201, u)
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	u, err := a.Users.GetByEmail(ctx, body.Email)
	if err != nil || u == nil || !auth.CheckPassword(u.PasswordHash, body.Password) {
		writeError(w, 401, "invalid_credentials", "invalid email or password")
		return
	}
	// Use configured TTL
	token, err := auth.GenerateJWT(u.Email, u.Role, a.TTL, a.Secret)
	if err != nil {
		writeError(w, 500, "token_error", "failed to issue token")
		return
	}
	writeJSON(w, 200, map[string]any{"token": token, "role": u.Role})
}

// Me returns current user info based on placeholder auth header.
func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	// Expect Authorization: Bearer <token>
	u, status := a.currentUser(r)
	if status != 200 {
		writeError(w, status, "unauthorized", http.StatusText(status))
		return
	}
	writeJSON(w, 200, map[string]any{"email": u.Email, "role": u.Role, "id": u.ID})
}

// AdminSummary returns aggregate info (requires admin or superadmin)
func (a *API) AdminSummary(w http.ResponseWriter, r *http.Request) {
	u, status := a.currentUser(r)
	if status != 200 {
		writeError(w, status, "unauthorized", http.StatusText(status))
		return
	}
	if u.Role != models.RoleAdmin && u.Role != models.RoleSuperAdmin {
		writeError(w, 403, "forbidden", "insufficient role")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	counts, err := a.Users.RoleCounts(ctx)
	if err != nil {
		writeError(w, 500, "db_error", "failed to compute role counts")
		return
	}
	writeJSON(w, 200, map[string]any{"roles": counts})
}

// LinkStats returns all persisted per-link tx stats (auth required; can be public if desired)
func (a *API) LinkStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	stats, err := a.LinkStats.GetAll(ctx)
	if err != nil {
		writeError(w, 500, "db_error", "failed to load link stats")
		return
	}
	q := r.URL.Query()
	// since can be RFC3339 or relative like -1h, -15m, -30s
	if sinceStr := q.Get("since"); sinceStr != "" {
		var ref time.Time
		if strings.HasPrefix(sinceStr, "-") { // relative
			if d, perr := time.ParseDuration(strings.TrimPrefix(sinceStr, "-")); perr == nil {
				ref = time.Now().Add(-d)
			}
		} else if t, perr := time.Parse(time.RFC3339, sinceStr); perr == nil {
			ref = t
		}
		if !ref.IsZero() {
			filtered := make([]repository.LinkStat, 0, len(stats))
			for _, s := range stats {
				if !s.UpdatedAt.Before(ref) {
					filtered = append(filtered, s)
				}
			}
			stats = filtered
		}
	}
	// node=123,456 filter
	if nodesStr := q.Get("node"); nodesStr != "" {
		wanted := map[int]struct{}{}
		for _, part := range strings.Split(nodesStr, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if n, err := strconv.Atoi(part); err == nil {
				wanted[n] = struct{}{}
			}
		}
		if len(wanted) > 0 {
			filtered := make([]repository.LinkStat, 0, len(stats))
			for _, s := range stats {
				if _, ok := wanted[s.Node]; ok {
					filtered = append(filtered, s)
				}
			}
			stats = filtered
		}
	}
	// sort parameter
	switch q.Get("sort") {
	case "tx_seconds_desc":
		sort.Slice(stats, func(i, j int) bool { return stats[i].TotalTxSeconds > stats[j].TotalTxSeconds })
	case "tx_seconds_asc":
		sort.Slice(stats, func(i, j int) bool { return stats[i].TotalTxSeconds < stats[j].TotalTxSeconds })
	case "node_asc":
		sort.Slice(stats, func(i, j int) bool { return stats[i].Node < stats[j].Node })
	case "node_desc":
		sort.Slice(stats, func(i, j int) bool { return stats[i].Node > stats[j].Node })
	case "recent_desc":
		sort.Slice(stats, func(i, j int) bool { return stats[i].UpdatedAt.After(stats[j].UpdatedAt) })
	}
	// limit
	if limStr := q.Get("limit"); limStr != "" {
		if lim, err := strconv.Atoi(limStr); err == nil && lim > 0 && lim < len(stats) {
			stats = stats[:lim]
		}
	}
	writeJSON(w, 200, map[string]any{"stats": stats, "generated_at": time.Now().UTC()})
}

// TopLinkStatsHandler returns top N links by total_tx_seconds (default) or by tx rate (requires connected_since)
// Query: /api/link-stats/top?limit=N&mode=tx_seconds|tx_rate
func (a *API) TopLinkStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	stats, err := a.LinkStats.GetAll(ctx)
	if err != nil {
		writeError(w, 500, "db_error", "failed to load link stats")
		return
	}
	q := r.URL.Query()
	mode := q.Get("mode")
	if mode == "" {
		mode = "tx_seconds"
	}
	// compute rate if requested
	type row struct {
		repository.LinkStat
		Rate float64
	}
	rows := make([]row, 0, len(stats))
	now := time.Now()
	for _, s := range stats {
		rw := row{LinkStat: s}
		if mode == "tx_rate" && s.ConnectedSince != nil {
			dur := now.Sub(*s.ConnectedSince).Seconds()
			if dur > 0 {
				rw.Rate = float64(s.TotalTxSeconds) / dur
			}
		}
		rows = append(rows, rw)
	}
	switch mode {
	case "tx_rate":
		sort.Slice(rows, func(i, j int) bool { return rows[i].Rate > rows[j].Rate })
	default: // tx_seconds
		sort.Slice(rows, func(i, j int) bool { return rows[i].TotalTxSeconds > rows[j].TotalTxSeconds })
	}
	limit := 10
	if limStr := q.Get("limit"); limStr != "" {
		if lim, err := strconv.Atoi(limStr); err == nil && lim > 0 {
			limit = lim
		}
	}
	if limit > len(rows) {
		limit = len(rows)
	}
	out := make([]any, 0, limit)
	for i := 0; i < limit; i++ {
		r := rows[i]
		if mode == "tx_rate" {
			out = append(out, map[string]any{"node": r.Node, "total_tx_seconds": r.TotalTxSeconds, "rate": r.Rate, "connected_since": r.ConnectedSince, "updated_at": r.UpdatedAt})
		} else {
			out = append(out, map[string]any{"node": r.Node, "total_tx_seconds": r.TotalTxSeconds, "connected_since": r.ConnectedSince, "updated_at": r.UpdatedAt})
		}
	}
	writeJSON(w, 200, map[string]any{"mode": mode, "limit": limit, "results": out, "generated_at": time.Now().UTC()})
}

// helper: parse bearer JWT and load user
func (a *API) currentUser(r *http.Request) (*repository.SafeUser, int) {
	authz := r.Header.Get("Authorization")
	if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
		return nil, 401
	}
	tok := strings.TrimPrefix(authz, "Bearer ")
	email, role, exp, err := auth.ParseJWT(tok, a.Secret)
	if err != nil || time.Now().After(exp) {
		return nil, 401
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	usr, err := a.Users.GetByEmail(ctx, email)
	if err != nil {
		return nil, 500
	}
	if usr == nil {
		return nil, 404
	}
	if usr.Role != role {
		return nil, 401
	} // role mismatch (token stale?)
	return &repository.SafeUser{ID: usr.ID, Email: usr.Email, Role: usr.Role}, 200
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// DashboardSummary public minimal placeholder.
func (a *API) DashboardSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	counts, err := a.Users.RoleCounts(ctx)
	if err != nil {
		writeError(w, 500, "db_error", "failed to compute role counts")
		return
	}
	total, err := a.Users.Count(ctx)
	if err != nil {
		writeError(w, 500, "db_error", "failed to count users")
		return
	}
	new24, err := a.Users.NewUsersSince(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		writeError(w, 500, "db_error", "failed to compute recent users")
		return
	}
	writeJSON(w, 200, map[string]any{
		"roles":        counts,
		"total_users":  total,
		"new_last_24h": new24,
		"generated_at": time.Now().UTC(),
	})
}
