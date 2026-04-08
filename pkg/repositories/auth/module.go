package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"

	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger logger.ILogger
	Db     interfaces.Querier
}

type repo struct {
	logger logger.ILogger
	db     interfaces.Querier
}

func New(p Params) interfaces.AuthRepo {
	return &repo{
		logger: p.Logger,
		db:     p.Db,
	}
}

func (r *repo) FindUserByIdentifier(ctx context.Context, identifier string) (structs.User, error) {
	identifier = strings.TrimSpace(identifier)
	query := `SELECT
		u.id,
		u.company_id,
		u.username,
		u.phone,
		u.email,
		u.password_hash,
		u.first_name,
		u.last_name,
		u.status,
		u.is_super_admin,
		u.failed_login_count,
		u.locked_until,
		u.last_login_at,
		u.created_at,
		u.updated_at
	FROM users u
	WHERE lower(u.username) = lower($1)
		OR lower(COALESCE(u.email, '')) = lower($1)
		OR u.phone = $1
	LIMIT 1;`

	return r.scanUser(r.db.QueryRow(ctx, query, identifier))
}

func (r *repo) GetUserByID(ctx context.Context, userID int64) (structs.User, error) {
	query := `SELECT
		u.id,
		u.company_id,
		u.username,
		u.phone,
		u.email,
		u.password_hash,
		u.first_name,
		u.last_name,
		u.status,
		u.is_super_admin,
		u.failed_login_count,
		u.locked_until,
		u.last_login_at,
		u.created_at,
		u.updated_at
	FROM users u
	WHERE u.id = $1
	LIMIT 1;`

	return r.scanUser(r.db.QueryRow(ctx, query, userID))
}

func (r *repo) RegisterFailedLogin(ctx context.Context, userID int64, maxAttempts int, lockUntil time.Time) (int, *time.Time, error) {
	query := `UPDATE users
	SET failed_login_count = failed_login_count + 1,
		locked_until = CASE
			WHEN failed_login_count + 1 >= $2 THEN $3
			ELSE locked_until
		END,
		updated_at = NOW()
	WHERE id = $1
	RETURNING failed_login_count, locked_until;`

	var (
		failedCount int
		lockedUntil sql.NullTime
	)
	if err := r.db.QueryRow(ctx, query, userID, maxAttempts, lockUntil).Scan(&failedCount, &lockedUntil); err != nil {
		return 0, nil, err
	}

	if lockedUntil.Valid {
		lockedTime := lockedUntil.Time
		return failedCount, &lockedTime, nil
	}
	return failedCount, nil, nil
}

func (r *repo) UpdateLoginSuccess(ctx context.Context, userID int64, loginAt time.Time) error {
	query := `UPDATE users
	SET failed_login_count = 0,
		locked_until = NULL,
		last_login_at = $2,
		updated_at = $2
	WHERE id = $1;`

	_, err := r.db.Exec(ctx, query, userID, loginAt)
	return err
}

func (r *repo) CreateAuthSession(ctx context.Context, session *structs.AuthSession) error {
	query := `INSERT INTO auth_sessions (
		user_id,
		refresh_hash,
		expires_at,
		ip,
		user_agent,
		rotated_from_session_id
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_at;`

	var rotatedFrom interface{}
	if session.RotatedFromSessionID != nil {
		rotatedFrom = *session.RotatedFromSessionID
	}

	return r.db.QueryRow(ctx, query,
		session.UserID,
		session.RefreshHash,
		session.ExpiresAt,
		session.IP,
		session.UserAgent,
		rotatedFrom,
	).Scan(&session.ID, &session.CreatedAt)
}

func (r *repo) GetAuthSessionByHash(ctx context.Context, refreshHash string) (structs.AuthSession, error) {
	query := `SELECT
		s.id,
		s.user_id,
		s.refresh_hash,
		s.expires_at,
		s.ip,
		s.user_agent,
		s.revoked_at,
		s.rotated_from_session_id,
		s.created_at
	FROM auth_sessions s
	WHERE s.refresh_hash = $1
	LIMIT 1;`

	var (
		session              structs.AuthSession
		ip, userAgent        sql.NullString
		revokedAt, createdAt sql.NullTime
		rotatedFromSessionID sql.NullInt64
	)

	err := r.db.QueryRow(ctx, query, refreshHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshHash,
		&session.ExpiresAt,
		&ip,
		&userAgent,
		&revokedAt,
		&rotatedFromSessionID,
		&createdAt,
	)
	if err != nil {
		return structs.AuthSession{}, err
	}

	if ip.Valid {
		session.IP = ip.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	if revokedAt.Valid {
		revokedAtValue := revokedAt.Time
		session.RevokedAt = &revokedAtValue
	}
	if rotatedFromSessionID.Valid {
		rotated := rotatedFromSessionID.Int64
		session.RotatedFromSessionID = &rotated
	}
	if createdAt.Valid {
		session.CreatedAt = createdAt.Time
	}

	return session, nil
}

func (r *repo) RevokeAuthSessionByHash(ctx context.Context, refreshHash string, revokedAt time.Time) error {
	query := `UPDATE auth_sessions
	SET revoked_at = $2
	WHERE refresh_hash = $1
		AND revoked_at IS NULL;`

	_, err := r.db.Exec(ctx, query, refreshHash, revokedAt)
	return err
}

func (r *repo) RevokeAuthSessionByID(ctx context.Context, sessionID int64, revokedAt time.Time) error {
	query := `UPDATE auth_sessions
	SET revoked_at = $2
	WHERE id = $1
		AND revoked_at IS NULL;`

	_, err := r.db.Exec(ctx, query, sessionID, revokedAt)
	return err
}

func (r *repo) GetPermissionAssignments(ctx context.Context, userID int64) ([]structs.PermissionAssignment, error) {
	query := `SELECT
		p.code,
		ur.scope_type,
		ur.scope_id,
		ur.own_only
	FROM user_roles ur
	JOIN role_permissions rp ON rp.role_id = ur.role_id
	JOIN permissions p ON p.id = rp.permission_id
	WHERE ur.user_id = $1;`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]structs.PermissionAssignment, 0)
	for rows.Next() {
		var (
			entry   structs.PermissionAssignment
			scopeID sql.NullInt64
		)

		if err := rows.Scan(&entry.PermissionCode, &entry.ScopeType, &scopeID, &entry.OwnOnly); err != nil {
			return nil, err
		}
		if scopeID.Valid {
			id := scopeID.Int64
			entry.ScopeID = &id
		}
		result = append(result, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repo) GetUserScopes(ctx context.Context, userID int64) ([]structs.RoleScope, error) {
	query := `SELECT
		ur.scope_type,
		ur.scope_id,
		ur.own_only
	FROM user_roles ur
	WHERE ur.user_id = $1;`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]structs.RoleScope, 0)
	for rows.Next() {
		var (
			scope   structs.RoleScope
			scopeID sql.NullInt64
		)

		if err := rows.Scan(&scope.ScopeType, &scopeID, &scope.OwnOnly); err != nil {
			return nil, err
		}
		if scopeID.Valid {
			id := scopeID.Int64
			scope.ScopeID = &id
		}
		result = append(result, scope)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repo) CreateAuditLog(ctx context.Context, entry structs.AuditLog) error {
	query := `INSERT INTO audit_logs (
		user_id,
		action,
		module,
		resource,
		resource_id,
		meta,
		created_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7);`

	meta := entry.Meta
	if meta == nil {
		meta = map[string]interface{}{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	var userID interface{}
	if entry.UserID != nil {
		userID = *entry.UserID
	}

	var resourceID interface{}
	if entry.ResourceID != nil {
		resourceID = *entry.ResourceID
	}

	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err = r.db.Exec(ctx, query, userID, entry.Action, entry.Module, entry.Resource, resourceID, metaJSON, createdAt)
	return err
}

func (r *repo) scanUser(row pgx.Row) (structs.User, error) {
	var (
		user                     structs.User
		companyID                sql.NullInt64
		phone, email, lastName   sql.NullString
		lockedUntil, lastLoginAt sql.NullTime
	)

	err := row.Scan(
		&user.ID,
		&companyID,
		&user.UserName,
		&phone,
		&email,
		&user.PasswordHash,
		&user.FirstName,
		&lastName,
		&user.Status,
		&user.IsSuperAdmin,
		&user.FailedLoginCount,
		&lockedUntil,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return structs.User{}, err
	}

	if companyID.Valid {
		user.CompanyID = companyID.Int64
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if lockedUntil.Valid {
		locked := lockedUntil.Time
		user.LockedUntil = &locked
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}

	return user, nil
}


func (r *repo) GetTrustedDeviceByPendingTokenHash(ctx context.Context, pendingTokenHash string) (*structs.TrustedDevice, error) {
	query := `SELECT
		id,
		fingerprint_hash,
		browser_name,
		browser_version,
		os_name,
		os_version,
		device_name,
		pending_token_hash,
		pending_expires_at,
		verify_target,
		verify_code_hash,
		verify_expires_at,
		trusted_token_hash,
		trusted_at,
		last_seen_at,
		created_at,
		updated_at
	FROM trusted_devices
	WHERE pending_token_hash = $1
	LIMIT 1;`

	return r.scanTrustedDevice(r.db.QueryRow(ctx, query, pendingTokenHash))
}

func (r *repo) GetTrustedDeviceByTrustedTokenHash(ctx context.Context, trustedTokenHash string) (*structs.TrustedDevice, error) {
	query := `SELECT
		id,
		fingerprint_hash,
		browser_name,
		browser_version,
		os_name,
		os_version,
		device_name,
		pending_token_hash,
		pending_expires_at,
		verify_target,
		verify_code_hash,
		verify_expires_at,
		trusted_token_hash,
		trusted_at,
		last_seen_at,
		created_at,
		updated_at
	FROM trusted_devices
	WHERE trusted_token_hash = $1
	LIMIT 1;`

	return r.scanTrustedDevice(r.db.QueryRow(ctx, query, trustedTokenHash))
}

func (r *repo) UpsertTrustedDevicePending(ctx context.Context, device *structs.TrustedDevice) error {
	query := `INSERT INTO trusted_devices (
		fingerprint_hash,
		browser_name,
		browser_version,
		os_name,
		os_version,
		device_name,
		pending_token_hash,
		pending_expires_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	ON CONFLICT (fingerprint_hash) DO UPDATE SET
		browser_name = EXCLUDED.browser_name,
		browser_version = EXCLUDED.browser_version,
		os_name = EXCLUDED.os_name,
		os_version = EXCLUDED.os_version,
		device_name = EXCLUDED.device_name,
		pending_token_hash = EXCLUDED.pending_token_hash,
		pending_expires_at = EXCLUDED.pending_expires_at,
		verify_target = NULL,
		verify_code_hash = NULL,
		verify_expires_at = NULL,
		updated_at = NOW()
	RETURNING id, created_at, updated_at;`

	return r.db.QueryRow(ctx, query,
		device.FingerprintHash,
		device.BrowserName,
		device.BrowserVersion,
		device.OSName,
		device.OSVersion,
		device.DeviceName,
		device.PendingTokenHash,
		device.PendingExpiresAt,
	).Scan(&device.ID, &device.CreatedAt, &device.UpdatedAt)
}

func (r *repo) SetTrustedDeviceVerificationChallenge(ctx context.Context, deviceID int64, target string, codeHash string, expiresAt time.Time) error {
	query := `UPDATE trusted_devices
	SET verify_target = $2,
		verify_code_hash = $3,
		verify_expires_at = $4,
		updated_at = NOW()
	WHERE id = $1;`

	_, err := r.db.Exec(ctx, query, deviceID, target, codeHash, expiresAt)
	return err
}

func (r *repo) MarkTrustedDeviceVerified(ctx context.Context, deviceID int64, trustedTokenHash string, verifiedAt time.Time) error {
	query := `UPDATE trusted_devices
	SET trusted_token_hash = $2,
		trusted_at = $3,
		last_seen_at = $3,
		pending_token_hash = NULL,
		pending_expires_at = NULL,
		verify_target = NULL,
		verify_code_hash = NULL,
		verify_expires_at = NULL,
		updated_at = $3
	WHERE id = $1;`

	_, err := r.db.Exec(ctx, query, deviceID, trustedTokenHash, verifiedAt)
	return err
}

func (r *repo) TouchTrustedDevice(ctx context.Context, deviceID int64, seenAt time.Time) error {
	query := `UPDATE trusted_devices
	SET last_seen_at = $2,
		updated_at = $2
	WHERE id = $1;`

	_, err := r.db.Exec(ctx, query, deviceID, seenAt)
	return err
}

func (r *repo) scanTrustedDevice(row pgx.Row) (*structs.TrustedDevice, error) {
	var (
		device                               structs.TrustedDevice
		browserName, browserVersion          sql.NullString
		osName, osVersion, deviceName        sql.NullString
		pendingTokenHash, trustedTokenHash   sql.NullString
		verifyTarget, verifyCodeHash         sql.NullString
		pendingExpiresAt, verifyExpiresAt    sql.NullTime
		trustedAt                            sql.NullTime
		lastSeenAt                           sql.NullTime
	)

	err := row.Scan(
		&device.ID,
		&device.FingerprintHash,
		&browserName,
		&browserVersion,
		&osName,
		&osVersion,
		&deviceName,
		&pendingTokenHash,
		&pendingExpiresAt,
		&verifyTarget,
		&verifyCodeHash,
		&verifyExpiresAt,
		&trustedTokenHash,
		&trustedAt,
		&lastSeenAt,
		&device.CreatedAt,
		&device.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if browserName.Valid {
		device.BrowserName = browserName.String
	}
	if browserVersion.Valid {
		device.BrowserVersion = browserVersion.String
	}
	if osName.Valid {
		device.OSName = osName.String
	}
	if osVersion.Valid {
		device.OSVersion = osVersion.String
	}
	if deviceName.Valid {
		device.DeviceName = deviceName.String
	}
	if pendingTokenHash.Valid {
		device.PendingTokenHash = pendingTokenHash.String
	}
	if verifyTarget.Valid {
		device.VerifyTarget = verifyTarget.String
	}
	if verifyCodeHash.Valid {
		device.VerifyCodeHash = verifyCodeHash.String
	}
	if trustedTokenHash.Valid {
		device.TrustedTokenHash = trustedTokenHash.String
	}
	if pendingExpiresAt.Valid {
		value := pendingExpiresAt.Time
		device.PendingExpiresAt = &value
	}
	if trustedAt.Valid {
		value := trustedAt.Time
		device.TrustedAt = &value
	}
	if verifyExpiresAt.Valid {
		value := verifyExpiresAt.Time
		device.VerifyExpiresAt = &value
	}
	if lastSeenAt.Valid {
		value := lastSeenAt.Time
		device.LastSeenAt = &value
	}

	return &device, nil
}

