package trace

import (
	"context"
	"fmt"
	"log"
)

func formatPrefix(ctx context.Context) string {
	if tc := FromContext(ctx); tc != nil {
		return fmt.Sprintf("[trace_id=%s span_id=%s] ", tc.TraceID, tc.SpanID)
	}
	return "[trace_id=- span_id=-] "
}

// LogInfo logs an informational message with correlated trace context.
func LogInfo(ctx context.Context, format string, args ...any) {
	log.Printf(formatPrefix(ctx)+format, args...)
}

// LogError logs an error message and optional underlying error with correlated trace context.
func LogError(ctx context.Context, err error, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if err != nil {
		log.Printf(formatPrefix(ctx)+"ERROR: %s (err: %v)", msg, err)
	} else {
		log.Printf(formatPrefix(ctx)+"ERROR: %s", msg)
	}
}
