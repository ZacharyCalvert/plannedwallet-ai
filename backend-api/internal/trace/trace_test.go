package trace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseTraceparent_Valid(t *testing.T) {
	header := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	tc, err := ParseTraceparent(header)
	if err != nil {
		t.Fatalf("expected valid traceparent, got error: %v", err)
	}
	if tc.Version != "00" {
		t.Errorf("expected version 00, got %s", tc.Version)
	}
	if tc.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("unexpected trace id: %s", tc.TraceID)
	}
	if tc.ParentID != "00f067aa0ba902b7" {
		t.Errorf("unexpected parent id: %s", tc.ParentID)
	}
	if tc.TraceFlags != "01" {
		t.Errorf("expected trace flags 01, got %s", tc.TraceFlags)
	}
	if len(tc.SpanID) != 16 || tc.SpanID == allZerosSpanID {
		t.Errorf("expected new valid 16-hex span id, got %s", tc.SpanID)
	}
}

func TestParseTraceparent_Invalid(t *testing.T) {
	invalidCases := []string{
		"",
		"invalid",
		"ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
		"00-short-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-toolongspan0000000-01",
	}

	for _, c := range invalidCases {
		if _, err := ParseTraceparent(c); err == nil {
			t.Errorf("expected error for case %q, but got nil", c)
		}
	}
}

func TestNewTraceContext_FallbackAndState(t *testing.T) {
	// Fallback case when invalid header
	tc := NewTraceContext("invalid", "congo=t61rcWkgMzE")
	if len(tc.TraceID) != 32 || tc.TraceID == allZerosTraceID {
		t.Errorf("expected valid generated trace id, got %s", tc.TraceID)
	}
	if tc.TraceState != "congo=t61rcWkgMzE" {
		t.Errorf("expected tracestate to be preserved, got %s", tc.TraceState)
	}

	// Preserved case when valid
	valid := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	tc2 := NewTraceContext(valid, "")
	if tc2.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("expected preserved trace ID, got %s", tc2.TraceID)
	}
	if !strings.HasPrefix(tc2.String(), "00-4bf92f3577b34da6a3ce929d0e0e4736-") {
		t.Errorf("unexpected string output: %s", tc2.String())
	}
}

func TestContextCarrying(t *testing.T) {
	tc := NewTraceContext("", "")
	ctx := WithTrace(context.Background(), tc)

	extracted := FromContext(ctx)
	if extracted == nil || extracted.TraceID != tc.TraceID {
		t.Fatalf("expected extracted trace to match, got %+v", extracted)
	}

	if TraceIDFromContext(ctx) != tc.TraceID {
		t.Errorf("TraceIDFromContext mismatch")
	}

	if TraceIDFromContext(context.Background()) != "" {
		t.Errorf("expected empty trace id for empty context")
	}
}

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())

	var capturedTraceID string
	r.GET("/test", func(c *gin.Context) {
		capturedTraceID = TraceIDFromContext(c.Request.Context())
		c.String(http.StatusOK, "ok")
	})

	// 1. Without traceparent header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	respTraceparent := w.Header().Get(HeaderTraceparent)
	if respTraceparent == "" || !strings.Contains(respTraceparent, capturedTraceID) {
		t.Errorf("expected response traceparent containing %s, got %s", capturedTraceID, respTraceparent)
	}

	// 2. With incoming traceparent header
	incoming := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set(HeaderTraceparent, incoming)
	req2.Header.Set(HeaderTracestate, "rojo=1")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if capturedTraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("expected trace id 4bf92f3577b34da6a3ce929d0e0e4736, got %s", capturedTraceID)
	}
	if w2.Header().Get(HeaderTracestate) != "rojo=1" {
		t.Errorf("expected tracestate rojo=1, got %s", w2.Header().Get(HeaderTracestate))
	}
}
