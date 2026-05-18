package structs

import "time"

const (
	UserStatusActive = "active"
)

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type AuthMeta struct {
	IP        string
	UserAgent string
}

type AuthTokens struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	UserID                int64
	UserName              string
	CompanyID             int64
}

type AuthResponse struct {
	AccessToken          string    `json:"accessToken"`
	AccessTokenExpiresAt time.Time `json:"accessTokenExpiresAt"`
	TokenType            string    `json:"tokenType"`
	UserID               int64     `json:"userId"`
	UserName             string    `json:"userName"`
	CompanyID            int64     `json:"companyId"`
}

type AccessClaims struct {
	Sub          int64  `json:"sub"`
	CompanyID    int64  `json:"company_id,omitempty"`
	IsSuperAdmin bool   `json:"is_super_admin"`
	JTI          string `json:"jti"`
	Exp          int64  `json:"exp"`
	Iat          int64  `json:"iat"`
}

type AuthSession struct {
	ID                   int64
	UserID               int64
	RefreshHash          string
	ExpiresAt            time.Time
	IP                   string
	UserAgent            string
	RevokedAt            *time.Time
	RotatedFromSessionID *int64
	CreatedAt            time.Time
}

type PermissionAssignment struct {
	PermissionCode string
	ScopeType      string
	ScopeID        *int64
	OwnOnly        bool
}

type RoleScope struct {
	ScopeType string
	ScopeID   *int64
	OwnOnly   bool
}

type RequestScope struct {
	CompanyID   int64
	BranchID    int64
	WarehouseID int64
	OwnerUserID int64
}

type UserPrincipal struct {
	UserID                int64
	UserName              string
	CompanyID             int64
	IsSuperAdmin          bool
	PermissionAssignments []PermissionAssignment
	RoleScopes            []RoleScope
}

type AuditLog struct {
	UserID     *int64
	Action     string
	Module     string
	Resource   string
	ResourceID *string
	Meta       map[string]interface{}
	CreatedAt  time.Time
}

type TrustedDevice struct {
	ID                 int64
	FingerprintHash    string
	BrowserName        string
	BrowserVersion     string
	OSName             string
	OSVersion          string
	DeviceName         string
	PendingTokenHash   string
	PendingExpiresAt   *time.Time
	VerifyTarget       string
	VerifyCodeHash     string
	VerifyExpiresAt    *time.Time
	TrustedTokenHash   string
	TrustedAt          *time.Time
	LastSeenAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type DeviceCheckRequest struct {
	DeviceInfo DeviceInfo `json:"deviceInfo" binding:"required"`
}

type DeviceInfo struct {
	BrowserName    string `json:"browserName"`
	BrowserVersion string `json:"browserVersion"`
	OSName         string `json:"osName"`
	OSVersion      string `json:"osVersion"`
	DeviceName     string `json:"deviceName"`
}

type DeviceCheckResponse struct {
	Status string `json:"status"` // "verified" | "verification_required"
}

type DeviceVerificationRequest struct {
	Identifier       string     `json:"identifier" binding:"required"`
	VerificationCode string     `json:"verificationCode,omitempty"`
	DeviceInfo       DeviceInfo `json:"deviceInfo" binding:"required"`
}

type DeviceVerificationResponse struct {
	Verified bool   `json:"verified"`
	CodeSent bool   `json:"codeSent,omitempty"`
	Message  string `json:"message"`
}
