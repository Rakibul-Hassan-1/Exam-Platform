package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"examplatform/internal/models"
)

// Claims mirrors the minimal set of fields the platform needs from a
// token: who the user is, their role, and standard expiry fields.
type Claims struct {
	Sub   string     `json:"sub"`
	Name  string     `json:"name"`
	Role  models.Role `json:"role"`
	IssAt int64      `json:"iat"`
	ExpAt int64      `json:"exp"`
}

var ErrInvalidToken = errors.New("invalid or expired token")

const tokenTTL = 24 * time.Hour

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// IssueToken creates a signed HS256 JWT for the given user.
func IssueToken(secret string, userID, name string, role models.Role) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now()
	claims := Claims{
		Sub:   userID,
		Name:  name,
		Role:  role,
		IssAt: now.Unix(),
		ExpAt: now.Add(tokenTTL).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := b64(headerJSON) + "." + b64(claimsJSON)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := mac.Sum(nil)

	return unsigned + "." + b64(sig), nil
}

// ParseToken verifies the signature and expiry of a token and returns its claims.
func ParseToken(secret, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	unsigned := parts[0] + "." + parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	expectedSig := mac.Sum(nil)

	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare(expectedSig, gotSig) != 1 {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() > claims.ExpAt {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}
