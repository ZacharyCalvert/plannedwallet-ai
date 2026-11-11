package trace

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/models"
)

const ContextKey = "traceContext"

func applyResponseHeaders(c *gin.Context, tc *TraceContext) {
	c.Header(HeaderTraceparent, tc.String())
	if tc.TraceState != "" {
		c.Header(HeaderTracestate, tc.TraceState)
	}
}

func logCompletion(ctx context.Context, method, path string, status int, latency time.Duration) {
	if status >= 400 {
		LogError(ctx, nil, "Completed %s %s -> %d (in %v)", method, path, status, latency)
		return
	}
	LogInfo(ctx, "Completed %s %s -> %d (in %v)", method, path, status, latency)
}

func logClientTrace(ctx context.Context, tc *TraceContext, clientID string) {
	LogInfo(ctx, "Received %s: %s", HeaderClientID, clientID)
	LogInfo(ctx, "Trace context [trace_id=%s span_id=%s] correlated with %s: %s", tc.TraceID, tc.SpanID, HeaderClientID, clientID)
}

func enforceClientID(c *gin.Context, ctx context.Context) (string, bool) {
	clientID := strings.TrimSpace(c.GetHeader(HeaderClientID))
	if clientID == "" {
		LogError(ctx, nil, "Missing or empty required %s header", HeaderClientID)
		c.AbortWithStatusJSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "MISSING_CLIENT_ID",
			Message:   "Missing or invalid required X-Client-ID header.",
			Timestamp: time.Now().UTC(),
		})
		return "", false
	}
	return clientID, true
}

func handleClientID(c *gin.Context, ctx context.Context, tc *TraceContext) bool {
	if c.Request.Method == http.MethodOptions {
		if clientID := strings.TrimSpace(c.GetHeader(HeaderClientID)); clientID != "" {
			tc.ClientID = clientID
			c.Set("clientID", clientID)
			logClientTrace(ctx, tc, clientID)
		}
		return true
	}
	clientID, ok := enforceClientID(c, ctx)
	if !ok {
		return false
	}
	tc.ClientID = clientID
	c.Set("clientID", clientID)
	logClientTrace(ctx, tc, clientID)
	return true
}

// Middleware injects W3C Trace Context and enforces X-Client-ID on requests.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		tc := NewTraceContext(c.GetHeader(HeaderTraceparent), c.GetHeader(HeaderTracestate))

		ctx := WithTrace(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		c.Set(ContextKey, tc)
		c.Set("traceID", tc.TraceID)
		applyResponseHeaders(c, tc)

		if !handleClientID(c, ctx, tc) {
			logCompletion(ctx, c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
			return
		}

		LogInfo(ctx, "Started %s %s from %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())

		c.Next()

		logCompletion(ctx, c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}
