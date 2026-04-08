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

	"nurmed/internal/structs"
)

var (
	ErrPendingDeviceNotFound   = errors.New("pending device not found")
	ErrVerificationCodeExpired = errors.New("verification code expired")
	ErrVerificationCodeInvalid = errors.New("verification code invalid")
)

// CheckDevice проверяет, является ли устройство новым или требует верификации
func (s *service) CheckDevice(ctx context.Context, deviceInfo structs.DeviceInfo, trustedToken string) (structs.DeviceCheckResponse, string, error) {
	response := structs.DeviceCheckResponse{Status: "verification_required"}
	fingerprintHash := generateFingerprintHash(deviceInfo)

	if trustedToken != "" {
		device, err := s.repo.GetTrustedDeviceByTrustedTokenHash(ctx, hashToken(trustedToken))
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return response, "", err
		}
		if err == nil && device != nil && device.FingerprintHash == fingerprintHash && device.TrustedAt != nil {
			_ = s.repo.TouchTrustedDevice(ctx, device.ID, time.Now().UTC())
			return structs.DeviceCheckResponse{Status: "verified"}, "", nil
		}
	}

	pendingToken, err := RandomToken(32)
	if err != nil {
		return response, "", err
	}

	now := time.Now().UTC()
	device := &structs.TrustedDevice{
		FingerprintHash:  fingerprintHash,
		BrowserName:      deviceInfo.BrowserName,
		BrowserVersion:   deviceInfo.BrowserVersion,
		OSName:           deviceInfo.OSName,
		OSVersion:        deviceInfo.OSVersion,
		DeviceName:       deviceInfo.DeviceName,
		PendingTokenHash: hashToken(pendingToken),
		PendingExpiresAt: ptrTime(now.Add(s.devicePendingTTL)),
	}

	if err := s.repo.UpsertTrustedDevicePending(ctx, device); err != nil {
		return response, "", err
	}

	s.audit(ctx, structs.AuditLog{
		Action:   "auth.device_check",
		Module:   "auth",
		Resource: "device",
		Meta: map[string]interface{}{
			"fingerprint_hash": fingerprintHash,
			"browser":          deviceInfo.BrowserName,
			"device_name":      deviceInfo.DeviceName,
			"status":           response.Status,
		},
		CreatedAt: now,
	})

	return response, pendingToken, nil
}

// VerifyDevice проверяет и завершает процесс верификации устройства
func (s *service) VerifyDevice(ctx context.Context, request structs.DeviceVerificationRequest, pendingToken string) (string, bool, error) {
	pendingToken = strings.TrimSpace(pendingToken)
	identifier := normalizeIdentifier(request.Identifier)
	code := strings.TrimSpace(request.VerificationCode)

	if pendingToken == "" || identifier == "" {
		return "", false, ErrPendingDeviceNotFound
	}

	device, err := s.repo.GetTrustedDeviceByPendingTokenHash(ctx, hashToken(pendingToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, ErrPendingDeviceNotFound
		}
		return "", false, err
	}
	if device == nil || device.PendingExpiresAt == nil || !device.PendingExpiresAt.After(time.Now().UTC()) {
		return "", false, ErrPendingDeviceNotFound
	}

	if device.FingerprintHash != generateFingerprintHash(request.DeviceInfo) {
		return "", false, ErrForbidden
	}

	if code == "" {
		generatedCode, err := generateVerificationCode()
		if err != nil {
			return "", false, err
		}
		codeHash := hashToken(generatedCode)
		expiresAt := time.Now().UTC().Add(s.deviceCodeTTL)

		if err := s.repo.SetTrustedDeviceVerificationChallenge(ctx, device.ID, identifier, codeHash, expiresAt); err != nil {
			return "", false, err
		}

		// TODO: подключить реальную отправку email/sms провайдером.
		s.logger.Info(ctx, fmt.Sprintf("device verify code for %s: %s", maskIdentifier(identifier), generatedCode))

		s.audit(ctx, structs.AuditLog{
			Action:   "auth.device_verify_code_sent",
			Module:   "auth",
			Resource: "device",
			Meta: map[string]interface{}{
				"target": maskIdentifier(identifier),
			},
			CreatedAt: time.Now().UTC(),
		})

		return "", true, nil
	}

	if normalizeIdentifier(device.VerifyTarget) != identifier {
		return "", false, ErrVerificationCodeInvalid
	}
	if device.VerifyExpiresAt == nil || !device.VerifyExpiresAt.After(time.Now().UTC()) {
		return "", false, ErrVerificationCodeExpired
	}
	if hashToken(code) != device.VerifyCodeHash {
		return "", false, ErrVerificationCodeInvalid
	}

	trustedToken, err := s.completeTrustedDevice(ctx, device)
	if err != nil {
		return "", false, err
	}

	return trustedToken, false, nil
}

// TrustPendingDevice доверяет устройству с ожидающим токеном
func (s *service) TrustPendingDevice(ctx context.Context, pendingToken string) (string, error) {
	pendingToken = strings.TrimSpace(pendingToken)
	if pendingToken == "" {
		return "", nil
	}

	device, err := s.repo.GetTrustedDeviceByPendingTokenHash(ctx, hashToken(pendingToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if device == nil || device.PendingExpiresAt == nil || !device.PendingExpiresAt.After(time.Now().UTC()) {
		return "", nil
	}

	return s.completeTrustedDevice(ctx, device)
}

func (s *service) completeTrustedDevice(ctx context.Context, device *structs.TrustedDevice) (string, error) {
	trustedToken, err := RandomToken(32)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	if err := s.repo.MarkTrustedDeviceVerified(ctx, device.ID, hashToken(trustedToken), now); err != nil {
		return "", err
	}

	s.audit(ctx, structs.AuditLog{
		Action:   "auth.device_trusted",
		Module:   "auth",
		Resource: "device",
		Meta: map[string]interface{}{
			"fingerprint_hash": device.FingerprintHash,
			"browser":          device.BrowserName,
			"device_name":      device.DeviceName,
		},
		CreatedAt: now,
	})

	return trustedToken, nil
}

func generateFingerprintHash(deviceInfo structs.DeviceInfo) string {
	combined := fmt.Sprintf("%s_%s_%s_%s_%s",
		deviceInfo.BrowserName,
		deviceInfo.BrowserVersion,
		deviceInfo.OSName,
		deviceInfo.OSVersion,
		deviceInfo.DeviceName,
	)

	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

func generateVerificationCode() (string, error) {
	// TODO: после реализации отправка кода через EMAIL/ SMS провайдера,
	// раскомментировать и использовать генерацию рандомного кода вместо статического.
	//var b [4]byte
	//if _, err := rand.Read(b[:]); err != nil {
	//	return "", err
	//}
	//value := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	//if value < 0 {
	//	value = -value
	//}
	//code := value % 1000000
	//return fmt.Sprintf("%06d", code), nil

	value := 123456
	return fmt.Sprintf("%06d", value), nil
}

func normalizeIdentifier(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func maskIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if len(identifier) <= 4 {
		return "****"
	}
	return identifier[:2] + "****" + identifier[len(identifier)-2:]
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
