package interfaces

import (
	"context"
	"time"

	"nurmed/internal/structs"
)

type AuthRepo interface {
	FindUserByIdentifier(ctx context.Context, identifier string) (structs.User, error)
	GetUserByID(ctx context.Context, userID int64) (structs.User, error)
	RegisterFailedLogin(ctx context.Context, userID int64, maxAttempts int, lockUntil time.Time) (int, *time.Time, error)
	UpdateLoginSuccess(ctx context.Context, userID int64, loginAt time.Time) error
	CreateAuthSession(ctx context.Context, session *structs.AuthSession) error
	GetAuthSessionByHash(ctx context.Context, refreshHash string) (structs.AuthSession, error)
	RevokeAuthSessionByHash(ctx context.Context, refreshHash string, revokedAt time.Time) error
	RevokeAuthSessionByID(ctx context.Context, sessionID int64, revokedAt time.Time) error
	GetPermissionAssignments(ctx context.Context, userID int64) ([]structs.PermissionAssignment, error)
	GetUserScopes(ctx context.Context, userID int64) ([]structs.RoleScope, error)
	CreateAuditLog(ctx context.Context, entry structs.AuditLog) error
	GetTrustedDeviceByPendingTokenHash(ctx context.Context, pendingTokenHash string) (*structs.TrustedDevice, error)
	GetTrustedDeviceByTrustedTokenHash(ctx context.Context, trustedTokenHash string) (*structs.TrustedDevice, error)
	UpsertTrustedDevicePending(ctx context.Context, device *structs.TrustedDevice) error
	SetTrustedDeviceVerificationChallenge(ctx context.Context, deviceID int64, target string, codeHash string, expiresAt time.Time) error
	MarkTrustedDeviceVerified(ctx context.Context, deviceID int64, trustedTokenHash string, verifiedAt time.Time) error
	TouchTrustedDevice(ctx context.Context, deviceID int64, seenAt time.Time) error
}
