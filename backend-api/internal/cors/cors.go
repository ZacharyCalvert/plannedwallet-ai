package cors

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/trace"
)

var defaultAllowedClients = []string{
	"www.plannedwallet.com",
	"local.plannedwallet.com",
	"localhost.plannedwallet.com",
	"stage.plannedwallet.com",
	"dev.plannedwallet.com",
}

// CleanHost removes any port or scheme from the host string and returns lowercase host.
func CleanHost(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
			return strings.ToLower(u.Hostname())
		}
	}
	if h, _, err := net.SplitHostPort(raw); err == nil {
		return strings.ToLower(h)
	}
	return raw
}

// ParseAllowedClients splits a comma-separated list of domains into cleaned hostnames.
func ParseAllowedClients(env string) []string {
	parts := strings.Split(env, ",")
	clients := make([]string, 0, len(parts))
	for _, p := range parts {
		if cleaned := CleanHost(p); cleaned != "" {
			clients = append(clients, cleaned)
		}
	}
	return clients
}

// LoadAllowedClients reads environment variables or returns default allowed clients.
func LoadAllowedClients() []string {
	keys := []string{"CORS_ALLOWED_CLIENTS", "ALLOWED_CLIENTS", "CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_HOSTS"}
	for _, k := range keys {
		if val := os.Getenv(k); strings.TrimSpace(val) != "" {
			return ParseAllowedClients(val)
		}
	}
	return defaultAllowedClients
}

// IsAllowedHost checks if host is present in allowed list.
func IsAllowedHost(host string, allowed []string) bool {
	clean := CleanHost(host)
	if clean == "" {
		return false
	}
	for _, a := range allowed {
		if clean == a {
			return true
		}
	}
	return false
}

func getRequestHost(c *gin.Context) string {
	if c.Request.Host != "" {
		return c.Request.Host
	}
	return c.GetHeader("Host")
}

func applyCORSHeaders(c *gin.Context, origin string) {
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
	c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Client-ID, traceparent, tracestate")
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Max-Age", "86400")
	c.Header("Access-Control-Expose-Headers", "traceparent, tracestate, X-Client-ID")
}

func handlePreflight(c *gin.Context, allowed []string) {
	host := CleanHost(getRequestHost(c))
	origin := c.GetHeader("Origin")
	originHost := CleanHost(origin)

	if !IsAllowedHost(host, allowed) && !IsAllowedHost(originHost, allowed) {
		trace.LogError(c.Request.Context(), nil, "CORS pre-flight rejected for host: %q, origin: %q", host, origin)
		c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
			Error:     "CORS_FORBIDDEN",
			Message:   "Host is not an allowed CORS client.",
			Timestamp: time.Now().UTC(),
		})
		return
	}

	allowOrigin := origin
	if allowOrigin == "" {
		allowOrigin = "https://" + host
	}
	applyCORSHeaders(c, allowOrigin)
	trace.LogInfo(c.Request.Context(), "CORS pre-flight challenge accepted for host: %q", host)
	c.Status(http.StatusNoContent)
	c.Abort()
}

func handleActual(c *gin.Context, allowed []string) {
	origin := c.GetHeader("Origin")
	if origin != "" && IsAllowedHost(CleanHost(origin), allowed) {
		applyCORSHeaders(c, origin)
	}
	c.Next()
}

// Middleware creates a Gin middleware that handles CORS challenges based on allowed clients.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed := LoadAllowedClients()
		if c.Request.Method == http.MethodOptions {
			handlePreflight(c, allowed)
			return
		}
		handleActual(c, allowed)
	}
}
