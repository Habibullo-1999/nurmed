package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	intauth "nurmed/internal/auth"
	"nurmed/internal/structs"
	"nurmed/pkg/config"
	"nurmed/pkg/logger"
)

const (
	ContextPrincipalKey = "auth.principal"
	ContextUserIDKey    = "auth.user_id"
	ContextScopeKey     = "auth.request_scope"
)

type Manager struct {
	authSvs      intauth.Service
	logger       logger.ILogger
	loginLimiter *requestLimiter
}

func New(authSvs intauth.Service, lg logger.ILogger, cfg config.Config) *Manager {
	loginLimit := cfg.GetInt("auth.loginRateLimitPerMinute")
	if loginLimit <= 0 {
		loginLimit = 20
	}

	return &Manager{
		authSvs:      authSvs,
		logger:       lg,
		loginLimiter: newRequestLimiter(loginLimit, time.Minute),
	}
}

func (m *Manager) LoginRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.loginLimiter.Allow(c.ClientIP()) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusTooManyRequests, structs.Response{
			Code:    http.StatusTooManyRequests,
			Message: "Too many login attempts. Try again later.",
		})
	}
}

func (m *Manager) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, structs.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		token := bearerToken(authorization)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, structs.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		principal, err := m.authSvs.VerifyAccessToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, structs.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		c.Set(ContextPrincipalKey, principal)
		c.Set(ContextUserIDKey, principal.UserID)
		c.Next()
	}
}

func (m *Manager) ScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, structs.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		scope := parseRequestScope(c)

		if !principal.IsSuperAdmin {
			if scope.CompanyID == 0 {
				if singleCompanyID, hasSingleCompany := singleScopeID(principal.RoleScopes, "company"); hasSingleCompany {
					scope.CompanyID = singleCompanyID
				}
			}

			if !isScopeAllowed(principal.RoleScopes, scope) {
				c.AbortWithStatusJSON(http.StatusForbidden, structs.Response{
					Code:    http.StatusForbidden,
					Message: "Forbidden",
				})
				return
			}
		}

		c.Set(ContextScopeKey, scope)
		c.Next()
	}
}

func (m *Manager) PermissionMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, structs.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		scope := GetRequestScope(c)
		if !m.authSvs.IsAllowed(principal, permission, scope) {
			c.AbortWithStatusJSON(http.StatusForbidden, structs.Response{
				Code:    http.StatusForbidden,
				Message: "Forbidden",
			})
			return
		}

		c.Next()
	}
}

func GetPrincipal(c *gin.Context) (structs.UserPrincipal, bool) {
	raw, ok := c.Get(ContextPrincipalKey)
	if !ok {
		return structs.UserPrincipal{}, false
	}
	principal, ok := raw.(structs.UserPrincipal)
	return principal, ok
}

func GetRequestScope(c *gin.Context) structs.RequestScope {
	raw, ok := c.Get(ContextScopeKey)
	if !ok {
		return structs.RequestScope{}
	}
	scope, ok := raw.(structs.RequestScope)
	if !ok {
		return structs.RequestScope{}
	}
	return scope
}

func parseRequestScope(c *gin.Context) structs.RequestScope {
	return structs.RequestScope{
		CompanyID:   readScopeID(c, "company_id", "X-Company-ID"),
		BranchID:    readScopeID(c, "branch_id", "X-Branch-ID"),
		WarehouseID: readScopeID(c, "warehouse_id", "X-Warehouse-ID"),
		OwnerUserID: readScopeID(c, "owner_user_id", "X-Owner-User-ID"),
	}
}

func readScopeID(c *gin.Context, queryKey, headerKey string) int64 {
	if value := strings.TrimSpace(c.Query(queryKey)); value != "" {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil && id > 0 {
			return id
		}
	}

	if value := strings.TrimSpace(c.GetHeader(headerKey)); value != "" {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil && id > 0 {
			return id
		}
	}

	return 0
}

func singleScopeID(scopes []structs.RoleScope, scopeType string) (int64, bool) {
	ids := make([]int64, 0)
	seen := map[int64]struct{}{}

	for _, scope := range scopes {
		if scope.ScopeType != scopeType || scope.ScopeID == nil {
			continue
		}
		if _, exists := seen[*scope.ScopeID]; exists {
			continue
		}
		seen[*scope.ScopeID] = struct{}{}
		ids = append(ids, *scope.ScopeID)
	}

	if len(ids) == 1 {
		return ids[0], true
	}
	return 0, false
}

func isScopeAllowed(scopes []structs.RoleScope, requestScope structs.RequestScope) bool {
	if requestScope.CompanyID == 0 && requestScope.BranchID == 0 && requestScope.WarehouseID == 0 && requestScope.OwnerUserID == 0 {
		return true
	}

	hasGlobal := false
	companies := map[int64]struct{}{}
	branches := map[int64]struct{}{}
	warehouses := map[int64]struct{}{}

	for _, scope := range scopes {
		switch scope.ScopeType {
		case "global":
			hasGlobal = true
		case "company":
			if scope.ScopeID != nil {
				companies[*scope.ScopeID] = struct{}{}
			}
		case "branch":
			if scope.ScopeID != nil {
				branches[*scope.ScopeID] = struct{}{}
			}
		case "warehouse":
			if scope.ScopeID != nil {
				warehouses[*scope.ScopeID] = struct{}{}
			}
		}
	}

	if hasGlobal {
		return true
	}

	if requestScope.CompanyID > 0 {
		if _, ok := companies[requestScope.CompanyID]; !ok {
			return false
		}
	}
	if requestScope.BranchID > 0 {
		if _, ok := branches[requestScope.BranchID]; !ok {
			return false
		}
	}
	if requestScope.WarehouseID > 0 {
		if _, ok := warehouses[requestScope.WarehouseID]; !ok {
			return false
		}
	}

	return true
}

func bearerToken(authorization string) string {
	parts := strings.SplitN(strings.TrimSpace(authorization), " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

type requestLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	buckets map[string]limitBucket
}

type limitBucket struct {
	Count     int
	WindowEnd time.Time
}

func newRequestLimiter(limit int, window time.Duration) *requestLimiter {
	return &requestLimiter{
		window:  window,
		limit:   limit,
		buckets: make(map[string]limitBucket),
	}
}

func (r *requestLimiter) Allow(key string) bool {
	if key == "" {
		key = "unknown"
	}

	now := time.Now().UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, exists := r.buckets[key]
	if !exists || now.After(bucket.WindowEnd) {
		r.buckets[key] = limitBucket{
			Count:     1,
			WindowEnd: now.Add(r.window),
		}
		return true
	}

	if bucket.Count >= r.limit {
		return false
	}

	bucket.Count++
	r.buckets[key] = bucket
	return true
}
