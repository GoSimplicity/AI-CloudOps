package action

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

// LinkTTL 飞书卡片操作链接的有效期。过期后签名仍然对得上，但校验拒绝。
const LinkTTL = 24 * time.Hour

var ErrInvalidLink = errors.New("signed action link is invalid or expired")

// Sign 用密钥对操作、指纹、附加参数和过期时间做 HMAC-SHA256。密钥为空时拒绝签发。
func Sign(key, actionName, fingerprint, extra string, expires time.Time) (string, error) {
	if strings.TrimSpace(key) == "" || actionName == "" || fingerprint == "" || expires.IsZero() {
		return "", ErrInvalidLink
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(canonical(actionName, fingerprint, extra, expires.Unix())))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Verify 校验签名和过期时间。密钥、签名或过期时间不合法时返回 ErrInvalidLink，不区分具体原因。
func Verify(key, actionName, fingerprint, extra, exp, sig string, now time.Time) error {
	if strings.TrimSpace(key) == "" || actionName == "" || fingerprint == "" || sig == "" || exp == "" {
		return ErrInvalidLink
	}
	expires, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || expires <= 0 {
		return ErrInvalidLink
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(canonical(actionName, fingerprint, extra, expires)))
	decoded, err := hex.DecodeString(sig)
	if err != nil || !hmac.Equal(decoded, mac.Sum(nil)) {
		return ErrInvalidLink
	}
	if !now.Before(time.Unix(expires, 0)) {
		return ErrInvalidLink
	}
	return nil
}

func canonical(actionName, fingerprint, extra string, expires int64) string {
	return actionName + "\n" + fingerprint + "\n" + extra + "\n" + strconv.FormatInt(expires, 10)
}
