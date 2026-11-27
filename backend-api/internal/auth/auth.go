package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/plannedwallet/internal/models"
)

const (
	DefaultExpirationSeconds = 900
	MobileTokenExpiration    = 180 * 24 * time.Hour
	LimitationMobile         = "mobile"
)

var (
	ErrInvalidToken   = errors.New("invalid or expired token")
	dummyPasswordHash = func() []byte {
		h, _ := bcrypt.GenerateFromPassword([]byte("constant-time-login-pad"), bcrypt.DefaultCost)
		return h
	}()
)

type Claims struct {
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	Subscriber    bool   `json:"subscriber"`
	TermsApproved bool   `json:"terms_approved"`
	Limitation    string `json:"limitation,omitempty"`
	DeviceHash    string `json:"device_hash,omitempty"`
	jwt.RegisteredClaims
}

type Service struct {
	secret     []byte
	expiration time.Duration
}

func NewService(secret string, expirationSeconds int) *Service {
	if secret == "" {
		secret = "development-secret-key-change-in-production"
	}
	if expirationSeconds <= 0 {
		expirationSeconds = DefaultExpirationSeconds
	}
	return &Service{
		secret:     []byte(secret),
		expiration: time.Duration(expirationSeconds) * time.Second,
	}
}

func (s *Service) ExpirationSeconds() int {
	return int(s.expiration.Seconds())
}

// MobileExpirationSeconds returns the lifetime of a long-lived mobile widget token in seconds.
func (s *Service) MobileExpirationSeconds() int {
	return int(MobileTokenExpiration.Seconds())
}

// HashMobileIdentifier returns the lowercase SHA-256 hex digest of a device identifier.
func HashMobileIdentifier(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

// ConstantTimeDeviceMatch compares the SHA-256 hash of a raw header identifier against expectedHash.
func ConstantTimeDeviceMatch(rawHeader, expectedHash string) bool {
	if strings.TrimSpace(rawHeader) == "" || expectedHash == "" {
		return false
	}
	got := HashMobileIdentifier(rawHeader)
	return subtle.ConstantTimeCompare([]byte(got), []byte(expectedHash)) == 1
}

func (s *Service) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *Service) CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// DummyCheckPassword runs bcrypt comparison against a dummy hash to equalize timing for unknown usernames.
func (s *Service) DummyCheckPassword(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
}

func (s *Service) GenerateToken(userID, username string, isSubscriber, termsApproved bool) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:        userID,
		Username:      username,
		Subscriber:    isSubscriber,
		TermsApproved: termsApproved,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateMobileToken issues a 6-month JWT scoped with limitation="mobile" and bound to deviceHash.
func (s *Service) GenerateMobileToken(tokenID, userID, username string, isSubscriber, termsApproved bool, deviceHash string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(MobileTokenExpiration)
	claims := Claims{
		UserID:        userID,
		Username:      username,
		Subscriber:    isSubscriber,
		TermsApproved: termsApproved,
		Limitation:    LimitationMobile,
		DeviceHash:    deviceHash,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	return signed, exp, err
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

func abortAuthError(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, models.ErrorResponse{
		Error:     code,
		Message:   msg,
		Timestamp: time.Now().UTC(),
	})
}

func (s *Service) extractBearerClaims(c *gin.Context) (*Claims, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		abortAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing Authorization header")
		return nil, false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		abortAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header must be Bearer <token>")
		return nil, false
	}
	claims, err := s.ValidateToken(parts[1])
	if err != nil {
		abortAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired access token")
		return nil, false
	}
	return claims, true
}

func setContextClaims(c *gin.Context, claims *Claims) {
	c.Set("userID", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("subscriber", claims.Subscriber)
	c.Set("termsApproved", claims.TermsApproved)
	c.Set("tokenID", claims.ID)
	c.Set("tokenLimitation", claims.Limitation)
}

// Middleware enforces full-scope Bearer JWT authentication and rejects limited tokens (e.g., mobile tokens).
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := s.extractBearerClaims(c)
		if !ok {
			return
		}
		if claims.Limitation != "" {
			abortAuthError(c, http.StatusForbidden, "TOKEN_SCOPE_LIMITED", "Limited mobile tokens cannot access standard API endpoints.")
			return
		}
		setContextClaims(c, claims)
		c.Next()
	}
}

// MobileMiddleware enforces mobile-limited Bearer JWT authentication and X-Mobile-Identifier device binding.
func (s *Service) MobileMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := s.extractBearerClaims(c)
		if !ok {
			return
		}
		if claims.Limitation != LimitationMobile {
			abortAuthError(c, http.StatusForbidden, "MOBILE_TOKEN_REQUIRED", "A mobile-scoped token is required for this endpoint.")
			return
		}
		mobileID := c.GetHeader("X-Mobile-Identifier")
		if !ConstantTimeDeviceMatch(mobileID, claims.DeviceHash) {
			abortAuthError(c, http.StatusUnauthorized, "INVALID_MOBILE_IDENTIFIER", "Missing or mismatched X-Mobile-Identifier header.")
			return
		}
		setContextClaims(c, claims)
		c.Next()
	}
}
