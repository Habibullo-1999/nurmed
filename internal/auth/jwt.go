package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"nurmed/internal/structs"
)

func SignHS256Token(claims structs.AccessClaims, secret []byte) (string, error) {
	header, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := encodeSegment(header) + "." + encodeSegment(payload)
	signature := computeHMAC(unsigned, secret)
	return unsigned + "." + encodeSegment(signature), nil
}

func VerifyHS256Token(token string, secret []byte, now time.Time) (structs.AccessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return structs.AccessClaims{}, errors.New("invalid token format")
	}

	unsigned := parts[0] + "." + parts[1]
	expectedSignature := computeHMAC(unsigned, secret)
	providedSignature, err := decodeSegment(parts[2])
	if err != nil {
		return structs.AccessClaims{}, err
	}

	if subtle.ConstantTimeCompare(expectedSignature, providedSignature) != 1 {
		return structs.AccessClaims{}, errors.New("invalid token signature")
	}

	payload, err := decodeSegment(parts[1])
	if err != nil {
		return structs.AccessClaims{}, err
	}

	var claims structs.AccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return structs.AccessClaims{}, err
	}

	if claims.Sub == 0 || claims.Exp == 0 {
		return structs.AccessClaims{}, errors.New("invalid token claims")
	}

	if now.Unix() >= claims.Exp {
		return structs.AccessClaims{}, errors.New("token expired")
	}

	return claims, nil
}

func RandomToken(bytesLen int) (string, error) {
	if bytesLen < 16 {
		return "", fmt.Errorf("token length is too small: %d", bytesLen)
	}
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func computeHMAC(data string, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func encodeSegment(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}
