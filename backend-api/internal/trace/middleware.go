package trace

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
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

// Middleware injects W3C Trace Context into request context and response headers.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		tc := NewTraceContext(c.GetHeader(HeaderTraceparent), c.GetHeader(HeaderTracestate))

		ctx := WithTrace(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		c.Set(ContextKey, tc)
		c.Set("traceID", tc.TraceID)
		applyResponseHeaders(c, tc)

		LogInfo(ctx, "Started %s %s from %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())

		c.Next()

		logCompletion(ctx, c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}
