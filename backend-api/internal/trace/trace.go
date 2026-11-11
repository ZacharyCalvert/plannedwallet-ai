package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	HeaderTraceparent = "traceparent"
	HeaderTracestate  = "tracestate"
	HeaderClientID    = "X-Client-ID"
	DefaultVersion    = "00"
	DefaultFlags      = "01"
)

var (
	traceparentRegex = regexp.MustCompile(`^([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$`)
	allZerosTraceID  = strings.Repeat("0", 32)
	allZerosSpanID   = strings.Repeat("0", 16)
)

// TraceContext represents parsed or generated W3C Trace Context fields.
type TraceContext struct {
	Version    string
	TraceID    string
	SpanID     string
	ParentID   string
	TraceFlags string
	TraceState string
	ClientID   string
}

func (tc *TraceContext) String() string {
	if tc == nil {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s-%s", tc.Version, tc.TraceID, tc.SpanID, tc.TraceFlags)
}

func generateRandomHex(byteCount int) string {
	b := make([]byte, byteCount)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("1", byteCount*2)
	}
	return hex.EncodeToString(b)
}

// GenerateTraceID generates a 16-byte random non-zero hex string (32 hex characters).
func GenerateTraceID() string {
	for {
		id := generateRandomHex(16)
		if id != allZerosTraceID {
			return id
		}
	}
}

// GenerateSpanID generates an 8-byte random non-zero hex string (16 hex characters).
func GenerateSpanID() string {
	for {
		id := generateRandomHex(8)
		if id != allZerosSpanID {
			return id
		}
	}
}

// ParseTraceparent validates and parses a W3C traceparent header string.
func ParseTraceparent(raw string) (*TraceContext, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	m := traceparentRegex.FindStringSubmatch(raw)
	if len(m) != 5 {
		return nil, errors.New("invalid traceparent format")
	}
	version, traceID, parentID, flags := m[1], m[2], m[3], m[4]
	if version == "ff" || traceID == allZerosTraceID || parentID == allZerosSpanID {
		return nil, errors.New("invalid traceparent values")
	}
	return &TraceContext{
		Version:    version,
		TraceID:    traceID,
		SpanID:     GenerateSpanID(),
		ParentID:   parentID,
		TraceFlags: flags,
	}, nil
}

// NewTraceContext initializes a TraceContext from incoming headers or generates fresh context.
func NewTraceContext(parentHeader, stateHeader string) *TraceContext {
	tc, err := ParseTraceparent(parentHeader)
	if err != nil {
		tc = &TraceContext{
			Version:    DefaultVersion,
			TraceID:    GenerateTraceID(),
			SpanID:     GenerateSpanID(),
			ParentID:   "",
			TraceFlags: DefaultFlags,
		}
	}
	tc.TraceState = strings.TrimSpace(stateHeader)
	return tc
}

type contextKey struct{}

var traceCtxKey = contextKey{}

// WithTrace returns a copy of parent context carrying the TraceContext.
func WithTrace(ctx context.Context, tc *TraceContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceCtxKey, tc)
}

// FromContext extracts the TraceContext from context, or returns nil if absent.
func FromContext(ctx context.Context) *TraceContext {
	if ctx == nil {
		return nil
	}
	tc, ok := ctx.Value(traceCtxKey).(*TraceContext)
	if !ok {
		return nil
	}
	return tc
}

// TraceIDFromContext extracts only the TraceID from context, or returns empty string.
func TraceIDFromContext(ctx context.Context) string {
	if tc := FromContext(ctx); tc != nil {
		return tc.TraceID
	}
	return ""
}

// ClientIDFromContext extracts the ClientID from context, or returns empty string.
func ClientIDFromContext(ctx context.Context) string {
	if tc := FromContext(ctx); tc != nil {
		return tc.ClientID
	}
	return ""
}
