package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/config"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrUserLocked         = errors.New("user locked")
)

type Params struct {
	fx.In
	Logger   logger.ILogger
	Config   config.Config
	AuthRepo interfaces.AuthRepo
}

type Service interface {
	HashPassword(password string) (string, error)
	Login(ctx context.Context, request structs.LoginRequest, meta structs.AuthMeta) (structs.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string, meta structs.AuthMeta) (structs.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string, userID *int64) error
	VerifyAccessToken(ctx context.Context, accessToken string) (structs.UserPrincipal, error)
	IsAllowed(principal structs.UserPrincipal, permission string, scope structs.RequestScope) bool
	RefreshCookieName() string
	RefreshCookieSecure() bool
	RefreshCookieDomain() string
	RefreshTokenTTL() time.Duration
}

type service struct {
	logger              logger.ILogger
	repo                interfaces.AuthRepo
	jwtSecret           []byte
	accessTTL           time.Duration
	refreshTTL          time.Duration
	maxFailedAttempts   int
	lockDuration        time.Duration
	refreshCookieName   string
	refreshCookieSecure bool
	refreshCookieDomain string
}

func New(p Params) Service {
	accessTTLMinutes := p.Config.GetInt("auth.accessTtlMinutes")
	if accessTTLMinutes <= 0 {
		accessTTLMinutes = 15
	}

	refreshTTLDays := p.Config.GetInt("auth.refreshTtlDays")
	if refreshTTLDays <= 0 {
		refreshTTLDays = 30
	}

	maxFailedAttempts := p.Config.GetInt("auth.maxFailedAttempts")
	if maxFailedAttempts <= 0 {
		maxFailedAttempts = 5
	}

	lockDurationMinutes := p.Config.GetInt("auth.lockDurationMinutes")
	if lockDurationMinutes <= 0 {
		lockDurationMinutes = 15
	}

	jwtSecret := p.Config.GetString("auth.jwtSecret")
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production-at-least-32-bytes"
		p.Logger.Warning(nil, "auth.jwtSecret is empty, using development fallback")
	}

	refreshCookieName := p.Config.GetString("auth.refreshCookieName")
	if refreshCookieName == "" {
		refreshCookieName = "refresh_token"
	}

	refreshCookieSecure := true
	if raw := p.Config.Get("auth.secureCookies"); raw != nil {
		refreshCookieSecure = p.Config.GetBool("auth.secureCookies")
	}

	return &service{
		logger:              p.Logger,
		repo:                p.AuthRepo,
		jwtSecret:           []byte(jwtSecret),
		accessTTL:           time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL:          time.Duration(refreshTTLDays) * 24 * time.Hour,
		maxFailedAttempts:   maxFailedAttempts,
		lockDuration:        time.Duration(lockDurationMinutes) * time.Minute,
		refreshCookieName:   refreshCookieName,
		refreshCookieSecure: refreshCookieSecure,
		refreshCookieDomain: p.Config.GetString("auth.cookieDomain"),
	}
}

func (s *service) HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *service) RefreshCookieName() string {
	return s.refreshCookieName
}

func (s *service) RefreshCookieSecure() bool {
	return s.refreshCookieSecure
}

func (s *service) RefreshCookieDomain() string {
	return s.refreshCookieDomain
}

func (s *service) RefreshTokenTTL() time.Duration {
	return s.refreshTTL
}

func (s *service) Login(ctx context.Context, request structs.LoginRequest, meta structs.AuthMeta) (structs.AuthTokens, error) {
	identifier := strings.TrimSpace(request.Identifier)
	if identifier == "" || request.Password == "" {
		return structs.AuthTokens{}, ErrInvalidCredentials
	}

	user, err := s.repo.FindUserByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.audit(ctx, structs.AuditLog{
				Action:   "auth.login_failed",
				Module:   "auth",
				Resource: "session",
				Meta: map[string]interface{}{
					"identifier": identifier,
					"reason":     "user_not_found",
					"ip":         meta.IP,
				},
				CreatedAt: time.Now().UTC(),
			})
			return structs.AuthTokens{}, ErrInvalidCredentials
		}
		return structs.AuthTokens{}, err
	}

	now := time.Now().UTC()

	if user.Status != structs.UserStatusActive {
		s.audit(ctx, structs.AuditLog{
			UserID:   &user.ID,
			Action:   "auth.login_failed",
			Module:   "auth",
			Resource: "session",
			Meta: map[string]interface{}{
				"reason": "user_inactive",
				"status": user.Status,
				"ip":     meta.IP,
			},
			CreatedAt: now,
		})
		return structs.AuthTokens{}, ErrUnauthorized
	}

	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		s.audit(ctx, structs.AuditLog{
			UserID:   &user.ID,
			Action:   "auth.login_blocked",
			Module:   "auth",
			Resource: "session",
			Meta: map[string]interface{}{
				"reason":       "account_locked",
				"locked_until": user.LockedUntil.Format(time.RFC3339),
				"ip":           meta.IP,
			},
			CreatedAt: now,
		})
		return structs.AuthTokens{}, ErrUserLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		lockUntil := now.Add(s.lockDuration)
		failedCount, newLockedUntil, repoErr := s.repo.RegisterFailedLogin(ctx, user.ID, s.maxFailedAttempts, lockUntil)
		if repoErr != nil {
			return structs.AuthTokens{}, repoErr
		}

		metaPayload := map[string]interface{}{
			"reason":       "invalid_password",
			"failed_count": failedCount,
			"ip":           meta.IP,
		}
		if newLockedUntil != nil {
			metaPayload["locked_until"] = newLockedUntil.Format(time.RFC3339)
		}
		s.audit(ctx, structs.AuditLog{
			UserID:    &user.ID,
			Action:    "auth.login_failed",
			Module:    "auth",
			Resource:  "session",
			Meta:      metaPayload,
			CreatedAt: now,
		})

		if newLockedUntil != nil && newLockedUntil.After(now) {
			return structs.AuthTokens{}, ErrUserLocked
		}
		return structs.AuthTokens{}, ErrInvalidCredentials
	}

	if err := s.repo.UpdateLoginSuccess(ctx, user.ID, now); err != nil {
		return structs.AuthTokens{}, err
	}

	tokens, err := s.issueTokens(ctx, user, meta, nil)
	if err != nil {
		return structs.AuthTokens{}, err
	}

	s.audit(ctx, structs.AuditLog{
		UserID:   &user.ID,
		Action:   "auth.login_success",
		Module:   "auth",
		Resource: "session",
		Meta: map[string]interface{}{
			"ip":         meta.IP,
			"user_agent": meta.UserAgent,
		},
		CreatedAt: now,
	})

	return tokens, nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string, meta structs.AuthMeta) (structs.AuthTokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return structs.AuthTokens{}, ErrUnauthorized
	}

	hash := hashToken(refreshToken)
	session, err := s.repo.GetAuthSessionByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return structs.AuthTokens{}, ErrUnauthorized
		}
		return structs.AuthTokens{}, err
	}

	now := time.Now().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return structs.AuthTokens{}, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return structs.AuthTokens{}, ErrUnauthorized
		}
		return structs.AuthTokens{}, err
	}

	if user.Status != structs.UserStatusActive {
		return structs.AuthTokens{}, ErrUnauthorized
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		return structs.AuthTokens{}, ErrUserLocked
	}

	if err := s.repo.RevokeAuthSessionByID(ctx, session.ID, now); err != nil {
		return structs.AuthTokens{}, err
	}

	tokens, err := s.issueTokens(ctx, user, meta, &session.ID)
	if err != nil {
		return structs.AuthTokens{}, err
	}

	s.audit(ctx, structs.AuditLog{
		UserID:   &user.ID,
		Action:   "auth.refresh_success",
		Module:   "auth",
		Resource: "session",
		Meta: map[string]interface{}{
			"ip":         meta.IP,
			"user_agent": meta.UserAgent,
		},
		CreatedAt: now,
	})

	return tokens, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string, userID *int64) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}

	now := time.Now().UTC()
	hash := hashToken(refreshToken)
	if err := s.repo.RevokeAuthSessionByHash(ctx, hash, now); err != nil {
		return err
	}

	s.audit(ctx, structs.AuditLog{
		UserID:    userID,
		Action:    "auth.logout",
		Module:    "auth",
		Resource:  "session",
		Meta:      map[string]interface{}{},
		CreatedAt: now,
	})

	return nil
}

func (s *service) VerifyAccessToken(ctx context.Context, accessToken string) (structs.UserPrincipal, error) {
	claims, err := VerifyHS256Token(accessToken, s.jwtSecret, time.Now().UTC())
	if err != nil {
		return structs.UserPrincipal{}, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, claims.Sub)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return structs.UserPrincipal{}, ErrUnauthorized
		}
		return structs.UserPrincipal{}, err
	}

	now := time.Now().UTC()
	if user.Status != structs.UserStatusActive {
		return structs.UserPrincipal{}, ErrUnauthorized
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		return structs.UserPrincipal{}, ErrUnauthorized
	}

	permissions, err := s.repo.GetPermissionAssignments(ctx, user.ID)
	if err != nil {
		return structs.UserPrincipal{}, err
	}

	scopes, err := s.repo.GetUserScopes(ctx, user.ID)
	if err != nil {
		return structs.UserPrincipal{}, err
	}

	return structs.UserPrincipal{
		UserID:                user.ID,
		UserName:              user.UserName,
		CompanyID:             user.CompanyID,
		IsSuperAdmin:          user.IsSuperAdmin,
		PermissionAssignments: permissions,
		RoleScopes:            scopes,
	}, nil
}

func (s *service) IsAllowed(principal structs.UserPrincipal, permission string, scope structs.RequestScope) bool {
	if principal.IsSuperAdmin {
		return true
	}

	hasPermission := false
	for _, assignment := range principal.PermissionAssignments {
		if assignment.PermissionCode != permission {
			continue
		}
		hasPermission = true

		if assignment.OwnOnly && scope.OwnerUserID != 0 && scope.OwnerUserID != principal.UserID {
			continue
		}

		if scope.CompanyID == 0 && scope.BranchID == 0 && scope.WarehouseID == 0 {
			return true
		}

		if matchesScope(assignment, scope) {
			return true
		}
	}

	if !hasPermission {
		s.logger.Warning(nil, fmt.Sprintf("permission denied: user=%d, permission=%s", principal.UserID, permission))
	}

	return false
}

func (s *service) issueTokens(ctx context.Context, user structs.User, meta structs.AuthMeta, rotatedFrom *int64) (structs.AuthTokens, error) {
	now := time.Now().UTC()

	jti, err := RandomToken(16)
	if err != nil {
		return structs.AuthTokens{}, err
	}

	claims := structs.AccessClaims{
		Sub:          user.ID,
		CompanyID:    user.CompanyID,
		IsSuperAdmin: user.IsSuperAdmin,
		JTI:          jti,
		Exp:          now.Add(s.accessTTL).Unix(),
		Iat:          now.Unix(),
	}

	accessToken, err := SignHS256Token(claims, s.jwtSecret)
	if err != nil {
		return structs.AuthTokens{}, err
	}

	refreshToken, err := RandomToken(32)
	if err != nil {
		return structs.AuthTokens{}, err
	}

	session := structs.AuthSession{
		UserID:               user.ID,
		RefreshHash:          hashToken(refreshToken),
		ExpiresAt:            now.Add(s.refreshTTL),
		IP:                   strings.TrimSpace(meta.IP),
		UserAgent:            strings.TrimSpace(meta.UserAgent),
		RotatedFromSessionID: rotatedFrom,
	}

	if err := s.repo.CreateAuthSession(ctx, &session); err != nil {
		return structs.AuthTokens{}, err
	}

	return structs.AuthTokens{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  now.Add(s.accessTTL),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: now.Add(s.refreshTTL),
		UserID:                user.ID,
		UserName:              user.UserName,
	}, nil
}

func (s *service) audit(ctx context.Context, entry structs.AuditLog) {
	if entry.Meta == nil {
		entry.Meta = map[string]interface{}{}
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if err := s.repo.CreateAuditLog(ctx, entry); err != nil {
		s.logger.Warning(ctx, "failed to write audit log", zap.Error(err))
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func matchesScope(assignment structs.PermissionAssignment, scope structs.RequestScope) bool {
	if assignment.ScopeType == "global" {
		return true
	}

	if assignment.ScopeID == nil {
		return false
	}

	switch assignment.ScopeType {
	case "company":
		if scope.CompanyID == 0 {
			return true
		}
		return *assignment.ScopeID == scope.CompanyID
	case "branch":
		if scope.BranchID == 0 {
			return true
		}
		return *assignment.ScopeID == scope.BranchID
	case "warehouse":
		if scope.WarehouseID == 0 {
			return true
		}
		return *assignment.ScopeID == scope.WarehouseID
	default:
		return false
	}
}
